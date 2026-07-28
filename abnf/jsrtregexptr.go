package abnf

// The POINTER-based face of the shared regular-expression engine.
//
// Every other caller of jsrtregex.go reaches it through the i64-HANDLE world: a
// compiler grammar emits js_rx* calls whose arguments are indices into the jsrt
// value table. The bash compiler cannot. It emits hand-written IR over a raw i8*
// string model - an arena bump-allocator, rt_strcat, rt_glob - and has no handle
// runtime at all, so `[[ $s =~ re ]]` had no way to reach the engine and the bash
// grammar (rightly) refused to fake it.
//
// This file bridges that, WITHOUT a second matcher: rt_regex_search below runs the
// same rxCompile/rxSearch as Ruby, Python, Kotlin, JavaScript and TypeScript. Only
// the calling convention differs - C strings in the IR machine's flat memory
// instead of handles, and BYTE offsets instead of rune indices.
//
// Registration is additive. The natives register themselves in an init(), and
// llvmmap.go's resolveExtern consults the registry before its own switch, so no
// existing extern is touched and a program that never calls these is unaffected.
//
// STATUS 2026-07-28: NOTHING REACHES THIS ANY MORE. bash-to-llvm-ir.abnf now EMITS
// its ERE engine as IR instead of declaring these two, and resolveExtern is only
// consulted for a function with no blocks - so a defined rt_regex_search shadows
// this one completely. The reason is the whole point of the bridge turning out to
// be its undoing: a Go native has no C symbol, so `-exe` had nothing to link,
// buildExecutable's stubUndefined gave the DECLARATION a `ret i32 0` body, and 0
// means NO MATCH. The bash ratchet passed 289/289 under llvm.Run and 282/289 in the
// clang-built executable, the seven being exactly the =~ assertions that need a
// match or a compile error. Byte-identity could not see it (both engines called
// this same Go code and agreed) and `clang -S -x ir` could not either (the module
// is well-formed; the symbol simply is not there). See tests/clang-check.sh.
//
// The file is kept, unregistered from nothing and costing nothing, because the
// pointer-based calling convention below - C strings in flat memory, BYTE offsets
// rather than rune indices - is the contract any FUTURE raw-IR grammar would want,
// and because the nativeExtern hook it installs is shared machinery. If you reach
// for it, remember what it cannot do: be linked into a native binary.

import (
	"sync"
	"unicode/utf8"
)

// nativePtrExterns is the registry of POINTER-based natives for the raw-IR path.
// Unlike the js_* externs, which are installed per jsrt runtime, these are built
// into the IR interpreter itself: a module compiled by a grammar with no handle
// runtime (bash, batch) runs with ma.externs == nil and only ever reaches
// resolveExtern's own table.
var nativePtrExterns = map[string]func(ma *machine, args []uint64) uint64{}

// nativeExtern is the hook resolveExtern calls before its built-in switch.
func nativeExtern(ma *machine, name string) func(args []uint64) uint64 {
	f, ok := nativePtrExterns[name]
	if !ok {
		return nil
	}
	return func(args []uint64) uint64 { return f(ma, args) }
}

// ----- Memory helpers -----

// rxPtrCStr reads a NUL-terminated string out of the machine's flat memory. A null
// or out-of-range pointer reads as the empty string rather than panicking, because
// an optional argument (the flags) is legitimately passed as null.
func rxPtrCStr(ma *machine, addr uint64) string {
	if addr == 0 || addr >= uint64(len(ma.mem)) {
		return ""
	}
	end := addr
	for end < uint64(len(ma.mem)) && ma.mem[end] != 0 {
		end++
	}
	return string(ma.mem[addr:end])
}

// rxPtrCopyStr writes a NUL-terminated string into fresh machine memory and answers
// its address.
func rxPtrCopyStr(ma *machine, s string) uint64 {
	addr := ma.alloc(uint64(len(s)) + 1)
	copy(ma.mem[addr:], s)
	return addr
}

// rxPtrDecode splits the subject into runes AND the byte offset each rune starts
// at, so a rune index from the engine can be reported back as the BYTE index the
// caller's i8* slicing needs.
//
// Byte offsets, not code points, are the right answer here and this is the one
// entry point where the question is settled rather than deferred: bash is
// byte-oriented, ${BASH_REMATCH[n]} is cut out of a byte buffer, and a rune index
// would silently mis-slice the moment a subject contained a non-ASCII character.
// utf8.DecodeRuneInString reports size 1 for an invalid byte, so the mapping stays
// exact even for a subject that is not valid UTF-8 (each stray byte matches as
// U+FFFD, which is a matching limitation, never an offset one).
//
// Above the BMP this is exact too - Go's []rune holds a whole code point and its
// byte offset is its real one. The UTF-16-versus-rune divergence flagged for the
// handle-based externs does not arise, because nothing here crosses to the JS twin.
func rxPtrDecode(s string) ([]rune, []int) {
	runes := make([]rune, 0, len(s))
	byteAt := make([]int, 0, len(s)+1)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		runes = append(runes, r)
		byteAt = append(byteAt, i)
		i += size
	}
	return runes, append(byteAt, len(s))
}

// ----- The last compile error, for rt_regex_error -----

var (
	rxPtrErrMu sync.Mutex
	rxPtrErr   = map[*machine]string{}
)

func rxPtrSetErr(ma *machine, msg string) {
	rxPtrErrMu.Lock()
	rxPtrErr[ma] = msg
	rxPtrErrMu.Unlock()
}

func rxPtrGetErr(ma *machine) string {
	rxPtrErrMu.Lock()
	defer rxPtrErrMu.Unlock()
	return rxPtrErr[ma]
}

// ----- rt_regex_search -----
//
//	i32 rt_regex_search(i8* pattern, i8* subject, i8* flags, i32* caps, i32 ncaps)
//
// Answers
//	> 0  matched: the number of capture PAIRS, i.e. ngroups + 1, exactly the
//	     ${#BASH_REMATCH[@]} bash reports
//	  0  no match  (bash: [[ ]] status 1)
//	 -1  the pattern did not compile (bash: status 2); rt_regex_error() has the text
//	 -2  the match hit the engine's step cap on a pathological pattern
//
// `caps` is a caller-allocated array of `ncaps` i32 SLOTS (so ncaps/2 pairs).
// Slot 2i holds the BEGIN byte offset of group i and slot 2i+1 its END; group 0 is
// the whole match. A group that did not take part reads -1/-1, which is bash's
// empty ${BASH_REMATCH[i]}. The whole array is filled with -1 first, so a caller
// that ignores the return value can never read a stale offset, and on no-match or
// error every slot is -1. If the match has more pairs than fit, the return value is
// still the true count and only the first ncaps/2 pairs are written - so a caller
// can detect truncation by comparing the answer against ncaps/2.
//
// `flags` is the engine's neutral set - i, s, m, x - or null/empty for none. bash's
// =~ is ERE, so it passes null normally and "i" under `shopt -s nocasematch`. It
// must NOT pass o (JavaScript Annex B octal) or q (Java \Q...\E): without o an
// out-of-range backreference is the "invalid group reference" error, which is the
// right answer for a POSIX ERE.
func rxPtrSearch(ma *machine, args []uint64) uint64 {
	pattern := rxPtrCStr(ma, args[0])
	subject := rxPtrCStr(ma, args[1])
	flags := rxPtrCStr(ma, args[2])
	capsAddr := args[3]
	ncaps := int(int32(uint32(args[4])))

	// Clear first: no caller can read a stale offset, whatever it does with the
	// answer.
	if capsAddr != 0 {
		for i := 0; i < ncaps; i++ {
			ma.store(capsAddr+uint64(4*i), uint64(uint32(0xffffffff)), 4)
		}
	}

	re := rxGet(pattern, flags)
	if !re.ok {
		rxPtrSetErr(ma, re.err)
		return uint64(uint32(0xffffffff)) // -1
	}
	rxPtrSetErr(ma, "")

	text, byteAt := rxPtrDecode(subject)
	m, ok := re.search(text, 0)
	if !ok {
		return uint64(uint32(0xfffffffe)) // -2, step limit
	}
	if m == nil {
		return 0
	}

	pairs := re.ngroups + 1
	room := ncaps / 2
	if capsAddr != 0 {
		for g := 0; g < pairs && g < room; g++ {
			for half := 0; half < 2; half++ {
				slot := 2*g + half
				v := -1
				if slot < len(m.caps) && m.caps[slot] >= 0 {
					v = byteAt[m.caps[slot]]
				}
				ma.store(capsAddr+uint64(4*slot), uint64(uint32(int32(v))), 4)
			}
		}
	}
	return uint64(uint32(int32(pairs)))
}

// ----- rt_regex_error -----
//
//	i8* rt_regex_error()
//
// The text of the last pattern that failed to compile in this program, as a
// NUL-terminated string in machine memory, or a pointer to "" when the last
// rt_regex_search compiled fine. Only meaningful right after a -1. It exists so a
// -1 is actionable: bash prints its own diagnostic and exits 2, and this is the
// message to put in it.
func rxPtrError(ma *machine, args []uint64) uint64 {
	return rxPtrCopyStr(ma, rxPtrGetErr(ma))
}

func init() {
	nativePtrExterns["rt_regex_search"] = rxPtrSearch
	nativePtrExterns["rt_regex_error"] = rxPtrError
}
