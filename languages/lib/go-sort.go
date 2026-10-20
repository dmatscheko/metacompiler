// Package sort: Go's sort package WRITTEN IN GO, compiled with a program that
// imports "sort" (core.importFile in both go grammars), so one implementation
// serves the interpreter, llvm.Run and a native build alike.
//
// The order of equal elements after Sort and Slice is part of what a program
// prints, so the algorithms below the API are Go's own, unchanged: pattern-
// defeating quicksort and the SymMerge stable sort of sort/zsortfunc.go
// (Copyright 2022 The Go Authors, BSD-style license). sort.Ints, Float64s and
// Strings run slices.Sort in Go, whose generated pdqsort is the same algorithm
// over `cmp.Less`, so they take the same path with that comparison.
package sort

type Interface interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
}

type lessSwap struct {
	Less func(i, j int) bool
	Swap func(i, j int)
}

type sortedHint int

const (
	unknownHint sortedHint = iota
	increasingHint
	decreasingHint
)

type xorshift uint64

func (r *xorshift) Next() uint64 {
	*r ^= *r << 13
	*r ^= *r >> 7
	*r ^= *r << 17
	return uint64(*r)
}

// bits.Len.
func bitsLen(x uint) int {
	n := 0
	for x != 0 {
		x >>= 1
		n++
	}
	return n
}

func nextPowerOfTwo(length int) uint {
	shift := uint(bitsLen(uint(length)))
	return uint(1 << shift)
}

func Sort(data Interface) {
	n := data.Len()
	if n <= 1 {
		return
	}
	pdqsort_func(lessSwap{data.Less, data.Swap}, 0, n, bitsLen(uint(n)))
}

func Stable(data Interface) { stable_func(lessSwap{data.Less, data.Swap}, data.Len()) }

func IsSorted(data Interface) bool {
	n := data.Len()
	for i := n - 1; i > 0; i-- {
		if data.Less(i, i-1) {
			return false
		}
	}
	return true
}

type reverse struct {
	Interface
}

func (r reverse) Less(i, j int) bool { return r.Interface.Less(j, i) }

func Reverse(data Interface) Interface { return &reverse{data} }

// A slice of any element type reaches these []any parameters unconverted: this
// value model allows it where Go's type checker would not, and this file is only
// ever compiled by this project's grammars. It stands in for reflectlite.Swapper.
func anyLen(s []any) int { return len(s) }
func swapper(s []any) func(i, j int) {
	return func(i, j int) { s[i], s[j] = s[j], s[i] }
}

func Slice(x any, less func(i, j int) bool) {
	n := anyLen(x)
	pdqsort_func(lessSwap{less, swapper(x)}, 0, n, bitsLen(uint(n)))
}

func SliceStable(x any, less func(i, j int) bool) {
	stable_func(lessSwap{less, swapper(x)}, anyLen(x))
}

func SliceIsSorted(x any, less func(i, j int) bool) bool {
	for i := anyLen(x) - 1; i > 0; i-- {
		if less(i, i-1) {
			return false
		}
	}
	return true
}

// cmp.Less: a NaN orders before every other float.
func isNaN(f float64) bool        { return f != f }
func floatLess(x, y float64) bool { return x < y || (isNaN(x) && !isNaN(y)) }

func Ints(x []int) {
	Slice(x, func(i, j int) bool { return x[i] < x[j] })
}
func Float64s(x []float64) {
	Slice(x, func(i, j int) bool { return floatLess(x[i], x[j]) })
}
func Strings(x []string) {
	Slice(x, func(i, j int) bool { return x[i] < x[j] })
}
func IntsAreSorted(x []int) bool {
	return SliceIsSorted(x, func(i, j int) bool { return x[i] < x[j] })
}
func Float64sAreSorted(x []float64) bool {
	return SliceIsSorted(x, func(i, j int) bool { return floatLess(x[i], x[j]) })
}
func StringsAreSorted(x []string) bool {
	return SliceIsSorted(x, func(i, j int) bool { return x[i] < x[j] })
}

type IntSlice []int

func (x IntSlice) Len() int           { return len(x) }
func (x IntSlice) Less(i, j int) bool { return x[i] < x[j] }
func (x IntSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x IntSlice) Sort()              { Sort(x) }
func (p IntSlice) Search(x int) int   { return SearchInts(p, x) }

type Float64Slice []float64

func (x Float64Slice) Len() int             { return len(x) }
func (x Float64Slice) Less(i, j int) bool   { return floatLess(x[i], x[j]) }
func (x Float64Slice) Swap(i, j int)        { x[i], x[j] = x[j], x[i] }
func (x Float64Slice) Sort()                { Sort(x) }
func (p Float64Slice) Search(x float64) int { return SearchFloat64s(p, x) }

type StringSlice []string

func (x StringSlice) Len() int            { return len(x) }
func (x StringSlice) Less(i, j int) bool  { return x[i] < x[j] }
func (x StringSlice) Swap(i, j int)       { x[i], x[j] = x[j], x[i] }
func (x StringSlice) Sort()               { Sort(x) }
func (p StringSlice) Search(x string) int { return SearchStrings(p, x) }

// ----- binary search -----

func Search(n int, f func(int) bool) int {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if !f(h) {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

func Find(n int, cmp func(int) int) (i int, found bool) {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if cmp(h) > 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	return i, i < n && cmp(i) == 0
}

func SearchInts(a []int, x int) int {
	return Search(len(a), func(i int) bool { return a[i] >= x })
}
func SearchFloat64s(a []float64, x float64) int {
	return Search(len(a), func(i int) bool { return a[i] >= x })
}
func SearchStrings(a []string, x string) int {
	return Search(len(a), func(i int) bool { return a[i] >= x })
}

// ----- sort/zsortfunc.go -----

// insertionSort_func sorts data[a:b] using insertion sort.
func insertionSort_func(data lessSwap, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

// siftDown_func implements the heap property on data[lo:hi].
// first is an offset into the array where the root of the heap lies.
func siftDown_func(data lessSwap, lo, hi, first int) {
	root := lo
	for {
		child := 2*root + 1
		if child >= hi {
			break
		}
		if child+1 < hi && data.Less(first+child, first+child+1) {
			child++
		}
		if !data.Less(first+root, first+child) {
			return
		}
		data.Swap(first+root, first+child)
		root = child
	}
}

func heapSort_func(data lessSwap, a, b int) {
	first := a
	lo := 0
	hi := b - a

	// Build heap with greatest element at top.
	for i := (hi - 1) / 2; i >= 0; i-- {
		siftDown_func(data, i, hi, first)
	}

	// Pop elements, largest first, into end of data.
	for i := hi - 1; i >= 0; i-- {
		data.Swap(first, first+i)
		siftDown_func(data, lo, i, first)
	}
}

// pdqsort_func sorts data[a:b].
// The algorithm based on pattern-defeating quicksort(pdqsort), but without the optimizations from BlockQuicksort.
// pdqsort paper: https://arxiv.org/pdf/2106.05123.pdf
// C++ implementation: https://github.com/orlp/pdqsort
// Rust implementation: https://docs.rs/pdqsort/latest/pdqsort/
// limit is the number of allowed bad (very unbalanced) pivots before falling back to heapsort.
func pdqsort_func(data lessSwap, a, b, limit int) {
	const maxInsertion = 12

	var (
		wasBalanced    = true // whether the last partitioning was reasonably balanced
		wasPartitioned = true // whether the slice was already partitioned
	)

	for {
		length := b - a

		if length <= maxInsertion {
			insertionSort_func(data, a, b)
			return
		}

		// Fall back to heapsort if too many bad choices were made.
		if limit == 0 {
			heapSort_func(data, a, b)
			return
		}

		// If the last partitioning was imbalanced, we need to breaking patterns.
		if !wasBalanced {
			breakPatterns_func(data, a, b)
			limit--
		}

		pivot, hint := choosePivot_func(data, a, b)
		if hint == decreasingHint {
			reverseRange_func(data, a, b)
			// The chosen pivot was pivot-a elements after the start of the array.
			// After reversing it is pivot-a elements before the end of the array.
			// The idea came from Rust's implementation.
			pivot = (b - 1) - (pivot - a)
			hint = increasingHint
		}

		// The slice is likely already sorted.
		if wasBalanced && wasPartitioned && hint == increasingHint {
			if partialInsertionSort_func(data, a, b) {
				return
			}
		}

		// Probably the slice contains many duplicate elements, partition the slice into
		// elements equal to and elements greater than the pivot.
		if a > 0 && !data.Less(a-1, pivot) {
			mid := partitionEqual_func(data, a, b, pivot)
			a = mid
			continue
		}

		mid, alreadyPartitioned := partition_func(data, a, b, pivot)
		wasPartitioned = alreadyPartitioned

		leftLen, rightLen := mid-a, b-mid
		balanceThreshold := length / 8
		if leftLen < rightLen {
			wasBalanced = leftLen >= balanceThreshold
			pdqsort_func(data, a, mid, limit)
			a = mid + 1
		} else {
			wasBalanced = rightLen >= balanceThreshold
			pdqsort_func(data, mid+1, b, limit)
			b = mid
		}
	}
}

// partition_func does one quicksort partition.
// Let p = data[pivot]
// Moves elements in data[a:b] around, so that data[i]<p and data[j]>=p for i<newpivot and j>newpivot.
// On return, data[newpivot] = p
func partition_func(data lessSwap, a, b, pivot int) (newpivot int, alreadyPartitioned bool) {
	data.Swap(a, pivot)
	i, j := a+1, b-1 // i and j are inclusive of the elements remaining to be partitioned

	for i <= j && data.Less(i, a) {
		i++
	}
	for i <= j && !data.Less(j, a) {
		j--
	}
	if i > j {
		data.Swap(j, a)
		return j, true
	}
	data.Swap(i, j)
	i++
	j--

	for {
		for i <= j && data.Less(i, a) {
			i++
		}
		for i <= j && !data.Less(j, a) {
			j--
		}
		if i > j {
			break
		}
		data.Swap(i, j)
		i++
		j--
	}
	data.Swap(j, a)
	return j, false
}

// partitionEqual_func partitions data[a:b] into elements equal to data[pivot] followed by elements greater than data[pivot].
// It assumed that data[a:b] does not contain elements smaller than the data[pivot].
func partitionEqual_func(data lessSwap, a, b, pivot int) (newpivot int) {
	data.Swap(a, pivot)
	i, j := a+1, b-1 // i and j are inclusive of the elements remaining to be partitioned

	for {
		for i <= j && !data.Less(a, i) {
			i++
		}
		for i <= j && data.Less(a, j) {
			j--
		}
		if i > j {
			break
		}
		data.Swap(i, j)
		i++
		j--
	}
	return i
}

// partialInsertionSort_func partially sorts a slice, returns true if the slice is sorted at the end.
func partialInsertionSort_func(data lessSwap, a, b int) bool {
	const (
		maxSteps         = 5  // maximum number of adjacent out-of-order pairs that will get shifted
		shortestShifting = 50 // don't shift any elements on short arrays
	)
	i := a + 1
	for j := 0; j < maxSteps; j++ {
		for i < b && !data.Less(i, i-1) {
			i++
		}

		if i == b {
			return true
		}

		if b-a < shortestShifting {
			return false
		}

		data.Swap(i, i-1)

		// Shift the smaller one to the left.
		if i-a >= 2 {
			for j := i - 1; j >= 1; j-- {
				if !data.Less(j, j-1) {
					break
				}
				data.Swap(j, j-1)
			}
		}
		// Shift the greater one to the right.
		if b-i >= 2 {
			for j := i + 1; j < b; j++ {
				if !data.Less(j, j-1) {
					break
				}
				data.Swap(j, j-1)
			}
		}
	}
	return false
}

// breakPatterns_func scatters some elements around in an attempt to break some patterns
// that might cause imbalanced partitions in quicksort.
func breakPatterns_func(data lessSwap, a, b int) {
	length := b - a
	if length >= 8 {
		random := xorshift(length)
		modulus := nextPowerOfTwo(length)

		for idx := a + (length/4)*2 - 1; idx <= a+(length/4)*2+1; idx++ {
			other := int(uint(random.Next()) & (modulus - 1))
			if other >= length {
				other -= length
			}
			data.Swap(idx, a+other)
		}
	}
}

// choosePivot_func chooses a pivot in data[a:b].
//
// [0,8): chooses a static pivot.
// [8,shortestNinther): uses the simple median-of-three method.
// [shortestNinther,∞): uses the Tukey ninther method.
func choosePivot_func(data lessSwap, a, b int) (pivot int, hint sortedHint) {
	const (
		shortestNinther = 50
		maxSwaps        = 4 * 3
	)

	l := b - a

	var (
		swaps int
		i     = a + l/4*1
		j     = a + l/4*2
		k     = a + l/4*3
	)

	if l >= 8 {
		if l >= shortestNinther {
			// Tukey ninther method, the idea came from Rust's implementation.
			i = medianAdjacent_func(data, i, &swaps)
			j = medianAdjacent_func(data, j, &swaps)
			k = medianAdjacent_func(data, k, &swaps)
		}
		// Find the median among i, j, k and stores it into j.
		j = median_func(data, i, j, k, &swaps)
	}

	switch swaps {
	case 0:
		return j, increasingHint
	case maxSwaps:
		return j, decreasingHint
	default:
		return j, unknownHint
	}
}

// order2_func returns x,y where data[x] <= data[y], where x,y=a,b or x,y=b,a.
func order2_func(data lessSwap, a, b int, swaps *int) (int, int) {
	if data.Less(b, a) {
		*swaps++
		return b, a
	}
	return a, b
}

// median_func returns x where data[x] is the median of data[a],data[b],data[c], where x is a, b, or c.
func median_func(data lessSwap, a, b, c int, swaps *int) int {
	a, b = order2_func(data, a, b, swaps)
	b, c = order2_func(data, b, c, swaps)
	a, b = order2_func(data, a, b, swaps)
	return b
}

// medianAdjacent_func finds the median of data[a - 1], data[a], data[a + 1] and stores the index into a.
func medianAdjacent_func(data lessSwap, a int, swaps *int) int {
	return median_func(data, a-1, a, a+1, swaps)
}

func reverseRange_func(data lessSwap, a, b int) {
	i := a
	j := b - 1
	for i < j {
		data.Swap(i, j)
		i++
		j--
	}
}

func swapRange_func(data lessSwap, a, b, n int) {
	for i := 0; i < n; i++ {
		data.Swap(a+i, b+i)
	}
}

func stable_func(data lessSwap, n int) {
	blockSize := 20 // must be > 0
	a, b := 0, blockSize
	for b <= n {
		insertionSort_func(data, a, b)
		a = b
		b += blockSize
	}
	insertionSort_func(data, a, n)

	for blockSize < n {
		a, b = 0, 2*blockSize
		for b <= n {
			symMerge_func(data, a, a+blockSize, b)
			a = b
			b += 2 * blockSize
		}
		if m := a + blockSize; m < n {
			symMerge_func(data, a, m, n)
		}
		blockSize *= 2
	}
}

// symMerge_func merges the two sorted subsequences data[a:m] and data[m:b] using
// the SymMerge algorithm from Pok-Son Kim and Arne Kutzner, "Stable Minimum
// Storage Merging by Symmetric Comparisons", in Susanne Albers and Tomasz
// Radzik, editors, Algorithms - ESA 2004, volume 3221 of Lecture Notes in
// Computer Science, pages 714-723. Springer, 2004.
//
// Let M = m-a and N = b-n. Wolog M < N.
// The recursion depth is bound by ceil(log(N+M)).
// The algorithm needs O(M*log(N/M + 1)) calls to data.Less.
// The algorithm needs O((M+N)*log(M)) calls to data.Swap.
//
// The paper gives O((M+N)*log(M)) as the number of assignments assuming a
// rotation algorithm which uses O(M+N+gcd(M+N)) assignments. The argumentation
// in the paper carries through for Swap operations, especially as the block
// swapping rotate uses only O(M+N) Swaps.
//
// symMerge assumes non-degenerate arguments: a < m && m < b.
// Having the caller check this condition eliminates many leaf recursion calls,
// which improves performance.
func symMerge_func(data lessSwap, a, m, b int) {
	// Avoid unnecessary recursions of symMerge
	// by direct insertion of data[a] into data[m:b]
	// if data[a:m] only contains one element.
	if m-a == 1 {
		// Use binary search to find the lowest index i
		// such that data[i] >= data[a] for m <= i < b.
		// Exit the search loop with i == b in case no such index exists.
		i := m
		j := b
		for i < j {
			h := int(uint(i+j) >> 1)
			if data.Less(h, a) {
				i = h + 1
			} else {
				j = h
			}
		}
		// Swap values until data[a] reaches the position before i.
		for k := a; k < i-1; k++ {
			data.Swap(k, k+1)
		}
		return
	}

	// Avoid unnecessary recursions of symMerge
	// by direct insertion of data[m] into data[a:m]
	// if data[m:b] only contains one element.
	if b-m == 1 {
		// Use binary search to find the lowest index i
		// such that data[i] > data[m] for a <= i < m.
		// Exit the search loop with i == m in case no such index exists.
		i := a
		j := m
		for i < j {
			h := int(uint(i+j) >> 1)
			if !data.Less(m, h) {
				i = h + 1
			} else {
				j = h
			}
		}
		// Swap values until data[m] reaches the position i.
		for k := m; k > i; k-- {
			data.Swap(k, k-1)
		}
		return
	}

	mid := int(uint(a+b) >> 1)
	n := mid + m
	var start, r int
	if m > mid {
		start = n - b
		r = mid
	} else {
		start = a
		r = m
	}
	p := n - 1

	for start < r {
		c := int(uint(start+r) >> 1)
		if !data.Less(p-c, c) {
			start = c + 1
		} else {
			r = c
		}
	}

	end := n - start
	if start < m && m < end {
		rotate_func(data, start, m, end)
	}
	if a < start && start < mid {
		symMerge_func(data, a, start, mid)
	}
	if mid < end && end < b {
		symMerge_func(data, mid, end, b)
	}
}

// rotate_func rotates two consecutive blocks u = data[a:m] and v = data[m:b] in data:
// Data of the form 'x u v y' is changed to 'x v u y'.
// rotate performs at most b-a many calls to data.Swap,
// and it assumes non-degenerate arguments: a < m && m < b.
func rotate_func(data lessSwap, a, m, b int) {
	i := m - a
	j := b - m

	for i != j {
		if i > j {
			swapRange_func(data, m-i, m, j)
			i -= j
		} else {
			swapRange_func(data, m-i, m+j-i, i)
			j -= i
		}
	}
	// i == j
	swapRange_func(data, m-i, m, i)
}
