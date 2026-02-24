package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// stripMarkdown は Markdown 記法を簡易的に除去してプレーンテキストに変換する。
var reMarkdown = regexp.MustCompile(`(?m)^#{1,6}\s+|[*_~` + "`" + `\[\]()>]+`)

func plainTextFromMarkdown(md string, maxLen int) string {
	s := reMarkdown.ReplaceAllString(md, "")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len([]rune(s)) > maxLen {
		return string([]rune(s)[:maxLen])
	}
	return s
}

// parseDeadline は "YYYY-MM-DD" または RFC3339 の文字列を *time.Time にパースする。
// 空文字の場合は nil を返す。
func parseDeadline(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}

// hasJSONKey は raw に key が含まれるか判定する
func hasJSONKey(raw map[string]json.RawMessage, key string) bool {
	_, ok := raw[key]
	return ok
}

// ConnectAccountFunc は v2 API でアカウント作成+オンボーディング URL を返す関数
type ConnectAccountFunc func(ctx context.Context, projectID string) (onboardingURL string, err error)

// ProjectHandler はプロジェクト CRUD の HTTP ハンドラ
type ProjectHandler struct {
	connectAccountFunc ConnectAccountFunc     // nil = Stripe not configured
	projectService     service.ProjectService
	activityService    service.ActivityService // optional, nil = skip
}

// NewProjectHandler は ProjectHandler を生成する
func NewProjectHandler(projectService service.ProjectService, connectAccountFunc ConnectAccountFunc) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, connectAccountFunc: connectAccountFunc}
}

// NewProjectHandlerWithActivity は ActivityService 付きの ProjectHandler を生成する
func NewProjectHandlerWithActivity(projectService service.ProjectService, connectAccountFunc ConnectAccountFunc, actSvc service.ActivityService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, connectAccountFunc: connectAccountFunc, activityService: actSvc}
}

// List は GET /api/projects を処理する
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")     // "new" (default) or "hot"
	cursor := r.URL.Query().Get("cursor") // cursor-based pagination
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	result, err := h.projectService.List(r.Context(), sort, limit, cursor)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
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
		Name             string                       `json:"name"`
		Description      string                       `json:"description"`
		Overview         string                       `json:"overview"`
		ShareMessage     string                       `json:"share_message"`
		Deadline         *string                      `json:"deadline"`
		Status           string                       `json:"status"`
		OwnerWantMonthly *int                         `json:"owner_want_monthly"`
		CostItems        []model.ProjectCostItemInput `json:"cost_items"`
		Alerts           *model.ProjectAlerts         `json:"alerts"`
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
		Overview:         req.Overview,
		ShareMessage:     req.ShareMessage,
		Status:           req.Status,
		OwnerWantMonthly: req.OwnerWantMonthly,
		Alerts:           req.Alerts,
	}
	for i, input := range req.CostItems {
		project.CostItems = append(project.CostItems, &model.ProjectCostItem{
			Label:         input.Label,
			UnitType:      input.UnitType,
			AmountMonthly: input.AmountMonthly,
			RatePerDay:    input.RatePerDay,
			DaysPerMonth:  input.DaysPerMonth,
			SortOrder:     i,
		})
	}
	if req.Deadline != nil {
		project.Deadline = parseDeadline(*req.Deadline)
	}

	// overview → description 自動生成: overview があり description が空の場合
	if project.Description == "" && project.Overview != "" {
		project.Description = plainTextFromMarkdown(project.Overview, 200)
	}

	// ホストの場合: Stripe Connect 不要 → 直接 active にする
	// 一般オーナーの場合: draft で作成し v2 API でアカウント作成 → オンボーディング URL を返す
	isHost := auth.IsHostFromContext(r.Context())
	if project.Status == "" {
		if isHost {
			project.Status = "active"
		} else if h.connectAccountFunc != nil {
			project.Status = "draft"
		}
	}

	if err := h.projectService.Create(r.Context(), project); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}

	// Record activity (fire-and-forget)
	if h.activityService != nil {
		_ = h.activityService.Record(r.Context(), &model.ActivityItem{
			Type:      "project_created",
			ProjectID: project.ID,
			ActorName: &userID,
		})
	}

	if h.connectAccountFunc != nil && !isHost {
		onboardingURL, err := h.connectAccountFunc(r.Context(), project.ID)
		if err != nil {
			log.Printf("stripe: failed to create account for project %s: %v", project.ID, err)
		} else {
			project.StripeConnectURL = onboardingURL
		}
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

	if b, ok := raw["name"]; ok {
		var v string
		_ = json.Unmarshal(b, &v)
		if v != "" {
			existing.Name = v
		}
	}
	if b, ok := raw["description"]; ok {
		var v string
		_ = json.Unmarshal(b, &v)
		existing.Description = v
	}
	if b, ok := raw["overview"]; ok {
		var v string
		_ = json.Unmarshal(b, &v)
		existing.Overview = v
	}
	if b, ok := raw["share_message"]; ok {
		var v string
		_ = json.Unmarshal(b, &v)
		existing.ShareMessage = v
	}
	if b, ok := raw["status"]; ok {
		var v string
		_ = json.Unmarshal(b, &v)
		if v != "" {
			existing.Status = v
		}
	}
	if hasJSONKey(raw, "owner_want_monthly") {
		var v *int
		_ = json.Unmarshal(raw["owner_want_monthly"], &v)
		existing.OwnerWantMonthly = v
	}
	if b, ok := raw["deadline"]; ok {
		var v *string
		_ = json.Unmarshal(b, &v)
		if v != nil {
			existing.Deadline = parseDeadline(*v)
		} else {
			existing.Deadline = nil
		}
	}
	if b, ok := raw["cost_items"]; ok {
		var inputs []model.ProjectCostItemInput
		_ = json.Unmarshal(b, &inputs)
		existing.CostItems = nil
		for i, input := range inputs {
			existing.CostItems = append(existing.CostItems, &model.ProjectCostItem{
				Label:         input.Label,
				UnitType:      input.UnitType,
				AmountMonthly: input.AmountMonthly,
				RatePerDay:    input.RatePerDay,
				DaysPerMonth:  input.DaysPerMonth,
				SortOrder:     i,
			})
		}
	}
	if b, ok := raw["alerts"]; ok {
		var v *model.ProjectAlerts
		_ = json.Unmarshal(b, &v)
		if v != nil {
			existing.Alerts = v
		}
	}

	if err := h.projectService.Update(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "update_failed"})
		return
	}

	// Record activity (fire-and-forget)
	if h.activityService != nil {
		_ = h.activityService.Record(r.Context(), &model.ActivityItem{
			Type:      "project_updated",
			ProjectID: existing.ID,
			ActorName: &userID,
		})
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
