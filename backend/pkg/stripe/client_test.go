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

// --- CreateCustomer tests ---

func TestRealClient_CreateCustomer_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/customers" {
			t.Errorf("expected /v1/customers, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if got := string(body); got != "email=test%40example.com" {
			t.Errorf("unexpected body: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cus_test123"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	id, err := c.CreateCustomer(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cus_test123" {
		t.Errorf("expected cus_test123, got %q", id)
	}
}

func TestRealClient_CreateCustomer_NotConfigured(t *testing.T) {
	c := NewClient("", "")
	_, err := c.CreateCustomer(context.Background(), "test@example.com")
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}

// --- CreateOffSessionPaymentIntent tests ---

func TestRealClient_CreateOffSessionPaymentIntent_Succeeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/payment_intents" {
			t.Errorf("expected /v1/payment_intents, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Stripe-Account"); got != "acct_owner1" {
			t.Errorf("expected Stripe-Account=acct_owner1, got %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		for _, expected := range []string{"amount=1000", "currency=jpy", "customer=cus_abc", "payment_method=pm_xyz", "confirm=true", "off_session=true"} {
			if !contains(bodyStr, expected) {
				t.Errorf("body missing %q: %s", expected, bodyStr)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"pi_test","status":"succeeded","client_secret":""}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	result, err := c.CreateOffSessionPaymentIntent(context.Background(), OffSessionPaymentParams{
		CustomerID:      "cus_abc",
		PaymentMethodID: "pm_xyz",
		Amount:          1000,
		Currency:        "jpy",
		StripeAccountID: "acct_owner1",
		Metadata:        map[string]string{"project_id": "proj-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentIntentID != "pi_test" {
		t.Errorf("expected pi_test, got %q", result.PaymentIntentID)
	}
	if result.Status != "succeeded" {
		t.Errorf("expected succeeded, got %q", result.Status)
	}
}

func TestRealClient_CreateOffSessionPaymentIntent_RequiresAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"pi_3ds","status":"requires_action","client_secret":"pi_3ds_secret_abc"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	result, err := c.CreateOffSessionPaymentIntent(context.Background(), OffSessionPaymentParams{
		CustomerID:      "cus_abc",
		PaymentMethodID: "pm_xyz",
		Amount:          2000,
		Currency:        "jpy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "requires_action" {
		t.Errorf("expected requires_action, got %q", result.Status)
	}
	if result.ClientSecret != "pi_3ds_secret_abc" {
		t.Errorf("expected client secret, got %q", result.ClientSecret)
	}
}

func TestRealClient_CreateOffSessionPaymentIntent_StripeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(402)
		fmt.Fprint(w, `{"error":{"message":"card expired"}}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	_, err := c.CreateOffSessionPaymentIntent(context.Background(), OffSessionPaymentParams{
		CustomerID:      "cus_abc",
		PaymentMethodID: "pm_xyz",
		Amount:          1000,
		Currency:        "jpy",
	})
	if err == nil {
		t.Fatal("expected error for card expired")
	}
}

func TestRealClient_CreateOffSessionPaymentIntent_NotConfigured(t *testing.T) {
	c := NewClient("", "")
	_, err := c.CreateOffSessionPaymentIntent(context.Background(), OffSessionPaymentParams{})
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}

// --- ListPaymentMethods tests ---

func TestRealClient_ListPaymentMethods_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/payment_methods" {
			t.Errorf("expected /v1/payment_methods, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("customer"); got != "cus_abc" {
			t.Errorf("expected customer=cus_abc, got %q", got)
		}
		if got := r.URL.Query().Get("type"); got != "card" {
			t.Errorf("expected type=card, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"pm_1","card":{"brand":"visa","last4":"4242","exp_month":12,"exp_year":2027}},{"id":"pm_2","card":{"brand":"mastercard","last4":"5555","exp_month":6,"exp_year":2028}}]}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	methods, err := c.ListPaymentMethods(context.Background(), "cus_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(methods))
	}
	if methods[0].ID != "pm_1" || methods[0].Brand != "visa" || methods[0].Last4 != "4242" {
		t.Errorf("unexpected first method: %+v", methods[0])
	}
}

func TestRealClient_ListPaymentMethods_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "sk_test")
	methods, err := c.ListPaymentMethods(context.Background(), "cus_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 0 {
		t.Errorf("expected empty list, got %d", len(methods))
	}
}

func TestRealClient_ListPaymentMethods_NotConfigured(t *testing.T) {
	c := NewClient("", "")
	_, err := c.ListPaymentMethods(context.Background(), "cus_abc")
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}

// contains checks if s contains substr (helper for form-encoded body checks)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// newTestClient creates a RealClient that points to a test server instead of Stripe.
func newTestClient(baseURL, secretKey string) *RealClient {
	c := NewClient(secretKey, "")
	c.baseURL = baseURL
	return c
}
