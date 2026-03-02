package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// SubscriptionRepository handles persistence for subscriptions.
type SubscriptionRepository interface {
	Create(ctx context.Context, s *model.Subscription) error
	GetByID(ctx context.Context, id string) (*model.Subscription, error)
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*model.Subscription, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error)
	Patch(ctx context.Context, id string, patch model.SubscriptionPatch) error
	Delete(ctx context.Context, id string) error
	DeleteByStripeSubscriptionID(ctx context.Context, stripeSubID string) error
	MigrateToken(ctx context.Context, token string, userID string) (int, error)
}
