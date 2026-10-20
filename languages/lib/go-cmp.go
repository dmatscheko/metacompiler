// Package cmp: Go's cmp package WRITTEN IN GO, compiled with a program that
// imports "cmp" (core.importFile in both go grammars), so one implementation
// serves the interpreter, llvm.Run and a native build alike.
package cmp

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// A NaN orders before every other value and equals another NaN.
func Less[T Ordered](x, y T) bool {
	return (isNaN(x) && !isNaN(y)) || x < y
}

func Compare[T Ordered](x, y T) int {
	xNaN := isNaN(x)
	yNaN := isNaN(y)
	if xNaN {
		if yNaN {
			return 0
		}
		return -1
	}
	if yNaN {
		return +1
	}
	if x < y {
		return -1
	}
	if x > y {
		return +1
	}
	return 0
}

func isNaN[T Ordered](x T) bool {
	return x != x
}

// The first argument that is not its type's zero value. A type parameter's zero
// value is not known at run time in this value model, so each argument is tested
// against the zero of its own dynamic type; when every one is zero, the last is
// that zero.
func Or[T comparable](vals ...T) T {
	for _, val := range vals {
		if !isZero(val) {
			return val
		}
	}
	if len(vals) == 0 {
		var zero T
		return zero
	}
	return vals[len(vals)-1]
}

func isZero(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case bool:
		return !x
	case string:
		return x == ""
	case int:
		return x == 0
	case int8:
		return x == 0
	case int16:
		return x == 0
	case int32:
		return x == 0
	case int64:
		return x == 0
	case uint:
		return x == 0
	case uint8:
		return x == 0
	case uint16:
		return x == 0
	case uint32:
		return x == 0
	case uint64:
		return x == 0
	case float32:
		return x == 0
	case float64:
		return x == 0
	}
	return false
}
