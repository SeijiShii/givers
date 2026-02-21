package auth

import "context"

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
