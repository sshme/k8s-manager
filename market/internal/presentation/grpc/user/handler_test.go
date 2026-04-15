package user

import (
	"context"
	"errors"
	"testing"
	"time"

	appuser "k8s-manager/market/internal/application/user"
	domainuser "k8s-manager/market/internal/domain/user"
	usersv1 "k8s-manager/proto/gen/v1/users"
)

// Since Handler uses concrete *user.Service type, we need a different approach
// Let's create tests that work with the actual service but test the handler's delegation

func TestHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name           string
		req            *usersv1.CreateUserRequest
		setupMock      func(*mockRepository)
		expectedError  bool
		validateResult func(*testing.T, *usersv1.CreateUserResponse)
	}{
		{
			name: "successful user creation",
			req: &usersv1.CreateUserRequest{
				Name: "testuser",
			},
			setupMock: func(m *mockRepository) {
				m.createFunc = func(ctx context.Context, u *domainuser.User) (*domainuser.User, error) {
					now := time.Now()
					return &domainuser.User{
						ID:        1,
						Name:      u.Name,
						Type:      u.Type,
						Role:      u.Role,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, resp *usersv1.CreateUserResponse) {
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
				if resp.User == nil {
					t.Fatal("expected non-nil user")
				}
				if resp.User.Name != "testuser" {
					t.Errorf("expected user name 'testuser', got '%s'", resp.User.Name)
				}
				if resp.User.Type != usersv1.Type_USER {
					t.Errorf("expected user type USER, got %v", resp.User.Type)
				}
			},
		},
		{
			name: "service returns error",
			req: &usersv1.CreateUserRequest{
				Name: "testuser",
			},
			setupMock: func(m *mockRepository) {
				m.createFunc = func(ctx context.Context, u *domainuser.User) (*domainuser.User, error) {
					return nil, errors.New("repository error")
				}
			},
			expectedError: true,
		},
		{
			name: "empty name returns error",
			req: &usersv1.CreateUserRequest{
				Name: "",
			},
			setupMock: func(m *mockRepository) {
				// Repository should not be called for invalid input
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			service := appuser.NewService(mockRepo)
			handler := NewHandler(service)
			ctx := context.Background()

			resp, err := handler.CreateUser(ctx, tt.req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, resp)
				}
			}
		})
	}
}

func TestHandler_GetUser(t *testing.T) {
	tests := []struct {
		name           string
		req            *usersv1.GetUserRequest
		setupMock      func(*mockRepository)
		expectedError  bool
		validateResult func(*testing.T, *usersv1.GetUserResponse)
	}{
		{
			name: "successful user retrieval",
			req: &usersv1.GetUserRequest{
				Id: "1",
			},
			setupMock: func(m *mockRepository) {
				m.getByIDFunc = func(ctx context.Context, id int64) (*domainuser.User, error) {
					now := time.Now()
					return &domainuser.User{
						ID:        id,
						Name:      "testuser",
						Type:      usersv1.Type_USER,
						Role:      usersv1.Role_BASE,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, resp *usersv1.GetUserResponse) {
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
				if resp.User == nil {
					t.Fatal("expected non-nil user")
				}
				if resp.User.Id != 1 {
					t.Errorf("expected user id 1, got %d", resp.User.Id)
				}
				if resp.User.Name != "testuser" {
					t.Errorf("expected user name 'testuser', got '%s'", resp.User.Name)
				}
			},
		},
		{
			name: "user not found",
			req: &usersv1.GetUserRequest{
				Id: "999",
			},
			setupMock: func(m *mockRepository) {
				m.getByIDFunc = func(ctx context.Context, id int64) (*domainuser.User, error) {
					return nil, nil
				}
			},
			expectedError: true,
		},
		{
			name: "invalid user id",
			req: &usersv1.GetUserRequest{
				Id: "invalid",
			},
			setupMock: func(m *mockRepository) {
				m.getByIDFunc = func(ctx context.Context, id int64) (*domainuser.User, error) {
					return nil, nil
				}
			},
			expectedError: true,
		},
		{
			name: "repository error",
			req: &usersv1.GetUserRequest{
				Id: "1",
			},
			setupMock: func(m *mockRepository) {
				m.getByIDFunc = func(ctx context.Context, id int64) (*domainuser.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			service := appuser.NewService(mockRepo)
			handler := NewHandler(service)
			ctx := context.Background()

			resp, err := handler.GetUser(ctx, tt.req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, resp)
				}
			}
		})
	}
}

func TestHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name           string
		req            *usersv1.ListUsersRequest
		setupMock      func(*mockRepository)
		expectedError  bool
		validateResult func(*testing.T, *usersv1.ListUsersResponse)
	}{
		{
			name: "successful list users",
			req: &usersv1.ListUsersRequest{
				Limit:      10,
				PageNumber: 1,
				Query:      "",
			},
			setupMock: func(m *mockRepository) {
				m.listFunc = func(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error) {
					now := time.Now()
					return []*domainuser.User{
						{
							ID:        1,
							Name:      "user1",
							Type:      usersv1.Type_USER,
							Role:      usersv1.Role_BASE,
							CreatedAt: now,
							UpdatedAt: now,
						},
						{
							ID:        2,
							Name:      "user2",
							Type:      usersv1.Type_USER,
							Role:      usersv1.Role_ADMIN,
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, resp *usersv1.ListUsersResponse) {
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
				if len(resp.Users) != 2 {
					t.Errorf("expected 2 users, got %d", len(resp.Users))
				}
				if resp.Users[0].Name != "user1" {
					t.Errorf("expected first user name 'user1', got '%s'", resp.Users[0].Name)
				}
				if resp.Users[1].Name != "user2" {
					t.Errorf("expected second user name 'user2', got '%s'", resp.Users[1].Name)
				}
			},
		},
		{
			name: "empty list",
			req: &usersv1.ListUsersRequest{
				Limit:      10,
				PageNumber: 1,
				Query:      "",
			},
			setupMock: func(m *mockRepository) {
				m.listFunc = func(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error) {
					return []*domainuser.User{}, nil
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, resp *usersv1.ListUsersResponse) {
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
				if len(resp.Users) != 0 {
					t.Errorf("expected 0 users, got %d", len(resp.Users))
				}
			},
		},
		{
			name: "default limit when limit is 0",
			req: &usersv1.ListUsersRequest{
				Limit:      0,
				PageNumber: 1,
				Query:      "",
			},
			setupMock: func(m *mockRepository) {
				m.listFunc = func(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error) {
					if limit != 20 {
						t.Errorf("expected default limit 20, got %d", limit)
					}
					return []*domainuser.User{}, nil
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, resp *usersv1.ListUsersResponse) {
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
			},
		},
		{
			name: "repository error",
			req: &usersv1.ListUsersRequest{
				Limit:      10,
				PageNumber: 1,
				Query:      "",
			},
			setupMock: func(m *mockRepository) {
				m.listFunc = func(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			service := appuser.NewService(mockRepo)
			handler := NewHandler(service)
			ctx := context.Background()

			resp, err := handler.ListUsers(ctx, tt.req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, resp)
				}
			}
		})
	}
}

// mockRepository is a mock implementation of domainuser.Repository for testing
type mockRepository struct {
	createFunc  func(ctx context.Context, u *domainuser.User) (*domainuser.User, error)
	getByIDFunc func(ctx context.Context, id int64) (*domainuser.User, error)
	listFunc    func(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error)
	countFunc   func(ctx context.Context, query string) (int64, error)
}

func (m *mockRepository) Create(ctx context.Context, u *domainuser.User) (*domainuser.User, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, u)
	}
	return nil, errors.New("createFunc not set")
}

func (m *mockRepository) GetByID(ctx context.Context, id int64) (*domainuser.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("getByIDFunc not set")
}

func (m *mockRepository) List(ctx context.Context, limit, offset int32, query string) ([]*domainuser.User, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset, query)
	}
	return nil, errors.New("listFunc not set")
}

func (m *mockRepository) Count(ctx context.Context, query string) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, query)
	}
	return 0, errors.New("countFunc not set")
}
