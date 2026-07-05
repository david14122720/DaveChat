package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ConfigHandler struct {
	TurnURL      string
	TurnUsername string
	TurnPassword string
}

func (h *ConfigHandler) GetWebRTCConfig(c echo.Context) error {
	iceServers := []map[string]interface{}{
		{"urls": "stun:stun.l.google.com:19302"},
		{"urls": "stun:stun1.l.google.com:19302"},
	}
	if h.TurnURL != "" {
		turnServer := map[string]interface{}{
			"urls":       h.TurnURL,
			"username":   h.TurnUsername,
			"credential": h.TurnPassword,
		}
		iceServers = append(iceServers, turnServer)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"iceServers": iceServers,
	})
}
