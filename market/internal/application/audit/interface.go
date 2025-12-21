package audit

import (
	"context"
)

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogCreate(ctx context.Context, entityType string, entityID int64, userID string, newValue interface{}) error
	LogUpdate(ctx context.Context, entityType string, entityID int64, userID string, oldValue, newValue interface{}) error
	LogDelete(ctx context.Context, entityType string, entityID int64, userID string, reason string, oldValue interface{}) error
	LogStatusChange(ctx context.Context, entityType string, entityID int64, userID string, reason string, oldStatus, newStatus string) error
}

