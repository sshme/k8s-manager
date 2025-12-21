package user

import (
	"time"

	usersv1 "k8s-manager/proto/gen/v1/users"
)

// User represents a user entity in the domain
type User struct {
	ID        int64
	Name      string
	Type      usersv1.Type
	Role      usersv1.Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ToProto converts domain User to proto User
func (u *User) ToProto() *usersv1.User {
	return &usersv1.User{
		Id:        u.ID,
		Name:      u.Name,
		Type:      u.Type,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

// FromProto creates domain User from proto User
func FromProto(p *usersv1.User) (*User, error) {
	createdAt, err := time.Parse(time.RFC3339, p.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        p.Id,
		Name:      p.Name,
		Type:      p.Type,
		Role:      p.Role,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

