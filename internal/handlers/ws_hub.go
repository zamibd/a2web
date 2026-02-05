package handlers

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to the parent.
type Hub struct {
	// Registered clients.
	// Map sessionID -> Parent Connection (User)
	parents map[string]*websocket.Conn

	// Map sessionID -> Kid Connection (just for tracking if needed, but mainly we just read from it)
	// Actually we might handle kids in a simple handler that looks up the parent in this Hub.

	mu sync.RWMutex
}

var GlobalHub = Hub{
	parents: make(map[string]*websocket.Conn),
}

func (h *Hub) RegisterParent(sessionID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing if any (single parent per session for simplicity as per requirements "logged-in user hears")
	if existing, ok := h.parents[sessionID]; ok {
		existing.Close()
	}
	h.parents[sessionID] = conn
}

func (h *Hub) UnregisterParent(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, ok := h.parents[sessionID]; ok {
		conn.Close()
		delete(h.parents, sessionID)
	}
}

func (h *Hub) GetParent(sessionID string) *websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.parents[sessionID]
}
