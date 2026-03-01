package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock StripeService
// ---------------------------------------------------------------------------

type mockStripeService struct {
	createAccountAndOnboardingFunc func(ctx context.Context, projectID string) (string, error)
	completeOnboardingFunc         func(ctx context.Context, projectID string) error
	refreshOnboardingFunc          func(ctx context.Context, projectID string) (string, error)
	createCheckoutFunc             func(ctx context.Context, req service.CheckoutRequest) (string, error)
	processWebhookFunc             func(ctx context.Context, payload []byte, sigHeader string) error
}

func (m *mockStripeService) CreateAccountAndOnboarding(ctx context.Context, projectID string) (string, error) {
	if m.createAccountAndOnboardingFunc != nil {
		return m.createAccountAndOnboardingFunc(ctx, projectID)
	}
	return "https://connect.stripe.com/setup/mock", nil
}
func (m *mockStripeService) CompleteOnboarding(ctx context.Context, projectID string) error {
	if m.completeOnboardingFunc != nil {
		return m.completeOnboardingFunc(ctx, projectID)
	}
	return nil
}
func (m *mockStripeService) RefreshOnboarding(ctx context.Context, projectID string) (string, error) {
	if m.refreshOnboardingFunc != nil {
		return m.refreshOnboardingFunc(ctx, projectID)
	}
	return "https://connect.stripe.com/setup/refresh", nil
}
func (m *mockStripeService) CreateCheckout(ctx context.Context, req service.CheckoutRequest) (string, error) {
	if m.createCheckoutFunc != nil {
		return m.createCheckoutFunc(ctx, req)
	}
	return "https://checkout.stripe.com/test", nil
}
func (m *mockStripeService) ProcessWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if m.processWebhookFunc != nil {
		return m.processWebhookFunc(ctx, payload, sigHeader)
	}
	return nil
}

// mockStripeSessionValidator implements auth.SessionValidator for StripeHandler tests
type mockStripeSessionValidator struct {
	validateFunc func(ctx context.Context, token string) (string, error)
}

func (m *mockStripeSessionValidator) ValidateSession(ctx context.Context, token string) (string, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return "", errors.New("invalid")
}

// ---------------------------------------------------------------------------
// GET /api/stripe/onboarding/return
// ---------------------------------------------------------------------------

func TestStripeHandler_OnboardingReturn_MissingProjectID(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/return", nil)
	rec := httptest.NewRecorder()
	h.OnboardingReturn(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project_id, got %d", rec.Code)
	}
}

func TestStripeHandler_OnboardingReturn_Success(t *testing.T) {
	var capturedProjectID string
	mock := &mockStripeService{
		completeOnboardingFunc: func(_ context.Context, projectID string) error {
			capturedProjectID = projectID
			return nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/return?project_id=proj-1", nil)
	rec := httptest.NewRecorder()
	h.OnboardingReturn(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedProjectID != "proj-1" {
		t.Errorf("expected projectID=proj-1, got %q", capturedProjectID)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "stripe_connected=1") {
		t.Errorf("expected redirect with stripe_connected=1, got %q", loc)
	}
}

func TestStripeHandler_OnboardingReturn_NotYetComplete(t *testing.T) {
	mock := &mockStripeService{
		completeOnboardingFunc: func(_ context.Context, _ string) error {
			return errors.New("stripe: onboarding not yet complete")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/return?project_id=proj-1", nil)
	rec := httptest.NewRecorder()
	h.OnboardingReturn(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "stripe_pending=1") {
		t.Errorf("expected redirect with stripe_pending=1, got %q", loc)
	}
}

func TestStripeHandler_OnboardingReturn_ServiceError(t *testing.T) {
	mock := &mockStripeService{
		completeOnboardingFunc: func(_ context.Context, _ string) error {
			return errors.New("stripe error")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/return?project_id=proj-1", nil)
	rec := httptest.NewRecorder()
	h.OnboardingReturn(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 (redirect to error page), got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "stripe_error=1") {
		t.Errorf("expected redirect with stripe_error=1, got %q", loc)
	}
}

// ---------------------------------------------------------------------------
// GET /api/stripe/onboarding/refresh
// ---------------------------------------------------------------------------

func TestStripeHandler_OnboardingRefresh_MissingProjectID(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/refresh", nil)
	rec := httptest.NewRecorder()
	h.OnboardingRefresh(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project_id, got %d", rec.Code)
	}
}

func TestStripeHandler_OnboardingRefresh_Success(t *testing.T) {
	mock := &mockStripeService{
		refreshOnboardingFunc: func(_ context.Context, projectID string) (string, error) {
			return "https://connect.stripe.com/setup/new-link", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/refresh?project_id=proj-1", nil)
	rec := httptest.NewRecorder()
	h.OnboardingRefresh(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "https://connect.stripe.com/setup/new-link" {
		t.Errorf("expected redirect to onboarding URL, got %q", loc)
	}
}

func TestStripeHandler_OnboardingRefresh_ServiceError(t *testing.T) {
	mock := &mockStripeService{
		refreshOnboardingFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("stripe error")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/onboarding/refresh?project_id=proj-1", nil)
	rec := httptest.NewRecorder()
	h.OnboardingRefresh(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 (redirect to error page), got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "stripe_error=1") {
		t.Errorf("expected redirect with stripe_error=1, got %q", loc)
	}
}

// ---------------------------------------------------------------------------
// POST /api/donations/checkout
// ---------------------------------------------------------------------------

func TestStripeHandler_Checkout_Success(t *testing.T) {
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			if req.ProjectID != "proj-1" || req.Amount != 1000 {
				t.Errorf("unexpected checkout req: %+v", req)
			}
			return "https://checkout.stripe.com/test-session", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "checkout.stripe.com") {
		t.Errorf("expected checkout URL in response, got: %s", rec.Body.String())
	}
}

func TestStripeHandler_Checkout_DonorToken_PassedThrough(t *testing.T) {
	var capturedReq service.CheckoutRequest
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			capturedReq = req
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil) // no session validator

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":500,"currency":"jpy","donor_token":"tok_abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedReq.DonorType != "token" {
		t.Errorf("expected DonorType=token, got %q", capturedReq.DonorType)
	}
	if capturedReq.DonorID != "tok_abc" {
		t.Errorf("expected DonorID=tok_abc, got %q", capturedReq.DonorID)
	}
}

func TestStripeHandler_Checkout_AnonymousDonor_SetsDonorTokenCookie(t *testing.T) {
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			if req.DonorType != "token" {
				t.Errorf("expected DonorType=token, got %q", req.DonorType)
			}
			if req.DonorID == "" {
				t.Error("expected auto-generated DonorID, got empty")
			}
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	// Check donor_token cookie
	var donorCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "donor_token" {
			donorCookie = c
			break
		}
	}
	if donorCookie == nil {
		t.Fatal("expected donor_token cookie to be set")
	}
	if !donorCookie.HttpOnly {
		t.Error("expected donor_token cookie to be HttpOnly")
	}
	if donorCookie.Value == "" {
		t.Error("expected donor_token cookie value to be non-empty")
	}
}

func TestStripeHandler_Checkout_LoggedInUser_NoDonorTokenCookie(t *testing.T) {
	sv := &mockStripeSessionValidator{
		validateFunc: func(_ context.Context, token string) (string, error) {
			if token == "valid-session" {
				return "user-1", nil
			}
			return "", errors.New("invalid")
		},
	}

	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			if req.DonorType != "user" {
				t.Errorf("expected DonorType=user, got %q", req.DonorType)
			}
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", sv)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "valid-session"})
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == "donor_token" {
			t.Error("expected no donor_token cookie for logged-in user")
		}
	}
}

func TestStripeHandler_Checkout_ExistingDonorToken_PreservedAndSetsCookie(t *testing.T) {
	var capturedReq service.CheckoutRequest
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			capturedReq = req
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":500,"currency":"jpy","donor_token":"existing-token-123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedReq.DonorID != "existing-token-123" {
		t.Errorf("expected DonorID=existing-token-123, got %q", capturedReq.DonorID)
	}

	var donorCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "donor_token" {
			donorCookie = c
			break
		}
	}
	if donorCookie == nil {
		t.Fatal("expected donor_token cookie to be set")
	}
	if donorCookie.Value != "existing-token-123" {
		t.Errorf("expected donor_token cookie value=existing-token-123, got %q", donorCookie.Value)
	}
}

// ---------------------------------------------------------------------------
// Recurring donation requires login (#21)
// ---------------------------------------------------------------------------

func TestStripeHandler_Checkout_RecurringRequiresLogin(t *testing.T) {
	// Anonymous donor (no session cookie) with is_recurring=true → 400
	serviceCalled := false
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, _ service.CheckoutRequest) (string, error) {
			serviceCalled = true
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy","is_recurring":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for anonymous recurring donation, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if serviceCalled {
		t.Error("expected service NOT to be called for anonymous recurring donation")
	}
	if !strings.Contains(rec.Body.String(), "recurring_requires_login") {
		t.Errorf("expected error containing 'recurring_requires_login', got: %s", rec.Body.String())
	}
}

func TestStripeHandler_Checkout_RecurringAllowedForLoggedInUser(t *testing.T) {
	// Logged-in user with is_recurring=true → 200
	sv := &mockStripeSessionValidator{
		validateFunc: func(_ context.Context, token string) (string, error) {
			if token == "valid-session" {
				return "user-1", nil
			}
			return "", errors.New("invalid")
		},
	}
	var capturedReq service.CheckoutRequest
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			capturedReq = req
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", sv)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy","is_recurring":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "valid-session"})
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for logged-in recurring, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !capturedReq.IsRecurring {
		t.Error("expected IsRecurring=true in checkout request")
	}
	if capturedReq.DonorType != "user" {
		t.Errorf("expected DonorType=user, got %q", capturedReq.DonorType)
	}
}

func TestStripeHandler_Checkout_OneTimeAllowedForAnonymous(t *testing.T) {
	// Anonymous with is_recurring=false → 200 (regression: one-time should still work)
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, req service.CheckoutRequest) (string, error) {
			if req.IsRecurring {
				t.Error("expected IsRecurring=false")
			}
			return "https://checkout.stripe.com/test", nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy","is_recurring":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for anonymous one-time donation, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestStripeHandler_Checkout_InvalidJSON(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout",
		bytes.NewBufferString(`{bad}`))
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStripeHandler_Checkout_ProjectRequired(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout",
		bytes.NewBufferString(`{"amount":1000}`))
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project_id, got %d", rec.Code)
	}
}

func TestStripeHandler_Checkout_ServiceError(t *testing.T) {
	mock := &mockStripeService{
		createCheckoutFunc: func(_ context.Context, _ service.CheckoutRequest) (string, error) {
			return "", errors.New("project stripe account not connected")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	body := bytes.NewBufferString(`{"project_id":"proj-1","amount":1000,"currency":"jpy"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/donations/checkout", body)
	rec := httptest.NewRecorder()
	h.Checkout(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/webhooks/stripe
// ---------------------------------------------------------------------------

func TestStripeHandler_Webhook_MissingSignature(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe",
		bytes.NewBufferString(`{"type":"payment_intent.succeeded"}`))
	// No Stripe-Signature header
	rec := httptest.NewRecorder()
	h.Webhook(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing signature, got %d", rec.Code)
	}
}

func TestStripeHandler_Webhook_InvalidSignature(t *testing.T) {
	mock := &mockStripeService{
		processWebhookFunc: func(_ context.Context, _ []byte, _ string) error {
			return errors.New("signature verification failed")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe",
		bytes.NewBufferString(`{"type":"x"}`))
	req.Header.Set("Stripe-Signature", "t=123,v1=invalid")
	rec := httptest.NewRecorder()
	h.Webhook(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid signature, got %d", rec.Code)
	}
}

func TestStripeHandler_Webhook_Success(t *testing.T) {
	var capturedPayload []byte
	mock := &mockStripeService{
		processWebhookFunc: func(_ context.Context, payload []byte, _ string) error {
			capturedPayload = payload
			return nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)

	body := `{"type":"payment_intent.succeeded","id":"pi_test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe",
		bytes.NewBufferString(body))
	req.Header.Set("Stripe-Signature", "t=123,v1=valid")
	rec := httptest.NewRecorder()
	h.Webhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if string(capturedPayload) != body {
		t.Errorf("unexpected payload: %q", string(capturedPayload))
	}
}
