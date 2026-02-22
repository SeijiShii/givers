package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
	pkgstripe "github.com/givers/backend/pkg/stripe"
)

// ---------------------------------------------------------------------------
// Mock StripeClient
// ---------------------------------------------------------------------------

type mockStripeClient struct {
	generateConnectURLFunc    func(projectID string) string
	exchangeConnectCodeFunc   func(ctx context.Context, code string) (string, error)
	createCheckoutSessionFunc func(ctx context.Context, params pkgstripe.CheckoutParams) (string, error)
	verifyWebhookSignatureFunc func(payload []byte, sigHeader string) error
	parseWebhookEventFunc     func(payload []byte) (pkgstripe.WebhookEvent, error)
}

func (m *mockStripeClient) GenerateConnectURL(projectID string) string {
	if m.generateConnectURLFunc != nil {
		return m.generateConnectURLFunc(projectID)
	}
	return ""
}
func (m *mockStripeClient) ExchangeConnectCode(ctx context.Context, code string) (string, error) {
	if m.exchangeConnectCodeFunc != nil {
		return m.exchangeConnectCodeFunc(ctx, code)
	}
	return "", nil
}
func (m *mockStripeClient) CreateCheckoutSession(ctx context.Context, params pkgstripe.CheckoutParams) (string, error) {
	if m.createCheckoutSessionFunc != nil {
		return m.createCheckoutSessionFunc(ctx, params)
	}
	return "", nil
}
func (m *mockStripeClient) VerifyWebhookSignature(payload []byte, sigHeader string) error {
	if m.verifyWebhookSignatureFunc != nil {
		return m.verifyWebhookSignatureFunc(payload, sigHeader)
	}
	return nil
}
func (m *mockStripeClient) ParseWebhookEvent(payload []byte) (pkgstripe.WebhookEvent, error) {
	if m.parseWebhookEventFunc != nil {
		return m.parseWebhookEventFunc(payload)
	}
	return pkgstripe.WebhookEvent{}, nil
}

// ---------------------------------------------------------------------------
// Mock ProjectRepository (slim: only methods needed by StripeService)
// ---------------------------------------------------------------------------

type mockProjectRepoForStripe struct {
	getByIDFunc            func(ctx context.Context, id string) (*mockProjectBasic, error)
	updateStripeConnectFunc func(ctx context.Context, projectID, stripeAccountID string) error
}

// We don't want to import model here, so we use a simple struct
type mockProjectBasic struct {
	id              string
	ownerID         string
	stripeAccountID string
}

// ---------------------------------------------------------------------------
// Tests: GenerateConnectURL
// ---------------------------------------------------------------------------

func TestStripeService_GenerateConnectURL_WithClientID(t *testing.T) {
	mock := &mockStripeClient{
		generateConnectURLFunc: func(projectID string) string {
			if projectID != "proj-1" {
				t.Errorf("expected project-1, got %q", projectID)
			}
			return "https://connect.stripe.com/oauth/authorize?client_id=ca_xxx&state=proj-1"
		},
	}
	svc := newTestStripeService(mock)
	url := svc.GenerateConnectURL("proj-1")
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestStripeService_GenerateConnectURL_EmptyWhenNotConfigured(t *testing.T) {
	mock := &mockStripeClient{
		generateConnectURLFunc: func(_ string) string { return "" },
	}
	svc := newTestStripeService(mock)
	url := svc.GenerateConnectURL("proj-1")
	if url != "" {
		t.Errorf("expected empty URL when not configured, got %q", url)
	}
}

// ---------------------------------------------------------------------------
// Tests: CompleteConnect
// ---------------------------------------------------------------------------

func TestStripeService_CompleteConnect_Success(t *testing.T) {
	ctx := context.Background()
	var savedProjectID, savedAccountID string

	stripeClient := &mockStripeClient{
		exchangeConnectCodeFunc: func(_ context.Context, code string) (string, error) {
			if code != "auth_code_123" {
				t.Errorf("expected code=auth_code_123, got %q", code)
			}
			return "acct_test123", nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		updateStripeConnectFunc: func(_ context.Context, projectID, stripeAccountID string) error {
			savedProjectID = projectID
			savedAccountID = stripeAccountID
			return nil
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	if err := svc.CompleteConnect(ctx, "auth_code_123", "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedProjectID != "proj-1" {
		t.Errorf("expected projectID=proj-1, got %q", savedProjectID)
	}
	if savedAccountID != "acct_test123" {
		t.Errorf("expected stripeAccountID=acct_test123, got %q", savedAccountID)
	}
}

func TestStripeService_CompleteConnect_StripeError(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		exchangeConnectCodeFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("invalid code")
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, &mockStripeProjectRepo{})

	err := svc.CompleteConnect(ctx, "bad_code", "proj-1")
	if err == nil {
		t.Error("expected error on Stripe failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateCheckout
// ---------------------------------------------------------------------------

func TestStripeService_CreateCheckout_Success(t *testing.T) {
	ctx := context.Background()

	stripeClient := &mockStripeClient{
		createCheckoutSessionFunc: func(_ context.Context, params pkgstripe.CheckoutParams) (string, error) {
			if params.StripeAccountID != "acct_owner" {
				t.Errorf("expected StripeAccountID=acct_owner, got %q", params.StripeAccountID)
			}
			if params.Amount != 1000 {
				t.Errorf("expected Amount=1000, got %d", params.Amount)
			}
			return "https://checkout.stripe.com/test", nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (string, error) {
			if id == "proj-1" {
				return "acct_owner", nil // return stripeAccountID
			}
			return "", errors.New("not found")
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	url, err := svc.CreateCheckout(ctx, CheckoutRequest{
		ProjectID:   "proj-1",
		Amount:      1000,
		Currency:    "jpy",
		IsRecurring: false,
		FrontendURL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://checkout.stripe.com/test" {
		t.Errorf("unexpected URL: %q", url)
	}
}

func TestStripeService_CreateCheckout_ProjectNotConnected(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", nil // empty stripeAccountID = not connected
		},
	}
	svc := newTestStripeServiceWithRepo(&mockStripeClient{}, projectRepo)

	_, err := svc.CreateCheckout(ctx, CheckoutRequest{
		ProjectID: "proj-1",
		Amount:    500,
		Currency:  "jpy",
	})
	if err == nil {
		t.Error("expected error when project has no stripe account")
	}
}

func TestStripeService_CreateCheckout_AmountTooLow(t *testing.T) {
	ctx := context.Background()
	svc := newTestStripeServiceWithRepo(&mockStripeClient{}, &mockStripeProjectRepo{})
	_, err := svc.CreateCheckout(ctx, CheckoutRequest{Amount: 0})
	if err == nil {
		t.Error("expected error for amount=0")
	}
}

// ---------------------------------------------------------------------------
// Tests: ProcessWebhook
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_InvalidSignature(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error {
			return errors.New("invalid signature")
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, &mockStripeProjectRepo{})

	err := svc.ProcessWebhook(ctx, []byte(`{}`), "bad-sig")
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestStripeService_ProcessWebhook_ValidSignature_UnknownEvent(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc: func(_ []byte) (pkgstripe.WebhookEvent, error) {
			return pkgstripe.WebhookEvent{Type: "unknown.event"}, nil
		},
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{})

	if err := svc.ProcessWebhook(ctx, []byte(`{"type":"unknown.event"}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error for unknown event: %v", err)
	}
}

func TestStripeService_ProcessWebhook_PaymentIntentSucceeded_CreatesDonation(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation

	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_test",
		Amount:   1500,
		Currency: "jpy",
		Metadata: map[string]string{
			"project_id": "proj-1",
			"donor_type": "user",
			"donor_id":   "user-1",
			"message":    "頑張れ",
		},
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded", ID: "evt_test"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, donationRepo)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdDonation == nil {
		t.Fatal("expected donation to be created")
	}
	if createdDonation.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got %q", createdDonation.ProjectID)
	}
	if createdDonation.Amount != 1500 {
		t.Errorf("expected Amount=1500, got %d", createdDonation.Amount)
	}
	if createdDonation.DonorType != "user" {
		t.Errorf("expected DonorType=user, got %q", createdDonation.DonorType)
	}
	if createdDonation.StripePaymentID != "pi_test" {
		t.Errorf("expected StripePaymentID=pi_test, got %q", createdDonation.StripePaymentID)
	}
}

func TestStripeService_ProcessWebhook_PaymentIntentSucceeded_MissingProjectID(t *testing.T) {
	ctx := context.Background()
	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_test",
		Amount:   1000,
		Currency: "jpy",
		Metadata: map[string]string{}, // no project_id
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{})

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err == nil {
		t.Error("expected error when project_id missing from metadata")
	}
}

// ---------------------------------------------------------------------------
// Helpers / mock project repo for stripe tests
// ---------------------------------------------------------------------------

type mockStripeProjectRepo struct {
	getByIDFunc             func(ctx context.Context, id string) (string, error) // returns stripeAccountID
	updateStripeConnectFunc func(ctx context.Context, projectID, stripeAccountID string) error
}

func (m *mockStripeProjectRepo) GetStripeAccountID(ctx context.Context, id string) (string, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return "", nil
}
func (m *mockStripeProjectRepo) UpdateStripeConnect(ctx context.Context, projectID, stripeAccountID string) error {
	if m.updateStripeConnectFunc != nil {
		return m.updateStripeConnectFunc(ctx, projectID, stripeAccountID)
	}
	return nil
}

type mockStripeDonationRepo struct {
	createFunc func(ctx context.Context, d *model.Donation) error
}

func (m *mockStripeDonationRepo) Create(ctx context.Context, d *model.Donation) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, d)
	}
	return nil
}

func newTestStripeService(client pkgstripe.Client) StripeService {
	return NewStripeService(client, &mockStripeProjectRepo{}, &mockStripeDonationRepo{}, "https://example.com")
}

func newTestStripeServiceWithRepo(client pkgstripe.Client, repo StripeProjectRepo) StripeService {
	return NewStripeService(client, repo, &mockStripeDonationRepo{}, "https://example.com")
}

func newTestStripeServiceFull(client pkgstripe.Client, projectRepo StripeProjectRepo, donationRepo StripeDonationRepo) StripeService {
	return NewStripeService(client, projectRepo, donationRepo, "https://example.com")
}
