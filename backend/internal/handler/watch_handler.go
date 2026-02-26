package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// WatchHandler はウォッチ機能の HTTP ハンドラ
type WatchHandler struct {
	watchService service.WatchService
}

// NewWatchHandler は WatchHandler を生成する
func NewWatchHandler(watchService service.WatchService) *WatchHandler {
	return &WatchHandler{watchService: watchService}
}

// Watch は POST /api/projects/{id}/watch を処理する（認証必須・冪等）
func (h *WatchHandler) Watch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id_required"})
		return
	}

	if err := h.watchService.Watch(r.Context(), userID, projectID); err != nil {
		slog.Error("watch failed", "error", err, "project_id", projectID, "user_id", userID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Unwatch は DELETE /api/projects/{id}/watch を処理する（認証必須・冪等）
func (h *WatchHandler) Unwatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id_required"})
		return
	}

	if err := h.watchService.Unwatch(r.Context(), userID, projectID); err != nil {
		slog.Error("unwatch failed", "error", err, "project_id", projectID, "user_id", userID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ListWatches は GET /api/me/watches を処理する（認証必須）
func (h *WatchHandler) ListWatches(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projects, err := h.watchService.ListWatchedProjects(r.Context(), userID)
	if err != nil {
		slog.Error("list watches failed", "error", err, "user_id", userID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	// nil スライスを空配列として返す
	if projects == nil {
		projects = []*model.Project{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]*model.Project{"projects": projects})
}
