package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/censys/scan-takehome/pkg/processor"
)

func TestUpsertLatest(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	scan := &processor.ServiceScan{
		IP:         "192.0.2.1",
		Port:       443,
		Service:    "HTTPS",
		ObservedAt: time.Unix(873237600, 0).UTC(), // significant day :)
		Response:   "hello",
		MessageID:  "msg-1",
	}

	changed, err := repo.UpsertLatest(ctx, scan)
	if err != nil {
		t.Fatalf("UpsertLatest insert failed: %v", err)
	}
	if !changed {
		t.Fatalf("expected insert to report changed")
	}

	stored, err := repo.Fetch(ctx, scan.IP, scan.Port, scan.Service)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if stored == nil {
		t.Fatalf("expected stored record")
	}
	if stored.MessageID != "msg-1" {
		t.Fatalf("expected message_id msg-1, got %s", stored.MessageID)
	}

	older := &processor.ServiceScan{
		IP:         scan.IP,
		Port:       scan.Port,
		Service:    scan.Service,
		ObservedAt: scan.ObservedAt.Add(-time.Hour),
		Response:   "old",
		MessageID:  "msg-0",
	}

	changed, err = repo.UpsertLatest(ctx, older)
	if err != nil {
		t.Fatalf("UpsertLatest older failed: %v", err)
	}
	if changed {
		t.Fatalf("expected stale update to report unchanged")
	}

	newer := &processor.ServiceScan{
		IP:         scan.IP,
		Port:       scan.Port,
		Service:    scan.Service,
		ObservedAt: scan.ObservedAt.Add(time.Hour),
		Response:   "new",
		MessageID:  "msg-2",
	}

	changed, err = repo.UpsertLatest(ctx, newer)
	if err != nil {
		t.Fatalf("UpsertLatest newer failed: %v", err)
	}
	if !changed {
		t.Fatalf("expected newer update to change record")
	}

	stored, err = repo.Fetch(ctx, scan.IP, scan.Port, scan.Service)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if stored == nil || stored.Response != "new" {
		t.Fatalf("expected response new, got %+v", stored)
	}
}
