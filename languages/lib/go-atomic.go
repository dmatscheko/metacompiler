// Package atomic: Go's sync/atomic package WRITTEN IN GO, compiled with a program
// that imports "sync/atomic" (core.importFile in both go grammars). Goroutines here
// are cooperative and never interleave inside a function (see lib/go-sync.go), so
// each operation is the plain read-modify-write.
package atomic

func AddInt32(addr *int32, delta int32) int32 {
	*addr += delta
	return *addr
}

func AddInt64(addr *int64, delta int64) int64 {
	*addr += delta
	return *addr
}

func AddUint32(addr *uint32, delta uint32) uint32 {
	*addr += delta
	return *addr
}

func AddUint64(addr *uint64, delta uint64) uint64 {
	*addr += delta
	return *addr
}

func LoadInt32(addr *int32) int32    { return *addr }
func LoadInt64(addr *int64) int64    { return *addr }
func LoadUint32(addr *uint32) uint32 { return *addr }
func LoadUint64(addr *uint64) uint64 { return *addr }

func StoreInt32(addr *int32, val int32)    { *addr = val }
func StoreInt64(addr *int64, val int64)    { *addr = val }
func StoreUint32(addr *uint32, val uint32) { *addr = val }
func StoreUint64(addr *uint64, val uint64) { *addr = val }

func SwapInt32(addr *int32, new int32) (old int32) {
	old = *addr
	*addr = new
	return
}

func SwapInt64(addr *int64, new int64) (old int64) {
	old = *addr
	*addr = new
	return
}

func CompareAndSwapInt32(addr *int32, old, new int32) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapInt64(addr *int64, old, new int64) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

type Int32 struct{ v int32 }

func (x *Int32) Load() int32           { return x.v }
func (x *Int32) Store(val int32)       { x.v = val }
func (x *Int32) Add(delta int32) int32 { x.v += delta; return x.v }
func (x *Int32) Swap(new int32) int32 {
	old := x.v
	x.v = new
	return old
}
func (x *Int32) CompareAndSwap(old, new int32) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

type Int64 struct{ v int64 }

func (x *Int64) Load() int64           { return x.v }
func (x *Int64) Store(val int64)       { x.v = val }
func (x *Int64) Add(delta int64) int64 { x.v += delta; return x.v }
func (x *Int64) Swap(new int64) int64 {
	old := x.v
	x.v = new
	return old
}
func (x *Int64) CompareAndSwap(old, new int64) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

type Bool struct{ v bool }

func (x *Bool) Load() bool     { return x.v }
func (x *Bool) Store(val bool) { x.v = val }
func (x *Bool) Swap(new bool) bool {
	old := x.v
	x.v = new
	return old
}
