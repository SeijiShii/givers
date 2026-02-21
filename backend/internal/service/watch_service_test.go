package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockWatchRepository — WatchRepository のモック
// ---------------------------------------------------------------------------

type mockWatchRepository struct {
	watchFunc              func(ctx context.Context, userID, projectID string) error
	unwatchFunc            func(ctx context.Context, userID, projectID string) error
	listWatchedProjectFunc func(ctx context.Context, userID string) ([]*model.Project, error)
}

func (m *mockWatchRepository) Watch(ctx context.Context, userID, projectID string) error {
	if m.watchFunc != nil {
		return m.watchFunc(ctx, userID, projectID)
	}
	return nil
}

func (m *mockWatchRepository) Unwatch(ctx context.Context, userID, projectID string) error {
	if m.unwatchFunc != nil {
		return m.unwatchFunc(ctx, userID, projectID)
	}
	return nil
}

func (m *mockWatchRepository) ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error) {
	if m.listWatchedProjectFunc != nil {
		return m.listWatchedProjectFunc(ctx, userID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests: WatchService.Watch
// ---------------------------------------------------------------------------

func TestWatchService_Watch_CallsRepository(t *testing.T) {
	var capturedUserID, capturedProjectID string
	mock := &mockWatchRepository{
		watchFunc: func(ctx context.Context, userID, projectID string) error {
			capturedUserID = userID
			capturedProjectID = projectID
			return nil
		},
	}

	svc := NewWatchService(mock)
	ctx := context.Background()
	if err := svc.Watch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Watch returned unexpected error: %v", err)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedProjectID != "project-1" {
		t.Errorf("expected projectID=project-1, got %q", capturedProjectID)
	}
}

func TestWatchService_Watch_PropagatesError(t *testing.T) {
	mock := &mockWatchRepository{
		watchFunc: func(ctx context.Context, userID, projectID string) error {
			return errors.New("db error")
		},
	}

	svc := NewWatchService(mock)
	if err := svc.Watch(context.Background(), "user-1", "project-1"); err == nil {
		t.Error("expected error from Watch, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: WatchService.Unwatch
// ---------------------------------------------------------------------------

func TestWatchService_Unwatch_CallsRepository(t *testing.T) {
	var capturedUserID, capturedProjectID string
	mock := &mockWatchRepository{
		unwatchFunc: func(ctx context.Context, userID, projectID string) error {
			capturedUserID = userID
			capturedProjectID = projectID
			return nil
		},
	}

	svc := NewWatchService(mock)
	ctx := context.Background()
	if err := svc.Unwatch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Unwatch returned unexpected error: %v", err)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedProjectID != "project-1" {
		t.Errorf("expected projectID=project-1, got %q", capturedProjectID)
	}
}

func TestWatchService_Unwatch_PropagatesError(t *testing.T) {
	mock := &mockWatchRepository{
		unwatchFunc: func(ctx context.Context, userID, projectID string) error {
			return errors.New("db error")
		},
	}

	svc := NewWatchService(mock)
	if err := svc.Unwatch(context.Background(), "user-1", "project-1"); err == nil {
		t.Error("expected error from Unwatch, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: WatchService.ListWatchedProjects
// ---------------------------------------------------------------------------

func TestWatchService_ListWatchedProjects_CallsRepository(t *testing.T) {
	want := []*model.Project{
		{ID: "p1", Name: "Alpha"},
		{ID: "p2", Name: "Beta"},
	}
	var capturedUserID string
	mock := &mockWatchRepository{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			capturedUserID = userID
			return want, nil
		},
	}

	svc := NewWatchService(mock)
	got, err := svc.ListWatchedProjects(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListWatchedProjects returned unexpected error: %v", err)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 projects, got %d", len(got))
	}
	if got[0].ID != "p1" || got[1].ID != "p2" {
		t.Errorf("unexpected projects: %v", got)
	}
}

func TestWatchService_ListWatchedProjects_ReturnsEmptySlice(t *testing.T) {
	mock := &mockWatchRepository{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			return []*model.Project{}, nil
		},
	}

	svc := NewWatchService(mock)
	got, err := svc.ListWatchedProjects(context.Background(), "user-no-watches")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

func TestWatchService_ListWatchedProjects_PropagatesError(t *testing.T) {
	mock := &mockWatchRepository{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewWatchService(mock)
	_, err := svc.ListWatchedProjects(context.Background(), "user-1")
	if err == nil {
		t.Error("expected error from ListWatchedProjects, got nil")
	}
}
