package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// Defaults mirror docker-compose emulator settings (see docker-compose.yml).
	defaultProjectID      = "test-project" // created by mk-topic/mk-subscription services.
	defaultSubscriptionID = "scan-sub"     // created by mk-subscription service.
	defaultEmulatorHost   = "localhost:8085"

	// Local persistence defaults to sqlite, stored in workspace
	defaultDatastore = "sqlite"
	defaultDBPath    = "data/mini_scan.db"

	// Sensible single-node processing defaults; tune via env vars when scaling.
	defaultWorkerCount  = 4 // Vertical scaling on same node
	defaultAckExtension = 60 * time.Second
)

// Config aggregates runtime settings for the processor service.
type Config struct {
	ProjectID       string
	SubscriptionID  string
	EmulatorHost    string
	DBPath          string
	Datastore       string
	WorkerCount     int
	AckExtension    time.Duration
	ShutdownTimeout time.Duration
}

// Load reads configuration from environment variables, applying defaults and validation.
func Load() (*Config, error) {
	projectID := readEnvOrDefault("PUBSUB_PROJECT_ID", defaultProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("PUBSUB_PROJECT_ID is required")
	}

	subscriptionID := readEnvOrDefault("PUBSUB_SUBSCRIPTION_ID", defaultSubscriptionID)
	if subscriptionID == "" {
		return nil, fmt.Errorf("PUBSUB_SUBSCRIPTION_ID is required")
	}

	emulatorHost := readEnvOrDefault("PUBSUB_EMULATOR_HOST", defaultEmulatorHost)

	dbPath := readEnvOrDefault("DB_PATH", defaultDBPath)

	datastore := readEnvOrDefault("DATASTORE", defaultDatastore)

	workerCount, err := parsePositiveInt("WORKER_COUNT", defaultWorkerCount)
	if err != nil {
		return nil, err
	}

	ackExtensionSeconds, err := parsePositiveInt("ACK_EXTENSION_SECONDS", int(defaultAckExtension.Seconds()))
	if err != nil {
		return nil, err
	}

	shutdownTimeoutSeconds, err := parsePositiveInt("SHUTDOWN_TIMEOUT_SECONDS", 30)
	if err != nil {
		return nil, err
	}

	return &Config{
		ProjectID:       projectID,
		SubscriptionID:  subscriptionID,
		EmulatorHost:    emulatorHost,
		DBPath:          dbPath,
		Datastore:       datastore,
		WorkerCount:     workerCount,
		AckExtension:    time.Duration(ackExtensionSeconds) * time.Second,
		ShutdownTimeout: time.Duration(shutdownTimeoutSeconds) * time.Second,
	}, nil
}

func readEnvOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func parsePositiveInt(key string, fallback int) (int, error) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return parsed, nil
}
