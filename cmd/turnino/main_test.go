package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fvlgnn/turnino/internal/credential"
	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("TURNINO_TEST_VALUE", "configured")
	if got := envOr("TURNINO_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envOr() = %q, want configured", got)
	}

	t.Setenv("TURNINO_TEST_VALUE", "")
	if got := envOr("TURNINO_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("envOr() = %q, want fallback", got)
	}
}

func TestAdminResetCommand(t *testing.T) {
	dataDir := t.TempDir()
	var output bytes.Buffer
	if err := runAdminCommand([]string{"reset", "--data-dir", dataDir, "--confirm"}, &output); err != nil {
		t.Fatalf("runAdminCommand() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3; output: %q", len(lines), output.String())
	}
	tokenHash := credential.Hash(lines[2])
	store, err := sqlitestore.Open(context.Background(), filepath.Join(dataDir, "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	valid, err := store.ValidateAdminToken(context.Background(), tokenHash[:])
	if err != nil || !valid {
		t.Fatalf("ValidateAdminToken() = %v, %v; want true, nil", valid, err)
	}
}

func TestAdminInitCommand(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "new-data-directory")
	var output bytes.Buffer
	if err := runAdminCommand([]string{"init", "--data-dir", dataDir}, &output); err != nil {
		t.Fatalf("runAdminCommand() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3; output: %q", len(lines), output.String())
	}
	keyHash := credential.Hash(lines[2])
	store, err := sqlitestore.Open(context.Background(), filepath.Join(dataDir, "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	valid, err := store.ValidateAdminToken(context.Background(), keyHash[:])
	if err != nil || !valid {
		t.Fatalf("ValidateAdminToken() = %v, %v; want true, nil", valid, err)
	}

	if err := runAdminCommand([]string{"init", "--data-dir", dataDir}, &bytes.Buffer{}); err == nil {
		t.Fatal("second admin init error = nil, want already configured error")
	}
}

func TestAdminResetRequiresConfirmation(t *testing.T) {
	if err := runAdminCommand([]string{"reset", "--data-dir", t.TempDir()}, &bytes.Buffer{}); err == nil {
		t.Fatal("runAdminCommand() error = nil, want confirmation error")
	}
}

func TestStopHTTPServerCancelsLongLivedRequests(t *testing.T) {
	requestContext, cancelRequests := context.WithCancel(context.Background())
	requestStarted := make(chan struct{})
	requestStopped := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
		close(requestStopped)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	server := &http.Server{
		Handler:     handler,
		BaseContext: func(net.Listener) context.Context { return requestContext },
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	responseErr := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if response != nil {
			response.Body.Close()
		}
		responseErr <- err
	}()
	<-requestStarted

	if err := stopHTTPServer(server, cancelRequests, time.Second); err != nil {
		t.Fatalf("stopHTTPServer() error = %v", err)
	}
	select {
	case <-requestStopped:
	case <-time.After(time.Second):
		t.Fatal("long-lived request was not canceled")
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want http.ErrServerClosed", err)
	}
	<-responseErr
}
