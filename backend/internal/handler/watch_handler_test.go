package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// mockWatchService — WatchService のモック
// ---------------------------------------------------------------------------

type mockWatchService struct {
	watchFunc              func(ctx context.Context, userID, projectID string) error
	unwatchFunc            func(ctx context.Context, userID, projectID string) error
	listWatchedProjectFunc func(ctx context.Context, userID string) ([]*model.Project, error)
}

func (m *mockWatchService) Watch(ctx context.Context, userID, projectID string) error {
	if m.watchFunc != nil {
		return m.watchFunc(ctx, userID, projectID)
	}
	return nil
}

func (m *mockWatchService) Unwatch(ctx context.Context, userID, projectID string) error {
	if m.unwatchFunc != nil {
		return m.unwatchFunc(ctx, userID, projectID)
	}
	return nil
}

func (m *mockWatchService) ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error) {
	if m.listWatchedProjectFunc != nil {
		return m.listWatchedProjectFunc(ctx, userID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// POST /api/projects/{id}/watch
// ---------------------------------------------------------------------------

func TestWatchHandler_Watch_Success(t *testing.T) {
	var capturedUserID, capturedProjectID string
	mock := &mockWatchService{
		watchFunc: func(ctx context.Context, userID, projectID string) error {
			capturedUserID = userID
			capturedProjectID = projectID
			return nil
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("POST /api/projects/{id}/watch", http.HandlerFunc(h.Watch))

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-42/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedProjectID != "project-42" {
		t.Errorf("expected projectID=project-42, got %q", capturedProjectID)
	}
}

func TestWatchHandler_Watch_Unauthorized(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("POST /api/projects/{id}/watch", http.HandlerFunc(h.Watch))

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/watch", nil)
	// No auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWatchHandler_Watch_MissingProjectID(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	// Call Watch directly without a mux so PathValue("id") is empty
	req := httptest.NewRequest(http.MethodPost, "/api/projects//watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.Watch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project id, got %d", rec.Code)
	}
}

func TestWatchHandler_Watch_ServiceError(t *testing.T) {
	mock := &mockWatchService{
		watchFunc: func(ctx context.Context, userID, projectID string) error {
			return errors.New("db error")
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("POST /api/projects/{id}/watch", http.HandlerFunc(h.Watch))

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestWatchHandler_Watch_AlreadyWatching_ReturnsOK(t *testing.T) {
	// Idempotent: service returns nil even if already watching
	mock := &mockWatchService{
		watchFunc: func(ctx context.Context, userID, projectID string) error {
			return nil // idempotent — no error even if already watching
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("POST /api/projects/{id}/watch", http.HandlerFunc(h.Watch))

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (idempotent watch), got %d", rec.Code)
	}
}

func TestWatchHandler_Watch_ResponseContentType(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("POST /api/projects/{id}/watch", http.HandlerFunc(h.Watch))

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/projects/{id}/watch
// ---------------------------------------------------------------------------

func TestWatchHandler_Unwatch_Success(t *testing.T) {
	var capturedUserID, capturedProjectID string
	mock := &mockWatchService{
		unwatchFunc: func(ctx context.Context, userID, projectID string) error {
			capturedUserID = userID
			capturedProjectID = projectID
			return nil
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}/watch", http.HandlerFunc(h.Unwatch))

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-42/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedProjectID != "project-42" {
		t.Errorf("expected projectID=project-42, got %q", capturedProjectID)
	}
}

func TestWatchHandler_Unwatch_Unauthorized(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}/watch", http.HandlerFunc(h.Unwatch))

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/watch", nil)
	// No auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWatchHandler_Unwatch_MissingProjectID(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects//watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.Unwatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project id, got %d", rec.Code)
	}
}

func TestWatchHandler_Unwatch_ServiceError(t *testing.T) {
	mock := &mockWatchService{
		unwatchFunc: func(ctx context.Context, userID, projectID string) error {
			return errors.New("db error")
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}/watch", http.HandlerFunc(h.Unwatch))

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestWatchHandler_Unwatch_NotWatching_ReturnsOK(t *testing.T) {
	// Idempotent: service returns nil even if not watching
	mock := &mockWatchService{
		unwatchFunc: func(ctx context.Context, userID, projectID string) error {
			return nil // idempotent
		},
	}
	h := NewWatchHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}/watch", http.HandlerFunc(h.Unwatch))

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/watch", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (idempotent unwatch), got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/me/watches
// ---------------------------------------------------------------------------

func TestWatchHandler_ListWatches_Success(t *testing.T) {
	want := []*model.Project{
		{ID: "p1", Name: "Alpha"},
		{ID: "p2", Name: "Beta"},
	}
	mock := &mockWatchService{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			if userID != "user-1" {
				return nil, nil
			}
			return want, nil
		},
	}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Projects []*model.Project `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(resp.Projects))
	}
	if resp.Projects[0].ID != "p1" || resp.Projects[1].ID != "p2" {
		t.Errorf("unexpected projects in response: %v", resp.Projects)
	}
}

func TestWatchHandler_ListWatches_Unauthorized(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	// No auth in context
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWatchHandler_ListWatches_EmptyList(t *testing.T) {
	mock := &mockWatchService{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			return []*model.Project{}, nil
		},
	}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Projects []*model.Project `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Projects == nil {
		t.Error("expected non-nil (empty) projects slice, got nil")
	}
	if len(resp.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(resp.Projects))
	}
}

func TestWatchHandler_ListWatches_ServiceError(t *testing.T) {
	mock := &mockWatchService{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestWatchHandler_ListWatches_ContentType(t *testing.T) {
	mock := &mockWatchService{}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

func TestWatchHandler_ListWatches_ProjectFields(t *testing.T) {
	want := &model.Project{ID: "p1", Name: "Test Project", Status: "active", OwnerID: "owner-1"}
	mock := &mockWatchService{
		listWatchedProjectFunc: func(ctx context.Context, userID string) ([]*model.Project, error) {
			return []*model.Project{want}, nil
		},
	}
	h := NewWatchHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/me/watches", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	h.ListWatches(rec, req)

	var resp struct {
		Projects []*model.Project `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Projects))
	}
	got := resp.Projects[0]
	if got.ID != want.ID {
		t.Errorf("expected ID=%q, got %q", want.ID, got.ID)
	}
	if got.Name != want.Name {
		t.Errorf("expected Name=%q, got %q", want.Name, got.Name)
	}
	if got.Status != want.Status {
		t.Errorf("expected Status=%q, got %q", want.Status, got.Status)
	}
	if got.OwnerID != want.OwnerID {
		t.Errorf("expected OwnerID=%q, got %q", want.OwnerID, got.OwnerID)
	}
}
