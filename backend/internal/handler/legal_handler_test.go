package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /api/legal/:type tests
// ---------------------------------------------------------------------------

// TestLegalHandler_Terms_Success verifies that terms.md is served with 200.
func TestLegalHandler_Terms_Success(t *testing.T) {
	dir := t.TempDir()
	content := "# Terms of Service\n\nThese are the terms."
	if err := os.WriteFile(filepath.Join(dir, "terms.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != content {
		t.Errorf("expected body %q, got %q", content, rec.Body.String())
	}
}

// TestLegalHandler_Privacy_Success verifies that privacy.md is served correctly.
func TestLegalHandler_Privacy_Success(t *testing.T) {
	dir := t.TempDir()
	content := "# Privacy Policy\n\nWe respect your privacy."
	if err := os.WriteFile(filepath.Join(dir, "privacy.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write privacy.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/privacy", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != content {
		t.Errorf("expected body %q, got %q", content, rec.Body.String())
	}
}

// TestLegalHandler_Disclaimer_Success verifies that disclaimer.md is served correctly.
func TestLegalHandler_Disclaimer_Success(t *testing.T) {
	dir := t.TempDir()
	content := "# Disclaimer\n\nNo warranties."
	if err := os.WriteFile(filepath.Join(dir, "disclaimer.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write disclaimer.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/disclaimer", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestLegalHandler_NotFound_MissingFile verifies 404 when the file does not exist.
func TestLegalHandler_NotFound_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// No files written — directory is empty

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing file, got %d", rec.Code)
	}
}

// TestLegalHandler_PathTraversal_Rejected verifies that path traversal attempts are rejected.
func TestLegalHandler_PathTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	// Place a sensitive file one level up from the legal docs directory.
	sensitiveFile := filepath.Join(dir, "secret.md")
	if err := os.WriteFile(sensitiveFile, []byte("secret content"), 0o644); err != nil {
		t.Fatalf("write secret.md: %v", err)
	}

	// Create a subdirectory for legal docs
	legalDir := filepath.Join(dir, "legal")
	if err := os.MkdirAll(legalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: legalDir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	// Attempt path traversal via the {type} path parameter
	req := httptest.NewRequest(http.MethodGet, "/api/legal/..%2Fsecret", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Must NOT return 200 — either 400 or 404
	if rec.Code == http.StatusOK {
		t.Errorf("path traversal must not succeed, got 200 with body: %s", rec.Body.String())
	}
}

// TestLegalHandler_PathTraversal_DotDot verifies raw dot-dot is rejected.
func TestLegalHandler_PathTraversal_DotDot(t *testing.T) {
	dir := t.TempDir()
	legalDir := filepath.Join(dir, "legal")
	if err := os.MkdirAll(legalDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: legalDir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	// "foo" is not in the allowlist → 404 (safe path, type simply not allowed)
	req := httptest.NewRequest(http.MethodGet, "/api/legal/foo", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// foo.md does not exist → 404 (safe path, just missing file)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent type 'foo', got %d", rec.Code)
	}
}

// TestLegalHandler_ContentTypeMarkdown verifies the response Content-Type header.
func TestLegalHandler_ContentTypeMarkdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "terms.md"), []byte("# Terms"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "text/markdown; charset=utf-8" {
		t.Errorf("expected Content-Type=text/markdown; charset=utf-8, got %q", ct)
	}
}

// TestLegalHandler_InvalidType_NotFound verifies that unknown types (not terms/privacy/disclaimer) return 404.
func TestLegalHandler_InvalidType_NotFound(t *testing.T) {
	dir := t.TempDir()
	// Place a file with a non-allowed name to ensure the allowlist is enforced
	if err := os.WriteFile(filepath.Join(dir, "other.md"), []byte("other"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/other", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// "other" is not an allowed type → 404
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for disallowed type 'other', got %d", rec.Code)
	}
}

// TestLegalHandler_UnreadableFile verifies 500 when the file exists but cannot be read
// (e.g. permission denied).
func TestLegalHandler_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission tests are not meaningful")
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "terms.md")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Remove read permission so os.ReadFile returns a non-ErrNotExist error.
	if err := os.Chmod(filePath, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filePath, 0o644) })

	h := NewLegalHandler(LegalConfig{DocsDir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for unreadable file, got %d", rec.Code)
	}
}
