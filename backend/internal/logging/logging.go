package logging

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

// Setup configures the global slog default with a JSON handler.
// Log level is controlled by the LOG_LEVEL environment variable
// (DEBUG, INFO, WARN, ERROR). Defaults to INFO.
// ERROR-level logs automatically include a stack trace.
func Setup() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	json := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	slog.SetDefault(slog.New(&stackHandler{Handler: json}))
}

func parseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Fatal logs at Error level and exits with code 1.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// stackHandler wraps a slog.Handler and appends a stack trace for ERROR+.
type stackHandler struct {
	slog.Handler
}

func (h *stackHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		r.AddAttrs(slog.String("stacktrace", string(buf[:n])))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *stackHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &stackHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *stackHandler) WithGroup(name string) slog.Handler {
	return &stackHandler{Handler: h.Handler.WithGroup(name)}
}
