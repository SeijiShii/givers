package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/givers/backend/internal/handler"
	"github.com/givers/backend/internal/logging"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/internal/storage"
	"github.com/givers/backend/pkg/auth"
	pkgstripe "github.com/givers/backend/pkg/stripe"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()    // backend/.env or CWD
	_ = godotenv.Load("../.env") // project root
	logging.Setup()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://givers:givers@localhost:5432/givers?sslmode=disable"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4321"
	}

	pool, err := repository.NewPool(context.Background(), dbURL)
	if err != nil {
		logging.Fatal("failed to connect to database", "error", err)
	}
	defer pool.Close()

	userRepo := repository.NewPgUserRepository(pool)
	projectRepo := repository.NewPgProjectRepository(pool)
	contactRepo := repository.NewPgContactRepository(pool)
	watchRepo := repository.NewPgWatchRepository(pool)
	projectUpdateRepo := repository.NewPgProjectUpdateRepository(pool)
	platformHealthRepo := repository.NewPgPlatformHealthRepository(pool)
	donationRepo := repository.NewPgDonationRepository(pool)
	subscriptionRepo := repository.NewPgSubscriptionRepository(pool)
	activityRepo := repository.NewPgActivityRepository(pool)
	costPresetRepo := repository.NewPgCostPresetRepository(pool)
	sessionRepo := repository.NewPgSessionRepository(pool)

	authService := service.NewAuthService(userRepo)
	projectService := service.NewProjectService(projectRepo)
	contactService := service.NewContactService(contactRepo)
	watchService := service.NewWatchService(watchRepo)
	projectUpdateService := service.NewProjectUpdateService(projectUpdateRepo)
	platformHealthService := service.NewPlatformHealthService(platformHealthRepo)
	sessionSvc := service.NewSessionService(sessionRepo)
	adminUserService := service.NewAdminUserServiceWithSessions(userRepo, sessionRepo)
	// Stripe 設定（未設定の場合は Stripe 機能を無効化）
	stripeClient := pkgstripe.NewClient(
		os.Getenv("STRIPE_SECRET_KEY"),
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	activityService := service.NewActivityService(activityRepo)
	milestoneService := service.NewMilestoneService(projectRepo, donationRepo, activityRepo)
	alertRepo := repository.NewPgDonationAlertRepository(pool)
	alertService := service.NewDonationAlertService(projectRepo, alertRepo)
	stripeService := service.NewStripeServiceFull(stripeClient, projectRepo, donationRepo, subscriptionRepo, userRepo, frontendURL, activityRepo, milestoneService, alertService)
	donationService := service.NewDonationService(donationRepo)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, stripeClient)
	costPresetService := service.NewCostPresetService(costPresetRepo)
	consentRepo := repository.NewPgConsentRepository(pool)
	consentService := service.NewConsentService(consentRepo)

	authRequired := os.Getenv("AUTH_REQUIRED") == "true"
	hostEmails := auth.ParseHostEmails(os.Getenv("HOST_EMAILS"))

	legalDocsDir := os.Getenv("LEGAL_DOCS_DIR")
	if legalDocsDir == "" {
		legalDocsDir = "./legal"
	}

	h := handler.New(pool, frontendURL)
	authHandler := handler.NewAuthHandler(authService, handler.AuthConfig{
		GoogleClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:  os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		GoogleRedirectPath:  "/api/auth/google/callback",
		GitHubRedirectPath:  "/api/auth/github/callback",
		DiscordRedirectPath: "/api/auth/discord/callback",
		FrontendURL:         frontendURL,
	}, sessionSvc)
	providersHandler := handler.NewProvidersHandler(handler.ProvidersConfig{
		GitHubClientID:  os.Getenv("GITHUB_CLIENT_ID"),
		DiscordClientID: os.Getenv("DISCORD_CLIENT_ID"),
		AppleClientID:   os.Getenv("APPLE_CLIENT_ID"),
		EnableEmail:     os.Getenv("ENABLE_EMAIL_LOGIN") == "true",
	})
	meHandler := handler.NewMeHandlerWithDonation(userRepo, sessionSvc, hostEmails, donationRepo)
	// Stripe が設定されている場合のみ v2 API でアカウント作成+オンボーディングを行う
	var connectAccountFunc handler.ConnectAccountFunc
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		connectAccountFunc = stripeService.CreateAccountAndOnboarding
	}
	stripeHandler := handler.NewStripeHandler(stripeService, frontendURL, sessionSvc)
	projectHandler := handler.NewProjectHandlerWithActivity(projectService, connectAccountFunc, activityService, consentService)
	contactHandler := handler.NewContactHandler(contactService)
	legalHandler := handler.NewLegalHandler(handler.LegalConfig{DocsDir: legalDocsDir})
	watchHandler := handler.NewWatchHandler(watchService)
	updateHandler := handler.NewProjectUpdateHandler(projectUpdateService, projectService)
	hostHandler := handler.NewHostHandler(platformHealthService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService, projectService, donationRepo)
	donationHandler := handler.NewDonationHandler(donationService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	activityHandler := handler.NewActivityHandler(activityService)
	chartHandler := handler.NewChartHandler(projectService, donationRepo)
	costPresetHandler := handler.NewCostPresetHandler(costPresetService)
	ownerDonationHandler := handler.NewOwnerDonationHandler(donationService, projectService)
	refundService := service.NewRefundService(donationRepo, projectRepo, stripeClient)
	refundHandler := handler.NewRefundHandler(refundService, projectService)
	badgeHandler := handler.NewBadgeHandler(projectService)
	announcementRepo := repository.NewPgAnnouncementRepository(pool)
	announcementService := service.NewAnnouncementService(announcementRepo)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	adminAlertHandler := handler.NewAdminAlertHandler(alertService)
	messageRepo := repository.NewPgMessageRepository(pool)
	messageService := service.NewMessageService(messageRepo, userRepo)
	messageHandler := handler.NewMessageHandler(messageService)

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}
	imageStorage := storage.NewLocalStorage(uploadsDir, "/uploads")
	imageHandler := handler.NewImageHandler(imageStorage, projectService, projectRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/auth/providers", providersHandler.Providers)
	mux.HandleFunc("GET /api/auth/google/login", authHandler.GoogleLoginURL)
	mux.HandleFunc("GET /api/auth/google/callback", authHandler.GoogleCallback)
	mux.HandleFunc("GET /api/auth/github/login", authHandler.GitHubLoginURL)
	mux.HandleFunc("GET /api/auth/github/callback", authHandler.GitHubCallback)
	mux.HandleFunc("GET /api/auth/discord/login", authHandler.DiscordLoginURL)
	mux.HandleFunc("GET /api/auth/discord/callback", authHandler.DiscordCallback)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/finalize", authHandler.FinalizeLogin)
	mux.HandleFunc("GET /api/me", meHandler.Me)
	mux.HandleFunc("POST /api/contact", contactHandler.Submit)
	mux.HandleFunc("GET /api/legal/{type}", legalHandler.Legal)

	// プロジェクト API（一覧・詳細は認証不要）
	mux.Handle("GET /api/projects", http.HandlerFunc(projectHandler.List))
	mux.Handle("GET /api/projects/{id}", http.HandlerFunc(projectHandler.Get))

	// 認証必要エンドポイント
	hostMW := auth.HostMiddleware(hostEmails, func(ctx context.Context, userID string) (string, error) {
		u, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return u.Email, nil
	})
	wrapAuth := func(next http.Handler) http.Handler {
		if authRequired {
			return auth.RequireAuth(sessionSvc)(hostMW(next))
		}
		return auth.DevAuth(hostMW(next))
	}
	wrapOptionalAuth := func(next http.Handler) http.Handler {
		if authRequired {
			return auth.OptionalAuth(sessionSvc)(next)
		}
		return auth.DevAuth(next)
	}
	mux.Handle("GET /api/me/projects", wrapAuth(http.HandlerFunc(projectHandler.MyProjects)))
	mux.Handle("POST /api/projects", wrapAuth(http.HandlerFunc(projectHandler.Create)))
	mux.Handle("PUT /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Update)))
	mux.Handle("DELETE /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Delete)))
	mux.Handle("PATCH /api/projects/{id}/status", wrapAuth(http.HandlerFunc(projectHandler.PatchStatus)))
	mux.Handle("POST /api/projects/{id}/image", wrapAuth(http.HandlerFunc(imageHandler.Upload)))
	mux.Handle("DELETE /api/projects/{id}/image", wrapAuth(http.HandlerFunc(imageHandler.Delete)))

	// プロジェクト更新 API
	mux.Handle("GET /api/projects/{id}/updates", http.HandlerFunc(updateHandler.List))
	mux.Handle("POST /api/projects/{id}/updates", wrapAuth(http.HandlerFunc(updateHandler.Create)))
	mux.Handle("PUT /api/projects/{id}/updates/{uid}", wrapAuth(http.HandlerFunc(updateHandler.UpdateUpdate)))
	mux.Handle("DELETE /api/projects/{id}/updates/{uid}", wrapAuth(http.HandlerFunc(updateHandler.Delete)))

	// ウォッチ API（認証必須）
	mux.Handle("POST /api/projects/{id}/watch", wrapAuth(http.HandlerFunc(watchHandler.Watch)))
	mux.Handle("DELETE /api/projects/{id}/watch", wrapAuth(http.HandlerFunc(watchHandler.Unwatch)))
	mux.Handle("GET /api/me/watches", wrapAuth(http.HandlerFunc(watchHandler.ListWatches)))

	// Admin routes (host-only — handler enforces IsHostFromContext)
	mux.Handle("GET /api/admin/contacts", wrapAuth(http.HandlerFunc(contactHandler.AdminList)))
	mux.Handle("PATCH /api/admin/contacts/{id}/status", wrapAuth(http.HandlerFunc(contactHandler.UpdateStatus)))
	mux.Handle("GET /api/admin/users", wrapAuth(http.HandlerFunc(adminUserHandler.List)))
	mux.Handle("GET /api/admin/users/{id}", wrapAuth(http.HandlerFunc(adminUserHandler.GetByID)))
	mux.Handle("PATCH /api/admin/users/{id}/suspend", wrapAuth(http.HandlerFunc(adminUserHandler.Suspend)))
	mux.Handle("GET /api/admin/disclosure-export", wrapAuth(http.HandlerFunc(adminUserHandler.DisclosureExport)))
	mux.Handle("GET /api/admin/alerts", wrapAuth(http.HandlerFunc(adminAlertHandler.List)))
	mux.Handle("PATCH /api/admin/alerts/{id}/status", wrapAuth(http.HandlerFunc(adminAlertHandler.UpdateStatus)))

	// お知らせ API（一覧はオプショナル認証、unread-count/read は認証必須）
	mux.Handle("GET /api/announcements", wrapOptionalAuth(http.HandlerFunc(announcementHandler.List)))
	mux.Handle("GET /api/announcements/unread-count", wrapAuth(http.HandlerFunc(announcementHandler.UnreadCount)))
	mux.Handle("POST /api/announcements/{id}/read", wrapAuth(http.HandlerFunc(announcementHandler.MarkRead)))
	// お知らせ管理 API（ホストのみ — handler 内で IsHostFromContext チェック）
	mux.Handle("GET /api/admin/announcements", wrapAuth(http.HandlerFunc(announcementHandler.AdminList)))
	mux.Handle("POST /api/admin/announcements", wrapAuth(http.HandlerFunc(announcementHandler.AdminCreate)))
	mux.Handle("PUT /api/admin/announcements/{id}", wrapAuth(http.HandlerFunc(announcementHandler.AdminUpdate)))
	mux.Handle("PATCH /api/admin/announcements/{id}/visibility", wrapAuth(http.HandlerFunc(announcementHandler.AdminSetVisibility)))

	// メッセージ API（認証必須）
	mux.Handle("GET /api/me/messages/threads", wrapAuth(http.HandlerFunc(messageHandler.ListThreads)))
	mux.Handle("GET /api/me/messages/threads/{id}", wrapAuth(http.HandlerFunc(messageHandler.GetThread)))
	mux.Handle("GET /api/me/messages/threads/{id}/messages", wrapAuth(http.HandlerFunc(messageHandler.ListMessages)))
	mux.Handle("POST /api/me/messages/threads/{id}/messages", wrapAuth(http.HandlerFunc(messageHandler.SendMessage)))
	mux.Handle("GET /api/me/messages/unread-count", wrapAuth(http.HandlerFunc(messageHandler.UnreadCount)))
	// メッセージ管理 API（ホストのみ — handler 内で IsHostFromContext チェック）
	mux.Handle("POST /api/admin/messages/threads", wrapAuth(http.HandlerFunc(messageHandler.AdminCreateThread)))
	mux.Handle("GET /api/admin/messages/threads", wrapAuth(http.HandlerFunc(messageHandler.AdminListThreads)))
	mux.Handle("PATCH /api/admin/messages/threads/{id}/active", wrapAuth(http.HandlerFunc(messageHandler.AdminSetThreadActive)))

	// Platform health (no auth required)
	mux.HandleFunc("GET /api/host", hostHandler.Get)

	// Activity feed & chart (no auth required)
	mux.HandleFunc("GET /api/activity", activityHandler.GlobalFeed)
	mux.HandleFunc("GET /api/projects/{id}/activity", activityHandler.ProjectFeed)
	mux.HandleFunc("GET /api/projects/{id}/chart", chartHandler.Chart)
	mux.HandleFunc("GET /api/projects/{id}/badge.svg", badgeHandler.Badge)

	// Owner donation history (owner or host auth required)
	mux.Handle("GET /api/projects/{id}/donations", wrapAuth(http.HandlerFunc(ownerDonationHandler.List)))

	// Refund routes (auth required)
	mux.Handle("POST /api/projects/{id}/donations/{donationId}/refund", wrapAuth(http.HandlerFunc(refundHandler.RefundByOwner)))
	mux.Handle("POST /api/me/donations/{donationId}/refund", wrapAuth(http.HandlerFunc(refundHandler.RefundByDonor)))

	// Donation routes (auth required)
	mux.Handle("GET /api/me/donations", wrapAuth(http.HandlerFunc(donationHandler.List)))
	mux.Handle("POST /api/me/migrate-from-token", wrapAuth(http.HandlerFunc(donationHandler.MigrateFromToken)))

	// Subscription routes (auth required)
	mux.Handle("GET /api/me/subscriptions", wrapAuth(http.HandlerFunc(subscriptionHandler.List)))
	mux.Handle("PATCH /api/me/subscriptions/{id}", wrapAuth(http.HandlerFunc(subscriptionHandler.Patch)))
	mux.Handle("DELETE /api/me/subscriptions/{id}", wrapAuth(http.HandlerFunc(subscriptionHandler.Delete)))

	// Cost preset routes (auth required)
	mux.Handle("GET /api/me/cost-presets", wrapAuth(http.HandlerFunc(costPresetHandler.List)))
	mux.Handle("POST /api/me/cost-presets", wrapAuth(http.HandlerFunc(costPresetHandler.Create)))
	mux.Handle("PUT /api/me/cost-presets/reorder", wrapAuth(http.HandlerFunc(costPresetHandler.Reorder)))
	mux.Handle("PUT /api/me/cost-presets/{id}", wrapAuth(http.HandlerFunc(costPresetHandler.Update)))
	mux.Handle("DELETE /api/me/cost-presets/{id}", wrapAuth(http.HandlerFunc(costPresetHandler.Delete)))

	// Stripe routes (no auth — Stripe handles security via signatures/state)
	mux.HandleFunc("GET /api/stripe/onboarding/return", stripeHandler.OnboardingReturn)
	mux.HandleFunc("GET /api/stripe/onboarding/refresh", stripeHandler.OnboardingRefresh)
	// Rate limit on checkout endpoint (10 req/min per IP)
	checkoutRL := handler.NewRateLimiter(10)
	mux.Handle("POST /api/donations/checkout", checkoutRL.Middleware(http.HandlerFunc(stripeHandler.Checkout)))
	mux.Handle("POST /api/donations/quick", checkoutRL.Middleware(http.HandlerFunc(stripeHandler.QuickDonate)))
	mux.HandleFunc("POST /api/webhooks/stripe", stripeHandler.Webhook)

	// 画像ファイルの静的配信
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Middleware chain: RequestLogger → SecurityHeaders → CORS → mux
	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler.RequestLogger(handler.SecurityHeaders(h.CORS(mux))),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
