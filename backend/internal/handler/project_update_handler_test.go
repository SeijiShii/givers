package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// mockProjectUpdateService — ProjectUpdateService のモック
// ---------------------------------------------------------------------------

type mockProjectUpdateService struct {
	listFunc   func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error)
	getFunc    func(ctx context.Context, id string) (*model.ProjectUpdate, error)
	createFunc func(ctx context.Context, update *model.ProjectUpdate) error
	updateFunc func(ctx context.Context, update *model.ProjectUpdate) error
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockProjectUpdateService) ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectID, includeHidden)
	}
	return nil, nil
}

func (m *mockProjectUpdateService) GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProjectUpdateService) Create(ctx context.Context, update *model.ProjectUpdate) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, update)
	}
	return nil
}

func (m *mockProjectUpdateService) Update(ctx context.Context, update *model.ProjectUpdate) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, update)
	}
	return nil
}

func (m *mockProjectUpdateService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// Note: mockProjectService is declared in project_handler_test.go (same package).

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newUpdateMux builds a ServeMux with all four project-update routes registered.
func newUpdateMux(h *ProjectUpdateHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /api/projects/{id}/updates", http.HandlerFunc(h.List))
	mux.Handle("POST /api/projects/{id}/updates", http.HandlerFunc(h.Create))
	mux.Handle("PUT /api/projects/{id}/updates/{uid}", http.HandlerFunc(h.UpdateUpdate))
	mux.Handle("DELETE /api/projects/{id}/updates/{uid}", http.HandlerFunc(h.Delete))
	return mux
}

func ptr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// GET /api/projects/{id}/updates — List
// ---------------------------------------------------------------------------

func TestProjectUpdateHandler_List_ReturnsUpdates(t *testing.T) {
	now := time.Now()
	want := []*model.ProjectUpdate{
		{ID: "u1", ProjectID: "project-1", Body: "first update", Visible: true, CreatedAt: now},
		{ID: "u2", ProjectID: "project-1", Body: "second update", Visible: true, CreatedAt: now},
	}
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			return want, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Updates []*model.ProjectUpdate `json:"updates"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Updates) != 2 {
		t.Errorf("expected 2 updates, got %d", len(resp.Updates))
	}
	if resp.Updates[0].ID != "u1" || resp.Updates[1].ID != "u2" {
		t.Errorf("unexpected update IDs: %v", resp.Updates)
	}
}

func TestProjectUpdateHandler_List_EmptyArray(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			return []*model.ProjectUpdate{}, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Updates []*model.ProjectUpdate `json:"updates"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Updates == nil {
		t.Error("expected non-nil (empty) updates array, got nil")
	}
	if len(resp.Updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(resp.Updates))
	}
}

func TestProjectUpdateHandler_List_ProjectNotFound_Returns404(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/no-such-project/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_List_OwnerSeesAllUpdates(t *testing.T) {
	// Owner should see all updates including hidden (includeHidden=true)
	var capturedIncludeHidden bool
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			capturedIncludeHidden = includeHidden
			return nil, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "owner-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !capturedIncludeHidden {
		t.Error("expected includeHidden=true for project owner, got false")
	}
}

func TestProjectUpdateHandler_List_NonOwnerSeesOnlyVisible(t *testing.T) {
	var capturedIncludeHidden bool
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			capturedIncludeHidden = includeHidden
			return nil, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	// Authenticated as a different user (not owner)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "other-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedIncludeHidden {
		t.Error("expected includeHidden=false for non-owner, got true")
	}
}

func TestProjectUpdateHandler_List_UnauthenticatedSeesOnlyVisible(t *testing.T) {
	var capturedIncludeHidden bool
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			capturedIncludeHidden = includeHidden
			return nil, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	// No auth in context
	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedIncludeHidden {
		t.Error("expected includeHidden=false for unauthenticated user, got true")
	}
}

func TestProjectUpdateHandler_List_ServiceError_Returns500(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			return nil, errors.New("db error")
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_List_ContentType(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project-1/updates", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// POST /api/projects/{id}/updates — Create
// ---------------------------------------------------------------------------

func TestProjectUpdateHandler_Create_Success(t *testing.T) {
	var capturedUpdate *model.ProjectUpdate
	updateSvc := &mockProjectUpdateService{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			update.ID = "new-id"
			return nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"title": "New Release", "body": "We shipped a new version"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedUpdate == nil {
		t.Fatal("expected Create to be called with an update, got nil")
	}
	if capturedUpdate.Body != "We shipped a new version" {
		t.Errorf("expected Body=We shipped a new version, got %q", capturedUpdate.Body)
	}
	if capturedUpdate.Title == nil || *capturedUpdate.Title != "New Release" {
		t.Errorf("expected Title=New Release, got %v", capturedUpdate.Title)
	}
	if capturedUpdate.ProjectID != "project-1" {
		t.Errorf("expected ProjectID=project-1, got %q", capturedUpdate.ProjectID)
	}
	if capturedUpdate.AuthorID != "user-1" {
		t.Errorf("expected AuthorID=user-1, got %q", capturedUpdate.AuthorID)
	}
}

func TestProjectUpdateHandler_Create_ResponseContainsCreatedUpdate(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			update.ID = "new-id"
			update.CreatedAt = time.Now()
			return nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "minimal update"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var got model.ProjectUpdate
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "new-id" {
		t.Errorf("expected ID=new-id in response, got %q", got.ID)
	}
}

func TestProjectUpdateHandler_Create_Unauthorized(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "some update"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	// No auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_ForbiddenForNonOwner(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "actual-owner"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "some update"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "not-the-owner"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-owner, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_ProjectNotFound(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "some update"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/no-such-project/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing project, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_MissingBody_Returns400(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"title": "only title, no body"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing body, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_EmptyBody_Returns400(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader("{invalid json"))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Create_ServiceError_Returns500(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			return errors.New("db error")
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "some update"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/project-1/updates", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/projects/{id}/updates/{uid} — UpdateUpdate
// ---------------------------------------------------------------------------

func TestProjectUpdateHandler_UpdateUpdate_Success(t *testing.T) {
	existing := &model.ProjectUpdate{
		ID:        "u1",
		ProjectID: "project-1",
		AuthorID:  "user-1",
		Body:      "old body",
		Visible:   true,
	}
	var capturedUpdate *model.ProjectUpdate
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "new body", "visible": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedUpdate == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedUpdate.Body != "new body" {
		t.Errorf("expected Body=new body, got %q", capturedUpdate.Body)
	}
	if capturedUpdate.Visible {
		t.Error("expected Visible=false after update, got true")
	}
}

func TestProjectUpdateHandler_UpdateUpdate_Unauthorized(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	// No auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_ForbiddenForNonAuthor(t *testing.T) {
	existing := &model.ProjectUpdate{
		ID:        "u1",
		ProjectID: "project-1",
		AuthorID:  "original-author",
		Body:      "body",
		Visible:   true,
	}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "updated by impostor"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "different-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-author, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_UpdateNotFound(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return nil, errors.New("not found")
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/nonexistent", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_WrongProject_Returns404(t *testing.T) {
	// Update belongs to a different project
	existing := &model.ProjectUpdate{
		ID:        "u1",
		ProjectID: "project-OTHER",
		AuthorID:  "user-1",
		Body:      "body",
		Visible:   true,
	}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 when update belongs to different project, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_InvalidJSON(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "user-1", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader("{bad json"))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_ServiceError(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "user-1", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			return errors.New("db error")
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_CanUpdateTitle(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "user-1", Body: "body", Visible: true}
	var capturedUpdate *model.ProjectUpdate
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"title": "New Title"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedUpdate.Title == nil || *capturedUpdate.Title != "New Title" {
		t.Errorf("expected Title=New Title, got %v", capturedUpdate.Title)
	}
}

func TestProjectUpdateHandler_UpdateUpdate_ResponseContainsUpdate(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "user-1", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			return nil
		},
	}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	body := `{"body": "new body"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/project-1/updates/u1", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var got model.ProjectUpdate
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", got.ID)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/projects/{id}/updates/{uid} — Delete
// ---------------------------------------------------------------------------

func TestProjectUpdateHandler_Delete_Success_OwnerCanDelete(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "other-author", Body: "body", Visible: true}
	var capturedDeleteID string
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			capturedDeleteID = id
			return nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "owner-user"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "owner-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedDeleteID != "u1" {
		t.Errorf("expected Delete called with u1, got %q", capturedDeleteID)
	}

	var resp map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp["ok"] {
		t.Error("expected {ok: true} in response")
	}
}

func TestProjectUpdateHandler_Delete_Success_HostCanDelete(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "other-author", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		deleteFunc: func(ctx context.Context, id string) error { return nil },
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "actual-owner"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	ctx := auth.WithUserID(req.Context(), "host-user")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for host, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectUpdateHandler_Delete_Unauthorized(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	// No auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Delete_ForbiddenForRandomUser(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "other-author", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "actual-owner"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "random-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-owner/non-host, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Delete_UpdateNotFound(t *testing.T) {
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return nil, errors.New("not found")
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/nonexistent", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Delete_ProjectNotFound(t *testing.T) {
	updateSvc := &mockProjectUpdateService{}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/no-project/updates/u1", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing project, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Delete_ServiceError(t *testing.T) {
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "author", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			return errors.New("db error")
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

func TestProjectUpdateHandler_Delete_WrongProject_Returns404(t *testing.T) {
	// Update belongs to a different project than the URL
	existing := &model.ProjectUpdate{ID: "u1", ProjectID: "project-OTHER", AuthorID: "user-1", Body: "body", Visible: true}
	updateSvc := &mockProjectUpdateService{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return existing, nil
		},
	}
	projectSvc := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "project-1", OwnerID: "user-1"}, nil
		},
	}
	h := NewProjectUpdateHandler(updateSvc, projectSvc)
	mux := newUpdateMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project-1/updates/u1", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 when update belongs to different project, got %d", rec.Code)
	}
}
