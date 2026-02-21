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

// hasJSONKey は raw に key が含まれるか判定する
func hasJSONKey(raw map[string]json.RawMessage, key string) bool {
	_, ok := raw[key]
	return ok
}

// ProjectHandler はプロジェクト CRUD の HTTP ハンドラ
type ProjectHandler struct {
	projectService service.ProjectService
}

// NewProjectHandler は ProjectHandler を生成する
func NewProjectHandler(projectService service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// List は GET /api/projects を処理する
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	projects, err := h.projectService.List(r.Context(), limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projects)
}

// Get は GET /api/projects/{id} を処理する
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id_required"})
		return
	}

	project, err := h.projectService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(project)
}

// MyProjects は GET /api/me/projects を処理する（認証必須）
func (h *ProjectHandler) MyProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	projects, err := h.projectService.ListByOwnerID(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projects)
}

// Create は POST /api/projects を処理する（認証必須）
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Name             string                 `json:"name"`
		Description      string                 `json:"description"`
		Deadline         *string                `json:"deadline"`
		Status           string                 `json:"status"`
		OwnerWantMonthly *int                   `json:"owner_want_monthly"`
		Costs            *model.ProjectCosts   `json:"costs"`
		Alerts           *model.ProjectAlerts  `json:"alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name_required"})
		return
	}

	project := &model.Project{
		OwnerID:          userID,
		Name:             req.Name,
		Description:      req.Description,
		Status:           req.Status,
		OwnerWantMonthly: req.OwnerWantMonthly,
		Costs:            req.Costs,
		Alerts:           req.Alerts,
	}
	if req.Deadline != nil && *req.Deadline != "" {
		// TODO: parse deadline from string (RFC3339 or YYYY-MM-DD)
		// project.Deadline = parsed
	}

	if err := h.projectService.Create(r.Context(), project); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(project)
}

// Update は PUT /api/projects/{id} を処理する（認証必須）
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id_required"})
		return
	}

	existing, err := h.projectService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if existing.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	var req struct {
		Name             string                 `json:"name"`
		Description      string                 `json:"description"`
		Deadline         *string                `json:"deadline"`
		Status           string                 `json:"status"`
		OwnerWantMonthly *int                   `json:"owner_want_monthly"`
		Costs            *model.ProjectCosts   `json:"costs"`
		Alerts           *model.ProjectAlerts  `json:"alerts"`
	}
	if b, ok := raw["name"]; ok {
		_ = json.Unmarshal(b, &req.Name)
	}
	if b, ok := raw["description"]; ok {
		_ = json.Unmarshal(b, &req.Description)
	}
	if b, ok := raw["deadline"]; ok {
		_ = json.Unmarshal(b, &req.Deadline)
	}
	if b, ok := raw["status"]; ok {
		_ = json.Unmarshal(b, &req.Status)
	}
	if b, ok := raw["owner_want_monthly"]; ok {
		_ = json.Unmarshal(b, &req.OwnerWantMonthly)
	}
	if b, ok := raw["costs"]; ok {
		_ = json.Unmarshal(b, &req.Costs)
	}
	if b, ok := raw["alerts"]; ok {
		_ = json.Unmarshal(b, &req.Alerts)
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if hasJSONKey(raw, "owner_want_monthly") {
		existing.OwnerWantMonthly = req.OwnerWantMonthly
	}
	if req.Costs != nil {
		existing.Costs = req.Costs
	}
	if req.Alerts != nil {
		existing.Alerts = req.Alerts
	}

	if err := h.projectService.Update(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

// PatchStatus は PATCH /api/projects/{id}/status を処理する（認証必須・オーナーまたはホスト）。
// 許可されるステータス: "active", "frozen"
func (h *ProjectHandler) PatchStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	id := r.PathValue("id")

	existing, err := h.projectService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	isHost := auth.IsHostFromContext(r.Context())
	if !isHost && existing.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.Status != "active" && req.Status != "frozen" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_status"})
		return
	}

	existing.Status = req.Status
	if err := h.projectService.Update(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(existing)
}

// Delete は DELETE /api/projects/{id} を処理する（認証必須・オーナーのみ）。
// 論理削除（status を "deleted" に更新）。
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	id := r.PathValue("id")

	existing, err := h.projectService.GetByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if existing.OwnerID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := h.projectService.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "delete_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
