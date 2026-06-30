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
		return true
	},
}

type WSMessage struct {
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload,omitempty"`
	RoomID   string          `json:"room_id,omitempty"`
	Target   string          `json:"target,omitempty"`
	CallType string          `json:"call_type,omitempty"`
}

func HandleWebSocket(hub *Hub, c echo.Context) error {
	userID := c.Get("user_id").(string)
	conn, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	client := &Client{UserID: userID, Send: make(chan []byte, 256), Hub: hub}
	hub.Register(client)
	client.Hub.TouchPresence(userID)
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
		client.Hub.TouchPresence(client.UserID)
		return nil
	})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		client.Hub.TouchPresence(client.UserID)
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
	hub := client.Hub
	switch msg.Type {
	case "signal:offer":
		msg.Type = "call-offer"
	case "signal:answer":
		msg.Type = "call-answer"
	case "signal:ice-candidate":
		msg.Type = "ice-candidate"
	case "call:reject":
		msg.Type = "call-reject"
	case "call:end", "call:busy":
		msg.Type = "call-end"
	}
	switch msg.Type {
	case "call-offer":
		target := msg.Target
		if target == "" || target == client.UserID {
			return
		}
		if !hub.IsOnline(target) {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "target user is offline"})
			select { case client.Send <- errMsg: default: }
			return
		}
		if hub.IsInCall(target) {
			busyMsg, _ := json.Marshal(map[string]string{"type": "call-busy", "from": target})
			select { case client.Send <- busyMsg: default: }
			return
		}
		callerID := client.UserID
		calleeID := target
		if msg.CallType != "" {
			hub.SetCallType(callerID, msg.CallType)
			hub.SetCallType(calleeID, msg.CallType)
		}
		hub.SetCallState(callerID, "calling")
		hub.SetCallState(calleeID, "ringing")
		hub.SetCallPeer(callerID, calleeID)
		hub.SetCallPeer(calleeID, callerID)
		relay, _ := json.Marshal(map[string]interface{}{"type": "call-offer", "payload": msg.Payload, "from": callerID, "call_type": msg.CallType})
		hub.SendToUser(calleeID, relay)
		hub.StartCallTimeout(calleeID, func() {
			if hub.GetCallState(calleeID) != "ringing" {
				return
			}
			hub.RecordCallEnded(callerID, "missed")
			timeoutMsg, _ := json.Marshal(map[string]string{"type": "call-timeout", "from": calleeID})
			hub.SendToUser(callerID, timeoutMsg)
			hub.ClearCallState(callerID)
			hub.ClearCallState(calleeID)
			hub.StopCallTimeout(calleeID)
			hub.ClearCallPeers(callerID)
			hub.ClearCallPeers(calleeID)
		})
	case "call-answer":
		calleeID := client.UserID
		callerID := msg.Target
		if callerID == "" {
			callerID = hub.GetCallPeer(calleeID)
		}
		if callerID == "" {
			return
		}
		hub.StopCallTimeout(calleeID)
		hub.SetCallState(calleeID, "connected")
		hub.SetCallState(callerID, "connected")
		startedToCaller, _ := json.Marshal(map[string]string{"type": "call-started", "from": calleeID})
		hub.SendToUser(callerID, startedToCaller)
		startedToCallee, _ := json.Marshal(map[string]string{"type": "call-started", "from": callerID})
		hub.SendToUser(calleeID, startedToCallee)
		hub.RecordCallStarted(callerID, calleeID)
		answerRelay, _ := json.Marshal(map[string]interface{}{"type": "call-answer", "payload": msg.Payload, "from": calleeID})
		hub.SendToUser(callerID, answerRelay)
	case "ice-candidate":
		target := msg.Target
		if target == "" {
			return
		}
		relay, _ := json.Marshal(map[string]interface{}{"type": "ice-candidate", "payload": msg.Payload, "from": client.UserID})
		hub.SendToUser(target, relay)
	case "call-end":
		userID := client.UserID
		peerID := hub.GetCallPeer(userID)
		if peerID == "" {
			return
		}
		hub.RecordCallEnded(userID, "completed")
		hub.StopCallTimeout(userID)
		hub.StopCallTimeout(peerID)
		hub.ClearCallState(userID)
		hub.ClearCallState(peerID)
		hub.ClearCallPeers(userID)
		hub.ClearCallPeers(peerID)
		endRelay, _ := json.Marshal(map[string]string{"type": "call-end", "from": userID})
		hub.SendToUser(peerID, endRelay)
	case "call-reject":
		userID := client.UserID
		peerID := hub.GetCallPeer(userID)
		if peerID == "" {
			return
		}
		hub.RecordCallEnded(userID, "rejected")
		hub.StopCallTimeout(userID)
		hub.StopCallTimeout(peerID)
		hub.ClearCallState(userID)
		hub.ClearCallState(peerID)
		hub.ClearCallPeers(userID)
		hub.ClearCallPeers(peerID)
		rejectRelay, _ := json.Marshal(map[string]string{"type": "call-reject", "from": userID})
		hub.SendToUser(peerID, rejectRelay)
	case "signal:join-room":
		if msg.RoomID != "" {
			hub.JoinRoom(msg.RoomID, client.UserID)
			payload, _ := json.Marshal(map[string]string{"type": "user:joined", "user_id": client.UserID, "room_id": msg.RoomID})
			hub.SendToRoom(msg.RoomID, client.UserID, payload)
		}
	case "signal:leave-room":
		if msg.RoomID != "" {
			hub.LeaveRoom(msg.RoomID, client.UserID)
			payload, _ := json.Marshal(map[string]string{"type": "user:left", "user_id": client.UserID, "room_id": msg.RoomID})
			hub.SendToRoom(msg.RoomID, client.UserID, payload)
		}
	case "ping":
		pong, _ := json.Marshal(map[string]string{"type": "pong"})
		select { case client.Send <- pong: default: }
	}
}
