package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fvlgnn/turnino/internal/credential"
	"github.com/fvlgnn/turnino/internal/domain"
	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
)

func runBackupCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("backup", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dataDir := flags.String("data-dir", envOr("TURNINO_DATA_DIR", "./data"), "persistent data directory")
	destination := flags.String("output", "", "new backup database path")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse backup options: %w", err)
	}
	if *destination == "" {
		return errors.New("usage: turnino backup --output PATH [--data-dir PATH]")
	}
	sourcePath, err := existingDatabase(*dataDir)
	if err != nil {
		return err
	}
	destinationPath, err := prepareNewOutput(*destination)
	if err != nil {
		return err
	}
	if destinationPath == sourcePath {
		return errors.New("backup destination must differ from the source database")
	}
	if err := reserveOutput(destinationPath); err != nil {
		return err
	}
	store, err := sqlitestore.Open(context.Background(), sourcePath)
	if err != nil {
		return fmt.Errorf("open database for backup: %w", err)
	}
	defer store.Close()
	if err := store.Backup(context.Background(), destinationPath); err != nil {
		os.Remove(destinationPath)
		return err
	}
	if err := os.Chmod(destinationPath, 0o600); err != nil {
		return fmt.Errorf("protect backup file: %w", err)
	}
	fmt.Fprintf(output, "Backup created: %s\n", destinationPath)
	return nil
}

func runExportCommand(args []string, output io.Writer) error {
	if len(args) == 0 || args[0] != "calls" {
		return errors.New("usage: turnino export calls --output PATH [--data-dir PATH]")
	}
	flags := flag.NewFlagSet("export calls", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dataDir := flags.String("data-dir", envOr("TURNINO_DATA_DIR", "./data"), "persistent data directory")
	destination := flags.String("output", "", "new CSV output path")
	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse export options: %w", err)
	}
	if *destination == "" {
		return errors.New("usage: turnino export calls --output PATH [--data-dir PATH]")
	}
	destinationPath, err := prepareNewOutput(*destination)
	if err != nil {
		return err
	}
	sourcePath, err := existingDatabase(*dataDir)
	if err != nil {
		return err
	}
	store, err := sqlitestore.Open(context.Background(), sourcePath)
	if err != nil {
		return fmt.Errorf("open database for export: %w", err)
	}
	defer store.Close()
	records, err := store.ListCallRecords(context.Background())
	if err != nil {
		return err
	}
	file, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create call export: %w", err)
	}
	completed := false
	defer func() {
		file.Close()
		if !completed {
			os.Remove(destinationPath)
		}
	}()
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"id", "number", "kind", "queue_id", "queue", "counter_id", "counter", "called_at"}); err != nil {
		return fmt.Errorf("write call export header: %w", err)
	}
	for _, record := range records {
		if err := writer.Write([]string{
			strconv.FormatInt(record.ID, 10), record.Number, string(record.Kind),
			strconv.FormatInt(record.QueueID, 10), record.QueueName,
			strconv.FormatInt(record.CounterID, 10), record.CounterName,
			record.CalledAt.Format("2006-01-02T15:04:05Z"),
		}); err != nil {
			return fmt.Errorf("write call export row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush call export: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close call export: %w", err)
	}
	completed = true
	fmt.Fprintf(output, "Call history exported: %s (%d records)\n", destinationPath, len(records))
	return nil
}

func runDemoCommand(args []string, output io.Writer) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage: turnino demo create --output PATH")
	}
	flags := flag.NewFlagSet("demo create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	destination := flags.String("output", "", "new demonstration database path")
	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse demo options: %w", err)
	}
	if *destination == "" {
		return errors.New("usage: turnino demo create --output PATH")
	}
	destinationPath, err := prepareNewOutput(*destination)
	if err != nil {
		return err
	}
	if err := reserveOutput(destinationPath); err != nil {
		return err
	}
	if err := createDemoDatabase(context.Background(), destinationPath); err != nil {
		os.Remove(destinationPath)
		return err
	}
	if err := os.Chmod(destinationPath, 0o600); err != nil {
		return fmt.Errorf("protect demo database: %w", err)
	}
	fmt.Fprintf(output, "Synthetic demonstration database created: %s\n", destinationPath)
	return nil
}

func createDemoDatabase(ctx context.Context, destination string) error {
	store, err := sqlitestore.Open(ctx, destination)
	if err != nil {
		return fmt.Errorf("create demonstration database: %w", err)
	}
	defer store.Close()
	general, err := store.CreateQueue(ctx, domain.CreateQueueParams{Name: "General services", Prefix: "A", Digits: 3, MinValue: 1, MaxValue: 999})
	if err != nil {
		return err
	}
	information, err := store.CreateQueue(ctx, domain.CreateQueueParams{Name: "Information", Prefix: "I", Digits: 2, MinValue: 1, MaxValue: 99})
	if err != nil {
		return err
	}
	firstToken, err := credential.Generate()
	if err != nil {
		return err
	}
	firstHash := credential.Hash(firstToken)
	if _, err := store.CreateCounter(ctx, "Counter 1", firstHash[:]); err != nil {
		return err
	}
	secondToken, err := credential.Generate()
	if err != nil {
		return err
	}
	secondHash := credential.Hash(secondToken)
	if _, err := store.CreateCounter(ctx, "Counter 2", secondHash[:]); err != nil {
		return err
	}
	for index := 0; index < 8; index++ {
		hash := firstHash[:]
		if index%3 == 2 {
			hash = secondHash[:]
		}
		if _, err := store.CallNext(ctx, general.ID, hash); err != nil {
			return err
		}
	}
	for range 3 {
		if _, err := store.CallNext(ctx, information.ID, secondHash[:]); err != nil {
			return err
		}
	}
	return nil
}

func prepareNewOutput(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	if _, err := os.Stat(absolute); err == nil {
		return "", fmt.Errorf("output already exists: %s", absolute)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o750); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	return absolute, nil
}

func reserveOutput(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("reserve output file: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("close output file: %w", err)
	}
	return nil
}

func existingDatabase(dataDir string) (string, error) {
	path, err := filepath.Abs(filepath.Join(dataDir, "turnino.db"))
	if err != nil {
		return "", fmt.Errorf("resolve source database path: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("open source database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("source database is not a regular file: %s", path)
	}
	return path, nil
}
