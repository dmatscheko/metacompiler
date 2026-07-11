//go:build ignore

// Go subset big test 1: DATA STRUCTURES AND ALGORITHMS.
//
// Sorting (insertion, selection, bubble, quick, merge), binary search and lower
// bound, an array-backed stack, a growing queue, a binary min-heap (with heap
// sort), and a binary search tree built in an index arena (no pointers). Every
// result is checked against an expected value; main() ends with os.Exit(fails),
// so the run exits 0 exactly when every check passes, byte-identically under both
// engines.

package main

import (
	"fmt"
	"os"
)

var fails = 0

func check(name string, got int, want int) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func checkBool(name string, got bool, want bool) {
	g := 0
	w := 0
	if got {
		g = 1
	}
	if want {
		w = 1
	}
	check(name, g, w)
}

// ----- small numeric helpers -----

func absi(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// ----- slice helpers -----

func cloneSlice(s []int) []int {
	out := []int{}
	for _, v := range s {
		out = append(out, v)
	}
	return out
}

func subSlice(s []int, lo int, hi int) []int {
	out := []int{}
	for i := lo; i < hi; i++ {
		out = append(out, s[i])
	}
	return out
}

func equalSlice(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isSorted(s []int) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}

func sumSlice(s []int) int {
	t := 0
	for _, v := range s {
		t += v
	}
	return t
}

func swap(s []int, i int, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

// ----- sorting: all mutate/return an ascending order -----

func insertionSort(s []int) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

func selectionSort(s []int) {
	n := len(s)
	for i := 0; i < n; i++ {
		lo := i
		for j := i + 1; j < n; j++ {
			if s[j] < s[lo] {
				lo = j
			}
		}
		swap(s, i, lo)
	}
}

func bubbleSort(s []int) {
	n := len(s)
	for pass := 0; pass < n; pass++ {
		swapped := false
		for i := 1; i < n-pass; i++ {
			if s[i-1] > s[i] {
				swap(s, i-1, i)
				swapped = true
			}
		}
		if !swapped {
			break
		}
	}
}

func partition(s []int, lo int, hi int) int {
	pivot := s[hi]
	i := lo - 1
	for j := lo; j < hi; j++ {
		if s[j] <= pivot {
			i++
			swap(s, i, j)
		}
	}
	swap(s, i+1, hi)
	return i + 1
}

func quickSort(s []int, lo int, hi int) {
	if lo >= hi {
		return
	}
	p := partition(s, lo, hi)
	quickSort(s, lo, p-1)
	quickSort(s, p+1, hi)
}

func merge(a []int, b []int) []int {
	out := []int{}
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	for i < len(a) {
		out = append(out, a[i])
		i++
	}
	for j < len(b) {
		out = append(out, b[j])
		j++
	}
	return out
}

func mergeSort(s []int) []int {
	n := len(s)
	if n <= 1 {
		return cloneSlice(s)
	}
	mid := n / 2
	left := mergeSort(subSlice(s, 0, mid))
	right := mergeSort(subSlice(s, mid, n))
	return merge(left, right)
}

// ----- searching -----

func binarySearch(s []int, target int) int {
	lo := 0
	hi := len(s) - 1
	for lo <= hi {
		mid := (lo + hi) / 2
		if s[mid] == target {
			return mid
		}
		if s[mid] < target {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return -1
}

// first index i with s[i] >= target (len when none)
func lowerBound(s []int, target int) int {
	lo := 0
	hi := len(s)
	for lo < hi {
		mid := (lo + hi) / 2
		if s[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// ----- array-backed stack -----

type Stack struct {
	data []int
	sp   int
}

func (s *Stack) push(v int) {
	if s.sp < len(s.data) {
		s.data[s.sp] = v
	} else {
		s.data = append(s.data, v)
	}
	s.sp++
}

func (s *Stack) pop() int {
	s.sp--
	return s.data[s.sp]
}

func (s *Stack) peek() int {
	return s.data[s.sp-1]
}

func (s *Stack) size() int {
	return s.sp
}

func (s *Stack) empty() bool {
	return s.sp == 0
}

// ----- growing queue (head index into an append-only buffer) -----

type Queue struct {
	data []int
	head int
}

func (q *Queue) enqueue(v int) {
	q.data = append(q.data, v)
}

func (q *Queue) dequeue() int {
	v := q.data[q.head]
	q.head++
	return v
}

func (q *Queue) size() int {
	return len(q.data) - q.head
}

func (q *Queue) empty() bool {
	return q.size() == 0
}

// ----- binary min-heap -----

type Heap struct {
	data []int
}

func (h *Heap) size() int {
	return len(h.data)
}

func (h *Heap) push(v int) {
	h.data = append(h.data, v)
	i := len(h.data) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if h.data[parent] <= h.data[i] {
			break
		}
		swap(h.data, parent, i)
		i = parent
	}
}

func (h *Heap) pop() int {
	n := len(h.data)
	top := h.data[0]
	h.data[0] = h.data[n-1]
	// drop the last element by rebuilding without it
	nd := []int{}
	for i := 0; i < n-1; i++ {
		nd = append(nd, h.data[i])
	}
	h.data = nd
	n = len(h.data)
	i := 0
	for 2*i+1 < n {
		l := 2*i + 1
		r := 2*i + 2
		small := i
		if h.data[l] < h.data[small] {
			small = l
		}
		if r < n && h.data[r] < h.data[small] {
			small = r
		}
		if small == i {
			break
		}
		swap(h.data, i, small)
		i = small
	}
	return top
}

// ----- binary search tree in an index arena -----

type Node struct {
	val   int
	left  int
	right int
}

type Tree struct {
	nodes []Node
	root  int
}

func newTree() Tree {
	return Tree{[]Node{}, -1}
}

func (t *Tree) insert(v int) {
	t.nodes = append(t.nodes, Node{v, -1, -1})
	idx := len(t.nodes) - 1
	if t.root == -1 {
		t.root = idx
		return
	}
	cur := t.root
	for cur != -1 {
		if v < t.nodes[cur].val {
			if t.nodes[cur].left == -1 {
				t.nodes[cur].left = idx
				return
			}
			cur = t.nodes[cur].left
		} else {
			if t.nodes[cur].right == -1 {
				t.nodes[cur].right = idx
				return
			}
			cur = t.nodes[cur].right
		}
	}
}

func (t *Tree) contains(v int) bool {
	cur := t.root
	for cur != -1 {
		if v == t.nodes[cur].val {
			return true
		}
		if v < t.nodes[cur].val {
			cur = t.nodes[cur].left
		} else {
			cur = t.nodes[cur].right
		}
	}
	return false
}

func (t *Tree) inorder(node int, out []int) []int {
	if node == -1 {
		return out
	}
	out = t.inorder(t.nodes[node].left, out)
	out = append(out, t.nodes[node].val)
	out = t.inorder(t.nodes[node].right, out)
	return out
}

func (t *Tree) height(node int) int {
	if node == -1 {
		return 0
	}
	lh := t.height(t.nodes[node].left)
	rh := t.height(t.nodes[node].right)
	return 1 + maxInt(lh, rh)
}

func main() {
	// The reference multiset and its known sorted order.
	base := []int{5, 2, 9, 1, 5, 6, 3, 8, 7, 4, 0, 9}
	want := []int{0, 1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 9}

	// Every sort produces the same ascending order.
	a1 := cloneSlice(base)
	insertionSort(a1)
	checkBool("insertion sorts", equalSlice(a1, want), true)
	checkBool("insertion isSorted", isSorted(a1), true)

	a2 := cloneSlice(base)
	selectionSort(a2)
	checkBool("selection sorts", equalSlice(a2, want), true)

	a3 := cloneSlice(base)
	bubbleSort(a3)
	checkBool("bubble sorts", equalSlice(a3, want), true)

	a4 := cloneSlice(base)
	quickSort(a4, 0, len(a4)-1)
	checkBool("quick sorts", equalSlice(a4, want), true)

	a5 := mergeSort(base)
	checkBool("merge sorts", equalSlice(a5, want), true)
	checkBool("merge is pure", equalSlice(base, []int{5, 2, 9, 1, 5, 6, 3, 8, 7, 4, 0, 9}), true)

	check("sum preserved", sumSlice(a1), sumSlice(base))

	// already sorted / single / empty edge cases
	single := []int{42}
	insertionSort(single)
	check("single sort", single[0], 42)
	empty := []int{}
	quickSort(empty, 0, -1)
	check("empty sort", len(empty), 0)

	// binary search on the sorted slice
	check("bsearch found 0", binarySearch(want, 0), 0)
	check("bsearch found 7", binarySearch(want, 7), 8)
	check("bsearch missing", binarySearch(want, 100), -1)
	check("bsearch missing neg", binarySearch(want, -1), -1)
	allPresent := true
	for _, v := range []int{0, 1, 2, 3, 4, 6, 7, 8} {
		if binarySearch(want, v) < 0 {
			allPresent = false
		}
	}
	checkBool("bsearch all present", allPresent, true)

	// lower bound
	check("lb of 5", lowerBound(want, 5), 5)
	check("lb of 4", lowerBound(want, 4), 4)
	check("lb below", lowerBound(want, -3), 0)
	check("lb above", lowerBound(want, 50), len(want))

	// stack: LIFO
	st := Stack{[]int{}, 0}
	for _, v := range []int{10, 20, 30} {
		st.push(v)
	}
	check("stack size", st.size(), 3)
	check("stack peek", st.peek(), 30)
	check("stack pop a", st.pop(), 30)
	check("stack pop b", st.pop(), 20)
	st.push(99)
	check("stack pop c", st.pop(), 99)
	check("stack pop d", st.pop(), 10)
	checkBool("stack empty", st.empty(), true)

	// reverse a sequence with the stack
	rin := []int{1, 2, 3, 4, 5}
	rs := Stack{[]int{}, 0}
	for _, v := range rin {
		rs.push(v)
	}
	rev := []int{}
	for !rs.empty() {
		rev = append(rev, rs.pop())
	}
	checkBool("stack reverse", equalSlice(rev, []int{5, 4, 3, 2, 1}), true)

	// queue: FIFO
	q := Queue{[]int{}, 0}
	for _, v := range []int{7, 8, 9} {
		q.enqueue(v)
	}
	check("queue size", q.size(), 3)
	check("queue deq a", q.dequeue(), 7)
	check("queue deq b", q.dequeue(), 8)
	q.enqueue(11)
	check("queue deq c", q.dequeue(), 9)
	check("queue deq d", q.dequeue(), 11)
	checkBool("queue empty", q.empty(), true)

	// heap: repeated pop yields ascending order (heap sort)
	h := Heap{[]int{}}
	for _, v := range base {
		h.push(v)
	}
	check("heap size", h.size(), len(base))
	hout := []int{}
	for h.size() > 0 {
		hout = append(hout, h.pop())
	}
	checkBool("heap sort", equalSlice(hout, want), true)

	// bst: inorder traversal is sorted (with duplicates kept)
	t := newTree()
	for _, v := range base {
		t.insert(v)
	}
	tout := t.inorder(t.root, []int{})
	checkBool("bst inorder sorted", equalSlice(tout, want), true)
	check("bst node count", len(t.nodes), len(base))
	checkBool("bst contains 6", t.contains(6), true)
	checkBool("bst contains 5", t.contains(5), true)
	checkBool("bst missing 42", t.contains(42), false)
	checkBool("bst missing -7", t.contains(-7), false)
	// height is at least ceil(log2(n)) and at most n
	hgt := t.height(t.root)
	checkBool("bst height range", hgt >= 4 && hgt <= len(base), true)

	// a degenerate (already sorted) insert order makes a linked list of height n
	lin := newTree()
	for i := 1; i <= 6; i++ {
		lin.insert(i)
	}
	check("bst degenerate height", lin.height(lin.root), 6)

	if fails == 0 {
		fmt.Println("Go big test 1 (data structures and algorithms) passed")
	}
	os.Exit(fails)
}
