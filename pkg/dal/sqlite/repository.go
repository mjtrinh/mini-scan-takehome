package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/censys/scan-takehome/pkg/dal"
	"github.com/censys/scan-takehome/pkg/processor"
)

// Repository persists ServiceScan records in SQLite.
type Repository struct {
	db *sql.DB
}

var _ dal.Repository = (*Repository)(nil)

// New opens (or creates) the SQLite database at the provided path and ensures
// the schema exists.
func New(path string) (*Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path must not be empty")
	}

	if err := ensureDir(path); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("error setting journal mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("error setting busy timeout: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Repository{db: db}, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func initSchema(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS service_scans (
	ip TEXT NOT NULL,
	port INTEGER NOT NULL,
	service TEXT NOT NULL,
	observed_at INTEGER NOT NULL,
	response TEXT NOT NULL,
	message_id TEXT NOT NULL,
	PRIMARY KEY (ip, port, service)
);
`
	_, err := db.Exec(ddl)
	return err
}

// UpsertLatest inserts or updates the stored scan if the incoming observation
// is newer than what is currently recorded. It returns true when the stored
// record changed.
func (r *Repository) UpsertLatest(ctx context.Context, scan *processor.ServiceScan) (bool, error) {
	if scan == nil {
		return false, fmt.Errorf("scan must not be nil")
	}

	observed := scan.ObservedAt.UTC().Unix()
	// `excluded` is sqlite's automatic alias for row that triggers a conflict
	query := `
INSERT INTO service_scans (ip, port, service, observed_at, response, message_id)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(ip, port, service)
DO UPDATE SET
	observed_at = excluded.observed_at,
	response = excluded.response,
	message_id = excluded.message_id
WHERE excluded.observed_at > service_scans.observed_at;
`

	res, err := r.db.ExecContext(ctx, query,
		scan.IP,
		scan.Port,
		scan.Service,
		observed,
		scan.Response,
		scan.MessageID,
	)
	if err != nil {
		return false, fmt.Errorf("error during upsert exec: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error counting rows affected: %w", err)
	}

	return rows > 0, nil
}

// Close releases the underlying database resources.
func (r *Repository) Close() error {
	return r.db.Close()
}

// Fetch retrieves a stored scan for testing or inspection purposes.
func (r *Repository) Fetch(ctx context.Context, ip string, port uint32, service string) (*processor.ServiceScan, error) {
	query := `
SELECT ip, port, service, observed_at, response, message_id
FROM service_scans
WHERE ip = ? AND port = ? AND service = ?;
`
	row := r.db.QueryRowContext(ctx, query, ip, port, service)

	var (
		result   processor.ServiceScan
		observed int64
	)
	if err := row.Scan(&result.IP, &result.Port, &result.Service, &observed, &result.Response, &result.MessageID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error fetching scan: %w", err)
	}
	result.ObservedAt = time.Unix(observed, 0).UTC()
	return &result, nil
}
