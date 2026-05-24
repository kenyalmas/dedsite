package main

import (
	"database/sql"
	"dedsite/internal/db"
	"dedsite/internal/handlers"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	_ "modernc.org/sqlite"
)

var (
	adapter    *httpadapter.HandlerAdapter
	bootstrap  sync.Once
	bootErr    error
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

	projectRoot, err := resolveProjectRoot()
	if err != nil {
		return err
	}

	tmpl, err := template.ParseGlob(filepath.Join(projectRoot, "templates", "*.html"))
	if err != nil {
		return err
	}
	if tmpl, err = tmpl.ParseGlob(filepath.Join(projectRoot, "templates", "partials", "*.html")); err != nil {
		return err
	}

	app := handlers.New(store, tmpl, true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.Home)
	mux.HandleFunc("GET /section/{slug}", app.Section)
	mux.HandleFunc("GET /project/{slug}", app.Project)
	mux.HandleFunc("GET /admin", app.Admin)
	mux.HandleFunc("GET /admin/login", app.AdminLogin)
	mux.HandleFunc("POST /admin/login", app.AdminLogin)
	mux.HandleFunc("POST /admin/logout", app.AdminLogout)
	mux.HandleFunc("GET /admin/sections/{slug}/entries/new", app.AdminEntryForm)
	mux.HandleFunc("GET /admin/sections/{slug}/entries/{id}/edit", app.AdminEditEntryForm)
	mux.HandleFunc("POST /admin/sections/{slug}/entries", app.AdminCreateEntry)
	mux.HandleFunc("POST /admin/sections/{slug}/entries/{id}", app.AdminUpdateEntry)
	mux.HandleFunc("DELETE /admin/items/{id}", app.AdminDeleteEntry)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(projectRoot, "static")))))
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

func resolveProjectRoot() (string, error) {
	candidates := []string{}
	if taskRoot := strings.TrimSpace(os.Getenv("LAMBDA_TASK_ROOT")); taskRoot != "" {
		candidates = append(candidates, taskRoot)
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates, wd)
	}
	candidates = append(candidates, ".")

	for _, root := range candidates {
		templatesPath := filepath.Join(root, "templates")
		staticPath := filepath.Join(root, "static")
		if dirExists(templatesPath) && dirExists(staticPath) {
			return root, nil
		}
	}
	return "", fmt.Errorf("could not locate project root; checked %v", candidates)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
