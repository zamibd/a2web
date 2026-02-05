package handlers

import (
	"net/http"
	"os"

	"github.com/zamibd/a2web/internal/auth"
	"github.com/zamibd/a2web/internal/database"
	"github.com/zamibd/a2web/internal/models"
)

func (h *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil {
			http.Redirect(w, r, "/login-page", http.StatusFound)
			return
		}

		claims, err := auth.ValidateJWT(c.Value)
		if err != nil || claims.Role != string(models.RoleAdmin) {
			h.Logger.Warn("Admin access denied", "error", err, "role", claims.Role)
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (h *Handler) AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// List Users
	rows, err := database.DB.Query("SELECT id, mobile, role, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		h.Logger.Error("DB Error fetching users", "error", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Mobile, &u.Role, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	// List Sessions
	rows2, err := database.DB.Query("SELECT id, user_id, name, status, created_at FROM sessions ORDER BY created_at DESC")
	if err != nil {
		h.Logger.Error("DB Error fetching sessions", "error", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	var sessions []models.Session
	for rows2.Next() {
		var s models.Session
		if err := rows2.Scan(&s.ID, &s.UserID, &s.Name, &s.Status, &s.CreatedAt); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	if err := h.Templates["admin.html"].ExecuteTemplate(w, "layout", map[string]interface{}{
		"Title":    "Admin Dashboard",
		"Users":    users,
		"Sessions": sessions,
	}); err != nil {
		h.Logger.Error("Template execution error", "template", "admin.html", "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (h *Handler) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// Delete from DB
	_, err := database.DB.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	// Delete File
	_ = os.Remove("./storage/" + sessionID + ".webm")
	h.Logger.Info("Session deleted", "id", sessionID)

	w.Write([]byte("")) // Return empty to remove element or refresh
}

func (h *Handler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// Delete from DB (Sessions should cascade or be handled manually if no Foreign Key cascade)
	// SQLite supports FK but needs PRAGMA foreign_keys = ON; usually.
	// For safety, let's delete sessions first.

	// Get session IDs to delete files
	rows, _ := database.DB.Query("SELECT id FROM sessions WHERE user_id = ?", userID)
	if rows != nil {
		for rows.Next() {
			var sid string
			rows.Scan(&sid)
			os.Remove("./storage/" + sid + ".webm")
		}
		rows.Close()
	}

	database.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		h.Logger.Error("DB Error deleting user", "error", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("User deleted", "id", userID)
	w.Write([]byte(""))
}
