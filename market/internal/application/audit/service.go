package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s-manager/market/internal/domain/audit"
)

// Service handles audit logging
type Service struct {
	auditRepo audit.AuditRepository
}

// NewService creates a new audit service
func NewService(auditRepo audit.AuditRepository) *Service {
	return &Service{
		auditRepo: auditRepo,
	}
}

// LogCreate logs a create action
func (s *Service) LogCreate(ctx context.Context, entityType string, entityID int64, userID string, newValue interface{}) error {
	return s.auditRepo.Create(ctx, &audit.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "create",
		UserID:     userID,
		NewValue:   mustJSON(newValue),
	})
}

// LogUpdate logs an update action
func (s *Service) LogUpdate(ctx context.Context, entityType string, entityID int64, userID string, oldValue, newValue interface{}) error {
	return s.auditRepo.Create(ctx, &audit.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "update",
		UserID:     userID,
		OldValue:   mustJSON(oldValue),
		NewValue:   mustJSON(newValue),
	})
}

// LogDelete logs a delete action
func (s *Service) LogDelete(ctx context.Context, entityType string, entityID int64, userID string, reason string, oldValue interface{}) error {
	return s.auditRepo.Create(ctx, &audit.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "delete",
		UserID:     userID,
		Reason:     reason,
		OldValue:   mustJSON(oldValue),
	})
}

// LogStatusChange logs a status change action
func (s *Service) LogStatusChange(ctx context.Context, entityType string, entityID int64, userID string, reason string, oldStatus, newStatus string) error {
	return s.auditRepo.Create(ctx, &audit.AuditLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "status_change",
		UserID:     userID,
		Reason:     reason,
		OldValue:   `{"status": "` + oldStatus + `"}`,
		NewValue:   `{"status": "` + newStatus + `"}`,
	})
}

// ListAuditLogs retrieves audit logs for an entity
func (s *Service) ListAuditLogs(ctx context.Context, entityType string, entityID int64, limit, offset int) ([]*audit.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, entityType, entityID, limit, offset)
}

func mustJSON(value interface{}) string {
	if value == nil {
		return ""
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"marshal_error":%q}`, err.Error())
	}
	return string(payload)
}
