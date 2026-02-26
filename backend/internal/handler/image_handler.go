package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/internal/storage"
	"github.com/givers/backend/pkg/auth"
)

const maxImageSize = 2 << 20 // 2 MB

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ImageUpdateRepo はプロジェクトの image_url を更新するためのリポジトリインターフェース
type ImageUpdateRepo interface {
	UpdateImageURL(ctx context.Context, projectID, imageURL string) error
}

// ImageHandler はプロジェクト画像のアップロード・削除を処理する
type ImageHandler struct {
	storage        storage.Storage
	projectService service.ProjectService
	imageRepo      ImageUpdateRepo
}

// NewImageHandler は ImageHandler を生成する
func NewImageHandler(store storage.Storage, ps service.ProjectService, repo ImageUpdateRepo) *ImageHandler {
	return &ImageHandler{storage: store, projectService: ps, imageRepo: repo}
}

// Upload は POST /api/projects/{id}/image を処理する
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	project, err := h.projectService.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if project.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file_too_large"})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "image_required"})
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file_too_large"})
		return
	}

	ct := header.Header.Get("Content-Type")
	ext, ok := allowedContentTypes[ct]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_content_type"})
		return
	}

	// 既存画像を削除
	if project.ImageURL != "" {
		oldKey := strings.TrimPrefix(project.ImageURL, "/uploads/")
		_ = h.storage.Delete(r.Context(), oldKey)
	}

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	key := path.Join("projects", projectID, hex.EncodeToString(b)+ext)
	imageURL, err := h.storage.Save(r.Context(), key, file, ct)
	if err != nil {
		slog.Error("image upload failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upload_failed"})
		return
	}

	if err := h.imageRepo.UpdateImageURL(r.Context(), projectID, imageURL); err != nil {
		slog.Error("image url update failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"image_url": imageURL})
}

// Delete は DELETE /api/projects/{id}/image を処理する
func (h *ImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	project, err := h.projectService.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if project.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if project.ImageURL != "" {
		oldKey := strings.TrimPrefix(project.ImageURL, "/uploads/")
		_ = h.storage.Delete(r.Context(), oldKey)
	}

	if err := h.imageRepo.UpdateImageURL(r.Context(), projectID, ""); err != nil {
		slog.Error("image url clear failed", "error", err, "project_id", projectID)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
