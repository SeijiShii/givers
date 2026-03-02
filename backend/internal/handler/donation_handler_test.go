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
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock DonationService
// ---------------------------------------------------------------------------

type mockDonationService struct {
	listByUserFunc      func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	migrateFunc         func(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error)
	listProjectMsgsFunc func(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error)
}

func (m *mockDonationService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockDonationService) MigrateToken(ctx context.Context, token, userID string) (*service.MigrateTokenResult, error) {
	if m.migrateFunc != nil {
		return m.migrateFunc(ctx, token, userID)
	}
	return nil, nil
}
func (m *mockDonationService) ListProjectMessages(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
	if m.listProjectMsgsFunc != nil {
		return m.listProjectMsgsFunc(ctx, projectID, limit, offset, sort, donor)
	}
	return &model.DonationMessageResult{Messages: []*model.DonationMessage{}, Total: 0}, nil
}
func (m *mockDonationService) ListByProjectForOwner(_ context.Context, _ string, _, _ int, _, _, _ string) (*model.OwnerDonationResult, error) {
	return &model.OwnerDonationResult{Donations: []*model.OwnerDonationItem{}, Total: 0}, nil
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

