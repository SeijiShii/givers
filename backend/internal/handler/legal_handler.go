package handler

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// allowedLegalTypes is the allowlist of legal document type names.
// Only these values may be requested via GET /api/legal/{type}.
var allowedLegalTypes = map[string]bool{
	"terms":        true,
	"privacy":      true,
	"disclaimer":   true,
	"commerce-law": true,
}

// allowedLangs is the allowlist of language codes for the ?lang= query parameter.
// Requests with a lang value not in this set fall back to "en".
var allowedLangs = map[string]bool{
	"en": true,
	"ja": true,
}

// LegalConfig holds configuration for the LegalHandler.
type LegalConfig struct {
	// DocsDir is the root directory from which legal Markdown files are read.
	// Files are expected at DocsDir/{lang}/{type}.md.
	// Corresponds to the LEGAL_DOCS_DIR environment variable.
	DocsDir string
}

// LegalHandler handles GET /api/legal/{type}.
type LegalHandler struct {
	cfg LegalConfig
}

// NewLegalHandler creates a LegalHandler with the given configuration.
func NewLegalHandler(cfg LegalConfig) *LegalHandler {
	return &LegalHandler{cfg: cfg}
}

// Legal handles GET /api/legal/{type}[?lang=<lang>].
// Returns the Markdown content of the requested legal document.
//
// Language resolution order:
//  1. Use ?lang= if it is in the allowedLangs list.
//  2. Fall back to "en" for unknown or missing lang values.
//  3. If the resolved-language file is absent, fall back to "en".
//
// Responds 404 when neither the requested language nor the en/ fallback exists.
// Rejects path traversal attempts with 400.
func (h *LegalHandler) Legal(w http.ResponseWriter, r *http.Request) {
	docType := r.PathValue("type")

	// Security: reject any traversal characters in the type path segment before allowlist check.
	if strings.Contains(docType, "/") || strings.Contains(docType, "\\") || strings.Contains(docType, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Allowlist check: only known document types are valid.
	if !allowedLegalTypes[docType] {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Resolve the requested language; fall back to "en" for invalid/missing values.
	lang := r.URL.Query().Get("lang")
	if !allowedLangs[lang] {
		lang = "en"
	}

	// Security: validate the lang value against the allowlist (defense in depth).
	// The allowedLangs check above already covers this, but guard against any edge cases.
	if strings.Contains(lang, "/") || strings.Contains(lang, "\\") || strings.Contains(lang, "..") {
		lang = "en"
	}

	// Build the absolute root directory path.
	absDir, err := filepath.Abs(h.cfg.DocsDir)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	content, err := h.readLegalFile(absDir, lang, docType)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the requested lang file is missing and lang is not already "en",
			// try the "en" fallback.
			if lang != "en" {
				content, err = h.readLegalFile(absDir, "en", docType)
			}
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// readLegalFile reads {absDir}/{lang}/{docType}.md and verifies the resolved path
// stays within absDir (defense-in-depth path traversal guard).
func (h *LegalHandler) readLegalFile(absDir, lang, docType string) ([]byte, error) {
	filePath := filepath.Join(absDir, lang, docType+".md")

	// Confirm the resolved path is still within DocsDir.
	if !strings.HasPrefix(filePath, absDir+string(filepath.Separator)) && filePath != absDir {
		return nil, &pathTraversalError{path: filePath}
	}

	return os.ReadFile(filePath)
}

// pathTraversalError is returned when a constructed path escapes the DocsDir.
// It does NOT implement fs.ErrNotExist so callers can distinguish it from a missing file.
type pathTraversalError struct {
	path string
}

func (e *pathTraversalError) Error() string {
	return "path traversal detected: " + e.path
}
