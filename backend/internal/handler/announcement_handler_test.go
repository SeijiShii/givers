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
// mockAnnouncementService
// ---------------------------------------------------------------------------

type mockAnnouncementService struct {
	createFunc        func(ctx context.Context, a *model.Announcement) error
	getByIDFunc       func(ctx context.Context, id string) (*model.Announcement, error)
	updateFunc        func(ctx context.Context, a *model.Announcement) error
	setVisibilityFunc func(ctx context.Context, id string, visible bool) error
	listVisibleFunc   func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error)
	listAllFunc       func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	markReadFunc      func(ctx context.Context, userID, announcementID string) error
	unreadCountFunc   func(ctx context.Context, userID string) (int, error)
}

func (m *mockAnnouncementService) Create(ctx context.Context, a *model.Announcement) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, a)
	}
	return nil
}

func (m *mockAnnouncementService) GetByID(ctx context.Context, id string) (*model.Announcement, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAnnouncementService) Update(ctx context.Context, a *model.Announcement) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, a)
	}
	return nil
}

func (m *mockAnnouncementService) SetVisibility(ctx context.Context, id string, visible bool) error {
	if m.setVisibilityFunc != nil {
		return m.setVisibilityFunc(ctx, id, visible)
	}
	return nil
}

func (m *mockAnnouncementService) ListVisible(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
	if m.listVisibleFunc != nil {
		return m.listVisibleFunc(ctx, userID, limit, cursor)
	}
	return []*model.Announcement{}, "", nil
}

func (m *mockAnnouncementService) ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	if m.listAllFunc != nil {
		return m.listAllFunc(ctx, limit, cursor)
	}
	return []*model.Announcement{}, "", nil
}

func (m *mockAnnouncementService) MarkRead(ctx context.Context, userID, announcementID string) error {
	if m.markReadFunc != nil {
		return m.markReadFunc(ctx, userID, announcementID)
	}
	return nil
}

func (m *mockAnnouncementService) UnreadCount(ctx context.Context, userID string) (int, error) {
	if m.unreadCountFunc != nil {
		return m.unreadCountFunc(ctx, userID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newAnnouncementMux(h *AnnouncementHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/announcements", h.List)
	mux.HandleFunc("GET /api/announcements/unread-count", h.UnreadCount)
	mux.HandleFunc("POST /api/announcements/{id}/read", h.MarkRead)
	mux.HandleFunc("GET /api/admin/announcements", h.AdminList)
	mux.HandleFunc("POST /api/admin/announcements", h.AdminCreate)
	mux.HandleFunc("PUT /api/admin/announcements/{id}", h.AdminUpdate)
	mux.HandleFunc("PATCH /api/admin/announcements/{id}/visibility", h.AdminSetVisibility)
	return mux
}

func hostCtx(ctx context.Context, userID string) context.Context {
	ctx = auth.WithUserID(ctx, userID)
	ctx = auth.WithIsHost(ctx, true)
	return ctx
}

// ---------------------------------------------------------------------------
// GET /api/announcements — List
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_List_ReturnsAnnouncements(t *testing.T) {
	now := time.Now()
	want := []*model.Announcement{
		{ID: "a1", Title: "Test 1", Severity: "info", PublishedAt: now},
		{ID: "a2", Title: "Test 2", Severity: "warn", PublishedAt: now},
	}
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			return want, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Announcements []*model.Announcement `json:"announcements"`
		NextCursor    *string               `json:"next_cursor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Announcements) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Announcements))
	}
}

func TestAnnouncementHandler_List_PassesUserID(t *testing.T) {
	var capturedUserID string
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			capturedUserID = userID
			return []*model.Announcement{}, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
}

func TestAnnouncementHandler_List_EmptyUserIDForUnauthenticated(t *testing.T) {
	var capturedUserID string
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			capturedUserID = userID
			return []*model.Announcement{}, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedUserID != "" {
		t.Errorf("expected empty userID for unauthenticated, got %q", capturedUserID)
	}
}

func TestAnnouncementHandler_List_ServiceError_Returns500(t *testing.T) {
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			return nil, "", errors.New("db error")
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_List_RespectsLimitParam(t *testing.T) {
	var capturedLimit int
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			capturedLimit = limit
			return []*model.Announcement{}, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements?limit=5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedLimit != 5 {
		t.Errorf("expected limit=5, got %d", capturedLimit)
	}
}

func TestAnnouncementHandler_List_DefaultLimit(t *testing.T) {
	var capturedLimit int
	svc := &mockAnnouncementService{
		listVisibleFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
			capturedLimit = limit
			return []*model.Announcement{}, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if capturedLimit != 20 {
		t.Errorf("expected default limit=20, got %d", capturedLimit)
	}
}

// ---------------------------------------------------------------------------
// GET /api/announcements/unread-count
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_UnreadCount_Success(t *testing.T) {
	svc := &mockAnnouncementService{
		unreadCountFunc: func(ctx context.Context, userID string) (int, error) {
			return 3, nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements/unread-count", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 3 {
		t.Errorf("expected count=3, got %d", resp.Count)
	}
}

func TestAnnouncementHandler_UnreadCount_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/announcements/unread-count", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/announcements/{id}/read — MarkRead
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_MarkRead_Success(t *testing.T) {
	var capturedAID string
	svc := &mockAnnouncementService{
		markReadFunc: func(ctx context.Context, userID, announcementID string) error {
			capturedAID = announcementID
			return nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPost, "/api/announcements/a1/read", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedAID != "a1" {
		t.Errorf("expected announcementID=a1, got %q", capturedAID)
	}
}

func TestAnnouncementHandler_MarkRead_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPost, "/api/announcements/a1/read", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/admin/announcements — AdminCreate
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_AdminCreate_Success(t *testing.T) {
	var captured *model.Announcement
	svc := &mockAnnouncementService{
		createFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			a.ID = "new-id"
			return nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Maintenance", "body": "Server maintenance tomorrow", "severity": "warn"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatal("expected Create to be called")
	}
	if captured.Title != "Maintenance" {
		t.Errorf("expected Title=Maintenance, got %q", captured.Title)
	}
	if captured.Severity != "warn" {
		t.Errorf("expected Severity=warn, got %q", captured.Severity)
	}
	if captured.AuthorID != "host-1" {
		t.Errorf("expected AuthorID=host-1, got %q", captured.AuthorID)
	}
}

func TestAnnouncementHandler_AdminCreate_WithScheduledPublishedAt(t *testing.T) {
	var captured *model.Announcement
	svc := &mockAnnouncementService{
		createFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			a.ID = "new-id"
			return nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Scheduled", "body": "...", "published_at": "2026-03-10T09:00:00+09:00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatal("expected Create to be called")
	}
	if captured.PublishedAt.IsZero() {
		t.Error("expected PublishedAt to be set for scheduled post")
	}
}

func TestAnnouncementHandler_AdminCreate_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminCreate_ForbiddenForNonHost(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), "regular-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminCreate_TitleRequired(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"body": "no title"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminCreate_InvalidJSON(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader("{bad"))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminCreate_InvalidSeverity(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Test", "severity": "critical"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid severity, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminCreate_ServiceError(t *testing.T) {
	svc := &mockAnnouncementService{
		createFunc: func(ctx context.Context, a *model.Announcement) error {
			return errors.New("db error")
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/admin/announcements/{id} — AdminUpdate
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_AdminUpdate_Success(t *testing.T) {
	existing := &model.Announcement{ID: "a1", Title: "Old", Body: "Old body", Severity: "info", Visible: true}
	var captured *model.Announcement
	svc := &mockAnnouncementService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Announcement, error) {
			return existing, nil
		},
		updateFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			return nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"title": "New Title", "severity": "warn"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/announcements/a1", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatal("expected Update to be called")
	}
	if captured.Title != "New Title" {
		t.Errorf("expected Title=New Title, got %q", captured.Title)
	}
	if captured.Severity != "warn" {
		t.Errorf("expected Severity=warn, got %q", captured.Severity)
	}
	// Body should stay unchanged (partial update)
	if captured.Body != "Old body" {
		t.Errorf("expected Body=Old body (unchanged), got %q", captured.Body)
	}
}

func TestAnnouncementHandler_AdminUpdate_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/announcements/a1", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminUpdate_ForbiddenForNonHost(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/announcements/a1", strings.NewReader(`{}`))
	req = req.WithContext(auth.WithUserID(req.Context(), "regular-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminUpdate_NotFound(t *testing.T) {
	svc := &mockAnnouncementService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Announcement, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/announcements/nonexistent", strings.NewReader(`{}`))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/admin/announcements/{id}/visibility — AdminSetVisibility
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_AdminSetVisibility_Hide(t *testing.T) {
	var capturedID string
	var capturedVisible bool
	svc := &mockAnnouncementService{
		setVisibilityFunc: func(ctx context.Context, id string, visible bool) error {
			capturedID = id
			capturedVisible = visible
			return nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	body := `{"visible": false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/announcements/a1/visibility", strings.NewReader(body))
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "a1" {
		t.Errorf("expected id=a1, got %q", capturedID)
	}
	if capturedVisible {
		t.Error("expected visible=false, got true")
	}
}

func TestAnnouncementHandler_AdminSetVisibility_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/announcements/a1/visibility", strings.NewReader(`{"visible":false}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminSetVisibility_ForbiddenForNonHost(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/announcements/a1/visibility", strings.NewReader(`{"visible":false}`))
	req = req.WithContext(auth.WithUserID(req.Context(), "regular-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/admin/announcements — AdminList
// ---------------------------------------------------------------------------

func TestAnnouncementHandler_AdminList_Success(t *testing.T) {
	want := []*model.Announcement{
		{ID: "a1", Title: "Test", Severity: "info", Visible: true},
	}
	svc := &mockAnnouncementService{
		listAllFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return want, "", nil
		},
	}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/announcements", nil)
	req = req.WithContext(hostCtx(req.Context(), "host-1"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminList_Unauthorized(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/announcements", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAnnouncementHandler_AdminList_ForbiddenForNonHost(t *testing.T) {
	svc := &mockAnnouncementService{}
	h := NewAnnouncementHandler(svc)
	mux := newAnnouncementMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/announcements", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), "regular-user"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}
