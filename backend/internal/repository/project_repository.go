package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectRepository はプロジェクト永続化のインターフェース
type ProjectRepository interface {
	List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error)
	GetByID(ctx context.Context, id string) (*model.Project, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error)
	Create(ctx context.Context, project *model.Project) error
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, id string) error
	// SaveStripeAccountID は stripe_account_id のみを保存する（status は変更しない）
	SaveStripeAccountID(ctx context.Context, projectID, stripeAccountID string) error
	// ActivateProject はプロジェクトの status を 'active' に更新する
	ActivateProject(ctx context.Context, projectID string) error
}
