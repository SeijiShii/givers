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
	"terms":      true,
	"privacy":    true,
	"disclaimer": true,
}

// LegalConfig holds configuration for the LegalHandler.
type LegalConfig struct {
	// DocsDir is the directory from which legal Markdown files are read.
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

// Legal handles GET /api/legal/{type}.
// Returns the Markdown content of the requested legal document.
// Responds 404 when the document does not exist.
// Rejects path traversal attempts with 400.
func (h *LegalHandler) Legal(w http.ResponseWriter, r *http.Request) {
	docType := r.PathValue("type")

	// Security: reject any traversal characters before allowlist check.
	if strings.Contains(docType, "/") || strings.Contains(docType, "\\") || strings.Contains(docType, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Allowlist check: only terms, privacy, disclaimer are valid.
	if !allowedLegalTypes[docType] {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Build the absolute file path and verify it stays within DocsDir.
	absDir, err := filepath.Abs(h.cfg.DocsDir)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(absDir, docType+".md")

	// Confirm the resolved path is still within DocsDir (defense in depth).
	if !strings.HasPrefix(filePath, absDir+string(filepath.Separator)) && filePath != absDir {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
