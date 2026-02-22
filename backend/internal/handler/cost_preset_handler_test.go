package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock CostPresetService
// ---------------------------------------------------------------------------

type mockCostPresetService struct {
	listFunc   func(ctx context.Context, userID string) ([]*model.CostPreset, error)
	createFunc func(ctx context.Context, userID, label, unitType string) (*model.CostPreset, error)
	updateFunc func(ctx context.Context, id, userID string, patch model.CostPresetPatch) error
	deleteFunc func(ctx context.Context, id, userID string) error
	reorderFunc func(ctx context.Context, userID string, ids []string) error
}

func (m *mockCostPresetService) List(ctx context.Context, userID string) ([]*model.CostPreset, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID)
	}
	return nil, nil
}
func (m *mockCostPresetService) Create(ctx context.Context, userID, label, unitType string) (*model.CostPreset, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, label, unitType)
	}
	return nil, nil
}
func (m *mockCostPresetService) Update(ctx context.Context, id, userID string, patch model.CostPresetPatch) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, userID, patch)
	}
	return nil
}
func (m *mockCostPresetService) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}
func (m *mockCostPresetService) Reorder(ctx context.Context, userID string, ids []string) error {
	if m.reorderFunc != nil {
		return m.reorderFunc(ctx, userID, ids)
	}
	return nil
}

// ---------------------------------------------------------------------------
// GET /api/me/cost-presets
// ---------------------------------------------------------------------------

func TestCostPresetHandler_List_RequiresAuth(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := httptest.NewRequest(http.MethodGet, "/api/me/cost-presets", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCostPresetHandler_List_Success(t *testing.T) {
	presets := []*model.CostPreset{
		{ID: "p1", Label: "サーバー費用", UnitType: "monthly"},
	}
	mock := &mockCostPresetService{
		listFunc: func(_ context.Context, userID string) ([]*model.CostPreset, error) {
			return presets, nil
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/cost-presets", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Presets []*model.CostPreset `json:"presets"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Presets) != 1 || resp.Presets[0].ID != "p1" {
		t.Errorf("expected 1 preset, got %v", resp.Presets)
	}
}

func TestCostPresetHandler_List_ServiceError(t *testing.T) {
	mock := &mockCostPresetService{
		listFunc: func(_ context.Context, _ string) ([]*model.CostPreset, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodGet, "/api/me/cost-presets", "")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/me/cost-presets
// ---------------------------------------------------------------------------

func TestCostPresetHandler_Create_RequiresAuth(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := httptest.NewRequest(http.MethodPost, "/api/me/cost-presets",
		strings.NewReader(`{"label":"X","unit_type":"monthly"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Create_Success(t *testing.T) {
	var capturedLabel, capturedUnitType string
	mock := &mockCostPresetService{
		createFunc: func(_ context.Context, userID, label, unitType string) (*model.CostPreset, error) {
			capturedLabel = label
			capturedUnitType = unitType
			return &model.CostPreset{ID: "new-id", UserID: userID, Label: label, UnitType: unitType}, nil
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodPost, "/api/me/cost-presets", `{"label":"カスタム費用","unit_type":"daily_x_days"}`)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedLabel != "カスタム費用" || capturedUnitType != "daily_x_days" {
		t.Errorf("unexpected label=%q unitType=%q", capturedLabel, capturedUnitType)
	}
}

func TestCostPresetHandler_Create_LabelRequired(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := userAuthRequest(http.MethodPost, "/api/me/cost-presets", `{"unit_type":"monthly"}`)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing label, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Create_ServiceError(t *testing.T) {
	mock := &mockCostPresetService{
		createFunc: func(_ context.Context, _, _, _ string) (*model.CostPreset, error) {
			return nil, errors.New("invalid unit_type")
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodPost, "/api/me/cost-presets", `{"label":"X","unit_type":"bad"}`)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/me/cost-presets/{id}
// ---------------------------------------------------------------------------

func TestCostPresetHandler_Update_RequiresAuth(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := httptest.NewRequest(http.MethodPut, "/api/me/cost-presets/p1",
		strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Update_Success(t *testing.T) {
	var capturedPatch model.CostPresetPatch
	mock := &mockCostPresetService{
		updateFunc: func(_ context.Context, id, userID string, patch model.CostPresetPatch) error {
			capturedPatch = patch
			return nil
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodPut, "/api/me/cost-presets/p1", `{"label":"Updated"}`)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedPatch.Label == nil || *capturedPatch.Label != "Updated" {
		t.Errorf("expected label=Updated in patch, got %v", capturedPatch)
	}
}

func TestCostPresetHandler_Update_Forbidden(t *testing.T) {
	mock := &mockCostPresetService{
		updateFunc: func(_ context.Context, _, _ string, _ model.CostPresetPatch) error {
			return service.ErrForbidden
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodPut, "/api/me/cost-presets/p1", `{"label":"X"}`)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/me/cost-presets/{id}
// ---------------------------------------------------------------------------

func TestCostPresetHandler_Delete_RequiresAuth(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/me/cost-presets/p1", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Delete_Success(t *testing.T) {
	var capturedID string
	mock := &mockCostPresetService{
		deleteFunc: func(_ context.Context, id, _ string) error {
			capturedID = id
			return nil
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/cost-presets/p1", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedID != "p1" {
		t.Errorf("expected id=p1, got %q", capturedID)
	}
}

func TestCostPresetHandler_Delete_Forbidden(t *testing.T) {
	mock := &mockCostPresetService{
		deleteFunc: func(_ context.Context, _, _ string) error {
			return service.ErrForbidden
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodDelete, "/api/me/cost-presets/p1", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Delete(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/me/cost-presets/reorder
// ---------------------------------------------------------------------------

func TestCostPresetHandler_Reorder_RequiresAuth(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := httptest.NewRequest(http.MethodPut, "/api/me/cost-presets/reorder",
		strings.NewReader(`{"ids":["p1","p2"]}`))
	rec := httptest.NewRecorder()
	h.Reorder(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Reorder_Success(t *testing.T) {
	var capturedIDs []string
	mock := &mockCostPresetService{
		reorderFunc: func(_ context.Context, _ string, ids []string) error {
			capturedIDs = ids
			return nil
		},
	}
	h := NewCostPresetHandler(mock)

	req := userAuthRequest(http.MethodPut, "/api/me/cost-presets/reorder", `{"ids":["p3","p1","p2"]}`)
	rec := httptest.NewRecorder()
	h.Reorder(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if len(capturedIDs) != 3 || capturedIDs[0] != "p3" {
		t.Errorf("unexpected ids: %v", capturedIDs)
	}
}

func TestCostPresetHandler_Reorder_InvalidJSON(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := userAuthRequest(http.MethodPut, "/api/me/cost-presets/reorder", `{bad`)
	rec := httptest.NewRecorder()
	h.Reorder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCostPresetHandler_Reorder_EmptyIDs(t *testing.T) {
	h := NewCostPresetHandler(&mockCostPresetService{})
	req := userAuthRequest(http.MethodPut, "/api/me/cost-presets/reorder", `{"ids":[]}`)
	rec := httptest.NewRecorder()
	h.Reorder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty ids, got %d", rec.Code)
	}
}

// helper: auth request with userID from context (auth.WithIsHost not set)
func costPresetAuthRequest(method, url, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(r.Context(), "user-1")
	return r.WithContext(ctx)
}
