package websocket

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type Client struct {
	UserID string
	Send   chan []byte
	Hub    *Hub
	Conn   interface{}
	Rooms  map[string]bool
}

type Hub struct {
	db             *sql.DB
	mu             sync.RWMutex
	clients        map[string]*Client
	rooms          map[string]map[string]*Client
	callStates     map[string]string
	CallTimeouts   map[string]*time.Timer
	callPeers      map[string]string
	callStartTimes map[string]time.Time
	callTypes          map[string]string
	lastPresenceTouch  map[string]time.Time
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		db:             db,
		clients:        make(map[string]*Client),
		rooms:          make(map[string]map[string]*Client),
		callStates:     make(map[string]string),
		CallTimeouts:   make(map[string]*time.Timer),
		callPeers:      make(map[string]string),
		callStartTimes: make(map[string]time.Time),
		callTypes:         make(map[string]string),
		lastPresenceTouch: make(map[string]time.Time),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	if old, ok := h.clients[client.UserID]; ok {
		close(old.Send)
	}
	h.clients[client.UserID] = client
	h.setPresence(client.UserID, "online")
	h.mu.Unlock()

	h.broadcastPresence(client.UserID, "online", client)
}

func (h *Hub) Unregister(client *Client) {
	var peerClient *Client
	var peerID string
	var notifyPeer bool
	var wasInCall bool
	var callPeerID string
	var callType string

	h.mu.Lock()
	if existing, ok := h.clients[client.UserID]; ok && existing == client {
		delete(h.clients, client.UserID)
		h.setPresence(client.UserID, "offline")
		delete(h.lastPresenceTouch, client.UserID)
	}
	for roomID := range client.Rooms {
		if members, ok := h.rooms[roomID]; ok {
			delete(members, client.UserID)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
	if pid, ok := h.callPeers[client.UserID]; ok {
		peerID = pid
		notifyPeer = true
		wasInCall = true
		callPeerID = pid
		callType = h.callTypes[client.UserID]

		if pc, ok := h.clients[peerID]; ok {
			peerClient = pc
		}
		h.callStates[peerID] = "idle"
		delete(h.callPeers, peerID)
		delete(h.callPeers, client.UserID)
		delete(h.callStartTimes, client.UserID)
		delete(h.callStartTimes, peerID)
		delete(h.callTypes, client.UserID)
		delete(h.callTypes, peerID)
		if t, ok := h.CallTimeouts[peerID]; ok {
			t.Stop()
			delete(h.CallTimeouts, peerID)
		}
		if t, ok := h.CallTimeouts[client.UserID]; ok {
			t.Stop()
			delete(h.CallTimeouts, client.UserID)
		}
	}
	delete(h.callStates, client.UserID)
	h.mu.Unlock()

	h.broadcastPresence(client.UserID, "offline", nil)

	if wasInCall && h.db != nil {
		id := newUUID()
		now := time.Now()
		if callType == "" {
			callType = "audio"
		}
		if _, err := h.db.Exec(`
			INSERT INTO call_logs (id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs)
			VALUES (?, ?, ?, ?, 'cancelled', ?, ?, 0)
		`, id, callPeerID, client.UserID, callType, now, now); err != nil {
			log.Printf("Error recording cancelled call log: %v", err)
		}
	}

	if notifyPeer && peerClient != nil {
		msg, _ := json.Marshal(map[string]string{
			"type":   "call-end",
			"from":   client.UserID,
			"reason": "disconnected",
		})
		select {
		case peerClient.Send <- msg:
		default:
		}
	}
}

func (h *Hub) broadcastPresence(userID string, status string, excludeClient *Client) {
	msg := map[string]interface{}{
		"type":      "presence_update",
		"user_id":   userID,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}
	msgBytes, _ := json.Marshal(msg)

	h.mu.RLock()
	for _, client := range h.clients {
		if client != excludeClient {
			select {
			case client.Send <- msgBytes:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) setPresence(userID, status string) {
	if h.db == nil {
		return
	}
	if _, err := h.db.Exec("UPDATE profiles SET status = ?, last_seen = NOW(3) WHERE id = ?", status, userID); err != nil {
		log.Printf("presence update failed for user %s: %v", userID, err)
	}
}

func (h *Hub) TouchPresence(userID string) {
	h.mu.Lock()
	_, online := h.clients[userID]
	if !online {
		h.mu.Unlock()
		return
	}
	if last, ok := h.lastPresenceTouch[userID]; ok && time.Since(last) < 30*time.Second {
		h.mu.Unlock()
		return
	}
	h.lastPresenceTouch[userID] = time.Now()
	h.mu.Unlock()
	h.setPresence(userID, "online")
}

func (h *Hub) ResetPresence() {
	if h.db == nil {
		return
	}
	if _, err := h.db.Exec("UPDATE profiles SET status = 'offline' WHERE status = 'online'"); err != nil {
		log.Printf("presence reset failed: %v", err)
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
		client.Rooms[roomID] = true
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
	if client, ok := h.clients[userID]; ok {
		delete(client.Rooms, roomID)
	}
}

func (h *Hub) SendToUser(userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if client, ok := h.clients[userID]; ok {
		select {
		case client.Send <- msg:
		default:
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

func (h *Hub) SetCallState(userID, state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callStates[userID] = state
}

func (h *Hub) GetCallState(userID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	state, ok := h.callStates[userID]
	if !ok {
		return "idle"
	}
	return state
}

func (h *Hub) IsInCall(userID string) bool {
	state := h.GetCallState(userID)
	return state == "calling" || state == "connected"
}

func (h *Hub) ClearCallState(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callStates[userID] = "idle"
}

func (h *Hub) StartCallTimeout(userID string, callback func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.CallTimeouts[userID]; ok {
		t.Stop()
	}
	h.CallTimeouts[userID] = time.AfterFunc(30*time.Second, callback)
}

func (h *Hub) StopCallTimeout(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.CallTimeouts[userID]; ok {
		t.Stop()
		delete(h.CallTimeouts, userID)
	}
}

func (h *Hub) SetCallPeer(userID, peerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callPeers[userID] = peerID
}

func (h *Hub) GetCallPeer(userID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.callPeers[userID]
}

func (h *Hub) ClearCallPeers(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if peerID, ok := h.callPeers[userID]; ok {
		delete(h.callPeers, peerID)
	}
	delete(h.callPeers, userID)
}

func (h *Hub) SetCallType(userID, callType string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callTypes[userID] = callType
}

func (h *Hub) GetCallType(userID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ct, ok := h.callTypes[userID]
	if !ok {
		return "audio"
	}
	return ct
}

func (h *Hub) RecordCallStarted(callerID, calleeID string) {
	if h.db == nil {
		return
	}

	h.mu.Lock()
	now := time.Now()
	h.callStartTimes[callerID] = now
	h.callStartTimes[calleeID] = now
	callType := h.callTypes[callerID]
	h.mu.Unlock()

	id := newUUID()
	_, err := h.db.Exec(`
		INSERT INTO call_logs (id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs)
		VALUES (?, ?, ?, ?, 'completed', ?, NULL, 0)
	`, id, callerID, calleeID, callType, now)
	if err != nil {
		log.Printf("Error inserting call log: %v", err)
	}
}

func (h *Hub) RecordCallEnded(userID string, status string) {
	if h.db == nil {
		return
	}

	h.mu.Lock()
	startedAt, hasStart := h.callStartTimes[userID]
	peerID := h.callPeers[userID]
	callType := h.callTypes[userID]

	delete(h.callStartTimes, userID)
	if peerID != "" {
		delete(h.callStartTimes, peerID)
	}
	delete(h.callTypes, userID)
	if peerID != "" {
		delete(h.callTypes, peerID)
	}
	h.mu.Unlock()

	now := time.Now()

	if hasStart && peerID != "" {
		duration := int(time.Since(startedAt).Seconds())
		_, err := h.db.Exec(`
			UPDATE call_logs
			SET ended_at = ?, duration_secs = ?, status = ?
			WHERE (caller_id = ? OR callee_id = ?) AND ended_at IS NULL
			ORDER BY started_at DESC LIMIT 1
		`, now, duration, status, userID, userID)
		if err != nil {
			log.Printf("Error updating call log: %v", err)
		}
		return
	}

	if peerID == "" {
		return
	}

	var callerID, calleeID string
	if status == "missed" {
		callerID = userID
		calleeID = peerID
	} else {
		callerID = peerID
		calleeID = userID
	}

	id := newUUID()
	_, err := h.db.Exec(`
		INSERT INTO call_logs (id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0)
	`, id, callerID, calleeID, callType, status, now, now)
	if err != nil {
		log.Printf("Error inserting call log: %v", err)
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate UUID: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
