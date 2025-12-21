package publisher

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s-manager/market/internal/domain/plugin"
)

// PostgresPublisherRepository implements PublisherRepository
type PostgresPublisherRepository struct {
	db *sql.DB
}

// NewPostgresPublisherRepository creates a new PostgreSQL publisher repository
func NewPostgresPublisherRepository(db *sql.DB) *PostgresPublisherRepository {
	return &PostgresPublisherRepository{db: db}
}

// Create creates a new publisher
func (r *PostgresPublisherRepository) Create(ctx context.Context, pub *plugin.Publisher) (*plugin.Publisher, error) {
	query := `
		INSERT INTO publishers (name, description, website_url, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		pub.Name, pub.Description, pub.WebsiteURL,
	).Scan(&pub.ID, &createdAt, &updatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}
	
	pub.CreatedAt = createdAt
	pub.UpdatedAt = updatedAt
	return pub, nil
}

// GetByID retrieves a publisher by ID
func (r *PostgresPublisherRepository) GetByID(ctx context.Context, id int64) (*plugin.Publisher, error) {
	query := `
		SELECT id, name, description, website_url, created_at, updated_at
		FROM publishers
		WHERE id = $1
	`
	
	pub := &plugin.Publisher{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pub.ID, &pub.Name, &pub.Description, &pub.WebsiteURL, &pub.CreatedAt, &pub.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}
	
	return pub, nil
}

// GetByName retrieves a publisher by name
func (r *PostgresPublisherRepository) GetByName(ctx context.Context, name string) (*plugin.Publisher, error) {
	query := `
		SELECT id, name, description, website_url, created_at, updated_at
		FROM publishers
		WHERE name = $1
	`
	
	pub := &plugin.Publisher{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&pub.ID, &pub.Name, &pub.Description, &pub.WebsiteURL, &pub.CreatedAt, &pub.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}
	
	return pub, nil
}

// List retrieves a list of publishers
func (r *PostgresPublisherRepository) List(ctx context.Context, limit, offset int) ([]*plugin.Publisher, int64, error) {
	// Count total
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM publishers").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count publishers: %w", err)
	}
	
	// Get publishers
	query := `
		SELECT id, name, description, website_url, created_at, updated_at
		FROM publishers
		ORDER BY name
		LIMIT $1 OFFSET $2
	`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list publishers: %w", err)
	}
	defer rows.Close()
	
	var publishers []*plugin.Publisher
	for rows.Next() {
		pub := &plugin.Publisher{}
		err := rows.Scan(
			&pub.ID, &pub.Name, &pub.Description, &pub.WebsiteURL, &pub.CreatedAt, &pub.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan publisher: %w", err)
		}
		publishers = append(publishers, pub)
	}
	
	return publishers, total, nil
}

