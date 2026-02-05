package handlers

import (
	"net/http"

	"github.com/zamibd/a2web/internal/auth"
	"github.com/zamibd/a2web/internal/database"
)

func (h *Handler) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.Templates["login.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title": "Login",
	}); err != nil {
		h.Logger.Error("Template execution error", "template", "login.html", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (h *Handler) RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.Templates["register.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title": "Register",
	}); err != nil {
		h.Logger.Error("Template execution error", "template", "register.html", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (h *Handler) KidsPageHandler(w http.ResponseWriter, r *http.Request) {
	// Public page, no auth needed? Or maybe just session validation.
	// For now, assume public access if they have the link.

	// Get Session ID from URL
	// URL: /kids/{session_id}
	// Note: r.URL.Path logic might be better handled by a router like Chi/Mux, but standard lib is fine.

	sessionID := r.URL.Path[len("/kids/"):]

	// Validate session exists
	var name string
	err := database.DB.QueryRow("SELECT name FROM sessions WHERE id = ?", sessionID).Scan(&name)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if err := h.Templates["kids.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title":       "Live Mic",
		"SessionID":   sessionID,
		"SessionName": name,
	}); err != nil {
		h.Logger.Error("Template execution error", "template", "kids.html", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (h *Handler) ParentPageHandler(w http.ResponseWriter, r *http.Request) {
	// Protected by middleware usually
	c, _ := r.Cookie("token")
	claims, _ := auth.ValidateJWT(c.Value)

	sessionID := r.URL.Path[len("/user/"):]

	// Verify ownership
	var userID int64
	err := database.DB.QueryRow("SELECT user_id FROM sessions WHERE id = ?", sessionID).Scan(&userID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if userID != claims.UserID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := h.Templates["parent.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title":     "Monitor",
		"SessionID": sessionID,
	}); err != nil {
		h.Logger.Error("Template execution error", "template", "parent.html", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}
