package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-manager/market/internal/domain/user"
	usersv1 "k8s-manager/proto/gen/v1/users"
)

// PostgresRepository implements user.Repository using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL user repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new user in the database
func (r *PostgresRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	query := `
		INSERT INTO users (name, type, role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var id int64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, u.Name, int32(u.Type), int32(u.Role)).
		Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	u.ID = id
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt

	return u, nil
}

// GetByID retrieves a user by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	query := `
		SELECT id, name, type, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	u := &user.User{}
	var typeVal, roleVal int32

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&typeVal,
		&roleVal,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	u.Type = usersv1.Type(typeVal)
	u.Role = usersv1.Role(roleVal)

	return u, nil
}

// List retrieves a list of users with pagination and optional search
func (r *PostgresRepository) List(ctx context.Context, limit, offset int32, query string) ([]*user.User, error) {
	var sqlQuery strings.Builder
	args := []interface{}{}
	argPos := 1

	sqlQuery.WriteString(`
		SELECT id, name, type, role, created_at, updated_at
		FROM users
	`)

	if query != "" {
		sqlQuery.WriteString(fmt.Sprintf(" WHERE name ILIKE $%d", argPos))
		args = append(args, "%"+query+"%")
		argPos++
	}

	sqlQuery.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlQuery.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]*user.User, 0)
	for rows.Next() {
		u := &user.User{}
		var typeVal, roleVal int32

		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&typeVal,
			&roleVal,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		u.Type = usersv1.Type(typeVal)
		u.Role = usersv1.Role(roleVal)

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// Count returns the total number of users matching the query
func (r *PostgresRepository) Count(ctx context.Context, query string) (int64, error) {
	var sqlQuery strings.Builder
	args := []interface{}{}
	argPos := 1

	sqlQuery.WriteString("SELECT COUNT(*) FROM users")

	if query != "" {
		sqlQuery.WriteString(fmt.Sprintf(" WHERE name ILIKE $%d", argPos))
		args = append(args, "%"+query+"%")
	}

	var count int64
	err := r.db.QueryRowContext(ctx, sqlQuery.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

