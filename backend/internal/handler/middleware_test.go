package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- A8: SecurityHeaders middleware tests ---

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	SecurityHeaders(inner).ServeHTTP(rec, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
		"X-XSS-Protection":      "0",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	for name, want := range headers {
		got := rec.Header().Get(name)
		if got != want {
			t.Errorf("%s: want %q, got %q", name, want, got)
		}
	}
}

func TestSecurityHeaders_CSP(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	SecurityHeaders(inner).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header not set")
	}

	requiredDirectives := []string{
		"default-src",
		"script-src",
		"frame-ancestors 'none'",
	}
	for _, d := range requiredDirectives {
		if !strings.Contains(csp, d) {
			t.Errorf("CSP missing directive %q: %s", d, csp)
		}
	}
}

func TestSecurityHeaders_HSTS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	SecurityHeaders(inner).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Fatal("Strict-Transport-Security header not set")
	}
	if !strings.Contains(hsts, "max-age=") {
		t.Errorf("HSTS missing max-age: %s", hsts)
	}
}

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	SecurityHeaders(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", rec.Code)
	}
}

// --- A9: RateLimiter tests ---

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(10)
	handler := rl.Middleware(inner)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/donations/checkout", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(5)
	handler := rl.Middleware(inner)

	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/donations/checkout", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 6th request, got %d", lastCode)
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(2)
	handler := rl.Middleware(inner)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got %d", rec.Code)
	}
}

func TestRateLimiter_ReturnsRetryAfterHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(1)
	handler := rl.Middleware(inner)

	req1 := httptest.NewRequest("POST", "/", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest("POST", "/", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
	if ra := rec2.Header().Get("Retry-After"); ra == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestRateLimiter_XForwardedFor_RightmostTrusted(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(1)
	// trustedProxyCount=1 means we read the rightmost (last) entry
	handler := rl.Middleware(inner)

	// Simulate: client 203.0.113.50 → nginx adds it as rightmost entry
	// Attacker cannot spoof this because nginx overwrites the last position
	req1 := httptest.NewRequest("POST", "/", nil)
	req1.RemoteAddr = "10.0.0.99:1234"
	req1.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Second request from same real client IP should be blocked,
	// even if attacker prepends a spoofed IP
	req2 := httptest.NewRequest("POST", "/", nil)
	req2.RemoteAddr = "10.0.0.99:1234"
	req2.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.50")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for same real client IP, got %d", rec2.Code)
	}
}

func TestRateLimiter_XForwardedFor_SpoofedLeftmostIgnored(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(1)
	handler := rl.Middleware(inner)

	// First request: real client is 203.0.113.50
	req1 := httptest.NewRequest("POST", "/", nil)
	req1.RemoteAddr = "10.0.0.99:1234"
	req1.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request should succeed, got %d", rec1.Code)
	}

	// Attacker tries to spoof a different leftmost IP — should still be blocked
	// because we read the rightmost (trusted proxy) entry
	req2 := httptest.NewRequest("POST", "/", nil)
	req2.RemoteAddr = "10.0.0.99:1234"
	req2.Header.Set("X-Forwarded-For", "9.9.9.9, 203.0.113.50")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("spoofed leftmost IP should not bypass rate limit, got %d", rec2.Code)
	}
}
