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
	// MigrateToken migrates donations from donor_type='token' to donor_type='user'.
	// Returns the number of rows updated.
	MigrateToken(ctx context.Context, token string, userID string) (int, error)
	// CurrentMonthSumByProject returns the total donation amount for a project in the current month.
	CurrentMonthSumByProject(ctx context.Context, projectID string) (int, error)
	// MonthlySumByProject returns monthly donation totals for a project (last 12 months).
	MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error)
	// ListByProject returns donations for a specific project.
	ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*model.Donation, error)
}
