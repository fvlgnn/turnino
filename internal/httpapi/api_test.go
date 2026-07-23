package httpapi_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fvlgnn/turnino/internal/httpapi"
	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
)

func TestAdminAndCounterTokenFlow(t *testing.T) {
	const adminToken = "test-administrator-token-32-chars"
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })
	adminHash := sha256.Sum256([]byte(adminToken))
	created, err := store.BootstrapAdmin(ctx, adminHash[:])
	if err != nil || !created {
		t.Fatalf("BootstrapAdmin() = %v, %v; want true, nil", created, err)
	}

	server := httptest.NewServer(httpapi.New(store))
	t.Cleanup(server.Close)
	requestJSON(t, http.MethodPost, server.URL+"/admin/queues", `{}`, "", http.StatusUnauthorized)

	queue := requestJSON(t, http.MethodPost, server.URL+"/admin/queues", `{
		"name":"General", "prefix":"A", "digits":3, "min_value":1, "max_value":999
	}`, adminToken, http.StatusCreated)
	counter := requestJSON(t, http.MethodPost, server.URL+"/admin/counters", `{"name":"Counter 1"}`, adminToken, http.StatusCreated)
	queueID := strconv.Itoa(int(queue["id"].(float64)))
	counterID := strconv.Itoa(int(counter["id"].(float64)))
	counterToken := counter["token"].(string)

	currentCounter := requestJSON(t, http.MethodGet, server.URL+"/counters/me", "", counterToken, http.StatusOK)
	if got := currentCounter["name"]; got != "Counter 1" {
		t.Fatalf("current counter name = %v, want Counter 1", got)
	}

	call := requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", counterToken, http.StatusCreated)
	if got := call["number"]; got != "A001" {
		t.Fatalf("call number = %v, want A001", got)
	}

	rotated := requestJSON(t, http.MethodPost, server.URL+"/admin/counters/"+counterID+"/token", "", adminToken, http.StatusOK)
	requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", counterToken, http.StatusUnauthorized)
	newCounterToken := rotated["token"].(string)
	requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", newCounterToken, http.StatusCreated)

	requestJSON(t, http.MethodDelete, server.URL+"/admin/counters/"+counterID, "", adminToken, http.StatusNoContent)
	requestJSON(t, http.MethodGet, server.URL+"/counters/me", "", newCounterToken, http.StatusUnauthorized)
	requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", newCounterToken, http.StatusUnauthorized)
	reactivated := requestJSON(t, http.MethodPost, server.URL+"/admin/counters/"+counterID+"/reactivate", "", adminToken, http.StatusOK)
	requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", reactivated["token"].(string), http.StatusCreated)
}

func TestAdminAPIUnavailableBeforeBootstrap(t *testing.T) {
	store, err := sqlitestore.Open(context.Background(), filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })
	server := httptest.NewServer(httpapi.New(store))
	t.Cleanup(server.Close)

	requestJSON(t, http.MethodPost, server.URL+"/admin/queues", `{}`, "any-token", http.StatusServiceUnavailable)
}

func requestJSON(t *testing.T, method, url, body, token string, wantStatus int) map[string]any {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s error = %v", method, url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d", method, url, response.StatusCode, wantStatus)
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode %s %s response: %v", method, url, err)
	}
	return result
}
