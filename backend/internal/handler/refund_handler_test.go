package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock RefundService
// ---------------------------------------------------------------------------

type mockRefundService struct {
	refundByOwnerFunc func(ctx context.Context, projectID, donationID, userID string, isHost bool) error
	refundByDonorFunc func(ctx context.Context, donationID, userID string) error
}

func (m *mockRefundService) RefundByOwner(ctx context.Context, projectID, donationID, userID string, isHost bool) error {
	if m.refundByOwnerFunc != nil {
		return m.refundByOwnerFunc(ctx, projectID, donationID, userID, isHost)
	}
	return nil
}

func (m *mockRefundService) RefundByDonor(ctx context.Context, donationID, userID string) error {
	if m.refundByDonorFunc != nil {
		return m.refundByDonorFunc(ctx, donationID, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests: POST /api/projects/{id}/donations/{donationId}/refund
// ---------------------------------------------------------------------------

func TestRefundHandler_RefundByOwner_Success(t *testing.T) {
	var calledProjectID, calledDonationID, calledUserID string

	refundSvc := &mockRefundService{
		refundByOwnerFunc: func(_ context.Context, projectID, donationID, userID string, isHost bool) error {
			calledProjectID = projectID
			calledDonationID = donationID
			calledUserID = userID
			if isHost {
				t.Error("expected isHost=false")
			}
			return nil
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	req.SetPathValue("id", "proj-1")
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "owner-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if calledProjectID != "proj-1" {
		t.Errorf("expected projectID=proj-1, got %q", calledProjectID)
	}
	if calledDonationID != "don-1" {
		t.Errorf("expected donationID=don-1, got %q", calledDonationID)
	}
	if calledUserID != "owner-1" {
		t.Errorf("expected userID=owner-1, got %q", calledUserID)
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "refunded" {
		t.Errorf("expected status=refunded, got %q", resp["status"])
	}
}

func TestRefundHandler_RefundByOwner_Unauthorized(t *testing.T) {
	h := NewRefundHandler(&mockRefundService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRefundHandler_RefundByOwner_Forbidden(t *testing.T) {
	refundSvc := &mockRefundService{
		refundByOwnerFunc: func(_ context.Context, _, _, _ string, _ bool) error {
			return service.ErrForbidden
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	req.SetPathValue("id", "proj-1")
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "other-user")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestRefundHandler_RefundByOwner_AlreadyRefunded(t *testing.T) {
	refundSvc := &mockRefundService{
		refundByOwnerFunc: func(_ context.Context, _, _, _ string, _ bool) error {
			return repository.ErrAlreadyRefunded
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	req.SetPathValue("id", "proj-1")
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "owner-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "already_refunded" {
		t.Errorf("expected error=already_refunded, got %q", resp["error"])
	}
}

func TestRefundHandler_RefundByOwner_InternalError(t *testing.T) {
	refundSvc := &mockRefundService{
		refundByOwnerFunc: func(_ context.Context, _, _, _ string, _ bool) error {
			return errors.New("stripe failure")
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	req.SetPathValue("id", "proj-1")
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "owner-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestRefundHandler_RefundByOwner_HostPassesIsHost(t *testing.T) {
	var gotIsHost bool
	refundSvc := &mockRefundService{
		refundByOwnerFunc: func(_ context.Context, _, _, _ string, isHost bool) error {
			gotIsHost = isHost
			return nil
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/donations/don-1/refund", nil)
	req.SetPathValue("id", "proj-1")
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "host-user")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByOwner(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !gotIsHost {
		t.Error("expected isHost=true for host user")
	}
}

// ---------------------------------------------------------------------------
// Tests: POST /api/me/donations/{donationId}/refund
// ---------------------------------------------------------------------------

func TestRefundHandler_RefundByDonor_Success(t *testing.T) {
	var calledDonationID, calledUserID string

	refundSvc := &mockRefundService{
		refundByDonorFunc: func(_ context.Context, donationID, userID string) error {
			calledDonationID = donationID
			calledUserID = userID
			return nil
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/me/donations/don-1/refund", nil)
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "donor-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByDonor(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if calledDonationID != "don-1" {
		t.Errorf("expected donationID=don-1, got %q", calledDonationID)
	}
	if calledUserID != "donor-1" {
		t.Errorf("expected userID=donor-1, got %q", calledUserID)
	}
}

func TestRefundHandler_RefundByDonor_Unauthorized(t *testing.T) {
	h := NewRefundHandler(&mockRefundService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/me/donations/don-1/refund", nil)
	rec := httptest.NewRecorder()
	h.RefundByDonor(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRefundHandler_RefundByDonor_Forbidden(t *testing.T) {
	refundSvc := &mockRefundService{
		refundByDonorFunc: func(_ context.Context, _, _ string) error {
			return service.ErrForbidden
		},
	}
	h := NewRefundHandler(refundSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/me/donations/don-1/refund", nil)
	req.SetPathValue("donationId", "don-1")
	ctx := auth.WithUserID(req.Context(), "wrong-user")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.RefundByDonor(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}
