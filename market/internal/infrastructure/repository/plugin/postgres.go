package plugin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"k8s-manager/market/internal/domain/plugin"
)

// PostgresPluginRepository implements PluginRepository
type PostgresPluginRepository struct {
	db *sql.DB
}

// NewPostgresPluginRepository creates a new PostgreSQL plugin repository
func NewPostgresPluginRepository(db *sql.DB) *PostgresPluginRepository {
	return &PostgresPluginRepository{db: db}
}

// Create creates a new plugin
func (r *PostgresPluginRepository) Create(ctx context.Context, p *plugin.Plugin) (*plugin.Plugin, error) {
	query := `
		INSERT INTO plugins (identifier, name, description, category, publisher_id, status, trust_status, source_url, docs_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		p.Identifier, p.Name, p.Description, p.Category, p.PublisherID,
		string(p.Status), string(p.TrustStatus), p.SourceURL, p.DocsURL,
	).Scan(&p.ID, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt
	return p, nil
}

// GetByID retrieves a plugin by ID
func (r *PostgresPluginRepository) GetByID(ctx context.Context, id int64) (*plugin.Plugin, error) {
	query := `
		SELECT id, identifier, name, description, category, publisher_id, status, trust_status, source_url, docs_url, created_at, updated_at
		FROM plugins
		WHERE id = $1
	`

	p := &plugin.Plugin{}
	var statusStr, trustStatusStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Identifier, &p.Name, &p.Description, &p.Category, &p.PublisherID,
		&statusStr, &trustStatusStr, &p.SourceURL, &p.DocsURL, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	p.Status = plugin.PluginStatus(statusStr)
	p.TrustStatus = plugin.TrustStatus(trustStatusStr)
	return p, nil
}

// GetByIdentifier retrieves a plugin by identifier
func (r *PostgresPluginRepository) GetByIdentifier(ctx context.Context, identifier string) (*plugin.Plugin, error) {
	query := `
		SELECT id, identifier, name, description, category, publisher_id, status, trust_status, source_url, docs_url, created_at, updated_at
		FROM plugins
		WHERE identifier = $1
	`

	p := &plugin.Plugin{}
	var statusStr, trustStatusStr string
	err := r.db.QueryRowContext(ctx, query, identifier).Scan(
		&p.ID, &p.Identifier, &p.Name, &p.Description, &p.Category, &p.PublisherID,
		&statusStr, &trustStatusStr, &p.SourceURL, &p.DocsURL, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	p.Status = plugin.PluginStatus(statusStr)
	p.TrustStatus = plugin.TrustStatus(trustStatusStr)
	return p, nil
}

// List retrieves a list of plugins with filtering and pagination
func (r *PostgresPluginRepository) List(ctx context.Context, filter *plugin.PluginFilter, limit, offset int) ([]*plugin.Plugin, int64, error) {
	var conditions []string
	var args []interface{}
	argPos := 1

	if filter != nil {
		if filter.Name != "" {
			conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argPos))
			args = append(args, "%"+filter.Name+"%")
			argPos++
		}
		if filter.Query != "" {
			conditions = append(conditions, fmt.Sprintf("(identifier ILIKE $%d OR name ILIKE $%d OR description ILIKE $%d)", argPos, argPos, argPos))
			args = append(args, "%"+filter.Query+"%")
			argPos++
		}
		if filter.Category != "" {
			conditions = append(conditions, fmt.Sprintf("category = $%d", argPos))
			args = append(args, filter.Category)
			argPos++
		}
		if filter.PublisherID > 0 {
			conditions = append(conditions, fmt.Sprintf("publisher_id = $%d", argPos))
			args = append(args, filter.PublisherID)
			argPos++
		}
		if filter.TrustStatus != "" {
			conditions = append(conditions, fmt.Sprintf("trust_status = $%d", argPos))
			args = append(args, string(filter.TrustStatus))
			argPos++
		}
		if filter.Status != "" {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
			args = append(args, string(filter.Status))
			argPos++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM plugins " + whereClause
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count plugins: %w", err)
	}

	// Get plugins
	query := fmt.Sprintf(`
		SELECT id, identifier, name, description, category, publisher_id, status, trust_status, source_url, docs_url, created_at, updated_at
		FROM plugins
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plugins: %w", err)
	}
	defer rows.Close()

	var plugins []*plugin.Plugin
	for rows.Next() {
		p := &plugin.Plugin{}
		var statusStr, trustStatusStr string
		err := rows.Scan(
			&p.ID, &p.Identifier, &p.Name, &p.Description, &p.Category, &p.PublisherID,
			&statusStr, &trustStatusStr, &p.SourceURL, &p.DocsURL, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan plugin: %w", err)
		}
		p.Status = plugin.PluginStatus(statusStr)
		p.TrustStatus = plugin.TrustStatus(trustStatusStr)
		plugins = append(plugins, p)
	}

	return plugins, total, nil
}

// Update updates a plugin
func (r *PostgresPluginRepository) Update(ctx context.Context, p *plugin.Plugin) (*plugin.Plugin, error) {
	query := `
		UPDATE plugins
		SET name = $2, description = $3, category = $4, source_url = $5, docs_url = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		p.ID, p.Name, p.Description, p.Category, p.SourceURL, p.DocsURL,
	).Scan(&p.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update plugin: %w", err)
	}

	return p, nil
}

// UpdateStatus updates plugin status
func (r *PostgresPluginRepository) UpdateStatus(ctx context.Context, id int64, status plugin.PluginStatus, reason string) error {
	query := `
		UPDATE plugins
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, string(status))
	if err != nil {
		return fmt.Errorf("failed to update plugin status: %w", err)
	}

	// Log to audit
	auditQuery := `
		INSERT INTO audit_log (entity_type, entity_id, action, reason, new_value, created_at)
		VALUES ('plugin', $1, 'status_change', $2, $3, NOW())
	`
	_, _ = r.db.ExecContext(ctx, auditQuery, id, reason, fmt.Sprintf(`{"status": "%s"}`, status))

	return nil
}

// UpdateTrustStatus updates plugin trust status (community/verified/official)
func (r *PostgresPluginRepository) UpdateTrustStatus(ctx context.Context, id int64, trustStatus plugin.TrustStatus, reason string) error {
	query := `
		UPDATE plugins
		SET trust_status = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, string(trustStatus))
	if err != nil {
		return fmt.Errorf("failed to update plugin trust status: %w", err)
	}

	auditQuery := `
		INSERT INTO audit_log (entity_type, entity_id, action, reason, new_value, created_at)
		VALUES ('plugin', $1, 'trust_status_change', $2, $3, NOW())
	`
	_, _ = r.db.ExecContext(ctx, auditQuery, id, reason, fmt.Sprintf(`{"trust_status": "%s"}`, trustStatus))

	return nil
}
