package config

import (
	"os"
	"strings"
)

const (
	// Defaults mirror docker-compose emulator settings (see docker-compose.yml).
	defaultProjectID      = "test-project" // created by mk-topic/mk-subscription services.
	defaultSubscriptionID = "scan-sub"     // created by mk-subscription service.
	defaultEmulatorHost   = "localhost:8085"

	// Local persistence defaults to sqlite, stored in workspace
	defaultDatastore = "sqlite"
	defaultDBPath    = "data/mini_scan.db"
)

// Config aggregates runtime settings for the processor service.
type Config struct {
	ProjectID      string
	SubscriptionID string
	EmulatorHost   string
	Datastore      string
	DBPath         string
}

// Load reads config from environment variables, applying defaults.
// NOTE: none of these are expected to be set for this exercise;
// they demonstrate how we drop this other pubsubs and datastores in.
// We can also use these to tune pubsub settings, single-node concurrency, etc.
func Load() (*Config, error) {
	projectID := readEnv("PUBSUB_PROJECT_ID", defaultProjectID)
	subscriptionID := readEnv("PUBSUB_SUBSCRIPTION_ID", defaultSubscriptionID)
	emulatorHost := readEnv("PUBSUB_EMULATOR_HOST", defaultEmulatorHost)
	datastore := readEnv("DATASTORE", defaultDatastore)
	dbPath := readEnv("DB_PATH", defaultDBPath)

	return &Config{
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		EmulatorHost:   emulatorHost,
		Datastore:      datastore,
		DBPath:         dbPath,
	}, nil
}

// readEnv returns a key's value from environment, or fallback
func readEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}
