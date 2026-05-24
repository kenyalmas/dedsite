package main

import (
	"database/sql"
	"dedsite/internal/db"
	"dedsite/internal/handlers"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	_ "modernc.org/sqlite"
)

var (
	//go:embed templates/*.html templates/partials/*.html static/**
	siteFS    embed.FS
	adapter   *httpadapter.HandlerAdapter
	bootstrap sync.Once
	bootErr   error
)

func initServer() error {
	// Netlify/AWS Lambda only guarantees write access under /tmp.
	dbDir := filepath.Join(os.TempDir(), "dedsite")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	conn, err := sql.Open("sqlite", filepath.Join(dbDir, "site.db"))
	if err != nil {
		return err
	}
	if err := db.Migrate(conn); err != nil {
		return err
	}
	store := db.NewStore(conn)
	if err := store.SeedDefaults(); err != nil {
		return err
	}
	if err := seedAdminFromEnv(store); err != nil {
		return err
	}

	tmpl, err := template.ParseFS(siteFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return err
	}

	app := handlers.New(store, tmpl, true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.Home)
	mux.HandleFunc("GET /section/{slug}", app.Section)
	mux.HandleFunc("GET /project/{slug}", app.Project)
	mux.HandleFunc("GET /awareness", app.Awareness)
	mux.HandleFunc("GET /awareness/data", app.AwarenessData)
	mux.HandleFunc("GET /admin", app.Admin)
	mux.HandleFunc("GET /admin/login", app.AdminLogin)
	mux.HandleFunc("POST /admin/login", app.AdminLogin)
	mux.HandleFunc("POST /admin/logout", app.AdminLogout)
	mux.HandleFunc("GET /admin/sections/{slug}/entries/new", app.AdminEntryForm)
	mux.HandleFunc("GET /admin/sections/{slug}/entries/{id}/edit", app.AdminEditEntryForm)
	mux.HandleFunc("POST /admin/sections/{slug}/entries", app.AdminCreateEntry)
	mux.HandleFunc("POST /admin/sections/{slug}/entries/{id}", app.AdminUpdateEntry)
	mux.HandleFunc("DELETE /admin/items/{id}", app.AdminDeleteEntry)
	staticFS, err := fs.Sub(siteFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /{path...}", app.NotFound)

	adapter = httpadapter.New(mux)
	return nil
}

func main() {
	lambda.Start(func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		bootstrap.Do(func() {
			bootErr = initServer()
			if bootErr != nil {
				log.Printf("bootstrap failed: %v", bootErr)
			}
		})
		if bootErr != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "startup failure: " + bootErr.Error(),
			}, nil
		}
		if adapter == nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, errors.New("handler not initialized")
		}
		return adapter.Proxy(req)
	})
}

func seedAdminFromEnv(store db.Store) error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return errors.New("ADMIN_USERNAME and ADMIN_PASSWORD must both be set")
	}
	if len(password) < 12 {
		return errors.New("ADMIN_PASSWORD must be at least 12 characters")
	}
	return store.SetAdminPassword(username, password)
}
