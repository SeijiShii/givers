package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestRealClient_VerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test_secret"
	c := NewClient("sk_test", "ca_test", secret)

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
	c := NewClient("sk_test", "ca_test", "whsec_test_secret")
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sigHeader := fmt.Sprintf("t=%s,v1=wrongsignature", ts)

	if err := c.VerifyWebhookSignature([]byte(`{}`), sigHeader); err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestRealClient_VerifyWebhookSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_test_secret"
	c := NewClient("sk_test", "ca_test", secret)

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
	c := NewClient("sk_test", "ca_test", "") // empty webhook secret
	if err := c.VerifyWebhookSignature([]byte(`{}`), "t=123,v1=abc"); err == nil {
		t.Error("expected error when not configured")
	}
}

func TestRealClient_GenerateConnectURL_WithClientID(t *testing.T) {
	c := NewClient("sk_test", "ca_test_client_id", "whsec")
	url := c.GenerateConnectURL("project-123")
	if url == "" {
		t.Error("expected non-empty URL")
	}
	if !contains(url, "client_id=ca_test_client_id") {
		t.Errorf("expected client_id in URL, got: %s", url)
	}
	if !contains(url, "state=project-123") {
		t.Errorf("expected state=project-123 in URL, got: %s", url)
	}
}

func TestRealClient_GenerateConnectURL_EmptyWhenNotConfigured(t *testing.T) {
	c := NewClient("sk_test", "", "whsec") // empty ConnectClientID
	url := c.GenerateConnectURL("project-123")
	if url != "" {
		t.Errorf("expected empty URL, got: %s", url)
	}
}

func TestRealClient_ParseWebhookEvent(t *testing.T) {
	c := NewClient("", "", "")
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
