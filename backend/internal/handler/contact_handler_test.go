package handler

import (
	"bytes"
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
// Mock ContactService
// ---------------------------------------------------------------------------

type mockContactService struct {
	submitFunc func(ctx context.Context, msg *model.ContactMessage) error
	listFunc   func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error)
}

func (m *mockContactService) Submit(ctx context.Context, msg *model.ContactMessage) error {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, msg)
	}
	return nil
}

func (m *mockContactService) List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, opts)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// POST /api/contact tests
// ---------------------------------------------------------------------------

func TestContactHandler_Submit_Success(t *testing.T) {
	var captured *model.ContactMessage
	mock := &mockContactService{
		submitFunc: func(ctx context.Context, msg *model.ContactMessage) error {
			captured = msg
			return nil
		},
	}
	h := NewContactHandler(mock)

	body := `{"email":"test@example.com","name":"Alice","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatal("expected Submit to be called with a ContactMessage, got nil")
	}
	if captured.Email != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %q", captured.Email)
	}
	if captured.Name != "Alice" {
		t.Errorf("expected name=Alice, got %q", captured.Name)
	}
	if captured.Message != "Hello!" {
		t.Errorf("expected message=Hello!, got %q", captured.Message)
	}
}

// TestContactHandler_Submit_EmailRequired verifies that omitting email returns 400.
func TestContactHandler_Submit_EmailRequired(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"name":"Bob","message":"Hi there"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error field in response body")
	}
}

// TestContactHandler_Submit_MessageRequired verifies that omitting message returns 400.
func TestContactHandler_Submit_MessageRequired(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestContactHandler_Submit_NameOptional verifies that name is optional.
func TestContactHandler_Submit_NameOptional(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"email":"anon@example.com","message":"Anonymous message"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 (name is optional), got %d — body: %s", rec.Code, rec.Body.String())
	}
}

// TestContactHandler_Submit_MessageTooLong verifies that messages exceeding 5000 chars return 400.
func TestContactHandler_Submit_MessageTooLong(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	longMsg := strings.Repeat("a", 5001)
	body, _ := json.Marshal(map[string]string{
		"email":   "test@example.com",
		"message": longMsg,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for message > 5000 chars, got %d", rec.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "message_too_long" {
		t.Errorf("expected error=message_too_long, got %q", resp["error"])
	}
}

// TestContactHandler_Submit_MessageAtMaxLength verifies 5000 char message is accepted.
func TestContactHandler_Submit_MessageAtMaxLength(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	maxMsg := strings.Repeat("x", 5000)
	body, _ := json.Marshal(map[string]string{
		"email":   "test@example.com",
		"message": maxMsg,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 at exactly 5000 chars, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

// TestContactHandler_Submit_InvalidJSON verifies that malformed JSON returns 400.
func TestContactHandler_Submit_InvalidJSON(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// TestContactHandler_Submit_ServiceError verifies that a service failure returns 500.
func TestContactHandler_Submit_ServiceError(t *testing.T) {
	mock := &mockContactService{
		submitFunc: func(ctx context.Context, msg *model.ContactMessage) error {
			return errors.New("db connection lost")
		},
	}
	h := NewContactHandler(mock)

	body := `{"email":"test@example.com","message":"Hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

// TestContactHandler_Submit_EmptyEmail verifies that empty email string returns 400.
func TestContactHandler_Submit_EmptyEmail(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"email":"","message":"Hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty email, got %d", rec.Code)
	}
}

// TestContactHandler_Submit_EmptyMessage verifies that empty message string returns 400.
func TestContactHandler_Submit_EmptyMessage(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"email":"test@example.com","message":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty message, got %d", rec.Code)
	}
}

// TestContactHandler_Submit_ContentTypeJSON verifies the response Content-Type header.
func TestContactHandler_Submit_ContentTypeJSON(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	body := `{"email":"t@e.com","message":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// GET /api/admin/contacts tests
// ---------------------------------------------------------------------------

// TestAdminContactsHandler_ListAll_HostOnly verifies that a non-host user gets 403.
func TestAdminContactsHandler_ListAll_RequiresHost(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	// No user in context at all → 401
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 (no auth), got %d", rec.Code)
	}
}

// TestAdminContactsHandler_ListAll_NonHost_Returns403 verifies that a non-host user gets 403.
func TestAdminContactsHandler_ListAll_NonHost_Returns403(t *testing.T) {
	mock := &mockContactService{}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	// Non-host user in context
	ctx := auth.WithUserID(req.Context(), "regular-user-id")
	ctx = auth.WithIsHost(ctx, false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-host user, got %d", rec.Code)
	}
}

// TestAdminContactsHandler_ListAll_Success verifies host can list all contacts.
func TestAdminContactsHandler_ListAll_Success(t *testing.T) {
	now := time.Now()
	messages := []*model.ContactMessage{
		{ID: "1", Email: "a@b.com", Name: "Alice", Message: "Hi", Status: "unread", CreatedAt: now},
		{ID: "2", Email: "c@d.com", Name: "", Message: "Hello", Status: "read", CreatedAt: now},
	}
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			return messages, nil
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Messages []*model.ContactMessage `json:"messages"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(resp.Messages))
	}
}

// TestAdminContactsHandler_ListUnread_Filter verifies status filter is forwarded to service.
func TestAdminContactsHandler_ListUnread_Filter(t *testing.T) {
	var capturedOpts model.ContactListOptions
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			capturedOpts = opts
			return nil, nil
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts?status=unread", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedOpts.Status != "unread" {
		t.Errorf("expected status=unread forwarded to service, got %q", capturedOpts.Status)
	}
}

// TestAdminContactsHandler_Pagination verifies limit/offset are forwarded to service.
func TestAdminContactsHandler_Pagination(t *testing.T) {
	var capturedOpts model.ContactListOptions
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			capturedOpts = opts
			return nil, nil
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts?limit=10&offset=20", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedOpts.Limit != 10 {
		t.Errorf("expected limit=10, got %d", capturedOpts.Limit)
	}
	if capturedOpts.Offset != 20 {
		t.Errorf("expected offset=20, got %d", capturedOpts.Offset)
	}
}

// TestAdminContactsHandler_ServiceError verifies 500 on service failure.
func TestAdminContactsHandler_ServiceError(t *testing.T) {
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			return nil, errors.New("database error")
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on service error, got %d", rec.Code)
	}
}

// TestAdminContactsHandler_DefaultPagination verifies default limit when not specified.
func TestAdminContactsHandler_DefaultPagination(t *testing.T) {
	var capturedOpts model.ContactListOptions
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			capturedOpts = opts
			return nil, nil
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if capturedOpts.Limit != 20 {
		t.Errorf("expected default limit=20, got %d", capturedOpts.Limit)
	}
	if capturedOpts.Offset != 0 {
		t.Errorf("expected default offset=0, got %d", capturedOpts.Offset)
	}
}

// TestAdminContactsHandler_EmptyList verifies empty list returns [] not null.
func TestAdminContactsHandler_EmptyList(t *testing.T) {
	mock := &mockContactService{
		listFunc: func(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
			return []*model.ContactMessage{}, nil
		},
	}
	h := NewContactHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts", nil)
	ctx := auth.WithUserID(req.Context(), "host-user-id")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AdminList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Messages []*model.ContactMessage `json:"messages"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Messages == nil {
		t.Error("expected non-nil (empty) messages slice, got nil")
	}
}
