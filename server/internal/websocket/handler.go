package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket
	},
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	RoomID  string          `json:"room_id,omitempty"`
	Target  string          `json:"target,omitempty"`
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and manages the client lifecycle.
// Uses Echo context to extract user_id and upgrade the connection.
func HandleWebSocket(hub *Hub, c echo.Context) error {
	userID := c.Get("user_id").(string)

	conn, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}

	client := &Client{
		UserID: userID,
		Send:   make(chan []byte, 256),
		Hub:    hub,
	}

	hub.Register(client)

	// Start read/write pumps
	go writePump(client, conn)
	go readPump(client, conn)

	return nil
}

func readPump(client *Client, conn *websocket.Conn) {
	defer func() {
		client.Hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		handleMessage(client, msg)
	}
}

func writePump(client *Client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func handleMessage(client *Client, msg WSMessage) {
	switch msg.Type {
	case "signal:offer", "signal:answer", "signal:ice-candidate":
		// Relay signaling message to target user
		if msg.Target != "" {
			payload, _ := json.Marshal(map[string]interface{}{
				"type":    msg.Type,
				"payload": msg.Payload,
				"from":    client.UserID,
			})
			client.Hub.SendToUser(msg.Target, payload)
		}

	case "signal:join-room":
		if msg.RoomID != "" {
			client.Hub.JoinRoom(msg.RoomID, client.UserID)
			// Notify room members
			payload, _ := json.Marshal(map[string]string{
				"type":    "user:joined",
				"user_id": client.UserID,
				"room_id": msg.RoomID,
			})
			client.Hub.SendToRoom(msg.RoomID, client.UserID, payload)
		}

	case "signal:leave-room":
		if msg.RoomID != "" {
			client.Hub.LeaveRoom(msg.RoomID, client.UserID)
			payload, _ := json.Marshal(map[string]string{
				"type":    "user:left",
				"user_id": client.UserID,
				"room_id": msg.RoomID,
			})
			client.Hub.SendToRoom(msg.RoomID, client.UserID, payload)
		}

	case "call:reject", "call:busy", "call:end":
		if msg.Target != "" {
			payload, _ := json.Marshal(map[string]interface{}{
				"type": msg.Type,
				"from": client.UserID,
			})
			client.Hub.SendToUser(msg.Target, payload)
		}

	case "ping":
		pong, _ := json.Marshal(map[string]string{"type": "pong"})
		select {
		case client.Send <- pong:
		default:
		}
	}
}
