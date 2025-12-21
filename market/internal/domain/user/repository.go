package user

import (
	"context"
)

// Repository defines the interface for user persistence
type Repository interface {
	Create(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context, limit, offset int32, query string) ([]*User, error)
	Count(ctx context.Context, query string) (int64, error)
}

