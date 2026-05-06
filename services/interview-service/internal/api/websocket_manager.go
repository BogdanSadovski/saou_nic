package api

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketManager manages real-time connections for sessions.
//
// Concurrency model:
//   - sessions/writeMu maps are guarded by `mu`.
//   - Each *websocket.Conn has a dedicated write mutex held while writing,
//     to satisfy gorilla/websocket's "only one goroutine writes at a time"
//     contract (concurrent broadcasts to the same conn would otherwise
//     corrupt frames or panic).
type WebSocketManager struct {
	sessions map[uuid.UUID]map[*websocket.Conn]struct{}
	writeMu  map[*websocket.Conn]*sync.Mutex
	mu       sync.RWMutex
	upgrader websocket.Upgrader
	logger   *logrus.Logger
}

// NewWebSocketManager creates a new WebSocket manager.
func NewWebSocketManager(logger *logrus.Logger) *WebSocketManager {
	return &WebSocketManager{
		sessions: make(map[uuid.UUID]map[*websocket.Conn]struct{}),
		writeMu:  make(map[*websocket.Conn]*sync.Mutex),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: logger,
	}
}

// connWriteLock returns the write mutex for the given connection, creating
// one if necessary. Safe to call concurrently.
func (wsm *WebSocketManager) connWriteLock(conn *websocket.Conn) *sync.Mutex {
	wsm.mu.RLock()
	if m, ok := wsm.writeMu[conn]; ok {
		wsm.mu.RUnlock()
		return m
	}
	wsm.mu.RUnlock()

	wsm.mu.Lock()
	defer wsm.mu.Unlock()
	if m, ok := wsm.writeMu[conn]; ok {
		return m
	}
	m := &sync.Mutex{}
	wsm.writeMu[conn] = m
	return m
}

// CollaborationWebSocket handles WebSocket connection for collaboration.
func (wsm *WebSocketManager) CollaborationWebSocket(sessionID uuid.UUID) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsm.upgrader.Upgrade(w, r, nil)
		if err != nil {
			wsm.logger.WithError(err).Error("failed to upgrade connection")
			return
		}
		defer conn.Close()

		userID := getUserIDFromContext(r.Context())
		wsm.registerClient(sessionID, conn)
		defer wsm.unregisterClient(sessionID, conn)

		wsm.logger.WithFields(logrus.Fields{
			"session_id": sessionID.String(),
			"user_id":    userID.String(),
		}).Info("client connected")

		if err := wsm.sendMessage(conn, map[string]interface{}{
			"type": "connected",
			"user": map[string]string{"id": userID.String()},
		}); err != nil {
			wsm.logger.WithError(err).Error("failed to send connected message")
			return
		}

		wsm.broadcast(sessionID, map[string]interface{}{
			"type":    "user_joined",
			"user_id": userID.String(),
		}, conn)

		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					wsm.logger.WithError(err).Error("websocket error")
				}
				break
			}

			msgType, ok := msg["type"].(string)
			if !ok {
				continue
			}

			switch msgType {
			case "note_update":
				wsm.handleNoteUpdate(sessionID, msg, conn)
			case "score_update":
				wsm.handleScoreUpdate(sessionID, msg, conn)
			case "typing":
				wsm.handleTyping(sessionID, msg, conn)
			case "ping":
				if err := wsm.sendMessage(conn, map[string]interface{}{"type": "pong"}); err != nil {
					wsm.logger.WithError(err).Error("failed to send pong")
				}
			default:
				wsm.logger.WithField("type", msgType).Debug("unknown message type")
			}
		}

		wsm.broadcast(sessionID, map[string]interface{}{
			"type":    "user_left",
			"user_id": userID.String(),
		}, conn)
	}
}

func (wsm *WebSocketManager) registerClient(sessionID uuid.UUID, conn *websocket.Conn) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if wsm.sessions[sessionID] == nil {
		wsm.sessions[sessionID] = make(map[*websocket.Conn]struct{})
	}
	wsm.sessions[sessionID][conn] = struct{}{}
}

func (wsm *WebSocketManager) unregisterClient(sessionID uuid.UUID, conn *websocket.Conn) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if clients, ok := wsm.sessions[sessionID]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(wsm.sessions, sessionID)
		}
	}
	delete(wsm.writeMu, conn)
}

func (wsm *WebSocketManager) broadcast(sessionID uuid.UUID, message interface{}, exclude *websocket.Conn) {
	wsm.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(wsm.sessions[sessionID]))
	for conn := range wsm.sessions[sessionID] {
		if conn != exclude {
			clients = append(clients, conn)
		}
	}
	wsm.mu.RUnlock()

	for _, conn := range clients {
		if err := wsm.sendMessage(conn, message); err != nil {
			wsm.logger.WithError(err).Warn("failed to broadcast websocket message")
		}
	}
}

func (wsm *WebSocketManager) broadcastAll(sessionID uuid.UUID, message interface{}) {
	wsm.broadcast(sessionID, message, nil)
}

func (wsm *WebSocketManager) sendMessage(conn *websocket.Conn, message interface{}) error {
	lock := wsm.connWriteLock(conn)
	lock.Lock()
	defer lock.Unlock()
	return conn.WriteJSON(message)
}

func (wsm *WebSocketManager) handleNoteUpdate(sessionID uuid.UUID, msg map[string]interface{}, sender *websocket.Conn) {
	wsm.broadcast(sessionID, map[string]interface{}{
		"type":      "note_update",
		"data":      msg["data"],
		"user":      msg["user"],
		"timestamp": msg["timestamp"],
	}, sender)

	wsm.logger.WithField("session_id", sessionID.String()).Debug("note update broadcasted")
}

func (wsm *WebSocketManager) handleScoreUpdate(sessionID uuid.UUID, msg map[string]interface{}, sender *websocket.Conn) {
	wsm.broadcast(sessionID, map[string]interface{}{
		"type":      "score_update",
		"data":      msg["data"],
		"user":      msg["user"],
		"timestamp": msg["timestamp"],
	}, sender)

	wsm.logger.WithField("session_id", sessionID.String()).Debug("score update broadcasted")
}

func (wsm *WebSocketManager) handleTyping(sessionID uuid.UUID, msg map[string]interface{}, sender *websocket.Conn) {
	wsm.broadcast(sessionID, map[string]interface{}{
		"type": "typing",
		"user": msg["user"],
	}, sender)
}

func (wsm *WebSocketManager) GetConnectionCount(sessionID uuid.UUID) int {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	return len(wsm.sessions[sessionID])
}

func (wsm *WebSocketManager) GetActiveSessions() map[string]int {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	result := make(map[string]int)
	for sessionID, clients := range wsm.sessions {
		result[sessionID.String()] = len(clients)
	}
	return result
}
