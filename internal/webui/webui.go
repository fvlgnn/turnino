package webui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

// files contains the complete browser UI so releases remain single binaries.
//
//go:embed templates/*.html static/*
var files embed.FS

type pageData struct {
	Version string
}

// New builds the server-rendered browser interface.
func New(version string) (http.Handler, error) {
	tmpl, err := template.ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	staticFiles, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("open static assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", pageData{Version: version}); err != nil {
			slog.Error("render home page", "error", err)
		}
	})

	return mux, nil
}
