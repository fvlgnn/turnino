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
	if migrations != 2 {
		t.Fatalf("migration count = %d, want 2", migrations)
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
	if call.SequenceValue != 0 {
		t.Fatalf("sequence after rollback = %d, want 0", call.SequenceValue)
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
			values <- call.SequenceValue
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

	if err := store.RevokeCounter(ctx, 1); err != nil {
		t.Fatalf("RevokeCounter() error = %v", err)
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
	var currentValue int
	if err := store.db.QueryRowContext(ctx, "SELECT current_value FROM queues WHERE id = ?", queue.ID).Scan(&currentValue); err != nil {
		t.Fatalf("read queue current value: %v", err)
	}
	if currentValue != -1 {
		t.Fatalf("current value after rejected calls = %d, want -1", currentValue)
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
	if call.SequenceValue != 0 {
		t.Fatalf("sequence after rejected old token = %d, want 0", call.SequenceValue)
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
