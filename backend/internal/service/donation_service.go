package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ErrForbidden is returned when a user tries to modify another user's resource.
var ErrForbidden = errors.New("forbidden")

// MigrateTokenResult holds the result of a token migration.
type MigrateTokenResult struct {
	MigratedCount   int
	AlreadyMigrated bool
}

// SubscriptionManager manages Stripe subscription lifecycle.
type SubscriptionManager interface {
	PauseSubscription(ctx context.Context, subscriptionID string) error
	ResumeSubscription(ctx context.Context, subscriptionID string) error
	CancelSubscription(ctx context.Context, subscriptionID string) error
}

// DonationService provides business logic for donation management.
type DonationService interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	Patch(ctx context.Context, id, userID string, patch model.DonationPatch) error
	Delete(ctx context.Context, id, userID string) error
	MigrateToken(ctx context.Context, token, userID string) (*MigrateTokenResult, error)
}

type donationService struct {
	repo repository.DonationRepository
	sm   SubscriptionManager
}

// NewDonationService creates a DonationService. sm can be nil to skip Stripe calls.
func NewDonationService(repo repository.DonationRepository, sm SubscriptionManager) DonationService {
	return &donationService{repo: repo, sm: sm}
}

func (s *donationService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

func (s *donationService) Patch(ctx context.Context, id, userID string, patch model.DonationPatch) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.DonorType != "user" || d.DonorID != userID {
		return ErrForbidden
	}

	// Stripe subscription pause/resume for recurring donations
	if patch.Paused != nil && d.IsRecurring && d.StripeSubscriptionID != "" && s.sm != nil {
		if *patch.Paused {
			if err := s.sm.PauseSubscription(ctx, d.StripeSubscriptionID); err != nil {
				return fmt.Errorf("stripe pause: %w", err)
			}
		} else {
			if err := s.sm.ResumeSubscription(ctx, d.StripeSubscriptionID); err != nil {
				return fmt.Errorf("stripe resume: %w", err)
			}
		}
	}

	return s.repo.Patch(ctx, id, patch)
}

func (s *donationService) Delete(ctx context.Context, id, userID string) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.DonorType != "user" || d.DonorID != userID {
		return ErrForbidden
	}

	// Cancel Stripe subscription before deleting
	if d.IsRecurring && d.StripeSubscriptionID != "" && s.sm != nil {
		if err := s.sm.CancelSubscription(ctx, d.StripeSubscriptionID); err != nil {
			return fmt.Errorf("stripe cancel: %w", err)
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *donationService) MigrateToken(ctx context.Context, token, userID string) (*MigrateTokenResult, error) {
	if token == "" {
		return nil, errors.New("donor_token is required")
	}
	count, err := s.repo.MigrateToken(ctx, token, userID)
	if err != nil {
		return nil, err
	}
	return &MigrateTokenResult{
		MigratedCount:   count,
		AlreadyMigrated: count == 0,
	}, nil
}
