/* Go undefined name. A name no declaration binds is a compile error even where
 * it is never reached:
 *   ./go-fail-test-undefined.go:48:15: undefined: zz
 * so this file SHOULD fail in both halves before it prints anything. Everything
 * above that line is a name gc resolves: package names declared further down,
 * an import alias, type parameters, struct literal keys, a named result, a
 * method value and locals of every binder. */

package main

import (
	"fmt"
	str "strings"
)

type pair[K comparable, V any] struct {
	k K
	v V
}

func mk[K comparable, V any](k K, v V) pair[K, V] { return pair[K, V]{k: k, v: v} }

func (p pair[K, V]) key() K { return p.k }

func sum(xs ...int) (total int) {
	for _, x := range xs {
		total += x
	}
	return
}

func main() {
	fmt.Println("compiled")
	p := mk("a", later)
	f := p.key
	switch v := any(p.v).(type) {
	case int:
		fmt.Println(f(), v, sum(1, 2), str.ToUpper("x"), Mode)
	}
	ch := make(chan int, 1)
	ch <- 1
	select {
	case got := <-ch:
		_ = got
	}
	g := func(n int) int { return n * later }
	if false {
		fmt.Println(zz, g(1))
	}
}

const (
	Mode = iota
)

var later = 2
