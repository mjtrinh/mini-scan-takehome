# Mini-Scan

Hello!

As you've heard by now, Censys scans the internet at an incredible scale. Processing the results necessitates scaling horizontally across thousands of machines. One key aspect of our architecture is the use of distributed queues to pass data between machines.

---

The `docker-compose.yml` file sets up a toy example of a scanner. It spins up a Google Pub/Sub emulator, creates a topic and subscription, and publishes scan results to the topic. It can be run via `docker compose up`.

Your job is to build the data processing side. It should:

1. Pull scan results from the subscription `scan-sub`.
2. Maintain an up-to-date record of each unique `(ip, port, service)`. This should contain when the service was last scanned and a string containing the service's response.

> **_NOTE_**
> The scanner can publish data in two formats, shown below. In both of the following examples, the service response should be stored as: `"hello world"`.
>
> ```javascript
> {
>   // ...
>   "data_version": 1,
>   "data": {
>     "response_bytes_utf8": "aGVsbG8gd29ybGQ="
>   }
> }
>
> {
>   // ...
>   "data_version": 2,
>   "data": {
>     "response_str": "hello world"
>   }
> }
> ```

Your processing application should be able to be scaled horizontally, but this isn't something you need to actually do. The processing application should use `at-least-once` semantics where ever applicable.

You may write this in any languages you choose, but Go would be preferred.

You may use any data store of your choosing, with `sqlite` being one example. Like our own code, we expect the code structure to make it easy to switch data stores.

Please note that Google Pub/Sub is best effort ordering and we want to keep the latest scan. While the example scanner does not publish scans at a rate where this would be an issue, we expect the application to be able to handle extreme out of orderness. Consider what would happen if the application received a scan that is 24 hours old.

---

Please upload the code to a publicly accessible GitHub, GitLab or other public code repository account. This README file should be updated, briefly documenting your solution. Like our own code, we expect testing instructions: whether it’s an automated test framework, or simple manual steps.

To help set expectations, we believe you should aim to take no more than 4 hours on this task.

We understand that you have other responsibilities, so if you think you’ll need more than 5 business days, just let us know when you expect to send a reply.

Please don’t hesitate to ask any follow-up questions for clarification.

---

## Solution Overview

This repository now includes a processing service (`cmd/processor`) that consumes scan messages from the Pub/Sub emulator, normalizes the payload (handling both `data_version` `1` and `2`), and persists the freshest observation for each `(ip, port, service)` tuple in SQLite. The processor uses an environment-driven configuration layer and a Data Access Layer (`/dal`) abstraction so alternative pubsubs and datastores can be swapped in without changing business logic.

### Key pieces
- **Processor** (`cmd/processor`): Pulls from subscription `scan-sub`, decodes responses, and performs newest-wins upserts.
- **Data access layer** (`pkg/dal`): Defines the repository interface; `pkg/dal/sqlite` provides the default implementation using the Go SQLite driver.
- **Scanner** (`cmd/scanner`): Unchanged sample publisher that emits random scans to the emulator.
- **Configuration** (`pkg/config`): Reads environment variables (with sensible defaults matching `docker-compose.yml`).

### Configuration
All processor settings are controlled via environment variables (defaults align with `docker-compose.yml`):

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

2. Inspect stored results (optional):
   ```sh
   docker compose exec processor sqlite3 /data/mini_scan.db \
     'SELECT ip, port, service, observed_at, response FROM service_scans LIMIT 5;'
   ```

3. Tear everything down:
   ```sh
   docker compose down
   ```

### Unit tests

- Unit tests:
  ```sh
  GOCACHE=$(pwd)/.gocache \
  GOMODCACHE=$(pwd)/.gopath/pkg/mod \
  go test ./...
  ```