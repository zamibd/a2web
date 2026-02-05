package handlers

import (
	"net/http"

	"github.com/zamibd/a2web/internal/auth"
	"github.com/zamibd/a2web/internal/database"
	"github.com/zamibd/a2web/internal/models"
)

func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from token (Middleware should have validated it, but we need the claims)
	c, _ := r.Cookie("token")
	claims, _ := auth.ValidateJWT(c.Value)

	rows, err := database.DB.Query("SELECT id, name, status, created_at FROM sessions WHERE user_id = ? ORDER BY created_at DESC", claims.UserID)
	if err != nil {
		h.Logger.Error("Database error fetching sessions", "error", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.CreatedAt); err != nil {
			h.Logger.Error("Row scan error", "error", err)
			continue
		}
		sessions = append(sessions, s)
	}

	if err := h.Templates["dashboard.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title":    "Dashboard",
		"Sessions": sessions,
	}); err != nil {
		h.Logger.Error("Template execution error", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (h *Handler) CreateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	c, _ := r.Cookie("token")
	claims, _ := auth.ValidateJWT(c.Value)

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		h.Logger.Error("Error generating session ID", "error", err)
		http.Error(w, "Error generating ID", http.StatusInternalServerError)
		return
	}

	// Simple name for now, or get from request
	name := "Session " + sessionID[:8]

	_, err = database.DB.Exec("INSERT INTO sessions (id, user_id, name) VALUES (?, ?, ?)", sessionID, claims.UserID, name)
	if err != nil {
		h.Logger.Error("Database error creating session", "error", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return just the new row for HTMX to prepend
	// For now, let's just redirect or return a simple fragment
	// Let's create a partial template or just return HTML string for simplicity if "HX-Request" header exists
	// But let's stick to full page reload or simple fragment for now.

	// HTMX response: render just the list item
	w.Header().Set("Content-Type", "text/html")
	// TODO: Use a proper template fragment
	w.Write([]byte(`<li><a href="/user/` + sessionID + `">` + name + `</a> (Active)</li>`))
}
