package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/givers/backend/internal/service"
)

// HostHandler handles GET /api/host.
type HostHandler struct {
	svc service.PlatformHealthService
}

// NewHostHandler creates a HostHandler.
func NewHostHandler(svc service.PlatformHealthService) *HostHandler {
	return &HostHandler{svc: svc}
}

type hostResponse struct {
	MonthlyCost       int    `json:"monthly_cost"`
	CurrentMonthly    int    `json:"current_monthly"`
	WarningThreshold  int    `json:"warning_threshold"`
	CriticalThreshold int    `json:"critical_threshold"`
	Rate              int    `json:"rate"`
	Signal            string `json:"signal"`
}

// Get handles GET /api/host — returns platform health (no auth required).
func (h *HostHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health, err := h.svc.Get(r.Context())
	if err != nil {
		slog.Error("platform health get failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	_ = json.NewEncoder(w).Encode(hostResponse{
		MonthlyCost:       health.MonthlyCost,
		CurrentMonthly:    health.CurrentMonthly,
		WarningThreshold:  health.WarningThreshold,
		CriticalThreshold: health.CriticalThreshold,
		Rate:              health.Rate(),
		Signal:            health.Signal(),
	})
}
