package webui

import (
	"context"
	"crypto/subtle"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/fvlgnn/turnino/internal/credential"
	"github.com/fvlgnn/turnino/internal/domain"
)

// files contains the complete browser UI so releases remain single binaries.
//
//go:embed templates/*.html static/*
var files embed.FS

type Store interface {
	AdminConfigured(context.Context) (bool, error)
	ValidateAdminToken(context.Context, []byte) (bool, error)
	ListQueues(context.Context) ([]domain.Queue, error)
	CreateQueue(context.Context, domain.CreateQueueParams) (domain.Queue, error)
	ListCounters(context.Context) ([]domain.Counter, error)
	CreateCounter(context.Context, string, []byte) (domain.Counter, error)
	RevokeCounter(context.Context, int64) error
	RotateCounterToken(context.Context, int64, []byte) error
	ReactivateCounter(context.Context, int64, []byte) error
}

type app struct {
	version  string
	store    Store
	tmpl     *template.Template
	sessions sessionStore
}

type session struct {
	CSRFToken string
	ExpiresAt time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
}

type pageData struct {
	Version         string
	AdminConfigured bool
	CSRFToken       string
	Error           string
	Queues          []domain.Queue
	Counters        []domain.Counter
	NewCounterToken string
}

func New(version string, store Store) (http.Handler, error) {
	tmpl, err := template.ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	staticFiles, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("open static assets: %w", err)
	}
	application := &app{
		version: version,
		store:   store,
		tmpl:    tmpl,
		sessions: sessionStore{
			sessions: make(map[string]session),
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))
	mux.HandleFunc("GET /", application.home)
	mux.HandleFunc("GET /admin", application.admin)
	mux.HandleFunc("POST /admin/login", application.login)
	mux.HandleFunc("POST /admin/logout", application.logout)
	mux.HandleFunc("POST /admin/queues", application.createQueue)
	mux.HandleFunc("POST /admin/counters", application.createCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/revoke", application.revokeCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/rotate", application.rotateCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/reactivate", application.reactivateCounter)
	return mux, nil
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "index.html", pageData{Version: a.version})
}

func (a *app) admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if current, ok := a.currentSession(r); ok {
		a.renderDashboard(w, r, current, "")
		return
	}
	configured, err := a.store.AdminConfigured(r.Context())
	if err != nil {
		http.Error(w, "Could not check administrator configuration", http.StatusInternalServerError)
		return
	}
	a.render(w, "admin-login.html", pageData{Version: a.version, AdminConfigured: configured})
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !a.parseForm(w, r) {
		return
	}
	token := r.PostFormValue("token")
	tokenHash := credential.Hash(token)
	valid, err := a.store.ValidateAdminToken(r.Context(), tokenHash[:])
	if err != nil {
		http.Error(w, "Could not validate administrator token", http.StatusInternalServerError)
		return
	}
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		a.render(w, "admin-login.html", pageData{Version: a.version, AdminConfigured: true, Error: "Invalid administrator token."})
		return
	}
	sessionID, err := credential.Generate()
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	csrfToken, err := credential.Generate()
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	a.sessions.put(sessionID, session{CSRFToken: csrfToken, ExpiresAt: time.Now().Add(12 * time.Hour)})
	http.SetCookie(w, &http.Cookie{
		Name: "turnino_admin", Value: sessionID, Path: "/admin", HttpOnly: true,
		Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: int((12 * time.Hour).Seconds()),
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	if cookie, err := r.Cookie("turnino_admin"); err == nil {
		a.sessions.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "turnino_admin", Path: "/admin", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *app) createQueue(w http.ResponseWriter, r *http.Request) {
	current, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	digits, digitsErr := strconv.Atoi(r.PostFormValue("digits"))
	minValue, minErr := strconv.Atoi(r.PostFormValue("min_value"))
	maxValue, maxErr := strconv.Atoi(r.PostFormValue("max_value"))
	if digitsErr != nil || minErr != nil || maxErr != nil {
		a.redirectAdminError(w, r, "Number policy contains invalid values.")
		return
	}
	_, err := a.store.CreateQueue(r.Context(), domain.CreateQueueParams{
		Name: r.PostFormValue("name"), Prefix: r.PostFormValue("prefix"),
		Digits: digits, MinValue: minValue, MaxValue: maxValue,
	})
	if err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	_ = current
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *app) createCounter(w http.ResponseWriter, r *http.Request) {
	current, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	token, err := credential.Generate()
	if err != nil {
		http.Error(w, "Could not generate counter token", http.StatusInternalServerError)
		return
	}
	tokenHash := credential.Hash(token)
	if _, err := a.store.CreateCounter(r.Context(), r.PostFormValue("name"), tokenHash[:]); err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	a.renderDashboard(w, r, current, token)
}

func (a *app) revokeCounter(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	counterID, err := parseID(r.PathValue("counterID"))
	if err != nil {
		a.redirectAdminError(w, r, "Invalid counter ID.")
		return
	}
	if err := a.store.RevokeCounter(r.Context(), counterID); err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *app) rotateCounter(w http.ResponseWriter, r *http.Request) {
	current, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	counterID, err := parseID(r.PathValue("counterID"))
	if err != nil {
		a.redirectAdminError(w, r, "Invalid counter ID.")
		return
	}
	token, err := credential.Generate()
	if err != nil {
		http.Error(w, "Could not generate counter token", http.StatusInternalServerError)
		return
	}
	tokenHash := credential.Hash(token)
	if err := a.store.RotateCounterToken(r.Context(), counterID, tokenHash[:]); err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	a.renderDashboard(w, r, current, token)
}

func (a *app) reactivateCounter(w http.ResponseWriter, r *http.Request) {
	current, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	counterID, err := parseID(r.PathValue("counterID"))
	if err != nil {
		a.redirectAdminError(w, r, "Invalid counter ID.")
		return
	}
	token, err := credential.Generate()
	if err != nil {
		http.Error(w, "Could not generate counter token", http.StatusInternalServerError)
		return
	}
	tokenHash := credential.Hash(token)
	if err := a.store.ReactivateCounter(r.Context(), counterID, tokenHash[:]); err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	a.renderDashboard(w, r, current, token)
}

func (a *app) renderDashboard(w http.ResponseWriter, r *http.Request, current session, newCounterToken string) {
	w.Header().Set("Cache-Control", "no-store")
	queues, err := a.store.ListQueues(r.Context())
	if err != nil {
		http.Error(w, "Could not load queues", http.StatusInternalServerError)
		return
	}
	counters, err := a.store.ListCounters(r.Context())
	if err != nil {
		http.Error(w, "Could not load counters", http.StatusInternalServerError)
		return
	}
	a.render(w, "admin.html", pageData{
		Version: a.version, CSRFToken: current.CSRFToken, Error: r.URL.Query().Get("error"),
		Queues: queues, Counters: counters, NewCounterToken: newCounterToken,
	})
}

func (a *app) authorizeForm(w http.ResponseWriter, r *http.Request) (session, bool) {
	current, ok := a.currentSession(r)
	if !ok {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return session{}, false
	}
	if !a.parseForm(w, r) {
		return session{}, false
	}
	provided := []byte(r.PostFormValue("csrf_token"))
	expected := []byte(current.CSRFToken)
	if len(provided) != len(expected) || subtle.ConstantTimeCompare(provided, expected) != 1 {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return session{}, false
	}
	return current, true
}

func (a *app) currentSession(r *http.Request) (session, bool) {
	cookie, err := r.Cookie("turnino_admin")
	if err != nil {
		return session{}, false
	}
	return a.sessions.get(cookie.Value)
}

func (a *app) parseForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return false
	}
	return true
}

func (a *app) redirectAdminError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, "/admin?error="+url.QueryEscape(message), http.StatusSeeOther)
}

func (a *app) render(w http.ResponseWriter, name string, data pageData) {
	if err := a.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render page", "template", name, "error", err)
	}
}

func (s *sessionStore) put(id string, value session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = value
}

func (s *sessionStore) get(id string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.sessions[id]
	if !ok || time.Now().After(value.ExpiresAt) {
		delete(s.sessions, id)
		return session{}, false
	}
	return value, true
}

func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid ID")
	}
	return id, nil
}
