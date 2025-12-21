package user

import (
	"context"
	"errors"
	"fmt"

	"k8s-manager/market/internal/domain/user"
	usersv1 "k8s-manager/proto/gen/v1/users"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Service handles user business logic
type Service struct {
	repo user.Repository
}

// NewService creates a new user service
func NewService(repo user.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, req *usersv1.CreateUserRequest) (*usersv1.CreateUserResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	domainUser := &user.User{
		Name: req.Name,
		Type: usersv1.Type_USER,
		Role: usersv1.Role_BASE,
	}

	created, err := s.repo.Create(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &usersv1.CreateUserResponse{
		User: created.ToProto(),
	}, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	var id int64
	if _, err := fmt.Sscanf(req.Id, "%d", &id); err != nil {
		return nil, fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}

	domainUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if domainUser == nil {
		return nil, ErrUserNotFound
	}

	return &usersv1.GetUserResponse{
		User: domainUser.ToProto(),
	}, nil
}

// ListUsers retrieves a list of users with pagination
func (s *Service) ListUsers(ctx context.Context, req *usersv1.ListUsersRequest) (*usersv1.ListUsersResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20 // default limit
	}

	pageNumber := req.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}

	offset := (pageNumber - 1) * limit

	users, err := s.repo.List(ctx, limit, offset, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	protoUsers := make([]*usersv1.User, 0, len(users))
	for _, u := range users {
		protoUsers = append(protoUsers, u.ToProto())
	}

	return &usersv1.ListUsersResponse{
		Users: protoUsers,
	}, nil
}
