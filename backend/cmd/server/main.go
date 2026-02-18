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

	mechHandler := &handlers.MechHandlerSQLite{DB: sqlDB}
	feedbackHandler := handlers.NewFeedbackHandler()

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

	// CORS middleware
	handler := corsMiddleware(mux)

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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
