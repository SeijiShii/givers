package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockDB struct {
	pingFunc func(ctx context.Context) error
}

func (m *mockDB) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func TestHealth_OK(t *testing.T) {
	h := New(&mockDB{}, "http://localhost:3000")
	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %q", resp.Status)
	}
	if resp.Message != "GIVErS API" {
		t.Errorf("expected message='GIVErS API', got %q", resp.Message)
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	h := New(&mockDB{
		pingFunc: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}, "http://localhost:3000")

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "unhealthy" {
		t.Errorf("expected status=unhealthy, got %q", resp.Status)
	}
}
