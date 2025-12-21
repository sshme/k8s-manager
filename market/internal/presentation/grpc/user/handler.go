package user

import (
	"context"

	"k8s-manager/market/internal/application/user"
	usersv1 "k8s-manager/proto/gen/v1/users"
)

// Handler implements the gRPC UserService
type Handler struct {
	usersv1.UnimplementedUserServiceServer
	service *user.Service
}

// NewHandler creates a new gRPC user handler
func NewHandler(service *user.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateUser handles CreateUser gRPC request
func (h *Handler) CreateUser(ctx context.Context, req *usersv1.CreateUserRequest) (*usersv1.CreateUserResponse, error) {
	return h.service.CreateUser(ctx, req)
}

// GetUser handles GetUser gRPC request
func (h *Handler) GetUser(ctx context.Context, req *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	return h.service.GetUser(ctx, req)
}

// ListUsers handles ListUsers gRPC request
func (h *Handler) ListUsers(ctx context.Context, req *usersv1.ListUsersRequest) (*usersv1.ListUsersResponse, error) {
	return h.service.ListUsers(ctx, req)
}

