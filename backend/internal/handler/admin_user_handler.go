package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// disclosureDonationLister is a minimal interface for listing donations by project.
type disclosureDonationLister interface {
	ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*model.Donation, error)
}

// AdminUserHandler handles admin user management endpoints.
type AdminUserHandler struct {
	adminSvc       service.AdminUserService
	projectSvc     service.ProjectService
	donationLister disclosureDonationLister // optional: nil disables type=donation
}

// NewAdminUserHandler creates an AdminUserHandler.
func NewAdminUserHandler(adminSvc service.AdminUserService, projectSvc service.ProjectService, donationLister disclosureDonationLister) *AdminUserHandler {
	return &AdminUserHandler{adminSvc: adminSvc, projectSvc: projectSvc, donationLister: donationLister}
}

func requireHost(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := auth.UserIDFromContext(r.Context()); !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return false
	}
	if !auth.IsHostFromContext(r.Context()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return false
	}
	return true
}

// List handles GET /api/admin/users (host-only).
func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireHost(w, r) {
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	users, err := h.adminSvc.ListUsers(r.Context(), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "list_failed"})
		return
	}
	if users == nil {
		users = []*model.User{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"users": users})
}

type suspendRequest struct {
	Suspended bool `json:"suspended"`
}

// Suspend handles PATCH /api/admin/users/:id/suspend (host-only).
func (h *AdminUserHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireHost(w, r) {
		return
	}

	id := r.PathValue("id")

	var req suspendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	// ホスト自身の利用停止を禁止（解除は許可）
	if req.Suspended {
		if callerID, ok := auth.UserIDFromContext(r.Context()); ok && callerID == id {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "cannot_suspend_self"})
			return
		}
	}

	if err := h.adminSvc.SuspendUser(r.Context(), id, req.Suspended); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "suspend_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// DisclosureExport handles GET /api/admin/disclosure-export (host-only).
// Query params: type=user|project, id=<uuid>
func (h *AdminUserHandler) DisclosureExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireHost(w, r) {
		return
	}

	exportType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")

	if exportType == "" || id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "type_and_id_required"})
		return
	}

	switch exportType {
	case "user":
		user, err := h.adminSvc.GetUser(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "export_failed"})
			return
		}
		_ = json.NewEncoder(w).Encode(user)

	case "project":
		project, err := h.projectSvc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "export_failed"})
			return
		}
		_ = json.NewEncoder(w).Encode(project)

	case "donation":
		if h.donationLister == nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "donation_export_not_configured"})
			return
		}
		// id = project_id — まずプロジェクト名を取得
		project, err := h.projectSvc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "export_failed"})
			return
		}
		donations, err := h.donationLister.ListByProject(r.Context(), id, 10000, 0)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "export_failed"})
			return
		}
		if donations == nil {
			donations = []*model.Donation{}
		}
		var total int
		for _, d := range donations {
			total += d.Amount
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"project_id":   id,
			"project_name": project.Name,
			"donations":    donations,
			"total":        total,
		})

	default:
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_type"})
	}
}
