package artifact

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s-manager/market/internal/domain/plugin"
)

// PostgresArtifactRepository implements ArtifactRepository
type PostgresArtifactRepository struct {
	db *sql.DB
}

// NewPostgresArtifactRepository creates a new PostgreSQL artifact repository
func NewPostgresArtifactRepository(db *sql.DB) *PostgresArtifactRepository {
	return &PostgresArtifactRepository{db: db}
}

// Create creates a new artifact
func (r *PostgresArtifactRepository) Create(ctx context.Context, art *plugin.Artifact) (*plugin.Artifact, error) {
	query := `
		INSERT INTO artifacts (release_id, os, arch, type, size, checksum, storage_path, signature, key_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id, created_at
	`
	
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		art.ReleaseID, art.OS, art.Arch, art.Type, art.Size,
		art.Checksum, art.StoragePath, art.Signature, art.KeyID,
	).Scan(&art.ID, &createdAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}
	
	art.CreatedAt = createdAt
	return art, nil
}

// GetByID retrieves an artifact by ID
func (r *PostgresArtifactRepository) GetByID(ctx context.Context, id int64) (*plugin.Artifact, error) {
	query := `
		SELECT id, release_id, os, arch, type, size, checksum, storage_path, signature, key_id, created_at
		FROM artifacts
		WHERE id = $1
	`
	
	art := &plugin.Artifact{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&art.ID, &art.ReleaseID, &art.OS, &art.Arch, &art.Type, &art.Size,
		&art.Checksum, &art.StoragePath, &art.Signature, &art.KeyID, &art.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}
	
	return art, nil
}

// GetByReleaseIDAndPlatform retrieves an artifact by release ID and platform
func (r *PostgresArtifactRepository) GetByReleaseIDAndPlatform(ctx context.Context, releaseID int64, os, arch string) (*plugin.Artifact, error) {
	query := `
		SELECT id, release_id, os, arch, type, size, checksum, storage_path, signature, key_id, created_at
		FROM artifacts
		WHERE release_id = $1 AND os = $2 AND arch = $3
	`
	
	art := &plugin.Artifact{}
	err := r.db.QueryRowContext(ctx, query, releaseID, os, arch).Scan(
		&art.ID, &art.ReleaseID, &art.OS, &art.Arch, &art.Type, &art.Size,
		&art.Checksum, &art.StoragePath, &art.Signature, &art.KeyID, &art.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}
	
	return art, nil
}

// ListByReleaseID retrieves all artifacts for a release
func (r *PostgresArtifactRepository) ListByReleaseID(ctx context.Context, releaseID int64) ([]*plugin.Artifact, error) {
	query := `
		SELECT id, release_id, os, arch, type, size, checksum, storage_path, signature, key_id, created_at
		FROM artifacts
		WHERE release_id = $1
		ORDER BY os, arch
	`
	
	rows, err := r.db.QueryContext(ctx, query, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()
	
	var artifacts []*plugin.Artifact
	for rows.Next() {
		art := &plugin.Artifact{}
		err := rows.Scan(
			&art.ID, &art.ReleaseID, &art.OS, &art.Arch, &art.Type, &art.Size,
			&art.Checksum, &art.StoragePath, &art.Signature, &art.KeyID, &art.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, art)
	}
	
	return artifacts, nil
}

// Delete deletes an artifact
func (r *PostgresArtifactRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM artifacts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}
	return nil
}

