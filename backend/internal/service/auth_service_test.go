package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// mockUserRepository は UserRepository のモック
type mockUserRepository struct {
	findByGoogleIDFunc  func(ctx context.Context, googleID string) (*model.User, error)
	findByEmailFunc     func(ctx context.Context, email string) (*model.User, error)
	createFunc          func(ctx context.Context, user *model.User) error
	updateProviderIDErr error
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindByGitHubID(ctx context.Context, githubID string) (*model.User, error) {
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	if m.findByGoogleIDFunc != nil {
		return m.findByGoogleIDFunc(ctx, googleID)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) UpdateProviderID(ctx context.Context, userID, column, value string) error {
	return m.updateProviderIDErr
}

func (m *mockUserRepository) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return nil, nil
}

func (m *mockUserRepository) Suspend(ctx context.Context, id string, suspend bool) error {
	return nil
}

func TestAuthService_GetOrCreateUserFromGoogle_ExistingUser(t *testing.T) {
	ctx := context.Background()
	existingUser := &model.User{ID: "1", Email: "a@example.com", GoogleID: "google-123", Name: "A"}

	mock := &mockUserRepository{
		findByGoogleIDFunc: func(ctx context.Context, googleID string) (*model.User, error) {
			if googleID == "google-123" {
				return existingUser, nil
			}
			return nil, nil
		},
	}

	svc := NewAuthService(mock)
	info := &GoogleUserInfo{Sub: "google-123", Email: "a@example.com", Name: "A"}

	u, err := svc.GetOrCreateUserFromGoogle(ctx, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != existingUser.ID {
		t.Errorf("expected ID %q, got %q", existingUser.ID, u.ID)
	}
}

func TestAuthService_GetOrCreateUserFromGoogle_NewUser(t *testing.T) {
	ctx := context.Background()
	var createdUser *model.User

	mock := &mockUserRepository{
		findByGoogleIDFunc: func(ctx context.Context, googleID string) (*model.User, error) {
			return nil, errors.New("not found") // 存在しない
		},
		createFunc: func(ctx context.Context, user *model.User) error {
			user.ID = "new-id"
			createdUser = user
			return nil
		},
	}

	svc := NewAuthService(mock)
	info := &GoogleUserInfo{Sub: "google-456", Email: "b@example.com", Name: "B"}

	u, err := svc.GetOrCreateUserFromGoogle(ctx, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "new-id" {
		t.Errorf("expected ID new-id, got %q", u.ID)
	}
	if createdUser.Email != "b@example.com" {
		t.Errorf("expected email b@example.com, got %q", createdUser.Email)
	}
}
