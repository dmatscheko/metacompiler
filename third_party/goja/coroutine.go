package goja

// A Coroutine runs a function on a call stack of its own, so the function can
// suspend (Runtime.Yield) at any depth of calls, host calls included, and be
// continued there by the next Resume.
//
// The body runs on its own goroutine over its own vm. Resume and Yield are a
// strictly alternating handshake over two unbuffered channels, so exactly one
// goroutine runs at any moment and the Runtime needs no locking: the only
// switch is r.vm, which every entry into JavaScript from Go reads, so it names
// the vm of whichever side runs. A coroutine that is never finished leaves its
// goroutine parked for the rest of the process.
type Coroutine struct {
	r       *Runtime
	vm      *vm
	fn      Callable
	outer   *vm // the resumer's vm, while the body runs
	started bool
	done    bool
	running bool
	resume  chan Value
	yields  chan coStep
}

// coStep is one handshake result: a yielded value, the body's return value
// (done), or what the body panicked with, which Resume re-raises.
type coStep struct {
	v        Value
	done     bool
	panicVal interface{}
}

// NewCoroutine makes a coroutine over fn, which is called with no arguments by
// the first Resume.
func (r *Runtime) NewCoroutine(fn Callable) *Coroutine {
	cvm := &vm{r: r}
	cvm.init()
	cvm.maxCallStackSize = r.vm.maxCallStackSize
	c := &Coroutine{r: r, vm: cvm, fn: fn}
	cvm.co = c
	return c
}

// Resume runs the body until its next Yield, answering the yielded value and
// false, or until it returns, answering its value and true. v is what that
// Yield answers inside the body; the first Resume's v is not delivered. A throw
// out of the body is raised again here. A finished coroutine answers
// (undefined, true); resuming a running one is a TypeError.
func (c *Coroutine) Resume(v Value) (Value, bool) {
	if c.done {
		return _undefined, true
	}
	if c.running {
		panic(c.r.NewTypeError("coroutine is already running"))
	}
	if !c.started {
		c.started = true
		c.resume = make(chan Value)
		c.yields = make(chan coStep)
		go func() {
			defer func() {
				if p := recover(); p != nil {
					c.yields <- coStep{done: true, panicVal: p}
				}
			}()
			<-c.resume
			ret, err := c.fn(_undefined)
			if err != nil {
				panic(err)
			}
			c.yields <- coStep{v: ret, done: true}
		}()
	}
	c.running = true
	c.outer = c.r.vm
	c.r.vm = c.vm
	c.resume <- v
	st := <-c.yields
	c.r.vm = c.outer
	c.running = false
	if st.done {
		c.done = true
	}
	if st.panicVal != nil {
		panic(st.panicVal)
	}
	return st.v, st.done
}

// Yield suspends the running coroutine, handing v to the Resume that ran it,
// and answers the value of the Resume that continues it. Outside a coroutine
// it is a TypeError.
func (r *Runtime) Yield(v Value) Value {
	c := r.vm.co
	if c == nil {
		panic(r.NewTypeError("yield outside a coroutine"))
	}
	c.yields <- coStep{v: v}
	return <-c.resume
}
