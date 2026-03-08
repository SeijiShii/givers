package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	pkgstripe "github.com/givers/backend/pkg/stripe"
)

// ---------------------------------------------------------------------------
// Mock StripeClient
// ---------------------------------------------------------------------------

type mockStripeClient struct {
	createConnectedAccountFunc func(ctx context.Context, params pkgstripe.CreateAccountParams) (string, error)
	createAccountLinkFunc      func(ctx context.Context, accountID, returnURL, refreshURL string) (string, error)
	getAccountOnboardedFunc    func(ctx context.Context, accountID string) (bool, error)
	createCheckoutSessionFunc  func(ctx context.Context, params pkgstripe.CheckoutParams) (string, error)
	verifyWebhookSignatureFunc func(payload []byte, sigHeader string) error
	parseWebhookEventFunc      func(payload []byte) (pkgstripe.WebhookEvent, error)
}

func (m *mockStripeClient) CreateConnectedAccount(ctx context.Context, params pkgstripe.CreateAccountParams) (string, error) {
	if m.createConnectedAccountFunc != nil {
		return m.createConnectedAccountFunc(ctx, params)
	}
	return "acct_mock", nil
}
func (m *mockStripeClient) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
	if m.createAccountLinkFunc != nil {
		return m.createAccountLinkFunc(ctx, accountID, returnURL, refreshURL)
	}
	return "https://connect.stripe.com/setup/mock", nil
}
func (m *mockStripeClient) GetAccountOnboarded(ctx context.Context, accountID string) (bool, error) {
	if m.getAccountOnboardedFunc != nil {
		return m.getAccountOnboardedFunc(ctx, accountID)
	}
	return true, nil
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
func (m *mockStripeClient) PauseSubscription(_ context.Context, _ string) error  { return nil }
func (m *mockStripeClient) ResumeSubscription(_ context.Context, _ string) error { return nil }
func (m *mockStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (m *mockStripeClient) UpdateSubscriptionAmount(_ context.Context, _ string, _ int) error {
	return nil
}
func (m *mockStripeClient) CreateRefund(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockStripeClient) GetInvoicePaymentIntent(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockStripeClient) CreateCustomer(_ context.Context, _ string) (string, error) {
	return "cus_mock", nil
}
func (m *mockStripeClient) CreateOffSessionPaymentIntent(_ context.Context, _ pkgstripe.OffSessionPaymentParams) (*pkgstripe.OffSessionPaymentResult, error) {
	return &pkgstripe.OffSessionPaymentResult{PaymentIntentID: "pi_mock", Status: "succeeded"}, nil
}
func (m *mockStripeClient) ListPaymentMethods(_ context.Context, _ string) ([]pkgstripe.PaymentMethodSummary, error) {
	return []pkgstripe.PaymentMethodSummary{{ID: "pm_mock", Brand: "visa", Last4: "4242", ExpMonth: 12, ExpYear: 2027}}, nil
}

// ---------------------------------------------------------------------------
// Tests: CreateAccountAndOnboarding
// ---------------------------------------------------------------------------

func TestStripeService_CreateAccountAndOnboarding_Success(t *testing.T) {
	ctx := context.Background()
	var savedProjectID, savedAccountID string

	stripeClient := &mockStripeClient{
		createConnectedAccountFunc: func(_ context.Context, params pkgstripe.CreateAccountParams) (string, error) {
			if params.Country != "jp" {
				t.Errorf("expected country=jp, got %q", params.Country)
			}
			return "acct_test123", nil
		},
		createAccountLinkFunc: func(_ context.Context, accountID, _, _ string) (string, error) {
			if accountID != "acct_test123" {
				t.Errorf("expected accountID=acct_test123, got %q", accountID)
			}
			return "https://connect.stripe.com/setup/test", nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		saveStripeAccountIDFunc: func(_ context.Context, projectID, stripeAccountID string) error {
			savedProjectID = projectID
			savedAccountID = stripeAccountID
			return nil
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	url, err := svc.CreateAccountAndOnboarding(ctx, "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://connect.stripe.com/setup/test" {
		t.Errorf("unexpected URL: %q", url)
	}
	if savedProjectID != "proj-1" {
		t.Errorf("expected projectID=proj-1, got %q", savedProjectID)
	}
	if savedAccountID != "acct_test123" {
		t.Errorf("expected stripeAccountID=acct_test123, got %q", savedAccountID)
	}
}

func TestStripeService_CreateAccountAndOnboarding_StripeError(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		createConnectedAccountFunc: func(_ context.Context, _ pkgstripe.CreateAccountParams) (string, error) {
			return "", errors.New("stripe error")
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, &mockStripeProjectRepo{})

	_, err := svc.CreateAccountAndOnboarding(ctx, "proj-1")
	if err == nil {
		t.Error("expected error on Stripe failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: CompleteOnboarding
// ---------------------------------------------------------------------------

func TestStripeService_CompleteOnboarding_Success(t *testing.T) {
	ctx := context.Background()
	var activatedProjectID string

	stripeClient := &mockStripeClient{
		getAccountOnboardedFunc: func(_ context.Context, accountID string) (bool, error) {
			if accountID != "acct_test123" {
				t.Errorf("expected accountID=acct_test123, got %q", accountID)
			}
			return true, nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, id string) (string, error) {
			return "acct_test123", nil
		},
		activateProjectFunc: func(_ context.Context, projectID string) error {
			activatedProjectID = projectID
			return nil
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	if err := svc.CompleteOnboarding(ctx, "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if activatedProjectID != "proj-1" {
		t.Errorf("expected activatedProjectID=proj-1, got %q", activatedProjectID)
	}
}

func TestStripeService_CompleteOnboarding_NotYetComplete(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		getAccountOnboardedFunc: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, _ string) (string, error) {
			return "acct_test123", nil
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	err := svc.CompleteOnboarding(ctx, "proj-1")
	if err == nil {
		t.Error("expected error when onboarding not complete")
	}
}

func TestStripeService_CompleteOnboarding_NoAccount(t *testing.T) {
	ctx := context.Background()
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", nil // empty = no account
		},
	}
	svc := newTestStripeServiceWithRepo(&mockStripeClient{}, projectRepo)

	err := svc.CompleteOnboarding(ctx, "proj-1")
	if err == nil {
		t.Error("expected error when no account linked")
	}
}

// ---------------------------------------------------------------------------
// Tests: RefreshOnboarding
// ---------------------------------------------------------------------------

func TestStripeService_RefreshOnboarding_Success(t *testing.T) {
	ctx := context.Background()
	stripeClient := &mockStripeClient{
		createAccountLinkFunc: func(_ context.Context, accountID, _, _ string) (string, error) {
			if accountID != "acct_test123" {
				t.Errorf("expected accountID=acct_test123, got %q", accountID)
			}
			return "https://connect.stripe.com/setup/refresh", nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, _ string) (string, error) {
			return "acct_test123", nil
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	url, err := svc.RefreshOnboarding(ctx, "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://connect.stripe.com/setup/refresh" {
		t.Errorf("unexpected URL: %q", url)
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

func TestStripeService_CreateCheckout_PlatformDirect(t *testing.T) {
	ctx := context.Background()
	var capturedParams pkgstripe.CheckoutParams

	stripeClient := &mockStripeClient{
		createCheckoutSessionFunc: func(_ context.Context, params pkgstripe.CheckoutParams) (string, error) {
			capturedParams = params
			return "https://checkout.stripe.com/platform", nil
		},
	}
	projectRepo := &mockStripeProjectRepo{
		getByIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", nil // empty stripeAccountID = platform direct
		},
	}
	svc := newTestStripeServiceWithRepo(stripeClient, projectRepo)

	url, err := svc.CreateCheckout(ctx, CheckoutRequest{
		ProjectID: "proj-host",
		Amount:    500,
		Currency:  "jpy",
	})
	if err != nil {
		t.Fatalf("expected no error for platform-direct checkout, got: %v", err)
	}
	if url != "https://checkout.stripe.com/platform" {
		t.Errorf("unexpected URL: %q", url)
	}
	if capturedParams.StripeAccountID != "" {
		t.Errorf("expected empty StripeAccountID for platform-direct, got %q", capturedParams.StripeAccountID)
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

func TestStripeService_ProcessWebhook_PaymentIntentSucceeded_Idempotent(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_duplicate",
		Amount:   1500,
		Currency: "jpy",
		Metadata: map[string]string{
			"project_id": "proj-1",
			"donor_type": "user",
			"donor_id":   "user-1",
		},
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded", ID: "evt_dup"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, _ *model.Donation) error {
			return repository.ErrDuplicate // simulate UNIQUE constraint violation
		},
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, donationRepo)

	// Duplicate should be silently ignored — no error returned
	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("expected no error for duplicate payment_intent, got: %v", err)
	}
}

func TestStripeService_ProcessWebhook_SubscriptionCreated_Idempotent(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID: "sub_duplicate",
		Metadata: map[string]string{
			"project_id": "proj-1",
			"donor_type": "user",
			"donor_id":   "user-1",
		},
		Plan: &struct {
			Amount   int    `json:"amount"`
			Currency string `json:"currency"`
		}{Amount: 2000, Currency: "jpy"},
	}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.created", ID: "evt_dup_sub"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, _ *model.Donation) error {
			return repository.ErrDuplicate
		},
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, donationRepo)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("expected no error for duplicate subscription, got: %v", err)
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

func TestStripeService_ProcessWebhook_SubscriptionCreated_CreatesDonation(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation

	obj := pkgstripe.WebhookEventObject{
		ID:       "sub_test",
		Amount:   0, // subscription の amount はトップレベルではなく plan に
		Currency: "",
		Metadata: map[string]string{
			"project_id":  "proj-2",
			"donor_type":  "user",
			"donor_id":    "user-2",
			"message":     "毎月応援します",
			"is_recurring": "true",
		},
		Plan: &struct {
			Amount   int    `json:"amount"`
			Currency string `json:"currency"`
		}{Amount: 2000, Currency: "jpy"},
	}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.created", ID: "evt_sub"}
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
		t.Fatal("expected donation to be created for subscription")
	}
	if createdDonation.ProjectID != "proj-2" {
		t.Errorf("expected ProjectID=proj-2, got %q", createdDonation.ProjectID)
	}
	if createdDonation.Amount != 2000 {
		t.Errorf("expected Amount=2000, got %d", createdDonation.Amount)
	}
	if !createdDonation.IsRecurring {
		t.Error("expected IsRecurring=true")
	}
	if createdDonation.StripeSubscriptionID != "sub_test" {
		t.Errorf("expected StripeSubscriptionID=sub_test, got %q", createdDonation.StripeSubscriptionID)
	}
}

func TestStripeService_ProcessWebhook_SubscriptionDeleted_DeletesDonation(t *testing.T) {
	ctx := context.Background()
	var deletedSubID string

	obj := pkgstripe.WebhookEventObject{ID: "sub_to_delete"}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.deleted", ID: "evt_del"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		deleteBySubscriptionIDFunc: func(_ context.Context, subID string) error {
			deletedSubID = subID
			return nil
		},
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, donationRepo)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedSubID != "sub_to_delete" {
		t.Errorf("expected deleted sub_to_delete, got %q", deletedSubID)
	}
}

// ---------------------------------------------------------------------------
// Tests: Activity recording on donation creation
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_PaymentIntentSucceeded_RecordsActivity(t *testing.T) {
	ctx := context.Background()
	var recordedActivity *model.ActivityItem

	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_act",
		Amount:   3000,
		Currency: "jpy",
		Metadata: map[string]string{
			"project_id": "proj-1",
			"donor_type": "user",
			"donor_id":   "user-1",
			"message":    "応援してます",
		},
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded", ID: "evt_act"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{}, activityRecorder)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recordedActivity == nil {
		t.Fatal("expected activity to be recorded")
	}
	if recordedActivity.Type != "donation" {
		t.Errorf("expected type=donation, got %q", recordedActivity.Type)
	}
	if recordedActivity.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got %q", recordedActivity.ProjectID)
	}
	if recordedActivity.Amount == nil || *recordedActivity.Amount != 3000 {
		t.Errorf("expected Amount=3000, got %v", recordedActivity.Amount)
	}
	if recordedActivity.ActorName == nil || *recordedActivity.ActorName != "user-1" {
		t.Errorf("expected ActorName=user-1, got %v", recordedActivity.ActorName)
	}
	if recordedActivity.Message != "応援してます" {
		t.Errorf("expected Message=応援してます, got %q", recordedActivity.Message)
	}
}

func TestStripeService_ProcessWebhook_SubscriptionCreated_RecordsActivity(t *testing.T) {
	ctx := context.Background()
	var recordedActivity *model.ActivityItem

	obj := pkgstripe.WebhookEventObject{
		ID: "sub_act",
		Metadata: map[string]string{
			"project_id": "proj-2",
			"donor_type": "token",
			"donor_id":   "tok-abc",
		},
		Plan: &struct {
			Amount   int    `json:"amount"`
			Currency string `json:"currency"`
		}{Amount: 2000, Currency: "jpy"},
	}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.created", ID: "evt_sub_act"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{}, activityRecorder)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recordedActivity == nil {
		t.Fatal("expected activity to be recorded for subscription")
	}
	if recordedActivity.Type != "donation" {
		t.Errorf("expected type=donation, got %q", recordedActivity.Type)
	}
	if recordedActivity.Amount == nil || *recordedActivity.Amount != 2000 {
		t.Errorf("expected Amount=2000, got %v", recordedActivity.Amount)
	}
}

func TestStripeService_ProcessWebhook_ActivityRecordError_DoesNotFailWebhook(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_act_err",
		Amount:   1000,
		Currency: "jpy",
		Metadata: map[string]string{"project_id": "proj-1", "donor_type": "user", "donor_id": "user-1"},
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded", ID: "evt_act_err"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, _ *model.ActivityItem) error {
			return errors.New("activity db error")
		},
	}
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{}, activityRecorder)

	// Activity recording error should NOT cause webhook processing to fail
	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("expected no error even when activity recording fails, got: %v", err)
	}
}

func TestStripeService_ProcessWebhook_NilActivityRecorder_StillWorks(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID:       "pi_nil",
		Amount:   1000,
		Currency: "jpy",
		Metadata: map[string]string{"project_id": "proj-1", "donor_type": "user", "donor_id": "user-1"},
	}
	event := pkgstripe.WebhookEvent{Type: "payment_intent.succeeded", ID: "evt_nil"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	// nil activity recorder — should not panic
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{}, nil)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("expected no error with nil activity recorder, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers / mock repos for stripe tests
// ---------------------------------------------------------------------------

type mockStripeProjectRepo struct {
	getByIDFunc             func(ctx context.Context, id string) (string, error) // returns stripeAccountID
	saveStripeAccountIDFunc func(ctx context.Context, projectID, stripeAccountID string) error
	activateProjectFunc     func(ctx context.Context, projectID string) error
}

func (m *mockStripeProjectRepo) GetStripeAccountID(ctx context.Context, id string) (string, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return "", nil
}
func (m *mockStripeProjectRepo) SaveStripeAccountID(ctx context.Context, projectID, stripeAccountID string) error {
	if m.saveStripeAccountIDFunc != nil {
		return m.saveStripeAccountIDFunc(ctx, projectID, stripeAccountID)
	}
	return nil
}
func (m *mockStripeProjectRepo) ActivateProject(ctx context.Context, projectID string) error {
	if m.activateProjectFunc != nil {
		return m.activateProjectFunc(ctx, projectID)
	}
	return nil
}

type mockStripeDonationRepo struct {
	createFunc                    func(ctx context.Context, d *model.Donation) error
	getByIDFunc                   func(ctx context.Context, id string) (*model.Donation, error)
	deleteBySubscriptionIDFunc    func(ctx context.Context, subscriptionID string) error
	getByStripeSubscriptionIDFunc func(ctx context.Context, subscriptionID string) (*model.Donation, error)
	patchFunc                     func(ctx context.Context, id string, patch model.DonationPatch) error
}

func (m *mockStripeDonationRepo) Create(ctx context.Context, d *model.Donation) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, d)
	}
	return nil
}

func (m *mockStripeDonationRepo) GetByID(ctx context.Context, id string) (*model.Donation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockStripeDonationRepo) DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error {
	if m.deleteBySubscriptionIDFunc != nil {
		return m.deleteBySubscriptionIDFunc(ctx, subscriptionID)
	}
	return nil
}

func (m *mockStripeDonationRepo) GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error) {
	if m.getByStripeSubscriptionIDFunc != nil {
		return m.getByStripeSubscriptionIDFunc(ctx, subscriptionID)
	}
	return nil, nil
}

func (m *mockStripeDonationRepo) Patch(ctx context.Context, id string, patch model.DonationPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, patch)
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

type mockStripeActivityRecorder struct {
	insertFunc func(ctx context.Context, a *model.ActivityItem) error
}

func (m *mockStripeActivityRecorder) Insert(ctx context.Context, a *model.ActivityItem) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, a)
	}
	return nil
}

func newTestStripeServiceWithActivity(client pkgstripe.Client, projectRepo StripeProjectRepo, donationRepo StripeDonationRepo, activityRecorder StripeActivityRecorder) StripeService {
	return NewStripeServiceWithActivity(client, projectRepo, donationRepo, "https://example.com", activityRecorder, nil)
}

// ---------------------------------------------------------------------------
// Mock StripeSubscriptionRepo
// ---------------------------------------------------------------------------

type mockStripeSubscriptionRepo struct {
	createFunc                    func(ctx context.Context, s *model.Subscription) error
	getByStripeSubscriptionIDFunc func(ctx context.Context, stripeSubID string) (*model.Subscription, error)
	deleteByStripeSubscriptionIDFunc func(ctx context.Context, stripeSubID string) error
	patchFunc                     func(ctx context.Context, id string, patch model.SubscriptionPatch) error
}

func (m *mockStripeSubscriptionRepo) Create(ctx context.Context, s *model.Subscription) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	return nil
}
func (m *mockStripeSubscriptionRepo) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*model.Subscription, error) {
	if m.getByStripeSubscriptionIDFunc != nil {
		return m.getByStripeSubscriptionIDFunc(ctx, stripeSubID)
	}
	return nil, nil
}
func (m *mockStripeSubscriptionRepo) DeleteByStripeSubscriptionID(ctx context.Context, stripeSubID string) error {
	if m.deleteByStripeSubscriptionIDFunc != nil {
		return m.deleteByStripeSubscriptionIDFunc(ctx, stripeSubID)
	}
	return nil
}
func (m *mockStripeSubscriptionRepo) Patch(ctx context.Context, id string, patch model.SubscriptionPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, patch)
	}
	return nil
}

func newTestStripeServiceWithSubscriptionRepo(
	client pkgstripe.Client,
	projectRepo StripeProjectRepo,
	donationRepo StripeDonationRepo,
	subscriptionRepo StripeSubscriptionRepo,
	activityRecorder StripeActivityRecorder,
) StripeService {
	return NewStripeServiceFull(client, projectRepo, donationRepo, subscriptionRepo, nil, "https://example.com", activityRecorder, nil)
}

// ---------------------------------------------------------------------------
// Tests: invoice.payment_succeeded — creates donation record + handles next_billing_message
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_CreatesDonation(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation
	var recordedActivity *model.ActivityItem

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_test",
		Amount:       2000,
		Currency:     "jpy",
		Subscription: "sub_renew",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	subscriptionRepo := &mockStripeSubscriptionRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, subID string) (*model.Subscription, error) {
			if subID == "sub_renew" {
				return &model.Subscription{
					ID: "s1", ProjectID: "proj-1", DonorType: "user", DonorID: "user-1",
					Amount: 2000, Currency: "jpy",
					StripeSubscriptionID: "sub_renew",
				}, nil
			}
			return nil, nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, activityRecorder)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdDonation == nil {
		t.Fatal("expected donation to be created for renewal")
	}
	if createdDonation.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got %q", createdDonation.ProjectID)
	}
	if createdDonation.Amount != 2000 {
		t.Errorf("expected Amount=2000, got %d", createdDonation.Amount)
	}
	if createdDonation.Source != "subscription_renewal" {
		t.Errorf("expected Source=subscription_renewal, got %q", createdDonation.Source)
	}
	if createdDonation.StripeInvoiceID != "in_test" {
		t.Errorf("expected StripeInvoiceID=in_test, got %q", createdDonation.StripeInvoiceID)
	}
	if createdDonation.SubscriptionID != "s1" {
		t.Errorf("expected SubscriptionID=s1, got %q", createdDonation.SubscriptionID)
	}
	if !createdDonation.IsRecurring {
		t.Error("expected IsRecurring=true")
	}
	if recordedActivity == nil {
		t.Fatal("expected activity to be recorded")
	}
	if recordedActivity.ProjectID != "proj-1" {
		t.Errorf("expected activity ProjectID=proj-1, got %q", recordedActivity.ProjectID)
	}
}

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_WithNextBillingMessage(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation
	var patchedSubID string
	var patchedMessage *string

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_msg",
		Amount:       2000,
		Currency:     "jpy",
		Subscription: "sub_msg",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv_msg"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	subscriptionRepo := &mockStripeSubscriptionRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, subID string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: "s2", ProjectID: "proj-1", DonorType: "user", DonorID: "user-1",
				Amount: 2000, Currency: "jpy",
				StripeSubscriptionID: "sub_msg",
				NextBillingMessage:   "今月もよろしく",
			}, nil
		},
		patchFunc: func(_ context.Context, id string, patch model.SubscriptionPatch) error {
			patchedSubID = id
			patchedMessage = patch.NextBillingMessage
			return nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
	}
	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, &mockStripeActivityRecorder{})

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Donation should carry the next_billing_message
	if createdDonation == nil {
		t.Fatal("expected donation to be created")
	}
	if createdDonation.Message != "今月もよろしく" {
		t.Errorf("expected message='今月もよろしく', got %q", createdDonation.Message)
	}

	// next_billing_message should be cleared on subscription
	if patchedSubID != "s2" {
		t.Errorf("expected patch on subscription s2, got %q", patchedSubID)
	}
	if patchedMessage == nil || *patchedMessage != "" {
		t.Errorf("expected next_billing_message to be cleared, got %v", patchedMessage)
	}
}

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_FallbackDonationRepo(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation
	var patchedID string
	var patchedMessage *string

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_fallback",
		Amount:       1500,
		Currency:     "jpy",
		Subscription: "sub_old",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv_fb"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, subID string) (*model.Donation, error) {
			return &model.Donation{
				ID: "d1", ProjectID: "proj-2", DonorType: "user", DonorID: "user-2",
				Amount: 1500, Currency: "jpy",
				StripeSubscriptionID: "sub_old",
				NextBillingMessage:   "メッセージ",
			}, nil
		},
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
		patchFunc: func(_ context.Context, id string, patch model.DonationPatch) error {
			patchedID = id
			patchedMessage = patch.NextBillingMessage
			return nil
		},
	}
	// No subscriptionRepo — fallback to donationRepo
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, donationRepo, &mockStripeActivityRecorder{})

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdDonation == nil {
		t.Fatal("expected donation to be created via fallback")
	}
	if createdDonation.Source != "subscription_renewal" {
		t.Errorf("expected Source=subscription_renewal, got %q", createdDonation.Source)
	}

	// next_billing_message should be cleared on original donation
	if patchedID != "d1" {
		t.Errorf("expected patch on donation d1, got %q", patchedID)
	}
	if patchedMessage == nil || *patchedMessage != "" {
		t.Errorf("expected next_billing_message to be cleared, got %v", patchedMessage)
	}
}

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_Idempotent(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_dup",
		Amount:       2000,
		Currency:     "jpy",
		Subscription: "sub_dup",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv_dup"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	subscriptionRepo := &mockStripeSubscriptionRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, _ string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: "s1", ProjectID: "proj-1", DonorType: "user", DonorID: "user-1",
				Amount: 2000, Currency: "jpy", StripeSubscriptionID: "sub_dup",
			}, nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, _ *model.Donation) error {
			return repository.ErrDuplicate // simulate duplicate invoice
		},
	}
	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, nil)

	// Duplicate should be silently ignored
	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("expected no error for duplicate invoice, got: %v", err)
	}
}

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_NoSubscription_Skips(t *testing.T) {
	ctx := context.Background()

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_nosub",
		Amount:       500,
		Currency:     "jpy",
		Subscription: "", // no subscription — one-time invoice
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv3"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	svc := newTestStripeServiceFull(stripeClient, &mockStripeProjectRepo{}, &mockStripeDonationRepo{})

	// No subscription → just ignore
	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: subscription.created with subscriptionRepo
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_SubscriptionCreated_WithSubscriptionRepo(t *testing.T) {
	ctx := context.Background()
	var createdSub *model.Subscription
	var createdDonation *model.Donation

	obj := pkgstripe.WebhookEventObject{
		ID: "sub_new",
		Metadata: map[string]string{
			"project_id": "proj-3",
			"donor_type": "user",
			"donor_id":   "user-3",
			"message":    "毎月応援",
		},
		Plan: &struct {
			Amount   int    `json:"amount"`
			Currency string `json:"currency"`
		}{Amount: 3000, Currency: "jpy"},
	}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.created", ID: "evt_sub_new"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	subscriptionRepo := &mockStripeSubscriptionRepo{
		createFunc: func(_ context.Context, s *model.Subscription) error {
			s.ID = "sub-table-id"
			createdSub = s
			return nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
	}
	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, nil)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdSub == nil {
		t.Fatal("expected subscription to be created")
	}
	if createdSub.ProjectID != "proj-3" {
		t.Errorf("expected sub ProjectID=proj-3, got %q", createdSub.ProjectID)
	}
	if createdSub.Amount != 3000 {
		t.Errorf("expected sub Amount=3000, got %d", createdSub.Amount)
	}
	if createdSub.StripeSubscriptionID != "sub_new" {
		t.Errorf("expected sub StripeSubscriptionID=sub_new, got %q", createdSub.StripeSubscriptionID)
	}

	if createdDonation == nil {
		t.Fatal("expected donation to be created")
	}
	if createdDonation.SubscriptionID != "sub-table-id" {
		t.Errorf("expected donation SubscriptionID=sub-table-id, got %q", createdDonation.SubscriptionID)
	}
	if createdDonation.Source != "checkout" {
		t.Errorf("expected donation Source=checkout, got %q", createdDonation.Source)
	}
}

// ---------------------------------------------------------------------------
// Tests: subscription.deleted with subscriptionRepo
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_SubscriptionDeleted_WithSubscriptionRepo(t *testing.T) {
	ctx := context.Background()
	var deletedSubID string

	obj := pkgstripe.WebhookEventObject{ID: "sub_cancel"}
	event := pkgstripe.WebhookEvent{Type: "customer.subscription.deleted", ID: "evt_del2"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	subscriptionRepo := &mockStripeSubscriptionRepo{
		deleteByStripeSubscriptionIDFunc: func(_ context.Context, subID string) error {
			deletedSubID = subID
			return nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		deleteBySubscriptionIDFunc: func(_ context.Context, _ string) error {
			t.Error("donationRepo.DeleteByStripeSubscriptionID should not be called when subscriptionRepo is set")
			return nil
		},
	}
	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, nil)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedSubID != "sub_cancel" {
		t.Errorf("expected deleted sub_cancel, got %q", deletedSubID)
	}
}

// ---------------------------------------------------------------------------
// Tests: invoice.payment_succeeded — saves PaymentIntent for refund support
// ---------------------------------------------------------------------------

func TestStripeService_InvoicePaymentSucceeded_SavesPaymentIntent(t *testing.T) {
	ctx := context.Background()
	var createdDonation *model.Donation

	obj := pkgstripe.WebhookEventObject{
		ID:            "in_renewal",
		Amount:        3000,
		Currency:      "jpy",
		Subscription:  "sub_existing",
		PaymentIntent: "pi_from_invoice",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv2"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		parseWebhookEventFunc: func(_ []byte) (pkgstripe.WebhookEvent, error) {
			return event, nil
		},
	}

	subscriptionRepo := &mockStripeSubscriptionRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, stripeSubID string) (*model.Subscription, error) {
			return &model.Subscription{
				ID:                   "sub-table-1",
				ProjectID:            "proj-1",
				DonorType:            "user",
				DonorID:              "user-1",
				Amount:               3000,
				Currency:             "jpy",
				StripeSubscriptionID: stripeSubID,
			}, nil
		},
	}
	donationRepo := &mockStripeDonationRepo{
		createFunc: func(_ context.Context, d *model.Donation) error {
			createdDonation = d
			return nil
		},
	}

	svc := newTestStripeServiceWithSubscriptionRepo(stripeClient, &mockStripeProjectRepo{}, donationRepo, subscriptionRepo, nil)
	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdDonation == nil {
		t.Fatal("expected donation to be created")
	}
	if createdDonation.StripePaymentID != "pi_from_invoice" {
		t.Errorf("expected StripePaymentID=pi_from_invoice, got %q", createdDonation.StripePaymentID)
	}
	if createdDonation.StripeInvoiceID != "in_renewal" {
		t.Errorf("expected StripeInvoiceID=in_renewal, got %q", createdDonation.StripeInvoiceID)
	}
}
