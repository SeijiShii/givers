package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// DonationRepository handles persistence for donations.
type DonationRepository interface {
	// ListByUser returns donations where donor_type='user' and donor_id=userID.
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	// GetByID returns a single donation by ID.
	GetByID(ctx context.Context, id string) (*model.Donation, error)
	// Patch applies partial updates to a donation.
	Patch(ctx context.Context, id string, patch model.DonationPatch) error
	// Delete removes a donation (cancels recurring subscription).
	Delete(ctx context.Context, id string) error
	// MigrateToken migrates donations from donor_type='token' to donor_type='user'.
	// Returns the number of rows updated.
	MigrateToken(ctx context.Context, token string, userID string) (int, error)
}
