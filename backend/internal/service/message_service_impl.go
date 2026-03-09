package service

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

type messageServiceImpl struct {
	repo     repository.MessageRepository
	userRepo repository.UserRepository // optional: nil disables broadcast
}

// NewMessageService は MessageService の実装を生成する
func NewMessageService(repo repository.MessageRepository, userRepo repository.UserRepository) MessageService {
	return &messageServiceImpl{repo: repo, userRepo: userRepo}
}

func (s *messageServiceImpl) CreateThread(ctx context.Context, hostUserID, participantUserID, subject, body string) (*model.MessageThread, error) {
	thread := &model.MessageThread{
		HostUserID:        hostUserID,
		ParticipantUserID: participantUserID,
		Subject:           subject,
		Active:            true,
	}
	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return nil, err
	}

	msg := &model.Message{
		ThreadID: thread.ID,
		SenderID: hostUserID,
		Body:     body,
	}
	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}

	return thread, nil
}

func (s *messageServiceImpl) SendMessage(ctx context.Context, threadID, senderID, body string) (*model.Message, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if !thread.Active {
		return nil, errors.New("thread is inactive")
	}
	if thread.HostUserID != senderID && thread.ParticipantUserID != senderID {
		return nil, errors.New("forbidden: not a participant")
	}

	msg := &model.Message{
		ThreadID: threadID,
		SenderID: senderID,
		Body:     body,
	}
	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *messageServiceImpl) GetThread(ctx context.Context, threadID, requestingUserID string) (*model.MessageThread, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread.HostUserID != requestingUserID && thread.ParticipantUserID != requestingUserID {
		return nil, errors.New("forbidden: not a participant")
	}
	return thread, nil
}

func (s *messageServiceImpl) ListThreads(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	return s.repo.ListThreadsForUser(ctx, userID, limit, cursor)
}

func (s *messageServiceImpl) ListThreadsForHost(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	return s.repo.ListThreadsForHost(ctx, hostUserID, limit, cursor)
}

func (s *messageServiceImpl) ListMessages(ctx context.Context, threadID, requestingUserID string, limit int, cursor string) ([]*model.Message, string, error) {
	thread, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, "", err
	}
	if thread.HostUserID != requestingUserID && thread.ParticipantUserID != requestingUserID {
		return nil, "", errors.New("forbidden: not a participant")
	}

	messages, nextCursor, err := s.repo.ListMessages(ctx, threadID, limit, cursor)
	if err != nil {
		return nil, "", err
	}

	_ = s.repo.MarkThreadRead(ctx, requestingUserID, threadID)

	return messages, nextCursor, nil
}

func (s *messageServiceImpl) SetThreadActive(ctx context.Context, threadID, hostUserID string, active bool) error {
	thread, err := s.repo.GetThreadByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread.HostUserID != hostUserID {
		return errors.New("forbidden: only the host can change thread active state")
	}
	return s.repo.SetThreadActive(ctx, threadID, active)
}

func (s *messageServiceImpl) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadThreadCount(ctx, userID)
}

func (s *messageServiceImpl) BroadcastThread(ctx context.Context, hostUserID, subject, body string) (int, error) {
	if s.userRepo == nil {
		return 0, errors.New("broadcast not configured: userRepo is nil")
	}
	users, err := s.userRepo.List(ctx, 10000, 0)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, u := range users {
		if u.ID == hostUserID {
			continue
		}
		_, err := s.CreateThread(ctx, hostUserID, u.ID, subject, body)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
