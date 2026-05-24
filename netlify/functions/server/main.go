package main

import (
	"database/sql"
	"dedsite/internal/db"
	"dedsite/internal/handlers"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	_ "modernc.org/sqlite"
)

var adapter *httpadapter.HandlerAdapter

func init() {
	conn, err := sql.Open("sqlite", filepath.Join("data", "site.db"))
	if err != nil {
		panic(err)
	}
	if err := db.Migrate(conn); err != nil {
		panic(err)
	}
	store := db.NewStore(conn)
	if err := store.SeedDefaults(); err != nil {
		panic(err)
	}

	tmpl, err := template.ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		panic(err)
	}
	if tmpl, err = tmpl.ParseGlob(filepath.Join("templates", "partials", "*.html")); err != nil {
		panic(err)
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
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /{path...}", app.NotFound)

	adapter = httpadapter.New(mux)
}

func main() {
	lambda.Start(func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		if adapter == nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, errors.New("handler not initialized")
		}
		return adapter.Proxy(req)
	})
}

