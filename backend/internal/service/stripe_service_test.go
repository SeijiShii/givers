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
// Tests: invoice.payment_succeeded — next_billing_message (#19)
// ---------------------------------------------------------------------------

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_RecordsMessageAndClears(t *testing.T) {
	ctx := context.Background()
	var recordedActivity *model.ActivityItem
	var patchedID string
	var patchedMessage *string

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_test",
		Amount:       2000,
		Currency:     "jpy",
		Subscription: "sub_msg",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, subID string) (*model.Donation, error) {
			if subID == "sub_msg" {
				return &model.Donation{
					ID: "d1", ProjectID: "proj-1", DonorType: "user", DonorID: "user-1",
					IsRecurring: true, StripeSubscriptionID: "sub_msg",
					NextBillingMessage: "今月もよろしく",
				}, nil
			}
			return nil, errors.New("not found")
		},
		patchFunc: func(_ context.Context, id string, patch model.DonationPatch) error {
			patchedID = id
			patchedMessage = patch.NextBillingMessage
			return nil
		},
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, a *model.ActivityItem) error {
			recordedActivity = a
			return nil
		},
	}
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, donationRepo, activityRecorder)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Activity should be recorded with the next_billing_message
	if recordedActivity == nil {
		t.Fatal("expected activity to be recorded")
	}
	if recordedActivity.Message != "今月もよろしく" {
		t.Errorf("expected message='今月もよろしく', got %q", recordedActivity.Message)
	}
	if recordedActivity.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got %q", recordedActivity.ProjectID)
	}

	// next_billing_message should be cleared
	if patchedID != "d1" {
		t.Errorf("expected patch on donation d1, got %q", patchedID)
	}
	if patchedMessage == nil || *patchedMessage != "" {
		t.Errorf("expected next_billing_message to be cleared (empty string), got %v", patchedMessage)
	}
}

func TestStripeService_ProcessWebhook_InvoicePaymentSucceeded_NoMessage_SkipsActivity(t *testing.T) {
	ctx := context.Background()
	activityCalled := false

	obj := pkgstripe.WebhookEventObject{
		ID:           "in_nomsg",
		Amount:       1000,
		Currency:     "jpy",
		Subscription: "sub_nomsg",
	}
	event := pkgstripe.WebhookEvent{Type: "invoice.payment_succeeded", ID: "evt_inv2"}
	event.Data.Object = obj

	stripeClient := &mockStripeClient{
		verifyWebhookSignatureFunc: func(_ []byte, _ string) error { return nil },
		parseWebhookEventFunc:      func(_ []byte) (pkgstripe.WebhookEvent, error) { return event, nil },
	}
	donationRepo := &mockStripeDonationRepo{
		getByStripeSubscriptionIDFunc: func(_ context.Context, subID string) (*model.Donation, error) {
			return &model.Donation{
				ID: "d2", ProjectID: "proj-2", DonorType: "user", DonorID: "user-2",
				IsRecurring: true, StripeSubscriptionID: "sub_nomsg",
				NextBillingMessage: "", // no message
			}, nil
		},
	}
	activityRecorder := &mockStripeActivityRecorder{
		insertFunc: func(_ context.Context, a *model.ActivityItem) error {
			activityCalled = true
			return nil
		},
	}
	svc := newTestStripeServiceWithActivity(stripeClient, &mockStripeProjectRepo{}, donationRepo, activityRecorder)

	if err := svc.ProcessWebhook(ctx, []byte(`{}`), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if activityCalled {
		t.Error("expected no activity when next_billing_message is empty")
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
