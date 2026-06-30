package websocket

import (
	"sync"
)

// Client represents a connected WebSocket client
type Client struct {
	UserID string
	Send   chan []byte
	Hub    *Hub
	Conn   interface{} // will be set by the actual WebSocket connection
}

// Hub manages all WebSocket connections and room memberships
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client               // userID -> client (single connection per user for now)
	rooms   map[string]map[string]*Client     // roomID -> userID -> client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		rooms:   make(map[string]map[string]*Client),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// If user already has a connection, close the old one
	if old, ok := h.clients[client.UserID]; ok {
		close(old.Send)
	}
	h.clients[client.UserID] = client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.clients[client.UserID]; ok && existing == client {
		delete(h.clients, client.UserID)
	}
	// Remove from all rooms
	for roomID, members := range h.rooms {
		if _, ok := members[client.UserID]; ok {
			delete(members, client.UserID)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
}

func (h *Hub) JoinRoom(roomID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string]*Client)
	}
	if client, ok := h.clients[userID]; ok {
		h.rooms[roomID][userID] = client
	}
}

func (h *Hub) LeaveRoom(roomID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.rooms[roomID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) SendToUser(userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if client, ok := h.clients[userID]; ok {
		select {
		case client.Send <- msg:
		default:
			// Drop if client's send buffer is full
		}
	}
}

func (h *Hub) SendToRoom(roomID string, senderID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if members, ok := h.rooms[roomID]; ok {
		for userID, client := range members {
			if userID != senderID {
				select {
				case client.Send <- msg:
				default:
				}
			}
		}
	}
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}
