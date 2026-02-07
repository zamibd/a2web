package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/zamibd/a2web/internal/auth"
	"github.com/zamibd/a2web/internal/config"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return config.AppConfig.IsOriginAllowed(origin)
	},
}

func (h *Handler) KidWSHandler(w http.ResponseWriter, r *http.Request) {
	// URL: /ws/kid/{session_id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	sessionID := pathParts[3]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("Upgrade error", "error", err)
		return
	}
	defer conn.Close()

	// Open file for appending audio
	// Ensure directory exists
	// os.MkdirAll("./storage", 0755) // Already done in main

	filePath := "./storage/" + sessionID + ".webm"
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		h.Logger.Error("File open error", "error", err)
		return // Should probably close conn too
	}
	defer f.Close()

	isFirstChunk := true
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			h.Logger.Warn("Read error", "error", err)
			break
		}

		if messageType == websocket.BinaryMessage {
			// 0. Cache Init Segment (First Chunk)
			if isFirstChunk {
				GlobalHub.SetInitSegment(sessionID, p)
				isFirstChunk = false
			}

			// 1. Save to disk
			if _, err := f.Write(p); err != nil {
				h.Logger.Error("File write error", "error", err)
			}

			// 2. Relay to parent
			parentConn := GlobalHub.GetParent(sessionID)
			if parentConn != nil {
				// Prevent blocking if parent is slow
				// For now simple write, in production use channel/buffer
				if err := parentConn.WriteMessage(websocket.BinaryMessage, p); err != nil {
					h.Logger.Error("Relay error", "error", err)
					// Maybe unregister parent if write fails?
				}
			}
		}
	}
}

func (h *Handler) ParentWSHandler(w http.ResponseWriter, r *http.Request) {
	// URL: /ws/parent/{session_id}
	// Auth check should be done before or via cookie check here
	c, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Validate token... (skipping full validation for brevity, assuming middleware protected or simple check)
	// Actually, WS upgrade happens before middleware can wrapp properly sometimes, so let's valid here.
	if _, err := auth.ValidateJWT(c.Value); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	sessionID := pathParts[3]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("Upgrade error", "error", err)
		return
	}

	GlobalHub.RegisterParent(sessionID, conn)
	defer GlobalHub.UnregisterParent(sessionID)

	// Send Init Segment if available (Critical for late joiners)
	initSeg := GlobalHub.GetInitSegment(sessionID)
	if initSeg != nil {
		if err := conn.WriteMessage(websocket.BinaryMessage, initSeg); err != nil {
			h.Logger.Error("Init segment write error", "error", err)
			return
		}
	}

	// Keep connection alive, maybe read control messages?
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
