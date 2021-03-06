package machine

import (
	"context"
	"fmt"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"
)

// Routine is an interface representing a goroutine
type Routine interface {
	// Context returns the goroutines unique context that may be used for cancellation
	Context() context.Context
	// Cancel cancels the context returned from Context()
	Cancel()
	// PID() is the goroutines unique process id
	PID() string
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{}) error
	// PublishN publishes the object to the channel by name to the first N subscribers of the channel
	PublishN(channel string, obj interface{}, n int) error
	// Subscribe subscribes to a channel and executes the function on every message passed to it. It exits if the goroutines context is cancelled.
	Subscribe(channel string, handler func(obj interface{})) error
	// SubscribeN subscribes to the given channel until it receives N messages or its context is cancelled
	SubscribeN(channel string, n int, handler func(msg interface{})) error
	// SubscribeUntil subscribes to the given channel until the decider returns false for the first time. The subscription breaks when the routine's context is cancelled or the decider returns false.
	SubscribeUntil(channel string, decider func() bool, handler func(msg interface{})) error
	// SubscribeWhile subscribes to the given channel while the decider returns true. The subscription breaks when the routine's context is cancelled.
	SubscribeWhile(channel string, decider func() bool, handler func(msg interface{})) error
	// SubscribeFilter subscribes to the given channel with the given filter. The subscription breaks when the routine's context is cancelled.
	SubscribeFilter(channel string, filter func(msg interface{}) bool, handler func(msg interface{})) error
	// TraceLog logs a message within the goroutine execution tracer. ref: https://golang.org/pkg/runtime/trace/#example_
	TraceLog(message string)
	// Machine returns the underlying routine's machine instance
	Machine() *Machine
}

func (g *goRoutine) implements() Routine {
	return g
}

type goRoutine struct {
	machine  *Machine
	ctx      context.Context
	id       string
	tags     []string
	start    time.Time
	doneOnce sync.Once
	cancel   func()
}

func (r *goRoutine) Context() context.Context {
	return r.ctx
}

func (r *goRoutine) PID() string {
	return r.id
}

func (r *goRoutine) Tags() []string {
	sort.Strings(r.tags)
	return r.tags
}

func (r *goRoutine) Cancel() {
	r.cancel()
}

func (r *goRoutine) Start() time.Time {
	return r.start
}

func (r *goRoutine) Duration() time.Duration {
	return time.Since(r.start)
}

func (g *goRoutine) Publish(channel string, obj interface{}) error {
	return g.machine.pubsub.Publish(channel, obj)
}

func (g *goRoutine) PublishN(channel string, obj interface{}, n int) error {
	return g.machine.pubsub.PublishN(channel, obj, n)
}

func (g *goRoutine) Subscribe(channel string, handler func(obj interface{})) error {
	return g.machine.pubsub.Subscribe(g.ctx, channel, handler)
}

func (g *goRoutine) SubscribeN(channel string, n int, handler func(obj interface{})) error {
	return g.machine.pubsub.SubscribeN(g.ctx, channel, n, handler)
}

func (g *goRoutine) SubscribeWhile(channel string, decider func() bool, handler func(obj interface{})) error {
	return g.machine.pubsub.SubscribeWhile(g.ctx, channel, decider, handler)
}

func (g *goRoutine) SubscribeUntil(channel string, decider func() bool, handler func(obj interface{})) error {
	return g.machine.pubsub.SubscribeUntil(g.ctx, channel, decider, handler)
}

func (g *goRoutine) SubscribeFilter(channel string, filter func(obj interface{}) bool, handler func(obj interface{})) error {
	return g.machine.pubsub.SubscribeFilter(g.ctx, channel, filter, handler)
}

func (g *goRoutine) Machine() *Machine {
	return g.machine
}

func (g *goRoutine) done() {
	g.doneOnce.Do(func() {
		g.cancel()
		g.machine.mu.Lock()
		delete(g.machine.routines, g.id)
		g.machine.mu.Unlock()
	})
	routinePool.deallocateRoutine(g)
}

func (g *goRoutine) TraceLog(message string) {
	trace.Logf(g.ctx, strings.Join(g.tags, " "), fmt.Sprintf("%s %s", g.PID(), message))
}
