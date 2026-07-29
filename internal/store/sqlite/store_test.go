package sqlite

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/fvlgnn/turnino/internal/domain"
)

func TestOpenAppliesMigrationsOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "turnino.db")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	defer store.Close()

	var migrations int
	if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 1 {
		t.Fatalf("migration count = %d, want 1", migrations)
	}
	for table, unwanted := range map[string]map[string]bool{
		"queues": {"current_value": true, "sequence_size": true},
		"calls":  {"sequence_value": true},
	} {
		rows, err := store.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
		if err != nil {
			t.Fatalf("inspect %s schema: %v", table, err)
		}
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				rows.Close()
				t.Fatalf("scan %s schema: %v", table, err)
			}
			if unwanted[name] {
				rows.Close()
				t.Fatalf("%s still contains legacy column %q", table, name)
			}
		}
		rows.Close()
	}
}

func TestSettingsDefaultsAndUpdate(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	settings, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings != domain.DefaultSettings() {
		t.Fatalf("default settings = %#v, want %#v", settings, domain.DefaultSettings())
	}

	want := domain.Settings{
		InstallationName:         "  Municipal Hall  ",
		DefaultLanguage:          "IT",
		RecentCallsLimit:         15,
		SoundEnabled:             false,
		HighlightDurationSeconds: 8,
	}
	updated, err := store.UpdateSettings(ctx, want)
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if updated.InstallationName != "Municipal Hall" || updated.DefaultLanguage != "it" {
		t.Fatalf("normalized settings = %#v", updated)
	}
	got, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() after update error = %v", err)
	}
	if got != updated {
		t.Fatalf("stored settings = %#v, want %#v", got, updated)
	}
}

func TestUpdateSettingsRejectsInvalidValues(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	invalid := domain.DefaultSettings()
	invalid.RecentCallsLimit = 101
	if _, err := store.UpdateSettings(ctx, invalid); err == nil {
		t.Fatal("UpdateSettings() error = nil, want validation error")
	}
	if got, err := store.GetSettings(ctx); err != nil || got != domain.DefaultSettings() {
		t.Fatalf("settings after rejected update = %#v, %v", got, err)
	}
}

func TestBootstrapAdminIsWriteOnce(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	firstHash := sha256.Sum256([]byte("first-administrator-token"))
	created, err := store.BootstrapAdmin(ctx, firstHash[:])
	if err != nil || !created {
		t.Fatalf("first BootstrapAdmin() = %v, %v; want true, nil", created, err)
	}
	secondHash := sha256.Sum256([]byte("second-administrator-token"))
	created, err = store.BootstrapAdmin(ctx, secondHash[:])
	if err != nil || created {
		t.Fatalf("second BootstrapAdmin() = %v, %v; want false, nil", created, err)
	}
	valid, err := store.ValidateAdminToken(ctx, firstHash[:])
	if err != nil || !valid {
		t.Fatalf("ValidateAdminToken(first) = %v, %v; want true, nil", valid, err)
	}
	valid, err = store.ValidateAdminToken(ctx, secondHash[:])
	if err != nil || valid {
		t.Fatalf("ValidateAdminToken(second) = %v, %v; want false, nil", valid, err)
	}
}

func TestCallNextWrapsQueueRange(t *testing.T) {
	store, queue, tokenHash := testStore(t, 1, 2)
	ctx := context.Background()

	want := []string{"A01", "A02", "A01"}
	for i, expected := range want {
		call, err := store.CallNext(ctx, queue.ID, tokenHash)
		if err != nil {
			t.Fatalf("CallNext() #%d error = %v", i+1, err)
		}
		if call.Number != expected {
			t.Fatalf("CallNext() #%d number = %q, want %q", i+1, call.Number, expected)
		}
	}
}

func TestCallCustomSynchronizesQueueWhenEnabled(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 99)
	ctx := context.Background()
	if err := store.SetQueueCustomCalls(ctx, queue.ID, true); err != nil {
		t.Fatalf("SetQueueCustomCalls() error = %v", err)
	}

	call, err := store.CallCustom(ctx, queue.ID, "A37", tokenHash)
	if err != nil {
		t.Fatalf("CallCustom() error = %v", err)
	}
	if call.Number != "A37" || call.Position != 37 || call.Kind != domain.CallKindManual {
		t.Fatalf("custom call = %#v", call)
	}
	next, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil {
		t.Fatalf("CallNext() after custom call error = %v", err)
	}
	if next.Number != "A38" {
		t.Fatalf("next number = %q, want A38", next.Number)
	}
}

func TestCallCustomRequiresEnablementAndValidRange(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 99)
	ctx := context.Background()
	if _, err := store.CallCustom(ctx, queue.ID, "A37", tokenHash); !errors.Is(err, domain.ErrCustomCallsDisabled) {
		t.Fatalf("disabled CallCustom() error = %v, want ErrCustomCallsDisabled", err)
	}
	if err := store.SetQueueCustomCalls(ctx, queue.ID, true); err != nil {
		t.Fatalf("SetQueueCustomCalls() error = %v", err)
	}
	if _, err := store.CallCustom(ctx, queue.ID, "A100", tokenHash); !errors.Is(err, domain.ErrSequenceOutOfRange) {
		t.Fatalf("out-of-range CallCustom() error = %v, want ErrSequenceOutOfRange", err)
	}
}

func TestProgressiveQueueAdvancesWrapsAndAcceptsCustomNumber(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })
	queue, err := store.CreateQueue(ctx, domain.CreateQueueParams{
		Name: "Progressive", NumberingMode: domain.NumberingModeProgressive,
		AlphabeticStart: "A", AlphabeticEnd: "B", Digits: 1,
		MinValue: 0, MaxValue: 1, AllowCustomCalls: true,
	})
	if err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	tokenHash := sha256.Sum256([]byte("progressive-counter"))
	if _, err := store.CreateCounter(ctx, "Counter", tokenHash[:]); err != nil {
		t.Fatalf("CreateCounter() error = %v", err)
	}
	for index, want := range []string{"A0", "A1", "B0", "B1", "A0"} {
		call, err := store.CallNext(ctx, queue.ID, tokenHash[:])
		if err != nil || call.Number != want {
			t.Fatalf("CallNext() #%d = %q, %v; want %q", index+1, call.Number, err, want)
		}
	}
	custom, err := store.CallCustom(ctx, queue.ID, "b1", tokenHash[:])
	if err != nil || custom.Number != "B1" {
		t.Fatalf("CallCustom() = %q, %v; want B1", custom.Number, err)
	}
	next, err := store.CallNext(ctx, queue.ID, tokenHash[:])
	if err != nil || next.Number != "A0" {
		t.Fatalf("CallNext() after B1 = %q, %v; want A0", next.Number, err)
	}
	previous, err := store.CallPrevious(ctx, queue.ID, tokenHash[:])
	if err != nil || previous.Number != "B1" || previous.Kind != domain.CallKindPrevious {
		t.Fatalf("CallPrevious() after A0 = %#v, %v; want B1", previous, err)
	}
}

func TestQueuePolicyChangePreservesCompatibleNumberAndRequiresResetOtherwise(t *testing.T) {
	ctx := context.Background()
	store, queue, tokenHash := testStore(t, 0, 99)
	if _, err := store.CallCustom(ctx, queue.ID, "A37", tokenHash); !errors.Is(err, domain.ErrCustomCallsDisabled) {
		t.Fatalf("disabled CallCustom() error = %v", err)
	}
	for range 38 {
		if _, err := store.CallNext(ctx, queue.ID, tokenHash); err != nil {
			t.Fatalf("CallNext() error = %v", err)
		}
	}
	compatible := domain.CreateQueueParams{
		Name: "General", Prefix: "A", Digits: 2, MinValue: 20, MaxValue: 99,
	}
	updated, err := store.UpdateQueuePolicy(ctx, queue.ID, compatible, false)
	if err != nil || updated.CurrentNumber() != "A37" {
		t.Fatalf("compatible UpdateQueuePolicy() current = %q, error = %v", updated.CurrentNumber(), err)
	}
	incompatible := domain.CreateQueueParams{
		Name: "General", NumberingMode: domain.NumberingModeProgressive,
		AlphabeticStart: "B", AlphabeticEnd: "C", Digits: 2, MinValue: 0, MaxValue: 99,
	}
	if _, err := store.UpdateQueuePolicy(ctx, queue.ID, incompatible, false); !errors.Is(err, domain.ErrPolicyResetRequired) {
		t.Fatalf("incompatible UpdateQueuePolicy() error = %v, want ErrPolicyResetRequired", err)
	}
	updated, err = store.UpdateQueuePolicy(ctx, queue.ID, incompatible, true)
	if err != nil || updated.CurrentPosition != -1 {
		t.Fatalf("confirmed UpdateQueuePolicy() position = %d, error = %v", updated.CurrentPosition, err)
	}
	call, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil || call.Number != "B00" {
		t.Fatalf("CallNext() after reset = %q, %v; want B00", call.Number, err)
	}
}

func TestCounterCanResetQueueWithoutDeletingHistory(t *testing.T) {
	ctx := context.Background()
	store, queue, tokenHash := testStore(t, 0, 99)
	if _, err := store.CallNext(ctx, queue.ID, tokenHash); err != nil {
		t.Fatalf("CallNext() error = %v", err)
	}
	reset, err := store.ResetQueue(ctx, queue.ID, tokenHash)
	if err != nil || reset.CurrentPosition != -1 {
		t.Fatalf("ResetQueue() position = %d, error = %v", reset.CurrentPosition, err)
	}
	records, err := store.ListRecentCallRecords(ctx, 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("ListRecentCallRecords() count = %d, error = %v; want 1", len(records), err)
	}
	call, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil || call.Number != "A00" {
		t.Fatalf("CallNext() after reset = %q, %v; want A00", call.Number, err)
	}
}

func TestRecallLastRepeatsWithoutAdvancingQueue(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 99)
	ctx := context.Background()
	first, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil {
		t.Fatalf("CallNext() error = %v", err)
	}
	recalled, err := store.RecallLast(ctx, queue.ID, tokenHash)
	if err != nil {
		t.Fatalf("RecallLast() error = %v", err)
	}
	if recalled.Number != first.Number || recalled.Position != first.Position || recalled.Kind != domain.CallKindRecall {
		t.Fatalf("recalled call = %#v, first = %#v", recalled, first)
	}
	next, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil {
		t.Fatalf("CallNext() after recall error = %v", err)
	}
	if next.Position != first.Position+1 {
		t.Fatalf("position after recall = %d, want %d", next.Position, first.Position+1)
	}
}

func TestRecallLastRequiresPreviousCallForSameCounterAndQueue(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 99)
	if _, err := store.RecallLast(context.Background(), queue.ID, tokenHash); !errors.Is(err, domain.ErrNoPreviousCall) {
		t.Fatalf("RecallLast() error = %v, want ErrNoPreviousCall", err)
	}
}

func TestListRecentCallRecordsReturnsNewestFirst(t *testing.T) {
	store, queue, tokenHash := testStore(t, 1, 9)
	ctx := context.Background()
	for range 3 {
		if _, err := store.CallNext(ctx, queue.ID, tokenHash); err != nil {
			t.Fatalf("CallNext() error = %v", err)
		}
	}

	records, err := store.ListRecentCallRecords(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecentCallRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].Number != "A03" || records[1].Number != "A02" {
		t.Fatalf("recent records = %#v", records)
	}
	if records[0].QueueName != "General" || records[0].CounterName != "Counter 1" {
		t.Fatalf("enriched recent record = %#v", records[0])
	}
}

func TestCallNextRollsBackForUnavailableCounter(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 9)
	ctx := context.Background()

	invalidTokenHash := sha256.Sum256([]byte("invalid-counter-token"))
	_, err := store.CallNext(ctx, queue.ID, invalidTokenHash[:])
	if !errors.Is(err, domain.ErrCounterUnavailable) {
		t.Fatalf("CallNext() error = %v, want ErrCounterUnavailable", err)
	}
	call, err := store.CallNext(ctx, queue.ID, tokenHash)
	if err != nil {
		t.Fatalf("CallNext() after rollback error = %v", err)
	}
	if call.Position != 0 {
		t.Fatalf("position after rollback = %d, want 0", call.Position)
	}
}

func TestCallNextIsAtomicUnderConcurrency(t *testing.T) {
	const calls = 25
	store, queue, tokenHash := testStore(t, 0, calls-1)
	ctx := context.Background()

	values := make(chan int, calls)
	errs := make(chan error, calls)
	var wg sync.WaitGroup
	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			call, err := store.CallNext(ctx, queue.ID, tokenHash)
			if err != nil {
				errs <- err
				return
			}
			values <- call.Position
		}()
	}
	wg.Wait()
	close(errs)
	close(values)

	for err := range errs {
		t.Errorf("CallNext() error = %v", err)
	}
	got := make([]int, 0, calls)
	for value := range values {
		got = append(got, value)
	}
	sort.Ints(got)
	if len(got) != calls {
		t.Fatalf("successful calls = %d, want %d", len(got), calls)
	}
	for i, value := range got {
		if value != i {
			t.Fatalf("sorted sequence[%d] = %d, want %d; all values: %v", i, value, i, got)
		}
	}
}

func TestRevokeCounterRejectsTokenWithoutConsumingNumber(t *testing.T) {
	store, queue, tokenHash := testStore(t, 0, 9)
	ctx := context.Background()

	counter, err := store.CounterForToken(ctx, tokenHash)
	if err != nil || counter.Name != "Counter 1" {
		t.Fatalf("CounterForToken() = %#v, %v", counter, err)
	}
	if err := store.RevokeCounter(ctx, 1); err != nil {
		t.Fatalf("RevokeCounter() error = %v", err)
	}
	if _, err := store.CounterForToken(ctx, tokenHash); !errors.Is(err, domain.ErrCounterUnavailable) {
		t.Fatalf("CounterForToken() after revoke error = %v, want ErrCounterUnavailable", err)
	}
	if _, err := store.CallNext(ctx, queue.ID, tokenHash); !errors.Is(err, domain.ErrCounterUnavailable) {
		t.Fatalf("CallNext() with revoked token error = %v, want ErrCounterUnavailable", err)
	}

	newTokenHash := sha256.Sum256([]byte("replacement-counter-token"))
	if err := store.RotateCounterToken(ctx, 1, newTokenHash[:]); err != nil {
		t.Fatalf("RotateCounterToken() error = %v", err)
	}
	if _, err := store.CallNext(ctx, queue.ID, newTokenHash[:]); !errors.Is(err, domain.ErrCounterUnavailable) {
		t.Fatalf("CallNext() for revoked counter after rotation error = %v, want ErrCounterUnavailable", err)
	}
	var currentPosition int
	if err := store.db.QueryRowContext(ctx, "SELECT current_position FROM queues WHERE id = ?", queue.ID).Scan(&currentPosition); err != nil {
		t.Fatalf("read queue current position: %v", err)
	}
	if currentPosition != -1 {
		t.Fatalf("current position after rejected calls = %d, want -1", currentPosition)
	}
	if err := store.ReactivateCounter(ctx, 1, newTokenHash[:]); err != nil {
		t.Fatalf("ReactivateCounter() error = %v", err)
	}
	call, err := store.CallNext(ctx, queue.ID, newTokenHash[:])
	if err != nil {
		t.Fatalf("CallNext() after reactivation error = %v", err)
	}
	if call.Position != 0 {
		t.Fatalf("position after reactivation = %d, want 0", call.Position)
	}
}

func TestRotateCounterTokenInvalidatesPreviousToken(t *testing.T) {
	store, queue, oldTokenHash := testStore(t, 0, 9)
	ctx := context.Background()
	newTokenHash := sha256.Sum256([]byte("replacement-counter-token"))

	if err := store.RotateCounterToken(ctx, 1, newTokenHash[:]); err != nil {
		t.Fatalf("RotateCounterToken() error = %v", err)
	}
	if _, err := store.CallNext(ctx, queue.ID, oldTokenHash); !errors.Is(err, domain.ErrCounterUnavailable) {
		t.Fatalf("CallNext() with old token error = %v, want ErrCounterUnavailable", err)
	}
	call, err := store.CallNext(ctx, queue.ID, newTokenHash[:])
	if err != nil {
		t.Fatalf("CallNext() with new token error = %v", err)
	}
	if call.Position != 0 {
		t.Fatalf("position after rejected old token = %d, want 0", call.Position)
	}
}

func testStore(t *testing.T, minValue, maxValue int) (*Store, domain.Queue, []byte) {
	t.Helper()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "turnino.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	queue, err := store.CreateQueue(ctx, domain.CreateQueueParams{
		Name: "General", Prefix: "A", Digits: 2, MinValue: minValue, MaxValue: maxValue,
	})
	if err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	tokenHash := sha256.Sum256([]byte("counter-token"))
	_, err = store.CreateCounter(ctx, "Counter 1", tokenHash[:])
	if err != nil {
		t.Fatalf("CreateCounter() error = %v", err)
	}
	return store, queue, tokenHash[:]
}
