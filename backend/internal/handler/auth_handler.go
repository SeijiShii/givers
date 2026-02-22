package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ── Server-side OAuth state & one-time code storage ──────────────────────

// pendingStates stores OAuth state tokens server-side (replaces cookie-based state).
// This avoids cross-port cookie issues when the OAuth callback goes directly to
// the backend (:8080) but the state cookie was set via the Vite proxy (:4321).
var pendingStates sync.Map // state string → time.Time (expiry)

// pendingCodes stores one-time auth codes that relay session tokens through the
// Vite dev proxy so the session cookie is set on the frontend's origin (:4321).
var pendingCodes sync.Map // code string → string (session token)

func generateRandomString() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func storeOAuthState(state string) {
	pendingStates.Store(state, time.Now().Add(10*time.Minute))
}

func verifyAndDeleteOAuthState(state string) bool {
	val, ok := pendingStates.LoadAndDelete(state)
	if !ok {
		return false
	}
	return time.Now().Before(val.(time.Time))
}

func storeOneTimeCode(code, sessionToken string) {
	pendingCodes.Store(code, sessionToken)
	go func() {
		time.Sleep(60 * time.Second)
		pendingCodes.Delete(code)
	}()
}

func lookupOneTimeCode(code string) (string, bool) {
	val, ok := pendingCodes.LoadAndDelete(code)
	if !ok {
		return "", false
	}
	return val.(string), true
}

// ── Helpers ──────────────────────────────────────────────────────────────

// formatCookies はデバッグ用にクッキー名一覧を返す
func formatCookies(cookies []*http.Cookie) string {
	if len(cookies) == 0 {
		return "(none)"
	}
	names := make([]string, len(cookies))
	for i, c := range cookies {
		names[i] = fmt.Sprintf("%s=%s...", c.Name, truncate(c.Value, 8))
	}
	return strings.Join(names, ", ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ── Types ────────────────────────────────────────────────────────────────

var githubEndpoint = oauth2.Endpoint{
	AuthURL:  "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
}

// SessionCreatorDeleter はセッションの作成・削除を行うインターフェース
type SessionCreatorDeleter interface {
	CreateSession(ctx context.Context, userID string) (*model.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

// AuthHandler は認証関連の HTTP ハンドラ
type AuthHandler struct {
	authService  service.AuthService
	googleConfig *oauth2.Config
	githubConfig *oauth2.Config
	sessionSvc   SessionCreatorDeleter
	frontendURL  string
}

// AuthConfig は AuthHandler の設定
type AuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	GoogleRedirectPath string
	GitHubRedirectPath string
	FrontendURL        string
}

// NewAuthHandler は AuthHandler を生成する（DI: AuthService を注入）
func NewAuthHandler(authService service.AuthService, cfg AuthConfig, sessionSvc SessionCreatorDeleter) *AuthHandler {
	redirectBase := os.Getenv("BACKEND_URL")
	if redirectBase == "" {
		redirectBase = "http://localhost:8080"
	}

	googleConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  redirectBase + cfg.GoogleRedirectPath,
		Scopes:       []string{"profile", "email"},
		Endpoint:     google.Endpoint,
	}

	githubConfig := &oauth2.Config{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  redirectBase + cfg.GitHubRedirectPath,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     githubEndpoint,
	}

	return &AuthHandler{
		authService:  authService,
		googleConfig: googleConfig,
		githubConfig: githubConfig,
		sessionSvc:   sessionSvc,
		frontendURL:  cfg.FrontendURL,
	}
}

// ── Google OAuth ─────────────────────────────────────────────────────────

// googleUserInfo は Google userinfo API のレスポンス
type googleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GoogleLoginURL は Google OAuth の認証 URL を返す（GET /api/auth/google/login）
func (h *AuthHandler) GoogleLoginURL(w http.ResponseWriter, r *http.Request) {
	state := generateRandomString()
	storeOAuthState(state)
	url := h.googleConfig.AuthCodeURL(state)
	log.Printf("[AUTH] GoogleLoginURL: redirectURL=%s, clientID=%s...%s",
		h.googleConfig.RedirectURL,
		h.googleConfig.ClientID[:8],
		h.googleConfig.ClientID[len(h.googleConfig.ClientID)-4:])
	log.Printf("[AUTH] GoogleLoginURL: state=%s... (server-side), authURL length=%d", state[:8], len(url))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// GoogleCallback は OAuth コールバックを処理する（GET /api/auth/google/callback）
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] GoogleCallback: START — url=%s", r.URL.String())
	log.Printf("[AUTH] GoogleCallback: cookies received: %s", formatCookies(r.Cookies()))

	// Server-side state verification (no cookies needed)
	queryState := r.URL.Query().Get("state")
	if !verifyAndDeleteOAuthState(queryState) {
		log.Printf("[AUTH] GoogleCallback: FAIL — state verification failed (state=%s...)", truncate(queryState, 8))
		http.Redirect(w, r, h.frontendURL+"/?error=invalid_state", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: state verified OK (server-side)")

	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("[AUTH] GoogleCallback: FAIL — no code in query params")
		http.Redirect(w, r, h.frontendURL+"/?error=no_code", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: code received (length=%d)", len(code))

	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("[AUTH] GoogleCallback: FAIL — token exchange error: %v", err)
		http.Redirect(w, r, h.frontendURL+"/?error=exchange_failed", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: token exchange OK (type=%s, expiry=%v)", token.TokenType, token.Expiry)

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("[AUTH] GoogleCallback: FAIL — userinfo request error: %v", err)
		http.Redirect(w, r, h.frontendURL+"/?error=userinfo_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()
	log.Printf("[AUTH] GoogleCallback: userinfo response status=%d", resp.StatusCode)

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Printf("[AUTH] GoogleCallback: FAIL — userinfo decode error: %v", err)
		http.Redirect(w, r, h.frontendURL+"/?error=decode_failed", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: userinfo OK — sub=%s, email=%s, name=%s", info.Sub, info.Email, info.Name)

	user, err := h.authService.GetOrCreateUserFromGoogle(r.Context(), &service.GoogleUserInfo{
		Sub:   info.Sub,
		Email: info.Email,
		Name:  info.Name,
	})
	if err != nil {
		log.Printf("[AUTH] GoogleCallback: FAIL — GetOrCreateUser error: %v", err)
		http.Redirect(w, r, h.frontendURL+"/?error=create_user_failed", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: user OK — id=%s, email=%s, name=%s", user.ID, user.Email, user.Name)

	session, err := h.sessionSvc.CreateSession(r.Context(), user.ID)
	if err != nil {
		log.Printf("[AUTH] GoogleCallback: FAIL — session creation error: %v", err)
		http.Redirect(w, r, h.frontendURL+"/?error=session_failed", http.StatusFound)
		return
	}
	log.Printf("[AUTH] GoogleCallback: session created — token=%s..., expiresAt=%v", session.Token[:16], session.ExpiresAt)

	// One-time code relay: redirect through the Vite proxy so the session cookie
	// is set on the frontend's origin (localhost:4321), not the backend's (localhost:8080).
	oneTimeCode := generateRandomString()
	storeOneTimeCode(oneTimeCode, session.Token)
	redirectURL := h.frontendURL + "/api/auth/finalize?code=" + oneTimeCode
	log.Printf("[AUTH] GoogleCallback: SUCCESS — redirecting via code relay to %s", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// ── GitHub OAuth ─────────────────────────────────────────────────────────

// GitHubLoginURL は GitHub OAuth の認証 URL を返す（GET /api/auth/github/login）
func (h *AuthHandler) GitHubLoginURL(w http.ResponseWriter, r *http.Request) {
	state := generateRandomString()
	storeOAuthState(state)
	url := h.githubConfig.AuthCodeURL(state)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// githubUserInfo は GitHub API のレスポンス
type githubUserInfo struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GitHubCallback は OAuth コールバックを処理する（GET /api/auth/github/callback）
func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] GitHubCallback: START — url=%s", r.URL.String())

	queryState := r.URL.Query().Get("state")
	if !verifyAndDeleteOAuthState(queryState) {
		log.Printf("[AUTH] GitHubCallback: FAIL — state verification failed")
		http.Redirect(w, r, h.frontendURL+"/?error=invalid_state", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.frontendURL+"/?error=no_code", http.StatusFound)
		return
	}

	token, err := h.githubConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=exchange_failed", http.StatusFound)
		return
	}

	client := h.githubConfig.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=userinfo_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()

	var info githubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=decode_failed", http.StatusFound)
		return
	}

	// GitHub は email が private の場合 null になるため、別 API で取得を試みる
	if info.Email == "" {
		emailsResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailsResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if json.NewDecoder(emailsResp.Body).Decode(&emails) == nil {
				for _, e := range emails {
					if e.Primary {
						info.Email = e.Email
						break
					}
				}
				if info.Email == "" && len(emails) > 0 {
					info.Email = emails[0].Email
				}
			}
		}
	}

	user, err := h.authService.GetOrCreateUserFromGitHub(r.Context(), &service.GitHubUserInfo{
		ID:    info.ID,
		Login: info.Login,
		Email: info.Email,
		Name:  info.Name,
	})
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=create_user_failed", http.StatusFound)
		return
	}

	session, err := h.sessionSvc.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=session_failed", http.StatusFound)
		return
	}

	// One-time code relay (same as Google)
	oneTimeCode := generateRandomString()
	storeOneTimeCode(oneTimeCode, session.Token)
	http.Redirect(w, r, h.frontendURL+"/api/auth/finalize?code="+oneTimeCode, http.StatusFound)
}

// ── Finalize (code relay) ────────────────────────────────────────────────

// FinalizeLogin exchanges a one-time code for a session cookie.
// Called via the Vite proxy: browser → :4321/api/auth/finalize → proxy → :8080.
// The Set-Cookie in the response is associated with :4321 (frontend origin).
func (h *AuthHandler) FinalizeLogin(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("[AUTH] FinalizeLogin: FAIL — missing code")
		http.Redirect(w, r, h.frontendURL+"/?error=missing_code", http.StatusFound)
		return
	}

	sessionToken, ok := lookupOneTimeCode(code)
	if !ok {
		log.Printf("[AUTH] FinalizeLogin: FAIL — invalid or expired code")
		http.Redirect(w, r, h.frontendURL+"/?error=invalid_code", http.StatusFound)
		return
	}

	cookie := &http.Cookie{
		Name:     auth.SessionCookieName(),
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   auth.SessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("ENV") == "production",
	}
	http.SetCookie(w, cookie)
	log.Printf("[AUTH] FinalizeLogin: SUCCESS — Set-Cookie name=%s, path=%s, maxAge=%d",
		cookie.Name, cookie.Path, cookie.MaxAge)

	http.Redirect(w, r, h.frontendURL+"/", http.StatusFound)
}

// ── Logout ───────────────────────────────────────────────────────────────

// Logout はログアウトする（POST /api/auth/logout）
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// DB からセッション削除
	if cookie, err := r.Cookie(auth.SessionCookieName()); err == nil && cookie.Value != "" {
		_ = h.sessionSvc.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}
