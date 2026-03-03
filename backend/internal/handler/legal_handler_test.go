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
// Uses the new en/ subdirectory structure (no lang param → defaults to en/).
func TestLegalHandler_Terms_Success(t *testing.T) {
	dir := t.TempDir()
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	content := "# Terms of Service\n\nThese are the terms."
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte(content), 0o644); err != nil {
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
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	content := "# Privacy Policy\n\nWe respect your privacy."
	if err := os.WriteFile(filepath.Join(enDir, "privacy.md"), []byte(content), 0o644); err != nil {
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
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	content := "# Disclaimer\n\nNo warranties."
	if err := os.WriteFile(filepath.Join(enDir, "disclaimer.md"), []byte(content), 0o644); err != nil {
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

// TestLegalHandler_NotFound_MissingFile verifies 404 when the file does not exist in en/.
func TestLegalHandler_NotFound_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// No files written — directory is empty (no en/ subdir either)

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
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte("# Terms"), 0o644); err != nil {
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

// TestLegalHandler_InvalidType_NotFound verifies that unknown types (not allowed list) return 404.
func TestLegalHandler_InvalidType_NotFound(t *testing.T) {
	dir := t.TempDir()
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	// Place a file with a non-allowed name to ensure the allowlist is enforced
	if err := os.WriteFile(filepath.Join(enDir, "other.md"), []byte("other"), 0o644); err != nil {
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

// ---------------------------------------------------------------------------
// Multi-language (lang query param) tests
// ---------------------------------------------------------------------------

// TestLegalHandler_Lang_Default verifies that when no lang param is given, en/ is used.
func TestLegalHandler_Lang_Default(t *testing.T) {
	dir := t.TempDir()

	// Create en/ subdirectory with terms.md
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	enContent := "# Terms of Service (EN)\n\nEnglish terms."
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte(enContent), 0o644); err != nil {
		t.Fatalf("write en/terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	// No lang param — should return en/ file
	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != enContent {
		t.Errorf("expected en content, got %q", rec.Body.String())
	}
}

// TestLegalHandler_Lang_JA verifies that ?lang=ja returns ja/ file when it exists.
func TestLegalHandler_Lang_JA(t *testing.T) {
	dir := t.TempDir()

	// Create ja/ subdirectory with terms.md
	jaDir := filepath.Join(dir, "ja")
	if err := os.MkdirAll(jaDir, 0o755); err != nil {
		t.Fatalf("mkdir ja: %v", err)
	}
	jaContent := "# 利用規約\n\n日本語の利用規約。"
	if err := os.WriteFile(filepath.Join(jaDir, "terms.md"), []byte(jaContent), 0o644); err != nil {
		t.Fatalf("write ja/terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms?lang=ja", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != jaContent {
		t.Errorf("expected ja content, got %q", rec.Body.String())
	}
}

// TestLegalHandler_Lang_JA_Fallback_EN verifies that when ?lang=ja but ja/ file is missing,
// it falls back to en/ file.
func TestLegalHandler_Lang_JA_Fallback_EN(t *testing.T) {
	dir := t.TempDir()

	// Only create en/ file — no ja/ file
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	enContent := "# Terms of Service (EN fallback)\n\nFallback to English."
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte(enContent), 0o644); err != nil {
		t.Fatalf("write en/terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms?lang=ja", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (fallback to en), got %d — body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != enContent {
		t.Errorf("expected en fallback content, got %q", rec.Body.String())
	}
}

// TestLegalHandler_Lang_Invalid_Fallback_EN verifies that an invalid lang param falls back to en/.
func TestLegalHandler_Lang_Invalid_Fallback_EN(t *testing.T) {
	dir := t.TempDir()

	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	enContent := "# Terms (EN)"
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte(enContent), 0o644); err != nil {
		t.Fatalf("write en/terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	// "xx" is not in the allowed language list → should fall back to en
	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms?lang=xx", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (invalid lang falls back to en), got %d", rec.Code)
	}
	if rec.Body.String() != enContent {
		t.Errorf("expected en content for invalid lang, got %q", rec.Body.String())
	}
}

// TestLegalHandler_CommerceLaw verifies that commerce-law type is now allowed.
func TestLegalHandler_CommerceLaw(t *testing.T) {
	dir := t.TempDir()

	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	content := "# Specified Commercial Transaction Act"
	if err := os.WriteFile(filepath.Join(enDir, "commerce-law.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write commerce-law.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/commerce-law", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for commerce-law, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != content {
		t.Errorf("expected content %q, got %q", content, rec.Body.String())
	}
}

// TestLegalHandler_Lang_PathTraversal verifies that path traversal via lang param is rejected.
func TestLegalHandler_Lang_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	// Place a sensitive file outside legal dir
	if err := os.WriteFile(filepath.Join(dir, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret.md: %v", err)
	}

	legalDir := filepath.Join(dir, "legal")
	if err := os.MkdirAll(legalDir, 0o755); err != nil {
		t.Fatalf("mkdir legal: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: legalDir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	// Attempt traversal via lang param: ?lang=../
	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms?lang=../", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("path traversal via lang must not succeed, got 200 with body: %s", rec.Body.String())
	}
}

// TestLegalHandler_Lang_ContentType verifies Content-Type is still text/markdown with lang param.
func TestLegalHandler_Lang_ContentType(t *testing.T) {
	dir := t.TempDir()

	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	if err := os.WriteFile(filepath.Join(enDir, "terms.md"), []byte("# Terms"), 0o644); err != nil {
		t.Fatalf("write terms.md: %v", err)
	}

	h := NewLegalHandler(LegalConfig{DocsDir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/legal/{type}", h.Legal)

	req := httptest.NewRequest(http.MethodGet, "/api/legal/terms?lang=en", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "text/markdown; charset=utf-8" {
		t.Errorf("expected Content-Type=text/markdown; charset=utf-8, got %q", ct)
	}
}

// TestLegalHandler_UnreadableFile verifies 500 when the file exists but cannot be read
// (e.g. permission denied). Uses the new en/ subdirectory structure.
func TestLegalHandler_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission tests are not meaningful")
	}

	dir := t.TempDir()
	enDir := filepath.Join(dir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("mkdir en: %v", err)
	}
	filePath := filepath.Join(enDir, "terms.md")
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
