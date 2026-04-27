package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Session struct {
	ID        string    `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Manager struct {
	redisClient *redis.Client
	logger      *logrus.Logger
	mu          sync.RWMutex
	localCache  map[string]*Session
}

func NewManager(redisClient *redis.Client, logger *logrus.Logger) *Manager {
	return &Manager{
		redisClient: redisClient,
		logger:      logger,
		localCache:  make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(ctx context.Context, userID uuid.UUID, role string, ttl time.Duration) (*Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Role:      role,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}

	// Store in Redis
	key := m.sessionKey(sessionID)
	if err := m.redisClient.Set(ctx, key, session, ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// Also store in local cache for fast access
	m.mu.Lock()
	m.localCache[sessionID] = session
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"user_id":    userID,
		"role":       role,
	}).Info("session created")

	return session, nil
}

func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Check local cache first
	m.mu.RLock()
	if session, ok := m.localCache[sessionID]; ok {
		m.mu.RUnlock()
		if time.Now().After(session.ExpiresAt) {
			m.DeleteSession(ctx, sessionID)
			return nil, fmt.Errorf("session expired")
		}
		return session, nil
	}
	m.mu.RUnlock()

	// Fetch from Redis
	key := m.sessionKey(sessionID)
	var session Session
	if err := m.redisClient.Get(ctx, key).Scan(&session); err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Update local cache
	m.mu.Lock()
	m.localCache[sessionID] = &session
	m.mu.Unlock()

	return &session, nil
}

func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	key := m.sessionKey(sessionID)
	if err := m.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	m.mu.Lock()
	delete(m.localCache, sessionID)
	m.mu.Unlock()

	m.logger.WithField("session_id", sessionID).Info("session deleted")

	return nil
}

func (m *Manager) RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := m.sessionKey(sessionID)
	if err := m.redisClient.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	m.mu.Lock()
	if session, ok := m.localCache[sessionID]; ok {
		session.ExpiresAt = time.Now().Add(ttl)
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) ValidateSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		m.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (m *Manager) CleanupExpiredSessions(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, session := range m.localCache {
		if now.After(session.ExpiresAt) {
			delete(m.localCache, id)
			m.logger.WithField("session_id", id).Debug("cleaned up expired session from cache")
		}
	}
}

func (m *Manager) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}
