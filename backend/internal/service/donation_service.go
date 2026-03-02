package service

import (
	"context"
	"errors"

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
	UpdateSubscriptionAmount(ctx context.Context, subscriptionID string, newAmount int) error
}

// DonationService provides business logic for donation management.
type DonationService interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	MigrateToken(ctx context.Context, token, userID string) (*MigrateTokenResult, error)
	ListProjectMessages(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error)
	ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string) (*model.OwnerDonationResult, error)
}

type donationService struct {
	repo repository.DonationRepository
}

// NewDonationService creates a DonationService.
func NewDonationService(repo repository.DonationRepository) DonationService {
	return &donationService{repo: repo}
}

func (s *donationService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

func (s *donationService) ListProjectMessages(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
	return s.repo.ListMessagesByProject(ctx, projectID, limit, offset, sort, donor)
}

func (s *donationService) ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string) (*model.OwnerDonationResult, error) {
	return s.repo.ListByProjectForOwner(ctx, projectID, limit, offset, sort, sourceFilter, donorFilter)
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
