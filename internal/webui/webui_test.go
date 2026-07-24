package webui

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fvlgnn/turnino/internal/domain"
	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
)

func TestHomePage(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Your turn, made simple.",
		"Call the next number. Keep every counter, display, and waiting room in sync.",
		`href="/caller"`,
		"test-version",
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestCallerPageListsQueues(t *testing.T) {
	handler, store := testHandler(t)
	if _, err := store.CreateQueue(context.Background(), domain.CreateQueueParams{
		Name: "General", Prefix: "A", Digits: 3, MinValue: 1, MaxValue: 999,
	}); err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/caller", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Call a number",
		"Counter token",
		`id="caller-identity"`,
		"General",
		`data-queue-id="1"`,
		`src="/static/caller.js"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestCallerPageWithoutQueues(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/caller", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "No queues are configured.") {
		t.Fatalf("empty caller page = %d %q", response.Code, response.Body.String())
	}
}

func TestMonitorPageShowsRecentCallsAndTestMode(t *testing.T) {
	handler, store := testHandler(t)
	queue, err := store.CreateQueue(context.Background(), domain.CreateQueueParams{
		Name: "General services", Prefix: "A", Digits: 3, MinValue: 1, MaxValue: 999,
	})
	if err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	tokenHash := sha256.Sum256([]byte("monitor-test-counter-token"))
	if _, err := store.CreateCounter(context.Background(), "Counter 1", tokenHash[:]); err != nil {
		t.Fatalf("CreateCounter() error = %v", err)
	}
	if _, err := store.CallNext(context.Background(), queue.ID, tokenHash[:]); err != nil {
		t.Fatalf("CallNext() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/monitor?test=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Public monitor",
		"General services",
		"A001",
		"Counter 1",
		`src="/static/monitor.js"`,
		`href="/caller"`,
		`data-highlight-ms="5000"`,
		`id="monitor-sound"`,
		`id="monitor-fullscreen"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestMonitorPageWithoutCalls(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/monitor", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Waiting for the first call") {
		t.Fatalf("empty monitor page = %d %q", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "Open Caller for testing") {
		t.Fatal("normal monitor unexpectedly exposes test-mode link")
	}
	for _, unexpected := range []string{`id="monitor-sound"`, `id="monitor-fullscreen"`} {
		if strings.Contains(response.Body.String(), unexpected) {
			t.Fatalf("normal monitor unexpectedly contains %s", unexpected)
		}
	}
	if !strings.Contains(response.Body.String(), `class="monitor-connection compact"`) {
		t.Fatal("normal monitor does not contain the compact connection indicator")
	}
}

func TestItalianFromAcceptLanguage(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Language", "it-IT,it;q=0.9,en;q=0.8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{`<html lang="it">`, "Il tuo turno, senza complicazioni.", "Amministrazione"} {
		if !strings.Contains(body, expected) {
			t.Errorf("Italian response does not contain %q", expected)
		}
	}
}

func TestInstallationDefaultLanguageAndName(t *testing.T) {
	handler, store := testHandler(t)
	settings, err := store.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	settings.InstallationName = "Comune di Esempio"
	settings.DefaultLanguage = "it"
	if _, err := store.UpdateSettings(context.Background(), settings); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	for _, expected := range []string{`<html lang="it">`, "Comune di Esempio", "Il tuo turno, senza complicazioni."} {
		if !strings.Contains(body, expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestLanguageSelectionPersistsCookie(t *testing.T) {
	handler, _ := testHandler(t)
	selection := httptest.NewRequest(http.MethodGet, "/admin?lang=it", nil)
	selectionResponse := httptest.NewRecorder()
	handler.ServeHTTP(selectionResponse, selection)
	if selectionResponse.Code != http.StatusSeeOther || selectionResponse.Header().Get("Location") != "/admin" {
		t.Fatalf("language selection = %d %q, want 303 /admin", selectionResponse.Code, selectionResponse.Header().Get("Location"))
	}
	cookies := selectionResponse.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "turnino_lang" || cookies[0].Value != "it" {
		t.Fatalf("language cookies = %#v", cookies)
	}

	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), "non è configurata") {
		t.Fatal("language cookie did not select Italian")
	}
}

func TestTranslationsFallbackToEnglish(t *testing.T) {
	messages, err := loadCatalog()
	if err != nil {
		t.Fatalf("loadCatalog() error = %v", err)
	}
	delete(messages["it"], "home.tagline")
	if got := messages.text("it", "home.tagline"); got != "Your turn, made simple." {
		t.Fatalf("fallback translation = %q", got)
	}
	if got := messages.text("it", "missing.key"); got != "missing.key" {
		t.Fatalf("missing translation = %q", got)
	}
}

func TestDictionariesContainTheSameKeys(t *testing.T) {
	messages, err := loadCatalog()
	if err != nil {
		t.Fatalf("loadCatalog() error = %v", err)
	}
	for key := range messages["en"] {
		if messages["it"][key] == "" {
			t.Errorf("Italian dictionary is missing %q", key)
		}
	}
}

func TestAdminLoginCreateQueueAndCSRF(t *testing.T) {
	handler, store := testHandler(t)
	const adminToken = "test-administrator-token-32-chars"
	tokenHash := sha256.Sum256([]byte(adminToken))
	if _, err := store.BootstrapAdmin(context.Background(), tokenHash[:]); err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}

	login := formRequest(http.MethodPost, "/admin/login", url.Values{"token": {adminToken}}, nil)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d, want %d", loginResponse.Code, http.StatusSeeOther)
	}
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("login cookies = %#v, want one HttpOnly SameSite=Strict cookie", cookies)
	}

	dashboard := httptest.NewRequest(http.MethodGet, "/admin", nil)
	dashboard.AddCookie(cookies[0])
	dashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardResponse, dashboard)
	csrfMatch := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(dashboardResponse.Body.String())
	if len(csrfMatch) != 2 {
		t.Fatal("dashboard does not contain a CSRF token")
	}

	missingCSRF := formRequest(http.MethodPost, "/admin/queues", url.Values{}, cookies[0])
	missingCSRFResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingCSRFResponse, missingCSRF)
	if missingCSRFResponse.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, want %d", missingCSRFResponse.Code, http.StatusForbidden)
	}

	create := formRequest(http.MethodPost, "/admin/queues", url.Values{
		"csrf_token": {csrfMatch[1]}, "name": {"General"}, "prefix": {"A"},
		"digits": {"3"}, "min_value": {"1"}, "max_value": {"999"},
	}, cookies[0])
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusSeeOther {
		t.Fatalf("create queue status = %d, want %d", createResponse.Code, http.StatusSeeOther)
	}
	queues, err := store.ListQueues(context.Background())
	if err != nil || len(queues) != 1 || queues[0].Name != "General" {
		t.Fatalf("ListQueues() = %#v, %v", queues, err)
	}

	updateSettings := formRequest(http.MethodPost, "/admin/settings", url.Values{
		"csrf_token": {csrfMatch[1]}, "installation_name": {"Municipal Hall"},
		"default_language": {"it"}, "recent_calls_limit": {"12"},
		"highlight_duration_seconds": {"7"},
	}, cookies[0])
	updateSettingsResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateSettingsResponse, updateSettings)
	if updateSettingsResponse.Code != http.StatusSeeOther {
		t.Fatalf("update settings status = %d, want %d", updateSettingsResponse.Code, http.StatusSeeOther)
	}
	if location := updateSettingsResponse.Header().Get("Location"); location != "/admin?notice=settings-saved" {
		t.Fatalf("update settings redirect = %q, want settings confirmation", location)
	}
	settings, err := store.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings.InstallationName != "Municipal Hall" || settings.DefaultLanguage != "it" ||
		settings.RecentCallsLimit != 12 || settings.SoundEnabled ||
		settings.HighlightDurationSeconds != 7 {
		t.Fatalf("settings after form update = %#v", settings)
	}

	savedDashboard := httptest.NewRequest(http.MethodGet, "/admin?notice=settings-saved", nil)
	savedDashboard.AddCookie(cookies[0])
	savedDashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(savedDashboardResponse, savedDashboard)
	if !strings.Contains(savedDashboardResponse.Body.String(), "Impostazioni salvate.") {
		t.Fatal("saved settings confirmation is missing")
	}
}

func TestUnknownPageReturnsNotFound(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func testHandler(t *testing.T) (http.Handler, *sqlitestore.Store) {
	t.Helper()
	store, err := sqlitestore.Open(context.Background(), filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })
	handler, err := New("test-version", store)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return handler, store
}

func formRequest(method, target string, values url.Values, cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return request
}
