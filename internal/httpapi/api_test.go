package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
		"name":"General", "prefix":"A", "digits":3, "min_value":1, "max_value":999,
		"allow_custom_calls":true
	}`, adminToken, http.StatusCreated)
	counter := requestJSON(t, http.MethodPost, server.URL+"/admin/counters", `{"name":"Counter 1"}`, adminToken, http.StatusCreated)
	queueID := strconv.Itoa(int(queue["id"].(float64)))
	counterID := strconv.Itoa(int(counter["id"].(float64)))
	counterToken := counter["token"].(string)

	streamContext, cancelStream := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelStream()
	streamRequest, err := http.NewRequestWithContext(streamContext, http.MethodGet, server.URL+"/events/calls", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	streamResponse, err := http.DefaultClient.Do(streamRequest)
	if err != nil {
		t.Fatalf("open call event stream: %v", err)
	}
	defer streamResponse.Body.Close()
	if contentType := streamResponse.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("event stream content type = %q", contentType)
	}
	stream := bufio.NewReader(streamResponse.Body)
	initialEvent, initialData := readSSEEvent(t, stream)
	if initialEvent != "snapshot" || initialData != "[]" {
		t.Fatalf("initial SSE event = %q %s, want snapshot []", initialEvent, initialData)
	}

	currentCounter := requestJSON(t, http.MethodGet, server.URL+"/counters/me", "", counterToken, http.StatusOK)
	if got := currentCounter["name"]; got != "Counter 1" {
		t.Fatalf("current counter name = %v, want Counter 1", got)
	}

	call := requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", counterToken, http.StatusCreated)
	if got := call["number"]; got != "A001" {
		t.Fatalf("call number = %v, want A001", got)
	}
	eventName, eventData := readSSEEvent(t, stream)
	if eventName != "call" {
		t.Fatalf("live SSE event = %q, want call", eventName)
	}
	var liveCall map[string]any
	if err := json.Unmarshal([]byte(eventData), &liveCall); err != nil {
		t.Fatalf("decode live SSE call: %v", err)
	}
	if liveCall["number"] != "A001" || liveCall["queue"] != "General" || liveCall["counter"] != "Counter 1" {
		t.Fatalf("live SSE call = %#v", liveCall)
	}
	recalled := requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/recall", "", counterToken, http.StatusCreated)
	if recalled["number"] != "A001" || recalled["kind"] != "recall" {
		t.Fatalf("recalled call = %#v", recalled)
	}
	custom := requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/custom", `{"sequence_value":37}`, counterToken, http.StatusCreated)
	if custom["number"] != "A037" || custom["kind"] != "manual" {
		t.Fatalf("custom call = %#v", custom)
	}
	afterCustom := requestJSON(t, http.MethodPost, server.URL+"/queues/"+queueID+"/calls/next", "", counterToken, http.StatusCreated)
	if afterCustom["number"] != "A038" {
		t.Fatalf("call after custom number = %#v", afterCustom)
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

func readSSEEvent(t *testing.T, reader *bufio.Reader) (string, string) {
	t.Helper()
	var event, data string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE event: %v", err)
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			if event != "" {
				return event, data
			}
			continue
		}
		if value, found := strings.CutPrefix(line, "event: "); found {
			event = value
		}
		if value, found := strings.CutPrefix(line, "data: "); found {
			data = value
		}
	}
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
