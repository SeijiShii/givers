package service

import (
	"context"
	"fmt"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// SubscriptionService provides business logic for subscription management.
type SubscriptionService interface {
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error)
	GetByID(ctx context.Context, id, userID string) (*model.Subscription, error)
	Patch(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error
	Delete(ctx context.Context, id, userID string) error
}

type subscriptionService struct {
	repo repository.SubscriptionRepository
	sm   SubscriptionManager
}

// NewSubscriptionService creates a SubscriptionService. sm can be nil to skip Stripe calls.
func NewSubscriptionService(repo repository.SubscriptionRepository, sm SubscriptionManager) SubscriptionService {
	return &subscriptionService{repo: repo, sm: sm}
}

func (s *subscriptionService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

func (s *subscriptionService) GetByID(ctx context.Context, id, userID string) (*model.Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub.DonorType != "user" || sub.DonorID != userID {
		return nil, ErrForbidden
	}
	return sub, nil
}

func (s *subscriptionService) Patch(ctx context.Context, id, userID string, patch model.SubscriptionPatch) error {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sub.DonorType != "user" || sub.DonorID != userID {
		return ErrForbidden
	}

	// Stripe subscription pause/resume
	if patch.Paused != nil && sub.StripeSubscriptionID != "" && s.sm != nil {
		if *patch.Paused {
			if err := s.sm.PauseSubscription(ctx, sub.StripeSubscriptionID); err != nil {
				return fmt.Errorf("stripe pause: %w", err)
			}
		} else {
			if err := s.sm.ResumeSubscription(ctx, sub.StripeSubscriptionID); err != nil {
				return fmt.Errorf("stripe resume: %w", err)
			}
		}
	}

	// Stripe subscription amount update
	if patch.Amount != nil && *patch.Amount != sub.Amount && sub.StripeSubscriptionID != "" && s.sm != nil {
		if err := s.sm.UpdateSubscriptionAmount(ctx, sub.StripeSubscriptionID, *patch.Amount); err != nil {
			return fmt.Errorf("stripe update amount: %w", err)
		}
	}

	return s.repo.Patch(ctx, id, patch)
}

func (s *subscriptionService) Delete(ctx context.Context, id, userID string) error {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sub.DonorType != "user" || sub.DonorID != userID {
		return ErrForbidden
	}

	// Cancel Stripe subscription before deleting
	if sub.StripeSubscriptionID != "" && s.sm != nil {
		if err := s.sm.CancelSubscription(ctx, sub.StripeSubscriptionID); err != nil {
			return fmt.Errorf("stripe cancel: %w", err)
		}
	}

	return s.repo.Delete(ctx, id)
}
