// Package context: Go's context package WRITTEN IN GO, compiled with a program
// that imports "context" (core.importFile in both go grammars): cancellation
// travelling down a tree of contexts, and values. This runtime has no clock, so a
// deadline is recorded and reported but never passes on its own.
package context

import (
	"errors"
	"time"
)

type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}

var Canceled = errors.New("context canceled")
var DeadlineExceeded = errors.New("context deadline exceeded")

type CancelFunc func()

type emptyCtx struct{ name string }

func (emptyCtx) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }
func (emptyCtx) Done() <-chan struct{}                   { return nil }
func (emptyCtx) Err() error                              { return nil }
func (emptyCtx) Value(key any) any                       { return nil }
func (e emptyCtx) String() string                        { return e.name }

var background = emptyCtx{"context.Background"}
var todo = emptyCtx{"context.TODO"}

func Background() Context { return background }
func TODO() Context       { return todo }

type cancelCtx struct {
	parent   Context
	done     chan struct{}
	err      error
	cause    error
	children []*cancelCtx
	deadline time.Time
	hasDl    bool
}

func (c *cancelCtx) Deadline() (deadline time.Time, ok bool) {
	if c.hasDl {
		return c.deadline, true
	}
	return c.parent.Deadline()
}
func (c *cancelCtx) Done() <-chan struct{} { return c.done }
func (c *cancelCtx) Err() error            { return c.err }
func (c *cancelCtx) Value(key any) any     { return c.parent.Value(key) }
func (c *cancelCtx) String() string        { return "context.WithCancel" }

func (c *cancelCtx) cancel(err, cause error) {
	if c.err != nil {
		return
	}
	c.err = err
	if cause == nil {
		cause = err
	}
	c.cause = cause
	close(c.done)
	for _, ch := range c.children {
		ch.cancel(err, cause)
	}
	c.children = nil
}

// The nearest cancelable ancestor of parent, which cancels its children with it.
func parentCancel(parent Context) *cancelCtx {
	for {
		switch p := parent.(type) {
		case *cancelCtx:
			return p
		case *valueCtx:
			parent = p.parent
		default:
			return nil
		}
	}
}

func newCancelCtx(parent Context) *cancelCtx {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := &cancelCtx{parent: parent, done: make(chan struct{})}
	if p := parentCancel(parent); p != nil {
		if p.err != nil {
			c.cancel(p.err, p.cause)
		} else {
			p.children = append(p.children, c)
		}
	}
	return c
}

func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	return c, func() { c.cancel(Canceled, nil) }
}

type CancelCauseFunc func(cause error)

func WithCancelCause(parent Context) (ctx Context, cancel CancelCauseFunc) {
	c := newCancelCtx(parent)
	return c, func(cause error) { c.cancel(Canceled, cause) }
}

// Cause: why c was canceled - the cause given, or its error.
func Cause(c Context) error {
	if cc := parentCancel(c); cc != nil {
		return cc.cause
	}
	return c.Err()
}

func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	c := newCancelCtx(parent)
	c.deadline = d
	c.hasDl = true
	return c, func() { c.cancel(Canceled, nil) }
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

type valueCtx struct {
	parent   Context
	key, val any
}

func (c *valueCtx) Deadline() (deadline time.Time, ok bool) { return c.parent.Deadline() }
func (c *valueCtx) Done() <-chan struct{}                   { return c.parent.Done() }
func (c *valueCtx) Err() error                              { return c.parent.Err() }
func (c *valueCtx) Value(key any) any {
	if c.key == key {
		return c.val
	}
	return c.parent.Value(key)
}

func WithValue(parent Context, key, val any) Context {
	if key == nil {
		panic("nil key")
	}
	return &valueCtx{parent, key, val}
}
