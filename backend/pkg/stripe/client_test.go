package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRealClient_VerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test_secret"
	c := NewClient("sk_test", secret)

	ts := fmt.Sprintf("%d", time.Now().Unix())
	payload := []byte(`{"type":"payment_intent.succeeded"}`)

	// Compute valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(payload)))
	sig := hex.EncodeToString(mac.Sum(nil))
	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	if err := c.VerifyWebhookSignature(payload, sigHeader); err != nil {
		t.Fatalf("expected valid signature to pass, got: %v", err)
	}
}

func TestRealClient_VerifyWebhookSignature_Invalid(t *testing.T) {
	c := NewClient("sk_test", "whsec_test_secret")
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sigHeader := fmt.Sprintf("t=%s,v1=wrongsignature", ts)

	if err := c.VerifyWebhookSignature([]byte(`{}`), sigHeader); err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestRealClient_VerifyWebhookSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_test_secret"
	c := NewClient("sk_test", secret)

	// 10 minutes old
	ts := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	payload := []byte(`{}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(payload)))
	sig := hex.EncodeToString(mac.Sum(nil))
	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	if err := c.VerifyWebhookSignature(payload, sigHeader); err == nil {
		t.Error("expected error for expired timestamp")
	}
}

func TestRealClient_VerifyWebhookSignature_NotConfigured(t *testing.T) {
	c := NewClient("sk_test", "") // empty webhook secret
	if err := c.VerifyWebhookSignature([]byte(`{}`), "t=123,v1=abc"); err == nil {
		t.Error("expected error when not configured")
	}
}

func TestRealClient_ParseWebhookEvent(t *testing.T) {
	c := NewClient("", "")
	payload := []byte(`{"type":"customer.subscription.created","id":"sub_test"}`)
	event, err := c.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "customer.subscription.created" {
		t.Errorf("expected type=customer.subscription.created, got %q", event.Type)
	}
	if event.ID != "sub_test" {
		t.Errorf("expected id=sub_test, got %q", event.ID)
	}
}

func TestRealClient_ParseWebhookEvent_PaymentIntentSucceeded(t *testing.T) {
	c := NewClient("", "")
	payload := []byte(`{
		"type":"payment_intent.succeeded",
		"id":"evt_test",
		"data":{"object":{
			"id":"pi_test",
			"amount":1000,
			"currency":"jpy",
			"metadata":{"project_id":"proj-1","donor_type":"user","donor_id":"user-1","message":"hello"}
		}}
	}`)
	event, err := c.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Data.Object.Amount != 1000 {
		t.Errorf("expected amount=1000, got %d", event.Data.Object.Amount)
	}
	if event.Data.Object.Metadata["project_id"] != "proj-1" {
		t.Errorf("expected project_id=proj-1, got %q", event.Data.Object.Metadata["project_id"])
	}
	if event.Data.Object.Metadata["donor_type"] != "user" {
		t.Errorf("expected donor_type=user, got %q", event.Data.Object.Metadata["donor_type"])
	}
}

// --- CreateRefund tests ---

func TestRealClient_CreateRefund_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/refunds" {
			t.Errorf("expected /v1/refunds, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		if got := string(body); got != "payment_intent=pi_test123" {
			t.Errorf("unexpected body: %s", got)
		}

		// Verify Stripe-Account header for Connected Account
		if got := r.Header.Get("Stripe-Account"); got != "acct_owner1" {
			t.Errorf("expected Stripe-Account=acct_owner1, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"re_refund123","status":"succeeded"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	refundID, err := c.CreateRefund(context.Background(), "pi_test123", "acct_owner1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refundID != "re_refund123" {
		t.Errorf("expected re_refund123, got %q", refundID)
	}
}

func TestRealClient_CreateRefund_PlatformDirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Stripe-Account header for platform-direct
		if got := r.Header.Get("Stripe-Account"); got != "" {
			t.Errorf("expected empty Stripe-Account, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"re_platform1"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	refundID, err := c.CreateRefund(context.Background(), "pi_test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refundID != "re_platform1" {
		t.Errorf("expected re_platform1, got %q", refundID)
	}
}

func TestRealClient_CreateRefund_StripeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":{"message":"charge already refunded"}}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	_, err := c.CreateRefund(context.Background(), "pi_test", "")
	if err == nil {
		t.Fatal("expected error for Stripe error response")
	}
}

func TestRealClient_CreateRefund_NotConfigured(t *testing.T) {
	c := NewClient("", "")
	_, err := c.CreateRefund(context.Background(), "pi_test", "")
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}

// --- GetInvoicePaymentIntent tests ---

func TestRealClient_GetInvoicePaymentIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/invoices/in_test456" {
			t.Errorf("expected /v1/invoices/in_test456, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Stripe-Account"); got != "acct_connected" {
			t.Errorf("expected Stripe-Account=acct_connected, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"in_test456","payment_intent":"pi_from_invoice"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	piID, err := c.GetInvoicePaymentIntent(context.Background(), "in_test456", "acct_connected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if piID != "pi_from_invoice" {
		t.Errorf("expected pi_from_invoice, got %q", piID)
	}
}

func TestRealClient_GetInvoicePaymentIntent_NoPaymentIntent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"in_test","payment_intent":""}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	_, err := c.GetInvoicePaymentIntent(context.Background(), "in_test", "")
	if err == nil {
		t.Fatal("expected error for empty payment_intent")
	}
}

func TestRealClient_GetInvoicePaymentIntent_NotConfigured(t *testing.T) {
	c := NewClient("", "")
	_, err := c.GetInvoicePaymentIntent(context.Background(), "in_test", "")
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}

// --- ParseWebhookEvent: PaymentIntent field on invoice ---

func TestRealClient_ParseWebhookEvent_InvoicePaymentIntent(t *testing.T) {
	c := NewClient("", "")
	payload := []byte(`{
		"type":"invoice.payment_succeeded",
		"id":"evt_inv",
		"data":{"object":{
			"id":"in_test",
			"amount":2000,
			"currency":"jpy",
			"subscription":"sub_abc",
			"payment_intent":"pi_from_inv"
		}}
	}`)
	event, err := c.ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Data.Object.PaymentIntent != "pi_from_inv" {
		t.Errorf("expected payment_intent=pi_from_inv, got %q", event.Data.Object.PaymentIntent)
	}
}

// newTestClient creates a RealClient that points to a test server instead of Stripe.
func newTestClient(baseURL, secretKey string) *RealClient {
	c := NewClient(secretKey, "")
	c.baseURL = baseURL
	return c
}
