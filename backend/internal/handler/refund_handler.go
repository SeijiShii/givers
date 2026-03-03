package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// RefundHandler handles refund endpoints.
type RefundHandler struct {
	refundSvc  service.RefundService
	projectSvc service.ProjectService
}

// NewRefundHandler creates a RefundHandler.
func NewRefundHandler(refundSvc service.RefundService, projectSvc service.ProjectService) *RefundHandler {
	return &RefundHandler{refundSvc: refundSvc, projectSvc: projectSvc}
}

// RefundByOwner handles POST /api/projects/{id}/donations/{donationId}/refund.
func (h *RefundHandler) RefundByOwner(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	donationID := r.PathValue("donationId")
	isHost := auth.IsHostFromContext(r.Context())

	err := h.refundSvc.RefundByOwner(r.Context(), projectID, donationID, userID, isHost)
	if err != nil {
		h.writeError(w, err, projectID, donationID)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
}

// RefundByDonor handles POST /api/me/donations/{donationId}/refund.
func (h *RefundHandler) RefundByDonor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	donationID := r.PathValue("donationId")

	err := h.refundSvc.RefundByDonor(r.Context(), donationID, userID)
	if err != nil {
		h.writeError(w, err, "", donationID)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
}

func (h *RefundHandler) writeError(w http.ResponseWriter, err error, projectID, donationID string) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
	case errors.Is(err, repository.ErrAlreadyRefunded):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "already_refunded"})
	case errors.Is(err, repository.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	default:
		slog.Error("refund failed", "error", err, "project_id", projectID, "donation_id", donationID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "refund_failed"})
	}
}
