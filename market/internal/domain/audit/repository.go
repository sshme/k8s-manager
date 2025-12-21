package audit

import (
	"context"
)

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, entityType string, entityID int64, limit, offset int) ([]*AuditLog, int64, error)
}

