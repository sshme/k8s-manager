package release

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s-manager/market/internal/domain/plugin"
)

// PostgresReleaseRepository implements ReleaseRepository
type PostgresReleaseRepository struct {
	db *sql.DB
}

// NewPostgresReleaseRepository creates a new PostgreSQL release repository
func NewPostgresReleaseRepository(db *sql.DB) *PostgresReleaseRepository {
	return &PostgresReleaseRepository{db: db}
}

// Create creates a new release
func (r *PostgresReleaseRepository) Create(ctx context.Context, rel *plugin.Release) (*plugin.Release, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Check if version already exists
	var existingID int64
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM releases WHERE plugin_id = $1 AND version = $2",
		rel.PluginID, rel.Version,
	).Scan(&existingID)
	
	if err == nil {
		return nil, fmt.Errorf("release with version %s already exists for this plugin", rel.Version)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing release: %w", err)
	}
	
	// Insert release
	query := `
		INSERT INTO releases (plugin_id, version, published_at, changelog, min_cli_version, min_k8s_version, is_latest, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	
	var createdAt, updatedAt time.Time
	err = tx.QueryRowContext(ctx, query,
		rel.PluginID, rel.Version, rel.PublishedAt, rel.Changelog,
		rel.MinCLIVersion, rel.MinK8sVersion, rel.IsLatest,
	).Scan(&rel.ID, &createdAt, &updatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}
	
	rel.CreatedAt = createdAt
	rel.UpdatedAt = updatedAt
	
	// If this is the latest release, unset other latest flags
	if rel.IsLatest {
		_, err = tx.ExecContext(ctx,
			"UPDATE releases SET is_latest = FALSE WHERE plugin_id = $1 AND id != $2",
			rel.PluginID, rel.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update latest flags: %w", err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return rel, nil
}

// GetByID retrieves a release by ID
func (r *PostgresReleaseRepository) GetByID(ctx context.Context, id int64) (*plugin.Release, error) {
	query := `
		SELECT id, plugin_id, version, published_at, changelog, min_cli_version, min_k8s_version, is_latest, created_at, updated_at
		FROM releases
		WHERE id = $1
	`
	
	rel := &plugin.Release{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID, &rel.PluginID, &rel.Version, &rel.PublishedAt, &rel.Changelog,
		&rel.MinCLIVersion, &rel.MinK8sVersion, &rel.IsLatest, &rel.CreatedAt, &rel.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}
	
	return rel, nil
}

// GetByPluginIDAndVersion retrieves a release by plugin ID and version
func (r *PostgresReleaseRepository) GetByPluginIDAndVersion(ctx context.Context, pluginID int64, version string) (*plugin.Release, error) {
	query := `
		SELECT id, plugin_id, version, published_at, changelog, min_cli_version, min_k8s_version, is_latest, created_at, updated_at
		FROM releases
		WHERE plugin_id = $1 AND version = $2
	`
	
	rel := &plugin.Release{}
	err := r.db.QueryRowContext(ctx, query, pluginID, version).Scan(
		&rel.ID, &rel.PluginID, &rel.Version, &rel.PublishedAt, &rel.Changelog,
		&rel.MinCLIVersion, &rel.MinK8sVersion, &rel.IsLatest, &rel.CreatedAt, &rel.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}
	
	return rel, nil
}

// ListByPluginID retrieves releases for a plugin
func (r *PostgresReleaseRepository) ListByPluginID(ctx context.Context, pluginID int64, limit, offset int) ([]*plugin.Release, int64, error) {
	// Count total
	var total int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM releases WHERE plugin_id = $1",
		pluginID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count releases: %w", err)
	}
	
	// Get releases
	query := `
		SELECT id, plugin_id, version, published_at, changelog, min_cli_version, min_k8s_version, is_latest, created_at, updated_at
		FROM releases
		WHERE plugin_id = $1
		ORDER BY published_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.QueryContext(ctx, query, pluginID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list releases: %w", err)
	}
	defer rows.Close()
	
	var releases []*plugin.Release
	for rows.Next() {
		rel := &plugin.Release{}
		err := rows.Scan(
			&rel.ID, &rel.PluginID, &rel.Version, &rel.PublishedAt, &rel.Changelog,
			&rel.MinCLIVersion, &rel.MinK8sVersion, &rel.IsLatest, &rel.CreatedAt, &rel.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, rel)
	}
	
	return releases, total, nil
}

// SetLatest sets a release as the latest for a plugin
func (r *PostgresReleaseRepository) SetLatest(ctx context.Context, pluginID int64, releaseID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Unset all latest flags for this plugin
	_, err = tx.ExecContext(ctx,
		"UPDATE releases SET is_latest = FALSE WHERE plugin_id = $1",
		pluginID,
	)
	if err != nil {
		return fmt.Errorf("failed to unset latest flags: %w", err)
	}
	
	// Set this release as latest
	_, err = tx.ExecContext(ctx,
		"UPDATE releases SET is_latest = TRUE WHERE id = $1",
		releaseID,
	)
	if err != nil {
		return fmt.Errorf("failed to set latest: %w", err)
	}
	
	return tx.Commit()
}

// GetLatest retrieves the latest release for a plugin
func (r *PostgresReleaseRepository) GetLatest(ctx context.Context, pluginID int64) (*plugin.Release, error) {
	query := `
		SELECT id, plugin_id, version, published_at, changelog, min_cli_version, min_k8s_version, is_latest, created_at, updated_at
		FROM releases
		WHERE plugin_id = $1 AND is_latest = TRUE
		ORDER BY published_at DESC
		LIMIT 1
	`
	
	rel := &plugin.Release{}
	err := r.db.QueryRowContext(ctx, query, pluginID).Scan(
		&rel.ID, &rel.PluginID, &rel.Version, &rel.PublishedAt, &rel.Changelog,
		&rel.MinCLIVersion, &rel.MinK8sVersion, &rel.IsLatest, &rel.CreatedAt, &rel.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	
	return rel, nil
}

