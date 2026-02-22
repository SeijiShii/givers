package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock ActivityRepository
// ---------------------------------------------------------------------------

type mockActivityRepository struct {
	insertFunc        func(ctx context.Context, a *model.ActivityItem) error
	listGlobalFunc    func(ctx context.Context, limit int) ([]*model.ActivityItem, error)
	listByProjectFunc func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error)
}

func (m *mockActivityRepository) Insert(ctx context.Context, a *model.ActivityItem) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, a)
	}
	return nil
}
func (m *mockActivityRepository) ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
	if m.listGlobalFunc != nil {
		return m.listGlobalFunc(ctx, limit)
	}
	return nil, nil
}
func (m *mockActivityRepository) ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
	if m.listByProjectFunc != nil {
		return m.listByProjectFunc(ctx, projectID, limit)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Record tests
// ---------------------------------------------------------------------------

func TestActivityService_Record_Success(t *testing.T) {
	var captured *model.ActivityItem
	repo := &mockActivityRepository{
		insertFunc: func(ctx context.Context, a *model.ActivityItem) error {
			captured = a
			return nil
		},
	}
	svc := NewActivityService(repo)

	item := &model.ActivityItem{
		Type:      "donation",
		ProjectID: "p1",
		Amount:    intPtr(1000),
	}
	if err := svc.Record(context.Background(), item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil || captured.ProjectID != "p1" {
		t.Error("expected activity to be inserted")
	}
}

func TestActivityService_Record_PropagatesError(t *testing.T) {
	repo := &mockActivityRepository{
		insertFunc: func(ctx context.Context, a *model.ActivityItem) error {
			return errors.New("db error")
		},
	}
	svc := NewActivityService(repo)

	err := svc.Record(context.Background(), &model.ActivityItem{Type: "donation", ProjectID: "p1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListGlobal tests
// ---------------------------------------------------------------------------

func TestActivityService_ListGlobal_Success(t *testing.T) {
	now := time.Now()
	actor := "田中太郎"
	items := []*model.ActivityItem{
		{ID: "a1", Type: "donation", ProjectID: "p1", ProjectName: "テストプロジェクト", ActorName: &actor, Amount: intPtr(1000), CreatedAt: now},
		{ID: "a2", Type: "project_created", ProjectID: "p2", ProjectName: "新プロジェクト", ActorName: &actor, CreatedAt: now},
	}
	repo := &mockActivityRepository{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			if limit != 10 {
				t.Errorf("expected limit=10, got %d", limit)
			}
			return items, nil
		},
	}
	svc := NewActivityService(repo)

	got, err := svc.ListGlobal(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 items, got %d", len(got))
	}
	if got[0].Type != "donation" {
		t.Errorf("expected type=donation, got %q", got[0].Type)
	}
}

func TestActivityService_ListGlobal_PropagatesError(t *testing.T) {
	repo := &mockActivityRepository{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewActivityService(repo)

	_, err := svc.ListGlobal(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListByProject tests
// ---------------------------------------------------------------------------

func TestActivityService_ListByProject_Success(t *testing.T) {
	now := time.Now()
	items := []*model.ActivityItem{
		{ID: "a1", Type: "donation", ProjectID: "p1", ProjectName: "テスト", Amount: intPtr(500), CreatedAt: now},
	}
	repo := &mockActivityRepository{
		listByProjectFunc: func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
			if projectID != "p1" {
				t.Errorf("expected projectID=p1, got %q", projectID)
			}
			return items, nil
		},
	}
	svc := NewActivityService(repo)

	got, err := svc.ListByProject(context.Background(), "p1", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 item, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func intPtr(n int) *int { return &n }
