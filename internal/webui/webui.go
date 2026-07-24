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
	"strings"
	"sync"
	"time"

	"github.com/fvlgnn/turnino/internal/credential"
	"github.com/fvlgnn/turnino/internal/domain"
)

// files contains the complete browser UI so releases remain single binaries.
//
//go:embed templates/*.html static/* translations/*.json
var files embed.FS

type Store interface {
	AdminConfigured(context.Context) (bool, error)
	ValidateAdminToken(context.Context, []byte) (bool, error)
	GetSettings(context.Context) (domain.Settings, error)
	UpdateSettings(context.Context, domain.Settings) (domain.Settings, error)
	ListQueues(context.Context) ([]domain.Queue, error)
	CreateQueue(context.Context, domain.CreateQueueParams) (domain.Queue, error)
	SetQueueCustomCalls(context.Context, int64, bool) error
	ListCounters(context.Context) ([]domain.Counter, error)
	CreateCounter(context.Context, string, []byte) (domain.Counter, error)
	RevokeCounter(context.Context, int64) error
	RotateCounterToken(context.Context, int64, []byte) error
	ReactivateCounter(context.Context, int64, []byte) error
	ListRecentCallRecords(context.Context, int) ([]domain.CallRecord, error)
}

type app struct {
	version  string
	store    Store
	tmpl     *template.Template
	catalog  catalog
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
	Lang            string
	AdminConfigured bool
	CSRFToken       string
	Error           string
	Notice          string
	Queues          []domain.Queue
	Counters        []domain.Counter
	Settings        domain.Settings
	Calls           []domain.CallRecord
	TestMode        bool
	NewCounterToken string
	DisplayCounter  domain.Counter
}

func New(version string, store Store) (http.Handler, error) {
	catalog, err := loadCatalog()
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"t":   catalog.text,
		"mul": func(left, right int) int { return left * right },
	}).ParseFS(files, "templates/*.html")
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
		catalog: catalog,
		sessions: sessionStore{
			sessions: make(map[string]session),
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))
	mux.HandleFunc("GET /", application.home)
	mux.HandleFunc("GET /caller", application.caller)
	mux.HandleFunc("GET /counter-display", application.counterDisplay)
	mux.HandleFunc("GET /monitor", application.monitor)
	mux.HandleFunc("GET /admin", application.admin)
	mux.HandleFunc("POST /admin/login", application.login)
	mux.HandleFunc("POST /admin/logout", application.logout)
	mux.HandleFunc("POST /admin/settings", application.updateSettings)
	mux.HandleFunc("POST /admin/queues", application.createQueue)
	mux.HandleFunc("POST /admin/queues/{queueID}/custom-calls", application.setQueueCustomCalls)
	mux.HandleFunc("POST /admin/counters", application.createCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/revoke", application.revokeCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/rotate", application.rotateCounter)
	mux.HandleFunc("POST /admin/counters/{counterID}/reactivate", application.reactivateCounter)
	return application.languageSelection(mux), nil
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "index.html", a.page(r))
}

func (a *app) caller(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	queues, err := a.store.ListQueues(r.Context())
	if err != nil {
		http.Error(w, "Could not load queues", http.StatusInternalServerError)
		return
	}
	data := a.page(r)
	data.Queues = queues
	a.render(w, "caller.html", data)
}

func (a *app) counterDisplay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	data := a.page(r)
	counterID, err := strconv.ParseInt(r.URL.Query().Get("counter"), 10, 64)
	if err == nil && counterID > 0 {
		counters, listErr := a.store.ListCounters(r.Context())
		if listErr != nil {
			http.Error(w, "Could not load counters", http.StatusInternalServerError)
			return
		}
		for _, counter := range counters {
			if counter.ID == counterID && counter.Enabled {
				data.DisplayCounter = counter
				break
			}
		}
	}
	a.render(w, "counter-display.html", data)
}

func (a *app) monitor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	data := a.page(r)
	calls, err := a.store.ListRecentCallRecords(r.Context(), data.Settings.RecentCallsLimit)
	if err != nil {
		http.Error(w, "Could not load recent calls", http.StatusInternalServerError)
		return
	}
	data.Calls = calls
	data.TestMode = r.URL.Query().Get("test") == "1"
	a.render(w, "monitor.html", data)
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
	data := a.page(r)
	data.AdminConfigured = configured
	a.render(w, "admin-login.html", data)
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
		data := a.page(r)
		data.AdminConfigured = true
		data.Error = a.message(r, "admin.invalid_token")
		a.render(w, "admin-login.html", data)
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

func (a *app) updateSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	recentCallsLimit, recentCallsErr := strconv.Atoi(r.PostFormValue("recent_calls_limit"))
	highlightDuration, highlightDurationErr := strconv.Atoi(r.PostFormValue("highlight_duration_seconds"))
	if recentCallsErr != nil || highlightDurationErr != nil {
		a.redirectAdminError(w, r, a.message(r, "settings.invalid_numbers"))
		return
	}
	_, err := a.store.UpdateSettings(r.Context(), domain.Settings{
		InstallationName:         r.PostFormValue("installation_name"),
		DefaultLanguage:          r.PostFormValue("default_language"),
		RecentCallsLimit:         recentCallsLimit,
		SoundEnabled:             r.PostFormValue("sound_enabled") == "on",
		HighlightDurationSeconds: highlightDuration,
	})
	if err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/admin?notice=settings-saved", http.StatusSeeOther)
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
		a.redirectAdminError(w, r, a.message(r, "admin.invalid_number_policy"))
		return
	}
	_, err := a.store.CreateQueue(r.Context(), domain.CreateQueueParams{
		Name: r.PostFormValue("name"), Prefix: r.PostFormValue("prefix"),
		Digits: digits, MinValue: minValue, MaxValue: maxValue,
		AllowCustomCalls: r.PostFormValue("allow_custom_calls") == "on",
	})
	if err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
	_ = current
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *app) setQueueCustomCalls(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authorizeForm(w, r)
	if !ok {
		return
	}
	queueID, err := parseID(r.PathValue("queueID"))
	if err != nil {
		a.redirectAdminError(w, r, a.message(r, "admin.invalid_queue_id"))
		return
	}
	if err := a.store.SetQueueCustomCalls(r.Context(), queueID, r.PostFormValue("enabled") == "true"); err != nil {
		a.redirectAdminError(w, r, err.Error())
		return
	}
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
		a.redirectAdminError(w, r, a.message(r, "admin.invalid_counter_id"))
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
		a.redirectAdminError(w, r, a.message(r, "admin.invalid_counter_id"))
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
		a.redirectAdminError(w, r, a.message(r, "admin.invalid_counter_id"))
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
	data := a.page(r)
	data.CSRFToken = current.CSRFToken
	data.Error = r.URL.Query().Get("error")
	if r.URL.Query().Get("notice") == "settings-saved" {
		data.Notice = a.catalog.text(data.Lang, "settings.saved")
	}
	data.Queues = queues
	data.Counters = counters
	data.NewCounterToken = newCounterToken
	a.render(w, "admin.html", data)
}

func (a *app) languageSelection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
		if r.Method == http.MethodGet && (requested == "en" || requested == "it") {
			http.SetCookie(w, &http.Cookie{
				Name: "turnino_lang", Value: requested, Path: "/", MaxAge: 365 * 24 * 60 * 60,
				Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
			})
			target := *r.URL
			query := target.Query()
			query.Del("lang")
			target.RawQuery = query.Encode()
			http.Redirect(w, r, target.String(), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) page(r *http.Request) pageData {
	settings := domain.DefaultSettings()
	stored, err := a.store.GetSettings(r.Context())
	if err != nil {
		slog.Error("load settings", "error", err)
	} else {
		settings = stored
	}
	return pageData{
		Version:  a.version,
		Lang:     a.language(r, settings.DefaultLanguage),
		Settings: settings,
	}
}

func (a *app) message(r *http.Request, key string) string {
	data := a.page(r)
	return a.catalog.text(data.Lang, key)
}

func (a *app) language(r *http.Request, fallback string) string {
	if cookie, err := r.Cookie("turnino_lang"); err == nil {
		if cookie.Value == "en" || cookie.Value == "it" {
			return cookie.Value
		}
	}
	for _, preference := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		language := strings.ToLower(strings.TrimSpace(strings.SplitN(preference, ";", 2)[0]))
		if separator := strings.IndexAny(language, "-_"); separator >= 0 {
			language = language[:separator]
		}
		if language == "it" {
			return "it"
		}
		if language == "en" {
			return "en"
		}
	}
	return normalizeLanguage(fallback)
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
