package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

const maxMessageLength = 5000

// ContactHandler handles contact form submission and admin listing.
type ContactHandler struct {
	contactService service.ContactService
}

// NewContactHandler creates a ContactHandler with the given service.
func NewContactHandler(contactService service.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

// submitRequest is the expected JSON body for POST /api/contact.
type submitRequest struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Submit handles POST /api/contact.
// email and message are required; name is optional; message max 5000 chars.
func (h *ContactHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "email_required"})
		return
	}

	if req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_required"})
		return
	}

	if len([]rune(req.Message)) > maxMessageLength {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "message_too_long"})
		return
	}

	msg := &model.ContactMessage{
		Email:   req.Email,
		Name:    req.Name,
		Message: req.Message,
		Status:  "unread",
	}

	if err := h.contactService.Submit(r.Context(), msg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "submit_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// adminListResponse is the JSON response for GET /api/admin/contacts.
type adminListResponse struct {
	Messages []*model.ContactMessage `json:"messages"`
}

// AdminList handles GET /api/admin/contacts (host-only).
// Supports query params: status (all/unread/read), limit, offset.
func (h *ContactHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	// Must be authenticated
	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// Must be host
	if !auth.IsHostFromContext(r.Context()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	opts := model.ContactListOptions{
		Status: r.URL.Query().Get("status"),
		Limit:  20,
		Offset: 0,
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			opts.Limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			opts.Offset = n
		}
	}

	messages, err := h.contactService.List(r.Context(), opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "list_failed"})
		return
	}

	// Return [] not null for empty lists
	if messages == nil {
		messages = []*model.ContactMessage{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminListResponse{Messages: messages})
}

// updateStatusRequest is the expected JSON body for PATCH /api/admin/contacts/{id}/status.
type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /api/admin/contacts/{id}/status (host-only).
// Accepts {"status": "read"} or {"status": "unread"}.
func (h *ContactHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	if !auth.IsHostFromContext(r.Context()) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	id := r.PathValue("id")

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Status != "read" && req.Status != "unread" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_status"})
		return
	}

	if err := h.contactService.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
