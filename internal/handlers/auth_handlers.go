package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zamibd/a2web/internal/auth"
	"github.com/zamibd/a2web/internal/database"
	"github.com/zamibd/a2web/internal/models"
)

type RegisterRequest struct {
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
}

func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Warn("Invalid register request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Mobile == "" || req.Password == "" {
		http.Error(w, "Mobile and password required", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.Logger.Error("Error hashing password", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = database.DB.Exec("INSERT INTO users (mobile, password_hash) VALUES (?, ?)", req.Mobile, hash)
	if err != nil {
		h.Logger.Warn("User registration failed", "mobile", req.Mobile, "error", err)
		http.Error(w, "User already exists or database error", http.StatusConflict)
		return
	}

	h.Logger.Info("User registered successfully", "mobile", req.Mobile)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	err := database.DB.QueryRow("SELECT id, password_hash, role FROM users WHERE mobile = ?", req.Mobile).Scan(&user.ID, &user.PasswordHash, &user.Role)
	if err != nil {
		h.Logger.Warn("Login failed: user not found", "mobile", req.Mobile)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		h.Logger.Warn("Login failed: invalid password", "mobile", req.Mobile)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(user.ID, string(user.Role))
	if err != nil {
		h.Logger.Error("Error generating JWT", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	h.Logger.Info("User logged in", "user_id", user.ID)
	w.Write([]byte("Logged in successfully"))
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})
	w.Write([]byte("Logged out"))
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		claims, err := auth.ValidateJWT(c.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check role if needed, or set context
		// For now just pass
		_ = claims
		next.ServeHTTP(w, r)
	}
}
