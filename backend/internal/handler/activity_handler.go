package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
)

// ActivityHandler handles activity feed endpoints.
type ActivityHandler struct {
	svc service.ActivityService
}

// NewActivityHandler creates an ActivityHandler.
func NewActivityHandler(svc service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// GlobalFeed handles GET /api/activity?limit=N (no auth required).
func (h *ActivityHandler) GlobalFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	items, err := h.svc.ListGlobal(r.Context(), limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "activity_failed"})
		return
	}
	if items == nil {
		items = []*model.ActivityItem{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"activities": items})
}

// ProjectFeed handles GET /api/projects/{id}/activity (no auth required).
func (h *ActivityHandler) ProjectFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectID := r.PathValue("id")

	items, err := h.svc.ListByProject(r.Context(), projectID, 20)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "activity_failed"})
		return
	}
	if items == nil {
		items = []*model.ActivityItem{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"activities": items})
}
