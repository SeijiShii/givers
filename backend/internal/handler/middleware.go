package handler

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SecurityHeaders adds security response headers (CSP, X-Frame-Options, etc.)
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-XSS-Protection", "0")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'none'")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter provides IP-based rate limiting using a sliding window.
type RateLimiter struct {
	maxPerMinute      int
	trustedProxyCount int
	mu                sync.Mutex
	clients           map[string]*clientWindow
}

type clientWindow struct {
	timestamps []time.Time
}

// NewRateLimiter creates a rate limiter with the given requests-per-minute limit.
// Assumes a single trusted reverse proxy (nginx) by default.
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		maxPerMinute:      maxPerMinute,
		trustedProxyCount: 1,
		clients:           make(map[string]*clientWindow),
	}
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop periodically removes stale entries from the clients map.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		windowStart := now.Add(-time.Minute)
		rl.mu.Lock()
		for ip, cw := range rl.clients {
			valid := cw.timestamps[:0]
			for _, ts := range cw.timestamps {
				if ts.After(windowStart) {
					valid = append(valid, ts)
				}
			}
			cw.timestamps = valid
			if len(cw.timestamps) == 0 {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an http.Handler that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)
		now := time.Now()
		windowStart := now.Add(-1 * time.Minute)

		rl.mu.Lock()
		cw, ok := rl.clients[ip]
		if !ok {
			cw = &clientWindow{}
			rl.clients[ip] = cw
		}

		// Prune timestamps outside the window; in-place filter on shared backing array
		valid := cw.timestamps[:0]
		for _, ts := range cw.timestamps {
			if ts.After(windowStart) {
				valid = append(valid, ts)
			}
		}
		cw.timestamps = valid

		if len(cw.timestamps) >= rl.maxPerMinute {
			oldest := cw.timestamps[0]
			retryAfter := oldest.Add(time.Minute).Sub(now)
			rl.mu.Unlock()

			w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			if err := json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
			}); err != nil {
				log.Printf("[RateLimiter] failed to write response: %v", err)
			}
			return
		}

		cw.timestamps = append(cw.timestamps, now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func retryAfterSeconds(d time.Duration) string {
	secs := int(d.Seconds()) + 1
	if secs < 1 {
		secs = 1
	}
	return strconv.Itoa(secs)
}

// clientIP extracts the real client IP, reading from the rightmost trusted
// proxy position in X-Forwarded-For to prevent spoofing.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" && rl.trustedProxyCount > 0 {
		parts := strings.Split(xff, ",")
		// The rightmost entry added by our infrastructure is at
		// index len(parts) - trustedProxyCount.
		idx := len(parts) - rl.trustedProxyCount
		if idx >= 0 && idx < len(parts) {
			return strings.TrimSpace(parts[idx])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
