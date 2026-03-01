package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
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
	SaveStripeAccountID(ctx context.Context, projectID, stripeAccountID string) error
	ActivateProject(ctx context.Context, projectID string) error
}

// StripeDonationRepo は Webhook イベントで寄付レコードを操作するためのミニマムインターフェース
type StripeDonationRepo interface {
	Create(ctx context.Context, d *model.Donation) error
	DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error
	GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error)
	Patch(ctx context.Context, id string, patch model.DonationPatch) error
}

// StripeActivityRecorder は寄付確定時にアクティビティを記録するためのミニマムインターフェース
type StripeActivityRecorder interface {
	Insert(ctx context.Context, a *model.ActivityItem) error
}

// StripeMilestoneNotifier は寄付確定時にマイルストーンチェックを行うインターフェース
type StripeMilestoneNotifier interface {
	NotifyDonation(ctx context.Context, projectID string) error
}

// StripeService は Stripe 連携のビジネスロジック
type StripeService interface {
	// CreateAccountAndOnboarding は v2 API でアカウント作成 → Account Link URL を返す
	CreateAccountAndOnboarding(ctx context.Context, projectID string) (onboardingURL string, err error)
	// CompleteOnboarding はオンボーディング完了を確認し、完了なら status='active' にする
	CompleteOnboarding(ctx context.Context, projectID string) error
	// RefreshOnboarding は新しい Account Link URL を生成する（再オンボーディング用）
	RefreshOnboarding(ctx context.Context, projectID string) (onboardingURL string, err error)
	// CreateCheckout は Stripe Checkout Session を作成し URL を返す
	CreateCheckout(ctx context.Context, req CheckoutRequest) (string, error)
	// ProcessWebhook は Webhook のシグネチャを検証してイベントを処理する
	ProcessWebhook(ctx context.Context, payload []byte, sigHeader string) error
}

// StripeServiceImpl は StripeService の実装
type StripeServiceImpl struct {
	client             pkgstripe.Client
	projectRepo        StripeProjectRepo
	donationRepo       StripeDonationRepo
	activityRecorder   StripeActivityRecorder  // optional, nil = skip
	milestoneNotifier  StripeMilestoneNotifier // optional, nil = skip
	frontendURL        string
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

// NewStripeServiceWithActivity は ActivityRecorder + MilestoneNotifier 付きの StripeServiceImpl を生成する
func NewStripeServiceWithActivity(client pkgstripe.Client, projectRepo StripeProjectRepo, donationRepo StripeDonationRepo, frontendURL string, activityRecorder StripeActivityRecorder, milestoneNotifier StripeMilestoneNotifier) StripeService {
	return &StripeServiceImpl{
		client:            client,
		projectRepo:       projectRepo,
		donationRepo:      donationRepo,
		activityRecorder:  activityRecorder,
		milestoneNotifier: milestoneNotifier,
		frontendURL:       frontendURL,
	}
}

// CreateAccountAndOnboarding は v2 API でアカウントを作成し、Account Link URL を返す
func (s *StripeServiceImpl) CreateAccountAndOnboarding(ctx context.Context, projectID string) (string, error) {
	// v2 アカウント作成
	accountID, err := s.client.CreateConnectedAccount(ctx, pkgstripe.CreateAccountParams{
		Country: "jp",
	})
	if err != nil {
		return "", fmt.Errorf("stripe create account: %w", err)
	}

	// stripe_account_id を DB に保存（status は draft のまま）
	if err := s.projectRepo.SaveStripeAccountID(ctx, projectID, accountID); err != nil {
		return "", fmt.Errorf("save stripe account id: %w", err)
	}

	// Account Link 作成
	returnURL := s.frontendURL + "/api/stripe/onboarding/return?project_id=" + projectID
	refreshURL := s.frontendURL + "/api/stripe/onboarding/refresh?project_id=" + projectID
	onboardingURL, err := s.client.CreateAccountLink(ctx, accountID, returnURL, refreshURL)
	if err != nil {
		return "", fmt.Errorf("stripe create account link: %w", err)
	}

	return onboardingURL, nil
}

// CompleteOnboarding はオンボーディング完了を確認し、完了なら status='active' にする
func (s *StripeServiceImpl) CompleteOnboarding(ctx context.Context, projectID string) error {
	accountID, err := s.projectRepo.GetStripeAccountID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get stripe account id: %w", err)
	}
	if accountID == "" {
		return errors.New("stripe: no account linked to project")
	}

	onboarded, err := s.client.GetAccountOnboarded(ctx, accountID)
	if err != nil {
		return fmt.Errorf("stripe get account status: %w", err)
	}
	if !onboarded {
		return errors.New("stripe: onboarding not yet complete")
	}

	return s.projectRepo.ActivateProject(ctx, projectID)
}

// RefreshOnboarding は新しい Account Link URL を生成する
func (s *StripeServiceImpl) RefreshOnboarding(ctx context.Context, projectID string) (string, error) {
	accountID, err := s.projectRepo.GetStripeAccountID(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("get stripe account id: %w", err)
	}
	if accountID == "" {
		return "", errors.New("stripe: no account linked to project")
	}

	returnURL := s.frontendURL + "/api/stripe/onboarding/return?project_id=" + projectID
	refreshURL := s.frontendURL + "/api/stripe/onboarding/refresh?project_id=" + projectID
	return s.client.CreateAccountLink(ctx, accountID, returnURL, refreshURL)
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
	// stripeAccountID が空の場合はプラットフォームアカウントで直接決済
	// （ホストのプロジェクトは Stripe Connect 不要）

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
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.payment_succeeded":
		return s.handleInvoicePaymentSucceeded(ctx, event)
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
	if err := s.donationRepo.Create(ctx, d); err != nil && !errors.Is(err, repository.ErrDuplicate) {
		return err
	}
	s.recordDonationActivity(ctx, projectID, donorID, obj.Amount, obj.Metadata["message"])
	s.notifyMilestone(ctx, projectID)
	return nil
}

func (s *StripeServiceImpl) handleSubscriptionCreated(ctx context.Context, event pkgstripe.WebhookEvent) error {
	obj := event.Data.Object
	projectID := obj.Metadata["project_id"]
	if projectID == "" {
		return errors.New("stripe webhook: customer.subscription.created missing project_id in metadata")
	}

	donorType := obj.Metadata["donor_type"]
	if donorType == "" {
		donorType = "token"
	}
	donorID := obj.Metadata["donor_id"]

	amount := obj.Amount
	currency := obj.Currency
	if obj.Plan != nil {
		amount = obj.Plan.Amount
		currency = obj.Plan.Currency
	}
	if currency == "" {
		currency = "jpy"
	}

	d := &model.Donation{
		ProjectID:            projectID,
		DonorType:            donorType,
		DonorID:              donorID,
		Amount:               amount,
		Currency:             currency,
		Message:              obj.Metadata["message"],
		IsRecurring:          true,
		StripeSubscriptionID: obj.ID,
	}
	if err := s.donationRepo.Create(ctx, d); err != nil && !errors.Is(err, repository.ErrDuplicate) {
		return err
	}
	s.recordDonationActivity(ctx, projectID, donorID, amount, obj.Metadata["message"])
	s.notifyMilestone(ctx, projectID)
	return nil
}

// recordDonationActivity は寄付確定時に activity を記録する（失敗しても無視）
func (s *StripeServiceImpl) recordDonationActivity(ctx context.Context, projectID, donorID string, amount int, message string) {
	if s.activityRecorder == nil {
		return
	}
	var actorName *string
	if donorID != "" {
		actorName = &donorID
	}
	_ = s.activityRecorder.Insert(ctx, &model.ActivityItem{
		Type:      "donation",
		ProjectID: projectID,
		ActorName: actorName,
		Amount:    &amount,
		Message:   message,
	})
}

// notifyMilestone は寄付確定時にマイルストーンチェックを実行する（失敗しても無視）
func (s *StripeServiceImpl) notifyMilestone(ctx context.Context, projectID string) {
	if s.milestoneNotifier == nil {
		return
	}
	_ = s.milestoneNotifier.NotifyDonation(ctx, projectID)
}

func (s *StripeServiceImpl) handleSubscriptionDeleted(ctx context.Context, event pkgstripe.WebhookEvent) error {
	subscriptionID := event.Data.Object.ID
	if subscriptionID == "" {
		return errors.New("stripe webhook: customer.subscription.deleted missing subscription ID")
	}
	return s.donationRepo.DeleteByStripeSubscriptionID(ctx, subscriptionID)
}

// handleInvoicePaymentSucceeded はサブスクの請求成功時に next_billing_message をアクティビティに記録してクリアする (#19)
func (s *StripeServiceImpl) handleInvoicePaymentSucceeded(ctx context.Context, event pkgstripe.WebhookEvent) error {
	obj := event.Data.Object
	subscriptionID := obj.Subscription
	if subscriptionID == "" {
		return nil // one-time invoice, skip
	}

	d, err := s.donationRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil || d == nil {
		return nil // donation not found, skip silently
	}

	if d.NextBillingMessage == "" {
		return nil // no message to record
	}

	// Record activity with the message
	s.recordDonationActivity(ctx, d.ProjectID, d.DonorID, d.Amount, d.NextBillingMessage)

	// Clear next_billing_message
	empty := ""
	_ = s.donationRepo.Patch(ctx, d.ID, model.DonationPatch{NextBillingMessage: &empty})

	return nil
}
