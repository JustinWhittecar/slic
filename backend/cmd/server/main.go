package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"context"
	"time"
	"strings"

	"github.com/JustinWhittecar/slic/internal/db"
	"github.com/JustinWhittecar/slic/internal/handlers"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dbPath := os.Getenv("SLIC_DB_PATH")
	if dbPath == "" {
		dbPath = "slic.db"
	}

	sqlDB, err := db.ConnectSQLite(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer sqlDB.Close()

	// User DB (writable, separate from read-only mech DB)
	userDBPath := os.Getenv("SLIC_USER_DB_PATH")
	if userDBPath == "" {
		userDBPath = "users.db"
	}
	userDB, err := db.ConnectUserDB(userDBPath)
	if err != nil {
		log.Fatalf("Failed to connect to user DB: %v", err)
	}
	defer userDB.Close()

	mechHandler := &handlers.MechHandlerSQLite{DB: sqlDB}
	feedbackHandler := handlers.NewFeedbackHandler()
	authHandler := handlers.NewAuthHandler(userDB)
	collectionHandler := &handlers.CollectionHandler{DB: userDB, MecDB: sqlDB}
	listsHandler := &handlers.ListsHandler{DB: userDB}
	modelsHandler := &handlers.ModelsHandler{DB: sqlDB}
	preferencesHandler := &handlers.PreferencesHandler{DB: userDB}
	eventsHandler := handlers.NewEventsHandler(userDB)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Mech API
	mux.HandleFunc("GET /api/mechs", mechHandler.List)
	mux.HandleFunc("GET /api/mechs/{id}", mechHandler.GetByID)

	// Feedback
	mux.HandleFunc("POST /api/feedback", feedbackHandler.Submit)

	// Auth
	mux.HandleFunc("GET /api/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("GET /api/auth/callback", authHandler.Callback)
	mux.HandleFunc("GET /api/auth/me", authHandler.Me)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)

	// Physical models
	mux.HandleFunc("GET /api/models", modelsHandler.List)

	// Collection (protected)
	mux.HandleFunc("GET /api/collection", handlers.RequireAuth(collectionHandler.List))
	mux.HandleFunc("GET /api/collection/summary", handlers.RequireAuth(collectionHandler.Summary))
	mux.HandleFunc("PUT /api/collection/{modelId}", handlers.RequireAuth(collectionHandler.Put))
	mux.HandleFunc("DELETE /api/collection/{modelId}", handlers.RequireAuth(collectionHandler.Delete))

	// Lists (protected)
	mux.HandleFunc("GET /api/lists", handlers.RequireAuth(listsHandler.ListAll))
	mux.HandleFunc("POST /api/lists", handlers.RequireAuth(listsHandler.Create))
	mux.HandleFunc("GET /api/lists/{id}", listsHandler.Get)
	mux.HandleFunc("PUT /api/lists/{id}", handlers.RequireAuth(listsHandler.Update))
	mux.HandleFunc("DELETE /api/lists/{id}", handlers.RequireAuth(listsHandler.Delete))

	// Preferences (protected)
	mux.HandleFunc("GET /api/preferences", handlers.RequireAuth(preferencesHandler.Get))
	mux.HandleFunc("PUT /api/preferences", handlers.RequireAuth(preferencesHandler.Put))
	mux.HandleFunc("DELETE /api/preferences", handlers.RequireAuth(preferencesHandler.Delete))

	// Events (analytics)
	mux.HandleFunc("POST /api/events", eventsHandler.Track)
	mux.HandleFunc("GET /api/events/stats", handlers.RequireAuth(eventsHandler.Stats))

	// Shared lists (public)
	mux.HandleFunc("GET /api/shared/{shareCode}", listsHandler.SharedView)

	// Embedded frontend (SPA fallback)
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		// Check if file exists in embedded FS
		f, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback: serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	// Wrap with auth middleware (populates user context) then CORS
	handler := corsMiddleware(authHandler.AuthMiddleware(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("SLIC server listening on :%s", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allow specific origins for credentials support
		allowed := origin == "https://starleagueintelligencecommand.com" ||
			origin == "http://localhost:5173" ||
			origin == "http://localhost:8080"
		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// Same-origin requests
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
