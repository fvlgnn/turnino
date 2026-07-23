package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

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

func TestAdminResetRequiresConfirmation(t *testing.T) {
	if err := runAdminCommand([]string{"reset", "--data-dir", t.TempDir()}, &bytes.Buffer{}); err == nil {
		t.Fatal("runAdminCommand() error = nil, want confirmation error")
	}
}
