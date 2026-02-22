package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/givers/backend/internal/model"
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
	DonorType   string // "user" or "token"
	DonorID     string // user_id or donor_token
}

// StripeProjectRepo は StripeService が必要とするプロジェクト操作のミニマムインターフェース
type StripeProjectRepo interface {
	GetStripeAccountID(ctx context.Context, projectID string) (string, error)
	UpdateStripeConnect(ctx context.Context, projectID, stripeAccountID string) error
}

// StripeDonationRepo は Webhook イベントで寄付レコードを作成するためのミニマムインターフェース
type StripeDonationRepo interface {
	Create(ctx context.Context, d *model.Donation) error
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
	client       pkgstripe.Client
	projectRepo  StripeProjectRepo
	donationRepo StripeDonationRepo
	frontendURL  string
}

// NewStripeService は StripeServiceImpl を生成する
func NewStripeService(client pkgstripe.Client, projectRepo StripeProjectRepo, donationRepo StripeDonationRepo, frontendURL string) StripeService {
	return &StripeServiceImpl{
		client:       client,
		projectRepo:  projectRepo,
		donationRepo: donationRepo,
		frontendURL:  frontendURL,
	}
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
		DonorType:       req.DonorType,
		DonorID:         req.DonorID,
	}
	return s.client.CreateCheckoutSession(ctx, params)
}

// ProcessWebhook は Webhook シグネチャを検証してイベントを処理する
func (s *StripeServiceImpl) ProcessWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if err := s.client.VerifyWebhookSignature(payload, sigHeader); err != nil {
		return fmt.Errorf("webhook signature: %w", err)
	}
	event, err := s.client.ParseWebhookEvent(payload)
	if err != nil {
		return err
	}
	switch event.Type {
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event)
	}
	return nil
}

func (s *StripeServiceImpl) handlePaymentIntentSucceeded(ctx context.Context, event pkgstripe.WebhookEvent) error {
	obj := event.Data.Object
	projectID := obj.Metadata["project_id"]
	if projectID == "" {
		return errors.New("stripe webhook: payment_intent.succeeded missing project_id in metadata")
	}

	donorType := obj.Metadata["donor_type"]
	if donorType == "" {
		donorType = "token"
	}
	donorID := obj.Metadata["donor_id"]

	currency := obj.Currency
	if currency == "" {
		currency = "jpy"
	}

	d := &model.Donation{
		ProjectID:       projectID,
		DonorType:       donorType,
		DonorID:         donorID,
		Amount:          obj.Amount,
		Currency:        currency,
		Message:         obj.Metadata["message"],
		IsRecurring:     obj.Metadata["is_recurring"] == "true",
		StripePaymentID: obj.ID,
	}
	return s.donationRepo.Create(ctx, d)
}
