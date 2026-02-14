package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ProjectServiceImpl は ProjectService の実装
type ProjectServiceImpl struct {
	projectRepo repository.ProjectRepository
}

// NewProjectService は ProjectServiceImpl を生成する（DI: ProjectRepository を注入）
func NewProjectService(projectRepo repository.ProjectRepository) ProjectService {
	return &ProjectServiceImpl{projectRepo: projectRepo}
}

// List はプロジェクト一覧を取得する
func (s *ProjectServiceImpl) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	return s.projectRepo.List(ctx, limit, offset)
}

// GetByID は ID でプロジェクトを取得する
func (s *ProjectServiceImpl) GetByID(ctx context.Context, id string) (*model.Project, error) {
	return s.projectRepo.GetByID(ctx, id)
}

// ListByOwnerID はオーナーIDでプロジェクト一覧を取得する
func (s *ProjectServiceImpl) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	return s.projectRepo.ListByOwnerID(ctx, ownerID)
}

// Create はプロジェクトを作成する
func (s *ProjectServiceImpl) Create(ctx context.Context, project *model.Project) error {
	if project.Status == "" {
		project.Status = "active"
	}
	return s.projectRepo.Create(ctx, project)
}

// Update はプロジェクトを更新する
func (s *ProjectServiceImpl) Update(ctx context.Context, project *model.Project) error {
	return s.projectRepo.Update(ctx, project)
}

// Delete はプロジェクトを削除する
func (s *ProjectServiceImpl) Delete(ctx context.Context, id string) error {
	return s.projectRepo.Delete(ctx, id)
}
