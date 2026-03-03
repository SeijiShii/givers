package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// RefundProjectRepo is the minimal project repo needed by RefundService.
type RefundProjectRepo interface {
	GetByID(ctx context.Context, id string) (*model.Project, error)
}

// RefundDonationRepo is the minimal donation repo needed by RefundService.
type RefundDonationRepo interface {
	GetByID(ctx context.Context, id string) (*model.Donation, error)
	SetRefundPending(ctx context.Context, id string) error
	CompleteRefund(ctx context.Context, id string, stripeRefundID string) error
	ClearRefundPending(ctx context.Context, id string) error
}

// RefundStripeClient is the minimal Stripe client needed by RefundService.
type RefundStripeClient interface {
	CreateRefund(ctx context.Context, paymentIntentID, stripeAccountID string) (string, error)
	GetInvoicePaymentIntent(ctx context.Context, invoiceID, stripeAccountID string) (string, error)
}

// RefundService handles the refund business logic.
type RefundService interface {
	// RefundByOwner allows a project owner to refund a donation.
	RefundByOwner(ctx context.Context, projectID, donationID, userID string, isHost bool) error
	// RefundByDonor allows a donor to refund their own donation.
	RefundByDonor(ctx context.Context, donationID, userID string) error
}

type refundService struct {
	donationRepo RefundDonationRepo
	projectRepo  RefundProjectRepo
	stripeClient RefundStripeClient
}

// NewRefundService creates a RefundService.
func NewRefundService(donationRepo RefundDonationRepo, projectRepo RefundProjectRepo, stripeClient RefundStripeClient) RefundService {
	return &refundService{
		donationRepo: donationRepo,
		projectRepo:  projectRepo,
		stripeClient: stripeClient,
	}
}

// RefundByOwner processes a refund initiated by a project owner.
func (s *refundService) RefundByOwner(ctx context.Context, projectID, donationID, userID string, isHost bool) error {
	// Verify project ownership
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}
	if project.OwnerID != userID && !isHost {
		return ErrForbidden
	}

	// Verify donation belongs to this project
	donation, err := s.donationRepo.GetByID(ctx, donationID)
	if err != nil {
		return fmt.Errorf("get donation: %w", err)
	}
	if donation.ProjectID != projectID {
		return ErrForbidden
	}

	return s.processRefund(ctx, donation, project.StripeAccountID)
}

// RefundByDonor processes a refund initiated by the donor.
func (s *refundService) RefundByDonor(ctx context.Context, donationID, userID string) error {
	donation, err := s.donationRepo.GetByID(ctx, donationID)
	if err != nil {
		return fmt.Errorf("get donation: %w", err)
	}
	if donation.DonorType != "user" || donation.DonorID != userID {
		return ErrForbidden
	}

	// Get project to find stripe_account_id
	project, err := s.projectRepo.GetByID(ctx, donation.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	return s.processRefund(ctx, donation, project.StripeAccountID)
}

// processRefund is the shared 2-phase refund flow.
func (s *refundService) processRefund(ctx context.Context, donation *model.Donation, stripeAccountID string) error {
	// Phase 1: Mark as pending in DB
	if err := s.donationRepo.SetRefundPending(ctx, donation.ID); err != nil {
		if errors.Is(err, repository.ErrAlreadyRefunded) {
			return err
		}
		return fmt.Errorf("set refund pending: %w", err)
	}

	// Resolve payment_intent ID
	paymentIntentID := donation.StripePaymentID
	if paymentIntentID == "" && donation.StripeInvoiceID != "" {
		// Subscription renewal: look up payment_intent from invoice
		piID, err := s.stripeClient.GetInvoicePaymentIntent(ctx, donation.StripeInvoiceID, stripeAccountID)
		if err != nil {
			// Rollback pending status
			_ = s.donationRepo.ClearRefundPending(ctx, donation.ID)
			return fmt.Errorf("get invoice payment intent: %w", err)
		}
		paymentIntentID = piID
	}
	if paymentIntentID == "" {
		_ = s.donationRepo.ClearRefundPending(ctx, donation.ID)
		return errors.New("no payment intent found for donation")
	}

	// Phase 2: Create refund in Stripe
	refundID, err := s.stripeClient.CreateRefund(ctx, paymentIntentID, stripeAccountID)
	if err != nil {
		// Rollback pending status
		_ = s.donationRepo.ClearRefundPending(ctx, donation.ID)
		return fmt.Errorf("stripe refund: %w", err)
	}

	// Phase 3: Mark as completed in DB
	if err := s.donationRepo.CompleteRefund(ctx, donation.ID, refundID); err != nil {
		return fmt.Errorf("complete refund: %w", err)
	}

	return nil
}
