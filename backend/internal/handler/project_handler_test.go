package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

// mockProjectService は ProjectService のモック
type mockProjectService struct {
	listFunc          func(ctx context.Context, limit, offset int) ([]*model.Project, error)
	getByIDFunc       func(ctx context.Context, id string) (*model.Project, error)
	listByOwnerIDFunc func(ctx context.Context, ownerID string) ([]*model.Project, error)
	createFunc        func(ctx context.Context, project *model.Project) error
	updateFunc        func(ctx context.Context, project *model.Project) error
	deleteFunc        func(ctx context.Context, id string) error
}

func (m *mockProjectService) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockProjectService) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockProjectService) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	if m.listByOwnerIDFunc != nil {
		return m.listByOwnerIDFunc(ctx, ownerID)
	}
	return nil, nil
}

func (m *mockProjectService) Create(ctx context.Context, project *model.Project) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, project)
	}
	return nil
}

func (m *mockProjectService) Update(ctx context.Context, project *model.Project) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, project)
	}
	return nil
}

func (m *mockProjectService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestProjectHandler_List(t *testing.T) {
	want := []*model.Project{{ID: "1", Name: "P1"}}
	mock := &mockProjectService{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.Project, error) {
			return want, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects", http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got []*model.Project
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "P1" {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestProjectHandler_Get_NotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects/{id}", http.HandlerFunc(h.Get))

	req := httptest.NewRequest("GET", "/api/projects/999", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProjectHandler_Get_Success(t *testing.T) {
	want := &model.Project{ID: "p1", Name: "Project 1"}
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			if id == "p1" {
				return want, nil
			}
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects/{id}", http.HandlerFunc(h.Get))

	req := httptest.NewRequest("GET", "/api/projects/p1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got model.Project
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestProjectHandler_MyProjects_Unauthorized(t *testing.T) {
	mock := &mockProjectService{}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("GET /api/me/projects", http.HandlerFunc(h.MyProjects))

	req := httptest.NewRequest("GET", "/api/me/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectHandler_MyProjects_WithAuth(t *testing.T) {
	want := []*model.Project{{ID: "1", OwnerID: "u1", Name: "Mine"}}
	mock := &mockProjectService{
		listByOwnerIDFunc: func(ctx context.Context, ownerID string) ([]*model.Project, error) {
			if ownerID != "u1" {
				t.Errorf("expected ownerID=u1, got %q", ownerID)
			}
			return want, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("GET /api/me/projects", http.HandlerFunc(h.MyProjects))

	req := httptest.NewRequest("GET", "/api/me/projects", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got []*model.Project
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].OwnerID != "u1" {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestProjectHandler_Create_Unauthorized(t *testing.T) {
	mock := &mockProjectService{}
	h := NewProjectHandler(mock)

	body := bytes.NewBufferString(`{"name":"New Project"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectHandler_Create_NameRequired(t *testing.T) {
	mock := &mockProjectService{}
	h := NewProjectHandler(mock)

	body := bytes.NewBufferString(`{"description":"only desc"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestProjectHandler_Create_Success(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock)

	body := bytes.NewBufferString(`{"name":"New Project","description":"Desc"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if created == nil || created.OwnerID != "u1" || created.Name != "New Project" {
		t.Errorf("expected owner=u1 name=New Project, got %v", created)
	}
}

func TestProjectHandler_Update_Forbidden(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user", Name: "P1"}, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/projects/{id}", http.HandlerFunc(h.Update))

	body := bytes.NewBufferString(`{"name":"Hacked"}`)
	req := httptest.NewRequest("PUT", "/api/projects/p1", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/projects/{id} tests
// ---------------------------------------------------------------------------

func TestProjectHandler_Delete_Success(t *testing.T) {
	var deletedID string
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Name: "P1"}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}", http.HandlerFunc(h.Delete))

	req := httptest.NewRequest("DELETE", "/api/projects/p1", nil)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if deletedID != "p1" {
		t.Errorf("expected Delete called with p1, got %q", deletedID)
	}
	var resp map[string]bool
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if !resp["ok"] {
		t.Error("expected ok=true in response")
	}
}

func TestProjectHandler_Delete_Unauthorized(t *testing.T) {
	h := NewProjectHandler(&mockProjectService{})

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}", http.HandlerFunc(h.Delete))

	req := httptest.NewRequest("DELETE", "/api/projects/p1", nil)
	// no auth in context
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectHandler_Delete_NotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}", http.HandlerFunc(h.Delete))

	req := httptest.NewRequest("DELETE", "/api/projects/no-such", nil)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProjectHandler_Delete_Forbidden(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user", Name: "P1"}, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}", http.HandlerFunc(h.Delete))

	req := httptest.NewRequest("DELETE", "/api/projects/p1", nil)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestProjectHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "u1"}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			return errors.New("db error")
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/projects/{id}", http.HandlerFunc(h.Delete))

	req := httptest.NewRequest("DELETE", "/api/projects/p1", nil)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/projects/{id}/status tests
// ---------------------------------------------------------------------------

func TestProjectHandler_PatchStatus_Success_Owner(t *testing.T) {
	var updated *model.Project
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Status: "active"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error {
			updated = p
			return nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	body := bytes.NewBufferString(`{"status":"frozen"}`)
	req := httptest.NewRequest("PATCH", "/api/projects/p1/status", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if updated == nil || updated.Status != "frozen" {
		t.Errorf("expected status=frozen updated, got %v", updated)
	}
}

func TestProjectHandler_PatchStatus_Success_Host(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "other-user", Status: "active"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error { return nil },
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	body := bytes.NewBufferString(`{"status":"frozen"}`)
	req := httptest.NewRequest("PATCH", "/api/projects/p1/status", body)
	ctx := auth.WithUserID(context.Background(), "host-user")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for host, got %d", rec.Code)
	}
}

func TestProjectHandler_PatchStatus_Unauthorized(t *testing.T) {
	h := NewProjectHandler(&mockProjectService{})

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	req := httptest.NewRequest("PATCH", "/api/projects/p1/status", bytes.NewBufferString(`{"status":"frozen"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProjectHandler_PatchStatus_Forbidden(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "other-user", Status: "active"}, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	body := bytes.NewBufferString(`{"status":"frozen"}`)
	req := httptest.NewRequest("PATCH", "/api/projects/p1/status", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestProjectHandler_PatchStatus_InvalidStatus(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Status: "active"}, nil
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	body := bytes.NewBufferString(`{"status":"deleted"}`)
	req := httptest.NewRequest("PATCH", "/api/projects/p1/status", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d", rec.Code)
	}
}

func TestProjectHandler_PatchStatus_NotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/projects/{id}/status", http.HandlerFunc(h.PatchStatus))

	body := bytes.NewBufferString(`{"status":"frozen"}`)
	req := httptest.NewRequest("PATCH", "/api/projects/no-such/status", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
