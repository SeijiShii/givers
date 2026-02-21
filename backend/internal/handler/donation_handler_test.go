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
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock DonationService
// ---------------------------------------------------------------------------

type mockDonationService struct {
	listByUserFunc  func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	patchFunc       func(ctx context.Context, id, userID string, patch model.DonationPatch) error
	deleteFunc      func(ctx context.Context, id, userID string) error
	migrateFunc     func(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error)
}

func (m *mockDonationService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockDonationService) Patch(ctx context.Context, id, userID string, patch model.DonationPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, userID, patch)
	}
	return nil
}
func (m *mockDonationService) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}
func (m *mockDonationService) MigrateToken(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error) {
	if m.migrateFunc != nil {
		return m.migrateFunc(ctx, token, userID)
	}
	return nil, nil
}

// helper: auth request for regular user
func userAuthRequest(method, url, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(r.Context(), "user-1")
	ctx = auth.WithIsHost(ctx, false)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// GET /api/me/donations tests
// ---------------------------------------------------------------------------

func TestDonationHandler_List_RequiresAuth(t *testing.T) {
	h := NewDonationHandler(&mockDonationService{})
	req := httptest.NewRequest(http.MethodGet, "/api/me/donations", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDonationHandler_List_Success(t *testing.T) {
	now := time.Now()
	donations := []*model.Donation{
		{ID: "d1", ProjectID: "p1", Amount: 1000, Currency: "jpy", CreatedAt: now},
	}
	mock := &mockDonationService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
			return donations, nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/donations", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Donations []*model.Donation `json:"donations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Donations) != 1 {
		t.Errorf("expected 1 donation, got %d", len(resp.Donations))
	}
}

func TestDonationHandler_List_EmptyReturnsEmptyArray(t *testing.T) {
	mock := &mockDonationService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
			return nil, nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/donations", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	var resp struct {
		Donations []*model.Donation `json:"donations"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Donations == nil {
		t.Error("expected non-nil donations slice, got nil")
	}
}

func TestDonationHandler_List_ServiceError(t *testing.T) {
	mock := &mockDonationService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/donations", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/me/donations/:id tests
// ---------------------------------------------------------------------------

func TestDonationHandler_Patch_RequiresAuth(t *testing.T) {
	h := NewDonationHandler(&mockDonationService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/me/donations/d1",
		strings.NewReader(`{"amount":2000}`))
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDonationHandler_Patch_Success(t *testing.T) {
	var capturedID, capturedUserID string
	var capturedPatch model.DonationPatch

	mock := &mockDonationService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.DonationPatch) error {
			capturedID = id
			capturedUserID = userID
			capturedPatch = patch
			return nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/donations/d1", `{"amount":2000,"paused":true}`)
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "d1" {
		t.Errorf("expected id=d1, got %q", capturedID)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedPatch.Amount == nil || *capturedPatch.Amount != 2000 {
		t.Error("expected amount=2000 in patch")
	}
}

func TestDonationHandler_Patch_Forbidden(t *testing.T) {
	mock := &mockDonationService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.DonationPatch) error {
			return service.ErrForbidden
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/donations/d1", `{"amount":2000}`)
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestDonationHandler_Patch_NotFound(t *testing.T) {
	mock := &mockDonationService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.DonationPatch) error {
			return repository.ErrNotFound
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/donations/d1", `{"amount":2000}`)
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDonationHandler_Patch_InvalidJSON(t *testing.T) {
	h := NewDonationHandler(&mockDonationService{})
	req := userAuthRequest(http.MethodPatch, "/api/me/donations/d1", `{bad`)
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/me/donations/:id tests
// ---------------------------------------------------------------------------

func TestDonationHandler_Delete_RequiresAuth(t *testing.T) {
	h := NewDonationHandler(&mockDonationService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/me/donations/d1", nil)
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDonationHandler_Delete_Success(t *testing.T) {
	var capturedID, capturedUserID string
	mock := &mockDonationService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			capturedID = id
			capturedUserID = userID
			return nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/donations/d1", "")
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "d1" || capturedUserID != "user-1" {
		t.Errorf("unexpected id=%q or userID=%q", capturedID, capturedUserID)
	}
}

func TestDonationHandler_Delete_Forbidden(t *testing.T) {
	mock := &mockDonationService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			return service.ErrForbidden
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/donations/d1", "")
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestDonationHandler_Delete_NotFound(t *testing.T) {
	mock := &mockDonationService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			return repository.ErrNotFound
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/donations/d1", "")
	req.SetPathValue("id", "d1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/me/migrate-from-token tests
// ---------------------------------------------------------------------------

func TestDonationHandler_MigrateToken_RequiresAuth(t *testing.T) {
	h := NewDonationHandler(&mockDonationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/me/migrate-from-token", nil)
	rec := httptest.NewRecorder()
	h.MigrateFromToken(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDonationHandler_MigrateToken_Success(t *testing.T) {
	mock := &mockDonationService{
		migrateFunc: func(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error) {
			return &service.MigrateTokenResult{MigratedCount: 2, AlreadyMigrated: false}, nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPost, "/api/me/migrate-from-token", "")
	req.AddCookie(&http.Cookie{Name: "donor_token", Value: "token-abc"})
	rec := httptest.NewRecorder()
	h.MigrateFromToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		MigratedCount   int  `json:"migrated_count"`
		AlreadyMigrated bool `json:"already_migrated"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.MigratedCount != 2 {
		t.Errorf("expected migrated_count=2, got %d", resp.MigratedCount)
	}
	if resp.AlreadyMigrated {
		t.Error("expected already_migrated=false")
	}
}

func TestDonationHandler_MigrateToken_AlreadyMigrated(t *testing.T) {
	mock := &mockDonationService{
		migrateFunc: func(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error) {
			return &service.MigrateTokenResult{MigratedCount: 0, AlreadyMigrated: true}, nil
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPost, "/api/me/migrate-from-token", "")
	req.AddCookie(&http.Cookie{Name: "donor_token", Value: "token-abc"})
	rec := httptest.NewRecorder()
	h.MigrateFromToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (idempotent), got %d", rec.Code)
	}
	var resp struct {
		AlreadyMigrated bool `json:"already_migrated"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.AlreadyMigrated {
		t.Error("expected already_migrated=true")
	}
}

func TestDonationHandler_MigrateToken_MissingCookie(t *testing.T) {
	mock := &mockDonationService{
		migrateFunc: func(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error) {
			return nil, errors.New("donor_token is required")
		},
	}
	h := NewDonationHandler(mock)

	req := userAuthRequest(http.MethodPost, "/api/me/migrate-from-token", "")
	// No donor_token cookie
	rec := httptest.NewRecorder()
	h.MigrateFromToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing cookie, got %d", rec.Code)
	}
}
