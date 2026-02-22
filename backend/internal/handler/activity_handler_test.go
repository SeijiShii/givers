package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Mock ActivityService
// ---------------------------------------------------------------------------

type mockActivityService struct {
	recordFunc        func(ctx context.Context, a *model.ActivityItem) error
	listGlobalFunc    func(ctx context.Context, limit int) ([]*model.ActivityItem, error)
	listByProjectFunc func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error)
}

func (m *mockActivityService) Record(ctx context.Context, a *model.ActivityItem) error {
	if m.recordFunc != nil {
		return m.recordFunc(ctx, a)
	}
	return nil
}
func (m *mockActivityService) ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
	if m.listGlobalFunc != nil {
		return m.listGlobalFunc(ctx, limit)
	}
	return nil, nil
}
func (m *mockActivityService) ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
	if m.listByProjectFunc != nil {
		return m.listByProjectFunc(ctx, projectID, limit)
	}
	return nil, nil
}

// Ensure mock implements interface
var _ service.ActivityService = (*mockActivityService)(nil)

// ---------------------------------------------------------------------------
// GET /api/activity tests
// ---------------------------------------------------------------------------

func TestActivityHandler_GlobalFeed_Success(t *testing.T) {
	now := time.Now()
	actor := "田中太郎"
	items := []*model.ActivityItem{
		{ID: "a1", Type: "donation", ProjectID: "p1", ProjectName: "テスト", ActorName: &actor, Amount: intPtrH(1000), CreatedAt: now},
		{ID: "a2", Type: "project_created", ProjectID: "p2", ProjectName: "新規", ActorName: &actor, CreatedAt: now},
	}
	mock := &mockActivityService{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			if limit != 10 {
				t.Errorf("expected default limit=10, got %d", limit)
			}
			return items, nil
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	rec := httptest.NewRecorder()
	h.GlobalFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Activities []*model.ActivityItem `json:"activities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Activities) != 2 {
		t.Errorf("expected 2 activities, got %d", len(resp.Activities))
	}
	if resp.Activities[0].Type != "donation" {
		t.Errorf("expected type=donation, got %q", resp.Activities[0].Type)
	}
	if resp.Activities[0].ProjectName != "テスト" {
		t.Errorf("expected project_name=テスト, got %q", resp.Activities[0].ProjectName)
	}
}

func TestActivityHandler_GlobalFeed_CustomLimit(t *testing.T) {
	var capturedLimit int
	mock := &mockActivityService{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/activity?limit=5", nil)
	rec := httptest.NewRecorder()
	h.GlobalFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedLimit != 5 {
		t.Errorf("expected limit=5, got %d", capturedLimit)
	}
}

func TestActivityHandler_GlobalFeed_EmptyReturnsEmptyArray(t *testing.T) {
	mock := &mockActivityService{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			return nil, nil
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	rec := httptest.NewRecorder()
	h.GlobalFeed(rec, req)

	var resp struct {
		Activities []*model.ActivityItem `json:"activities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Activities == nil {
		t.Error("expected non-nil activities array")
	}
}

func TestActivityHandler_GlobalFeed_ServiceError(t *testing.T) {
	mock := &mockActivityService{
		listGlobalFunc: func(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	rec := httptest.NewRecorder()
	h.GlobalFeed(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/projects/{id}/activity tests
// ---------------------------------------------------------------------------

func TestActivityHandler_ProjectFeed_Success(t *testing.T) {
	now := time.Now()
	items := []*model.ActivityItem{
		{ID: "a1", Type: "donation", ProjectID: "proj-1", ProjectName: "テスト", Amount: intPtrH(500), CreatedAt: now},
	}
	mock := &mockActivityService{
		listByProjectFunc: func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
			if projectID != "proj-1" {
				t.Errorf("expected projectID=proj-1, got %q", projectID)
			}
			return items, nil
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj-1/activity", nil)
	req.SetPathValue("id", "proj-1")
	rec := httptest.NewRecorder()
	h.ProjectFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Activities []*model.ActivityItem `json:"activities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Activities) != 1 {
		t.Errorf("expected 1 activity, got %d", len(resp.Activities))
	}
}

func TestActivityHandler_ProjectFeed_EmptyReturnsEmptyArray(t *testing.T) {
	mock := &mockActivityService{
		listByProjectFunc: func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
			return nil, nil
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj-1/activity", nil)
	req.SetPathValue("id", "proj-1")
	rec := httptest.NewRecorder()
	h.ProjectFeed(rec, req)

	var resp struct {
		Activities []*model.ActivityItem `json:"activities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Activities == nil {
		t.Error("expected non-nil activities array")
	}
}

func TestActivityHandler_ProjectFeed_ServiceError(t *testing.T) {
	mock := &mockActivityService{
		listByProjectFunc: func(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewActivityHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj-1/activity", nil)
	req.SetPathValue("id", "proj-1")
	rec := httptest.NewRecorder()
	h.ProjectFeed(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Helper (avoid conflict with donation_handler_test.go helpers)
// ---------------------------------------------------------------------------

func intPtrH(n int) *int { return &n }
