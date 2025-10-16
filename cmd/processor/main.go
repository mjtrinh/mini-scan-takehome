package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/censys/scan-takehome/pkg/config"
	"github.com/censys/scan-takehome/pkg/processor"
	sqlite "github.com/censys/scan-takehome/pkg/storage/sqlite"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Ensure emulator host is exported for the Pub/Sub client when running locally.
	if cfg.EmulatorHost != "" {
		if err := os.Setenv("PUBSUB_EMULATOR_HOST", cfg.EmulatorHost); err != nil {
			log.Fatalf("set emulator host: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown signals.
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownCh
		log.Printf("shutdown signal received; allowing up to %s for in-flight work", cfg.ShutdownTimeout)
		cancel()
		time.Sleep(cfg.ShutdownTimeout)
	}()

	repo, err := sqlite.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer repo.Close()

	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		log.Fatalf("pubsub client: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(cfg.SubscriptionID)
	sub.ReceiveSettings = pubsub.ReceiveSettings{
		NumGoroutines:          cfg.WorkerCount,
		MaxOutstandingMessages: cfg.WorkerCount * 4,
		MaxExtension:           cfg.AckExtension,
	}

	exists, err := sub.Exists(ctx)
	if err != nil {
		log.Fatalf("checking subscription %q: %v", cfg.SubscriptionID, err)
	}
	if !exists {
		log.Fatalf("subscription %q not found", cfg.SubscriptionID)
	}

	log.Printf("processor ready; project=%s subscription=%s emulator=%s db=%s", cfg.ProjectID, cfg.SubscriptionID, cfg.EmulatorHost, cfg.DBPath)

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		scan, err := processor.ParseScanEnvelope(msg.Data)
		if err != nil {
			log.Printf("message decode error id=%s: %v", msg.ID, err)
			msg.Ack()
			return
		}
		scan.MessageID = msg.ID

		changed, err := repo.UpsertLatest(ctx, scan)
		if err != nil {
			log.Printf("store error id=%s: %v", msg.ID, err)
			msg.Nack()
			return
		}

		if changed {
			log.Printf("stored scan ip=%s port=%d service=%s observed=%s", scan.IP, scan.Port, scan.Service, scan.ObservedAt.Format(time.RFC3339))
		} else {
			log.Printf("stale scan ignored ip=%s port=%d service=%s observed=%s", scan.IP, scan.Port, scan.Service, scan.ObservedAt.Format(time.RFC3339))
		}
		msg.Ack()
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("subscription receive error: %v", err)
	}

	log.Println("processor exiting")
}
