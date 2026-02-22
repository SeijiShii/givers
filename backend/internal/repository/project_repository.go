package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectRepository はプロジェクト永続化のインターフェース
type ProjectRepository interface {
	List(ctx context.Context, limit, offset int) ([]*model.Project, error)
	GetByID(ctx context.Context, id string) (*model.Project, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error)
	Create(ctx context.Context, project *model.Project) error
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, id string) error
	// UpdateStripeConnect は Stripe Connect 完了後に stripe_account_id と status='active' を保存する
	UpdateStripeConnect(ctx context.Context, projectID, stripeAccountID string) error
}
