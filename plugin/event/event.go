package event

import (
	"log/slog"
	"sort"
	"sync"
)

// Priority controls handler execution order. Higher priority handlers
// run first. If a handler cancels the context, lower-priority handlers
// are skipped.
type Priority int

const (
	PriorityLow      Priority = 0
	PriorityNormal   Priority = 1
	PriorityHigh     Priority = 2
	PriorityCritical Priority = 3
)

// Context carries cancellation state through the handler chain.
// A handler calls Cancel() to stop further processing and veto
// the action that fired the event.
type Context struct {
	cancel bool
}

// Cancel stops the event from propagating to lower-priority handlers
// and signals the caller to abort the action.
func (c *Context) Cancel() { c.cancel = true }

// Cancelled reports whether a handler called Cancel.
func (c *Context) Cancelled() bool { return c.cancel }

// Event is a multi-subscriber, priority-ordered event bus.
// Handlers are called synchronously in priority order (highest first).
// If a handler cancels the context, remaining handlers are skipped.
//
//	T = event data type (struct with pointer fields for mutability)
type Event[T any] struct {
	mu       sync.RWMutex
	handlers []handlerEntry[T]
}

type handlerEntry[T any] struct {
	fn       func(*Context, T)
	priority Priority
}

// NewEvent creates a new empty event.
func NewEvent[T any]() *Event[T] {
	return &Event[T]{}
}

// Register adds a handler at the given priority. Higher-priority
// handlers are called first. Registration is safe for concurrent use.
func (e *Event[T]) Register(fn func(*Context, T), priority Priority) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handlerEntry[T]{fn, priority})
	sort.SliceStable(e.handlers, func(i, j int) bool {
		return e.handlers[i].priority > e.handlers[j].priority
	})
}

// Unregister removes a handler. The fn must match the function
// registered previously by pointer identity.
func (e *Event[T]) Unregister(fn func(*Context, T)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, h := range e.handlers {
		if &h.fn == &fn {
			e.handlers = append(e.handlers[:i], e.handlers[i+1:]...)
			return
		}
	}
}

// Fire dispatches the event to all handlers in priority order.
// Returns the context so the caller can check Cancelled.
// If a handler cancels, remaining handlers are skipped.
func (e *Event[T]) Fire(data T) *Context {
	ctx := &Context{}
	e.mu.RLock()
	handlers := append([]handlerEntry[T](nil), e.handlers...)
	e.mu.RUnlock()
	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Default().Error("plugin event handler panicked", "panic", r)
					ctx.Cancel()
				}
			}()
			h.fn(ctx, data)
		}()
		if ctx.Cancelled() {
			break
		}
	}
	return ctx
}

// HasHandlers reports whether any handlers are registered.
func (e *Event[T]) HasHandlers() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.handlers) > 0
}

// Clear removes all handlers.
func (e *Event[T]) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = nil
}
