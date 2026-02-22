package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const oauthStateCookieName = "oauth_state"

// generateOAuthState は CSRF 対策用のランダム state 文字列を生成する
func generateOAuthState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// setStateCookie は state を HttpOnly クッキーに保存する
func setStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("ENV") == "production",
	})
}

// verifyOAuthState は state クッキーとクエリパラメータを照合する
func verifyOAuthState(r *http.Request) bool {
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	return cookie.Value == r.URL.Query().Get("state")
}

// clearStateCookie は state クッキーを削除する
func clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
}

var githubEndpoint = oauth2.Endpoint{
	AuthURL:  "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
}

// AuthHandler は認証関連の HTTP ハンドラ
type AuthHandler struct {
	authService     service.AuthService
	googleConfig    *oauth2.Config
	githubConfig    *oauth2.Config
	sessionSecret   []byte
	frontendURL     string
}

// AuthConfig は AuthHandler の設定
type AuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	GoogleRedirectPath string
	GitHubRedirectPath string
	SessionSecret      string
	FrontendURL        string
}

// NewAuthHandler は AuthHandler を生成する（DI: AuthService を注入）
func NewAuthHandler(authService service.AuthService, cfg AuthConfig) *AuthHandler {
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
		authService:   authService,
		googleConfig:  googleConfig,
		githubConfig:  githubConfig,
		sessionSecret: auth.SessionSecretBytes(cfg.SessionSecret),
		frontendURL:   cfg.FrontendURL,
	}
}

// googleUserInfo は Google userinfo API のレスポンス
type googleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GoogleLoginURL は Google OAuth の認証 URL を返す（GET /api/auth/google/login）
func (h *AuthHandler) GoogleLoginURL(w http.ResponseWriter, r *http.Request) {
	state := generateOAuthState()
	setStateCookie(w, state)
	url := h.googleConfig.AuthCodeURL(state)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// GoogleCallback は OAuth コールバックを処理する（GET /api/auth/google/callback）
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !verifyOAuthState(r) {
		clearStateCookie(w)
		http.Redirect(w, r, h.frontendURL+"/?error=invalid_state", http.StatusFound)
		return
	}
	clearStateCookie(w)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.frontendURL+"/?error=no_code", http.StatusFound)
		return
	}

	token, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=exchange_failed", http.StatusFound)
		return
	}

	client := h.googleConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=userinfo_failed", http.StatusFound)
		return
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=decode_failed", http.StatusFound)
		return
	}

	user, err := h.authService.GetOrCreateUserFromGoogle(r.Context(), &service.GoogleUserInfo{
		Sub:   info.Sub,
		Email: info.Email,
		Name:  info.Name,
	})
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/?error=create_user_failed", http.StatusFound)
		return
	}

	sessionToken := auth.CreateSessionToken(user.ID, h.sessionSecret)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName(),
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("ENV") == "production",
	})

	http.Redirect(w, r, h.frontendURL+"/", http.StatusFound)
}

// GitHubLoginURL は GitHub OAuth の認証 URL を返す（GET /api/auth/github/login）
func (h *AuthHandler) GitHubLoginURL(w http.ResponseWriter, r *http.Request) {
	state := generateOAuthState()
	setStateCookie(w, state)
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
	if !verifyOAuthState(r) {
		clearStateCookie(w)
		http.Redirect(w, r, h.frontendURL+"/?error=invalid_state", http.StatusFound)
		return
	}
	clearStateCookie(w)

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

	sessionToken := auth.CreateSessionToken(user.ID, h.sessionSecret)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName(),
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("ENV") == "production",
	})

	http.Redirect(w, r, h.frontendURL+"/", http.StatusFound)
}

// Logout はログアウトする（POST /api/auth/logout）
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
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
