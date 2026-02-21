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
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock AdminUserService
// ---------------------------------------------------------------------------

type mockAdminUserService struct {
	listUsersFunc  func(ctx context.Context, limit, offset int) ([]*model.User, error)
	suspendFunc    func(ctx context.Context, id string, suspend bool) error
	getUserFunc    func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockAdminUserService) ListUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(ctx, limit, offset)
	}
	return nil, nil
}
func (m *mockAdminUserService) SuspendUser(ctx context.Context, id string, suspend bool) error {
	if m.suspendFunc != nil {
		return m.suspendFunc(ctx, id, suspend)
	}
	return nil
}
func (m *mockAdminUserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return nil, nil
}

// Mock ProjectService for disclosure-export
type mockProjectServiceForAdmin struct {
	getByIDFunc func(ctx context.Context, id string) (*model.Project, error)
}

func (m *mockProjectServiceForAdmin) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockProjectServiceForAdmin) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockProjectServiceForAdmin) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockProjectServiceForAdmin) Create(ctx context.Context, p *model.Project) error { return nil }
func (m *mockProjectServiceForAdmin) Update(ctx context.Context, p *model.Project) error {
	return nil
}
func (m *mockProjectServiceForAdmin) Delete(ctx context.Context, id string) error {
	return nil
}

// helper: create a host-authenticated request
func adminHostRequest(method, url, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(r.Context(), "host-id")
	ctx = auth.WithIsHost(ctx, true)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// GET /api/admin/users tests
// ---------------------------------------------------------------------------

func TestAdminUserHandler_List_RequiresAuth(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAdminUserHandler_List_RequiresHost(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	ctx := auth.WithUserID(req.Context(), "regular-user")
	ctx = auth.WithIsHost(ctx, false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAdminUserHandler_List_Success(t *testing.T) {
	now := time.Now()
	users := []*model.User{
		{ID: "1", Email: "a@b.com", Name: "Alice", CreatedAt: now},
		{ID: "2", Email: "c@d.com", Name: "Bob", CreatedAt: now},
	}
	mock := &mockAdminUserService{
		listUsersFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return users, nil
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodGet, "/api/admin/users", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Users []*model.User `json:"users"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(resp.Users))
	}
}

func TestAdminUserHandler_List_DefaultPagination(t *testing.T) {
	var capturedLimit, capturedOffset int
	mock := &mockAdminUserService{
		listUsersFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			capturedLimit = limit
			capturedOffset = offset
			return nil, nil
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodGet, "/api/admin/users", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if capturedLimit != 50 {
		t.Errorf("expected default limit=50, got %d", capturedLimit)
	}
	if capturedOffset != 0 {
		t.Errorf("expected default offset=0, got %d", capturedOffset)
	}
}

func TestAdminUserHandler_List_ServiceError(t *testing.T) {
	mock := &mockAdminUserService{
		listUsersFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodGet, "/api/admin/users", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/admin/users/:id/suspend tests
// ---------------------------------------------------------------------------

func TestAdminUserHandler_Suspend_Success(t *testing.T) {
	var capturedID string
	var capturedSuspend bool
	mock := &mockAdminUserService{
		suspendFunc: func(ctx context.Context, id string, suspend bool) error {
			capturedID = id
			capturedSuspend = suspend
			return nil
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodPatch, "/api/admin/users/user-1/suspend", `{"suspended":true}`)
	req.SetPathValue("id", "user-1")
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "user-1" {
		t.Errorf("expected id=user-1, got %q", capturedID)
	}
	if !capturedSuspend {
		t.Error("expected suspend=true")
	}
}

func TestAdminUserHandler_Suspend_Unsuspend(t *testing.T) {
	var capturedSuspend bool
	mock := &mockAdminUserService{
		suspendFunc: func(ctx context.Context, id string, suspend bool) error {
			capturedSuspend = suspend
			return nil
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodPatch, "/api/admin/users/user-1/suspend", `{"suspended":false}`)
	req.SetPathValue("id", "user-1")
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedSuspend {
		t.Error("expected suspend=false")
	}
}

func TestAdminUserHandler_Suspend_NotFound(t *testing.T) {
	mock := &mockAdminUserService{
		suspendFunc: func(ctx context.Context, id string, suspend bool) error {
			return repository.ErrNotFound
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodPatch, "/api/admin/users/no-such/suspend", `{"suspended":true}`)
	req.SetPathValue("id", "no-such")
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAdminUserHandler_Suspend_RequiresAuth(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1/suspend",
		strings.NewReader(`{"suspended":true}`))
	req.SetPathValue("id", "u1")
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAdminUserHandler_Suspend_RequiresHost(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1/suspend",
		strings.NewReader(`{"suspended":true}`))
	req.SetPathValue("id", "u1")
	ctx := auth.WithUserID(req.Context(), "regular-user")
	ctx = auth.WithIsHost(ctx, false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAdminUserHandler_Suspend_InvalidJSON(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := adminHostRequest(http.MethodPatch, "/api/admin/users/u1/suspend", `{bad`)
	req.SetPathValue("id", "u1")
	rec := httptest.NewRecorder()
	h.Suspend(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/admin/disclosure-export tests
// ---------------------------------------------------------------------------

func TestAdminUserHandler_DisclosureExport_UserType(t *testing.T) {
	now := time.Now()
	user := &model.User{ID: "u1", Email: "a@b.com", Name: "Alice", CreatedAt: now}
	mock := &mockAdminUserService{
		getUserFunc: func(ctx context.Context, id string) (*model.User, error) {
			if id == "u1" {
				return user, nil
			}
			return nil, repository.ErrNotFound
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})

	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?type=user&id=u1", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "u1" {
		t.Errorf("expected id=u1 in response, got %v", resp["id"])
	}
}

func TestAdminUserHandler_DisclosureExport_ProjectType(t *testing.T) {
	now := time.Now()
	project := &model.Project{ID: "p1", Name: "Test Project", CreatedAt: now}
	mockProject := &mockProjectServiceForAdmin{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			if id == "p1" {
				return project, nil
			}
			return nil, repository.ErrNotFound
		},
	}
	h := NewAdminUserHandler(&mockAdminUserService{}, mockProject)

	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?type=project&id=p1", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "p1" {
		t.Errorf("expected id=p1 in response, got %v", resp["id"])
	}
}

func TestAdminUserHandler_DisclosureExport_MissingType(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?id=u1", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing type, got %d", rec.Code)
	}
}

func TestAdminUserHandler_DisclosureExport_MissingID(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?type=user", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", rec.Code)
	}
}

func TestAdminUserHandler_DisclosureExport_InvalidType(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?type=donation&id=x", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", rec.Code)
	}
}

func TestAdminUserHandler_DisclosureExport_NotFound(t *testing.T) {
	mock := &mockAdminUserService{
		getUserFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewAdminUserHandler(mock, &mockProjectServiceForAdmin{})
	req := adminHostRequest(http.MethodGet, "/api/admin/disclosure-export?type=user&id=no-such", "")
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAdminUserHandler_DisclosureExport_RequiresHost(t *testing.T) {
	h := NewAdminUserHandler(&mockAdminUserService{}, &mockProjectServiceForAdmin{})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/disclosure-export?type=user&id=u1", nil)
	rec := httptest.NewRecorder()
	h.DisclosureExport(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
