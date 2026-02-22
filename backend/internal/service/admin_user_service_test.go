package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock SessionRepository
// ---------------------------------------------------------------------------

type mockSessionRepository struct {
	createFunc         func(ctx context.Context, s *model.Session) error
	findByTokenFunc    func(ctx context.Context, token string) (*model.Session, error)
	deleteByTokenFunc  func(ctx context.Context, token string) error
	deleteByUserIDFunc func(ctx context.Context, userID string) error
}

func (m *mockSessionRepository) Create(ctx context.Context, s *model.Session) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	return nil
}
func (m *mockSessionRepository) FindByToken(ctx context.Context, token string) (*model.Session, error) {
	if m.findByTokenFunc != nil {
		return m.findByTokenFunc(ctx, token)
	}
	return nil, errors.New("not found")
}
func (m *mockSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	if m.deleteByTokenFunc != nil {
		return m.deleteByTokenFunc(ctx, token)
	}
	return nil
}
func (m *mockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserIDFunc != nil {
		return m.deleteByUserIDFunc(ctx, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock UserRepository (admin methods)
// ---------------------------------------------------------------------------

type mockAdminUserRepository struct {
	findByIDFunc    func(ctx context.Context, id string) (*model.User, error)
	findByGoogleID  func(ctx context.Context, googleID string) (*model.User, error)
	findByGitHubID  func(ctx context.Context, githubID string) (*model.User, error)
	createFunc      func(ctx context.Context, user *model.User) error
	listFunc        func(ctx context.Context, limit, offset int) ([]*model.User, error)
	suspendFunc     func(ctx context.Context, id string, suspend bool) error
}

func (m *mockAdminUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAdminUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	if m.findByGoogleID != nil {
		return m.findByGoogleID(ctx, googleID)
	}
	return nil, nil
}
func (m *mockAdminUserRepository) FindByGitHubID(ctx context.Context, githubID string) (*model.User, error) {
	if m.findByGitHubID != nil {
		return m.findByGitHubID(ctx, githubID)
	}
	return nil, nil
}
func (m *mockAdminUserRepository) Create(ctx context.Context, user *model.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	return nil
}
func (m *mockAdminUserRepository) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return nil, nil
}
func (m *mockAdminUserRepository) Suspend(ctx context.Context, id string, suspend bool) error {
	if m.suspendFunc != nil {
		return m.suspendFunc(ctx, id, suspend)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AdminUserService.ListUsers tests
// ---------------------------------------------------------------------------

func TestAdminUserService_ListUsers_ReturnsUsers(t *testing.T) {
	users := []*model.User{
		{ID: "1", Email: "a@b.com", Name: "Alice"},
		{ID: "2", Email: "c@d.com", Name: "Bob"},
	}
	mock := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return users, nil
		},
	}
	svc := NewAdminUserService(mock)

	got, err := svc.ListUsers(context.Background(), 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 users, got %d", len(got))
	}
}

func TestAdminUserService_ListUsers_PropagatesError(t *testing.T) {
	mock := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewAdminUserService(mock)

	_, err := svc.ListUsers(context.Background(), 20, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAdminUserService_ListUsers_ForwardsPagination(t *testing.T) {
	var capturedLimit, capturedOffset int
	mock := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			capturedLimit = limit
			capturedOffset = offset
			return nil, nil
		},
	}
	svc := NewAdminUserService(mock)

	_, _ = svc.ListUsers(context.Background(), 10, 50)
	if capturedLimit != 10 {
		t.Errorf("expected limit=10, got %d", capturedLimit)
	}
	if capturedOffset != 50 {
		t.Errorf("expected offset=50, got %d", capturedOffset)
	}
}

// ---------------------------------------------------------------------------
// AdminUserService.SuspendUser tests
// ---------------------------------------------------------------------------

func TestAdminUserService_SuspendUser_CallsSuspend(t *testing.T) {
	var capturedID string
	var capturedSuspend bool
	mock := &mockAdminUserRepository{
		suspendFunc: func(ctx context.Context, id string, suspend bool) error {
			capturedID = id
			capturedSuspend = suspend
			return nil
		},
	}
	svc := NewAdminUserService(mock)

	if err := svc.SuspendUser(context.Background(), "user-1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "user-1" {
		t.Errorf("expected id=user-1, got %q", capturedID)
	}
	if !capturedSuspend {
		t.Error("expected suspend=true")
	}
}

func TestAdminUserService_SuspendUser_NotFound(t *testing.T) {
	mock := &mockAdminUserRepository{
		suspendFunc: func(ctx context.Context, id string, suspend bool) error {
			return repository.ErrNotFound
		},
	}
	svc := NewAdminUserService(mock)

	err := svc.SuspendUser(context.Background(), "no-such", true)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminUserService_SuspendUser_DeletesSessions(t *testing.T) {
	var deletedUserID string
	sessionRepo := &mockSessionRepository{
		deleteByUserIDFunc: func(ctx context.Context, userID string) error {
			deletedUserID = userID
			return nil
		},
	}
	svc := NewAdminUserServiceWithSessions(&mockAdminUserRepository{}, sessionRepo)

	if err := svc.SuspendUser(context.Background(), "user-1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedUserID != "user-1" {
		t.Errorf("expected sessions deleted for user-1, got %q", deletedUserID)
	}
}

func TestAdminUserService_UnsuspendUser_DoesNotDeleteSessions(t *testing.T) {
	called := false
	sessionRepo := &mockSessionRepository{
		deleteByUserIDFunc: func(ctx context.Context, userID string) error {
			called = true
			return nil
		},
	}
	svc := NewAdminUserServiceWithSessions(&mockAdminUserRepository{}, sessionRepo)

	if err := svc.SuspendUser(context.Background(), "user-1", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected sessions NOT to be deleted on unsuspend")
	}
}

// ---------------------------------------------------------------------------
// AdminUserService.GetUser tests
// ---------------------------------------------------------------------------

func TestAdminUserService_GetUser_ReturnsUser(t *testing.T) {
	now := time.Now()
	expected := &model.User{ID: "u1", Email: "x@y.com", Name: "X", CreatedAt: now}
	mock := &mockAdminUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			if id == "u1" {
				return expected, nil
			}
			return nil, repository.ErrNotFound
		},
	}
	svc := NewAdminUserService(mock)

	got, err := svc.GetUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", got.ID)
	}
}
