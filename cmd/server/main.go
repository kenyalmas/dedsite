package main

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dedsite/internal/db"
	"dedsite/internal/handlers"

	_ "modernc.org/sqlite"
)

func main() {
	addr := env("ADDR", ":8080")
	dbPath := env("DATABASE_PATH", filepath.Join("data", "site.db"))

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	// SeedDefaults also syncs a few project-owned defaults into existing local DBs
	// while the site's content model is still being shaped.
	store := db.NewStore(conn)
	if err := store.SeedDefaults(); err != nil {
		log.Fatalf("seed database: %v", err)
	}

	tmpl, err := template.ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}
	if tmpl, err = tmpl.ParseGlob(filepath.Join("templates", "partials", "*.html")); err != nil {
		log.Fatalf("parse partial templates: %v", err)
	}

	app := handlers.New(store, tmpl)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.Home)
	mux.HandleFunc("GET /section/{slug}", app.Section)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Bind before logging the URL so port conflicts fail with a clear message.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v\nSet ADDR to another port, for example: $env:ADDR=':8081'; go run ./cmd/server", addr, err)
	}
	defer listener.Close()

	log.Printf("dedsite listening on http://localhost%s", addr)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
