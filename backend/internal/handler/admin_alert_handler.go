package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

// alertLister is the minimal interface for listing and managing alerts.
type alertLister interface {
	List(ctx context.Context, status string, limit, offset int) ([]*model.DonationAlert, int, error)
	UpdateStatus(ctx context.Context, id, status, resolvedBy string) error
}

// AdminAlertHandler handles admin alert management endpoints.
type AdminAlertHandler struct {
	alertSvc alertLister
}

// NewAdminAlertHandler creates an AdminAlertHandler.
func NewAdminAlertHandler(alertSvc alertLister) *AdminAlertHandler {
	return &AdminAlertHandler{alertSvc: alertSvc}
}

// List handles GET /api/admin/alerts (host-only).
func (h *AdminAlertHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requireHost(w, r) {
		return
	}

	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	alerts, total, err := h.alertSvc.List(r.Context(), status, limit, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to list alerts"})
		return
	}

	if alerts == nil {
		alerts = []*model.DonationAlert{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"alerts": alerts,
		"total":  total,
	})
}

// UpdateStatus handles PATCH /api/admin/alerts/{id}/status (host-only).
func (h *AdminAlertHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if !requireHost(w, r) {
		return
	}

	alertID := r.PathValue("id")
	if alertID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing alert id"})
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "status is required"})
		return
	}

	if body.Status != "acknowledged" && body.Status != "resolved" && body.Status != "new" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid status"})
		return
	}

	resolvedBy := ""
	if userID, ok := auth.UserIDFromContext(r.Context()); ok {
		resolvedBy = userID
	}

	if err := h.alertSvc.UpdateStatus(r.Context(), alertID, body.Status, resolvedBy); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "alert not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
