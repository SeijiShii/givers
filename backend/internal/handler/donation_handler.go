package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// DonationHandler handles donation-related endpoints.
type DonationHandler struct {
	svc service.DonationService
}

// NewDonationHandler creates a DonationHandler.
func NewDonationHandler(svc service.DonationService) *DonationHandler {
	return &DonationHandler{svc: svc}
}

// List handles GET /api/me/donations (auth required).
func (h *DonationHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	donations, err := h.svc.ListByUser(r.Context(), userID, 50, 0)
	if err != nil {
		slog.Error("donation list failed", "error", err, "user_id", userID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "list_failed"})
		return
	}
	if donations == nil {
		donations = []*model.Donation{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"donations": donations})
}

type donationPatchRequest struct {
	Amount *int  `json:"amount"`
	Paused *bool `json:"paused"`
}

// Patch handles PATCH /api/me/donations/:id (auth required).
func (h *DonationHandler) Patch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	id := r.PathValue("id")

	var req donationPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	patch := model.DonationPatch{Amount: req.Amount, Paused: req.Paused}
	if err := h.svc.Patch(r.Context(), id, userID, patch); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		slog.Error("donation patch failed", "error", err, "donation_id", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "patch_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// Delete handles DELETE /api/me/donations/:id (auth required).
func (h *DonationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	id := r.PathValue("id")

	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		slog.Error("donation delete failed", "error", err, "donation_id", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "delete_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// MigrateFromToken handles POST /api/me/migrate-from-token (auth required).
// Reads donor_token from Cookie and migrates anonymous donations to the current user.
func (h *DonationHandler) MigrateFromToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	cookie, err := r.Cookie("donor_token")
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "donor_token_missing"})
		return
	}

	result, err := h.svc.MigrateToken(r.Context(), cookie.Value, userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"migrated_count":   result.MigratedCount,
		"already_migrated": result.AlreadyMigrated,
	})
}
