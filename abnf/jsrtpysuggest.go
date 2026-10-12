package abnf

import "unicode/utf16"

// pyDidYouMean is CPython's suffix for an unexpected keyword argument,
// ". Did you mean 'x'?", or "" when no candidate is close enough. Twin of
// languages/lib/python-suggest.js, which explains the distance; like it, this
// measures UTF-16 code units.
func pyDidYouMean(name string, cands []string) string {
	if len(cands) >= 750 {
		return ""
	}
	nu := utf16.Encode([]rune(name))
	best, bestDist := "", -1
	for _, item := range cands {
		if item == name {
			continue
		}
		iu := utf16.Encode([]rune(item))
		maxd := (len(nu) + len(iu) + 3) * 2 / 6
		if bestDist >= 0 && bestDist-1 < maxd {
			maxd = bestDist - 1
		}
		d := pySuggestDistance(nu, iu, maxd)
		if d > maxd {
			continue
		}
		if bestDist < 0 || d < bestDist {
			best, bestDist = item, d
		}
	}
	if bestDist < 0 {
		return ""
	}
	return ". Did you mean '" + best + "'?"
}

func pySuggestSubCost(a, b uint16) int {
	if a%32 != b%32 {
		return 2
	}
	if a == b {
		return 0
	}
	if a >= 'A' && a <= 'Z' {
		a += 32
	}
	if b >= 'A' && b <= 'Z' {
		b += 32
	}
	if a == b {
		return 1
	}
	return 2
}

func pySuggestDistance(a, b []uint16, maxCost int) int {
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	if len(a) == 0 || len(b) == 0 {
		return (len(a) + len(b)) * 2
	}
	if len(a) > 40 || len(b) > 40 {
		return maxCost + 1
	}
	if len(b) < len(a) {
		a, b = b, a
	}
	if (len(b)-len(a))*2 > maxCost {
		return maxCost + 1
	}
	buf := make([]int, len(a))
	for i := range buf {
		buf[i] = (i + 1) * 2
	}
	result := 0
	for bi, code := range b {
		dist := bi * 2
		result = dist
		minimum := -1
		for ai := range a {
			sub := dist + pySuggestSubCost(code, a[ai])
			dist = buf[ai]
			insdel := result
			if dist < insdel {
				insdel = dist
			}
			result = insdel + 2
			if sub < result {
				result = sub
			}
			buf[ai] = result
			if minimum < 0 || result < minimum {
				minimum = result
			}
		}
		if minimum > maxCost {
			return maxCost + 1
		}
	}
	return result
}
