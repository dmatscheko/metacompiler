# Tickets

This file is the single list of everything still to fix, and the tracker the master
agent works from ([master-prompt.md](master-prompt.md) says how). The manual —
architecture, building, testing, traps — is
[working-on-this-project.md](working-on-this-project.md).

**Only the master edits this file.** Workers report to the master; the master marks
tickets and adds new ones.

## Conventions

- **IDs are stable.** A ticket keeps its `T-NNN` forever; new tickets take the next
  free number (see the counter below). Never renumber, never reuse. Never cite a
  ticket ID in code — code comments explain code, not history.
- **Status**: `OPEN` · `IN PROGRESS (worker, round N)` · `DONE (<commit>)` ·
  `PARTIAL (<commit>) - <what is left, as a new ticket T-NNN>` ·
  `BLOCKED - <why, what would unblock it>` · `DECLINED - <measurement or reason>`.
  A DONE / DECLINED ticket moves to "Closed" at the end of its section with a
  one-line verdict.
- **Priority**: `P1` wrong answer a user would see, with a real oracle here · `P2`
  wrong answer needing a value-model or floor change, or tooling that makes a gate
  unreliable · `P3` costed design, performance, corpus or cosmetic.
- **Group** = file ownership for parallel work (see the table). Tickets in one group
  go to one worker; `Shared:` names shared files a ticket must touch, which the master
  serialises.
- **Oracle**: what settles the answer. `spec-only` means no toolchain is installed
  here (C#, Dart, Kotlin compiler, PHP): fix only what the specification settles
  unambiguously, and say so in the report.
- **[V]** the defect was reproduced at the commit noted in "Last verified";
  **[U]** inherited, re-probe before assigning. Treat every ticket as a place to
  start probing, never as a specification: tickets have understated their defect
  and named the wrong fix before. Probe the fix as well as the defect.

Next free ID: **T-327** · Last verified: `17e73c7`; round tickets filed since are [V] at the commit that filed them (all [V] tickets were probed
against the oracle at or shortly before this commit).

## Groups and file ownership

| Group | Owns |
|---|---|
| PY | python-*.abnf, lib/python-rt.metajs, abnf/jsrtpy*.go, pystrmethod.go, pyarith.go, tests/python-* |
| RB | ruby-*.abnf, lib/ruby-rt.metajs, abnf/jsrtruby.go, tests/ruby-* |
| JS | js-*.abnf, typescript-*.abnf, lib/js-rt.metajs, abnf/jsrtjsprint.go, jsrtjsbig.go, tests/js-*, tests/typescript-* |
| JV | java-*.abnf, lib/java-rt.metajs, abnf/jsrtjava.go, abnf/jsrtjvm.go, tests/java-* |
| CS | csharp-*.abnf, lib/csharp-rt.metajs, abnf/jsrtcsharp.go, tests/csharp-* |
| KT | kotlin-*.abnf, lib/kotlin-rt.metajs, abnf/jsrtkotlin.go, abnf/jsrtregexkt.go, tests/kotlin-* |
| GO | go-*.abnf, lib/go-rt.metajs, lib/go-constfold.js, abnf/jsrtgolang.go, tests/go-*, tests/imports/ |
| SW | swift-*.abnf, lib/swift-rt.metajs, abnf/jsrtswift.go, tests/swift-* |
| DA | dart-*.abnf, lib/dart-rt.metajs, abnf/jsrtdart.go, tests/dart-* |
| PHP | php-*.abnf, lib/php-rt.metajs, abnf/jsrtphp.go, tests/php-* |
| LUA | lua-*.abnf, lib/lua-rt.metajs, abnf/jsrtlua.go, tests/lua-* |
| SH | bash-*.abnf, batch-*.abnf, lib/bash-rt.c, lib/batch-rt.c, tests/bash-*, tests/batch-* |
| C | c-*.abnf, tests/c-* |
| CORE | metajs-*.abnf, lib/compile-core.js, lib/interp-core.js, lib/runtime.c, lib/runtime.metajs, lib/regex.js, abnf/*.go not listed above (jsrt.go, frozen.go, commonscript.go, trace.go, llvmmap.go ...), main.go, test.sh, tests/*.sh, tools/ |

Shared files every group may need (`abnf/jsrt.go`, `lib/runtime.metajs`,
`lib/interp-core.js`, `lib/runtime.c`, `.vscode/launch.json`, `tests/gates.sh`): a
worker may make a SMALL, additive change there for its own ticket and must say so in
its report; anything structural there is a CORE ticket.

---

# PY — Python

### T-011 python: `d.items()` renders pairs as lists · P2 · PY · OPEN [V]
Same root cause as T-170 (no tuple type). Close together with T-170.

### T-311 python: ordering across kinds does not raise · P1 · PY · IN PROGRESS (PY+PHP+LUA worker, round 6) [V]
`sorted([1, "a"])`, `[1] < ["a"]`, `1 < "a"`: CPython TypeError "'<' not supported
between instances of 'str' and 'int'"; all engines answer.

### T-312 python: `str.split(sep=)`, `str.encode()`, no-argument constructors · P1 · PY · IN PROGRESS (PY+PHP+LUA worker, round 6) [V]
`"bob".split(sep="o")` ignores the keyword (CPython `['b', 'b']`; check every str
method's keywords). `str.encode()` returns the str (`len("é".encode())` 1, CPython 2;
`.encode().decode()` fails). `float()` prints `[object Object]` (interpreter) / `nan`
(compiled), CPython 0.0; `list()` / `tuple()` abort compiled ("'NoneType' object is
not iterable": None and a missing argument are both undefined there). Also `int()` of
non-ASCII digits, and 3.14's "Did you mean" suffix (P3).

---

# RB — Ruby

### T-325 ruby: `$!`, bare undefined calls, `Object#inspect`, Module class · P1 · RB · IN PROGRESS (JS+RB worker, round 6) [V]
`x = (raise "boom" rescue $!)` gives nil in all engines (MRI `#<RuntimeError: boom>`).
A bare `nope(1)` aborts ("unknown name" / "variable not defined"; MRI NoMethodError
"undefined method `nope' for main:Object"). `Object#inspect` calls a user #to_s (MRI 2.6
`#<P:0x...>` for a class defining only to_s). `Kernel.class` / `Comparable.class`
answer Class (MRI Module). `1.method(:zz)` raises NoMethodError for `method` (MRI
NameError "undefined method `zz' for class `Integer'").

### T-326 ruby: Math domain errors, Regexp gaps, `%w` nesting, class reopen · P1 · RB · IN PROGRESS (JS+RB worker, round 6) [V]
`Math.sqrt(-1)`, `Math.log(-1)` answer NaN (MRI Math::DomainError). `Regexp.new("ab",
Regexp::IGNORECASE)`: interpreter "uninitialized constant", compiled `#<Regexp>`.
`%r{a/b}.inspect` does not escape "/" (MRI `/a\/b/`). `%w[a [] b]` (nested brackets)
does not parse. Compiled: reopening a class builds a new class object (earlier class
@ivars lost); a module body's @ivars land on main. A Range/Enumerator NoMethodError
names the Array it delegates to (MRI `1..3:Range`); NoMethodError#name missing.

---

# JS — JavaScript and TypeScript

### T-047 js: a generator body's `this` is wrong in the NATIVE build only · P2 · JS · Shared: runtime.c · OPEN [V]
`function* g(){ yield this }; g().next().value` gives the generator object natively,
`undefined` in the other legs. `js_this` is one global slot in layer 2 and the emitter
arms it with the receiver of `.next()`. Needs a per-coroutine `this` in the floor's
control block (manual 7.8, high risk). Deliberately unasserted today; assert once fixed.

### T-048 js: `with` · P3 · JS · Shared: runtime.c · OPEN [V]
The interpreter can implement it exactly; the compiled engines cannot, because the
floor's scope is not object-backed - snapshot-in/write-back cannot see a property
added inside the body, which would ship a designed-in halves divergence. Needs an
object-backed scope in the floor, or DECLINED with that reason.

### T-049 js: `Symbol.iterator` · P3 · JS + CORE · OPEN [V]
No symbols exist in the value model; `for..of`, `yield*` and `js_iterable` test
structurally for a callable `next` in three engines. Needs a new tag in runtime.c, a
Go type, typeof/strict_eq/keysOf/dict-key arms in all engines, a well-known-symbol
registry, and computed keys in the class/object-literal emitters. Buys user classes
becoming iterable. Cost first; implement or DECLINE with the measurement.

### T-316 js: compiled own-key lists of builtins; Object.getOwnPropertyNames(Math); `"toString" in {}` · P1 · JS · IN PROGRESS (JS+RB worker, round 6) [V]
`Object.keys(JSON)` is "stringify,parse" in run and native (node and interpreters "");
`Object.keys(Object).length` 1 (node 0): the Go twin and layer 2 list those members as
keys. `Object.getOwnPropertyNames(Math)` [] in all legs (node 44 names). `"toString"
in {}` false (node true).

### T-317 js: generator close semantics with finally / a non-done delegate · P2 · JS · IN PROGRESS (JS+RB worker, round 6) [V]
`try{yield 1}finally{return 42}` then `g.return(7)`: node `{value:42,done:true}`, ours
`{value:7}`; `finally{yield 99}`: node `{value:99,done:false}`, ours
`{value:7,done:true}`; a yield* delegate whose return() answers {done:false}: node
yields it and stays in the yield*. No engine can yield during a close (closeBody /
gen_close). All engines.

### T-318 js: `super.toString()` without an own toString · P1 · JS · IN PROGRESS (JS+RB worker, round 6) [V]
In a class with no toString (e.g. extends Object): node "[object Object]", ours
TypeError "not a function". All engines.

### T-319 js: value reads of builtin methods on arrays/strings leak goja's host methods · P1 · JS · IN PROGRESS (JS+RB worker, round 6) [V]
`typeof [1].valueOf`: node "function"; goja-interp "function" (host leak), -frozen /
run / native "undefined" - a live goja-vs-frozen divergence the matrix does not
exercise; `[1].push` read as a value is bound under -frozen/compiled but takes `.call`
receivers under goja; `const f = "ab".toString; f.call([1])` aborts the goja
interpreter with a host error (node TypeError). Close the host leak and give every
leg node's answers.

---

# JV — Java

### T-308 java: astral characters in string literals become U+FFFD · P1 · JV · IN PROGRESS (SW+JV+CS worker, round 6) [V]
`"😀"` (literal or `\uD83D\uDE00`): `(int) s.charAt(0)` prints 65533 in all legs;
java 55357. Compare python T-200 / ruby T-187 (each fixed in its own grammar).

### T-309 java: `super.toString()` on implicit Object; qualified `new java.util.ArrayList` · P1 · JV · IN PROGRESS (SW+JV+CS worker, round 6) [V]
`super.toString()` aborts "unknown super method 'toString'" (java `A@...`); in the
interpreter `super.toString().startsWith(...)` does not parse. `new
java.util.ArrayList<>()` without an import: "unknown class: java.util.ArrayList"
(interpreter).

---

# CS — C#  (no dotnet here: spec-only, ECMA-334 and the .NET reference source)

### T-112 csharp: `Single` exponent-notation threshold is unverified · P3 · CS · BLOCKED - needs dotnet
`flt36`/`flt37` in tests/csharp-test-full.cs assert `1E+07`, `1E+20`,
`3.4028235E+38` for `float.ToString()`. .NET Core 3.0+ prints the shortest round-trip
digits and its switch to exponent notation may not be the fixed window `csFloStr32`
implements (16777216f may print `16777216`). Settle with a real dotnet; until then do
not change the renderer.

### T-310 csharp: the string and char API · P1 · CS · IN PROGRESS (SW+JV+CS worker, round 6) [V spec-only]
`char.IsDigit`/`char.IsLetter` unknown (interpreter); `ch.CompareTo('B')` fails in all
legs (spec -1); `string.Concat`, `new string('A', 3)`, `s.ToCharArray()` fail; the
interpreter has Substring/IndexOf/Equals while layer 2 has ToUpper/ToLower/Trim. Build
one C# string/char surface for all three engines (cf. java's lib/java-strings.js).

---

# KT — Kotlin  (no kotlinc; the real stdlib jar can be driven from java:
`~/.gradle/caches/modules-2/files-2.1/org.jetbrains.kotlin/kotlin-stdlib/2.2.21/*/kotlin-stdlib-2.2.21.jar`)

### T-080 kotlin: `Array`, `Sequence` and a Map's `values` say `List is empty.` · P2 · KT · BLOCKED - on T-081 [V]
`arrayOf<Int>().first()` → `Array is empty.`, `asSequence().first()` → `Sequence is
empty.`, `map.values.first()` → `Collection is empty.`, `Empty array can't be
reduced.`, `Array has more than one element.` (all read from the 2.2.21 jar). An Array,
a Sequence, a values view and a List are one host array: needs the nominal kind of
T-081 (jsArray in abnf/jsrt.go, the floor's array cell). reduceOrNull /
reduceRightOrNull were added in b38a051.
### T-081 boxed values know their KIND but not their declared class · P2 · KT + SW + CORE · OPEN [V]
Two boxed values of one primitive kind and different declared classes cannot be told
apart: kotlin's mixed Float/Double set key, swift's `Int` vs `Int64` (`8 as Int64 is
Int` is true where swiftc says false), kotlin's empty-range `first`, T-080, go's `%!s(uint64=...)` note (T-073) and T-240. One
mechanism likely closes all: a nominal type on the value. Design, cost, then
implement or split.

### T-322 kotlin: decimal `toFloat` rounds through double · P1 · KT · OPEN [V]
`"1.00000005960464477539062500000001".toFloat()` and `"1.0000000596046448".toFloat()`
give 1.0 in all engines; Float.parseFloat (the jar) 1.0000001. Round once to binary32.

---

# GO — Go  (oracle: `go run`)

### T-240 go: a value of a defined non-struct type is the underlying value · P2 · GO · OPEN [V]
`type MyInt int; var v interface{} = MyInt(4); _ = v.(int)` succeeds in all engines; go
panics. Same family: int vs int64, rune vs int32 in type switches/asserts. Value-model:
see T-081.

### T-266 go: pointers, nil typed values and panic values in fmt and panics · P2 · GO · OPEN [V]
A pointer to a struct is the struct: `&T{}.(T)` does not panic (go `interface {} is
*main.T, not main.T`), `%#v` of `&T{}` prints `main.T{...}` (go `&main.T{...}`). A nil
chan/func/pointer operand reads as a nil interface (`%T` prints `<nil>`); `%T` of a
func has no signature; `%T` of `&n` is a bad verb and native prints `&<nil>` where
llvm.Run prints `&5`. An uncaught `panic(T{1})` prints `{1}` (go `panic: (main.T)
0x...`, `panic: main.MyInt(3)`). Value-model; see T-081.

### T-321 go: pointer identity, `f(ys...)` on Println, re-panic suffix, errors.As · P1 · GO · OPEN [V]
`&T{1} == &T{1}` is true in both engines (go false; struct pointers compare by field),
so `errors.Is(err, find(0))` is wrong; `fmt.Println(ys...)` dies "TypeError: Not a
function" in the interpreter (check the compiler); an uncaught re-panic lacks go
1.23+'s " [recovered, repanicked]" suffix; errors.As is not modelled (needs a nil
pointer's static type). Related value-model limit: T-266.

---

# SW — Swift  (oracle: swiftc 6.1.2)

### T-300 swift: unqualified `.none` builds an enum case · P1 · SW · IN PROGRESS (SW+JV+CS worker, round 6) [V]
`let n: Int? = .none; print(n == nil)`: swiftc `true`, ours `false`. Fix needs to know
no user enum declares `none` (or resolve by the declared Optional type).

### T-301 swift: in-place mutation of an observed variable does not fire didSet · P1 · SW · IN PROGRESS (SW+JV+CS worker, round 6) [V]
`var a = [1] { didSet { print("d", a.count) } }; a.append(2); a[0] = 5`: swiftc
prints `d 2` twice; we print nothing. Only `=` and compound assignment are observed.

### T-302 swift: `print(x as Any)` on an Optional; nested Unicode.Scalar · P2 · SW · IN PROGRESS (SW+JV+CS worker, round 6) [V]
`print(Optional(7) as Any)` prints `7` (swiftc `Optional(7)`: the Optional-box
limit); a Unicode.Scalar in a collection prints `"é"` (swiftc `"\u{00E9}"`).

---

# DA — Dart  (no dart here: spec-only)

### T-323 dart: String remainder; layer 2 indexes strings by code point · P1 · DA · OPEN [V spec-only]
dart layer 2 indexes by code point (dtStrAt / dtStrRange) while the twin and the
interpreter use UTF-16 units: `"a\u{1F600}b".substring(3)` is "b" interpreted and
under llvm.Run, "" natively - a halves divergence. Left from T-282: `runes` (an
Iterable), allMatches/matchAsPrefix/splitMapJoin/replaceAllMapped (need RegExp/Match),
static String.fromCharCode(s) (the interpreter has no `String` name).

---

# PHP — PHP  (no php here: spec-only, php.net / php-src behaviour)

### T-313 php: partial application of a method declared with different arities · P1 · PHP · IN PROGRESS (PY+PHP+LUA worker, round 6) [V spec-only]
Compiler only: with `class A{function m($x,$y)}`, `class B{function m()}`, `class
C{function m()}`, `($a->m(?,2))(1)` prints "m"; spec "m12". Also: the bare `$GLOBALS`
array (iteration, var_dump) is unsupported.

---

# LUA — Lua  (oracle: lua 5.5 binary, suite is 5.4)

### T-314 lua: `next()` is a stub compiled; `tostring`/`tonumber`/`type`/`table.concat` missing · P1 · LUA · IN PROGRESS (PY+PHP+LUA worker, round 6) [V]
Compiled `next()` covers only the array part and returns only the key: `next({a=1})`
nil (lua `a 1`), `next({10,20})` 1 (lua `1 10`). `tostring`, `tonumber`, `type`,
`table.concat` are missing in all engines - sweep the whole base/table/math library
against lua 5.5 while there.

### T-315 lua: print writes invalid UTF-8 bytes as code points · P3 · LUA · IN PROGRESS (PY+PHP+LUA worker, round 6) [V]
lua writes the raw bytes. (math.log/log10/sin/atan last-digit differences vs C libm:
see T-293.)

---

# SH — bash / batch  (oracle: /opt/homebrew/bin/bash 5.3; no cmd.exe)

### T-320 bash: positional-parameter operator forms · P1 · SH · OPEN [V]
Interpreter: `set -- "a b" c "" d; printf '<%s>' "${@:2}"` gives `< bcd>` (bash
`<c><><d>`); same for "${*:2}", ${*@Q}/${@@Q} (empty), ${*:-x}/${@:-x} (answer x),
${*/c/C}, ${*#c}. "${!Z@}" one field (bash <Zq><Zw>); the compiler does not parse
`${!Z@}`; ${!h*} for an associative array h is empty (bash h). The default word of
${x:-$*} / ${x:-$@} is flattened with spaces under a null IFS (bash keeps fields).
(bash 5.3 leaks \x01 into `${a[*]%d}` under a null IFS - a bash bug, not mirrored.)

---

# C — C subset

---

# CORE — MetaJS, the engines, shared runtime, tooling

### T-154 replay-based generators repeat side effects in the interpreters · P2 · CORE + JS + PY · OPEN [V]
js, typescript and python INTERPRETER halves drive generators and async functions by
replaying the body, so effects before a `yield`/`await` run again: js
`take(inf(), 3)` prints `t0 t2 t5` (node `t0 t1 t2`); an async-generator trace is
garbage. The compiled halves are correct. Manual 7.14 explains the mechanism. Find a
real suspension for the interpreter halves that works under goja and -frozen, cost it,
implement or write the costed design here and DECLINE.

### T-158 frame elimination on arithmetic predicates · P3 · CORE + PY + RB · OPEN [V]
`rtTag`'s −13.9% on python came from removing ONE scope frame from ONE predicate.
The same `!pyIsObj(v)`-then-field-test shape sits at `pyIsFlo`, `pyIsNumeric`,
`pyInstClassOf`, `rbUserObj`, `rbIsClassObj`. Each needs its own call-site sizing and
a measurement against a dead-code matched control (tests/bench.sh header). Never
introduce a delegating wrapper on a hot path: that cost python +41% once.

### T-284 full case mapping (`"ß".upper()` is `SS`) · P2 · CORE · IN PROGRESS (CORE worker, round 5) [V]
Hosts, Go twins and the floor implement the simple per-rune mapping. Full mapping
(SpecialCasing + Final_Sigma) natively in C, then per language: python/ruby/swift/
kotlin/java/js want full, C#/Go want simple. metajs cm03-cm05 and cm08 assert the
simple answer node contradicts - correct them with the change.

### T-285 -frozen js memory, remainder · P3 · CORE · IN PROGRESS (CORE worker, round 5) [V]
js-test-full.js part 1 under -frozen still needs 5.3 GB: a released handle-table slot
is never reused (by design, so a failed analysis fails loudly), and InGen rescans the
whole prefix per call. Decide slot reuse, or make InGen incremental.

### T-286 -callgraph: a definition's line in the other languages · P3 · CORE · IN PROGRESS (CORE worker, round 5) [V]
js, ts, metajs, java, kotlin, go, lua and the rest still name a definition by its
body's line: their markers are `up.pos`, the END of the node. Needs the engine to
expose a tag's start position (`up.start`, past whitespace) in compiler.go and
frozen.go; that also changes -trace/-why line attribution.

### T-287 layer 2 `rtABigDivMag` binary-searches each quotient limb · P3 · CORE · IN PROGRESS (CORE worker, round 5) [V]
The interpreters now estimate the limb (magQuotEst); runtime.metajs's native twin
still searches. Port the estimate and bench.

### T-288 js-to-llvm-ir under goja turns a lone `\uD800` literal into U+FFFD · P1 · CORE · IN PROGRESS (CORE worker, round 5) [V]
`u === "\uD800A\uDC00"` is false under goja and natively, true under -frozen: the
literal is handed to Go as a UTF-8 string. A goja-vs-frozen divergence in the
emitted module. Carry lone surrogates through the goja bridge (WTF-8).

### T-289 native `d_asin` loses precision near 1 · P1 · CORE · Shared: runtime.c · IN PROGRESS (CORE worker, round 5) [V]
`1 - x*x` cancels (Go fuses it into an FMA on arm64): native math.asin is off by up to
907 ulps near 1; run vs native differ on 95/150 rows. `(1-x)*(1+x)` avoids it. Repro:
python `math.asin(x/(1+x))` for x = 37.9365... gives 1.3436684803445726, CPython
...729. Check d_acos and the other d_* for the same shape.

### T-293 correctly rounded transcendentals · P3 · CORE · OPEN [V]
CPython (Apple libm) and MRI are correctly rounded on exp/log/pow and nearly so on
sin/cos/atan/atan2; our Go-port legs are 1-9 ulps off on many rows (python T-265,
ruby T-229 declined for this reason). Design: CORE-MATH-style double-double with a
Ziv rounding test, one MetaJS module (interpreter + layer 2) + Go port + runtime.c
d_*. Cost it; implement or DECLINE.

### T-324 layer 2 `Array.sort` with a comparator fails · P1 · CORE · OPEN [V]
In a lib/*-rt.metajs, `xs.sort(function(a, b) {...})` dies natively "call of a non
function value: undefined". Fix the frozen/layer-2 builtin, assert it in the metajs
ratchet. Also: `gofmt -l abnf/` lists jsrtswift.go (a doubled blank line) and the
generated jsagrammar.go - make the generator gofmt-clean or exclude it knowingly.

### T-170 a real tuple type for Python · P3 · PY + CORE · OPEN [V]
`set_member` (runtime.c) accepts a named key only on tag 6; on tag 5 (array) it
demands a numeric index, so marking an array as a tuple is not representable natively
and a tuple must become a tag-6 BOX (copy the python FLOAT box, not the tag-14 floor
box - manual 7.6). Atomic: any subset is a `--cross` divergence. Sites: python
interpreter 17 `Array.isArray` + 49 tuple refs, python-rt.metajs 42 `pyIsArr`, python
compiler 54 tuple refs, jsrt.go `*jsArray`. Unblocks T-011, `%`-formatting arity (`"%d %d" % (1,2,3)` must say "not all arguments converted during string formatting"; `"%s" % [1, 2]` must stay `[1, 2]`),
`BaseException.__str__`, hashability as a real rule.

### T-171 moving floor bodies into layer 2 · P3 · CORE · OPEN [U]
The floor→layer-2 upcall costs 120 ns; a nine-iteration binary search over boxed
arrays costs 3.2 µs. `case_map` moved whole, passed every gate and was reverted at
15×/135×/2.0×. `d_pow` and its family cannot move (manual 7.7). The direction worth
taking is the VARIADIC PROLOGUE (`scope_get` and `ar_block` repacking args dominate
`sample` on the spread-heaviest go program).

### T-172 a catchable `GeneratorExit` · P3 · CORE + PY · OPEN [V]
`js_try`'s `pending != GEN_EXIT` guard is shared with fifteen languages (where a close
is a return completion and must not be catchable); `GEN_EXIT` has no Python class, so
`except Exception` (must not catch) and bare `except:` (must) cannot be told apart;
the floor needs `gen_close(g, sentinel)` reached through the tag-15 `return` member,
which has nowhere to pass one. New extern → three engines → plus coro-poc.

### T-173 coroutines as CPS in the emitter · P3 · CORE · OPEN [U]
Today: pthread + condvar, 4.1 µs per `next()` natively vs 0.75 µs under llvm.Run,
~18.8 KB per live suspended generator, and an abandoned generator parks its thread
forever. `swapcontext` is dead (deprecated on darwin; no `#include` in runtime.c).
The extern surface would stay identical.

### T-181 corpus: other languages · P3 · per language · OPEN [V]
batch 0 of 2 wine files parse; swift 80% (an 18-file lexical tail: suppressed
conformances, `@isolated(any)` in a type, `borrowing` as a bare type name, raw
multi-line strings with 3+ pound levels, parameter packs, form feed and NBSP); dart
96.6% (~53 of the remaining 117 are multitests whose surviving `//#` variants are
labelled syntax errors); php / csharp / java / c long tails. python 92.9% and ruby
98.1% are at their hard caps. Unicode identifier tables for swift/java/kotlin/csharp/
python must each come from that language's own toolchain or spec (python3 is Unicode
16.0, node 24 is 17.0).

### T-184 dart: `IndexError` vs `RangeError` text for `[1,2,3][7]` · P3 · DA · BLOCKED - needs a Dart VM to settle [U spec-only]
A reviewer argued the VM prints `RangeError (index): Invalid value: Not in inclusive
range 0..2: 7` (a plain RangeError); the current engines print the IndexError form
`RangeError (index): Index out of range: index should be less than 3: 7`, which is
believed to be the modern VM's. Leave as is unless a real dart run settles it.

---

# Closed

(The master moves DONE / DECLINED tickets here, one line each:
`T-NNN <title> — DONE <commit>` or `— DECLINED: <reason>`.)

T-001 python: integer division and modulo by zero never raise — DONE 28abe5b
T-002 python: `str.title()` treats `ß` as uncased — DONE 28abe5b
T-003 python: missing dict/set members — DONE 28abe5b
T-004 python: `except E as w` does not unbind `w` — DONE 28abe5b
T-005 python: an uncaught KeyError renders differently in the two halves — DONE 28abe5b
T-006 python: `__mro__` lacks `object` — DONE 28abe5b
T-007 python: `None.k = v` is not catchable — DONE 28abe5b
T-008 python: native `float(2**80)` is `nan` — DONE b46dc62
T-009 python: `%d` / `{:d}` on a non-integer operand is a three-way divergence — DONE b46dc62
T-010 python: an abandoned generator never runs its `finally` — DECLINED: CPython runs the finally at refcount drop (`del g` prints before the next line); that timing needs refcounting, which manual 7.8 rejects (cycles); a run-at-exit hook would print in the wrong place
T-183 python: native float formatting of huge bignums and `%`-arity — PARTIAL b46dc62 (%-arity/syntax errors catchable, exact %f of a bigint natively); rest is T-203
T-040 js/ts: identifier `\u` escapes are not resolved in the interpreter halves — DONE c1824e2
T-041 js: spreading a generator answers an empty array — PARTIAL 1e68549 (spread and array destructuring drain next()); `new Set(g())`/`Array.from(g())` are T-205
T-042 js: `Math.pow(±1, NaN/±Infinity)` gives 1 — DONE c191cbd
T-043 js/ts: `JSON.stringify` / `JSON.parse` are missing — DONE 7eb22ee
T-044 js: a derived class whose `super()` returns an object does not rebind `this` — DONE f0162a4
T-045 js: Error own-property introspection — DONE e6d1ab8
T-046 js: "is not a function" names the callee without node's spacing — PARTIAL f23e9e8 (V8 spacing/parens/folding via js-callee.js); word operators are T-209
T-050 js: sloppy-mode primitive `this` is not boxed — DECLINED: needs Number/String/Boolean wrapper objects (ToPrimitive, ===, typeof, member dispatch) in all four engines; out of proportion for sloppy-mode `this` boxing
T-180 corpus: JS reject side — PARTIAL 7e7be24 (reject 703→711, must-parse unchanged 2623); rest is T-210
T-192 java: the interpreter prints some doubles with too many digits — DONE 1bead91
T-193 java: overloads are chosen by argument count only — DONE eb6cbac
T-194 java: missing standard-library pieces — PARTIAL 26eedb1 (all members added; Character exact through U+024F); the rest is T-212
T-195 java: typed functional-interface lambdas do not convert their parameters — DONE 65d2b1a
T-196 java: `Math.exp(1)` is one ulp off — DONE 1c463b4 (the ticket had the values swapped: java prints 2.7182818284590455)
T-197 java: `Double.parseDouble` does not read hex significands — DONE 72bb39a
T-199 python: compiled complex values print as `[object Object]` — DONE 432e5a1
T-200 python: a string literal holding an astral character — DONE a9aadb7
T-201 python: setting an attribute on a function aborts — DONE 45198e8
T-202 python: class repr lacks the module prefix — DONE f6614d4
T-204 python: `None.k = 1` does not parse — DONE 11fbd8b a9e8807
T-203 python: int/int true division with a bigint operand rounds twice — PARTIAL fe88574 (the division is rounded once); the %-arity half waits on T-170 (noted there)
T-185 ruby: a negative numeric literal binds the sign before a method call — DONE 9d2fd6b
T-186 ruby: `y = x -1` with a local `x` is read as a command call — DONE 9d2fd6b (NotLocalCmd heuristic, see T-231)
T-187 ruby: `"\u{1F600}".length` is a halves divergence — DONE 34fc538
T-188 ruby: `defined?(3 + 3)` — DONE 5e96dc4
T-189 ruby: a top-level local is visible inside a `def` — DONE 8d6bda9
T-190 ruby: top-level `@ivar` aborts — DONE 2f25333
T-191 ruby: Rational/Complex exactness (rest of T-030) — DONE c343864 ca0f20d
T-090 swift: optional dictionary lookups trap on a missing key — DONE dfe498e
T-091 swift: `x as? Double ?? -1` does not parse — DONE dfe498e
T-092 swift: `Dictionary.first` answers a key, not a `(key, value)` tuple — DONE dfe498e
T-093 swift: remaining write-adoption cases — DONE dfe498e
T-094 swift: an extension on a builtin receiver — DONE dfe498e
T-095 swift: an extension-only member through a protocol-typed variable — DONE f459804 (receivers whose declared type the walk knows; call results / property chains stay dynamic, T-237)
T-070 go: the built-in `error` interface is not modelled — DONE 64995a2
T-071 go: a scalar type assertion does not panic on mismatch — DONE 64995a2 (int vs int64, rune vs int32, defined non-struct types stay indistinguishable: T-240)
T-072 go: an unrecovered native integer divide by zero dies through the C floor — DONE 5aeb8d0
T-073 go: fmt bad-verb notes for composites and wide ints — PARTIAL f09061c (composites walked per element); the uint64=/int64= note names wait on T-081
T-074 go: fmt verbs `%f %g %e %x %o %b` (and widths/flags) — DONE f09061c
T-182 go: `tests/go-test-widen.go` asserts `a.(int)` on a non-interface `a` — DONE c887ae4
T-120 php: an undefined `$variable` aborts — DONE dfacc00 (new floor primitive js_scope_own; no warning text is printed)
T-121 php: variable scope leaks into every function — PARTIAL dfacc00 (frame-local variables, global, use, closures); `$GLOBALS` is T-242
T-122 php: `empty($undef)` aborts; `$u[0] = 1` does not vivify — DONE 43216af
T-123 php: top-level `label:`/`goto` or a bare `{ … }` block aborts — DONE 6a92762 (no compile-core change needed)
T-130 lua: no `error` / `pcall` / `xpcall` — DONE d481647 867a60c 4a6a09d f7bc81a c51b379
T-131 lua: a subnormal float key read back through `pairs()` compares unequal — DONE 6575958 (every numeric key, not only subnormals)
T-205 js: `Array.from`, `Set` and iterable constructors — DONE 2cb5e1d (js only; TS is T-250)
T-206 js: `for (x in Math)` enumerates 43 host members compiled — DONE ffef117
T-207 ts: a dead second `function main()` in tests/typescript-test-full.ts — DONE 734e775
T-208 ts: parameter properties vs field initializers order differs between halves — DONE baf5002
T-209 js: "is not a function" glues word operators — DONE 2de9757
T-210 corpus: JS reject side, remainder — DONE 85387cd (reject 711→719, must-parse unchanged; `1++`, `for(([a]) of 0)` x4, `for((1+1) in l)`, `\u0000` DECLINED: each is inseparable by shape from an early/ must-parse twin)
T-212 java: Character classification beyond U+024F is approximate — DONE 637d1d4
T-213 java: StrictMath, `new String(...)`, ArrayList in the interpreter — DONE 1bd1b2b 7c52f99 (String identity is T-256)
T-214 java: overload resolution uses the argument's runtime class — DONE 1358498 e22bec9
T-215 java: duplicate assertion IDs in tests/java-test-full.java — DONE 58998c6
T-216 java: transcendentals other than exp vs java — DONE 7c52f99 04b091c (Math.sin/cos answer StrictMath: HotSpot's aarch64 intrinsic differs from fdlibm in the last place on ~3.5% of arguments and is not reproducible - DECLINED residual)
T-217 python: `range()` ignores its step and prints as a list — PARTIAL 8ad904e (step, errors); the range type is T-259
T-218 python: `x = None; x[0] = 1` is not a catchable TypeError — DONE e2c6c0b
T-219 python: `f().k += 1` evaluates `f()` twice — DONE eb01ea3
T-220 python: astral characters in indexing, iteration and find — DONE 9197d07 (UTF-16 kept, positions converted; js_pyidx)
T-221 python: `\N{...}` outside a 14-entry table silently becomes "" — DONE 901d9ef (unknown name is a SyntaxError; full table DECLINED: 34,137 names, 1.09 MB per grammar)
T-222 python: complex numbers - abs, infinities, general powers, division — DONE 7f150bf (software FMA in MetaJS; non-integral powers inherit the libm last-digit gap, T-265)
T-239 go: panic texts name user types without `main.` — DONE 13a26a4
T-241 go: fmt `%T`, `%p`, `%#v`, `%!(EXTRA ...)`, printability — DONE c907fc9 (pointer/nil-chan/func-signature limits are T-266)
T-233 swift: a Dictionary as a Sequence — DONE 235b31a
T-234 swift: `hasPrefix` / `contains` on String abort compiled — DONE 235b31a
T-235 swift: `Optional(2.5)` and `Optional.map` — DONE bf62c55 (`Optional<Int>(5)` is T-270)
T-236 swift: a bare trailing closure call in expression position, compiler only — DONE 07442bf
T-237 swift: extension-only members through a protocol type with no declared receiver — DONE bf62c55 (method-call results stay dynamic)
T-031 ruby: `respond_to?` on non-objects, `method(:x)` on block-taking names — DONE 132864d (shared table languages/lib/ruby-respond.js + Go copy kept equal by a go test; include_all is T-277)
T-223 ruby: `Integer#[]` (bit reference) answers nil — DONE 36b888d
T-224 ruby: `(m -1)` inside parentheses is a subtraction — DONE 49a8f5a
T-225 ruby: top-level `p self` — DONE f0ec96f
T-226 ruby: the interpreter ignores singleton `to_s`/`inspect`; no `instance_variable_set` — DONE 72978f1
T-227 ruby: `sym.length` aborts compiled — DONE 06fc50c
T-228 ruby: top-level locals and top-level defs share one namespace — DONE 00b5a45
T-229 ruby: `Math.atan2` is one ulp from MRI — DECLINED: MRI calls Apple libm, whose atan2 differs from Go math.Atan2 on 26%, from fdlibm on 9.4%, from the correctly rounded value on 1.6% of random pairs - no portable algorithm reproduces it; recorded at rubyMathCall
T-231 ruby: `NotLocalCmd` reads scopes from indentation — DECLINED: a guard decides the parse before any tag runs and must stay stateless, so a walk-time table cannot inform it; limit and reproduction recorded in both grammars
T-232 ruby: `defined?` residual gaps — DONE 25e257b
T-140 bash: unquoted `${a[@]@A}` splits into 3 words — DONE 11663e6 (every unquoted field list is re-split)
T-150 c: `(float)x` is a no-op — DONE 1209dab
T-198 kotlin: `String.toDouble()` accepts a numeric prefix — DONE b38a051
T-100 dart: an extension on `int` — DONE bc2c547 (spec-only)
T-101 dart: function-type subtyping — DONE bc2c547 (spec-only)
T-102 dart: chained `x as T is T` — DONE bc2c547 (spec: one typeTest/typeCast per relationalExpression; checks rewritten with parentheses)
T-151 MetaJS: the lexer reads `"\f"` / `"\v"` as letters — DONE 142cd62 (metajs-lint string-escape rule removed)
T-152 -frozen: `indexOf(x, from)` ignores `from` — DONE eb884bb
T-153 goja-vs-frozen case mapping is not pinned — PARTIAL 2bd797d (lone surrogates; goja vs -frozen pinned on every BMP code point); full mapping is T-284
T-155 -frozen needs ~12 GB for js ratchet part 1 — PARTIAL bafe930 (15.5 GB → 5.3 GB); rest is T-285
T-156 `-trace` output is not reproducible run to run — DONE 162be32
T-157 `-callgraph` names a function by the minimum marker in its body — PARTIAL ca0ee0e (python, ruby, php); rest is T-286
T-159 `rtIsNil` could serve js, kotlin, php — DONE 7a63320 (kotlin, php); js DECLINED: importing runtime.metajs costs js-rt.ll +16k lines, binary +12%, bench +0.73% to save four lines
T-211 the C floor's case table is an older Unicode — DONE a366e01 (Go unicode.CaseRanges, Unicode 17)
T-230 -frozen big-integer arithmetic costs millions of steps — DONE 997ee54 (interpreters' magDivMod; layer 2 rtABigDivMag is T-287)
T-238 tests/coro-poc/build.sh fails on main — DONE 1e843dc
T-260 python: `"a" + 1` concatenates — DONE 414fb7d
T-261 python: `print(..., end=, sep=)` — DONE 7830d84
T-262 python: `dict(enumerate(s))` — DONE 1b911f8
T-263 python: TypeErrors for objects that are not iterable / callable / sized — DONE d64ca37
T-264 python: `b"\N{BULLET}"` — DONE eb828fd
T-259 python: `range` is a materialised list — DECLINED: a new value kind through 53 pyIsArr + 32 Array.isArray sites and the shared *jsArray paths, to fix only repr/==/type; unblock with a value-model decision for non-list sequences (tuple, bytes, range together)
T-265 python: math functions and complex powers vs Apple libm last digit — DECLINED: Apple libm is correctly rounded on 780/780 rows of exp/log/pow (our Go-port legs: pow 44/150) - matching it means correctly-rounded functions (CORE-MATH style), not a port; T-293 tracks that design. Overflow/domain errors fixed in 4f8b86e
T-250 ts: no Set, Map, Array.from — DONE 107c693
T-251 js: `Math.hasOwnProperty("abs")` — DONE ab15b93
T-252 js: a class-instance iterator's `return()` is not called compiled — DONE 02cb79d
T-253 ts: `delete (Math as any).sqrt` aborts in the interpreter — DONE b1b9f75
T-254 js: "is not a function" for a comma/arrow callee — DONE 4cfd788 (ASI-separated and keyword-statement bodies declined)
T-269 swift: a top-level observed var does not fire — DONE 7425564
T-270 swift: generic-argument call syntax — DONE 6e99635 (with a Set type in all engines)
T-271 swift: missing String/Array/Dictionary members — DONE c54bba9 (String.Index prints as an Int offset; no ArraySlice type)
T-272 swift: an untyped closure parameter is not inferred from the return type — DONE 33151d0
T-273 ruby: dispatch gaps that differ between the legs — DONE 20b991d (154-row four-leg probe; 8 residual rows filed)
T-274 ruby: Boolean `&`, `|`, `^` in the interpreter — DONE b793cb4
T-275 ruby: a class's singleton `to_s` is ignored by rendering — DONE ce5b7d6
T-276 ruby: compiled instances carry an `args` slot; no `instance_variables` — DONE 73f5b30 (class objects: T-307)
T-277 ruby: `respond_to?(name, true)` ignores include_all — DONE da56e04 (visibility both ways)
T-255 java: `instanceof` an interface — DONE d06ef17
T-257 java: missing String API and core classes — DONE f7ed430 (java +1.78% paired, inside the gate; one-to-many case mapping is T-284)
T-258 java: overload residuals — DONE 4ad7079 (a candidate over a type variable keeps the runtime pick, deliberately)
T-256 java: String identity — DECLINED: needs a reference-string box in the JVM value model - ~60 java sites plus the shared floor (strict_eq, js_jadd, printing, dict keys, regex) and an unwrap on every string operation
T-299 csharp: `(int)t[1]` on a string is 0; -frozen aborts — DONE c254d07 (spec-only)
T-111 csharp: derived field initializers run after the base constructor — DONE 6d84bf8 (spec-only)
T-110 csharp: overload resolution is not "most specific" — DONE d88e042 (spec-only)
T-290 python: `sorted`/`min`/`max` with `key=` / `reverse=` — DONE 0a262d6 (lists order element-wise in every engine)
T-291 python: `sum()` of floats, `math.cbrt`, `int(x, base=)` keywords — DONE c178603 (math.cbrt within 1 ulp: macOS libm cbrt is not correctly rounded; see T-293)
T-292 python: unpacking with the wrong count — DONE 9e9b395
T-294 python: subclasses of builtins; `bytes.decode` — DONE 26691f0
T-242 php: `$GLOBALS` is not modelled — DONE 5372fe2 (spec-only; `$GLOBALS[e]`, not the bare array)
T-243 php: a function declared inside a function is not global — DONE b604c01 (spec-only)
T-244 php: an arrow fn captures a reference variable's cell — DONE 702f36c (spec-only)
T-245 lua: the string library lacks `find`, `gsub` (and check `match`, `gmatch`, `format`, `byte`, `char`, `reverse`) — DONE 2095ea9 5cc9efb (lib/lua-strlib.js, a port of lstrlib.c)
T-246 lua: errors lack lua 5.5's variable-name suffix — DONE 73cffca 65efcb2 bd6a989
T-247 lua: "bad argument" wording and argument checks — DONE 39f7fed
T-248 lua: smaller error-semantics gaps — DONE 7842c78
T-249 lua: numeric table keys are very slow natively — DONE 0d9b10d (-42% instructions on the ticket loop)
T-295 js: `yield*` answers the outer `return()` value — DONE 73c2117 (a delegate answering {done:false} at close is T-317)
T-296 js: Object.prototype methods on objects — DONE 950df0f
T-297 js: `class A extends Object {}` aborts the interpreter — DONE 73edcce
T-298 ts: a stray "END SECTIONS" marker at tests/typescript-test-full.ts ~1065 — DONE 3c9563d
T-267 go: `panic(any(nil))`, `any`, `fmt.Errorf` — DONE 5e648df (errors.As not modelled: T-321)
T-268 go: `gofmt -l abnf/` lists abnf/jsrtgolang.go — DONE bb3637c
T-278 bash: an assignment whose value starts with `=` — DONE 9438d8b
T-279 bash: unquoted `$*` / `${a[*]}` under a null IFS — DONE 0588fd9
T-280 bash: `tests/probe.sh bash` - llvm.Run leg has an extra trailing blank line — DONE 15bfa80
T-281 c: `long` → `float` rounds twice — DONE 7c300b5
T-283 kotlin: `toInt` and Unicode digits; hex-float precision — DONE fcc558b
T-282 dart: the String member surface is tiny — PARTIAL 06500be (spec-only; the common String members); rest is T-323
T-303 ruby: `#hash` / `#object_id` are constant 0 — DONE 6af52ff (String/Rational/Complex share an id per value: no identity in the model)
T-304 ruby: `Integer#lcm`, `#abs2`, `#conj`, `#real?` — DONE 77d835c
T-305 ruby: an unknown method aborts instead of raising NoMethodError — DONE 537185e (a bare `nope(1)` is T-325)
T-306 ruby: `p %w[a b].map(...)` and `p /x/.class` do not parse — DONE bec6cae
T-307 ruby: `C.instance_variables` on a class object — DONE 59ae38d (class reopen / module bodies: T-326)
T-020 ruby: `String#to_i` is a three-way divergence — DONE 4102cd8
T-021 ruby: `print`/`puts` emit earlier arguments before a later one raises — DONE 8c8066e
T-022 ruby: `defined?` kind classification residues — DONE 7fc500f
T-023 ruby: `{}.fetch(:a)`'s KeyError is not catchable — DONE 4eb2bc8
T-024 ruby: `NotImplementedError` sits under StandardError — DONE 4eb2bc8
T-025 ruby: the compiled halves lack `IOError` and `LocalJumpError` — DONE 4eb2bc8
T-026 ruby: interpreter lacks `Float#div`, `Float#divmod`, `Float#rationalize` — DONE 1a28284
T-027 ruby: `"\u{48 49}"` gives `H 49}` — DONE 4f5595d
T-028 ruby: parse gaps — DONE 0bdde28
T-029 ruby: `Kernel#Integer(str)` is imprecise past 2^53 — DONE 4102cd8
T-030 ruby: Rational/Complex exactness — PARTIAL 9c5d75c (Complex raises/conjugate/rect/polar/arg/abs done); rest is T-191
T-060 java: assignment conversion only happens at declarations — DONE a11485b
T-061 java: field defaults, Double/Float constants, Math.sqrt — DONE 9723535
T-062 java: `Integer.parseInt`'s NumberFormatException is not catchable compiled — DONE f2a0034

# Settled — do not re-open

**~196 items, one section, grouped by VERDICT and by nothing else** — no dates, no
rounds, no order, no commit archaeology, because none of it changes what to do
next. Each line is a verdict plus the shortest reason that stops someone
re-opening it. The detail is in git; the durable lessons are in the manual
(ch. 4 and 7).

**DECLINED after measurement — each is a number the next person should not have to
re-derive.** php's `isset`/`$this`
scope probe into layer 2 (**+0.77%, slower in 9 of 9 paired draws**, ~490 instructions
per probe call: unlike kotlin's, php's probe has NO miss arm layer 2 answers better,
so it is all hot arm) · swift's and dart's `sw_kget`/`sw_kset` into layer 2
(**+81.19% and +68.97%, 9 of 9** — `mod.swift`, a bare `while` over two locals, emits
4 `sw_kget` + 2 `sw_kset` PER ITERATION) · python's `py_setvar` containment walk (the
premise does not survive: the emitter already knows those names statically via
`pyInnerNames`, and the floor already owns the name so layer 2 cannot re-use it) ·
a shared `.abnf` module for the three emitter closures (**impossible, not merely
unwanted: an `:include`d file's `:startScript` is SILENTLY DISCARDED**, since
`GetStartScript` takes the first and includes are appended after the host's — there is
a warning for it now) · `rtTag` in its DELEGATING spelling (**+10.60%, slower 9/9** —
`120ba0f`'s regression re-run; the INLINED spelling is −13.90% and shipped) ·
`import`/`export` refused in a Script (+6 reject files, **−3 must-parse**, including
the js ratchet itself) · the goal symbol for TypeScript (the corpus has ZERO
`.mts`/`.module.ts` files and `tsc` decides module-ness from CONTENT) · ID_Start
tables for swift/java/kotlin/csharp/python (**each language's identifier set really is
different, and two toolchains proved it in both directions**) · swift's `stride(...)`
(absent from the subset, and its `for-in` is a shape already asserted) · making a
named function expression's inner name immutable (needs a const flag on the floor's
scope, or a scope-walking extern on every plain-name assignment, measured at
+21%..+43% elsewhere on that path).

**DECLINED after measurement.** matching Apple libm's last digit (ruby atan2: no portable algorithm reproduces it, 1.6% of pairs differ even from the correctly rounded value) · js sloppy-mode primitive `this` boxing (needs wrapper objects in every engine) · a generator's `finally` at collection (python needs refcount timing; no finalizers, manual 7.8) · `i in arr` after `delete` · iterator close on
`throw` (a depth stack cannot survive `for await`) · a catchable unknown-name error
in java/go/swift/csharp/metajs (COMPILE-time in each real toolchain) · kotlin
`:script` → `!"token"` · `case_map` into layer 2 (135×) · retiring the Go twin
(`llvm.Run` structurally cannot execute the C floor, manual 7.11) · dart `x as T`
as a throwing cast · an mtime skip for the generators (a `git checkout` would make
a stale `.ll` report CLEAN) · swiftc's **exit 133** on a trap (libc buffering, not
a language property, and our interpreter legs run *inside* `mec`, so at most 1 of 3
legs could match — widening the gap) · kotlin's unattributed **+2.28%** (five
controls exclude every attributable cause; it is module LAYOUT, ch. 4's 4.8%
effect, and it passes the gate) · a tuple-keyed python dict being **O(n·m)** (a
consequence of the tuple type, not independent work; +0.77% inside a 1.13% spread).

**The ITEM was wrong** — already fixed, or a false premise, oracle in hand.
python integer arithmetic past 2^53 · python's `&` int32-signed (both already
correct, each including the item's own example row) · ruby `%g` (already
implemented; the real gap was the `#` flag) · java's divide-by-zero message
(already right AND already asserted) · `(super as any)[k]` (real `tsc` rejects it,
TS1034) · dart type-parameter erasure (Dart reifies) · kotlin's nested throwable
name "changes every `Name@1`" (only the throwable path reads `__bname`) · swift's
two abort probes "become byte-identical once value semantics land" (they already
were; only the stale headers survived).

**The prescribed FIX was wrong.** C#'s generated message on every derived type
(fixes one name, breaks twenty-six) · ruby's malformed formats via "emit % and
rescan" (would have converged all three engines on behaviour MRI rejects, turning a
halves divergence into agreed-and-wrong) · ruby's `round(n)` via `rtExactDec` (the
true expansion of 2.675 is 2.67 and MRI says 2.68 — `round_half_up` carries a
midpoint correction) · swift's abort message via a floor change (Go already guards
in front of the shared layer) · python's unhashable-key check by refusing `list` (a
tuple IS a list here, so it would abort `{(1,2),(3,4)}` and `d[i,j]`).

**FIXED.** ruby: `Float#round/floor/ceil(n)` (MRI's own numeric.c, including the
`int_round_zero_p` cliff), malformed formats now RAISE, `equal?` (not a bare `===`
— the twin's `jsRat`/`jsCpx` are comparable Go structs, which would have been a
designed-in halves divergence), kernel `method(:x)` (a hit in the PARENTLESS scope
is a host builtin) · kotlin: out-of-range indexing (six defects, not two; the
bounds check is FREE because it folds into the read), user throwables via `__super`,
`js_ktexcmatch`'s hack self-retires behind its own guard · js/ts: private names and
`for..in` (one root cause, plus a second site in the compilers nobody had checked),
function own-properties (the side-table read must not be spelled `fp[k]`, which
walks goja's `Object.prototype`) · swift: nineteen Array/String members plus six
adjacent under real `swiftc`, and value semantics as copy-at-STORE (complete by
construction, which is why an in-place subscript write still mutates) · python:
unhashable keys — `dict`/`set` refused, arrays compared STRUCTURALLY · `--full`'s
total silently discarding a gapped half's assertions · `tools/floor-coverage`
(330/345 floor bodies reached) · `tools/metajs-lint` (two live defects on its first
run, both wrong answers in shipped native binaries that no dynamic gate could see)
· `-why`.

**Dropped, or moved out of this file.** The `.expected` fixture path ·
`getattr()`-bound python builtins with kwargs · `glibc < 2.34`'s `-ldl -lpthread` ·
`mod.py` headroom. Moved to the manual: the scope API (7.3), per-language
divergences (7.9), the byte-vs-character `printf` rule (7.7).
