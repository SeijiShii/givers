package service

import (
	"context"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ProjectUpdateServiceImpl は ProjectUpdateService の実装
type ProjectUpdateServiceImpl struct {
	repo repository.ProjectUpdateRepository
}

// NewProjectUpdateService は ProjectUpdateServiceImpl を生成する
func NewProjectUpdateService(repo repository.ProjectUpdateRepository) ProjectUpdateService {
	return &ProjectUpdateServiceImpl{repo: repo}
}

// ListByProjectID はプロジェクトに属する更新一覧を返す
func (s *ProjectUpdateServiceImpl) ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
	return s.repo.ListByProjectID(ctx, projectID, includeHidden)
}

// GetByID は ID で更新を取得する
func (s *ProjectUpdateServiceImpl) GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error) {
	return s.repo.GetByID(ctx, id)
}

// Create は新しい更新を作成する。visible をデフォルト true にする。
func (s *ProjectUpdateServiceImpl) Create(ctx context.Context, update *model.ProjectUpdate) error {
	update.Visible = true
	return s.repo.Create(ctx, update)
}

// Update は更新を保存する。updated_at を現在時刻にセットする。
func (s *ProjectUpdateServiceImpl) Update(ctx context.Context, update *model.ProjectUpdate) error {
	update.UpdatedAt = time.Now()
	return s.repo.Update(ctx, update)
}

// Delete はソフトデリートを行う（visible=false をセット）
func (s *ProjectUpdateServiceImpl) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
