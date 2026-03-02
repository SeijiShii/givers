package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/givers/backend/internal/service"
)

// BadgeHandler handles GET /api/projects/{id}/badge.svg.
type BadgeHandler struct {
	projectSvc service.ProjectService
}

// NewBadgeHandler creates a BadgeHandler.
func NewBadgeHandler(projectSvc service.ProjectService) *BadgeHandler {
	return &BadgeHandler{projectSvc: projectSvc}
}

// Badge returns a shields.io-style SVG badge for the project.
func (h *BadgeHandler) Badge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		h.renderBadge(w, "GIVErS", "not found", "#999")
		return
	}

	project, err := h.projectSvc.GetByID(r.Context(), id)
	if err != nil {
		slog.Debug("badge: project not found", "id", id, "error", err)
		w.WriteHeader(http.StatusNotFound)
		h.renderBadge(w, "GIVErS", "not found", "#999")
		return
	}

	if project.Status != "active" {
		h.renderBadge(w, "GIVErS", "inactive", "#999")
		return
	}

	// Determine target
	target := project.MonthlyTarget
	if target == 0 && project.OwnerWantMonthly != nil {
		target = *project.OwnerWantMonthly
	}

	if target == 0 {
		h.renderBadge(w, "GIVErS", "active", "#4c1")
		return
	}

	// Calculate achievement rate
	rate := project.CurrentMonthlyDonations * 100 / target

	// Signal thresholds
	warnThreshold := 60
	critThreshold := 30
	if project.Alerts != nil {
		warnThreshold = project.Alerts.WarningThreshold
		critThreshold = project.Alerts.CriticalThreshold
	}

	var color string
	switch {
	case rate >= warnThreshold:
		color = "#4c1" // green
	case rate >= critThreshold:
		color = "#dfb317" // yellow
	default:
		color = "#e05d44" // red
	}

	h.renderBadge(w, "GIVErS", fmt.Sprintf("%d%%", rate), color)
}

func (h *BadgeHandler) renderBadge(w http.ResponseWriter, left, right, rightColor string) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "max-age=300, s-maxage=300")

	leftWidth := measureText(left) + 10
	rightWidth := measureText(right) + 10
	totalWidth := leftWidth + rightWidth

	leftCenter := leftWidth * 5        // scaled by 10 for svg transform
	rightCenter := leftWidth*10 + rightWidth*5

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
  <title>%s: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
    <text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>
    <text x="%d" y="140" transform="scale(.1)">%s</text>
    <text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>
    <text x="%d" y="140" transform="scale(.1)">%s</text>
  </g>
</svg>`,
		totalWidth, escXML(left), escXML(right),
		escXML(left), escXML(right),
		totalWidth,
		leftWidth,
		leftWidth, rightWidth, rightColor,
		totalWidth,
		leftCenter, escXML(left),
		leftCenter, escXML(left),
		rightCenter, escXML(right),
		rightCenter, escXML(right),
	)

	_, _ = w.Write([]byte(svg))
}

// measureText approximates the pixel width for shields.io-style Verdana 11px text.
func measureText(s string) int {
	return int(float64(len(s))*6.8) + 2
}

// escXML escapes XML special characters to prevent injection.
func escXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
