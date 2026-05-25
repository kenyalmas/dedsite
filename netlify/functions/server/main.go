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
	"strings"
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

	tmpl, err := template.ParseFS(siteFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return err
	}

	app := handlers.New(store, tmpl, true, handlers.GoogleOAuthConfig{
		ClientID:      os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret:  os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		AllowedEmails: splitCSVEnv(os.Getenv("GOOGLE_OAUTH_ALLOWED_EMAILS")),
	})
	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, app)
	staticFS, err := fs.Sub(siteFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /{path...}", app.NotFound)

	adapter = httpadapter.New(mux)
	return nil
}

func splitCSVEnv(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.ToLower(strings.TrimSpace(part))
		if value != "" {
			values = append(values, value)
		}
	}
	return values
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
