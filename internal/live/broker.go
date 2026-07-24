package live

import (
	"context"
	"sync"

	"github.com/fvlgnn/turnino/internal/domain"
)

// Broker fans call events out to every connected monitor without making the
// caller wait for browsers. SQLite remains the source of truth; subscribers
// receive a fresh snapshot whenever they connect or reconnect.
type Broker struct {
	mu          sync.Mutex
	subscribers map[chan domain.CallRecord]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[chan domain.CallRecord]struct{})}
}

func (b *Broker) Subscribe(ctx context.Context) <-chan domain.CallRecord {
	events := make(chan domain.CallRecord, 16)
	b.mu.Lock()
	b.subscribers[events] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers, events)
		b.mu.Unlock()
	}()
	return events
}

func (b *Broker) Publish(call domain.CallRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for subscriber := range b.subscribers {
		select {
		case subscriber <- call:
		default:
			// Keep the most recent events for a temporarily slow browser.
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- call:
			default:
			}
		}
	}
}
