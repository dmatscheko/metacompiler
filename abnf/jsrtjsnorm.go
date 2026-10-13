package abnf

import (
	"sort"
	"strings"
)

// String.prototype.normalize for the js and typescript compilers: a port of
// languages/lib/js-norm.js (which the interpreters include and layer 2 imports), over
// the tables of languages/lib/unicode-norm.txt, which a program that names a
// .normalize member registers at boot through js_jsnormdata. js-norm.js's head
// comment has the format and the algorithm.

type jsvNormTables struct {
	kind  map[int]byte  // 'c', 'd' or 'k'
	dmap  map[int][]int // the decomposition, one level deep
	comp  map[[2]int]int
	lo    []int // the combining-class runs, ascending
	hi    []int
	class []int
}

var jsvNorm *jsvNormTables

func jsvNormHex(t string) int {
	v := 0
	for i := 0; i < len(t); i++ {
		c := int(t[i])
		if c <= '9' {
			v = v*16 + c - '0'
		} else {
			v = v*16 + c - 'a' + 10
		}
	}
	return v
}

func jsvNormData(text string) {
	if jsvNorm != nil {
		return
	}
	n := &jsvNormTables{kind: map[int]byte{}, dmap: map[int][]int{}, comp: map[[2]int]int{}}
	for len(text) > 0 {
		line := text
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			line, text = text[:i], text[i+1:]
		} else {
			text = ""
		}
		if len(line) == 0 {
			continue
		}
		var nums []string
		s := 1
		for j := 1; j <= len(line); j++ {
			if j == len(line) || line[j] == ' ' {
				if j > s {
					nums = append(nums, line[s:j])
				}
				s = j + 1
			}
		}
		switch line[0] {
		case 'n':
			c := 0
			for i := 0; i < len(nums[2]); i++ {
				c = c*10 + int(nums[2][i]-'0')
			}
			n.lo = append(n.lo, jsvNormHex(nums[0]))
			n.hi = append(n.hi, jsvNormHex(nums[1]))
			n.class = append(n.class, c)
		case 'c', 'd', 'k':
			cp := jsvNormHex(nums[0])
			m := make([]int, 0, len(nums)-1)
			for _, x := range nums[1:] {
				m = append(m, jsvNormHex(x))
			}
			n.kind[cp] = line[0]
			n.dmap[cp] = m
			if line[0] == 'c' {
				n.comp[[2]int{m[0], m[1]}] = cp
			}
		}
	}
	jsvNorm = n
}

func (n *jsvNormTables) classOf(cp int) int {
	if cp < 768 {
		return 0
	}
	i := sort.Search(len(n.hi), func(i int) bool { return n.hi[i] >= cp })
	if i < len(n.hi) && n.lo[i] <= cp {
		return n.class[i]
	}
	return 0
}

func (n *jsvNormTables) decomp(cp int, compat bool, out []int) []int {
	if cp >= 0xAC00 && cp <= 0xD7A3 {
		si := cp - 0xAC00
		out = append(out, 0x1100+si/588, 0x1161+(si%588)/28)
		if si%28 != 0 {
			out = append(out, 0x11A7+si%28)
		}
		return out
	}
	k, ok := n.kind[cp]
	if !ok || (k == 'k' && !compat) {
		return append(out, cp)
	}
	for _, x := range n.dmap[cp] {
		out = n.decomp(x, compat, out)
	}
	return out
}

func (n *jsvNormTables) compose(a, b int) int {
	if a >= 0x1100 && a < 0x1113 && b >= 0x1161 && b < 0x1176 {
		return 0xAC00 + ((a-0x1100)*21+(b-0x1161))*28
	}
	if a >= 0xAC00 && a <= 0xD7A3 && (a-0xAC00)%28 == 0 && b > 0x11A7 && b < 0x11C3 {
		return a + (b - 0x11A7)
	}
	if p, ok := n.comp[[2]int{a, b}]; ok {
		return p
	}
	return -1
}

// jsvNormalize is s in form, one of NFC, NFD, NFKC, NFKD. Twin of jsnormApply.
func (rt *jsrt) jsvNormalize(s string, form string) string {
	units := strUnits(s)
	quick := true
	for _, c := range units {
		if c >= 160 {
			quick = false
			break
		}
	}
	if quick {
		return s
	}
	n := jsvNorm
	if n == nil {
		rt.fail("normalize: no tables registered (js_jsnormdata)")
	}
	compat := form == "NFKC" || form == "NFKD"
	var cps []int
	for i := 0; i < len(units); i++ {
		c := int(units[i])
		if c >= 0xD800 && c <= 0xDBFF && i+1 < len(units) {
			if d := int(units[i+1]); d >= 0xDC00 && d <= 0xDFFF {
				cps = n.decomp(0x10000+(c-0xD800)*1024+(d-0xDC00), compat, cps)
				i++
				continue
			}
		}
		cps = n.decomp(c, compat, cps)
	}
	cls := make([]int, len(cps))
	for i, cp := range cps {
		cls[i] = n.classOf(cp)
	}
	for x := 1; x < len(cps); x++ {
		for y := x; y > 0 && cls[y] != 0 && cls[y-1] > cls[y]; y-- {
			cps[y], cps[y-1] = cps[y-1], cps[y]
			cls[y], cls[y-1] = cls[y-1], cls[y]
		}
	}
	if form == "NFC" || form == "NFKC" {
		var out []int
		si, last := -1, 0
		for i, cp := range cps {
			cc := cls[i]
			if si >= 0 && (len(out)-1 == si || (last != 0 && last < cc)) {
				if p := n.compose(out[si], cp); p >= 0 {
					out[si] = p
					continue
				}
			}
			if cc == 0 {
				si = len(out)
			}
			last = cc
			out = append(out, cp)
		}
		cps = out
	}
	var res []uint16
	for _, v := range cps {
		if v >= 0x10000 {
			res = append(res, uint16(0xD800+(v-0x10000)/1024), uint16(0xDC00+(v-0x10000)%1024))
		} else {
			res = append(res, uint16(v))
		}
	}
	return strFromUnits(res)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		m["js_jsnormdata"] = func(a []uint64) uint64 {
			jsvNormData(rt.toString(rt.unwrap(a[0])))
			return 0
		}
	})
}
