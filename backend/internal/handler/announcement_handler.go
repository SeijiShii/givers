package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// AnnouncementHandler はお知らせの HTTP ハンドラ
type AnnouncementHandler struct {
	svc service.AnnouncementService
}

// NewAnnouncementHandler は AnnouncementHandler を生成する
func NewAnnouncementHandler(svc service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

// List は GET /api/announcements を処理する（認証任意）
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")

	userID, _ := auth.UserIDFromContext(r.Context())

	announcements, nextCursor, err := h.svc.ListVisible(r.Context(), userID, limit, cursor)
	if err != nil {
		slog.Error("announcements list failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if announcements == nil {
		announcements = []*model.Announcement{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"announcements": announcements,
		"next_cursor":   nilIfEmpty(nextCursor),
	})
}

// UnreadCount は GET /api/announcements/unread-count を処理する（認証必須）
func (h *AnnouncementHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	count, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		slog.Error("unread count failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// MarkRead は POST /api/announcements/{id}/read を処理する（認証必須）
func (h *AnnouncementHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	announcementID := r.PathValue("id")

	if err := h.svc.MarkRead(r.Context(), userID, announcementID); err != nil {
		slog.Error("mark read failed", "error", err, "announcement_id", announcementID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// AdminList は GET /api/admin/announcements を処理する（ホストのみ）
func (h *AnnouncementHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")

	announcements, nextCursor, err := h.svc.ListAll(r.Context(), limit, cursor)
	if err != nil {
		slog.Error("admin announcements list failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if announcements == nil {
		announcements = []*model.Announcement{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"announcements": announcements,
		"next_cursor":   nilIfEmpty(nextCursor),
	})
}

var validSeverities = map[string]bool{"info": true, "warn": true, "error": true}

// AdminCreate は POST /api/admin/announcements を処理する（ホストのみ）
func (h *AnnouncementHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}
	authorID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Body        string  `json:"body"`
		Severity    string  `json:"severity"`
		PublishedAt *string `json:"published_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "title", "message": "required"}},
		})
		return
	}
	if len(req.Title) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "title", "message": "too_long"}},
		})
		return
	}
	if req.Severity != "" && !validSeverities[req.Severity] {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "severity", "message": "invalid"}},
		})
		return
	}

	a := &model.Announcement{
		AuthorID: authorID,
		Title:    req.Title,
		Body:     req.Body,
		Severity: req.Severity,
	}

	if req.PublishedAt != nil && *req.PublishedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.PublishedAt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{"field": "published_at", "message": "invalid_format"}},
			})
			return
		}
		a.PublishedAt = t
	}

	if err := h.svc.Create(r.Context(), a); err != nil {
		slog.Error("announcement create failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]*model.Announcement{"announcement": a})
}

// AdminUpdate は PUT /api/admin/announcements/{id} を処理する（ホストのみ）
func (h *AnnouncementHandler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}

	id := r.PathValue("id")

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil || existing == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Body        *string `json:"body"`
		Severity    *string `json:"severity"`
		PublishedAt *string `json:"published_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Body != nil {
		existing.Body = *req.Body
	}
	if req.Severity != nil {
		if !validSeverities[*req.Severity] {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{"field": "severity", "message": "invalid"}},
			})
			return
		}
		existing.Severity = *req.Severity
	}
	if req.PublishedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.PublishedAt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{"field": "published_at", "message": "invalid_format"}},
			})
			return
		}
		existing.PublishedAt = t
	}

	if err := h.svc.Update(r.Context(), existing); err != nil {
		slog.Error("announcement update failed", "error", err, "id", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]*model.Announcement{"announcement": existing})
}

// AdminSetVisibility は PATCH /api/admin/announcements/{id}/visibility を処理する
func (h *AnnouncementHandler) AdminSetVisibility(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}

	id := r.PathValue("id")

	var req struct {
		Visible bool `json:"visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if err := h.svc.SetVisibility(r.Context(), id, req.Visible); err != nil {
		slog.Error("set visibility failed", "error", err, "id", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// requireHost は認証とホスト権限をチェックする。失敗時はレスポンスを書き込んで false を返す。
func (h *AnnouncementHandler) requireHost(w http.ResponseWriter, r *http.Request) bool {
	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return false
	}
	if !auth.IsHostFromContext(r.Context()) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return false
	}
	return true
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
