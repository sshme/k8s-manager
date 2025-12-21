package audit

import (
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         int64
	EntityType string // 'plugin', 'release', 'artifact'
	EntityID   int64
	Action     string // 'create', 'update', 'delete', 'status_change'
	UserID     string
	Reason     string
	OldValue   string // JSON
	NewValue   string // JSON
	CreatedAt  time.Time
}

