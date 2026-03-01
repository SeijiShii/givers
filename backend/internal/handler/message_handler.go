package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// MessageHandler handles donation-message endpoints for project owners.
type MessageHandler struct {
	donationSvc service.DonationService
	projectSvc  service.ProjectService
}

// NewMessageHandler creates a MessageHandler.
func NewMessageHandler(donationSvc service.DonationService, projectSvc service.ProjectService) *MessageHandler {
	return &MessageHandler{donationSvc: donationSvc, projectSvc: projectSvc}
}

// List handles GET /api/projects/:id/messages (owner or host auth required).
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
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
	donor := ""

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
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
	if v := r.URL.Query().Get("donor"); v != "" {
		donor = v
	}

	result, err := h.donationSvc.ListProjectMessages(r.Context(), projectID, limit, offset, sort, donor)
	if err != nil {
		slog.Error("list project messages failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "list_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"messages": result.Messages,
		"total":    result.Total,
	})
}
