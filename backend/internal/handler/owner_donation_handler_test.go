package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock ownerDonationService
// ---------------------------------------------------------------------------

type mockOwnerDonationService struct {
	listByProjectForOwnerFunc func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error)
}

func (m *mockOwnerDonationService) ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
	if m.listByProjectForOwnerFunc != nil {
		return m.listByProjectForOwnerFunc(ctx, projectID, limit, offset, sort, sourceFilter, donorFilter, hasMessage)
	}
	return &model.OwnerDonationResult{Donations: []*model.OwnerDonationItem{}, Total: 0}, nil
}

// ---------------------------------------------------------------------------
// Mock ProjectService for owner donation handler tests
// ---------------------------------------------------------------------------

type mockOwnerProjectService struct {
	getByIDFunc func(ctx context.Context, id string) (*model.Project, error)
}

func (m *mockOwnerProjectService) List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
	return nil, nil
}
func (m *mockOwnerProjectService) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockOwnerProjectService) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockOwnerProjectService) Create(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockOwnerProjectService) Update(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockOwnerProjectService) Delete(ctx context.Context, id string) error {
	return nil
}

// helper: auth request for owner donation tests
func ownerAuthRequest(method, url, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(r.Context(), "user-1")
	ctx = auth.WithIsHost(ctx, false)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// GET /api/projects/:id/donations tests
// ---------------------------------------------------------------------------

func TestOwnerDonationHandler_List_RequiresAuth(t *testing.T) {
	h := NewOwnerDonationHandler(&mockOwnerDonationService{}, &mockOwnerProjectService{})
	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/donations", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestOwnerDonationHandler_List_ProjectNotFound(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewOwnerDonationHandler(&mockOwnerDonationService{}, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestOwnerDonationHandler_List_Forbidden_NotOwner(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user"}, nil
		},
	}
	h := NewOwnerDonationHandler(&mockOwnerDonationService{}, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestOwnerDonationHandler_List_HostCanAccess(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user"}, nil
		},
	}
	donMock := &mockOwnerDonationService{
		listByProjectForOwnerFunc: func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
			return &model.OwnerDonationResult{Donations: []*model.OwnerDonationItem{}, Total: 0}, nil
		},
	}
	h := NewOwnerDonationHandler(donMock, projMock)

	// Host user accessing another user's project donations
	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/donations", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(req.Context(), "host-user")
	ctx = auth.WithIsHost(ctx, true)
	req = req.WithContext(ctx)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for host, got %d", rec.Code)
	}
}

func TestOwnerDonationHandler_List_Success(t *testing.T) {
	now := time.Now()
	donorName := "Taro"
	message := "Keep it up!"
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	donMock := &mockOwnerDonationService{
		listByProjectForOwnerFunc: func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
			if projectID != "p1" {
				t.Errorf("expected projectID=p1, got %q", projectID)
			}
			if limit != 50 {
				t.Errorf("expected default limit=50, got %d", limit)
			}
			if sort != "desc" {
				t.Errorf("expected default sort=desc, got %q", sort)
			}
			return &model.OwnerDonationResult{
				Donations: []*model.OwnerDonationItem{
					{
						ID:          "d1",
						DonorName:   &donorName,
						DonorType:   "user",
						Amount:      1000,
						Currency:    "jpy",
						Message:     &message,
						Source:      "checkout",
						IsRecurring: false,
						CreatedAt:   now,
					},
				},
				Total: 1,
			}, nil
		},
	}
	h := NewOwnerDonationHandler(donMock, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Donations []*model.OwnerDonationItem `json:"donations"`
		Total     int                        `json:"total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Donations) != 1 {
		t.Errorf("expected 1 donation, got %d", len(resp.Donations))
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if resp.Donations[0].ID != "d1" {
		t.Errorf("expected id=d1, got %q", resp.Donations[0].ID)
	}
	if *resp.Donations[0].DonorName != "Taro" {
		t.Errorf("expected donor_name=Taro, got %q", *resp.Donations[0].DonorName)
	}
}

func TestOwnerDonationHandler_List_QueryParams(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	var capturedLimit, capturedOffset int
	var capturedSort, capturedSource, capturedDonor string
	var capturedHasMessage bool

	donMock := &mockOwnerDonationService{
		listByProjectForOwnerFunc: func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
			capturedLimit = limit
			capturedOffset = offset
			capturedSort = sort
			capturedSource = sourceFilter
			capturedDonor = donorFilter
			capturedHasMessage = hasMessage
			return &model.OwnerDonationResult{Donations: []*model.OwnerDonationItem{}, Total: 0}, nil
		},
	}
	h := NewOwnerDonationHandler(donMock, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations?limit=10&offset=5&sort=asc&source=checkout&donor=Taro&has_message=true", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedLimit != 10 {
		t.Errorf("expected limit=10, got %d", capturedLimit)
	}
	if capturedOffset != 5 {
		t.Errorf("expected offset=5, got %d", capturedOffset)
	}
	if capturedSort != "asc" {
		t.Errorf("expected sort=asc, got %q", capturedSort)
	}
	if capturedSource != "checkout" {
		t.Errorf("expected source=checkout, got %q", capturedSource)
	}
	if capturedDonor != "Taro" {
		t.Errorf("expected donor=Taro, got %q", capturedDonor)
	}
	if !capturedHasMessage {
		t.Errorf("expected has_message=true, got false")
	}
}

func TestOwnerDonationHandler_List_DefaultParams(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	var capturedLimit, capturedOffset int
	var capturedSort, capturedSource, capturedDonor string
	var capturedHasMessage bool

	donMock := &mockOwnerDonationService{
		listByProjectForOwnerFunc: func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
			capturedLimit = limit
			capturedOffset = offset
			capturedSort = sort
			capturedSource = sourceFilter
			capturedDonor = donorFilter
			capturedHasMessage = hasMessage
			return &model.OwnerDonationResult{Donations: []*model.OwnerDonationItem{}, Total: 0}, nil
		},
	}
	h := NewOwnerDonationHandler(donMock, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if capturedLimit != 50 {
		t.Errorf("expected default limit=50, got %d", capturedLimit)
	}
	if capturedOffset != 0 {
		t.Errorf("expected default offset=0, got %d", capturedOffset)
	}
	if capturedSort != "desc" {
		t.Errorf("expected default sort=desc, got %q", capturedSort)
	}
	if capturedSource != "" {
		t.Errorf("expected default source=\"\", got %q", capturedSource)
	}
	if capturedDonor != "" {
		t.Errorf("expected default donor=\"\", got %q", capturedDonor)
	}
	if capturedHasMessage {
		t.Errorf("expected default has_message=false, got true")
	}
}

func TestOwnerDonationHandler_List_ServiceError(t *testing.T) {
	projMock := &mockOwnerProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	donMock := &mockOwnerDonationService{
		listByProjectForOwnerFunc: func(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string, hasMessage bool) (*model.OwnerDonationResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewOwnerDonationHandler(donMock, projMock)

	req := ownerAuthRequest(http.MethodGet, "/api/projects/p1/donations", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
