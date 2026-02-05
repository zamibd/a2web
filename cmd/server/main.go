package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zamibd/a2web/internal/config"
	"github.com/zamibd/a2web/internal/database"
	"github.com/zamibd/a2web/internal/handlers"
	"github.com/zamibd/a2web/internal/middleware"
	"golang.org/x/time/rate"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Initialize Database
	if err := os.MkdirAll("./storage", 0755); err != nil {
		logger.Error("Failed to create storage directory", "error", err)
		os.Exit(1)
	}
	database.InitDB(config.AppConfig.DBPath)
	logger.Info("Database initialized", "path", config.AppConfig.DBPath)

	// 3. Parse Templates
	// Parse layout first
	layoutTmpl, err := template.ParseFiles("web/templates/layout.html")
	if err != nil {
		logger.Error("Failed to parse layout", "error", err)
		os.Exit(1)
	}

	// Define pages to pre-build
	pages := []string{
		"login.html", "register.html", "dashboard.html",
		"kids.html", "parent.html", "admin.html",
	}

	templateMap := make(map[string]*template.Template)

	for _, page := range pages {
		// Clone the layout for each page to prevent namespace pollution
		clone, err := layoutTmpl.Clone()
		if err != nil {
			logger.Error("Failed to clone layout", "page", page, "error", err)
			os.Exit(1)
		}

		// Parse the specific page file into the clone
		_, err = clone.ParseFiles("web/templates/" + page)
		if err != nil {
			logger.Error("Failed to parse page template", "page", page, "error", err)
			os.Exit(1)
		}

		templateMap[page] = clone
	}

	// 4. Initialize Handlers
	h := handlers.New(logger, templateMap)

	// 5. Setup Router & Middleware
	mux := http.NewServeMux()
	mw := middleware.New(logger)

	// Admin Routes
	mux.HandleFunc("/admin", h.AdminMiddleware(h.AdminDashboardHandler))
	mux.HandleFunc("/admin/user/delete", h.AdminMiddleware(h.DeleteUserHandler))
	mux.HandleFunc("/admin/session/delete", h.AdminMiddleware(h.DeleteSessionHandler))

	// Public Routes (Auth)
	// Apply rate limiting to login
	loginLimiter := mw.RateLimit(rate.Every(1*time.Minute/5), 5) // 5 requests per minute
	mux.Handle("/login", loginLimiter(http.HandlerFunc(h.LoginHandler)))
	mux.HandleFunc("/register", h.RegisterHandler)
	mux.HandleFunc("/logout", h.LogoutHandler)

	// Protected Routes
	mux.HandleFunc("/dashboard", handlers.AuthMiddleware(h.DashboardHandler))
	mux.HandleFunc("/session/create", handlers.AuthMiddleware(h.CreateSessionHandler))
	mux.HandleFunc("/user/", handlers.AuthMiddleware(h.ParentPageHandler))

	// Public Routes (Pages)
	mux.HandleFunc("/login-page", h.LoginPageHandler)
	mux.HandleFunc("/register-page", h.RegisterPageHandler)
	mux.HandleFunc("/kids/", h.KidsPageHandler)

	// WebSocket Routes
	mux.HandleFunc("/ws/kid/", h.KidWSHandler)       // Public, maybe protect with simple token later?
	mux.HandleFunc("/ws/parent/", h.ParentWSHandler) // Protected by cookie check inside

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Audio Streamer Backend Running"))
	})

	// Wrap mux with global logging middleware
	finalHandler := mw.Logging(mux)

	srv := &http.Server{
		Addr:    ":" + config.AppConfig.Port,
		Handler: finalHandler,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("Server started", "port", config.AppConfig.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		logger.Error("Error starting server", "error", err)

	case <-shutdown:
		logger.Info("Starting shutdown...")

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown did not complete", "error", err)
			if err := srv.Close(); err != nil {
				logger.Error("Could not stop http server", "error", err)
			}
		}
	}
}
