package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RunMigrations applies SQL migrations from dir once, in filename order.
func RunMigrations(ctx context.Context, db *sql.DB, dir string) error {
	if dir == "" {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("find migrations: %w", err)
	}
	sort.Strings(files)

	if len(files) == 0 {
		return nil
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, file := range files {
		name := filepath.Base(file)

		applied, err := migrationApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applyMigration(ctx, db, name, file); err != nil {
			return err
		}
	}

	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var appliedAt time.Time
	err := db.QueryRowContext(ctx, "SELECT applied_at FROM schema_migrations WHERE name = $1", name).Scan(&appliedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return true, nil
}

func applyMigration(ctx context.Context, db *sql.DB, name, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}

	return nil
}
