package abnf

// pySort is CPython's list.sort timsort: it sorts the keys k ascending by lt
// and permutes v, a distinct slice, in lockstep. Twin of
// languages/lib/python-sort.js, which explains why the comparison sequence is
// followed exactly; a panicking lt leaves both a permutation of their input.
type pySortState struct {
	k, v, tk, tv []interface{}
	lt           func(a, b interface{}) bool
	minGallop    int
	n            int
	base, ln     []int
	power        []int
}

func pySort(k, v []interface{}, lt func(a, b interface{}) bool) {
	n := len(k)
	if n < 2 {
		return
	}
	ms := &pySortState{k: k, v: v, lt: lt, minGallop: 7, n: n,
		tk: make([]interface{}, n), tv: make([]interface{}, n)}
	minrun := pySortMinrun(n)
	lo, rem := 0, n
	for rem > 0 {
		r := ms.countRun(lo, rem)
		if r < minrun {
			force := minrun
			if rem <= minrun {
				force = rem
			}
			ms.binarySort(lo, force, r)
			r = force
		}
		ms.foundNewRun(r)
		ms.base = append(ms.base, lo)
		ms.ln = append(ms.ln, r)
		ms.power = append(ms.power, 0)
		lo += r
		rem -= r
	}
	for len(ms.base) > 1 {
		i := len(ms.base) - 2
		if i > 0 && ms.ln[i-1] < ms.ln[i+1] {
			i--
		}
		ms.mergeAt(i)
	}
}

func pySortMinrun(n int) int {
	r := 0
	for n >= 64 {
		r |= n & 1
		n >>= 1
	}
	return n + r
}

func pySortRev(a []interface{}, lo, n int) {
	for i, j := lo, lo+n-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

func (ms *pySortState) revKV(lo, n int) {
	pySortRev(ms.k, lo, n)
	pySortRev(ms.v, lo, n)
}

func (ms *pySortState) countRun(lo, rem int) int {
	k, lt := ms.k, ms.lt
	n := 1
	for n < rem && !lt(k[lo+n], k[lo+n-1]) {
		n++
	}
	if n == rem {
		return n
	}
	if n > 1 {
		if lt(k[lo], k[lo+n-1]) {
			return n
		}
		ms.revKV(lo, n)
	}
	n++
	neq := 0
	for ; n < rem; n++ {
		if lt(k[lo+n], k[lo+n-1]) {
			if neq > 0 {
				neq++
				ms.revKV(lo+n-neq, neq)
				neq = 0
			}
		} else {
			if lt(k[lo+n-1], k[lo+n]) {
				break
			}
			neq++
		}
	}
	if neq > 0 {
		neq++
		ms.revKV(lo+n-neq, neq)
	}
	ms.revKV(lo, n)
	for n < rem && !lt(k[lo+n], k[lo+n-1]) {
		n++
	}
	return n
}

func (ms *pySortState) binarySort(lo, n, ok int) {
	k, v := ms.k, ms.v
	if ok == 0 {
		ok = 1
	}
	for ; ok < n; ok++ {
		l, r := 0, ok
		pivot := k[lo+ok]
		for l < r {
			m := (l + r) >> 1
			if ms.lt(pivot, k[lo+m]) {
				r = m
			} else {
				l = m + 1
			}
		}
		vp := v[lo+ok]
		for j := ok; j > l; j-- {
			k[lo+j] = k[lo+j-1]
			v[lo+j] = v[lo+j-1]
		}
		k[lo+l] = pivot
		v[lo+l] = vp
	}
}

func pySortPower(s1, n1, n2, n int) int {
	result := 0
	a := 2*s1 + n1
	b := a + n1 + n2
	for {
		result++
		if a >= n {
			a -= n
			b -= n
		} else if b >= n {
			break
		}
		a <<= 1
		b <<= 1
	}
	return result
}

func (ms *pySortState) foundNewRun(n2 int) {
	top := len(ms.base) - 1
	if top < 0 {
		return
	}
	power := pySortPower(ms.base[top], ms.ln[top], n2, ms.n)
	for len(ms.base) > 1 && ms.power[len(ms.base)-2] > power {
		ms.mergeAt(len(ms.base) - 2)
	}
	ms.power[len(ms.base)-1] = power
}

func (ms *pySortState) gallopLeft(key interface{}, a []interface{}, a0, n, hint int) int {
	lt := ms.lt
	p := a0 + hint
	lastofs, ofs := 0, 1
	if lt(a[p], key) {
		maxofs := n - hint
		for ofs < maxofs && lt(a[p+ofs], key) {
			lastofs = ofs
			ofs = ofs<<1 + 1
		}
		if ofs > maxofs {
			ofs = maxofs
		}
		lastofs += hint
		ofs += hint
	} else {
		maxofs := hint + 1
		for ofs < maxofs && !lt(a[p-ofs], key) {
			lastofs = ofs
			ofs = ofs<<1 + 1
		}
		if ofs > maxofs {
			ofs = maxofs
		}
		lastofs, ofs = hint-ofs, hint-lastofs
	}
	lastofs++
	for lastofs < ofs {
		m := lastofs + (ofs-lastofs)>>1
		if lt(a[a0+m], key) {
			lastofs = m + 1
		} else {
			ofs = m
		}
	}
	return ofs
}

func (ms *pySortState) gallopRight(key interface{}, a []interface{}, a0, n, hint int) int {
	lt := ms.lt
	p := a0 + hint
	lastofs, ofs := 0, 1
	if lt(key, a[p]) {
		maxofs := hint + 1
		for ofs < maxofs && lt(key, a[p-ofs]) {
			lastofs = ofs
			ofs = ofs<<1 + 1
		}
		if ofs > maxofs {
			ofs = maxofs
		}
		lastofs, ofs = hint-ofs, hint-lastofs
	} else {
		maxofs := n - hint
		for ofs < maxofs && !lt(key, a[p+ofs]) {
			lastofs = ofs
			ofs = ofs<<1 + 1
		}
		if ofs > maxofs {
			ofs = maxofs
		}
		lastofs += hint
		ofs += hint
	}
	lastofs++
	for lastofs < ofs {
		m := lastofs + (ofs-lastofs)>>1
		if lt(key, a[a0+m]) {
			ofs = m
		} else {
			lastofs = m + 1
		}
	}
	return ofs
}

// pySortCopy moves n keys and values; copy handles an overlap either way.
func pySortCopy(dk, dv []interface{}, di int, sk, sv []interface{}, si, n int) {
	copy(dk[di:di+n], sk[si:si+n])
	copy(dv[di:di+n], sv[si:si+n])
}

func (ms *pySortState) mergeAt(i int) {
	sa, na := ms.base[i], ms.ln[i]
	sb, nb := ms.base[i+1], ms.ln[i+1]
	ms.ln[i] = na + nb
	if i == len(ms.base)-3 {
		ms.base[i+1], ms.ln[i+1], ms.power[i+1] = ms.base[i+2], ms.ln[i+2], ms.power[i+2]
	}
	last := len(ms.base) - 1
	ms.base, ms.ln, ms.power = ms.base[:last], ms.ln[:last], ms.power[:last]
	g := ms.gallopRight(ms.k[sb], ms.k, sa, na, 0)
	sa += g
	na -= g
	if na == 0 {
		return
	}
	nb = ms.gallopLeft(ms.k[sa+na-1], ms.k, sb, nb, nb-1)
	if nb <= 0 {
		return
	}
	if na <= nb {
		ms.mergeLo(sa, na, sb, nb)
	} else {
		ms.mergeHi(sa, na, sb, nb)
	}
}

func (ms *pySortState) mergeLo(sa, na, sb, nb int) {
	k, v, tk, tv := ms.k, ms.v, ms.tk, ms.tv
	pySortCopy(tk, tv, 0, k, v, sa, na)
	d, pa, pb := sa, 0, sb
	k[d], v[d] = k[pb], v[pb]
	d++
	pb++
	nb--
	copyB := false
	done := nb == 0
	if !done && na == 1 {
		copyB, done = true, true
	}
	ok := false
	defer func() {
		if !ok {
			pySortCopy(k, v, d, tk, tv, pa, na)
		}
	}()
	mg := ms.minGallop
	for !done {
		acount, bcount := 0, 0
		for {
			if ms.lt(k[pb], tk[pa]) {
				k[d], v[d] = k[pb], v[pb]
				d++
				pb++
				bcount++
				acount = 0
				nb--
				if nb == 0 {
					done = true
					break
				}
				if bcount >= mg {
					break
				}
			} else {
				k[d], v[d] = tk[pa], tv[pa]
				d++
				pa++
				acount++
				bcount = 0
				na--
				if na == 1 {
					copyB, done = true, true
					break
				}
				if acount >= mg {
					break
				}
			}
		}
		if done {
			break
		}
		mg++
		for {
			if mg > 1 {
				mg--
			}
			ms.minGallop = mg
			g := ms.gallopRight(k[pb], tk, pa, na, 0)
			acount = g
			if g > 0 {
				pySortCopy(k, v, d, tk, tv, pa, g)
				d += g
				pa += g
				na -= g
				if na == 1 {
					copyB, done = true, true
					break
				}
				if na == 0 {
					done = true
					break
				}
			}
			k[d], v[d] = k[pb], v[pb]
			d++
			pb++
			nb--
			if nb == 0 {
				done = true
				break
			}
			g = ms.gallopLeft(tk[pa], k, pb, nb, 0)
			bcount = g
			if g > 0 {
				pySortCopy(k, v, d, k, v, pb, g)
				d += g
				pb += g
				nb -= g
				if nb == 0 {
					done = true
					break
				}
			}
			k[d], v[d] = tk[pa], tv[pa]
			d++
			pa++
			na--
			if na == 1 {
				copyB, done = true, true
				break
			}
			if acount < 7 && bcount < 7 {
				break
			}
		}
		if done {
			break
		}
		mg++
		ms.minGallop = mg
	}
	ok = true
	if copyB {
		pySortCopy(k, v, d, k, v, pb, nb)
		k[d+nb], v[d+nb] = tk[pa], tv[pa]
	} else {
		pySortCopy(k, v, d, tk, tv, pa, na)
	}
}

func (ms *pySortState) mergeHi(sa, na, sb, nb int) {
	k, v, tk, tv := ms.k, ms.v, ms.tk, ms.tv
	pySortCopy(tk, tv, 0, k, v, sb, nb)
	d, pa, pb := sb+nb-1, sa+na-1, nb-1
	k[d], v[d] = k[pa], v[pa]
	d--
	pa--
	na--
	copyA := false
	done := na == 0
	if !done && nb == 1 {
		copyA, done = true, true
	}
	ok := false
	defer func() {
		if !ok {
			pySortCopy(k, v, d-nb+1, tk, tv, 0, nb)
		}
	}()
	mg := ms.minGallop
	for !done {
		acount, bcount := 0, 0
		for {
			if ms.lt(tk[pb], k[pa]) {
				k[d], v[d] = k[pa], v[pa]
				d--
				pa--
				acount++
				bcount = 0
				na--
				if na == 0 {
					done = true
					break
				}
				if acount >= mg {
					break
				}
			} else {
				k[d], v[d] = tk[pb], tv[pb]
				d--
				pb--
				bcount++
				acount = 0
				nb--
				if nb == 1 {
					copyA, done = true, true
					break
				}
				if bcount >= mg {
					break
				}
			}
		}
		if done {
			break
		}
		mg++
		for {
			if mg > 1 {
				mg--
			}
			ms.minGallop = mg
			g := na - ms.gallopRight(tk[pb], k, sa, na, na-1)
			acount = g
			if g > 0 {
				d -= g
				pa -= g
				pySortCopy(k, v, d+1, k, v, pa+1, g)
				na -= g
				if na == 0 {
					done = true
					break
				}
			}
			k[d], v[d] = tk[pb], tv[pb]
			d--
			pb--
			nb--
			if nb == 1 {
				copyA, done = true, true
				break
			}
			g = nb - ms.gallopLeft(k[pa], tk, 0, nb, nb-1)
			bcount = g
			if g > 0 {
				d -= g
				pb -= g
				pySortCopy(k, v, d+1, tk, tv, pb+1, g)
				nb -= g
				if nb == 1 {
					copyA, done = true, true
					break
				}
				if nb == 0 {
					done = true
					break
				}
			}
			k[d], v[d] = k[pa], v[pa]
			d--
			pa--
			na--
			if na == 0 {
				done = true
				break
			}
			if acount < 7 && bcount < 7 {
				break
			}
		}
		if done {
			break
		}
		mg++
		ms.minGallop = mg
	}
	ok = true
	if copyA {
		pySortCopy(k, v, d+1-na, k, v, pa+1-na, na)
		d -= na
		k[d], v[d] = tk[pb], tv[pb]
	} else {
		pySortCopy(k, v, d-nb+1, tk, tv, 0, nb)
	}
}
