package events

import (
	"context"
	"sync"
)

// EventHandler handles a published event.
type EventHandler func(context.Context, Event) error

// Dispatcher interface allows event publication/subscription.
type Dispatcher interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType EventType, handler EventHandler)
}

// inMemoryDispatcher is a simple synchronous dispatcher.
type inMemoryDispatcher struct {
	mu        sync.RWMutex
	listeners map[EventType][]EventHandler
}

// NewInMemoryDispatcher creates a dispatcher instance.
func NewInMemoryDispatcher() Dispatcher {
	return &inMemoryDispatcher{
		listeners: make(map[EventType][]EventHandler),
	}
}

// Publish synchronously invokes handlers for the given event.
func (d *inMemoryDispatcher) Publish(ctx context.Context, event Event) error {
	d.mu.RLock()
	handlers := append([]EventHandler{}, d.listeners[event.Type]...)
	d.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			// continue processing other handlers despite errors
		}
	}
	return nil
}

// Subscribe registers a handler for the given event type.
func (d *inMemoryDispatcher) Subscribe(eventType EventType, handler EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[eventType] = append(d.listeners[eventType], handler)
}
