package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock PlatformHealthService
// ---------------------------------------------------------------------------

type mockPlatformHealthService struct {
	getFunc func(ctx context.Context) (*model.PlatformHealth, error)
}

func (m *mockPlatformHealthService) Get(ctx context.Context) (*model.PlatformHealth, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// GET /api/host tests
// ---------------------------------------------------------------------------

func TestHostHandler_Get_Success(t *testing.T) {
	health := &model.PlatformHealth{
		MonthlyCost:       50000,
		CurrentMonthly:    28000,
		WarningThreshold:  60,
		CriticalThreshold: 30,
		UpdatedAt:         time.Now(),
	}
	mock := &mockPlatformHealthService{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return health, nil
		},
	}
	h := NewHostHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		MonthlyCost       int    `json:"monthly_cost"`
		CurrentMonthly    int    `json:"current_monthly"`
		WarningThreshold  int    `json:"warning_threshold"`
		CriticalThreshold int    `json:"critical_threshold"`
		Rate              int    `json:"rate"`
		Signal            string `json:"signal"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MonthlyCost != 50000 {
		t.Errorf("expected monthly_cost=50000, got %d", resp.MonthlyCost)
	}
	if resp.CurrentMonthly != 28000 {
		t.Errorf("expected current_monthly=28000, got %d", resp.CurrentMonthly)
	}
	if resp.Rate != 56 {
		t.Errorf("expected rate=56, got %d", resp.Rate)
	}
	if resp.Signal != "yellow" {
		t.Errorf("expected signal=yellow, got %q", resp.Signal)
	}
}

func TestHostHandler_Get_ServiceError_Returns500(t *testing.T) {
	mock := &mockPlatformHealthService{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHostHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHostHandler_Get_ContentTypeJSON(t *testing.T) {
	mock := &mockPlatformHealthService{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return &model.PlatformHealth{WarningThreshold: 60, CriticalThreshold: 30}, nil
		},
	}
	h := NewHostHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

func TestHostHandler_Get_GreenSignal(t *testing.T) {
	mock := &mockPlatformHealthService{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return &model.PlatformHealth{
				MonthlyCost:       100,
				CurrentMonthly:    80,
				WarningThreshold:  60,
				CriticalThreshold: 30,
			}, nil
		},
	}
	h := NewHostHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	var resp struct {
		Signal string `json:"signal"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Signal != "green" {
		t.Errorf("expected signal=green, got %q", resp.Signal)
	}
}

func TestHostHandler_Get_RedSignal(t *testing.T) {
	mock := &mockPlatformHealthService{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return &model.PlatformHealth{
				MonthlyCost:       100,
				CurrentMonthly:    10,
				WarningThreshold:  60,
				CriticalThreshold: 30,
			}, nil
		},
	}
	h := NewHostHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/host", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	var resp struct {
		Signal string `json:"signal"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Signal != "red" {
		t.Errorf("expected signal=red, got %q", resp.Signal)
	}
}
