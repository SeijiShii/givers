package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/givers/backend/internal/service"
)

// StripeHandler は Stripe 関連の HTTP ハンドラ
type StripeHandler struct {
	svc         service.StripeService
	frontendURL string
}

// NewStripeHandler は StripeHandler を生成する
func NewStripeHandler(svc service.StripeService, frontendURL string) *StripeHandler {
	return &StripeHandler{svc: svc, frontendURL: frontendURL}
}

// ConnectCallback handles GET /api/stripe/connect/callback
// Stripe Connect OAuth 完了後のコールバック。code と state(project_id) を受け取る。
func (h *StripeHandler) ConnectCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	projectID := r.URL.Query().Get("state")

	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "code_required"})
		return
	}
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "state_required"})
		return
	}

	if err := h.svc.CompleteConnect(r.Context(), code, projectID); err != nil {
		redirectURL := h.frontendURL + "/projects/" + projectID + "?stripe_error=1"
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	http.Redirect(w, r, h.frontendURL+"/projects/"+projectID+"?stripe_connected=1", http.StatusFound)
}

// Checkout handles POST /api/donations/checkout
// Stripe Checkout Session を作成して checkout_url を返す。
func (h *StripeHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ProjectID   string `json:"project_id"`
		Amount      int    `json:"amount"`
		Currency    string `json:"currency"`
		IsRecurring bool   `json:"is_recurring"`
		Message     string `json:"message"`
		Locale      string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}
	if req.ProjectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "project_id_required"})
		return
	}

	checkoutURL, err := h.svc.CreateCheckout(r.Context(), service.CheckoutRequest{
		ProjectID:   req.ProjectID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		IsRecurring: req.IsRecurring,
		Message:     req.Message,
		Locale:      req.Locale,
		FrontendURL: h.frontendURL,
	})
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"checkout_url": checkoutURL})
}

// Webhook handles POST /api/webhooks/stripe
// Stripe Webhook シグネチャ検証後にイベントを処理する。
func (h *StripeHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_signature"})
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "read_body_failed"})
		return
	}

	if err := h.svc.ProcessWebhook(r.Context(), payload, sigHeader); err != nil {
		if strings.Contains(err.Error(), "signature") {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "signature_verification_failed"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "webhook_processing_failed"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"received": true})
}
