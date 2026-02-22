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
)

// ---------------------------------------------------------------------------
// Mock StripeService
// ---------------------------------------------------------------------------

type mockStripeService struct {
	generateConnectURLFunc func(projectID string) string
	completeConnectFunc    func(ctx context.Context, code, projectID string) error
	createCheckoutFunc     func(ctx context.Context, req service.CheckoutRequest) (string, error)
	processWebhookFunc     func(ctx context.Context, payload []byte, sigHeader string) error
}

func (m *mockStripeService) GenerateConnectURL(projectID string) string {
	if m.generateConnectURLFunc != nil {
		return m.generateConnectURLFunc(projectID)
	}
	return ""
}
func (m *mockStripeService) CompleteConnect(ctx context.Context, code, projectID string) error {
	if m.completeConnectFunc != nil {
		return m.completeConnectFunc(ctx, code, projectID)
	}
	return nil
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

// ---------------------------------------------------------------------------
// GET /api/stripe/connect/callback
// ---------------------------------------------------------------------------

func TestStripeHandler_ConnectCallback_MissingCode(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/connect/callback?state=proj-1", nil)
	rec := httptest.NewRecorder()
	h.ConnectCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rec.Code)
	}
}

func TestStripeHandler_ConnectCallback_MissingState(t *testing.T) {
	h := NewStripeHandler(&mockStripeService{}, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/connect/callback?code=auth_123", nil)
	rec := httptest.NewRecorder()
	h.ConnectCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing state, got %d", rec.Code)
	}
}

func TestStripeHandler_ConnectCallback_Success(t *testing.T) {
	var capturedCode, capturedProjectID string
	mock := &mockStripeService{
		completeConnectFunc: func(_ context.Context, code, projectID string) error {
			capturedCode = code
			capturedProjectID = projectID
			return nil
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/connect/callback?code=auth_123&state=proj-1", nil)
	rec := httptest.NewRecorder()
	h.ConnectCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedCode != "auth_123" || capturedProjectID != "proj-1" {
		t.Errorf("expected code=auth_123 state=proj-1, got code=%q state=%q", capturedCode, capturedProjectID)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "proj-1") {
		t.Errorf("expected redirect to contain project ID, got %q", loc)
	}
}

func TestStripeHandler_ConnectCallback_ServiceError(t *testing.T) {
	mock := &mockStripeService{
		completeConnectFunc: func(_ context.Context, _, _ string) error {
			return errors.New("stripe error")
		},
	}
	h := NewStripeHandler(mock, "https://example.com", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/stripe/connect/callback?code=bad&state=proj-1", nil)
	rec := httptest.NewRecorder()
	h.ConnectCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 (redirect to error page), got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error") {
		t.Errorf("expected redirect with error param, got %q", loc)
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
	h := NewStripeHandler(mock, "https://example.com", nil) // no session secret

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
