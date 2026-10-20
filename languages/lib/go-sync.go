// Package sync: Go's sync package WRITTEN IN GO, compiled with a program that
// imports "sync" (core.importFile in both go grammars), so one implementation
// serves the interpreter, llvm.Run and a native build alike.
//
// Goroutines here are cooperative: `go f()` queues f, and a receive that finds its
// channel empty runs the queue, each goroutine to completion. Nothing can preempt a
// goroutine between Lock and Unlock, so a Mutex only has to keep Go's rules about
// misuse, and WaitGroup.Wait runs the queue the way a blocked receive does.
package sync

import "os"

type WaitGroup struct {
	n int
}

func (wg *WaitGroup) Add(delta int) {
	wg.n += delta
	if wg.n < 0 {
		panic("sync: negative WaitGroup counter")
	}
}

func (wg *WaitGroup) Done() { wg.Add(-1) }

func (wg *WaitGroup) Wait() {
	if wg.n > 0 {
		runQueued()
	}
	if wg.n > 0 {
		fatal("all goroutines are asleep - deadlock!")
	}
}

func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// A select on a channel nothing will ever fill runs every queued goroutine before
// it takes its default arm.
func runQueued() {
	never := make(chan bool)
	select {
	case <-never:
	default:
	}
}

// Go reports a misused lock and a deadlock as a fatal error, which no recover
// stops: on stderr, exit status 2.
func fatal(msg string) {
	println("fatal error: " + msg)
	os.Exit(2)
}

type Mutex struct {
	locked bool
}

func (m *Mutex) Lock() {
	if m.locked {
		fatal("all goroutines are asleep - deadlock!")
	}
	m.locked = true
}

func (m *Mutex) TryLock() bool {
	if m.locked {
		return false
	}
	m.locked = true
	return true
}

func (m *Mutex) Unlock() {
	if !m.locked {
		fatal("sync: unlock of unlocked mutex")
	}
	m.locked = false
}

type RWMutex struct {
	w       bool
	readers int
}

func (rw *RWMutex) Lock() {
	if rw.w || rw.readers > 0 {
		fatal("all goroutines are asleep - deadlock!")
	}
	rw.w = true
}

func (rw *RWMutex) Unlock() {
	if !rw.w {
		fatal("sync: Unlock of unlocked RWMutex")
	}
	rw.w = false
}

func (rw *RWMutex) RLock() {
	if rw.w {
		fatal("all goroutines are asleep - deadlock!")
	}
	rw.readers++
}

func (rw *RWMutex) RUnlock() {
	if rw.readers <= 0 {
		fatal("sync: RUnlock of unlocked RWMutex")
	}
	rw.readers--
}

type Once struct {
	done bool
}

func (o *Once) Do(f func()) {
	if o.done {
		return
	}
	o.done = true
	f()
}
