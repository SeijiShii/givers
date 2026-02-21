package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectUpdateService はプロジェクト更新に関するビジネスロジックのインターフェース
type ProjectUpdateService interface {
	ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error)
	GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error)
	Create(ctx context.Context, update *model.ProjectUpdate) error
	Update(ctx context.Context, update *model.ProjectUpdate) error
	Delete(ctx context.Context, id string) error
}
