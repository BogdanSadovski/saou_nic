package websocket

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	MessageTypeCode       = "code_change"
	MessageTypeCursor     = "cursor_move"
	MessageTypeChat       = "chat_message"
	MessageTypeSystem     = "system"
	MessageTypeJoin       = "join"
	MessageTypeLeave      = "leave"
	MessageTypeRunTests   = "run_tests"
	MessageTypeTestResult = "test_result"
)

type Room struct {
	ID         uuid.UUID
	Clients    map[uuid.UUID]*Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
	Logger     *logrus.Logger
}

func NewRoom(id uuid.UUID, logger *logrus.Logger) *Room {
	return &Room{
		ID:         id,
		Clients:    make(map[uuid.UUID]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
		Logger:     logger,
	}
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.Register:
			r.Clients[client.ID] = client
			r.Logger.WithFields(logrus.Fields{
				"room_id":   r.ID,
				"client_id": client.ID,
				"clients":   len(r.Clients),
			}).Info("client registered")

			// Notify other clients about the join
			r.broadcastJoin(client)

		case client := <-r.Unregister:
			if _, ok := r.Clients[client.ID]; ok {
				delete(r.Clients, client.ID)
				close(client.Send)
				r.Logger.WithFields(logrus.Fields{
					"room_id":   r.ID,
					"client_id": client.ID,
					"clients":   len(r.Clients),
				}).Info("client unregistered")

				// Notify other clients about the leave
				r.broadcastLeave(client)

				// Clean up room if no clients left
				if len(r.Clients) == 0 {
					r.Logger.WithField("room_id", r.ID).Info("room empty, shutting down")
					return
				}
			}

		case message := <-r.Broadcast:
			// Route message based on type
			r.handleMessage(message)
		}
	}
}

func (r *Room) handleMessage(msg *Message) {
	r.Logger.WithFields(logrus.Fields{
		"room_id":   r.ID,
		"client_id": msg.ClientID,
		"type":      msg.Type,
	}).Debug("handling message")

	switch msg.Type {
	case MessageTypeCode:
		// Broadcast code changes to all other clients
		r.broadcastToOthers(msg.ClientID, msg)

	case MessageTypeCursor:
		// Broadcast cursor position to others
		r.broadcastToOthers(msg.ClientID, msg)

	case MessageTypeChat:
		// Broadcast chat messages to all clients (including sender)
		r.broadcastToAll(msg)

	case MessageTypeRunTests:
		// Forward to test runner service (placeholder)
		r.broadcastToOthers(msg.ClientID, &Message{
			Type:    MessageTypeSystem,
			Payload: json.RawMessage(`{"status": "running", "message": "Tests are being executed..."}`),
		})

	default:
		r.Logger.WithField("type", msg.Type).Warn("unknown message type")
	}
}

func (r *Room) broadcastJoin(client *Client) {
	payload, _ := json.Marshal(map[string]interface{}{
		"client_id": client.ID.String(),
		"user_id":   client.UserID.String(),
		"is_coder":  client.IsCoder,
	})

	msg := &Message{
		Type:    MessageTypeJoin,
		Payload: json.RawMessage(payload),
	}
	r.broadcastToAll(msg)
}

func (r *Room) broadcastLeave(client *Client) {
	payload, _ := json.Marshal(map[string]interface{}{
		"client_id": client.ID.String(),
		"user_id":   client.UserID.String(),
	})

	msg := &Message{
		Type:    MessageTypeLeave,
		Payload: json.RawMessage(payload),
	}
	r.broadcastToAll(msg)
}

func (r *Room) broadcastToOthers(senderID uuid.UUID, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		r.Logger.WithError(err).Error("failed to marshal message")
		return
	}

	for id, client := range r.Clients {
		if id == senderID {
			continue
		}
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(r.Clients, id)
		}
	}
}

func (r *Room) broadcastToAll(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		r.Logger.WithError(err).Error("failed to marshal message")
		return
	}

	for id, client := range r.Clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(r.Clients, id)
		}
	}
}
