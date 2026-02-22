package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock ChartDonationService
// ---------------------------------------------------------------------------

type mockChartDonationService struct {
	monthlySumFunc func(ctx context.Context, projectID string) ([]*model.MonthlySum, error)
}

func (m *mockChartDonationService) MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
	if m.monthlySumFunc != nil {
		return m.monthlySumFunc(ctx, projectID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// GET /api/projects/{id}/chart tests
// ---------------------------------------------------------------------------

func TestChartHandler_Chart_Success(t *testing.T) {
	projectMock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{
				ID:            id,
				MonthlyTarget: 50000,
				CostItems: []*model.ProjectCostItem{
					{UnitType: "monthly", AmountMonthly: 10000},
					{UnitType: "daily_x_days", RatePerDay: 5000, DaysPerMonth: 4},
				},
			}, nil
		},
	}
	donationMock := &mockChartDonationService{
		monthlySumFunc: func(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
			return []*model.MonthlySum{
				{Month: "2026-01", Amount: 30000},
				{Month: "2026-02", Amount: 45000},
			}, nil
		},
	}
	h := NewChartHandler(projectMock, donationMock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/chart", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Chart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Chart []*model.ChartDataPoint `json:"chart"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Chart) != 2 {
		t.Fatalf("expected 2 chart points, got %d", len(resp.Chart))
	}
	// minAmount = 10000 + 5000*4 = 30000
	if resp.Chart[0].MinAmount != 30000 {
		t.Errorf("expected minAmount=30000, got %d", resp.Chart[0].MinAmount)
	}
	if resp.Chart[0].TargetAmount != 50000 {
		t.Errorf("expected targetAmount=50000, got %d", resp.Chart[0].TargetAmount)
	}
	if resp.Chart[0].ActualAmount != 30000 {
		t.Errorf("expected actualAmount=30000 for 2026-01, got %d", resp.Chart[0].ActualAmount)
	}
	if resp.Chart[1].ActualAmount != 45000 {
		t.Errorf("expected actualAmount=45000 for 2026-02, got %d", resp.Chart[1].ActualAmount)
	}
}

func TestChartHandler_Chart_EmptyDonations(t *testing.T) {
	projectMock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, MonthlyTarget: 10000}, nil
		},
	}
	donationMock := &mockChartDonationService{
		monthlySumFunc: func(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
			return nil, nil
		},
	}
	h := NewChartHandler(projectMock, donationMock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/chart", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Chart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Chart []*model.ChartDataPoint `json:"chart"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Chart == nil {
		t.Error("expected non-nil chart array")
	}
	if len(resp.Chart) != 0 {
		t.Errorf("expected 0 chart points, got %d", len(resp.Chart))
	}
}

func TestChartHandler_Chart_ProjectNotFound(t *testing.T) {
	projectMock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewChartHandler(projectMock, &mockChartDonationService{})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/bad/chart", nil)
	req.SetPathValue("id", "bad")
	rec := httptest.NewRecorder()
	h.Chart(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestChartHandler_Chart_DonationError(t *testing.T) {
	projectMock := &mockProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id}, nil
		},
	}
	donationMock := &mockChartDonationService{
		monthlySumFunc: func(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewChartHandler(projectMock, donationMock)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/chart", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.Chart(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
