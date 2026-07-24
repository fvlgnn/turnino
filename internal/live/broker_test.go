package live

import (
	"context"
	"testing"
	"time"

	"github.com/fvlgnn/turnino/internal/domain"
)

func TestBrokerPublishesToEverySubscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	first := NewBroker()
	one := first.Subscribe(ctx)
	two := first.Subscribe(ctx)

	want := domain.CallRecord{Call: domain.Call{ID: 42, Number: "A042"}}
	first.Publish(want)
	for i, subscriber := range []<-chan domain.CallRecord{one, two} {
		select {
		case got := <-subscriber:
			if got.ID != want.ID || got.Number != want.Number {
				t.Fatalf("subscriber %d received %#v, want %#v", i, got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive the call", i)
		}
	}
	cancel()
}

func TestBrokerDoesNotBlockOnSlowSubscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	broker := NewBroker()
	events := broker.Subscribe(ctx)

	for id := int64(1); id <= 100; id++ {
		broker.Publish(domain.CallRecord{Call: domain.Call{ID: id}})
	}

	var latest int64
	for {
		select {
		case event := <-events:
			latest = event.ID
		default:
			if latest != 100 {
				t.Fatalf("latest buffered event = %d, want 100", latest)
			}
			return
		}
	}
}
