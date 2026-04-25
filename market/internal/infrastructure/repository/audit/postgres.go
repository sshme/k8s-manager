package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s-manager/market/internal/domain/audit"
)

// PostgresAuditRepository implements AuditRepository
type PostgresAuditRepository struct {
	db *sql.DB
}

// NewPostgresAuditRepository creates a new PostgreSQL audit repository
func NewPostgresAuditRepository(db *sql.DB) *PostgresAuditRepository {
	return &PostgresAuditRepository{db: db}
}

// Create creates a new audit log entry
func (r *PostgresAuditRepository) Create(ctx context.Context, log *audit.AuditLog) error {
	query := `
		INSERT INTO audit_log (entity_type, entity_id, action, user_id, reason, old_value, new_value, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`

	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		log.EntityType, log.EntityID, log.Action, log.UserID,
		nullableString(log.Reason), nullableJSON(log.OldValue), nullableJSON(log.NewValue),
	).Scan(&log.ID, &createdAt)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	log.CreatedAt = createdAt
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableJSON(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// List retrieves audit logs for an entity
func (r *PostgresAuditRepository) List(ctx context.Context, entityType string, entityID int64, limit, offset int) ([]*audit.AuditLog, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*) FROM audit_log
		WHERE entity_type = $1 AND entity_id = $2
	`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, entityType, entityID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get audit logs
	query := `
		SELECT id, entity_type, entity_id, action, user_id, reason, old_value, new_value, created_at
		FROM audit_log
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*audit.AuditLog
	for rows.Next() {
		log := &audit.AuditLog{}
		var oldValue, newValue sql.NullString
		err := rows.Scan(
			&log.ID, &log.EntityType, &log.EntityID, &log.Action,
			&log.UserID, &log.Reason, &oldValue, &newValue, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if oldValue.Valid {
			log.OldValue = oldValue.String
		}
		if newValue.Valid {
			log.NewValue = newValue.String
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}
