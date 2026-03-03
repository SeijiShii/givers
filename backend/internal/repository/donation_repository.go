package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// DonationRepository handles persistence for donations.
type DonationRepository interface {
	// Create inserts a new donation record.
	Create(ctx context.Context, d *model.Donation) error
	// ListByUser returns donations where donor_type='user' and donor_id=userID.
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	// GetByID returns a single donation by ID.
	GetByID(ctx context.Context, id string) (*model.Donation, error)
	// Patch applies partial updates to a donation.
	Patch(ctx context.Context, id string, patch model.DonationPatch) error
	// Delete removes a donation (cancels recurring subscription).
	Delete(ctx context.Context, id string) error
	// DeleteByStripeSubscriptionID removes a donation by its stripe_subscription_id.
	DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error
	// GetByStripeSubscriptionID returns a donation by its stripe_subscription_id.
	GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error)
	// MigrateToken migrates donations from donor_type='token' to donor_type='user'.
	// Returns the number of rows updated.
	MigrateToken(ctx context.Context, token string, userID string) (int, error)
	// CurrentMonthSumByProject returns the total donation amount for a project in the current month.
	CurrentMonthSumByProject(ctx context.Context, projectID string) (int, error)
	// MonthlySumByProject returns monthly donation totals for a project (last 12 months).
	MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error)
	// ListByProject returns donations for a specific project.
	ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*model.Donation, error)
	// ListMessagesByProject returns donation messages with donor names for a project.
	// sort must be "asc" or "desc". donor is a partial-match filter on display name.
	ListMessagesByProject(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error)
	// ListByProjectForOwner returns donations for a project with donor names, for owner viewing.
	// Supports pagination, sorting, source filtering, and donor name filtering.
	ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string) (*model.OwnerDonationResult, error)
	// SetRefundPending sets refund_status='pending' on a donation.
	// Returns ErrAlreadyRefunded if refund_status is not NULL, ErrNotFound if donation does not exist.
	SetRefundPending(ctx context.Context, id string) error
	// CompleteRefund sets refund_status='completed' and stripe_refund_id on a donation.
	CompleteRefund(ctx context.Context, id string, stripeRefundID string) error
	// ClearRefundPending resets refund_status to NULL (rollback on Stripe failure).
	ClearRefundPending(ctx context.Context, id string) error
}
