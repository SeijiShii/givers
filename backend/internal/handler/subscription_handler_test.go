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
)

// ---------------------------------------------------------------------------
// Mock SubscriptionService
// ---------------------------------------------------------------------------

type mockSubscriptionService struct {
	listByUserFunc func(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error)
	patchFunc      func(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error
	deleteFunc     func(ctx context.Context, id, userID string) error
}

func (m *mockSubscriptionService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockSubscriptionService) GetByID(_ context.Context, _, _ string) (*model.Subscription, error) {
	return nil, nil
}
func (m *mockSubscriptionService) Patch(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, userID, patch)
	}
	return nil
}
func (m *mockSubscriptionService) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// GET /api/me/subscriptions tests
// ---------------------------------------------------------------------------

func TestSubscriptionHandler_List_RequiresAuth(t *testing.T) {
	h := NewSubscriptionHandler(&mockSubscriptionService{})
	req := httptest.NewRequest(http.MethodGet, "/api/me/subscriptions", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_List_Success(t *testing.T) {
	now := time.Now()
	subs := []*model.Subscription{
		{ID: "s1", ProjectID: "p1", Amount: 1000, Currency: "jpy", CreatedAt: now, UpdatedAt: now},
	}
	mock := &mockSubscriptionService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
			return subs, nil
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/subscriptions", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Subscriptions []*model.Subscription `json:"subscriptions"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Subscriptions) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(resp.Subscriptions))
	}
}

func TestSubscriptionHandler_List_EmptyReturnsEmptyArray(t *testing.T) {
	mock := &mockSubscriptionService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
			return nil, nil
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/subscriptions", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	var resp struct {
		Subscriptions []*model.Subscription `json:"subscriptions"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Subscriptions == nil {
		t.Error("expected non-nil subscriptions slice, got nil")
	}
}

func TestSubscriptionHandler_List_ServiceError(t *testing.T) {
	mock := &mockSubscriptionService{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/subscriptions", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/me/subscriptions/:id tests
// ---------------------------------------------------------------------------

func TestSubscriptionHandler_Patch_RequiresAuth(t *testing.T) {
	h := NewSubscriptionHandler(&mockSubscriptionService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/me/subscriptions/s1",
		strings.NewReader(`{"amount":2000}`))
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Patch_Success(t *testing.T) {
	var capturedID, capturedUserID string
	var capturedPatch model.SubscriptionPatch

	mock := &mockSubscriptionService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
			capturedID = id
			capturedUserID = userID
			capturedPatch = patch
			return nil
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/subscriptions/s1", `{"amount":2000,"paused":true}`)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "s1" {
		t.Errorf("expected id=s1, got %q", capturedID)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedPatch.Amount == nil || *capturedPatch.Amount != 2000 {
		t.Error("expected amount=2000 in patch")
	}
}

func TestSubscriptionHandler_Patch_Forbidden(t *testing.T) {
	mock := &mockSubscriptionService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
			return service.ErrForbidden
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/subscriptions/s1", `{"amount":2000}`)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Patch_NotFound(t *testing.T) {
	mock := &mockSubscriptionService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
			return repository.ErrNotFound
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/subscriptions/s1", `{"amount":2000}`)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Patch_InvalidJSON(t *testing.T) {
	h := NewSubscriptionHandler(&mockSubscriptionService{})
	req := userAuthRequest(http.MethodPatch, "/api/me/subscriptions/s1", `{bad`)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Patch_NextBillingMessage(t *testing.T) {
	var capturedPatch model.SubscriptionPatch

	mock := &mockSubscriptionService{
		patchFunc: func(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
			capturedPatch = patch
			return nil
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodPatch, "/api/me/subscriptions/s1",
		`{"next_billing_message":"今月もありがとうございます！"}`)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Patch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedPatch.NextBillingMessage == nil {
		t.Fatal("expected NextBillingMessage to be set")
	}
	if *capturedPatch.NextBillingMessage != "今月もありがとうございます！" {
		t.Errorf("expected message='今月もありがとうございます！', got %q", *capturedPatch.NextBillingMessage)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/me/subscriptions/:id tests
// ---------------------------------------------------------------------------

func TestSubscriptionHandler_Delete_RequiresAuth(t *testing.T) {
	h := NewSubscriptionHandler(&mockSubscriptionService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/me/subscriptions/s1", nil)
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Delete_Success(t *testing.T) {
	var capturedID, capturedUserID string
	mock := &mockSubscriptionService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			capturedID = id
			capturedUserID = userID
			return nil
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/subscriptions/s1", "")
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "s1" || capturedUserID != "user-1" {
		t.Errorf("unexpected id=%q or userID=%q", capturedID, capturedUserID)
	}
}

func TestSubscriptionHandler_Delete_Forbidden(t *testing.T) {
	mock := &mockSubscriptionService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			return service.ErrForbidden
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/subscriptions/s1", "")
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestSubscriptionHandler_Delete_NotFound(t *testing.T) {
	mock := &mockSubscriptionService{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			return repository.ErrNotFound
		},
	}
	h := NewSubscriptionHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/subscriptions/s1", "")
	req.SetPathValue("id", "s1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
