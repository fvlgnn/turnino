package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fvlgnn/turnino/internal/credential"
	"github.com/fvlgnn/turnino/internal/httpapi"
	sqlitestore "github.com/fvlgnn/turnino/internal/store/sqlite"
	"github.com/fvlgnn/turnino/internal/webui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("turnino stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "admin":
			return runAdminCommand(os.Args[2:], os.Stdout)
		case "backup":
			return runBackupCommand(os.Args[2:], os.Stdout)
		case "export":
			return runExportCommand(os.Args[2:], os.Stdout)
		case "demo":
			return runDemoCommand(os.Args[2:], os.Stdout)
		}
	}

	addr := flag.String("addr", envOr("TURNINO_ADDR", ":8080"), "HTTP listen address")
	dataDir := flag.String("data-dir", envOr("TURNINO_DATA_DIR", "./data"), "persistent data directory")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("turnino %s (commit %s, built %s)\n", version, commit, date)
		return nil
	}

	if err := os.MkdirAll(*dataDir, 0o750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	db, err := sqlitestore.Open(context.Background(), filepath.Join(*dataDir, "turnino.db"))
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()
	adminToken := strings.TrimSpace(os.Getenv("TURNINO_ADMIN_TOKEN"))
	if adminToken != "" {
		if len(adminToken) < credential.MinAdminTokenLength {
			return fmt.Errorf("TURNINO_ADMIN_TOKEN must contain at least %d characters", credential.MinAdminTokenLength)
		}
		tokenHash := credential.Hash(adminToken)
		created, err := db.BootstrapAdmin(context.Background(), tokenHash[:])
		if err != nil {
			return fmt.Errorf("bootstrap administrator: %w", err)
		}
		if created {
			slog.Info("administrator credential configured; TURNINO_ADMIN_TOKEN can be removed before the next restart")
		} else {
			valid, err := db.ValidateAdminToken(context.Background(), tokenHash[:])
			if err != nil {
				return fmt.Errorf("validate configured administrator: %w", err)
			}
			if !valid {
				return errors.New("TURNINO_ADMIN_TOKEN does not match the administrator credential already stored")
			}
		}
	}
	adminConfigured, err := db.AdminConfigured(context.Background())
	if err != nil {
		return fmt.Errorf("check administrator configuration: %w", err)
	}
	if !adminConfigured {
		slog.Warn("administrator is not configured; set TURNINO_ADMIN_TOKEN and restart to enable administration APIs")
	}

	ui, err := webui.New(version, db)
	if err != nil {
		return fmt.Errorf("initialize web UI: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", httpapi.New(db)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/", ui)

	requestContext, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	server := &http.Server{
		Addr:              *addr,
		Handler:           securityHeaders(requestLogger(mux)),
		BaseContext:       func(net.Listener) context.Context { return requestContext },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("turnino listening", "addr", *addr, "data_dir", filepath.Clean(*dataDir), "version", version)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	case <-ctx.Done():
		stop()
		slog.Info("turnino stopping")
		if err := stopHTTPServer(server, cancelRequests, 10*time.Second); err != nil {
			return err
		}
		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP while stopping: %w", err)
		}
	}

	return nil
}

func stopHTTPServer(server *http.Server, cancelRequests context.CancelFunc, timeout time.Duration) error {
	cancelRequests()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return fmt.Errorf("force HTTP server close after graceful shutdown: %w", closeErr)
		}
		slog.Warn("graceful shutdown timed out; remaining connections were closed", "error", err)
	}
	return nil
}

func runAdminCommand(args []string, output io.Writer) error {
	if len(args) == 0 || args[0] != "reset" {
		return errors.New("usage: turnino admin reset --data-dir PATH --confirm")
	}
	flags := flag.NewFlagSet("admin reset", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dataDir := flags.String("data-dir", envOr("TURNINO_DATA_DIR", "./data"), "persistent data directory")
	confirm := flags.Bool("confirm", false, "confirm the offline credential reset")
	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("parse admin reset options: %w", err)
	}
	if !*confirm {
		return errors.New("admin reset requires --confirm; stop Turnino before running it")
	}
	db, err := sqlitestore.Open(context.Background(), filepath.Join(*dataDir, "turnino.db"))
	if err != nil {
		return fmt.Errorf("open database for offline admin reset: %w", err)
	}
	defer db.Close()
	token, err := credential.Generate()
	if err != nil {
		return fmt.Errorf("generate administrator credential: %w", err)
	}
	tokenHash := credential.Hash(token)
	if err := db.ResetAdmin(context.Background(), tokenHash[:]); err != nil {
		return err
	}
	fmt.Fprintln(output, "Administrator credential reset successfully.")
	fmt.Fprintln(output, "Store this token securely; it will not be shown again:")
	fmt.Fprintln(output, token)
	return nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
