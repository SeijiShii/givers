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
// mockContactRepository — in-memory stub for testing
// ---------------------------------------------------------------------------

type mockContactRepository struct {
	saveFunc         func(ctx context.Context, msg *model.ContactMessage) error
	listFunc         func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error)
	updateStatusFunc func(ctx context.Context, id string, status string) error
}

func (m *mockContactRepository) Save(ctx context.Context, msg *model.ContactMessage) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, msg)
	}
	return nil
}

func (m *mockContactRepository) List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, opts)
	}
	return nil, nil
}

func (m *mockContactRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Submit tests
// ---------------------------------------------------------------------------

func TestContactService_Submit_SetsUnreadStatus(t *testing.T) {
	var saved *model.ContactMessage
	mock := &mockContactRepository{
		saveFunc: func(ctx context.Context, msg *model.ContactMessage) error {
			saved = msg
			return nil
		},
	}
	svc := NewContactService(mock)

	msg := &model.ContactMessage{
		Email:   "test@example.com",
		Message: "Hello",
	}
	if err := svc.Submit(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if saved == nil {
		t.Fatal("expected Save to be called")
	}
	if saved.Status != "unread" {
		t.Errorf("expected status=unread, got %q", saved.Status)
	}
}

// TestContactService_Submit_SetsCreatedAt verifies the service sets CreatedAt/UpdatedAt.
func TestContactService_Submit_SetsTimestamps(t *testing.T) {
	before := time.Now()
	var saved *model.ContactMessage
	mock := &mockContactRepository{
		saveFunc: func(ctx context.Context, msg *model.ContactMessage) error {
			saved = msg
			return nil
		},
	}
	svc := NewContactService(mock)

	msg := &model.ContactMessage{
		Email:   "ts@example.com",
		Message: "Timestamps test",
	}
	if err := svc.Submit(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now()
	if saved.CreatedAt.Before(before) || saved.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", saved.CreatedAt, before, after)
	}
	if saved.UpdatedAt.Before(before) || saved.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in expected range", saved.UpdatedAt)
	}
}

// TestContactService_Submit_RepositoryError propagates repository errors.
func TestContactService_Submit_RepositoryError(t *testing.T) {
	mock := &mockContactRepository{
		saveFunc: func(ctx context.Context, msg *model.ContactMessage) error {
			return errors.New("db write failed")
		},
	}
	svc := NewContactService(mock)

	msg := &model.ContactMessage{Email: "e@e.com", Message: "Hi"}
	err := svc.Submit(context.Background(), msg)
	if err == nil {
		t.Error("expected error from repository, got nil")
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestContactService_List_ForwardsOptions(t *testing.T) {
	var capturedOpts model.ContactListOptions
	mock := &mockContactRepository{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			capturedOpts = opts
			return nil, nil
		},
	}
	svc := NewContactService(mock)

	opts := model.ContactListOptions{Status: "unread", Limit: 10, Offset: 5}
	_, err := svc.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOpts.Status != "unread" {
		t.Errorf("expected status=unread forwarded, got %q", capturedOpts.Status)
	}
	if capturedOpts.Limit != 10 {
		t.Errorf("expected limit=10 forwarded, got %d", capturedOpts.Limit)
	}
	if capturedOpts.Offset != 5 {
		t.Errorf("expected offset=5 forwarded, got %d", capturedOpts.Offset)
	}
}

// TestContactService_List_ReturnsMessages verifies messages are returned correctly.
func TestContactService_List_ReturnsMessages(t *testing.T) {
	now := time.Now()
	want := []*model.ContactMessage{
		{ID: "1", Email: "a@b.com", Message: "Hi", Status: "unread", CreatedAt: now},
	}
	mock := &mockContactRepository{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			return want, nil
		},
	}
	svc := NewContactService(mock)

	got, err := svc.List(context.Background(), model.ContactListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Errorf("expected %v, got %v", want, got)
	}
}

// TestContactService_List_RepositoryError propagates repository errors.
func TestContactService_List_RepositoryError(t *testing.T) {
	mock := &mockContactRepository{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			return nil, errors.New("db read failed")
		},
	}
	svc := NewContactService(mock)

	_, err := svc.List(context.Background(), model.ContactListOptions{})
	if err == nil {
		t.Error("expected error from repository, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdateStatus tests
// ---------------------------------------------------------------------------

// TestContactService_UpdateStatus_DelegatestoRepo verifies that the service
// calls the repository with the correct arguments.
func TestContactService_UpdateStatus_DelegatestoRepo(t *testing.T) {
	var capturedID, capturedStatus string
	mock := &mockContactRepository{
		updateStatusFunc: func(ctx context.Context, id string, status string) error {
			capturedID = id
			capturedStatus = status
			return nil
		},
	}
	svc := NewContactService(mock)

	if err := svc.UpdateStatus(context.Background(), "msg-1", "read"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "msg-1" {
		t.Errorf("expected id=msg-1, got %q", capturedID)
	}
	if capturedStatus != "read" {
		t.Errorf("expected status=read, got %q", capturedStatus)
	}
}

// TestContactService_UpdateStatus_NotFound propagates ErrNotFound.
func TestContactService_UpdateStatus_NotFound(t *testing.T) {
	mock := &mockContactRepository{
		updateStatusFunc: func(ctx context.Context, id string, status string) error {
			return repository.ErrNotFound
		},
	}
	svc := NewContactService(mock)

	err := svc.UpdateStatus(context.Background(), "no-such-id", "read")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestContactService_UpdateStatus_RepoError propagates arbitrary errors.
func TestContactService_UpdateStatus_RepoError(t *testing.T) {
	mock := &mockContactRepository{
		updateStatusFunc: func(ctx context.Context, id string, status string) error {
			return errors.New("db write failed")
		},
	}
	svc := NewContactService(mock)

	err := svc.UpdateStatus(context.Background(), "msg-1", "read")
	if err == nil {
		t.Error("expected error from repository, got nil")
	}
}
