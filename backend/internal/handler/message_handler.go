package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// MessageHandler はメッセージングの HTTP ハンドラ
type MessageHandler struct {
	svc service.MessageService
}

// NewMessageHandler は MessageHandler を生成する
func NewMessageHandler(svc service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// AdminCreateThread は POST /api/admin/messages/threads を処理する（ホストのみ）
func (h *MessageHandler) AdminCreateThread(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}
	hostUserID, _ := auth.UserIDFromContext(r.Context())

	var req struct {
		ParticipantUserID string `json:"participant_user_id"`
		Subject           string `json:"subject"`
		Body              string `json:"body"`
		Broadcast         bool   `json:"broadcast"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if !req.Broadcast && req.ParticipantUserID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "participant_user_id", "message": "required"}},
		})
		return
	}
	if req.Subject == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "subject", "message": "required"}},
		})
		return
	}
	if len(req.Subject) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "subject", "message": "too_long"}},
		})
		return
	}
	if req.Body == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "body", "message": "required"}},
		})
		return
	}

	if req.Broadcast {
		count, err := h.svc.BroadcastThread(r.Context(), hostUserID, req.Subject, req.Body)
		if err != nil {
			slog.Error("broadcast thread failed", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "broadcast_failed"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
		return
	}

	thread, err := h.svc.CreateThread(r.Context(), hostUserID, req.ParticipantUserID, req.Subject, req.Body)
	if err != nil {
		slog.Error("create thread failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]*model.MessageThread{"thread": thread})
}

// AdminListThreads は GET /api/admin/messages/threads を処理する（ホストのみ）
func (h *MessageHandler) AdminListThreads(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}
	hostUserID, _ := auth.UserIDFromContext(r.Context())

	limit, cursor := h.parsePagination(r)

	threads, nextCursor, err := h.svc.ListThreadsForHost(r.Context(), hostUserID, limit, cursor)
	if err != nil {
		slog.Error("admin list threads failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if threads == nil {
		threads = []*model.MessageThread{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"threads":     threads,
		"next_cursor": nilIfEmpty(nextCursor),
	})
}

// AdminSetThreadActive は PATCH /api/admin/messages/threads/{id}/active を処理する（ホストのみ）
func (h *MessageHandler) AdminSetThreadActive(w http.ResponseWriter, r *http.Request) {
	if !h.requireHost(w, r) {
		return
	}
	hostUserID, _ := auth.UserIDFromContext(r.Context())

	threadID := r.PathValue("id")

	var req struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if err := h.svc.SetThreadActive(r.Context(), threadID, hostUserID, req.Active); err != nil {
		if err.Error() == "not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		if err.Error() == "forbidden: only the host can change thread active state" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		slog.Error("set thread active failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ListThreads は GET /api/me/messages/threads を処理する（認証必須）
func (h *MessageHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	limit, cursor := h.parsePagination(r)

	threads, nextCursor, err := h.svc.ListThreads(r.Context(), userID, limit, cursor)
	if err != nil {
		slog.Error("list threads failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if threads == nil {
		threads = []*model.MessageThread{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"threads":     threads,
		"next_cursor": nilIfEmpty(nextCursor),
	})
}

// GetThread は GET /api/me/messages/threads/{id} を処理する（認証必須）
func (h *MessageHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	threadID := r.PathValue("id")

	thread, err := h.svc.GetThread(r.Context(), threadID, userID)
	if err != nil {
		if err.Error() == "not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		if err.Error() == "forbidden: not a participant" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		slog.Error("get thread failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]*model.MessageThread{"thread": thread})
}

// ListMessages は GET /api/me/messages/threads/{id}/messages を処理する（認証必須）
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	threadID := r.PathValue("id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")

	messages, nextCursor, err := h.svc.ListMessages(r.Context(), threadID, userID, limit, cursor)
	if err != nil {
		if err.Error() == "not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		if err.Error() == "forbidden: not a participant" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		slog.Error("list messages failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if messages == nil {
		messages = []*model.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"messages":    messages,
		"next_cursor": nilIfEmpty(nextCursor),
	})
}

// SendMessage は POST /api/me/messages/threads/{id}/messages を処理する（認証必須）
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	threadID := r.PathValue("id")

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}
	if req.Body == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"field": "body", "message": "required"}},
		})
		return
	}

	msg, err := h.svc.SendMessage(r.Context(), threadID, userID, req.Body)
	if err != nil {
		if err.Error() == "not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		if err.Error() == "thread is inactive" {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "thread_inactive"})
			return
		}
		if err.Error() == "forbidden: not a participant" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		slog.Error("send message failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "send_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]*model.Message{"message": msg})
}

// UnreadCount は GET /api/me/messages/unread-count を処理する（認証必須）
func (h *MessageHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	count, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		slog.Error("message unread count failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func (h *MessageHandler) requireHost(w http.ResponseWriter, r *http.Request) bool {
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

func (h *MessageHandler) parsePagination(r *http.Request) (int, string) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")
	return limit, cursor
}
