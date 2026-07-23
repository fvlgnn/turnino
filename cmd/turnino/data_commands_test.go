package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
)

func TestDemoBackupAndCallExportCommands(t *testing.T) {
	dataDir := t.TempDir()
	databasePath := filepath.Join(dataDir, "turnino.db")
	if err := runDemoCommand([]string{"create", "--output", databasePath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runDemoCommand() error = %v", err)
	}

	store, err := sqlitestore.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatalf("Open(demo) error = %v", err)
	}
	defer store.Close()
	queues, err := store.ListQueues(context.Background())
	if err != nil || len(queues) != 2 {
		t.Fatalf("demo queues = %d, %v; want 2, nil", len(queues), err)
	}
	counters, err := store.ListCounters(context.Background())
	if err != nil || len(counters) != 2 {
		t.Fatalf("demo counters = %d, %v; want 2, nil", len(counters), err)
	}
	records, err := store.ListCallRecords(context.Background())
	if err != nil || len(records) != 11 {
		t.Fatalf("demo calls = %d, %v; want 11, nil", len(records), err)
	}
	adminConfigured, err := store.AdminConfigured(context.Background())
	if err != nil || adminConfigured {
		t.Fatalf("demo AdminConfigured() = %v, %v; want false, nil", adminConfigured, err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := runBackupCommand([]string{"--data-dir", dataDir, "--output", backupPath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runBackupCommand() error = %v", err)
	}
	backup, err := sqlitestore.Open(context.Background(), backupPath)
	if err != nil {
		t.Fatalf("Open(backup) error = %v", err)
	}
	backupRecords, err := backup.ListCallRecords(context.Background())
	backup.Close()
	if err != nil || len(backupRecords) != len(records) {
		t.Fatalf("backup calls = %d, %v; want %d, nil", len(backupRecords), err, len(records))
	}

	exportPath := filepath.Join(t.TempDir(), "calls.csv")
	if err := runExportCommand([]string{"calls", "--data-dir", dataDir, "--output", exportPath}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runExportCommand() error = %v", err)
	}
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("open CSV export: %v", err)
	}
	rows, err := csv.NewReader(file).ReadAll()
	file.Close()
	if err != nil {
		t.Fatalf("read CSV export: %v", err)
	}
	if len(rows) != len(records)+1 {
		t.Fatalf("CSV rows = %d, want %d", len(rows), len(records)+1)
	}
	wantHeader := "id,number,kind,queue_id,queue,counter_id,counter,called_at"
	if got := strings.Join(rows[0], ","); got != wantHeader {
		t.Fatalf("CSV header = %q, want %q", got, wantHeader)
	}
}

func TestDataCommandsRefuseToOverwriteOutputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := runDemoCommand([]string{"create", "--output", path}, &bytes.Buffer{}); err == nil {
		t.Fatal("runDemoCommand() error = nil, want existing-output error")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing output changed: %q, %v", content, err)
	}
}

func TestBackupDoesNotCreateMissingSourceDatabase(t *testing.T) {
	dataDir := t.TempDir()
	output := filepath.Join(t.TempDir(), "backup.db")
	if err := runBackupCommand([]string{"--data-dir", dataDir, "--output", output}, &bytes.Buffer{}); err == nil {
		t.Fatal("runBackupCommand() error = nil, want missing-source error")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "turnino.db")); !os.IsNotExist(err) {
		t.Fatalf("missing source database was created: %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("backup output was created after source error: %v", err)
	}
}
