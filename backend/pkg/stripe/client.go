// Package stripe provides a lightweight Stripe API client for GIVErS.
// Uses raw HTTP calls (no SDK) to minimize external dependencies.
package stripe

import (
	"bytes"
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

// StripeAPIVersion は v2 API で使用する Stripe-Version ヘッダーの値
const StripeAPIVersion = "2025-04-30.basil"

// CreateAccountParams は v2 連結アカウント作成に必要なパラメータ
type CreateAccountParams struct {
	Email       string // 連結アカウントの連絡先メール
	DisplayName string // 表示名
	Country     string // "jp", "us" など
}

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
	// CreateConnectedAccount は v2 API で連結アカウントを作成し acct_... を返す
	CreateConnectedAccount(ctx context.Context, params CreateAccountParams) (string, error)
	// CreateAccountLink はオンボーディング用の Account Link URL を生成する
	CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error)
	// GetAccountOnboarded は連結アカウントのオンボーディング完了状態を返す
	GetAccountOnboarded(ctx context.Context, accountID string) (bool, error)
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
	SecretKey     string
	WebhookSecret string // whsec_...
	httpClient    *http.Client
}

// NewClient は RealClient を生成する
func NewClient(secretKey, webhookSecret string) *RealClient {
	return &RealClient{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ErrNotConfigured は Stripe が設定されていない場合のエラー
var ErrNotConfigured = errors.New("stripe: not configured")

// CreateConnectedAccount は Accounts v2 API で連結アカウントを作成する
func (c *RealClient) CreateConnectedAccount(ctx context.Context, params CreateAccountParams) (string, error) {
	if c.SecretKey == "" {
		return "", ErrNotConfigured
	}

	country := params.Country
	if country == "" {
		country = "jp"
	}

	body := map[string]any{
		"contact_email": params.Email,
		"display_name":  params.DisplayName,
		"dashboard":     "full",
		"identity": map[string]any{
			"country":     country,
			"entity_type": "individual",
		},
		"configuration": map[string]any{
			"merchant": map[string]any{
				"capabilities": map[string]any{
					"card_payments": map[string]any{
						"requested": true,
					},
				},
			},
		},
		"defaults": map[string]any{
			"currency": "jpy",
			"responsibilities": map[string]any{
				"fees_collector":   "stripe",
				"losses_collector": "stripe",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v2/core/accounts",
		bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Stripe-Version", StripeAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("stripe create account: %s", result.Error.Message)
	}
	if result.ID == "" {
		return "", errors.New("stripe create account: empty account ID in response")
	}
	return result.ID, nil
}

// CreateAccountLink は v2 API で Account Link（オンボーディング URL）を作成する
func (c *RealClient) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
	if c.SecretKey == "" {
		return "", ErrNotConfigured
	}

	body := map[string]any{
		"account": accountID,
		"use_case": map[string]any{
			"type": "account_onboarding",
			"account_onboarding": map[string]any{
				"configurations": []string{"merchant"},
				"return_url":     returnURL,
				"refresh_url":    refreshURL,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v2/core/account_links",
		bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Stripe-Version", StripeAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		URL   string `json:"url"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("stripe create account link: %s", result.Error.Message)
	}
	if result.URL == "" {
		return "", errors.New("stripe create account link: empty URL in response")
	}
	return result.URL, nil
}

// GetAccountOnboarded は v2 API で連結アカウントのオンボーディング完了状態を確認する
// currently_due が空の場合に true を返す
func (c *RealClient) GetAccountOnboarded(ctx context.Context, accountID string) (bool, error) {
	if c.SecretKey == "" {
		return false, ErrNotConfigured
	}

	endpoint := fmt.Sprintf("https://api.stripe.com/v2/core/accounts/%s?include=requirements", accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Stripe-Version", StripeAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Requirements *struct {
			CurrentlyDue []string `json:"currently_due"`
		} `json:"requirements"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if result.Error != nil {
		return false, fmt.Errorf("stripe get account: %s", result.Error.Message)
	}
	if result.Requirements == nil {
		return true, nil
	}
	return len(result.Requirements.CurrentlyDue) == 0, nil
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
