package abnf

// java.util.TimSort (JDK 24), comparison for comparison: the Go twin of
// languages/lib/java-timsort.js, which documents the algorithm and why the call
// sequence matters. jtsSort sorts a in place under cmp (a number whose sign orders
// x against y) and answers false when cmp was found to violate its contract.

type jtsState struct {
	a         []interface{}
	cmp       func(x, y interface{}) int
	minGallop int
	base, len []int
	bad       bool
}

func jtsSort(a []interface{}, cmp func(x, y interface{}) int) bool {
	n := len(a)
	if n < 2 {
		return true
	}
	if n < 32 {
		jtsBinarySort(a, 0, n, jtsCountRun(a, 0, n, cmp), cmp)
		return true
	}
	ts := &jtsState{a: a, cmp: cmp, minGallop: 7}
	minRun := jtsMinRun(n)
	lo, rem := 0, n
	for rem > 0 {
		r := jtsCountRun(a, lo, lo+rem, cmp)
		if r < minRun {
			force := minRun
			if rem <= minRun {
				force = rem
			}
			jtsBinarySort(a, lo, lo+force, lo+r, cmp)
			r = force
		}
		ts.base = append(ts.base, lo)
		ts.len = append(ts.len, r)
		ts.mergeCollapse()
		if ts.bad {
			return false
		}
		lo += r
		rem -= r
	}
	for len(ts.base) > 1 {
		k := len(ts.base) - 2
		if k > 0 && ts.len[k-1] < ts.len[k+1] {
			k--
		}
		ts.mergeAt(k)
		if ts.bad {
			return false
		}
	}
	return true
}

func jtsMinRun(n int) int {
	r := 0
	for n >= 32 {
		r |= n & 1
		n >>= 1
	}
	return n + r
}

func jtsCountRun(a []interface{}, lo, hi int, cmp func(x, y interface{}) int) int {
	runHi := lo + 1
	if runHi == hi {
		return 1
	}
	runHi++
	if cmp(a[lo+1], a[lo]) < 0 {
		for runHi < hi && cmp(a[runHi], a[runHi-1]) < 0 {
			runHi++
		}
		for i, j := lo, runHi-1; i < j; i, j = i+1, j-1 {
			a[i], a[j] = a[j], a[i]
		}
	} else {
		for runHi < hi && cmp(a[runHi], a[runHi-1]) >= 0 {
			runHi++
		}
	}
	return runHi - lo
}

func jtsBinarySort(a []interface{}, lo, hi, start int, cmp func(x, y interface{}) int) {
	if start == lo {
		start++
	}
	for ; start < hi; start++ {
		pivot := a[start]
		left, right := lo, start
		for left < right {
			mid := (left + right) >> 1
			if cmp(pivot, a[mid]) < 0 {
				right = mid
			} else {
				left = mid + 1
			}
		}
		copy(a[left+1:start+1], a[left:start])
		a[left] = pivot
	}
}

func (ts *jtsState) mergeCollapse() {
	for len(ts.base) > 1 {
		rl := ts.len
		n := len(ts.base) - 2
		if (n > 0 && rl[n-1] <= rl[n]+rl[n+1]) || (n > 1 && rl[n-2] <= rl[n]+rl[n-1]) {
			if rl[n-1] < rl[n+1] {
				n--
			}
		} else if rl[n] > rl[n+1] {
			return
		}
		ts.mergeAt(n)
		if ts.bad {
			return
		}
	}
}

func (ts *jtsState) mergeAt(i int) {
	a := ts.a
	base1, len1 := ts.base[i], ts.len[i]
	base2, len2 := ts.base[i+1], ts.len[i+1]
	ts.len[i] = len1 + len2
	if i == len(ts.base)-3 {
		ts.base[i+1] = ts.base[i+2]
		ts.len[i+1] = ts.len[i+2]
	}
	ts.base = ts.base[:len(ts.base)-1]
	ts.len = ts.len[:len(ts.len)-1]
	k := jtsGallopRight(a[base2], a, base1, len1, 0, ts.cmp)
	base1 += k
	len1 -= k
	if len1 == 0 {
		return
	}
	len2 = jtsGallopLeft(a[base1+len1-1], a, base2, len2, len2-1, ts.cmp)
	if len2 == 0 {
		return
	}
	if len1 <= len2 {
		ts.mergeLo(base1, len1, base2, len2)
	} else {
		ts.mergeHi(base1, len1, base2, len2)
	}
}

func jtsGallopLeft(key interface{}, a []interface{}, base, n, hint int, cmp func(x, y interface{}) int) int {
	lastOfs, ofs := 0, 1
	if cmp(key, a[base+hint]) > 0 {
		maxOfs := n - hint
		for ofs < maxOfs && cmp(key, a[base+hint+ofs]) > 0 {
			lastOfs = ofs
			ofs = ofs*2 + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		lastOfs += hint
		ofs += hint
	} else {
		maxOfs := hint + 1
		for ofs < maxOfs && cmp(key, a[base+hint-ofs]) <= 0 {
			lastOfs = ofs
			ofs = ofs*2 + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		lastOfs, ofs = hint-ofs, hint-lastOfs
	}
	lastOfs++
	for lastOfs < ofs {
		m := lastOfs + (ofs-lastOfs)/2
		if cmp(key, a[base+m]) > 0 {
			lastOfs = m + 1
		} else {
			ofs = m
		}
	}
	return ofs
}

func jtsGallopRight(key interface{}, a []interface{}, base, n, hint int, cmp func(x, y interface{}) int) int {
	ofs, lastOfs := 1, 0
	if cmp(key, a[base+hint]) < 0 {
		maxOfs := hint + 1
		for ofs < maxOfs && cmp(key, a[base+hint-ofs]) < 0 {
			lastOfs = ofs
			ofs = ofs*2 + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		lastOfs, ofs = hint-ofs, hint-lastOfs
	} else {
		maxOfs := n - hint
		for ofs < maxOfs && cmp(key, a[base+hint+ofs]) >= 0 {
			lastOfs = ofs
			ofs = ofs*2 + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		lastOfs += hint
		ofs += hint
	}
	lastOfs++
	for lastOfs < ofs {
		m := lastOfs + (ofs-lastOfs)/2
		if cmp(key, a[base+m]) < 0 {
			ofs = m
		} else {
			lastOfs = m + 1
		}
	}
	return ofs
}

func (ts *jtsState) mergeLo(base1, len1, base2, len2 int) {
	a, cmp := ts.a, ts.cmp
	tmp := append([]interface{}{}, a[base1:base1+len1]...)
	cursor1, cursor2, dest := 0, base2, base1
	a[dest] = a[cursor2]
	dest++
	cursor2++
	len2--
	if len2 == 0 {
		copy(a[dest:dest+len1], tmp[cursor1:cursor1+len1])
		return
	}
	if len1 == 1 {
		copy(a[dest:dest+len2], a[cursor2:cursor2+len2])
		a[dest+len2] = tmp[cursor1]
		return
	}
	minGallop := ts.minGallop
outer:
	for {
		count1, count2 := 0, 0
		for {
			if cmp(a[cursor2], tmp[cursor1]) < 0 {
				a[dest] = a[cursor2]
				dest++
				cursor2++
				count2++
				count1 = 0
				len2--
				if len2 == 0 {
					break outer
				}
			} else {
				a[dest] = tmp[cursor1]
				dest++
				cursor1++
				count1++
				count2 = 0
				len1--
				if len1 == 1 {
					break outer
				}
			}
			if (count1 | count2) >= minGallop {
				break
			}
		}
		for {
			count1 = jtsGallopRight(a[cursor2], tmp, cursor1, len1, 0, cmp)
			if count1 != 0 {
				copy(a[dest:dest+count1], tmp[cursor1:cursor1+count1])
				dest += count1
				cursor1 += count1
				len1 -= count1
				if len1 <= 1 {
					break outer
				}
			}
			a[dest] = a[cursor2]
			dest++
			cursor2++
			len2--
			if len2 == 0 {
				break outer
			}
			count2 = jtsGallopLeft(tmp[cursor1], a, cursor2, len2, 0, cmp)
			if count2 != 0 {
				copy(a[dest:dest+count2], a[cursor2:cursor2+count2])
				dest += count2
				cursor2 += count2
				len2 -= count2
				if len2 == 0 {
					break outer
				}
			}
			a[dest] = tmp[cursor1]
			dest++
			cursor1++
			len1--
			if len1 == 1 {
				break outer
			}
			minGallop--
			if count1 < 7 && count2 < 7 {
				break
			}
		}
		if minGallop < 0 {
			minGallop = 0
		}
		minGallop += 2
	}
	if minGallop < 1 {
		minGallop = 1
	}
	ts.minGallop = minGallop
	switch {
	case len1 == 1:
		copy(a[dest:dest+len2], a[cursor2:cursor2+len2])
		a[dest+len2] = tmp[cursor1]
	case len1 == 0:
		ts.bad = true
	default:
		copy(a[dest:dest+len1], tmp[cursor1:cursor1+len1])
	}
}

func (ts *jtsState) mergeHi(base1, len1, base2, len2 int) {
	a, cmp := ts.a, ts.cmp
	tmp := append([]interface{}{}, a[base2:base2+len2]...)
	cursor1, cursor2, dest := base1+len1-1, len2-1, base2+len2-1
	a[dest] = a[cursor1]
	dest--
	cursor1--
	len1--
	if len1 == 0 {
		copy(a[dest-(len2-1):dest+1], tmp[:len2])
		return
	}
	if len2 == 1 {
		dest -= len1
		cursor1 -= len1
		copy(a[dest+1:dest+1+len1], a[cursor1+1:cursor1+1+len1])
		a[dest] = tmp[cursor2]
		return
	}
	minGallop := ts.minGallop
outer:
	for {
		count1, count2 := 0, 0
		for {
			if cmp(tmp[cursor2], a[cursor1]) < 0 {
				a[dest] = a[cursor1]
				dest--
				cursor1--
				count1++
				count2 = 0
				len1--
				if len1 == 0 {
					break outer
				}
			} else {
				a[dest] = tmp[cursor2]
				dest--
				cursor2--
				count2++
				count1 = 0
				len2--
				if len2 == 1 {
					break outer
				}
			}
			if (count1 | count2) >= minGallop {
				break
			}
		}
		for {
			count1 = len1 - jtsGallopRight(tmp[cursor2], a, base1, len1, len1-1, cmp)
			if count1 != 0 {
				dest -= count1
				cursor1 -= count1
				len1 -= count1
				copy(a[dest+1:dest+1+count1], a[cursor1+1:cursor1+1+count1])
				if len1 == 0 {
					break outer
				}
			}
			a[dest] = tmp[cursor2]
			dest--
			cursor2--
			len2--
			if len2 == 1 {
				break outer
			}
			count2 = len2 - jtsGallopLeft(a[cursor1], tmp, 0, len2, len2-1, cmp)
			if count2 != 0 {
				dest -= count2
				cursor2 -= count2
				len2 -= count2
				copy(a[dest+1:dest+1+count2], tmp[cursor2+1:cursor2+1+count2])
				if len2 <= 1 {
					break outer
				}
			}
			a[dest] = a[cursor1]
			dest--
			cursor1--
			len1--
			if len1 == 0 {
				break outer
			}
			minGallop--
			if count1 < 7 && count2 < 7 {
				break
			}
		}
		if minGallop < 0 {
			minGallop = 0
		}
		minGallop += 2
	}
	if minGallop < 1 {
		minGallop = 1
	}
	ts.minGallop = minGallop
	switch {
	case len2 == 1:
		dest -= len1
		cursor1 -= len1
		copy(a[dest+1:dest+1+len1], a[cursor1+1:cursor1+1+len1])
		a[dest] = tmp[cursor2]
	case len2 == 0:
		ts.bad = true
	default:
		copy(a[dest-(len2-1):dest+1], tmp[:len2])
	}
}
