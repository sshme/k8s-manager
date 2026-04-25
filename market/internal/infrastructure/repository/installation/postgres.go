package installation

import (
	"context"
	"database/sql"
	"fmt"

	"k8s-manager/market/internal/domain/plugin"
)

// PostgresInstallationRepository implements InstallationRepository.
type PostgresInstallationRepository struct {
	db *sql.DB
}

// NewPostgresInstallationRepository creates a PostgreSQL installation repository.
func NewPostgresInstallationRepository(db *sql.DB) *PostgresInstallationRepository {
	return &PostgresInstallationRepository{db: db}
}

// Install marks a plugin as installed for a user.
func (r *PostgresInstallationRepository) Install(ctx context.Context, userID string, pluginID int64) (*plugin.PluginInstallation, error) {
	query := `
		INSERT INTO plugin_installations (user_id, plugin_id, installed_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, plugin_id)
		DO UPDATE SET installed_at = plugin_installations.installed_at
		RETURNING installed_at
	`

	installed := &plugin.PluginInstallation{
		UserID:   userID,
		PluginID: pluginID,
	}
	if err := r.db.QueryRowContext(ctx, query, userID, pluginID).Scan(&installed.InstalledAt); err != nil {
		return nil, fmt.Errorf("failed to install plugin: %w", err)
	}

	return installed, nil
}

// Uninstall removes a plugin from a user's installed plugin list.
func (r *PostgresInstallationRepository) Uninstall(ctx context.Context, userID string, pluginID int64) error {
	_, err := r.db.ExecContext(
		ctx,
		"DELETE FROM plugin_installations WHERE user_id = $1 AND plugin_id = $2",
		userID,
		pluginID,
	)
	if err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}
	return nil
}

// ListByUserID returns installed plugins for a user.
func (r *PostgresInstallationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*plugin.PluginInstallation, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM plugin_installations WHERE user_id = $1",
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count installed plugins: %w", err)
	}

	query := `
		SELECT
			i.user_id,
			i.plugin_id,
			i.installed_at,
			p.id,
			p.identifier,
			p.name,
			p.description,
			p.category,
			p.publisher_id,
			p.status,
			p.trust_status,
			p.source_url,
			p.docs_url,
			p.created_at,
			p.updated_at
		FROM plugin_installations i
		JOIN plugins p ON p.id = i.plugin_id
		WHERE i.user_id = $1
		ORDER BY i.installed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list installed plugins: %w", err)
	}
	defer rows.Close()

	var installations []*plugin.PluginInstallation
	for rows.Next() {
		installed := &plugin.PluginInstallation{Plugin: &plugin.Plugin{}}
		var status, trustStatus string
		if err := rows.Scan(
			&installed.UserID,
			&installed.PluginID,
			&installed.InstalledAt,
			&installed.Plugin.ID,
			&installed.Plugin.Identifier,
			&installed.Plugin.Name,
			&installed.Plugin.Description,
			&installed.Plugin.Category,
			&installed.Plugin.PublisherID,
			&status,
			&trustStatus,
			&installed.Plugin.SourceURL,
			&installed.Plugin.DocsURL,
			&installed.Plugin.CreatedAt,
			&installed.Plugin.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan installed plugin: %w", err)
		}
		installed.Plugin.Status = plugin.PluginStatus(status)
		installed.Plugin.TrustStatus = plugin.TrustStatus(trustStatus)
		installations = append(installations, installed)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate installed plugins: %w", err)
	}

	return installations, total, nil
}
