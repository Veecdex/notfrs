// Command server starts the Notes API: it loads config, connects to
// Postgres, registers routes, and listens for HTTP requests.
package main

import (
	"log"
	"net/http"

	"notes-app/internal/config"
	"notes-app/internal/db"
	"notes-app/internal/handlers"
	"notes-app/internal/middleware"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	conn, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer conn.Close()
	log.Println("connected to database")

	authHandler := &handlers.AuthHandler{DB: conn, JWTSecret: cfg.JWTSecret}
	noteHandler := &handlers.NoteHandler{DB: conn}
	profileHandler := &handlers.ProfileHandler{DB: conn}

	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(cfg.JWTSecret)

	// Public routes
	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)
	mux.HandleFunc("POST /api/forgot-password/check", authHandler.CheckEmail)
	mux.HandleFunc("POST /api/forgot-password/reset", authHandler.ResetPassword)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Protected routes: each one wrapped individually with requireAuth,
	// all on the same flat mux. No nested router, so there's exactly one
	// place any given method+path can match - easy to verify by eye.
	mux.Handle("GET /api/notes", requireAuth(http.HandlerFunc(noteHandler.List)))
	mux.Handle("POST /api/notes", requireAuth(http.HandlerFunc(noteHandler.Create)))
	mux.Handle("PUT /api/notes/{id}", requireAuth(http.HandlerFunc(noteHandler.Update)))
	mux.Handle("DELETE /api/notes/{id}", requireAuth(http.HandlerFunc(noteHandler.Delete)))
	mux.Handle("GET /api/me", requireAuth(http.HandlerFunc(profileHandler.Me)))
	mux.Handle("PUT /api/me", requireAuth(http.HandlerFunc(profileHandler.UpdateProfile)))
	mux.Handle("PUT /api/me/password", requireAuth(http.HandlerFunc(profileHandler.UpdatePassword)))
	mux.Handle("DELETE /api/me", requireAuth(http.HandlerFunc(profileHandler.DeleteAccount)))

	// Wrap everything in CORS so the frontend (a different origin) can call it.
	handler := middleware.CORS(cfg.FrontendOrigin)(mux)

	log.Printf("listening on http://localhost:%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
