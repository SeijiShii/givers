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

// DonationService provides business logic for donation management.
type DonationService interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	Patch(ctx context.Context, id, userID string, patch model.DonationPatch) error
	Delete(ctx context.Context, id, userID string) error
	MigrateToken(ctx context.Context, token, userID string) (*MigrateTokenResult, error)
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

func (s *donationService) Patch(ctx context.Context, id, userID string, patch model.DonationPatch) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.DonorType != "user" || d.DonorID != userID {
		return ErrForbidden
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
