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
		"Operator workstation",
		"Public monitor",
		`class="home-actions"`,
		`href="/caller"`,
		`href="/about"`,
		"test-version",
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestAboutPageOffersLicenseAndSource(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/about", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"About Turnino",
		"GNU AGPL-3.0-only",
		"Copyright (c) 2026-present Gianni Favilli (fvlgnn)",
		`href="https://github.com/fvlgnn/turnino"`,
		`href="https://github.com/fvlgnn/turnino/blob/main/LICENSE"`,
		`href="https://github.com/fvlgnn/turnino/blob/main/NOTICE"`,
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
		Name: "General", Prefix: "A", Digits: 3, MinValue: 1, MaxValue: 999, AllowCustomCalls: true,
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
		"Workstation settings",
		"Open workstation display",
		`id="caller-identity"`,
		"General",
		`data-queue-id="1"`,
		`class="caller-recall secondary"`,
		`class="caller-custom-form"`,
		`src="/static/caller.js"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
	body := response.Body.String()
	if strings.Contains(body, `id="caller-forget-token"`) {
		t.Error("caller unexpectedly exposes the forget-token action")
	}
	if strings.Index(body, `class="caller-queues"`) > strings.Index(body, `id="caller-token-settings"`) {
		t.Error("workstation settings appear before the primary queue actions")
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

func TestCounterDisplayPage(t *testing.T) {
	handler, _ := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/counter-display", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Counter display",
		"Waiting for a call",
		`src="/static/counter-display.js"`,
		`data-not-configured="Configure this display from the administration page."`,
		`data-display-counter-id="0"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q", expected)
		}
	}
}

func TestCounterDisplayCanBeConfiguredByCounterID(t *testing.T) {
	handler, store := testHandler(t)
	tokenHash := sha256.Sum256([]byte("counter-display-token"))
	counter, err := store.CreateCounter(context.Background(), "Counter 1", tokenHash[:])
	if err != nil {
		t.Fatalf("CreateCounter() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/counter-display?counter=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		`data-display-counter-id="1"`,
		`data-display-counter-name="Counter 1"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response does not contain %q for counter %#v", expected, counter)
		}
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
	if !strings.Contains(response.Body.String(), "non è ancora stato inizializzato") {
		t.Fatal("language cookie did not select Italian")
	}
	for _, expected := range []string{
		"./turnino admin init --data-dir ./data",
		"docker compose run --rm turnino admin init --data-dir /data",
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("unconfigured administration page does not contain %q", expected)
		}
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

	enableCustomCalls := formRequest(http.MethodPost, "/admin/queues/1/custom-calls", url.Values{
		"csrf_token": {csrfMatch[1]}, "enabled": {"true"},
	}, cookies[0])
	enableCustomCallsResponse := httptest.NewRecorder()
	handler.ServeHTTP(enableCustomCallsResponse, enableCustomCalls)
	if enableCustomCallsResponse.Code != http.StatusSeeOther {
		t.Fatalf("enable custom calls status = %d, want %d", enableCustomCallsResponse.Code, http.StatusSeeOther)
	}
	queues, err = store.ListQueues(context.Background())
	if err != nil || len(queues) != 1 || !queues[0].AllowCustomCalls {
		t.Fatalf("queues after enabling custom calls = %#v, %v", queues, err)
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

	rotate := formRequest(http.MethodPost, "/admin/access-key/rotate", url.Values{
		"csrf_token": {csrfMatch[1]}, "confirm_rotation": {"on"},
	}, cookies[0])
	rotateResponse := httptest.NewRecorder()
	handler.ServeHTTP(rotateResponse, rotate)
	if rotateResponse.Code != http.StatusOK {
		t.Fatalf("rotate admin access key status = %d, want %d", rotateResponse.Code, http.StatusOK)
	}
	keyMatch := regexp.MustCompile(`<code>([^<]+)</code>`).FindStringSubmatch(rotateResponse.Body.String())
	if len(keyMatch) != 2 {
		t.Fatal("rotated administrator access key is not shown")
	}
	newKeyHash := sha256.Sum256([]byte(keyMatch[1]))
	valid, err := store.ValidateAdminToken(context.Background(), newKeyHash[:])
	if err != nil || !valid {
		t.Fatalf("new administrator access key validation = %v, %v; want true, nil", valid, err)
	}
	valid, err = store.ValidateAdminToken(context.Background(), tokenHash[:])
	if err != nil || valid {
		t.Fatalf("old administrator access key validation = %v, %v; want false, nil", valid, err)
	}

	oldSession := httptest.NewRequest(http.MethodGet, "/admin", nil)
	oldSession.AddCookie(cookies[0])
	oldSessionResponse := httptest.NewRecorder()
	handler.ServeHTTP(oldSessionResponse, oldSession)
	if !strings.Contains(oldSessionResponse.Body.String(), "Chiave di accesso amministrativa") {
		t.Fatal("rotation did not invalidate the existing administrator session")
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
