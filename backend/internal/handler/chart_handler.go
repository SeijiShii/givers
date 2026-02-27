package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
)

// ChartDonationService is the subset of DonationRepository needed for chart data.
type ChartDonationService interface {
	MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error)
}

// ChartHandler handles GET /api/projects/{id}/chart.
type ChartHandler struct {
	projectSvc service.ProjectService
	donationSvc ChartDonationService
}

// NewChartHandler creates a ChartHandler.
func NewChartHandler(projectSvc service.ProjectService, donationSvc ChartDonationService) *ChartHandler {
	return &ChartHandler{projectSvc: projectSvc, donationSvc: donationSvc}
}

// Chart handles GET /api/projects/{id}/chart.
func (h *ChartHandler) Chart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectID := r.PathValue("id")

	project, err := h.projectSvc.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "project_not_found"})
		return
	}

	sums, err := h.donationSvc.MonthlySumByProject(r.Context(), projectID)
	if err != nil {
		slog.Error("chart data failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chart_failed"})
		return
	}

	var minAmount int
	if project.OwnerWantMonthly != nil {
		minAmount = *project.OwnerWantMonthly
	}
	targetAmount := project.MonthlyTarget

	// Build lookup from monthly sums
	sumMap := make(map[string]int, len(sums))
	for _, s := range sums {
		sumMap[s.Month] = s.Amount
	}

	// Build chart data points from sums (only months with data)
	points := make([]*model.ChartDataPoint, 0, len(sums))
	for _, s := range sums {
		points = append(points, &model.ChartDataPoint{
			Month:        s.Month,
			MinAmount:    minAmount,
			TargetAmount: targetAmount,
			ActualAmount: s.Amount,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"chart": points})
}
