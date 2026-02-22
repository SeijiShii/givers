package auth

import (
	"context"
	"net/http"
	"strings"
)

const isHostKey contextKey = "is_host"

// WithIsHost stores the host flag in the context.
func WithIsHost(ctx context.Context, isHost bool) context.Context {
	return context.WithValue(ctx, isHostKey, isHost)
}

// IsHostFromContext returns whether the authenticated user is a host.
// Returns false when not set.
func IsHostFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(isHostKey).(bool)
	return v
}

// ParseHostEmails parses a comma-separated list of host email addresses.
func ParseHostEmails(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	emails := make([]string, 0, len(parts))
	for _, p := range parts {
		if e := strings.TrimSpace(p); e != "" {
			emails = append(emails, e)
		}
	}
	return emails
}

// HostMiddleware creates middleware that sets the is_host flag based on user email.
// lookupEmail resolves a userID to an email address.
func HostMiddleware(hostEmails []string, lookupEmail func(ctx context.Context, userID string) (string, error)) func(http.Handler) http.Handler {
	set := make(map[string]bool, len(hostEmails))
	for _, e := range hostEmails {
		set[e] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID, ok := UserIDFromContext(ctx)
			if ok && len(set) > 0 {
				if email, err := lookupEmail(ctx, userID); err == nil && set[email] {
					ctx = WithIsHost(ctx, true)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
