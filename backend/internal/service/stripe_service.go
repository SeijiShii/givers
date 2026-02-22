package service

import (
	"context"
	"errors"
	"fmt"

	pkgstripe "github.com/givers/backend/pkg/stripe"
)

// CheckoutRequest は POST /api/donations/checkout のリクエスト
type CheckoutRequest struct {
	ProjectID   string
	Amount      int
	Currency    string
	IsRecurring bool
	Message     string
	Locale      string
	FrontendURL string
}

// StripeProjectRepo は StripeService が必要とするプロジェクト操作のミニマムインターフェース
type StripeProjectRepo interface {
	GetStripeAccountID(ctx context.Context, projectID string) (string, error)
	UpdateStripeConnect(ctx context.Context, projectID, stripeAccountID string) error
}

// StripeService は Stripe 連携のビジネスロジック
type StripeService interface {
	// GenerateConnectURL は Stripe Connect OAuth URL を生成する（API コールなし）
	GenerateConnectURL(projectID string) string
	// CompleteConnect は OAuth code を交換して stripe_account_id を保存する
	CompleteConnect(ctx context.Context, code, projectID string) error
	// CreateCheckout は Stripe Checkout Session を作成し URL を返す
	CreateCheckout(ctx context.Context, req CheckoutRequest) (string, error)
	// ProcessWebhook は Webhook のシグネチャを検証してイベントを処理する
	ProcessWebhook(ctx context.Context, payload []byte, sigHeader string) error
}

// StripeServiceImpl は StripeService の実装
type StripeServiceImpl struct {
	client      pkgstripe.Client
	projectRepo StripeProjectRepo
	frontendURL string
}

// NewStripeService は StripeServiceImpl を生成する
func NewStripeService(client pkgstripe.Client, projectRepo StripeProjectRepo, frontendURL string) StripeService {
	return &StripeServiceImpl{client: client, projectRepo: projectRepo, frontendURL: frontendURL}
}

// GenerateConnectURL は Stripe Connect OAuth URL を返す
func (s *StripeServiceImpl) GenerateConnectURL(projectID string) string {
	return s.client.GenerateConnectURL(projectID)
}

// CompleteConnect は OAuth code を交換して stripe_account_id を projects テーブルに保存する
func (s *StripeServiceImpl) CompleteConnect(ctx context.Context, code, projectID string) error {
	stripeAccountID, err := s.client.ExchangeConnectCode(ctx, code)
	if err != nil {
		return fmt.Errorf("stripe connect exchange: %w", err)
	}
	return s.projectRepo.UpdateStripeConnect(ctx, projectID, stripeAccountID)
}

// CreateCheckout はプロジェクトの stripe_account_id を取得して Checkout Session を作成する
func (s *StripeServiceImpl) CreateCheckout(ctx context.Context, req CheckoutRequest) (string, error) {
	if req.Amount <= 0 {
		return "", errors.New("amount must be greater than 0")
	}

	stripeAccountID, err := s.projectRepo.GetStripeAccountID(ctx, req.ProjectID)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}
	if stripeAccountID == "" {
		return "", errors.New("project stripe account not connected")
	}

	currency := req.Currency
	if currency == "" {
		currency = "jpy"
	}
	locale := req.Locale
	if locale == "" {
		locale = "ja"
	}

	params := pkgstripe.CheckoutParams{
		StripeAccountID: stripeAccountID,
		ProjectID:       req.ProjectID,
		Amount:          req.Amount,
		Currency:        currency,
		IsRecurring:     req.IsRecurring,
		Message:         req.Message,
		Locale:          locale,
		SuccessURL:      s.frontendURL + "/projects/" + req.ProjectID + "?donated=1",
		CancelURL:       s.frontendURL + "/projects/" + req.ProjectID,
	}
	return s.client.CreateCheckoutSession(ctx, params)
}

// ProcessWebhook は Webhook シグネチャを検証してイベントをログに記録する
func (s *StripeServiceImpl) ProcessWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if err := s.client.VerifyWebhookSignature(payload, sigHeader); err != nil {
		return fmt.Errorf("webhook signature: %w", err)
	}
	_, err := s.client.ParseWebhookEvent(payload)
	return err
}
