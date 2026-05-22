package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	options, args, err := parseOptions(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if len(args) > 0 && args[0] == "cli" {
		if err := runCLI(options, args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(args) > 0 && args[0] != "serve" {
		log.Fatalf("unknown command %q\nusage: dedsite [--port port] [--db path] [--no-seed] [serve] | dedsite [--db path] [--no-seed] cli [--json] [--pretty] <profile|sections|section|seed>", args[0])
	}

	if err := serve(options); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	addr   string
	dbPath string
	noSeed bool
}

type cliOptions struct {
	json   bool
	pretty bool
}

func parseOptions(args []string) (options, []string, error) {
	values := options{
		addr:   env("ADDR", ":8080"),
		dbPath: env("DATABASE_PATH", filepath.Join("data", "site.db")),
	}

	flags := flag.NewFlagSet("dedsite", flag.ContinueOnError)
	flags.StringVar(&values.addr, "port", values.addr, "server port or address")
	flags.StringVar(&values.dbPath, "db", values.dbPath, "SQLite database path")
	flags.BoolVar(&values.noSeed, "no-seed", false, "skip default content seeding")

	if err := flags.Parse(args); err != nil {
		return options{}, nil, err
	}

	values.addr = normalizeAddr(values.addr)
	return values, flags.Args(), nil
}

func parseCLIOptions(args []string) (cliOptions, []string, error) {
	values := cliOptions{}

	flags := flag.NewFlagSet("dedsite cli", flag.ContinueOnError)
	flags.BoolVar(&values.json, "json", false, "output JSON")
	flags.BoolVar(&values.pretty, "pretty", false, "pretty-print JSON output")

	if err := flags.Parse(args); err != nil {
		return cliOptions{}, nil, err
	}
	return values, flags.Args(), nil
}

func normalizeAddr(value string) string {
	if value == "" {
		return ":8080"
	}
	if value[0] == ':' {
		return value
	}
	return ":" + value
}

func serve(options options) error {
	conn, err := openStore(options.dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	// SeedDefaults also syncs a few project-owned defaults into existing local DBs
	// while the site's content model is still being shaped.
	store := db.NewStore(conn)
	if !options.noSeed {
		if err := store.SeedDefaults(); err != nil {
			return fmt.Errorf("seed database: %w", err)
		}
	}

	tmpl, err := template.ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}
	if tmpl, err = tmpl.ParseGlob(filepath.Join("templates", "partials", "*.html")); err != nil {
		return fmt.Errorf("parse partial templates: %w", err)
	}

	app := handlers.New(store, tmpl)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.Home)
	mux.HandleFunc("GET /section/{slug}", app.Section)
	mux.HandleFunc("GET /project/{slug}", app.Project)
	mux.HandleFunc("GET /admin", app.Admin)
	mux.HandleFunc("GET /admin/login", app.AdminLogin)
	mux.HandleFunc("POST /admin/login", app.AdminLogin)
	mux.HandleFunc("GET /admin/sections/{slug}/entries/new", app.AdminEntryForm)
	mux.HandleFunc("POST /admin/sections/{slug}/entries", app.AdminCreateEntry)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Bind before logging the URL so port conflicts fail with a clear message.
	listener, err := net.Listen("tcp", options.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w\nSet ADDR to another port, for example: $env:ADDR=':8081'; go run ./cmd/server", options.addr, err)
	}
	defer listener.Close()

	log.Printf("dedsite listening on http://localhost%s", options.addr)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func runCLI(options options, args []string) error {
	cli, args, err := parseCLIOptions(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("usage: dedsite cli [--json] [--pretty] <profile|sections|section|seed>")
	}

	conn, err := openStore(options.dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	store := db.NewStore(conn)

	switch args[0] {
	case "seed":
		if options.noSeed {
			return printValue(cli, map[string]string{"status": "seed skipped"})
		}
		if err := store.SeedDefaults(); err != nil {
			return fmt.Errorf("seed database: %w", err)
		}
		return printValue(cli, map[string]string{"status": "seeded default content"})
	case "profile":
		profile, err := store.Profile()
		if err != nil {
			return fmt.Errorf("load profile: %w", err)
		}
		if cli.json {
			return printValue(cli, profile)
		}
		fmt.Printf("%s\n%s\n%s\n\n%s\n", profile.Name, profile.Title, profile.Email, profile.Summary)
	case "sections":
		sections, err := store.Sections()
		if err != nil {
			return fmt.Errorf("load sections: %w", err)
		}
		if cli.json {
			return printValue(cli, sections)
		}
		for _, section := range sections {
			fmt.Printf("%s\t%s\t%d item(s)\n", section.Slug, section.Title, len(section.Items))
		}
	case "section":
		if len(args) < 2 {
			return errors.New("usage: dedsite cli section <slug>")
		}
		section, err := store.Section(args[1])
		if err != nil {
			return fmt.Errorf("load section %q: %w", args[1], err)
		}
		if cli.json {
			return printValue(cli, section)
		}
		fmt.Println(section.Title)
		for _, item := range section.Items {
			fmt.Printf("\n- %s\n", item.Title)
			if item.Subtitle != "" {
				fmt.Printf("  %s\n", item.Subtitle)
			}
			if item.Period != "" {
				fmt.Printf("  %s\n", item.Period)
			}
			if item.Description != "" {
				fmt.Printf("  %s\n", item.Description)
			}
			if len(item.Tags) > 0 {
				fmt.Printf("  tags: %s\n", join(item.Tags))
			}
		}
	default:
		return fmt.Errorf("unknown cli command %q", args[0])
	}

	return nil
}

func printValue(options cliOptions, value any) error {
	if !options.json {
		if status, ok := value.(map[string]string); ok {
			fmt.Println(status["status"])
			return nil
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	if options.pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(value)
}

func openStore(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return conn, nil
}

func join(values []string) string {
	if len(values) == 0 {
		return ""
	}

	out := values[0]
	for _, value := range values[1:] {
		out += ", " + value
	}
	return out
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
