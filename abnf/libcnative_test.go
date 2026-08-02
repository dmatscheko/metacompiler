package abnf

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/llir/llvm/ir"
)

// TestLibcNative is the differential test for the eighteen byte/address libc names that
// entered libcExterns/libcNative on 2026-08-02, once 93c814f had made a C `char` a real
// one-byte cell. The SAME operation is computed by libcNative over the machine's arena
// and by Go's own strings/bytes/strconv over ordinary Go values, for every edge case the
// hand-written implementations have a chance of getting wrong.
//
// WHY THIS LIVES HERE rather than in tests/c-test-full.c, which is where every other
// C-level claim in this tree is pinned. That file is run by BOTH halves of the language,
// and languages/c-interpreter.abnf has no standard library at all - its callFunction
// knows putchar and getchar and nothing else - so a section calling any of these names
// makes that half stop dead. test.sh --full records a half's assertion COUNT only when
// that half has ZERO gaps (see the `gaps -eq 0` guard in full_probe), so one red section
// costs the whole count and the two halves are reported as disagreeing. Measured, with
// the section present:
//
//	c-interpreter:  error without a usable position: function not defined: strcpy
//	                - 46 the libc byte/address family: function not defined: strcpy
//	c-to-llvm-ir:   FULL - 495 assertions, goja and -frozen byte-identical
//	c            MISMATCH: - 495          <- 0 languages disagreeing is an invariant
//
// So the ratchet file cannot host these until the interpreter grammar grows a libc, and
// the file's own header already puts the standard library out of scope. That leaves
// condition 1 of the bar at libcExterns (clang's real symbol answers what cc answers)
// measured by hand and recorded there, and condition 2 (llvm.Run implements the same
// thing) pinned here, where it can regress under `go test ./abnf/`.
//
// Go is a fair oracle for exactly the properties that matter: it compares bytes
// unsigned, like C, and it has no opinion about the arena.
func TestLibcNative(t *testing.T) {
	// A fresh machine per case, so a bump allocation in one cannot reach the next.
	newMa := func() *machine { return newMachine(ir.NewModule(), "") }
	put := func(ma *machine, s string) uint64 {
		a := ma.alloc(uint64(len(s)) + 1)
		copy(ma.mem[a:], s)
		return a
	}
	// room reserves a writable destination of n bytes, pre-filled with 'Q' so that any
	// byte an implementation fails to write is visible instead of being a lucky zero.
	room := func(ma *machine, n int) uint64 {
		a := ma.alloc(uint64(n))
		for i := 0; i < n; i++ {
			ma.mem[a+uint64(i)] = 'Q'
		}
		return a
	}
	raw := func(ma *machine, a uint64, n int) string { return string(ma.mem[a : a+uint64(n)]) }
	cstr := func(ma *machine, a uint64) string { return string(ma.mem[a:ma.cstrEnd(a)]) }
	call := func(ma *machine, name string, args ...uint64) uint64 {
		fn := libcNative(ma, name)
		if fn == nil {
			t.Fatalf("libcNative(%q) is nil, but libcExterns lists it - the two must stay in step", name)
		}
		return fn(args)
	}
	sgn := func(v uint64) int {
		switch s := signedOf(v, 32); {
		case s < 0:
			return -1
		case s > 0:
			return 1
		}
		return 0
	}

	// Every name in libcExterns must be implemented here, and nothing may be implemented
	// that is not listed - that is the bar documented at libcExterns, stated as a test.
	for name := range libcExterns {
		switch name {
		case "putchar", "getchar", "puts", "abs": // resolveExtern's own built-ins
			continue
		}
		if libcNative(newMa(), name) == nil {
			t.Errorf("libcExterns lists %q but libcNative does not implement it: "+
				"llvm.Run would panic where -exe links the real symbol", name)
		}
	}

	words := []string{"", "a", "b", "ab", "abc", "abd", "abcd", "hello", "xy", "xc",
		"\x01", "\x80", "\x7f\x80", "aaa", "abcabc", "  +13xyz"}

	// ----- strlen -----
	for _, s := range words {
		ma := newMa()
		if got, want := call(ma, "strlen", put(ma, s)), uint64(len(s)); got != want {
			t.Errorf("strlen(%q) = %d, want %d", s, got, want)
		}
	}

	// ----- strcmp / strncmp / memcmp: SIGN against Go, MAGNITUDE against the unsigned
	// byte difference Apple's libc returns (see byteDiff; a normalized -1/0/1 here would
	// make llvm.Run and the -exe binary disagree).
	for _, a := range words {
		for _, b := range words {
			ma := newMa()
			pa, pb := put(ma, a), put(ma, b)
			if got, want := sgn(call(ma, "strcmp", pa, pb)), strings.Compare(a, b); got != want {
				t.Errorf("sign of strcmp(%q, %q) = %d, want %d", a, b, got, want)
			}
			for n := 0; n <= 5; n++ {
				ta, tb := a, b
				if len(ta) > n {
					ta = ta[:n]
				}
				if len(tb) > n {
					tb = tb[:n]
				}
				got := sgn(call(ma, "strncmp", pa, pb, uint64(n)))
				if want := strings.Compare(ta, tb); got != want {
					t.Errorf("sign of strncmp(%q, %q, %d) = %d, want %d", a, b, n, got, want)
				}
			}
			// memcmp does not stop at a NUL, so it needs equal-length operands.
			if len(a) == len(b) {
				got := sgn(call(ma, "memcmp", pa, pb, uint64(len(a))))
				if want := bytes.Compare([]byte(a), []byte(b)); got != want {
					t.Errorf("sign of memcmp(%q, %q, %d) = %d, want %d", a, b, len(a), got, want)
				}
			}
		}
	}
	{
		ma := newMa()
		// The magnitudes measured from the real symbol, arm64-darwin, Apple clang 21.0.0.
		type mag struct {
			a, b string
			n    uint64
			want int64
		}
		for _, c := range []mag{{"xy", "xc", 2, 22}, {"xc", "xy", 2, -22},
			{"\x80", "\x01", 1, 127}, {"\x01", "\x80", 1, -127},
			{"a", "b", 1, -1}, {"abc", "abc", 3, 0}} {
			pa, pb := put(ma, c.a), put(ma, c.b)
			for _, name := range []string{"strcmp", "strncmp", "memcmp"} {
				args := []uint64{pa, pb}
				if name != "strcmp" {
					args = append(args, c.n)
				}
				if got := signedOf(call(ma, name, args...), 32); got != c.want {
					t.Errorf("%s(%q, %q) = %d, want %d (the UNSIGNED byte difference)",
						name, c.a, c.b, got, c.want)
				}
			}
		}
	}

	// ----- strcpy / strncpy. strncpy pads to n and does NOT terminate on truncation.
	for _, s := range words {
		ma := newMa()
		d := room(ma, len(s)+4)
		if got := call(ma, "strcpy", d, put(ma, s)); got != d {
			t.Errorf("strcpy answered %d, want its destination %d", got, d)
		}
		if got := cstr(ma, d); got != s {
			t.Errorf("strcpy(_, %q) wrote %q", s, got)
		}
		if ma.mem[d+uint64(len(s))+1] != 'Q' {
			t.Errorf("strcpy(_, %q) wrote past the terminator", s)
		}
		for n := 0; n <= len(s)+3; n++ {
			ma := newMa()
			d := room(ma, n+2)
			call(ma, "strncpy", d, put(ma, s), uint64(n))
			want := make([]byte, n)
			copy(want, s)
			if got := raw(ma, d, n); got != string(want) {
				t.Errorf("strncpy(_, %q, %d) wrote %q, want %q", s, n, got, string(want))
			}
			if ma.mem[d+uint64(n)] != 'Q' {
				t.Errorf("strncpy(_, %q, %d) wrote past n", s, n)
			}
		}
	}

	// ----- strcat / strncat. strncat appends at most n bytes and ALWAYS terminates.
	for _, a := range words {
		for _, b := range words {
			ma := newMa()
			d := room(ma, len(a)+len(b)+2)
			call(ma, "strcpy", d, put(ma, a))
			if got := call(ma, "strcat", d, put(ma, b)); got != d {
				t.Errorf("strcat answered %d, want its destination %d", got, d)
			}
			if got, want := cstr(ma, d), a+b; got != want {
				t.Errorf("strcat(%q, %q) = %q, want %q", a, b, got, want)
			}
			for n := 0; n <= len(b)+2; n++ {
				ma := newMa()
				d := room(ma, len(a)+len(b)+2)
				call(ma, "strcpy", d, put(ma, a))
				call(ma, "strncat", d, put(ma, b), uint64(n))
				tb := b
				if len(tb) > n {
					tb = tb[:n]
				}
				if got, want := cstr(ma, d), a+tb; got != want {
					t.Errorf("strncat(%q, %q, %d) = %q, want %q", a, b, n, got, want)
				}
			}
		}
	}

	// ----- strchr / strrchr / strstr. Both char searches see the TERMINATOR too, which
	// is why strchr(s, 0) is the end of the string and not a null answer.
	for _, s := range words {
		ma := newMa()
		p := put(ma, s)
		for _, ch := range []byte{'a', 'b', 'c', 'z', 0x80, 0} {
			want := strings.IndexByte(s, ch)
			if ch == 0 {
				want = len(s)
			}
			got := call(ma, "strchr", p, uint64(ch))
			if want < 0 {
				if got != 0 {
					t.Errorf("strchr(%q, %d) = %d, want null", s, ch, got)
				}
			} else if got != p+uint64(want) {
				t.Errorf("strchr(%q, %d) at offset %d, want %d", s, ch, got-p, want)
			}
			want = strings.LastIndexByte(s, ch)
			if ch == 0 {
				want = len(s)
			}
			got = call(ma, "strrchr", p, uint64(ch))
			if want < 0 {
				if got != 0 {
					t.Errorf("strrchr(%q, %d) = %d, want null", s, ch, got)
				}
			} else if got != p+uint64(want) {
				t.Errorf("strrchr(%q, %d) at offset %d, want %d", s, ch, got-p, want)
			}
		}
		for _, needle := range words {
			ma := newMa()
			h, n := put(ma, s), put(ma, needle)
			got, want := call(ma, "strstr", h, n), strings.Index(s, needle)
			if want < 0 {
				if got != 0 {
					t.Errorf("strstr(%q, %q) = %d, want null", s, needle, got)
				}
			} else if got != h+uint64(want) {
				t.Errorf("strstr(%q, %q) at offset %d, want %d", s, needle, got-h, want)
			}
		}
	}

	// ----- memcpy / memmove / memset / memchr, including the two overlap directions
	// that are the whole reason memmove exists.
	{
		const src = "abcdefgh"
		for _, n := range []int{0, 1, 3, 8} {
			ma := newMa()
			s, d := put(ma, src), room(ma, n+1)
			if got := call(ma, "memcpy", d, s, uint64(n)); got != d {
				t.Errorf("memcpy answered %d, want its destination %d", got, d)
			}
			if got := raw(ma, d, n); got != src[:n] {
				t.Errorf("memcpy(_, %q, %d) = %q", src, n, got)
			}
			if ma.mem[d+uint64(n)] != 'Q' {
				t.Errorf("memcpy(_, %q, %d) wrote past n", src, n)
			}
		}
		for _, shift := range []int{1, 3} {
			ma := newMa()
			b := room(ma, len(src)+shift)
			call(ma, "memcpy", b, put(ma, src), uint64(len(src)))
			call(ma, "memmove", b+uint64(shift), b, 5) // destination ABOVE source
			want := []byte(src + strings.Repeat("Q", shift))
			copy(want[shift:], src[:5])
			if got := raw(ma, b, len(want)); got != string(want) {
				t.Errorf("memmove up by %d = %q, want %q", shift, got, string(want))
			}
			ma = newMa()
			b = room(ma, len(src))
			call(ma, "memcpy", b, put(ma, src), uint64(len(src)))
			call(ma, "memmove", b, b+uint64(shift), 5) // destination BELOW source
			want = []byte(src)
			copy(want, src[shift:shift+5])
			if got := raw(ma, b, len(want)); got != string(want) {
				t.Errorf("memmove down by %d = %q, want %q", shift, got, string(want))
			}
		}
		for _, n := range []int{0, 1, 4} {
			ma := newMa()
			d := room(ma, n+1)
			if got := call(ma, "memset", d, 'z', uint64(n)); got != d {
				t.Errorf("memset answered %d, want its destination %d", got, d)
			}
			if got := raw(ma, d, n); got != strings.Repeat("z", n) {
				t.Errorf("memset(_, 'z', %d) = %q", n, got)
			}
			if ma.mem[d+uint64(n)] != 'Q' {
				t.Errorf("memset(_, 'z', %d) wrote past n", n)
			}
		}
		for _, ch := range []byte{'a', 'd', 'z'} {
			for _, n := range []int{0, 3, 8} {
				ma := newMa()
				s := put(ma, src)
				got, want := call(ma, "memchr", s, uint64(ch), uint64(n)), bytes.IndexByte([]byte(src[:n]), ch)
				if want < 0 {
					if got != 0 {
						t.Errorf("memchr(%q, %d, %d) = %d, want null", src, ch, n, got)
					}
				} else if got != s+uint64(want) {
					t.Errorf("memchr(%q, %d, %d) at offset %d, want %d", src, ch, n, got-s, want)
				}
			}
		}
	}

	// ----- atoi / atol: leading whitespace, one sign, digits, then stop.
	for _, s := range []string{"42", "-7", "+13", "  +13xyz", "\t\n 99", "abc", "", "-",
		"0", "007", "12x34", "-987654", "2147483647", "1234"} {
		ma := newMa()
		// The oracle: the longest prefix strconv accepts after the same trimming.
		body := strings.TrimLeft(s, " \t\n\v\f\r")
		i, sign := 0, ""
		if i < len(body) && (body[i] == '+' || body[i] == '-') {
			sign, i = string(body[i]), i+1
		}
		j := i
		for j < len(body) && body[j] >= '0' && body[j] <= '9' {
			j++
		}
		var want int64
		if j > i {
			want, _ = strconv.ParseInt(sign+body[i:j], 10, 64)
		}
		for _, name := range []string{"atoi", "atol"} {
			if got := signedOf(call(ma, name, put(ma, s)), 64); got != want {
				t.Errorf("%s(%q) = %d, want %d", name, s, got, want)
			}
		}
	}

	// ----- strdup: a fresh, independent copy that the caller may write through.
	for _, s := range words {
		ma := newMa()
		src := put(ma, s)
		d := call(ma, "strdup", src)
		if d == 0 || d == src {
			t.Errorf("strdup(%q) = %d, want a fresh non-null address (source %d)", s, d, src)
		}
		if got := cstr(ma, d); got != s {
			t.Errorf("strdup(%q) copied %q", s, got)
		}
		ma.mem[d] = 'Z'
		if len(s) > 0 && ma.mem[src] != s[0] {
			t.Errorf("writing the strdup copy of %q changed the original", s)
		}
	}
}
