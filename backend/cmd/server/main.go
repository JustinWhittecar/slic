package main

import (
	"database/sql"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"context"
	"strconv"
	"time"
	"strings"

	"github.com/JustinWhittecar/slic/internal/customerio"
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

	// Customer.io
	cioSiteID := os.Getenv("CIO_SITE_ID")
	cioAPIKey := os.Getenv("CIO_API_KEY")
	cioAppAPIKey := os.Getenv("CIO_APP_API_KEY")
	if cioSiteID == "" || cioAPIKey == "" {
		log.Println("[WARN] CIO_SITE_ID or CIO_API_KEY not set; Customer.io tracking disabled")
	}
	if cioAppAPIKey == "" {
		log.Println("[WARN] CIO_APP_API_KEY not set; Customer.io transactional emails disabled")
	}
	cioClient := customerio.New(cioSiteID, cioAPIKey, cioAppAPIKey)

	mechHandler := &handlers.MechHandlerSQLite{DB: sqlDB}
	feedbackHandler := handlers.NewFeedbackHandler(cioClient)
	authHandler := handlers.NewAuthHandler(userDB)
	authHandler.CIO = cioClient
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
	// Read index.html once for OG tag injection
	indexBytes, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to read index.html: %v", err)
	}
	indexHTML := string(indexBytes)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		// Check if file exists in embedded FS (skip /mech/ routes)
		if !strings.HasPrefix(path, "/mech/") {
			f, err := distFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// /mech/{id} — inject OG meta tags for social previews
		if strings.HasPrefix(r.URL.Path, "/mech/") {
			mechID := strings.TrimPrefix(r.URL.Path, "/mech/")
			if id, err := strconv.Atoi(mechID); err == nil {
				var chassis, modelCode, role string
				var tonnage int
				var bv sql.NullInt64
				var cr sql.NullFloat64
				err := sqlDB.QueryRow(`
					SELECT c.name, v.model_code, COALESCE(v.role,''), COALESCE(vs.tonnage, c.tonnage), v.battle_value, vs.combat_rating
					FROM variants v
					JOIN chassis c ON c.id = v.chassis_id
					LEFT JOIN variant_stats vs ON vs.variant_id = v.id
					WHERE v.id = ?`, id).Scan(&chassis, &modelCode, &role, &tonnage, &bv, &cr)
				if err == nil {
					title := fmt.Sprintf("%s %s — SLIC", chassis, modelCode)
					desc := fmt.Sprintf("%dt %s", tonnage, role)
					if bv.Valid {
						desc += fmt.Sprintf(" · BV %d", bv.Int64)
					}
					if cr.Valid {
						desc += fmt.Sprintf(" · Combat Rating %.1f", cr.Float64)
					}
					ogHTML := indexHTML
					ogHTML = strings.Replace(ogHTML,
						`<meta property="og:title" content="SLIC — BattleTech Mech Database" />`,
						fmt.Sprintf(`<meta property="og:title" content="%s" />`, html.EscapeString(title)), 1)
					ogHTML = strings.Replace(ogHTML,
						`<meta property="og:description" content="Browse 4,200+ mech variants with combat ratings from 1,000 Monte Carlo simulations. Build tournament lists with BV tracking." />`,
						fmt.Sprintf(`<meta property="og:description" content="%s" />`, html.EscapeString(desc)), 1)
					ogHTML = strings.Replace(ogHTML,
						`<meta property="og:url" content="https://starleagueintelligencecommand.com" />`,
						fmt.Sprintf(`<meta property="og:url" content="https://starleagueintelligencecommand.com/mech/%d" />`, id), 1)
					ogHTML = strings.Replace(ogHTML,
						`<title>SLIC — BattleTech Mech Database</title>`,
						fmt.Sprintf(`<title>%s</title>`, html.EscapeString(title)), 1)
					ogHTML = strings.Replace(ogHTML,
						`<meta name="description" content="BattleTech mech database with Monte Carlo combat ratings, BV efficiency analysis, and tournament list builder. 4,200+ variants." />`,
						fmt.Sprintf(`<meta name="description" content="%s" />`, html.EscapeString(desc)), 1)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.Write([]byte(ogHTML))
					return
				}
			}
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
