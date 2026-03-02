package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/givers/backend/internal/model"
)

func newBadgeMux(h *BadgeHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects/{id}/badge.svg", h.Badge)
	return mux
}

func TestBadgeHandler_ActiveProject_GreenSignal(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           10000,
				CurrentMonthlyDonations: 8000, // 80% → green (>= default 60)
				Alerts: &model.ProjectAlerts{
					WarningThreshold:  60,
					CriticalThreshold: 30,
				},
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "80%") {
		t.Errorf("expected 80%% in SVG, got %s", body)
	}
	if !strings.Contains(body, "#4c1") {
		t.Errorf("expected green color #4c1, got %s", body)
	}
}

func TestBadgeHandler_ActiveProject_YellowSignal(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           10000,
				CurrentMonthlyDonations: 4500, // 45% → yellow (>= 30, < 60)
				Alerts: &model.ProjectAlerts{
					WarningThreshold:  60,
					CriticalThreshold: 30,
				},
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "45%") {
		t.Errorf("expected 45%% in SVG, got %s", body)
	}
	if !strings.Contains(body, "#dfb317") {
		t.Errorf("expected yellow color #dfb317, got %s", body)
	}
}

func TestBadgeHandler_ActiveProject_RedSignal(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           10000,
				CurrentMonthlyDonations: 1000, // 10% → red (< 30)
				Alerts: &model.ProjectAlerts{
					WarningThreshold:  60,
					CriticalThreshold: 30,
				},
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "10%") {
		t.Errorf("expected 10%% in SVG, got %s", body)
	}
	if !strings.Contains(body, "#e05d44") {
		t.Errorf("expected red color #e05d44, got %s", body)
	}
}

func TestBadgeHandler_ActiveProject_NoTarget(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:            "p1",
				Status:        "active",
				MonthlyTarget: 0,
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "active") {
		t.Errorf("expected 'active' in SVG when no target, got %s", body)
	}
	if !strings.Contains(body, "#4c1") {
		t.Errorf("expected green color for active badge, got %s", body)
	}
}

func TestBadgeHandler_ActiveProject_UsesOwnerWantMonthly(t *testing.T) {
	owm := 5000
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           0,
				OwnerWantMonthly:        &owm,
				CurrentMonthlyDonations: 5000, // 100%
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "100%") {
		t.Errorf("expected 100%% when using OwnerWantMonthly, got %s", body)
	}
}

func TestBadgeHandler_InactiveProject(t *testing.T) {
	for _, status := range []string{"draft", "frozen", "deleted"} {
		t.Run(status, func(t *testing.T) {
			mock := &mockProjectService{
				getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
					return &model.Project{ID: "p1", Status: status}, nil
				},
			}
			h := NewBadgeHandler(mock)
			mux := newBadgeMux(h)

			req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 for %s, got %d", status, rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, "inactive") {
				t.Errorf("expected 'inactive' for status=%s, got %s", status, body)
			}
			if !strings.Contains(body, "#999") {
				t.Errorf("expected gray #999 for inactive, got %s", body)
			}
		})
	}
}

func TestBadgeHandler_ProjectNotFound(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/no-such/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "not found") {
		t.Errorf("expected 'not found' in error badge, got %s", body)
	}
}

func TestBadgeHandler_ContentType(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", Status: "active"}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("expected Content-Type=image/svg+xml, got %q", ct)
	}
}

func TestBadgeHandler_CacheControl(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", Status: "active"}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age=300") {
		t.Errorf("expected Cache-Control with max-age=300, got %q", cc)
	}
}

func TestBadgeHandler_CustomAlertThresholds(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           10000,
				CurrentMonthlyDonations: 7500, // 75%
				Alerts: &model.ProjectAlerts{
					WarningThreshold:  80, // custom: need 80% for green
					CriticalThreshold: 50,
				},
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	// 75% < 80 (custom warning), so should be yellow
	if !strings.Contains(body, "#dfb317") {
		t.Errorf("expected yellow for 75%% with custom warning=80, got %s", body)
	}
}

func TestBadgeHandler_DefaultThresholds(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:                      "p1",
				Status:                  "active",
				MonthlyTarget:           10000,
				CurrentMonthlyDonations: 6500, // 65% → green with default 60
				// no Alerts → use defaults
			}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "#4c1") {
		t.Errorf("expected green for 65%% with default threshold=60, got %s", body)
	}
}

func TestBadgeHandler_SVGContainsGIVErS(t *testing.T) {
	mock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", Status: "active"}, nil
		},
	}
	h := NewBadgeHandler(mock)
	mux := newBadgeMux(h)

	req := httptest.NewRequest("GET", "/api/projects/p1/badge.svg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "GIVErS") {
		t.Errorf("expected 'GIVErS' label in SVG, got %s", body)
	}
	if !strings.Contains(body, "<svg") {
		t.Errorf("expected valid SVG, got %s", body)
	}
}
