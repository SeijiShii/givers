// Package stripe provides a lightweight Stripe API client for GIVErS.
// Uses raw HTTP calls (no SDK) to minimize external dependencies.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CheckoutParams はチェックアウトセッション作成に必要なパラメータ
type CheckoutParams struct {
	StripeAccountID string // acct_... （プロジェクトオーナーの Connect アカウント）
	ProjectID       string
	Amount          int    // 金額（円）
	Currency        string // "jpy"
	IsRecurring     bool
	Message         string
	Locale          string // "ja" | "en" | "auto"
	SuccessURL      string
	CancelURL       string
	DonorType       string // "user" or "token" — metadata として保存
	DonorID         string // user_id or donor_token
}

// WebhookEventObject は payment_intent や subscription の data.object
type WebhookEventObject struct {
	ID       string            `json:"id"`
	Amount   int               `json:"amount"`
	Currency string            `json:"currency"`
	Metadata map[string]string `json:"metadata"`
	// subscription の場合のみ使用
	Plan *struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"plan"`
}

// WebhookEvent は Stripe Webhook のイベント
type WebhookEvent struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data struct {
		Object WebhookEventObject `json:"object"`
	} `json:"data"`
}

// Client は Stripe API クライアントのインターフェース
type Client interface {
	// GenerateConnectURL は Stripe Connect OAuth URL を生成する（API コールなし）
	GenerateConnectURL(projectID string) string
	// ExchangeConnectCode は OAuth code を stripe_account_id に交換する
	ExchangeConnectCode(ctx context.Context, code string) (string, error)
	// CreateCheckoutSession は Stripe Checkout Session を作成し URL を返す
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error)
	// VerifyWebhookSignature は Stripe-Signature ヘッダーを検証する
	VerifyWebhookSignature(payload []byte, sigHeader string) error
	// ParseWebhookEvent は Webhook ペイロードをパースする
	ParseWebhookEvent(payload []byte) (WebhookEvent, error)
	// PauseSubscription は定期課金を一時停止する
	PauseSubscription(ctx context.Context, subscriptionID string) error
	// ResumeSubscription は一時停止中の定期課金を再開する
	ResumeSubscription(ctx context.Context, subscriptionID string) error
	// CancelSubscription は定期課金をキャンセルする
	CancelSubscription(ctx context.Context, subscriptionID string) error
}

// RealClient は Stripe API への raw HTTP クライアント実装
type RealClient struct {
	SecretKey       string
	ConnectClientID string // ca_... （Standard Connect）
	WebhookSecret   string // whsec_...
	httpClient      *http.Client
}

// NewClient は RealClient を生成する
func NewClient(secretKey, connectClientID, webhookSecret string) *RealClient {
	return &RealClient{
		SecretKey:       secretKey,
		ConnectClientID: connectClientID,
		WebhookSecret:   webhookSecret,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

// ErrNotConfigured は Stripe が設定されていない場合のエラー
var ErrNotConfigured = errors.New("stripe: not configured")

// GenerateConnectURL は Stripe Connect Standard の OAuth URL を返す
func (c *RealClient) GenerateConnectURL(projectID string) string {
	if c.ConnectClientID == "" {
		return ""
	}
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", c.ConnectClientID)
	v.Set("scope", "read_write")
	v.Set("state", projectID)
	return "https://connect.stripe.com/oauth/authorize?" + v.Encode()
}

// ExchangeConnectCode は code を stripe_account_id (stripe_user_id) に交換する
func (c *RealClient) ExchangeConnectCode(ctx context.Context, code string) (string, error) {
	if c.SecretKey == "" {
		return "", ErrNotConfigured
	}
	data := url.Values{}
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://connect.stripe.com/oauth/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.SecretKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		StripeUserID string `json:"stripe_user_id"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("stripe connect error: %s — %s", result.Error, result.ErrorDesc)
	}
	return result.StripeUserID, nil
}

// CreateCheckoutSession は Stripe Checkout Session を作成し URL を返す
func (c *RealClient) CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error) {
	if c.SecretKey == "" {
		return "", ErrNotConfigured
	}

	data := url.Values{}
	if params.IsRecurring {
		data.Set("mode", "subscription")
		data.Set("line_items[0][price_data][recurring][interval]", "month")
		data.Set("line_items[0][price_data][product_data][name]", "月次サポート")
	} else {
		data.Set("mode", "payment")
		data.Set("line_items[0][price_data][product_data][name]", "寄付")
	}
	data.Set("line_items[0][price_data][currency]", params.Currency)
	data.Set("line_items[0][price_data][unit_amount]", strconv.Itoa(params.Amount))
	data.Set("line_items[0][quantity]", "1")
	data.Set("success_url", params.SuccessURL)
	data.Set("cancel_url", params.CancelURL)
	if params.Locale != "" {
		data.Set("locale", params.Locale)
	}
	if params.Message != "" {
		data.Set("metadata[message]", params.Message)
	}
	data.Set("metadata[project_id]", params.ProjectID)
	if params.DonorType != "" {
		data.Set("metadata[donor_type]", params.DonorType)
		data.Set("metadata[donor_id]", params.DonorID)
	}
	if params.IsRecurring {
		data.Set("metadata[is_recurring]", "true")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v1/checkout/sessions",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.SecretKey, "")
	if params.StripeAccountID != "" {
		req.Header.Set("Stripe-Account", params.StripeAccountID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var session struct {
		URL   string `json:"url"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}
	if session.Error != nil {
		return "", fmt.Errorf("stripe checkout error: %s", session.Error.Message)
	}
	return session.URL, nil
}

// VerifyWebhookSignature は Stripe-Signature ヘッダーを HMAC-SHA256 で検証する
func (c *RealClient) VerifyWebhookSignature(payload []byte, sigHeader string) error {
	if c.WebhookSecret == "" {
		return ErrNotConfigured
	}

	var timestamp string
	var signatures []string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("stripe: invalid signature header format")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("stripe: invalid timestamp in signature header")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return errors.New("stripe: webhook timestamp too old (replay attack protection)")
	}

	mac := hmac.New(sha256.New, []byte(c.WebhookSecret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return errors.New("stripe: signature verification failed")
}

// ParseWebhookEvent は Webhook ペイロードのイベントタイプと ID をパースする
func (c *RealClient) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return WebhookEvent{}, err
	}
	return event, nil
}

// PauseSubscription は pause_collection を設定してサブスクリプションを一時停止する
func (c *RealClient) PauseSubscription(ctx context.Context, subscriptionID string) error {
	if c.SecretKey == "" {
		return ErrNotConfigured
	}
	data := url.Values{}
	data.Set("pause_collection[behavior]", "void")

	return c.updateSubscription(ctx, subscriptionID, data)
}

// ResumeSubscription は pause_collection を解除してサブスクリプションを再開する
func (c *RealClient) ResumeSubscription(ctx context.Context, subscriptionID string) error {
	if c.SecretKey == "" {
		return ErrNotConfigured
	}
	data := url.Values{}
	data.Set("pause_collection", "")

	return c.updateSubscription(ctx, subscriptionID, data)
}

// CancelSubscription はサブスクリプションをキャンセルする
func (c *RealClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	if c.SecretKey == "" {
		return ErrNotConfigured
	}
	endpoint := fmt.Sprintf("https://api.stripe.com/v1/subscriptions/%s", subscriptionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.SecretKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("stripe cancel subscription: %s", errResp.Error.Message)
	}
	return nil
}

func (c *RealClient) updateSubscription(ctx context.Context, subscriptionID string, data url.Values) error {
	endpoint := fmt.Sprintf("https://api.stripe.com/v1/subscriptions/%s", subscriptionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.SecretKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("stripe update subscription: %s", errResp.Error.Message)
	}
	return nil
}
