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

// QuickDonateRequest はワンクリック再寄付のリクエスト
type QuickDonateRequest struct {
	UserID     string // ログインユーザーの ID
	DonationID string // 元の寄付 ID
}

// QuickDonateResult はワンクリック再寄付の結果
type QuickDonateResult struct {
	Status       string `json:"status"`        // "succeeded" or "requires_action"
	ClientSecret string `json:"client_secret,omitempty"` // requires_action 時のみ
	DonationID   string `json:"donation_id,omitempty"`   // succeeded 時のみ
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
	GetByID(ctx context.Context, id string) (*model.Donation, error)
	DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error
	GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error)
	Patch(ctx context.Context, id string, patch model.DonationPatch) error
}

// StripeSubscriptionRepo は Webhook イベントでサブスクリプションを操作するためのインターフェース
type StripeSubscriptionRepo interface {
	Create(ctx context.Context, s *model.Subscription) error
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*model.Subscription, error)
	DeleteByStripeSubscriptionID(ctx context.Context, stripeSubID string) error
	Patch(ctx context.Context, id string, patch model.SubscriptionPatch) error
}

// StripeActivityRecorder は寄付確定時にアクティビティを記録するためのミニマムインターフェース
type StripeActivityRecorder interface {
	Insert(ctx context.Context, a *model.ActivityItem) error
}

// StripeMilestoneNotifier は寄付確定時にマイルストーンチェックを行うインターフェース
type StripeMilestoneNotifier interface {
	NotifyDonation(ctx context.Context, projectID string) error
}

// StripeUserRepo は StripeService が必要とするユーザー操作のミニマムインターフェース
type StripeUserRepo interface {
	SaveStripeCustomerID(ctx context.Context, userID, customerID string) error
	GetStripeCustomerID(ctx context.Context, userID string) (string, error)
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
	// QuickDonate は保存済み PaymentMethod で過去の寄付と同額を即時決済する
	QuickDonate(ctx context.Context, req QuickDonateRequest) (*QuickDonateResult, error)
}

// StripeServiceImpl は StripeService の実装
type StripeServiceImpl struct {
	client             pkgstripe.Client
	projectRepo        StripeProjectRepo
	donationRepo       StripeDonationRepo
	subscriptionRepo   StripeSubscriptionRepo  // optional, nil = fallback to donationRepo
	userRepo           StripeUserRepo          // optional, nil = skip customer creation
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

// NewStripeServiceFull は全依存関係付きの StripeServiceImpl を生成する
func NewStripeServiceFull(client pkgstripe.Client, projectRepo StripeProjectRepo, donationRepo StripeDonationRepo, subscriptionRepo StripeSubscriptionRepo, userRepo StripeUserRepo, frontendURL string, activityRecorder StripeActivityRecorder, milestoneNotifier StripeMilestoneNotifier) StripeService {
	return &StripeServiceImpl{
		client:            client,
		projectRepo:       projectRepo,
		donationRepo:      donationRepo,
		subscriptionRepo:  subscriptionRepo,
		userRepo:          userRepo,
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

	// ログインユーザーの場合、Stripe Customer を解決（なければ作成）
	var customerID string
	if req.DonorType == "user" && req.DonorID != "" && s.userRepo != nil {
		cid, err := s.userRepo.GetStripeCustomerID(ctx, req.DonorID)
		if err == nil && cid != "" {
			customerID = cid
		} else if err == nil && cid == "" {
			// Customer 未作成 → 新規作成して保存
			newCID, err := s.client.CreateCustomer(ctx, "")
			if err == nil && newCID != "" {
				_ = s.userRepo.SaveStripeCustomerID(ctx, req.DonorID, newCID)
				customerID = newCID
			}
		}
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
		CustomerID:      customerID,
	}
	return s.client.CreateCheckoutSession(ctx, params)
}

// QuickDonate は保存済み PaymentMethod で過去の寄付と同額を即時決済する
func (s *StripeServiceImpl) QuickDonate(ctx context.Context, req QuickDonateRequest) (*QuickDonateResult, error) {
	if s.userRepo == nil {
		return nil, errors.New("quick donate: user repository not configured")
	}

	// 元の寄付を取得
	orig, err := s.donationRepo.GetByID(ctx, req.DonationID)
	if err != nil {
		return nil, fmt.Errorf("quick donate: get donation: %w", err)
	}

	// 権限チェック：自分の寄付のみ
	if orig.DonorType != "user" || orig.DonorID != req.UserID {
		return nil, errors.New("quick donate: not your donation")
	}

	// 定期寄付は対象外
	if orig.IsRecurring {
		return nil, errors.New("quick donate: recurring donations not supported")
	}

	// Stripe Customer ID を取得
	customerID, err := s.userRepo.GetStripeCustomerID(ctx, req.UserID)
	if err != nil || customerID == "" {
		return nil, errors.New("quick donate: no saved payment method. please donate via checkout first")
	}

	// PaymentMethod 一覧から最新を取得
	methods, err := s.client.ListPaymentMethods(ctx, customerID)
	if err != nil || len(methods) == 0 {
		return nil, errors.New("quick donate: no saved payment method. please donate via checkout first")
	}

	// プロジェクトの Stripe Account ID を取得
	stripeAccountID, err := s.projectRepo.GetStripeAccountID(ctx, orig.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("quick donate: get project: %w", err)
	}

	currency := orig.Currency
	if currency == "" {
		currency = "jpy"
	}

	// オフセッション決済
	result, err := s.client.CreateOffSessionPaymentIntent(ctx, pkgstripe.OffSessionPaymentParams{
		CustomerID:      customerID,
		PaymentMethodID: methods[0].ID,
		Amount:          orig.Amount,
		Currency:        currency,
		StripeAccountID: stripeAccountID,
		Metadata: map[string]string{
			"project_id": orig.ProjectID,
			"donor_type": "user",
			"donor_id":   req.UserID,
			"source":     "quick_donate",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("quick donate: payment failed: %w", err)
	}

	if result.Status == "succeeded" {
		// 寄付レコードを即作成
		d := &model.Donation{
			ProjectID:       orig.ProjectID,
			DonorType:       "user",
			DonorID:         req.UserID,
			Amount:          orig.Amount,
			Currency:        currency,
			Source:           "quick_donate",
			StripePaymentID: result.PaymentIntentID,
		}
		if createErr := s.donationRepo.Create(ctx, d); createErr != nil && !errors.Is(createErr, repository.ErrDuplicate) {
			return nil, fmt.Errorf("quick donate: create donation: %w", createErr)
		}
		s.recordDonationActivity(ctx, orig.ProjectID, req.UserID, orig.Amount, "")
		s.notifyMilestone(ctx, orig.ProjectID)
		return &QuickDonateResult{Status: "succeeded", DonationID: d.ID}, nil
	}

	// 3Dセキュア再認証が必要
	return &QuickDonateResult{
		Status:       "requires_action",
		ClientSecret: result.ClientSecret,
	}, nil
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

	// Create subscription management record
	var subscriptionID string
	if s.subscriptionRepo != nil {
		sub := &model.Subscription{
			ProjectID:            projectID,
			DonorType:            donorType,
			DonorID:              donorID,
			Amount:               amount,
			Currency:             currency,
			Message:              obj.Metadata["message"],
			StripeSubscriptionID: obj.ID,
		}
		if err := s.subscriptionRepo.Create(ctx, sub); err != nil && !errors.Is(err, repository.ErrDuplicate) {
			return err
		}
		subscriptionID = sub.ID
	}

	// Create initial donation record
	d := &model.Donation{
		ProjectID:            projectID,
		DonorType:            donorType,
		DonorID:              donorID,
		Amount:               amount,
		Currency:             currency,
		Message:              obj.Metadata["message"],
		IsRecurring:          true,
		Source:               "checkout",
		SubscriptionID:       subscriptionID,
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
	// Delete from subscriptions table (donation history is preserved)
	if s.subscriptionRepo != nil {
		return s.subscriptionRepo.DeleteByStripeSubscriptionID(ctx, subscriptionID)
	}
	// Fallback to old behavior if subscriptionRepo not configured
	return s.donationRepo.DeleteByStripeSubscriptionID(ctx, subscriptionID)
}

// handleInvoicePaymentSucceeded はサブスクの請求成功時に寄付レコードを作成し、アクティビティを記録する
func (s *StripeServiceImpl) handleInvoicePaymentSucceeded(ctx context.Context, event pkgstripe.WebhookEvent) error {
	obj := event.Data.Object
	subscriptionID := obj.Subscription
	if subscriptionID == "" {
		return nil // one-time invoice, skip
	}

	// Look up subscription info (prefer subscriptionRepo, fallback to donationRepo)
	var projectID, donorType, donorID, message string
	var amount int
	var currency string
	var subTableID string // subscription table ID for FK

	if s.subscriptionRepo != nil {
		sub, err := s.subscriptionRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
		if err != nil || sub == nil {
			return nil // subscription not found, skip silently
		}
		projectID = sub.ProjectID
		donorType = sub.DonorType
		donorID = sub.DonorID
		amount = sub.Amount
		currency = sub.Currency
		message = sub.NextBillingMessage
		subTableID = sub.ID
	} else {
		d, err := s.donationRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
		if err != nil || d == nil {
			return nil // donation not found, skip silently
		}
		projectID = d.ProjectID
		donorType = d.DonorType
		donorID = d.DonorID
		amount = d.Amount
		currency = d.Currency
		message = d.NextBillingMessage
	}

	// Override amount/currency from invoice if provided
	if obj.Amount > 0 {
		amount = obj.Amount
	}
	if obj.Currency != "" {
		currency = obj.Currency
	}
	if currency == "" {
		currency = "jpy"
	}

	// Determine message: next_billing_message takes priority, then subscription original message
	donationMessage := message

	// Create donation record for this renewal
	d := &model.Donation{
		ProjectID:            projectID,
		DonorType:            donorType,
		DonorID:              donorID,
		Amount:               amount,
		Currency:             currency,
		Message:              donationMessage,
		IsRecurring:          true,
		Source:               "subscription_renewal",
		SubscriptionID:       subTableID,
		StripeSubscriptionID: subscriptionID,
		StripeInvoiceID:      obj.ID,
		StripePaymentID:      obj.PaymentIntent, // 返金時に必要
	}
	if err := s.donationRepo.Create(ctx, d); err != nil && !errors.Is(err, repository.ErrDuplicate) {
		return err
	}

	// Record activity and notify milestone
	s.recordDonationActivity(ctx, projectID, donorID, amount, donationMessage)
	s.notifyMilestone(ctx, projectID)

	// Clear next_billing_message on subscription (if it was set)
	if message != "" {
		empty := ""
		if s.subscriptionRepo != nil && subTableID != "" {
			_ = s.subscriptionRepo.Patch(ctx, subTableID, model.SubscriptionPatch{NextBillingMessage: &empty})
		} else {
			// fallback: clear on donation record
			orig, _ := s.donationRepo.GetByStripeSubscriptionID(ctx, subscriptionID)
			if orig != nil {
				_ = s.donationRepo.Patch(ctx, orig.ID, model.DonationPatch{NextBillingMessage: &empty})
			}
		}
	}

	return nil
}
