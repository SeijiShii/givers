package handler

import (
	"encoding/json"
	"net/http"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ProjectUpdateHandler はプロジェクト更新の HTTP ハンドラ
type ProjectUpdateHandler struct {
	svc        service.ProjectUpdateService
	projectSvc service.ProjectService
}

// NewProjectUpdateHandler は ProjectUpdateHandler を生成する
func NewProjectUpdateHandler(svc service.ProjectUpdateService, projectSvc service.ProjectService) *ProjectUpdateHandler {
	return &ProjectUpdateHandler{svc: svc, projectSvc: projectSvc}
}

// List は GET /api/projects/{id}/updates を処理する（認証不要・公開）
// オーナーがアクセスした場合は非表示更新も含む。
func (h *ProjectUpdateHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := h.projectSvc.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	// オーナーは非表示更新も閲覧できる
	includeHidden := false
	if userID, ok := auth.UserIDFromContext(r.Context()); ok && userID == project.OwnerID {
		includeHidden = true
	}

	updates, err := h.svc.ListByProjectID(r.Context(), projectID, includeHidden)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	// nil スライスを空配列として返す
	if updates == nil {
		updates = []*model.ProjectUpdate{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]*model.ProjectUpdate{"updates": updates})
}

// Create は POST /api/projects/{id}/updates を処理する（認証必須・プロジェクトオーナーのみ）
func (h *ProjectUpdateHandler) Create(w http.ResponseWriter, r *http.Request) {
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if project.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		Title *string `json:"title"`
		Body  string  `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}
	if req.Body == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "body_required"})
		return
	}

	update := &model.ProjectUpdate{
		ProjectID: projectID,
		AuthorID:  userID,
		Title:     req.Title,
		Body:      req.Body,
	}
	if err := h.svc.Create(r.Context(), update); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(update)
}

// UpdateUpdate は PUT /api/projects/{id}/updates/{uid} を処理する（認証必須・更新作成者のみ）
func (h *ProjectUpdateHandler) UpdateUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	uid := r.PathValue("uid")

	existing, err := h.svc.GetByID(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	// 更新が指定プロジェクトに属するか確認
	if existing.ProjectID != projectID {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	// 更新の作成者のみ編集可能
	if existing.AuthorID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		Title   *string `json:"title"`
		Body    *string `json:"body"`
		Visible *bool   `json:"visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Title != nil {
		existing.Title = req.Title
	}
	if req.Body != nil {
		existing.Body = *req.Body
	}
	if req.Visible != nil {
		existing.Visible = *req.Visible
	}

	if err := h.svc.Update(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

// Delete は DELETE /api/projects/{id}/updates/{uid} を処理する
// （認証必須・プロジェクトオーナーまたはホストのみ。ソフトデリート）
func (h *ProjectUpdateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projectID := r.PathValue("id")
	uid := r.PathValue("uid")

	project, err := h.projectSvc.GetByID(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	existing, err := h.svc.GetByID(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	// 更新が指定プロジェクトに属するか確認
	if existing.ProjectID != projectID {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	// プロジェクトオーナーまたはホストのみ削除可能
	isOwner := project.OwnerID == userID
	isHost := auth.IsHostFromContext(r.Context())
	if !isOwner && !isHost {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := h.svc.Delete(r.Context(), uid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "delete_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
