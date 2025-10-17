# Mini-Scan
## Solution Overview

This repository now includes a processing service (`cmd/processor`) that consumes scan messages from the Pub/Sub emulator, normalizes the payload (handling both `data_version` `1` and `2`), and persists the freshest observation for each `(ip, port, service)` tuple in SQLite. The processor uses an environment-driven configuration layer and a Data Access Layer (`/dal`) abstraction so alternative pubsubs and datastores can be swapped in without changing business logic.

### Key pieces
- **Processor** (`cmd/processor`): Pulls from subscription `scan-sub`, decodes responses, and performs newest-wins upserts.
- **Data access layer** (`pkg/dal`): Defines the repository interface; `pkg/dal/sqlite` provides the default implementation using the Go SQLite driver.
   - `UpsertLatest` enables scans to be upserted into the table, and discards results older than the current latest result.
- **Scanner** (`cmd/scanner`): Unchanged publisher that emits random scans to the emulator.
- **Configuration** (`pkg/config`): Reads environment variables (with sensible defaults matching `docker-compose.yml`).

### Configuration
All processor settings are controlled via environment variables. Defaults align with `docker-compose.yml`. These are not expected to be used for this exercise; they demonstrate the ability to scan this system to other pubsubs and datastores.

| Variable | Description | Default |
| --- | --- | --- |
| `PUBSUB_PROJECT_ID` | Pub/Sub project ID | `test-project` |
| `PUBSUB_SUBSCRIPTION_ID` | Subscription to pull from | `scan-sub` |
| `PUBSUB_EMULATOR_HOST` | Emulator host/port | `localhost:8085` |
| `DATASTORE` | Datastore driver identifier | `sqlite` |
| `DB_PATH` | Datastore path (used by SQLite) | `data/mini_scan.db` |


## Manual validation

### Prerequisites
- Go 1.20+
- Docker & Docker Compose

### Running the stack

1. Start the full emulator + scanner + processor stack:
   ```sh
   docker compose up
   ```
   Logs will show the scanner publishing messages and the processor storing or skipping (stale) scans.

2. Inspect stored results:
   ```sh
   docker compose exec processor sqlite3 /data/mini_scan.db \
   'SELECT ip, port, service, observed_at, response
      FROM service_scans
      ORDER BY observed_at DESC
      LIMIT 5;'
   ```
   You should see ~1 new stored result per second.

3. Tear everything down with `Ctrl+C` or `docker compose down`.

### Unit tests

- Unit tests (automated end-to-end tests are omitted to keep the take-home harness simple):
  ```sh
  GOCACHE=$(pwd)/.gocache \
  GOMODCACHE=$(pwd)/.gopath/pkg/mod \
  go test ./...
  ```

---

## Appendix: Production Hardening

The current solution focuses on demonstrating end-to-end processing for the take-home. In a production deployment we would add:

- **Horizontal Scaling Backing Store**: Swap SQLite for a more powerful multi-writer datastore so multiple processor instances can safely share state. Update docs/README with scaling instructions once an additional datastore is available. I recommend Google Cloud Bigtable: as the backing for Google's own web crawler, it has proven scalability; it natively handles versioning; and its semantics handle the usecases laid out here at lower price than most relational offerings. We can also use a large Postgres cluster, or Google Cloud Spanner if preserving relational semantics is a priority.
- **CI/CD & Testing**: Automate `go test`, linting, and integration tests that stand up the Pub/Sub emulator, publish fixtures, and assert persisted state before deployment.
- **Resilience, Alerting, Metrics**: Configurable ack deadlines, exponential backoff for transient failures, alerting on repeated decode/store issues, health/ready probes, metrics on success/failure rates, latency (Prometheus or similar).
