package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
	"github.com/opentermius/opentermius/internal/ports"
)

type TerminalSession struct {
	ID        uuid.UUID
	VaultID   uuid.UUID
	HostLabel string
	HostID    uuid.UUID
	OwnerID   uuid.UUID
	StartedAt time.Time
	mu        sync.RWMutex
	listeners map[uuid.UUID]chan []byte
}

type TerminalUseCase struct {
	logRepo  ports.SessionLogRepository
	mu       sync.RWMutex
	sessions map[uuid.UUID]*TerminalSession
}

func NewTerminalUseCase(logRepo ports.SessionLogRepository) *TerminalUseCase {
	return &TerminalUseCase{
		logRepo:  logRepo,
		sessions: make(map[uuid.UUID]*TerminalSession),
	}
}

func (t *TerminalUseCase) CreateSession(vaultID, hostID, ownerID uuid.UUID, hostLabel string) *TerminalSession {
	session := &TerminalSession{
		ID:        uuid.New(),
		VaultID:   vaultID,
		HostID:    hostID,
		HostLabel: hostLabel,
		OwnerID:   ownerID,
		StartedAt: time.Now().UTC(),
		listeners: make(map[uuid.UUID]chan []byte),
	}

	t.mu.Lock()
	t.sessions[session.ID] = session
	t.mu.Unlock()

	return session
}

func (t *TerminalUseCase) GetSession(sessionID uuid.UUID) *TerminalSession {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sessions[sessionID]
}

func (s *TerminalSession) Subscribe(listenerID uuid.UUID) chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan []byte, 128)
	s.listeners[listenerID] = ch
	return ch
}

func (s *TerminalSession) Unsubscribe(listenerID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.listeners[listenerID]; ok {
		close(ch)
		delete(s.listeners, listenerID)
	}
}

func (s *TerminalSession) Broadcast(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.listeners {
		select {
		case ch <- data:
		default:
		}
	}
}

func (t *TerminalUseCase) CloseSession(ctx context.Context, sessionID uuid.UUID, recordedData string) error {
	t.mu.Lock()
	session, exists := t.sessions[sessionID]
	if exists {
		delete(t.sessions, sessionID)
	}
	t.mu.Unlock()

	if !exists {
		return domain.ErrNotFound
	}

	now := time.Now().UTC()
	log := &domain.SessionLog{
		ID:           uuid.New(),
		VaultID:      session.VaultID,
		UserID:       session.OwnerID,
		HostLabel:    session.HostLabel,
		StartedAt:    session.StartedAt,
		EndedAt:      &now,
		RecordedData: recordedData,
	}

	if t.logRepo != nil {
		return t.logRepo.Create(ctx, log)
	}
	return nil
}
