package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
)

// ---------------------------------------------------------------------------
// Mock ProjectService for message handler tests
// ---------------------------------------------------------------------------

type mockMessageProjectService struct {
	getByIDFunc func(ctx context.Context, id string) (*model.Project, error)
}

func (m *mockMessageProjectService) List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
	return nil, nil
}
func (m *mockMessageProjectService) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockMessageProjectService) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockMessageProjectService) Create(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockMessageProjectService) Update(ctx context.Context, project *model.Project) error {
	return nil
}
func (m *mockMessageProjectService) Delete(ctx context.Context, id string) error {
	return nil
}

// ---------------------------------------------------------------------------
// GET /api/projects/:id/messages tests
// ---------------------------------------------------------------------------

func TestMessageHandler_List_RequiresAuth(t *testing.T) {
	h := NewMessageHandler(&mockDonationService{}, &mockMessageProjectService{})
	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/messages", nil)
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMessageHandler_List_ProjectNotFound(t *testing.T) {
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewMessageHandler(&mockDonationService{}, projMock)

	req := userAuthRequest(http.MethodGet, "/api/projects/p1/messages", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMessageHandler_List_Forbidden_NotOwner(t *testing.T) {
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user"}, nil
		},
	}
	h := NewMessageHandler(&mockDonationService{}, projMock)

	req := userAuthRequest(http.MethodGet, "/api/projects/p1/messages", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestMessageHandler_List_Success(t *testing.T) {
	now := time.Now()
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	donMock := &mockDonationService{
		listProjectMsgsFunc: func(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
			if projectID != "p1" {
				t.Errorf("expected projectID=p1, got %q", projectID)
			}
			if limit != 50 {
				t.Errorf("expected default limit=50, got %d", limit)
			}
			if sort != "desc" {
				t.Errorf("expected default sort=desc, got %q", sort)
			}
			return &model.DonationMessageResult{
				Messages: []*model.DonationMessage{
					{DonorName: "Taro", Amount: 1000, Message: "Thanks!", CreatedAt: now, IsRecurring: false},
				},
				Total: 1,
			}, nil
		},
	}
	h := NewMessageHandler(donMock, projMock)

	req := userAuthRequest(http.MethodGet, "/api/projects/p1/messages", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Messages []*model.DonationMessage `json:"messages"`
		Total    int                      `json:"total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if resp.Messages[0].DonorName != "Taro" {
		t.Errorf("expected donor_name=Taro, got %q", resp.Messages[0].DonorName)
	}
}

func TestMessageHandler_List_QueryParams(t *testing.T) {
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	var capturedLimit, capturedOffset int
	var capturedSort, capturedDonor string

	donMock := &mockDonationService{
		listProjectMsgsFunc: func(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
			capturedLimit = limit
			capturedOffset = offset
			capturedSort = sort
			capturedDonor = donor
			return &model.DonationMessageResult{Messages: []*model.DonationMessage{}, Total: 0}, nil
		},
	}
	h := NewMessageHandler(donMock, projMock)

	req := userAuthRequest(http.MethodGet, "/api/projects/p1/messages?limit=10&offset=5&sort=asc&donor=Taro", "")
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
	if capturedDonor != "Taro" {
		t.Errorf("expected donor=Taro, got %q", capturedDonor)
	}
}

func TestMessageHandler_List_HostCanAccess(t *testing.T) {
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "other-user"}, nil
		},
	}
	donMock := &mockDonationService{
		listProjectMsgsFunc: func(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
			return &model.DonationMessageResult{Messages: []*model.DonationMessage{}, Total: 0}, nil
		},
	}
	h := NewMessageHandler(donMock, projMock)

	// Host user accessing another user's project messages
	req := httptest.NewRequest(http.MethodGet, "/api/projects/p1/messages", nil)
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

func TestMessageHandler_List_ServiceError(t *testing.T) {
	projMock := &mockMessageProjectService{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			return &model.Project{ID: "p1", OwnerID: "user-1"}, nil
		},
	}
	donMock := &mockDonationService{
		listProjectMsgsFunc: func(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewMessageHandler(donMock, projMock)

	req := userAuthRequest(http.MethodGet, "/api/projects/p1/messages", "")
	req.SetPathValue("id", "p1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// Suppress unused import warning
var _ = service.ErrForbidden
