package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type ClientMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID       uuid.UUID
	RoomID   uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	UserID   uuid.UUID
	IsCoder  bool // true if this client can write code
	Logger   *logrus.Logger
}

func NewClient(id, roomID, userID uuid.UUID, conn *websocket.Conn, isCoder bool, logger *logrus.Logger) *Client {
	return &Client{
		ID:      id,
		RoomID:  roomID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		UserID:  userID,
		IsCoder: isCoder,
		Logger:  logger,
	}
}

func (c *Client) ReadPump(room *Room) {
	defer func() {
		c.Logger.WithFields(logrus.Fields{
			"client_id": c.ID,
			"room_id":   c.RoomID,
		}).Info("client disconnecting")
		room.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.Logger.WithError(err).Error("websocket unexpected close")
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Logger.WithError(err).Warn("failed to parse client message")
			continue
		}

		room.Broadcast <- &Message{
			ClientID: c.ID,
			Type:     msg.Type,
			Payload:  msg.Payload,
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			c.Logger.WithError(err).Error("failed to write message")
			return
		}
	}
}

type Handler struct {
	upgrader websocket.Upgrader
	rooms    map[uuid.UUID]*Room
	mu       sync.RWMutex
	logger   *logrus.Logger
}

func NewHandler(logger *logrus.Logger) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// In production, validate origin properly
				return true
			},
		},
		rooms:  make(map[uuid.UUID]*Room),
		logger: logger,
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract parameters
	roomIDStr := r.URL.Query().Get("room_id")
	userIDStr := r.URL.Query().Get("user_id")
	isCoder := r.URL.Query().Get("role") == "coder"

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "invalid room_id", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Error("failed to upgrade connection")
		return
	}

	client := NewClient(
		uuid.New(),
		roomID,
		userID,
		conn,
		isCoder,
		h.logger,
	)

	// Get or create room
	room := h.getOrCreateRoom(roomID)
	room.Register <- client

	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"room_id":   roomID,
		"user_id":   userID,
	}).Info("new websocket client connected")

	go client.WritePump()
	go client.ReadPump(room)
}

func (h *Handler) getOrCreateRoom(roomID uuid.UUID) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[roomID]
	if !exists {
		room = NewRoom(roomID, h.logger)
		h.rooms[roomID] = room
		go room.Run()
	}

	return room
}

func (h *Handler) RemoveRoom(roomID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, roomID)
}

// Message represents a message within a room
type Message struct {
	ClientID uuid.UUID       `json:"client_id"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}
