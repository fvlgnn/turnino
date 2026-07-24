package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fvlgnn/turnino/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	dsn, err := dataSourceName(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	return nil
}

func (s *Store) Backup(ctx context.Context, destination string) error {
	if strings.TrimSpace(destination) == "" {
		return errors.New("backup destination is required")
	}
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return fmt.Errorf("backup sqlite database: %w", err)
	}
	return nil
}

func (s *Store) BootstrapAdmin(ctx context.Context, tokenHash []byte) (bool, error) {
	if len(tokenHash) != 32 {
		return false, errors.New("admin token hash must be 32 bytes")
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_credentials (id, token_hash)
		VALUES (1, ?)
		ON CONFLICT (id) DO NOTHING`, tokenHash)
	if err != nil {
		return false, fmt.Errorf("bootstrap admin credential: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read bootstrap admin result: %w", err)
	}
	return created == 1, nil
}

func (s *Store) AdminConfigured(ctx context.Context) (bool, error) {
	var configured bool
	if err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM admin_credentials WHERE id = 1)",
	).Scan(&configured); err != nil {
		return false, fmt.Errorf("check admin credential: %w", err)
	}
	return configured, nil
}

func (s *Store) ValidateAdminToken(ctx context.Context, tokenHash []byte) (bool, error) {
	if len(tokenHash) != 32 {
		return false, nil
	}
	var valid bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM admin_credentials WHERE id = 1 AND token_hash = ?
		)`, tokenHash,
	).Scan(&valid); err != nil {
		return false, fmt.Errorf("validate admin token: %w", err)
	}
	return valid, nil
}

func (s *Store) ResetAdmin(ctx context.Context, tokenHash []byte) error {
	if len(tokenHash) != 32 {
		return errors.New("admin token hash must be 32 bytes")
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_credentials (id, token_hash)
		VALUES (1, ?)
		ON CONFLICT (id) DO UPDATE SET token_hash = excluded.token_hash`, tokenHash); err != nil {
		return fmt.Errorf("reset admin credential: %w", err)
	}
	return nil
}

func (s *Store) GetSettings(ctx context.Context) (domain.Settings, error) {
	var settings domain.Settings
	err := s.db.QueryRowContext(ctx, `
		SELECT installation_name, default_language, recent_calls_limit,
		       sound_enabled, highlight_duration_seconds
		FROM settings WHERE id = 1`,
	).Scan(
		&settings.InstallationName,
		&settings.DefaultLanguage,
		&settings.RecentCallsLimit,
		&settings.SoundEnabled,
		&settings.HighlightDurationSeconds,
	)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("get settings: %w", err)
	}
	return settings, nil
}

func (s *Store) UpdateSettings(ctx context.Context, settings domain.Settings) (domain.Settings, error) {
	if err := settings.NormalizeAndValidate(); err != nil {
		return domain.Settings{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE settings
		SET installation_name = ?, default_language = ?, recent_calls_limit = ?,
		    sound_enabled = ?, highlight_duration_seconds = ?
		WHERE id = 1`,
		settings.InstallationName,
		settings.DefaultLanguage,
		settings.RecentCallsLimit,
		settings.SoundEnabled,
		settings.HighlightDurationSeconds,
	)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("update settings: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return domain.Settings{}, fmt.Errorf("read updated settings count: %w", err)
	}
	if updated != 1 {
		return domain.Settings{}, errors.New("settings row is missing")
	}
	return settings, nil
}

func (s *Store) CreateQueue(ctx context.Context, params domain.CreateQueueParams) (domain.Queue, error) {
	if err := params.NormalizeAndValidate(); err != nil {
		return domain.Queue{}, err
	}

	queue := domain.Queue{
		Name:             params.Name,
		Prefix:           params.Prefix,
		Digits:           params.Digits,
		MinValue:         params.MinValue,
		MaxValue:         params.MaxValue,
		CurrentValue:     params.MinValue - 1,
		AllowCustomCalls: params.AllowCustomCalls,
	}
	var createdAt int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO queues (name, prefix, digits, min_value, max_value, current_value, allow_custom_calls)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		queue.Name, queue.Prefix, queue.Digits, queue.MinValue, queue.MaxValue, queue.CurrentValue, queue.AllowCustomCalls,
	).Scan(&queue.ID, &createdAt)
	if err != nil {
		return domain.Queue{}, fmt.Errorf("create queue: %w", err)
	}
	queue.CreatedAt = time.Unix(createdAt, 0).UTC()
	return queue, nil
}

func (s *Store) ListQueues(ctx context.Context) ([]domain.Queue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, prefix, digits, min_value, max_value, current_value, allow_custom_calls, created_at
		FROM queues ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}
	defer rows.Close()
	var queues []domain.Queue
	for rows.Next() {
		var queue domain.Queue
		var createdAt int64
		if err := rows.Scan(&queue.ID, &queue.Name, &queue.Prefix, &queue.Digits, &queue.MinValue, &queue.MaxValue, &queue.CurrentValue, &queue.AllowCustomCalls, &createdAt); err != nil {
			return nil, fmt.Errorf("scan queue: %w", err)
		}
		queue.CreatedAt = time.Unix(createdAt, 0).UTC()
		queues = append(queues, queue)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queues: %w", err)
	}
	return queues, nil
}

func (s *Store) SetQueueCustomCalls(ctx context.Context, queueID int64, enabled bool) error {
	result, err := s.db.ExecContext(ctx, "UPDATE queues SET allow_custom_calls = ? WHERE id = ?", enabled, queueID)
	if err != nil {
		return fmt.Errorf("update queue custom calls: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated queue count: %w", err)
	}
	if updated == 0 {
		return domain.ErrQueueNotFound
	}
	return nil
}

func (s *Store) CreateCounter(ctx context.Context, name string, tokenHash []byte) (domain.Counter, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return domain.Counter{}, errors.New("counter name must contain between 1 and 100 characters")
	}
	if len(tokenHash) != 32 {
		return domain.Counter{}, errors.New("counter token hash must be 32 bytes")
	}

	counter := domain.Counter{Name: name, Enabled: true}
	var createdAt int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO counters (name, token_hash)
		VALUES (?, ?)
		RETURNING id, created_at`, name, tokenHash,
	).Scan(&counter.ID, &createdAt)
	if err != nil {
		return domain.Counter{}, fmt.Errorf("create counter: %w", err)
	}
	counter.CreatedAt = time.Unix(createdAt, 0).UTC()
	return counter, nil
}

func (s *Store) ListCounters(ctx context.Context) ([]domain.Counter, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, enabled, created_at FROM counters ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list counters: %w", err)
	}
	defer rows.Close()
	var counters []domain.Counter
	for rows.Next() {
		var counter domain.Counter
		var createdAt int64
		if err := rows.Scan(&counter.ID, &counter.Name, &counter.Enabled, &createdAt); err != nil {
			return nil, fmt.Errorf("scan counter: %w", err)
		}
		counter.CreatedAt = time.Unix(createdAt, 0).UTC()
		counters = append(counters, counter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counters: %w", err)
	}
	return counters, nil
}

func (s *Store) CounterForToken(ctx context.Context, tokenHash []byte) (domain.Counter, error) {
	if len(tokenHash) != 32 {
		return domain.Counter{}, domain.ErrCounterUnavailable
	}
	var counter domain.Counter
	var createdAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, enabled, created_at
		FROM counters
		WHERE token_hash = ? AND enabled = 1`, tokenHash,
	).Scan(&counter.ID, &counter.Name, &counter.Enabled, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Counter{}, domain.ErrCounterUnavailable
	}
	if err != nil {
		return domain.Counter{}, fmt.Errorf("get counter for token: %w", err)
	}
	counter.CreatedAt = time.Unix(createdAt, 0).UTC()
	return counter, nil
}

func (s *Store) RevokeCounter(ctx context.Context, counterID int64) error {
	result, err := s.db.ExecContext(ctx, "UPDATE counters SET enabled = 0 WHERE id = ?", counterID)
	if err != nil {
		return fmt.Errorf("revoke counter: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked counter count: %w", err)
	}
	if updated == 0 {
		return domain.ErrCounterNotFound
	}
	return nil
}

func (s *Store) RotateCounterToken(ctx context.Context, counterID int64, tokenHash []byte) error {
	if len(tokenHash) != 32 {
		return errors.New("counter token hash must be 32 bytes")
	}
	result, err := s.db.ExecContext(ctx, "UPDATE counters SET token_hash = ? WHERE id = ?", tokenHash, counterID)
	if err != nil {
		return fmt.Errorf("rotate counter token: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rotated counter count: %w", err)
	}
	if updated == 0 {
		return domain.ErrCounterNotFound
	}
	return nil
}

func (s *Store) ReactivateCounter(ctx context.Context, counterID int64, tokenHash []byte) error {
	if len(tokenHash) != 32 {
		return errors.New("counter token hash must be 32 bytes")
	}
	result, err := s.db.ExecContext(ctx,
		"UPDATE counters SET token_hash = ?, enabled = 1 WHERE id = ?", tokenHash, counterID)
	if err != nil {
		return fmt.Errorf("reactivate counter: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read reactivated counter count: %w", err)
	}
	if updated == 0 {
		return domain.ErrCounterNotFound
	}
	return nil
}

func (s *Store) CallNext(ctx context.Context, queueID int64, tokenHash []byte) (domain.Call, error) {
	if len(tokenHash) != 32 {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Call{}, fmt.Errorf("begin call transaction: %w", err)
	}
	defer tx.Rollback()

	var prefix string
	var digits, value int
	err = tx.QueryRowContext(ctx, `
		UPDATE queues
		SET current_value = CASE
			WHEN current_value >= max_value THEN min_value
			ELSE current_value + 1
		END
		WHERE id = ?
		RETURNING prefix, digits, current_value`, queueID,
	).Scan(&prefix, &digits, &value)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrQueueNotFound
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("advance queue: %w", err)
	}

	call := domain.Call{
		QueueID:       queueID,
		Number:        domain.FormatNumber(prefix, digits, value),
		SequenceValue: value,
		Kind:          domain.CallKindAutomatic,
	}
	var calledAt int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO calls (queue_id, counter_id, number, sequence_value, kind)
		SELECT ?, id, ?, ?, ?
		FROM counters
		WHERE token_hash = ? AND enabled = 1
		RETURNING id, counter_id, called_at`,
		call.QueueID, call.Number, call.SequenceValue, call.Kind, tokenHash,
	).Scan(&call.ID, &call.CounterID, &calledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("record call: %w", err)
	}
	call.CalledAt = time.Unix(calledAt, 0).UTC()

	if err := tx.Commit(); err != nil {
		return domain.Call{}, fmt.Errorf("commit call transaction: %w", err)
	}
	return call, nil
}

func (s *Store) CallCustom(ctx context.Context, queueID int64, sequenceValue int, tokenHash []byte) (domain.Call, error) {
	if len(tokenHash) != 32 {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Call{}, fmt.Errorf("begin custom call transaction: %w", err)
	}
	defer tx.Rollback()

	var prefix string
	var digits, minValue, maxValue int
	var allowed bool
	err = tx.QueryRowContext(ctx, `
		UPDATE queues SET current_value = current_value
		WHERE id = ?
		RETURNING prefix, digits, min_value, max_value, allow_custom_calls`, queueID,
	).Scan(&prefix, &digits, &minValue, &maxValue, &allowed)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrQueueNotFound
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("load queue for custom call: %w", err)
	}
	if !allowed {
		return domain.Call{}, domain.ErrCustomCallsDisabled
	}
	if sequenceValue < minValue || sequenceValue > maxValue {
		return domain.Call{}, domain.ErrSequenceOutOfRange
	}

	call := domain.Call{
		QueueID:       queueID,
		Number:        domain.FormatNumber(prefix, digits, sequenceValue),
		SequenceValue: sequenceValue,
		Kind:          domain.CallKindManual,
	}
	var calledAt int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO calls (queue_id, counter_id, number, sequence_value, kind)
		SELECT ?, id, ?, ?, ?
		FROM counters
		WHERE token_hash = ? AND enabled = 1
		RETURNING id, counter_id, called_at`,
		call.QueueID, call.Number, call.SequenceValue, call.Kind, tokenHash,
	).Scan(&call.ID, &call.CounterID, &calledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("record custom call: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE queues SET current_value = ? WHERE id = ?", sequenceValue, queueID); err != nil {
		return domain.Call{}, fmt.Errorf("synchronize queue after custom call: %w", err)
	}
	call.CalledAt = time.Unix(calledAt, 0).UTC()
	if err := tx.Commit(); err != nil {
		return domain.Call{}, fmt.Errorf("commit custom call transaction: %w", err)
	}
	return call, nil
}

func (s *Store) RecallLast(ctx context.Context, queueID int64, tokenHash []byte) (domain.Call, error) {
	if len(tokenHash) != 32 {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Call{}, fmt.Errorf("begin recall transaction: %w", err)
	}
	defer tx.Rollback()

	var lockedQueueID int64
	err = tx.QueryRowContext(ctx, `
		UPDATE queues SET current_value = current_value
		WHERE id = ?
		RETURNING id`, queueID,
	).Scan(&lockedQueueID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrQueueNotFound
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("lock queue for recall: %w", err)
	}

	var counterID int64
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM counters WHERE token_hash = ? AND enabled = 1", tokenHash,
	).Scan(&counterID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrCounterUnavailable
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("load counter for recall: %w", err)
	}

	call := domain.Call{QueueID: queueID, CounterID: counterID, Kind: domain.CallKindRecall}
	err = tx.QueryRowContext(ctx, `
		SELECT number, sequence_value
		FROM calls
		WHERE queue_id = ? AND counter_id = ?
		ORDER BY id DESC
		LIMIT 1`, queueID, counterID,
	).Scan(&call.Number, &call.SequenceValue)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Call{}, domain.ErrNoPreviousCall
	}
	if err != nil {
		return domain.Call{}, fmt.Errorf("load previous call: %w", err)
	}

	var calledAt int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO calls (queue_id, counter_id, number, sequence_value, kind)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, called_at`,
		call.QueueID, call.CounterID, call.Number, call.SequenceValue, call.Kind,
	).Scan(&call.ID, &calledAt)
	if err != nil {
		return domain.Call{}, fmt.Errorf("record recall: %w", err)
	}
	call.CalledAt = time.Unix(calledAt, 0).UTC()
	if err := tx.Commit(); err != nil {
		return domain.Call{}, fmt.Errorf("commit recall transaction: %w", err)
	}
	return call, nil
}

func (s *Store) ListCallRecords(ctx context.Context) ([]domain.CallRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT calls.id, calls.queue_id, calls.counter_id, calls.number,
		       calls.sequence_value, calls.kind, calls.called_at,
		       queues.name, counters.name
		FROM calls
		JOIN queues ON queues.id = calls.queue_id
		JOIN counters ON counters.id = calls.counter_id
		ORDER BY calls.id`)
	if err != nil {
		return nil, fmt.Errorf("list call records: %w", err)
	}
	defer rows.Close()
	records := make([]domain.CallRecord, 0)
	for rows.Next() {
		var record domain.CallRecord
		var calledAt int64
		if err := rows.Scan(
			&record.ID, &record.QueueID, &record.CounterID, &record.Number,
			&record.SequenceValue, &record.Kind, &calledAt,
			&record.QueueName, &record.CounterName,
		); err != nil {
			return nil, fmt.Errorf("scan call record: %w", err)
		}
		record.CalledAt = time.Unix(calledAt, 0).UTC()
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate call records: %w", err)
	}
	return records, nil
}

func (s *Store) ListRecentCallRecords(ctx context.Context, limit int) ([]domain.CallRecord, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("recent calls limit must be between 1 and 100")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT calls.id, calls.queue_id, calls.counter_id, calls.number,
		       calls.sequence_value, calls.kind, calls.called_at,
		       queues.name, counters.name
		FROM calls
		JOIN queues ON queues.id = calls.queue_id
		JOIN counters ON counters.id = calls.counter_id
		ORDER BY calls.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent call records: %w", err)
	}
	defer rows.Close()
	records := make([]domain.CallRecord, 0, limit)
	for rows.Next() {
		record, err := scanCallRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent call record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent call records: %w", err)
	}
	return records, nil
}

func (s *Store) CallRecord(ctx context.Context, callID int64) (domain.CallRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT calls.id, calls.queue_id, calls.counter_id, calls.number,
		       calls.sequence_value, calls.kind, calls.called_at,
		       queues.name, counters.name
		FROM calls
		JOIN queues ON queues.id = calls.queue_id
		JOIN counters ON counters.id = calls.counter_id
		WHERE calls.id = ?`, callID)
	record, err := scanCallRecord(row)
	if err != nil {
		return domain.CallRecord{}, fmt.Errorf("get call record: %w", err)
	}
	return record, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanCallRecord(row rowScanner) (domain.CallRecord, error) {
	var record domain.CallRecord
	var calledAt int64
	if err := row.Scan(
		&record.ID, &record.QueueID, &record.CounterID, &record.Number,
		&record.SequenceValue, &record.Kind, &calledAt,
		&record.QueueName, &record.CounterName,
	); err != nil {
		return domain.CallRecord{}, err
	}
	record.CalledAt = time.Unix(calledAt, 0).UTC()
	return record, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL DEFAULT (unixepoch())
		)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		var applied int
		if err := s.db.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", entry.Name(),
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied == 1 {
			continue
		}

		script, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version) VALUES (?)", entry.Name(),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func dataSourceName(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	query := u.Query()
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = query.Encode()
	return u.String(), nil
}
