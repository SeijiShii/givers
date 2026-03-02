package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ownerDonationService is the minimal interface for OwnerDonationHandler.
type ownerDonationService interface {
	ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string) (*model.OwnerDonationResult, error)
}

// OwnerDonationHandler handles donation listing endpoints for project owners.
type OwnerDonationHandler struct {
	donationSvc ownerDonationService
	projectSvc  service.ProjectService
}

// NewOwnerDonationHandler creates an OwnerDonationHandler.
func NewOwnerDonationHandler(donationSvc ownerDonationService, projectSvc service.ProjectService) *OwnerDonationHandler {
	return &OwnerDonationHandler{donationSvc: donationSvc, projectSvc: projectSvc}
}

// List handles GET /api/projects/:id/donations (owner or host auth required).
func (h *OwnerDonationHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")

	project, err := h.projectSvc.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "project_not_found"})
		return
	}

	isHost := auth.IsHostFromContext(r.Context())
	if !isHost && project.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	// Parse query params with defaults
	limit := 50
	offset := 0
	sort := "desc"
	source := ""
	donor := ""

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := r.URL.Query().Get("sort"); v == "asc" || v == "desc" {
		sort = v
	}
	if v := r.URL.Query().Get("source"); v != "" {
		source = v
	}
	if v := r.URL.Query().Get("donor"); v != "" {
		donor = v
	}

	result, err := h.donationSvc.ListByProjectForOwner(r.Context(), projectID, limit, offset, sort, source, donor)
	if err != nil {
		slog.Error("list owner donations failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "list_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"donations": result.Donations,
		"total":     result.Total,
	})
}
