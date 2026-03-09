package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// DonationAlertRepository handles persistence for donation alerts.
type DonationAlertRepository interface {
	// Create inserts a new donation alert record.
	Create(ctx context.Context, a *model.DonationAlert) error
	// List returns alerts filtered by status (empty = all) with pagination.
	List(ctx context.Context, status string, limit, offset int) ([]*model.DonationAlert, int, error)
	// UpdateStatus changes the status of an alert (acknowledged, resolved).
	UpdateStatus(ctx context.Context, id, status, resolvedBy string) error
	// CountRecentByDonor returns how many donations this donor made to this project within windowMinutes.
	CountRecentByDonor(ctx context.Context, projectID, donorType, donorID string, windowMinutes int) (int, error)
	// DailyTotalByDonor returns the total amount donated by this donor to this project today.
	DailyTotalByDonor(ctx context.Context, projectID, donorType, donorID string) (int, error)
	// RefundCountByDonor returns completed refunds for this donor on this project within windowDays.
	RefundCountByDonor(ctx context.Context, projectID, donorType, donorID string, windowDays int) (int, error)
	// HasRecentAlert checks if an alert of the same type exists within windowHours (deduplication).
	HasRecentAlert(ctx context.Context, projectID, donorType, donorID, alertType string, windowHours int) (bool, error)
}
