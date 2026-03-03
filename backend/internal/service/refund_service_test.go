package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock RefundProjectRepo
// ---------------------------------------------------------------------------

type mockRefundProjectRepo struct {
	getByIDFunc func(ctx context.Context, id string) (*model.Project, error)
}

func (m *mockRefundProjectRepo) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &model.Project{ID: id, OwnerID: "owner-1"}, nil
}

// ---------------------------------------------------------------------------
// Mock RefundDonationRepo
// ---------------------------------------------------------------------------

type mockRefundDonationRepo struct {
	getByIDFunc          func(ctx context.Context, id string) (*model.Donation, error)
	setRefundPendingFunc func(ctx context.Context, id string) error
	completeRefundFunc   func(ctx context.Context, id string, stripeRefundID string) error
	clearRefundPending   func(ctx context.Context, id string) error
}

func (m *mockRefundDonationRepo) GetByID(ctx context.Context, id string) (*model.Donation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockRefundDonationRepo) SetRefundPending(ctx context.Context, id string) error {
	if m.setRefundPendingFunc != nil {
		return m.setRefundPendingFunc(ctx, id)
	}
	return nil
}
func (m *mockRefundDonationRepo) CompleteRefund(ctx context.Context, id string, stripeRefundID string) error {
	if m.completeRefundFunc != nil {
		return m.completeRefundFunc(ctx, id, stripeRefundID)
	}
	return nil
}
func (m *mockRefundDonationRepo) ClearRefundPending(ctx context.Context, id string) error {
	if m.clearRefundPending != nil {
		return m.clearRefundPending(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock RefundStripeClient
// ---------------------------------------------------------------------------

type mockRefundStripeClient struct {
	createRefundFunc           func(ctx context.Context, paymentIntentID, stripeAccountID string) (string, error)
	getInvoicePaymentIntentFunc func(ctx context.Context, invoiceID, stripeAccountID string) (string, error)
}

func (m *mockRefundStripeClient) CreateRefund(ctx context.Context, paymentIntentID, stripeAccountID string) (string, error) {
	if m.createRefundFunc != nil {
		return m.createRefundFunc(ctx, paymentIntentID, stripeAccountID)
	}
	return "re_mock", nil
}
func (m *mockRefundStripeClient) GetInvoicePaymentIntent(ctx context.Context, invoiceID, stripeAccountID string) (string, error) {
	if m.getInvoicePaymentIntentFunc != nil {
		return m.getInvoicePaymentIntentFunc(ctx, invoiceID, stripeAccountID)
	}
	return "pi_mock", nil
}

// ---------------------------------------------------------------------------
// Tests: RefundByOwner
// ---------------------------------------------------------------------------

func TestRefundService_RefundByOwner_Success(t *testing.T) {
	ctx := context.Background()
	var pendingID, completedID, completedRefundID string

	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1", StripeAccountID: "acct_proj"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID:              id,
				ProjectID:       "proj-1",
				DonorType:       "user",
				DonorID:         "donor-1",
				StripePaymentID: "pi_test",
			}, nil
		},
		setRefundPendingFunc: func(_ context.Context, id string) error {
			pendingID = id
			return nil
		},
		completeRefundFunc: func(_ context.Context, id string, refundID string) error {
			completedID = id
			completedRefundID = refundID
			return nil
		},
	}
	stripeClient := &mockRefundStripeClient{
		createRefundFunc: func(_ context.Context, piID, acctID string) (string, error) {
			if piID != "pi_test" {
				t.Errorf("expected pi_test, got %q", piID)
			}
			if acctID != "acct_proj" {
				t.Errorf("expected acct_proj, got %q", acctID)
			}
			return "re_refund1", nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, stripeClient)
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pendingID != "don-1" {
		t.Errorf("expected SetRefundPending(don-1), got %q", pendingID)
	}
	if completedID != "don-1" {
		t.Errorf("expected CompleteRefund(don-1), got %q", completedID)
	}
	if completedRefundID != "re_refund1" {
		t.Errorf("expected refundID=re_refund1, got %q", completedRefundID)
	}
}

func TestRefundService_RefundByOwner_HostCanRefund(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1", StripeAccountID: "acct_proj"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, ProjectID: "proj-1", StripePaymentID: "pi_test"}, nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	// host user (not owner) should succeed with isHost=true
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "other-user", true)
	if err != nil {
		t.Fatalf("host should be able to refund, got: %v", err)
	}
}

func TestRefundService_RefundByOwner_NotOwner(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "other-user", false)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRefundService_RefundByOwner_DonationWrongProject(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, ProjectID: "other-project", StripePaymentID: "pi_test"}, nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for wrong project, got %v", err)
	}
}

func TestRefundService_RefundByOwner_AlreadyRefunded(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, ProjectID: "proj-1", StripePaymentID: "pi_test"}, nil
		},
		setRefundPendingFunc: func(_ context.Context, id string) error {
			return repository.ErrAlreadyRefunded
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if !errors.Is(err, repository.ErrAlreadyRefunded) {
		t.Errorf("expected ErrAlreadyRefunded, got %v", err)
	}
}

func TestRefundService_RefundByOwner_StripeError_RollsBack(t *testing.T) {
	ctx := context.Background()
	var clearedID string

	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1", StripeAccountID: "acct_proj"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, ProjectID: "proj-1", StripePaymentID: "pi_test"}, nil
		},
		clearRefundPending: func(_ context.Context, id string) error {
			clearedID = id
			return nil
		},
	}
	stripeClient := &mockRefundStripeClient{
		createRefundFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("stripe: charge already refunded")
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, stripeClient)
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if err == nil {
		t.Fatal("expected error for Stripe failure")
	}
	if clearedID != "don-1" {
		t.Errorf("expected ClearRefundPending(don-1), got %q", clearedID)
	}
}

func TestRefundService_RefundByOwner_SubscriptionRenewal_LooksUpInvoice(t *testing.T) {
	ctx := context.Background()
	var lookedUpInvoice string

	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1", StripeAccountID: "acct_proj"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			// No StripePaymentID, only StripeInvoiceID (old renewal before webhook fix)
			return &model.Donation{
				ID:              id,
				ProjectID:       "proj-1",
				StripeInvoiceID: "in_renewal",
			}, nil
		},
	}
	stripeClient := &mockRefundStripeClient{
		getInvoicePaymentIntentFunc: func(_ context.Context, invoiceID, acctID string) (string, error) {
			lookedUpInvoice = invoiceID
			return "pi_from_invoice", nil
		},
		createRefundFunc: func(_ context.Context, piID, _ string) (string, error) {
			if piID != "pi_from_invoice" {
				t.Errorf("expected pi_from_invoice, got %q", piID)
			}
			return "re_refund2", nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, stripeClient)
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lookedUpInvoice != "in_renewal" {
		t.Errorf("expected invoice lookup for in_renewal, got %q", lookedUpInvoice)
	}
}

func TestRefundService_RefundByOwner_NoPaymentInfo(t *testing.T) {
	ctx := context.Background()
	var clearedID string

	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, OwnerID: "owner-1"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			// No StripePaymentID, no StripeInvoiceID
			return &model.Donation{ID: id, ProjectID: "proj-1"}, nil
		},
		clearRefundPending: func(_ context.Context, id string) error {
			clearedID = id
			return nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	err := svc.RefundByOwner(ctx, "proj-1", "don-1", "owner-1", false)
	if err == nil {
		t.Fatal("expected error for no payment info")
	}
	if clearedID != "don-1" {
		t.Errorf("expected ClearRefundPending(don-1), got %q", clearedID)
	}
}

// ---------------------------------------------------------------------------
// Tests: RefundByDonor
// ---------------------------------------------------------------------------

func TestRefundService_RefundByDonor_Success(t *testing.T) {
	ctx := context.Background()
	var completedID string

	projectRepo := &mockRefundProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: id, StripeAccountID: "acct_proj"}, nil
		},
	}
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID:              id,
				ProjectID:       "proj-1",
				DonorType:       "user",
				DonorID:         "donor-1",
				StripePaymentID: "pi_test",
			}, nil
		},
		completeRefundFunc: func(_ context.Context, id string, _ string) error {
			completedID = id
			return nil
		},
	}

	svc := NewRefundService(donationRepo, projectRepo, &mockRefundStripeClient{})
	err := svc.RefundByDonor(ctx, "don-1", "donor-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completedID != "don-1" {
		t.Errorf("expected CompleteRefund(don-1), got %q", completedID)
	}
}

func TestRefundService_RefundByDonor_NotOwnerOfDonation(t *testing.T) {
	ctx := context.Background()
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID:        id,
				ProjectID: "proj-1",
				DonorType: "user",
				DonorID:   "other-user",
			}, nil
		},
	}

	svc := NewRefundService(donationRepo, &mockRefundProjectRepo{}, &mockRefundStripeClient{})
	err := svc.RefundByDonor(ctx, "don-1", "donor-1")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRefundService_RefundByDonor_TokenDonor_Forbidden(t *testing.T) {
	ctx := context.Background()
	donationRepo := &mockRefundDonationRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID:        id,
				ProjectID: "proj-1",
				DonorType: "token",
				DonorID:   "token-abc",
			}, nil
		},
	}

	svc := NewRefundService(donationRepo, &mockRefundProjectRepo{}, &mockRefundStripeClient{})
	err := svc.RefundByDonor(ctx, "don-1", "token-abc")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for token donor, got %v", err)
	}
}
