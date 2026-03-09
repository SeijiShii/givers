package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockMessageRepository — MessageRepository のモック
// ---------------------------------------------------------------------------

type mockMessageRepository struct {
	createThreadFunc      func(ctx context.Context, t *model.MessageThread) error
	getThreadByIDFunc     func(ctx context.Context, id string) (*model.MessageThread, error)
	setThreadActiveFunc   func(ctx context.Context, id string, active bool) error
	listThreadsForUserFunc func(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	listThreadsForHostFunc func(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	insertMessageFunc     func(ctx context.Context, msg *model.Message) error
	listMessagesFunc      func(ctx context.Context, threadID string, limit int, cursor string) ([]*model.Message, string, error)
	markThreadReadFunc    func(ctx context.Context, userID, threadID string) error
	unreadThreadCountFunc func(ctx context.Context, userID string) (int, error)
}

func (m *mockMessageRepository) CreateThread(ctx context.Context, t *model.MessageThread) error {
	if m.createThreadFunc != nil {
		return m.createThreadFunc(ctx, t)
	}
	return nil
}

func (m *mockMessageRepository) GetThreadByID(ctx context.Context, id string) (*model.MessageThread, error) {
	if m.getThreadByIDFunc != nil {
		return m.getThreadByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockMessageRepository) SetThreadActive(ctx context.Context, id string, active bool) error {
	if m.setThreadActiveFunc != nil {
		return m.setThreadActiveFunc(ctx, id, active)
	}
	return nil
}

func (m *mockMessageRepository) ListThreadsForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	if m.listThreadsForUserFunc != nil {
		return m.listThreadsForUserFunc(ctx, userID, limit, cursor)
	}
	return nil, "", nil
}

func (m *mockMessageRepository) ListThreadsForHost(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	if m.listThreadsForHostFunc != nil {
		return m.listThreadsForHostFunc(ctx, hostUserID, limit, cursor)
	}
	return nil, "", nil
}

func (m *mockMessageRepository) InsertMessage(ctx context.Context, msg *model.Message) error {
	if m.insertMessageFunc != nil {
		return m.insertMessageFunc(ctx, msg)
	}
	return nil
}

func (m *mockMessageRepository) ListMessages(ctx context.Context, threadID string, limit int, cursor string) ([]*model.Message, string, error) {
	if m.listMessagesFunc != nil {
		return m.listMessagesFunc(ctx, threadID, limit, cursor)
	}
	return nil, "", nil
}

func (m *mockMessageRepository) MarkThreadRead(ctx context.Context, userID, threadID string) error {
	if m.markThreadReadFunc != nil {
		return m.markThreadReadFunc(ctx, userID, threadID)
	}
	return nil
}

func (m *mockMessageRepository) UnreadThreadCount(ctx context.Context, userID string) (int, error) {
	if m.unreadThreadCountFunc != nil {
		return m.unreadThreadCountFunc(ctx, userID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Tests: MessageService.CreateThread
// ---------------------------------------------------------------------------

func TestMessageService_CreateThread_Success(t *testing.T) {
	var capturedThread *model.MessageThread
	var capturedMsg *model.Message
	mock := &mockMessageRepository{
		createThreadFunc: func(ctx context.Context, th *model.MessageThread) error {
			th.ID = "thread-1"
			th.CreatedAt = time.Now()
			th.UpdatedAt = time.Now()
			capturedThread = th
			return nil
		},
		insertMessageFunc: func(ctx context.Context, msg *model.Message) error {
			capturedMsg = msg
			return nil
		},
	}

	svc := NewMessageService(mock, nil)
	thread, err := svc.CreateThread(context.Background(), "host-1", "user-1", "件名テスト", "初回メッセージ")
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if thread == nil {
		t.Fatal("expected thread, got nil")
	}
	if capturedThread.HostUserID != "host-1" {
		t.Errorf("expected HostUserID=host-1, got %q", capturedThread.HostUserID)
	}
	if capturedThread.ParticipantUserID != "user-1" {
		t.Errorf("expected ParticipantUserID=user-1, got %q", capturedThread.ParticipantUserID)
	}
	if capturedThread.Subject != "件名テスト" {
		t.Errorf("expected Subject=件名テスト, got %q", capturedThread.Subject)
	}
	if !capturedThread.Active {
		t.Error("expected Active=true for new thread")
	}
	if capturedMsg == nil {
		t.Fatal("expected initial message to be inserted")
	}
	if capturedMsg.ThreadID != "thread-1" {
		t.Errorf("expected message ThreadID=thread-1, got %q", capturedMsg.ThreadID)
	}
	if capturedMsg.SenderID != "host-1" {
		t.Errorf("expected message SenderID=host-1, got %q", capturedMsg.SenderID)
	}
	if capturedMsg.Body != "初回メッセージ" {
		t.Errorf("expected message Body=初回メッセージ, got %q", capturedMsg.Body)
	}
}

func TestMessageService_CreateThread_CreateThreadError(t *testing.T) {
	mock := &mockMessageRepository{
		createThreadFunc: func(ctx context.Context, th *model.MessageThread) error {
			return errors.New("db error")
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.CreateThread(context.Background(), "host-1", "user-1", "件名", "本文")
	if err == nil {
		t.Error("expected error from CreateThread, got nil")
	}
}

func TestMessageService_CreateThread_InsertMessageError(t *testing.T) {
	mock := &mockMessageRepository{
		createThreadFunc: func(ctx context.Context, th *model.MessageThread) error {
			th.ID = "thread-1"
			return nil
		},
		insertMessageFunc: func(ctx context.Context, msg *model.Message) error {
			return errors.New("db error")
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.CreateThread(context.Background(), "host-1", "user-1", "件名", "本文")
	if err == nil {
		t.Error("expected error from InsertMessage, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.SendMessage
// ---------------------------------------------------------------------------

func TestMessageService_SendMessage_Success(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
				Active:            true,
			}, nil
		},
		insertMessageFunc: func(ctx context.Context, msg *model.Message) error {
			msg.ID = "msg-1"
			return nil
		},
	}

	svc := NewMessageService(mock, nil)
	msg, err := svc.SendMessage(context.Background(), "thread-1", "user-1", "返信テスト")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Body != "返信テスト" {
		t.Errorf("expected Body=返信テスト, got %q", msg.Body)
	}
}

func TestMessageService_SendMessage_ThreadNotFound(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.SendMessage(context.Background(), "thread-1", "user-1", "本文")
	if err == nil {
		t.Error("expected error for non-existent thread")
	}
}

func TestMessageService_SendMessage_ThreadInactive(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
				Active:            false,
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.SendMessage(context.Background(), "thread-1", "user-1", "本文")
	if err == nil {
		t.Error("expected error for inactive thread")
	}
}

func TestMessageService_SendMessage_NonParticipant(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
				Active:            true,
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.SendMessage(context.Background(), "thread-1", "user-999", "本文")
	if err == nil {
		t.Error("expected error for non-participant sender")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.GetThread
// ---------------------------------------------------------------------------

func TestMessageService_GetThread_Success(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	thread, err := svc.GetThread(context.Background(), "thread-1", "user-1")
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if thread.ID != "thread-1" {
		t.Errorf("expected ID=thread-1, got %q", thread.ID)
	}
}

func TestMessageService_GetThread_NonParticipant(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.GetThread(context.Background(), "thread-1", "user-999")
	if err == nil {
		t.Error("expected error for non-participant")
	}
}

func TestMessageService_GetThread_NotFound(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.GetThread(context.Background(), "thread-1", "user-1")
	if err == nil {
		t.Error("expected error for not found")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.ListThreads
// ---------------------------------------------------------------------------

func TestMessageService_ListThreads_CallsRepository(t *testing.T) {
	want := []*model.MessageThread{{ID: "t1"}}
	mock := &mockMessageRepository{
		listThreadsForUserFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
			return want, "next", nil
		},
	}

	svc := NewMessageService(mock, nil)
	got, cursor, err := svc.ListThreads(context.Background(), "user-1", 20, "")
	if err != nil {
		t.Fatalf("ListThreads: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 thread, got %d", len(got))
	}
	if cursor != "next" {
		t.Errorf("expected cursor=next, got %q", cursor)
	}
}

func TestMessageService_ListThreads_PropagatesError(t *testing.T) {
	mock := &mockMessageRepository{
		listThreadsForUserFunc: func(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
			return nil, "", errors.New("db error")
		},
	}

	svc := NewMessageService(mock, nil)
	_, _, err := svc.ListThreads(context.Background(), "user-1", 20, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.ListThreadsForHost
// ---------------------------------------------------------------------------

func TestMessageService_ListThreadsForHost_CallsRepository(t *testing.T) {
	want := []*model.MessageThread{{ID: "t1"}, {ID: "t2"}}
	mock := &mockMessageRepository{
		listThreadsForHostFunc: func(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
			return want, "", nil
		},
	}

	svc := NewMessageService(mock, nil)
	got, _, err := svc.ListThreadsForHost(context.Background(), "host-1", 20, "")
	if err != nil {
		t.Fatalf("ListThreadsForHost: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 threads, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.ListMessages
// ---------------------------------------------------------------------------

func TestMessageService_ListMessages_MarksRead(t *testing.T) {
	var markReadCalled bool
	var markReadUserID, markReadThreadID string
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
			}, nil
		},
		listMessagesFunc: func(ctx context.Context, threadID string, limit int, cursor string) ([]*model.Message, string, error) {
			return []*model.Message{{ID: "msg-1", Body: "hello"}}, "", nil
		},
		markThreadReadFunc: func(ctx context.Context, userID, threadID string) error {
			markReadCalled = true
			markReadUserID = userID
			markReadThreadID = threadID
			return nil
		},
	}

	svc := NewMessageService(mock, nil)
	msgs, _, err := svc.ListMessages(context.Background(), "thread-1", "user-1", 50, "")
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
	if !markReadCalled {
		t.Error("expected MarkThreadRead to be called")
	}
	if markReadUserID != "user-1" {
		t.Errorf("expected markRead userID=user-1, got %q", markReadUserID)
	}
	if markReadThreadID != "thread-1" {
		t.Errorf("expected markRead threadID=thread-1, got %q", markReadThreadID)
	}
}

func TestMessageService_ListMessages_NonParticipant(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:                "thread-1",
				HostUserID:        "host-1",
				ParticipantUserID: "user-1",
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	_, _, err := svc.ListMessages(context.Background(), "thread-1", "user-999", 50, "")
	if err == nil {
		t.Error("expected error for non-participant")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.SetThreadActive
// ---------------------------------------------------------------------------

func TestMessageService_SetThreadActive_HostCanToggle(t *testing.T) {
	var capturedActive bool
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:         "thread-1",
				HostUserID: "host-1",
			}, nil
		},
		setThreadActiveFunc: func(ctx context.Context, id string, active bool) error {
			capturedActive = active
			return nil
		},
	}

	svc := NewMessageService(mock, nil)
	if err := svc.SetThreadActive(context.Background(), "thread-1", "host-1", false); err != nil {
		t.Fatalf("SetThreadActive: %v", err)
	}
	if capturedActive {
		t.Error("expected active=false")
	}
}

func TestMessageService_SetThreadActive_NonHostRejected(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return &model.MessageThread{
				ID:         "thread-1",
				HostUserID: "host-1",
			}, nil
		},
	}

	svc := NewMessageService(mock, nil)
	err := svc.SetThreadActive(context.Background(), "thread-1", "user-1", false)
	if err == nil {
		t.Error("expected error for non-host trying to toggle active")
	}
}

func TestMessageService_SetThreadActive_ThreadNotFound(t *testing.T) {
	mock := &mockMessageRepository{
		getThreadByIDFunc: func(ctx context.Context, id string) (*model.MessageThread, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewMessageService(mock, nil)
	err := svc.SetThreadActive(context.Background(), "thread-1", "host-1", false)
	if err == nil {
		t.Error("expected error for not found thread")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.UnreadCount
// ---------------------------------------------------------------------------

func TestMessageService_UnreadCount_CallsRepository(t *testing.T) {
	mock := &mockMessageRepository{
		unreadThreadCountFunc: func(ctx context.Context, userID string) (int, error) {
			return 3, nil
		},
	}

	svc := NewMessageService(mock, nil)
	count, err := svc.UnreadCount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

func TestMessageService_UnreadCount_PropagatesError(t *testing.T) {
	mock := &mockMessageRepository{
		unreadThreadCountFunc: func(ctx context.Context, userID string) (int, error) {
			return 0, errors.New("db error")
		},
	}

	svc := NewMessageService(mock, nil)
	_, err := svc.UnreadCount(context.Background(), "user-1")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: MessageService.BroadcastThread
// ---------------------------------------------------------------------------

func TestMessageService_BroadcastThread_CreatesThreadsForAllUsers(t *testing.T) {
	var createdThreads []*model.MessageThread
	var createdMsgs []*model.Message
	msgRepo := &mockMessageRepository{
		createThreadFunc: func(ctx context.Context, th *model.MessageThread) error {
			th.ID = "thread-" + th.ParticipantUserID
			createdThreads = append(createdThreads, th)
			return nil
		},
		insertMessageFunc: func(ctx context.Context, msg *model.Message) error {
			createdMsgs = append(createdMsgs, msg)
			return nil
		},
	}
	userRepo := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return []*model.User{
				{ID: "user-1", Name: "Alice"},
				{ID: "host-1", Name: "Host"},
				{ID: "user-2", Name: "Bob"},
			}, nil
		},
	}

	svc := NewMessageService(msgRepo, userRepo)
	count, err := svc.BroadcastThread(context.Background(), "host-1", "お知らせ", "全員へのメッセージ")
	if err != nil {
		t.Fatalf("BroadcastThread: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 threads created (host excluded), got %d", count)
	}
	if len(createdThreads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(createdThreads))
	}
	if len(createdMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(createdMsgs))
	}
	// Verify host is excluded
	for _, th := range createdThreads {
		if th.ParticipantUserID == "host-1" {
			t.Error("host should be excluded from broadcast")
		}
	}
}

func TestMessageService_BroadcastThread_NoUsers(t *testing.T) {
	msgRepo := &mockMessageRepository{}
	userRepo := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return []*model.User{
				{ID: "host-1", Name: "Host"},
			}, nil
		},
	}

	svc := NewMessageService(msgRepo, userRepo)
	count, err := svc.BroadcastThread(context.Background(), "host-1", "件名", "本文")
	if err != nil {
		t.Fatalf("BroadcastThread: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 threads when only host exists, got %d", count)
	}
}

func TestMessageService_BroadcastThread_UserRepoError(t *testing.T) {
	msgRepo := &mockMessageRepository{}
	userRepo := &mockAdminUserRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.User, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewMessageService(msgRepo, userRepo)
	_, err := svc.BroadcastThread(context.Background(), "host-1", "件名", "本文")
	if err == nil {
		t.Error("expected error from user repo, got nil")
	}
}
