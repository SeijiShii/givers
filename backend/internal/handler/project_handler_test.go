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
	listFunc          func(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error)
	getByIDFunc       func(ctx context.Context, id string) (*model.Project, error)
	listByOwnerIDFunc func(ctx context.Context, ownerID string) ([]*model.Project, error)
	createFunc        func(ctx context.Context, project *model.Project) error
	updateFunc        func(ctx context.Context, project *model.Project) error
	deleteFunc        func(ctx context.Context, id string) error
}

func (m *mockProjectService) List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, sort, limit, cursor)
	}
	return &model.ProjectListResult{}, nil
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
	result := &model.ProjectListResult{
		Projects: []*model.Project{{ID: "1", Name: "P1"}},
	}
	mock := &mockProjectService{
		listFunc: func(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
			return result, nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects", http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got model.ProjectListResult
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Projects) != 1 || got.Projects[0].Name != "P1" {
		t.Errorf("expected 1 project P1, got %v", got.Projects)
	}
}

func TestProjectHandler_List_SortHot(t *testing.T) {
	var capturedSort string
	mock := &mockProjectService{
		listFunc: func(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
			capturedSort = sort
			return &model.ProjectListResult{
				Projects: []*model.Project{{ID: "hot-1", Name: "Hot"}},
			}, nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects", http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/projects?sort=hot", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedSort != "hot" {
		t.Errorf("expected sort=hot passed to service, got %q", capturedSort)
	}
}

func TestProjectHandler_List_WithCursor(t *testing.T) {
	var capturedCursor string
	mock := &mockProjectService{
		listFunc: func(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
			capturedCursor = cursor
			return &model.ProjectListResult{
				Projects:   []*model.Project{{ID: "2", Name: "P2"}},
				NextCursor: "next-id",
			}, nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("GET /api/projects", http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/projects?cursor=prev-id", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedCursor != "prev-id" {
		t.Errorf("expected cursor=prev-id, got %q", capturedCursor)
	}
	var got model.ProjectListResult
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.NextCursor != "next-id" {
		t.Errorf("expected next_cursor=next-id, got %q", got.NextCursor)
	}
}

func TestProjectHandler_Get_NotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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

// ---------------------------------------------------------------------------
// Create: activity recording tests (uses mockActivityService from activity_handler_test.go)
// ---------------------------------------------------------------------------

func TestProjectHandler_Create_RecordsProjectCreatedActivity(t *testing.T) {
	var recordedActivity *model.ActivityItem
	actSvc := &mockActivityService{
		recordFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandlerWithActivity(mock, nil, actSvc)

	body := bytes.NewBufferString(`{"name":"New Project","description":"Desc"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if recordedActivity == nil {
		t.Fatal("expected project_created activity to be recorded")
	}
	if recordedActivity.Type != "project_created" {
		t.Errorf("expected type=project_created, got %q", recordedActivity.Type)
	}
	if recordedActivity.ProjectID != "new-id" {
		t.Errorf("expected ProjectID=new-id, got %q", recordedActivity.ProjectID)
	}
	if recordedActivity.ActorName == nil || *recordedActivity.ActorName != "u1" {
		t.Errorf("expected ActorName=u1, got %v", recordedActivity.ActorName)
	}
}

func TestProjectHandler_Create_ActivityErrorDoesNotFail(t *testing.T) {
	actSvc := &mockActivityService{
		recordFunc: func(_ context.Context, _ *model.ActivityItem) error {
			return errors.New("activity db error")
		},
	}
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandlerWithActivity(mock, nil, actSvc)

	body := bytes.NewBufferString(`{"name":"P"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 even when activity recording fails, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Update: activity recording tests
// ---------------------------------------------------------------------------

func TestProjectHandler_Update_RecordsProjectUpdatedActivity(t *testing.T) {
	var recordedActivity *model.ActivityItem
	actSvc := &mockActivityService{
		recordFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Name: "P1"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error {
			return nil
		},
	}
	h := NewProjectHandlerWithActivity(mock, nil, actSvc)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/projects/{id}", http.HandlerFunc(h.Update))

	body := bytes.NewBufferString(`{"name":"Updated"}`)
	req := httptest.NewRequest("PUT", "/api/projects/p1", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if recordedActivity == nil {
		t.Fatal("expected project_updated activity to be recorded")
	}
	if recordedActivity.Type != "project_updated" {
		t.Errorf("expected type=project_updated, got %q", recordedActivity.Type)
	}
	if recordedActivity.ProjectID != "p1" {
		t.Errorf("expected ProjectID=p1, got %q", recordedActivity.ProjectID)
	}
	if recordedActivity.ActorName == nil || *recordedActivity.ActorName != "u1" {
		t.Errorf("expected ActorName=u1, got %v", recordedActivity.ActorName)
	}
}

// ---------------------------------------------------------------------------
// Create: existing tests
// ---------------------------------------------------------------------------

func TestProjectHandler_Create_Unauthorized(t *testing.T) {
	mock := &mockProjectService{}
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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

func TestProjectHandler_Create_WithDeadline_YYYYMMDD(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","deadline":"2025-12-31"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if created == nil || created.Deadline == nil {
		t.Fatal("expected deadline to be set")
	}
	if created.Deadline.Year() != 2025 || created.Deadline.Month() != 12 || created.Deadline.Day() != 31 {
		t.Errorf("expected 2025-12-31, got %v", created.Deadline)
	}
}

func TestProjectHandler_Create_WithDeadline_RFC3339(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","deadline":"2025-06-15T10:30:00Z"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if created == nil || created.Deadline == nil {
		t.Fatal("expected deadline to be set")
	}
	if created.Deadline.Year() != 2025 || created.Deadline.Month() != 6 || created.Deadline.Day() != 15 {
		t.Errorf("expected 2025-06-15, got %v", created.Deadline)
	}
}

func TestProjectHandler_Create_EmptyDeadline(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","deadline":""}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if created != nil && created.Deadline != nil {
		t.Errorf("expected nil deadline for empty string, got %v", created.Deadline)
	}
}

func TestProjectHandler_Update_Deadline(t *testing.T) {
	var updated *model.Project
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Name: "P1"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error {
			updated = p
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/projects/{id}", http.HandlerFunc(h.Update))

	body := bytes.NewBufferString(`{"deadline":"2026-03-01"}`)
	req := httptest.NewRequest("PUT", "/api/projects/p1", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if updated == nil || updated.Deadline == nil {
		t.Fatal("expected deadline to be set after update")
	}
	if updated.Deadline.Year() != 2026 || updated.Deadline.Month() != 3 || updated.Deadline.Day() != 1 {
		t.Errorf("expected 2026-03-01, got %v", updated.Deadline)
	}
}

func TestProjectHandler_Update_Forbidden(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user", Name: "P1"}, nil
		},
	}
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(&mockProjectService{}, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(&mockProjectService{}, nil)

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
	h := NewProjectHandler(mock, nil)

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
	h := NewProjectHandler(mock, nil)

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

// ---------------------------------------------------------------------------
// Create / Update: share_message tests
// ---------------------------------------------------------------------------

func TestProjectHandler_Create_WithShareMessage(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","share_message":"ぜひ応援してください！"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if created == nil || created.ShareMessage != "ぜひ応援してください！" {
		t.Errorf("expected share_message, got %q", created.ShareMessage)
	}
}

func TestProjectHandler_Update_ShareMessage(t *testing.T) {
	var updated *model.Project
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Name: "P1", ShareMessage: "old"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error {
			updated = p
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/projects/{id}", http.HandlerFunc(h.Update))

	body := bytes.NewBufferString(`{"share_message":"新しいメッセージ"}`)
	req := httptest.NewRequest("PUT", "/api/projects/p1", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if updated == nil || updated.ShareMessage != "新しいメッセージ" {
		t.Errorf("expected share_message='新しいメッセージ', got %q", updated.ShareMessage)
	}
}

func TestProjectHandler_Update_ShareMessageNotSentKeepsOld(t *testing.T) {
	var updated *model.Project
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "u1", Name: "P1", ShareMessage: "keep me"}, nil
		},
		updateFunc: func(ctx context.Context, p *model.Project) error {
			updated = p
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/projects/{id}", http.HandlerFunc(h.Update))

	body := bytes.NewBufferString(`{"name":"Updated Name"}`)
	req := httptest.NewRequest("PUT", "/api/projects/p1", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if updated == nil || updated.ShareMessage != "keep me" {
		t.Errorf("expected share_message to be preserved as 'keep me', got %q", updated.ShareMessage)
	}
}

// ---------------------------------------------------------------------------
// Create / Update: overview → description auto-fill tests
// ---------------------------------------------------------------------------

func TestProjectHandler_Create_OverviewFillsDescription(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","overview":"# My Project\n\nThis is a **detailed** overview."}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if created == nil {
		t.Fatal("expected project to be created")
	}
	if created.Overview != "# My Project\n\nThis is a **detailed** overview." {
		t.Errorf("expected overview to be set, got %q", created.Overview)
	}
	// description should be auto-filled from overview (plain text)
	if created.Description == "" {
		t.Error("expected description to be auto-filled from overview")
	}
	if created.Description == created.Overview {
		t.Error("description should be plain text, not raw Markdown")
	}
}

func TestProjectHandler_Create_ExplicitDescriptionNotOverridden(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}
	h := NewProjectHandler(mock, nil)

	body := bytes.NewBufferString(`{"name":"P","description":"Explicit desc","overview":"# Full overview"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "u1"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if created.Description != "Explicit desc" {
		t.Errorf("expected description='Explicit desc', got %q", created.Description)
	}
}

// ---------------------------------------------------------------------------
// Create: host vs regular owner (Stripe Connect skip)
// ---------------------------------------------------------------------------

func TestProjectHandler_Create_HostGetsActiveStatus(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "host-proj"
			return nil
		},
	}
	connectFunc := func(_ context.Context, id string) (string, error) { return "https://connect.stripe.com/setup?acct=" + id, nil }
	h := NewProjectHandler(mock, connectFunc)

	body := bytes.NewBufferString(`{"name":"Host Project"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	ctx := auth.WithUserID(context.Background(), "host-user")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if created == nil || created.Status != "active" {
		t.Errorf("expected status=active for host, got %q", created.Status)
	}

	// Response should NOT contain a Stripe Connect URL
	var resp model.Project
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.StripeConnectURL != "" {
		t.Errorf("expected no StripeConnectURL for host, got %q", resp.StripeConnectURL)
	}
}

func TestProjectHandler_Create_RegularOwnerGetsDraftStatus(t *testing.T) {
	var created *model.Project
	mock := &mockProjectService{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "owner-proj"
			return nil
		},
	}
	connectFunc := func(_ context.Context, id string) (string, error) { return "https://connect.stripe.com/setup?acct=" + id, nil }
	h := NewProjectHandler(mock, connectFunc)

	body := bytes.NewBufferString(`{"name":"Owner Project"}`)
	req := httptest.NewRequest("POST", "/api/projects", body)
	req = req.WithContext(auth.WithUserID(context.Background(), "regular-user"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if created == nil || created.Status != "draft" {
		t.Errorf("expected status=draft for regular owner, got %q", created.Status)
	}

	// Response should contain a Stripe Connect URL
	var resp model.Project
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.StripeConnectURL == "" {
		t.Error("expected StripeConnectURL for regular owner, got empty")
	}
}

func TestProjectHandler_PatchStatus_NotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewProjectHandler(mock, nil)

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
