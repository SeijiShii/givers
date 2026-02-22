package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectService はプロジェクトに関するビジネスロジックのインターフェース
type ProjectService interface {
	List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error)
	GetByID(ctx context.Context, id string) (*model.Project, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error)
	Create(ctx context.Context, project *model.Project) error
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, id string) error
}
