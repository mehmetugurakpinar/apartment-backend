package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EventType string

const (
	EventNewNotification   EventType = "new_notification"
	EventMaintenanceUpdate EventType = "maintenance_update"
	EventNewForumPost      EventType = "new_forum_post"
	EventNewTimelinePost   EventType = "new_timeline_post"
	EventPaymentConfirmed  EventType = "payment_confirmed"
	EventEmergencyAlert    EventType = "emergency_alert"
)

type WSMessage struct {
	Event EventType   `json:"event"`
	Data  interface{} `json:"data"`
}

type Client struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	BuildingID uuid.UUID
	Conn       *websocket.Conn
	Send       chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[uuid.UUID]*Client
	buildings  map[uuid.UUID]map[uuid.UUID]*Client // buildingID -> clientID -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	logger     *zap.Logger
}

type BroadcastMessage struct {
	BuildingID  uuid.UUID
	Message     WSMessage
	ExcludeUser *uuid.UUID
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		buildings:  make(map[uuid.UUID]map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if _, ok := h.buildings[client.BuildingID]; !ok {
				h.buildings[client.BuildingID] = make(map[uuid.UUID]*Client)
			}
			h.buildings[client.BuildingID][client.ID] = client
			h.mu.Unlock()

			h.logger.Info("WebSocket client connected",
				zap.String("user_id", client.UserID.String()),
				zap.String("building_id", client.BuildingID.String()),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if buildingClients, ok := h.buildings[client.BuildingID]; ok {
					delete(buildingClients, client.ID)
					if len(buildingClients) == 0 {
						delete(h.buildings, client.BuildingID)
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()

			h.logger.Info("WebSocket client disconnected",
				zap.String("user_id", client.UserID.String()),
			)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.buildings[msg.BuildingID]; ok {
				data, err := json.Marshal(msg.Message)
				if err != nil {
					h.logger.Error("Failed to marshal WS message", zap.Error(err))
					h.mu.RUnlock()
					continue
				}

				for _, client := range clients {
					if msg.ExcludeUser != nil && client.UserID == *msg.ExcludeUser {
						continue
					}
					select {
					case client.Send <- data:
					default:
						h.mu.RUnlock()
						h.mu.Lock()
						delete(h.clients, client.ID)
						if buildingClients, ok := h.buildings[client.BuildingID]; ok {
							delete(buildingClients, client.ID)
						}
						close(client.Send)
						h.mu.Unlock()
						h.mu.RLock()
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) BroadcastToBuilding(buildingID uuid.UUID, event EventType, data interface{}, excludeUser *uuid.UUID) {
	h.broadcast <- &BroadcastMessage{
		BuildingID:  buildingID,
		Message:     WSMessage{Event: event, Data: data},
		ExcludeUser: excludeUser,
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, event EventType, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msg, err := json.Marshal(WSMessage{Event: event, Data: data})
	if err != nil {
		return
	}

	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- msg:
			default:
			}
		}
	}
}

func WritePump(client *Client) {
	defer client.Conn.Close()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func ReadPump(hub *Hub, client *Client) {
	defer func() {
		hub.Unregister(client)
		client.Conn.Close()
	}()

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}
		// Currently we don't process incoming WS messages from clients
		// This could be extended for chat functionality
	}
}
