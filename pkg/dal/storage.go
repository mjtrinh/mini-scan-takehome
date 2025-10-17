package dal

import (
	"context"

	"github.com/censys/scan-takehome/pkg/processor"
)

// Repository defines the minimal contract required by the processor to persist scans.
type Repository interface {
	UpsertLatest(ctx context.Context, scan *processor.ServiceScan) (bool, error)
	Close() error
}
