# Tickets

This file is the single list of everything still to fix, and the tracker the master
agent works from ([master-prompt-work-tickets.md](master-prompt-work-tickets.md) says
how; [master-prompt-find-tickets.md](master-prompt-find-tickets.md) files new tickets). The manual —
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

Next free ID: **T-742** · Last verified: `17e73c7`; round tickets filed since are [V] at the commit that filed them (all [V] tickets were probed
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

---

# RB — Ruby

### T-740 ruby: `<<` through a method call reads a raw property; `=begin`/`=end`; dynamic define_method compiled; `p ?c` · P1 · RB · IN PROGRESS (RB worker, round 14) [V]
`a = [[1,2]]; a.first << 3; p a` → ruby `[[1, 2, 3]]`, ours "not a number in '<<'"; `h = {"abc"=>1}; h.keys[0] << "x"` mutates the
hash's internal key (ruby FrozenError): an append statement's target path does `o[s.key]` for a `.name` step instead of calling the
method (interpreter resolveRefRuby and the compiler) — P1. `=begin` / `=end` block comments fail to parse in both grammars — P2.
`[:a,:b].each { |n| define_method(n) { } }` / `define_method(var)` → NoMethodError in the compiler legs (needs runtime method
installation). `p ?c.frozen?` → interpreter prints nothing, compiler NameError for `p` (ruby false; `?c` read as a ternary).
Residue: `[]` on a String, format/sprintf/gets results and builtin-yielded block args (each_char) are unboxed until read through a
place; read-out key boxes are distinct (`h.keys[0].equal?(h.keys[0])` false, ruby true).

---

# JS — JavaScript and TypeScript

---

# JV — Java

---

# CS — C#  (no dotnet here: spec-only, ECMA-334 and the .NET reference source)

### T-112 csharp: `Single` exponent-notation threshold is unverified · P3 · CS · BLOCKED - needs dotnet
`flt36`/`flt37` in tests/csharp-test-full.cs assert `1E+07`, `1E+20`,
`3.4028235E+38` for `float.ToString()`. .NET Core 3.0+ prints the shortest round-trip
digits and its switch to exponent notation may not be the fixed window `csFloStr32`
implements (16777216f may print `16777216`). Settle with a real dotnet; until then do
not change the renderer.

---

# KT — Kotlin  (no kotlinc; the real stdlib jar can be driven from java:
`~/.gradle/caches/modules-2/files-2.1/org.jetbrains.kotlin/kotlin-stdlib/2.2.21/*/kotlin-stdlib-2.2.21.jar`)

---

# GO — Go  (oracle: `go run`)

---

# SW — Swift  (oracle: swiftc 6.1.2)

---

# DA — Dart  (no dart here: spec-only)

---

# PHP — PHP  (no php here: spec-only, php.net / php-src behaviour)

### T-596 php: binding and undefined-variable warnings are never printed · P3 · PHP · BLOCKED - needs a user decision
PHP CLI prints `Warning: Cannot unbind $this of closure using $this in <file> on line <n>` (and undefined-variable/-index
warnings) on stdout with display_errors=1, plus possibly a `PHP Warning:` log line on stderr depending on ini. No PHP engine here
has a warning channel or tracks runtime line numbers at those sites. Decide: which stream/ini model to emulate, and whether
line tracking at warning sites is worth its cost.
The same channel would carry `Deprecated: Implicit conversion from float 1.5 to int loses precision` (a float argument to an int
parameter), which no engine prints either.

---

# LUA — Lua  (oracle: lua 5.5 binary, suite is 5.4)

---

# SH — bash / batch  (oracle: /opt/homebrew/bin/bash 5.3; no cmd.exe)

---

# C — C subset

---

# CORE — MetaJS, the engines, shared runtime, tooling

### T-741 tooling: -callgraph residue — go function bodies unnamed, runtime helpers as nodes, method-call externs, entry functions · P3 · CORE · OPEN [V]
Go: a function body is a separate closure that never gets a name, so `main` calling `g` reads `jsf_78 -> g` (go left out of
EDGE_CASES). Runtime helpers still appear as definitions/callees: jsnum.N, sw_copy and other sw_*, lua_appendall/lua_mk/lua_str,
kt_safeget/kt_clsfind/kt_dread/kt_fprobe, dt_safeget, php_scope_probe/php_site_text (treat every defined non-jsf_N, non-scaffold
handle-IR function as fabric). phpmain / dartmain appear as definitions with edges (entry scaffolding). Python emits `<module> ->
main` from the guarded `if main is defined: main()` even without a main. Method calls through js_swmcall, js_csmcall, js_phmcall,
js_rxmcall, js_ktsmcall, js_pyrxmcall, js_jsmcall, js_gomcall make no edge (only js_mcall / js_supercall / js_jvsuper are known).
Some python prelude closures are unnamed (jsf_66..68).

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
T-308 java: astral characters in string literals become U+FFFD — DONE d5dbe6f
T-309 java: `super.toString()` on implicit Object; qualified `new java.util.ArrayList` — DONE d5dbe6f
T-310 csharp: the string and char API — DONE ec556e0 (spec-only; culture-sensitive Compare left absent)
T-300 swift: unqualified `.none` builds an enum case — DONE 587ef41
T-301 swift: in-place mutation of an observed variable does not fire didSet — DONE 7136d77 2b458da
T-302 swift: `print(x as Any)` on an Optional; nested Unicode.Scalar — DONE 12eb521 fa6adb5
T-311 python: ordering across kinds does not raise — DONE 91496f3
T-312 python: `str.split(sep=)`, `str.encode()`, no-argument constructors — PARTIAL 2ad3dec (keywords, encode, no-arg constructors, Unicode digits); bytes()/frozenset are T-330; the "Did you mean" suffix DECLINED: CPython adds it only to its own traceback, which no engine here prints
T-313 php: partial application of a method declared with different arities — DONE bddc9e3 (spec-only)
T-314 lua: `next()` is a stub compiled; `tostring`/`tonumber`/`type`/`table.concat` missing — DONE 158badc
T-284 full case mapping (`"ß".upper()` is `SS`) — DONE fb749a8 (one mapper: abnf/casemap.go + runtime.c case_str; full for js/ts/metajs/python/java/kotlin/ruby/swift, simple for C#/dart/bash)
T-285 -frozen js memory, remainder — PARTIAL 92b173d (handle slots reused after a quarantine: 5.3 GB → 1.2 GB); incremental InGen is T-333
T-286 -callgraph: a definition's line in the other languages — DONE 235e3e5 e645a3b (up.start; all 13 handle-IR languages)
T-287 layer 2 `rtABigDivMag` binary-searches each quotient limb — DONE 55bd00d (258e9 → 34e9 instructions on a bigint-division probe)
T-288 js-to-llvm-ir under goja turns a lone `\uD800` literal into U+FFFD — DONE d1b1a42
T-289 native `d_asin` loses precision near 1 — DONE 2dd1438
T-322 kotlin: decimal `toFloat` rounds through double — DONE adc9f07
T-327 swift: `d["k"] = nil`; mutating String methods — DONE c8e174b
T-321 go: pointer identity, `f(ys...)` on Println, re-panic suffix, errors.As — PARTIAL 201e4bf (errors.As/Join, re-panic chain, spread call); pointer identity is T-336
T-320 bash: positional-parameter operator forms — DONE 41d1aad (`${x:-"$@"}` under a null IFS is T-338)
T-323 dart: String remainder; layer 2 indexes strings by code point — PARTIAL 16aeb1b (spec-only: UTF-16 layer 2, runes, String statics, String as Pattern); RegExp is T-339
T-325 ruby: `$!`, bare undefined calls, `Object#inspect`, Module class — DONE d917e48
T-326 ruby: Math domain errors, Regexp gaps, `%w` nesting, class reopen — DONE 33adf84
T-316 js: compiled own-key lists of builtins; Object.getOwnPropertyNames(Math); `"toString" in {}` — DONE a75ebb9
T-317 js: generator close semantics with finally / a non-done delegate — DONE f890f15 (the replay interpreters still repeat an outside iterator's effects: T-154)
T-318 js: `super.toString()` without an own toString — DONE 9068454
T-319 js: value reads of builtin methods on arrays/strings leak goja's host methods — DONE d2ea6d5
T-329 python: a str method's argument-type error is uncatchable — DONE c91f375
T-330 python: `bytes(...)` and `frozenset` — DONE bef6a12
T-331 php: `array_key_exists`, `implode` and the array function library — DONE 10d75a8 4138925 (spec-only; languages/lib/php-stdlib.php; php bench +1.20% paired, unexplained layout, inside the gate)
T-328 Go twin: a lone-surrogate char renders as U+FFFD — DONE beefa74
T-324 layer 2 `Array.sort` with a comparator fails — DONE 75153cf (gofmt-clean too)
T-315 lua: print writes invalid UTF-8 bytes as code points — DONE cfad758 (writeBytes host global; an uncaught error's text still leaves through the text channels)
T-047 js: a generator body's `this` is wrong in the NATIVE build only — DONE a39d475 (every generator/async body, in the emitter only)
T-081 boxed values know their KIND but not their declared class — DECLINED as a shared mechanism: the box id is cheap (go +1.10% inside spread, +128 B) but the cost is static typing - go needs a constant type checker, kotlin needs literals typed by their expected type, swift can flag its own box. Split into T-346..T-348 and T-240
T-337 go: a deferred call that panics stops the remaining defers — DONE 814a459
T-340 swift: String.Index, removeSubrange/replaceSubrange, `append(contentsOf:)` — DONE 21e46e1
T-339 dart: RegExp; List.map/where/toList/first — DONE 678af39 (spec-only; the charAt lone-surrogate item is T-352)
T-338 bash: `@A/@a/@K/@k` on `$@`/`$*`; `${x:-"$@"}` under a null IFS; `${!e*}` — DONE 06ec98b
T-336 go: struct values alias; pointer identity — PARTIAL 46dd880 e5648d0 0458aa4 (struct value semantics and pointer ==); rest is T-350
T-344 python: str.format conversion errors; ord()/chr() errors uncatchable — DONE 5b40343 1da292b
T-345 php: hoisting, by-ref closure calls, string indexing, math library — DONE 5d85943 (spec-only; raw-byte echo is T-353)
T-332 js: a surrogate-pair escape inside one literal in the interpreter — DONE 96d3115
T-333 js: InGen rescans the whole prefix per call — DONE 3456e3a (77.4 s → 19.6 s, 1.21 GB → 770 MB under -frozen)
T-335 ruby: `casecmp` folds Unicode — DONE 63cd2c4
T-341 ruby: Enumerator printing, `{}.each` without a block, Symbol#inspect quoting — DONE 35389da c449a89
T-342 ruby: reopening a builtin class does not add methods — PARTIAL 2763dd9 (builtin reopen); the uncaught-error prefix is T-356
T-343 js: `Array.prototype`, `typeof Object` — PARTIAL ea31394 (typeof Object; Array/String/Number.prototype); rest is T-355
T-348 kotlin: `concatToString`, `String(charArray)`, `reversed()` on surrogates — DONE 645704a
T-080 kotlin: `Array`, `Sequence` and a Map's `values` say `List is empty.` — DONE a4ef472 (a kind field on the host array: jsArray.kind, the floor cell's field d, host ids 99/100)
T-347 kotlin: Long/Int identity in Any equality, sets and LongRange — PARTIAL a2b2800 5eecfc1 d57a423 (LongRange/LongArray Longs, float≠int through Any, Long sum); Int vs Long equality is T-357
T-353 php: echo of a lone byte prints U+FFFD — DONE 01cb044 d123350 (spec-only; raw bytes via writeBytes; only reached library functions are compiled, 1.02 s → 0.12 s)
T-354 python: `int.real` compiled — DONE 1fd5daa
T-351 bash: `declare -p` of an empty or declared-only array — DONE bf02b13
T-349 go: a method on a defined non-struct type aborts — DONE 1dfd541 186c060 (Stringer of a defined non-struct type: T-240)
T-350 go: pointer map keys; value-receiver copies; native struct map keys — DONE df44f10
T-346 swift: `Int64 is Int`; `type(of:)` — DONE 6cffef4 (Int64/UInt64 carry width 65 on swift's own box; no floor change)
T-359 goja: a lone surrogate in a kotlin string literal becomes U+FFFD — DONE 2c0a569
T-352 interpreter host `charAt` at a lone-surrogate index answers U+FFFD — DONE 5fed219
T-171 moving floor bodies into layer 2 — PARTIAL cf8be7a (variadic prologue js_gorest/js_pyrest in the floor: go -10.7%); go's slice building is T-366
T-154 replay-based generators repeat side effects in the interpreters — DECLINED: a host coroutine is impossible under the pinned goja (one LIFO VM stack; a second Runtime cannot call the first's closures); the feasible pure-MetaJS design (resumable statement drivers + an evaluation tape for the parked statement) is ~1,200 lines per grammar × js/ts/python. Replay is quadratic (13.9G → 44.1G instructions, n=400 → 800)
T-293 correctly rounded transcendentals — DECLINED for full correct rounding (~3,000-3,600 lines over three engines; still disagrees with Apple libm where libm is wrong, e.g. cos 33/600); pow alone is T-363
T-173 coroutines as CPS in the emitter — DECLINED: CPS in the emitter is ~1,200-1,800 lines in compile-core plus 100-200 per emitter, high risk; the cheaper stackful context switch is T-364
T-334 bash: `${x,,}` / `${x@L}` leave Σ and İ unchanged — DONE 4af15cf
T-356 ruby: uncaught-error prefix and stream; Exception methods compiled; small gaps — PARTIAL 3d8c473 (Exception methods compiled, %W, Enumerator to_s, each_char, stderr); MRI's per-site prefix and "from" lines DECLINED: a shadow frame stack costs +2.13% for one store per statement (measured), ~+7.7% with frames
T-355 js: missing builtin globals and `.prototype` objects — PARTIAL 9ed9581 (prototype views, statics as values, Boolean, Object.is); Date/Function/native-code spelling DECLINED (clock, runtime compilation, no source kept)
T-049 js: `Symbol.iterator` — DONE d7710ab (symbols as objects, no floor tag; limits in T-369)
T-048 js: `with` — DONE 3a11a57 (lexical lowering + runtime chain walk; exact without an object-backed scope)
T-362 bash: a scalar declared without a value — DONE af02bbd
T-361 go: pointer to a slice; package var init order; interface dispatch on defined types — DONE c1375a5
T-240 go: a value of a defined non-struct type is the underlying value — PARTIAL c1375a5 (boxed at interface stores whose static type is known); rest is T-370
T-266 go: pointers, nil typed values and panic values in fmt and panics — PARTIAL c1375a5 (fmt of pointers/chan/func, %T of nil typed values, panic of defined types); pointer boxes at interface stores DECLINED this round as too risky (manual 7.9 model); rest in T-370
T-358 kotlin: interpreter gaps - `System.currentTimeMillis`, `kotlin.math.*` — DONE 01caa10
T-360 python: int/float methods off the type; missing int/float methods — DONE 015bfaf
T-357 kotlin: integer literals do not take their expected type — PARTIAL c89e01c (literals adopt their expected type at every listed site; lib/kotlin-decltypes.js); Int≠Long equality is T-373
T-372 bash: `unset` of a caller's local reveals the outer binding — DONE 79c62f8
T-371 go: native float multiply on a defined float type; method values; value-receiver method sets — DONE 3b04c69 44b1354
T-365 go: `for range` over a slice is ~140k instructions per iteration natively — DONE 44b1354 (14,115M → 3,353M; nil range, per-iteration loop vars, continue in classic loops fixed too)
T-366 go: variadic calls build the slice object expensively — DONE cfe340c (5.9k → 3.1k per call; folding obj_new + 5 stores into one floor call DECLINED: a floor call costs about the same)
T-370 go: runtime types where the static type is not tracked — PARTIAL ff7a8a2 (func %T, untyped &n, struct fields/channels/method params); (b)+(d) are T-375; (a) generic type parameters at run time (~300 lines, boxing T breaks generic arithmetic) and (e) pointer boxes at every interface store (~200 lines, high regression risk against the "&P{1} is P{1}" model) DECLINED
T-374 python: number dunders; unknown methods on numbers abort — DONE 042fc703
T-373 kotlin: Int ≠ Long at equality — DONE 9c4bdfd4 0cd6a242 (strict Int/Long equality after closing the literal sites: 0 assertions broke; remainder in T-378)
T-368 dart: `List.join` — DONE 072de00a (spec-only)
T-367 python: a lone high surrogate before a non-surrogate is read as a pair — DONE fa9fdcd8
T-369 js: `var` is block-scoped; for-of/in reuse an outer variable; symbol limits — DONE b5390929 b4ec7662 d6a71e8a (js only; the TS port is T-379)
T-376 go: `len`/`cap`/index/append pay ~19k instructions in `js_is_type` — DONE 44b773a3 (bench go 384M → 100M instructions, output checked against go run)
T-375 go: `map[any]` keys of defined types; imported packages' defined types — DONE b44a5741
T-381 go: a slice of a defined type prints its elements raw — DONE f29be2a6
T-363 pow is up to 421 ulps off in every leg that uses math.Pow / d_pow — DONE faeecc87 (530/650 → 0/650 not correctly rounded; ~236k instructions per native pow)
T-172 a catchable `GeneratorExit` — DONE 6077a025 (close() is throw(GeneratorExit()); no floor change)
T-158 frame elimination on arithmetic predicates — DONE c545f6c8 (pyIsFlo + pyInstClassOf inlined: python -3.78%, 15/15 pairs; rbUserObj/rbIsClassObj measured no gain, left)
T-364 coroutines: a stackful context switch in the floor — DONE d9d59b6a (lib/coro-switch.S: 83k → 52k instructions per next(), pthread/dlopen gone; native-full on arm64 and x86-64)
T-377 python: NotImplemented; dunder fallbacks; round/shift edge cases — DONE 3ca7accf
T-378 kotlin: literal adoption remainder; HashMap; Char.toInt in the interpreter — DONE b6ef1660
T-379 ts: port var hoisting and symbol conversions to TypeScript — DONE 5d7ec9cc
T-380 js: string iteration by code point; Array values()/entries(); generator @@iterator — DONE 39d20f9b (compiled object→number conversion is T-385)
T-383 goja prints some doubles with the wrong digits in the interpreter legs — DONE 38b10604 4d01c067 (goja vendored at its pinned revision under third_party/goja with one patched file; 411,923 doubles agree with node)
T-388 php: `var_export`, `json_encode` / `json_decode` — DONE 7558be65 d5669a63 (spec-only; php-src corpus .phpt blocks match)
T-382 go: arrays of defined types; a package's own defined types inside its code — DONE e38dafcc (nested arrays / arrays in struct fields: T-390)
T-386 js: static block order; TS namespace privacy; isNaN; non-callable well-known members — DONE 7a8c3362
T-385 js: compiled arithmetic never converts an object operand — DONE aba557f8 (layer-2 typeof on the slow path; bench js -13%)
T-387 js: `toExponential` / `toFixed` / `toPrecision` exist only through goja's leak — DONE 216985a9
T-391 ts: an exported `let`/`var` in a namespace is a snapshot compiled — DONE a25bb14e
T-384 python: binary operators on mismatched builtin types; huge repeats; int ** float — DONE 65621bbd e76fcfa5
T-170 a real tuple type for Python — DONE 93334360 (a tuple is an array of kind 16; no runtime.c change)
T-011 python: `d.items()` renders pairs as lists — DONE 93334360
T-181 corpus: other languages — PARTIAL 38b08481..4999997a (swift 115→118/123 swiftc-accepted, batch 0→2/2, c 213→217, java 4323→4329, php 2681→2691, dart 3336→3339, csharp +1; no must-parse file lost); remaining tail is T-395/T-396
T-393 ts: namespace `this` through an exported variable; uninitialised exports in the interpreter — DONE 3d7770f6 (the tsc crash is TS18030 on `this?.#foo`, input tsc rejects)
T-390 go: static type of arithmetic on defined types; `case []T:`; nested arrays — DONE b9299286
T-389 php: string/number comparison, visibility, foreach over objects, method_exists — DONE b6da4a79 0b0d526b da00a2fa (spec-only; visibility modelled; php-src corpus interp 766→787, compiled 895→912)
T-397 php: foreach by value walks the live array — DONE 108cb5ad
T-399 ts interpreter: stores to namespace exports before declaration / to exported functions — DONE 6e3f3443
T-395 swift: `a?.b = c`; inout closure parameters; literals typed by context — DONE 196e9a27 eee094e4 (the wrong answers, key paths, postfix/inout operators); DECLINED with reasons: macros (compiler plugins), #line/#function/#file (call-site defaults + swiftc module name), `case A<T>()` (always a type error in Swift), `for await` (needs an executor), local property wrappers, debugger-only `$x1`, `@const` (swiftc 6.1.2 rejects it), prefix+postfix of one operator name
T-398 batch: `%~n0` inside `call :label`; native string arena — DONE 73990fef
T-403 swift: stored-property types keyed by name across types; implicit members to static funcs — DONE 64a8ae13 cd910ffc (and the -qq fix for bash/c/toys 4f397c98)
T-402 js/ts: function declarations are not hoisted — DONE e4a8fdc3 (with Annex B block functions)
T-404 swift: `.init(...)` as an implicit member; ambiguous static names — PARTIAL 8ed63e0f (declared-type slots: annotated let/var, property defaults, assignments, returns); sub-expression contexts are T-405
T-401 the php ratchet's -frozen run is at ~94M of the 100M step brake — DONE c00a8c2f (root cause: quadratic varsSince; php -frozen 93M → 31M steps, no split needed)
T-400 php: array pointer functions, ini_set, object ids, anonymous class names — DONE 4a852b8d (php-src corpus: interp 807→842, compiled 928→964, no losses)
T-392 the C floor's `str_to_num` reads "Inf…" as Infinity — DONE 56400ff3 (floor = strconv.ParseFloat; python float(str) validates the whole string)
T-394 python: complex from negative ** fractional; augmented-assignment errors; hash; views — DONE 40c43710 060190f5 fe27ddbc
T-405 swift: lazy static lets; implicit members in sub-expressions — DONE a577592b 51b679b0
T-407 php: method-name case across a parent chain; small library and runtime gaps — DONE 3ef2eb58 (overrides recorded at declaration; a per-call lower-case lookup measured +15.1%, not taken)
T-410 swift: `Self` as a value; expected types through `==`, generics and receivers — DONE 9f3db72d (plus synthesized struct == and Array.contains)
T-412 swift: inherited static/class methods compiled; a type parameter as a value; Set by context — DONE 08310e01 d75d7754
T-396 corpus long tails: csharp, php, dart, java — DONE c1b3ed36..a629d763 (csharp 2674→2709, php 2692→2736, dart 3339→3371, java compiler 4329→4331; no must-parse lost). Remainder DECLINED by measured cluster: java 10 files javac itself rejects + 4 attribution-only; php reserved words as names, PHP 8.5 forms, constructs PHP rejects; csharp unsafe pointers (no pointer model), Mono prototype syntax, #if, identifier escapes; dart 53 unsplit multitests, unreleased primary constructors
T-413 swift: Set-keyed dictionaries; builtin types as values compiled — DONE 5905648d 7870660a
T-414 csharp/dart: static field named Count; generic statics; target-typed default; function-typed fields — DONE 43826f1a..6a5ba560 (spec-only)
T-409 python: +2.36% bench after the T-394 commit; `__hash__ = None`; `id()` — DONE 3af2e89d 00631154 (bench python 20.73G → 12.40G, output identical to CPython)
T-408 string → float conversions in the other languages — DONE 8750f160 63e2e8ec e916fd11
T-406 the python ratchet's -frozen interpreter run is at ~87M of the 100M brake — DONE 2fc5f8e4 (python 89M→34M, lua 73M→51M, java 61M→60M; every other half ≤42M)
T-416 swift: Dictionary API gaps; array/dictionary type syntax as values — DONE dc8e3e7e
T-411 php: array / "Class::method" callables; named args to closures compiled; dynamic static calls; numeric prefixes — DONE 6cb9c923 2bb4e833 79aac241 (spec-only; bench php 7.17G → 0.91G, output identical)
T-415 php: error cases and precedence from the php-src corpus — PARTIAL 05df225e (4 of 5; goto out of a try is now a compile error); the goto signal is T-420
T-417 csharp: target-typed `default` beyond ==; nullable default; List.IndexOf; Keys/Values members — DONE 3f3adbeb (spec-only; plus the List/Dictionary/String BCL sweep, lib/csharp-collections.js)
T-418 swift: zip, type(of:) of empty collections, NaN spellings parse (sign/payload DECLINED: needs double-bit host builtins + payload-carrying soft-float in the shared C floor, ~300-400 lines over 10 files) — DONE 74e50eb0
T-422 swift: lazy Sequence iteration (sequence(), user Sequence/IteratorProtocol, AnyIterator) — DONE dd4e2a5a
T-423 swift: [Double]() element typing on append; inout through field/element — DONE 103a47c1
T-420 php: named-argument errors, class-relative callables, scalar property writes, compiled goto out of try/catch (js_ctl_goto), hashed key index for long arrays (n=4000: 643G→3.4G) — DONE 47a950f6
T-424 swift: member shadows global, line-start paren, closure defer, Array + literal typing, print(to:) through a path, inout write-back after defer/catch — DONE 89d8a44c
T-425 php: non-callable Error, case-insensitive class names, missing-class errors, builtin string callables, goto-out-of-finally compile error, LSB forwarding through callables, callback TypeErrors, numeral interpolation keys — DONE 5921cef7
T-421 csharp: String.Format/standard numeric formats, StringComparison/SplitOptions/Normalize, System.Linq operators, remaining default contexts, target-typed new(), typed empty collections, List capacity, collection initializers (culture collation stays a loud failure) — DONE b7898d34
T-426 php: array union, case-insensitive class names in catch/instanceof/extends, Name::class at compile time, early binding, Closure::fromCallable, builtin TypeErrors, relative callbacks in array_*, compile-time finally/break jump checks — DONE 97ee2042
T-427 csharp: System.Math, re-enumerable iterators (fresh enumerator per GetEnumerator), foreach over user GetEnumerator, explicit-interface member shadowing — DONE 14b6e8f4
T-419 python: object() and object's own methods/__new__, exact int**int past 4096, the 4300-digit int<->str limit, type | type unions; java/python ratchets split for the frozen step brake; __sizeof__ DECLINED (CPython allocator layout, not a rule); sys.set_int_max_str_digits moved to T-431 — DONE f43fa335
T-429 php: is_a/is_subclass_of/get_parent_class, object-operand TypeErrors, Closure/Generator debug types, leading-backslash lookups, SELF/PARENT/STATIC in any case, compiled instanceof Throwable — DONE 23e6ad05
T-428 core: permanent unmarked region for boot/layer-2 top level/interned literals/small ints; collector cost per unused library function 611 -> ~200 instr per collection; every handle-IR bench row -3..-7% — DONE 6b2f74d6
T-430 csharp: double constants and IsNaN family, LINQ/Join/Concat over a user IEnumerable, Math overload result types, Abs overflow, MidpointRounding, MathF, BigMul, DivRem, CopySign — DONE f557ea80
T-433 php: non-numeric string TypeErrors, string bitwise ops, Perl-style increments and inc/dec TypeErrors, closures/generators as objects, "true given" names, built-in interfaces, stdlib class-function wrappers — DONE 137aa3c7
T-434 csharp: tuple ToString, float/Single type object, BCL enum casts, extension fallback for inapplicable instance groups — DONE 36feec21
T-436 csharp: synthesized record ToString (positional, public fields/properties, base first); applicable instance method beats extension (count-only applicability; user PrintMembers and derived-of-ToString-base not modelled) — DONE 6ade117e
T-431 python: explicit None no longer takes a default (compiled), attribute hooks, own-attribute delete + delattr, GenericAlias repr, unions of values, exact 0/±1 powers, sys digit limit — DONE 6ea7f2a6
T-432 python: big-int <-> bits 16 at a time (hex(10**4300) native 29.8s -> 0.7s); java StringBuilder part moved to T-438 — DONE 44e2ba60
T-435 php: engine interfaces extend Traversable/UnitEnum/Stringable, Closure/Generator as classes, ~bool/~null TypeErrors, PHP 8.4 closure var_dump layout, *RECURSION* guard — DONE 3bd31a96
T-438 java: StringBuilder appends onto a short tail + 32-level join counter (-frozen ~6.7K -> ~1.5K steps per append); java ratchet one file again — DONE 148fc8d1
T-437 python: CPython arity TypeErrors (missing/too many/multiple values/unexpected keyword/positional-only) in all engines, live instance __dict__, property builtin + decorator evaluation order; `int | "s"` is a TypeError in CPython too (not a defect) — DONE 7cedc576
T-439 php: SPL in php-stdlib (ArrayIterator/ArrayObject/SplDoublyLinkedList/Stack/Queue/ObjectStorage, iterator_*, class_implements/parents, spl_object_hash), __FUNCTION__/__METHOD__ as compile-time constants, yield from an Iterator, ArrayAccess dispatch — DONE e6c70cfa
T-441 python: qualnames, dunder/exception __dict__, property names via __set_name__, CPython "Did you mean" suggestions, copy + dataclasses modules (slots/kw_only/InitVar/make_dataclass raise NotImplementedError), dir() of classes/instances — DONE c0828553
T-442 php: __FILE__/__LINE__/__DIR__, automatic Stringable, generator keys after yield from (compiled), BP_VAR_IS fetches for ?? and isset/empty on ArrayAccess, ArrayObject isset of null, engine exception interface order, ArrayAccess by reference; SplObjectStorage isset(null) is true in php-src (asserted) — DONE 27c9ccfc
T-443 python: user __hash__/__eq__ dict/set keys, class-level descriptor __get__, 3-arg type(), function attributes as callables, add_note, collections (deque/defaultdict/OrderedDict/Counter), sys.exit/SystemExit, bytearray as byte list — DONE cf92473b
T-444 php: echo writes per operand, magic methods in any case, ??= reads with isset semantics (offsetExists/__isset, single evaluation), basename/dirname/pathinfo; realpath moved to T-447 — DONE 561511a7
T-446 python: aliased and module-qualified imports of the Python-written library modules bind where they run; ImportError for missing names — DONE 5860929b
T-448 python: runtime stdlib modules bound on import with aliases and from-imports (math/re/sys/builtins objects; others bare), re/sys functions as values — DONE f3aab379
T-447 php: declared properties take precedence over magic (compiled presence test), __unset in both engines, isset() asks __isset only, realpath (js_realpath in the C floor, Go twin, interpreter host hook) — DONE 6b675553
T-440 core: bitmap allocator (sweep = bitmap swap, cursor allocation) + 4 MB trigger floor; native bench rows -3..-21% (java -18%, python -21%), ~+3 MB heap — DONE e8607d6e
T-445 core: literal cache home slot by multiplicative hash, hits migrate home, frame-less hit leaf; another -4..-7.5% per row; baseline re-recorded — DONE cdf85b85
T-452 java: System.arraycopy in all engines; ArrayList shifts/grows/copies with it (1,700 remove(0)+add on 300 elements: 93.7G -> 2.8G native); clear() reallocates — DONE 3e47ce3b
T-450 php: property visibility (instance + static, magic fallback), uninitialised typed properties, per-object magic recursion guards, by-reference ??= targets, property_exists, ?? with BP_VAR_IS magic — DONE d835b006
T-449 python: json/itertools/functools/string/os.path/random (MT19937) written in Python, star imports, runtime modules unbound until imported, module-function keywords; time/RegexFlag/typing contents moved to T-455 — PARTIAL 558e0c91
T-453 java: java.util.Arrays (no stream), wrapper compareTo, fail-fast ArrayList iterators (modCount), typed enhanced-for variables, fmt1-3 correct under en_US (oracle command in the header); ArrayStoreException DECLINED (no element types) — DONE 3b814d6d
T-451 core: MetaJS compiler SSA locals, cached global reads (js_scope_cget), js_call0..3 without argument arrays, guarded direct calls, host_call switch; layer-2 bench rows -55..-80% (java 5.98G -> 1.35G); baseline re-recorded — DONE edd1755f
T-454 php: method visibility (all call forms, callbacks, ctor, traits), typed-property coercion/TypeError incl. strict_types, typed statics uninitialised, per-class private slots — DONE b0aab707
T-455 python: replayed generators log one-shot pulls, user __next__/__iter__ laziness (next(), for, yield from), descriptor method calls (compiled), surrogate repr (Go twin), __doc__ — DONE 2785aca2
T-456 java: instanceof on boxes (java-specific test), wrapper members, array clone, java.util.stream (Stream/IntStream/Collectors/Optional), HashMap in java's order, qualified type names — DONE c6905a87
T-461 core: the goja-hosted MetaJS interpreter leaked JavaScript prototype members (array map/forEach, function bind, ...); it now answers only the floor's members, like -frozen, llvm.Run and native — DONE f04d906f
T-459 js, ts: interpreter generators log pulls on outside iterators (shared js-genlog.js), replayed from the log; was [5,7] for [5,6,7,8] — DONE 0f734f33
T-458 python: Unicode 16 printable table (736 runs, shared JS + Go twin) for repr and str.isprintable, all BMP code points verified against python3 — DONE f3549e3c
T-457 php: non-static methods called statically (Error / callback TypeError, $this forwarding), compiled first-class callables resolve at creation, is_callable — DONE 5b15f6f1
T-462 php: non-static callbacks bind the frame's $this, dynamic calls of non-static methods raise, compiled __call on miss paths, static closures hide $this, Closure bindTo/call/__invoke/bind — DONE 3ab97e6a
T-460 java: static initializers after installation (compiled), unbound built-in method refs, typed array classes (Object[] untagged, see T-464), Short/Byte.compare, Long/DoubleStream, HashSet/toSet, Comparator, interface overload dispatch — DONE 0accbdc4
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
T-530 probe.sh reports a false "step brake" for any module that CONTAINS the brake's wording — DONE f7b60537
T-554 python: `int.to_bytes` returns a list; `format(x, "%")` overflow prints `+Inf`; `%a` and f-string `!a` do not escape — PARTIAL 1bf72c14 (format % overflow, %a and !a done); to_bytes needs a bytes type, T-556
T-555 python: format/messages and missing str methods — PARTIAL 1bf72c14 dc6d4d1b (tuple index text, %* / %.*, z and ',_' errors, call-arg order, isidentifier, translate done); rest is T-557
T-541 python: float `//`, `%`, `divmod` use the naive formula instead of CPython's float_divmod — DONE 35a82dc9
T-542 python compiler: `divmod()` with a big-int operand returns (NaN, NaN) — DONE 35a82dc9
T-543 python: three-argument `pow()` is wrong in all three engines, each differently — DONE 35a82dc9
T-544 python: `abs`/`round`/`float`/`int` of a non-number answer garbage instead of TypeError — DONE 35a82dc9
T-545 python: repr of a self-containing list or dict never finishes — DONE c99d17e2
T-547 python interp: slice-assigning a list into itself — DONE c99d17e2
T-548 python: slice ValueErrors are uncatchable — DONE c99d17e2
T-546 python: a generator that re-enters itself never finishes — DONE c386912e
T-549 python: mutation during iteration is never detected (dict, set, `sort(key=)`) — DONE c386912e
T-551 python: `zip(..., strict=)` — compiled treats the keyword as an argument, the interpreter ignores it — DONE c6fcc4a9
T-552 python interp: unpacking in a comprehension target does not check the count — DONE c6fcc4a9
T-553 python: an int can carry negative zero — DONE c6fcc4a9
T-550 python: generator expressions are eager lists; `zip(it, it)` over one iterator yields nothing — DONE 0494ca00
T-465 go: the Go twin answers `>`/`>=` true when either side is NaN — DONE 50978db6
T-466 go: tuple assignment does not evaluate LHS index/pointer operands before assigning — DONE 859ae7da
T-467 go: `copy` with overlapping slices copies forward and smears — DONE 4cb4bb0e
T-468 go: deleting map entries during `for range` — DONE 1f303254
T-469 go: interface equality and `map[interface{}]` keys ignore the dynamic type for int vs float64 — DONE 1bfad68e
T-470 go: `clear(slice)` writes integer 0; `min`/`max` on NaN and -0; interp `min` of sized ints — DONE 5fbfb220
T-471 go: channel and `make` runtime panics missing or wrong; a floor runtime error is unrecoverable natively — DONE a7cca8b6
T-473 go: map key semantics — composite-key print order, and interp float keys NaN / -0 — DONE 5b039f53
T-474 go: `[]T(nil)` conversion crashes the compiled legs — DONE 04522d5c
T-475 go: interp `string(rune(r))` of an invalid code point is not U+FFFD — DONE 2478ce86
T-476 go: Unicode case — titlecase digraphs and simple-fold orbits — DONE 9c2e45b8
T-472 go: runes and bytes produced by literals, range and indexing have dynamic type `int` — PARTIAL ac022f51 (range/[]rune/literal runes int32, s[i] uint8); []byte elements are T-562
T-477 go: features where the engines disagree on support — PARTIAL 646e06a3 (make(map,hint), Sprintln, strings.Title); Fprintf/io.Writer and the math package are T-563
T-486 js: parseInt edge cases, a different leg wrong per case — DONE 690ef8af (the >2^53 non-power-of-two radix tail is implementation-defined in V8; all legs agree, recorded in T-566)
T-492 js: missing TypeErrors — const assignment, `in` on a primitive, `instanceof` a non-object — PARTIAL 99030ccb (const assignment, in / instanceof TypeErrors); P3 wording is T-565
T-493 js: low-value divergences and gaps — PARTIAL 27ea061e (strings.raw, toSorted/toReversed/toSpliced/with/copyWithin); rest is T-565
T-480 js interp: a non-string computed key (`o[1]=v`, `o[true]`, `o[null]`) is stored but not tracked as an own key — DONE af004511
T-481 js: own-key order ignores the array-index rule (indices ascending first, then strings in insertion order) — DONE af004511
T-478 js: a `let`/`const` in a loop head is one shared binding, not one per iteration — DONE d3cdf4ad
T-479 assignment evaluates the RHS before the LHS reference; compound assignment reads the LHS after the RHS (js, java) — DONE de41557c
T-485 js: the Go twin's `Math.max`/`Math.min` drop NaN next to ±Infinity — DONE 690ef8af
T-487 js interp (goja): ToInt32/ToUint32 of |x| ≥ 2^63 saturates — DONE 690ef8af
T-482 js compiler: labeled break/continue out of `try`/`finally` inside a loop loops forever or fails to build — DONE 54788504
T-483 js compiler: `for-of` over an array mutated during iteration uses the starting length — DONE 54788504
T-484 js: string `<`/`>` and default `sort()` compare code points, not UTF-16 code units — DONE 54788504
T-488 js: TypeError/RangeError raised by runtime builtins is uncatchable; `repeat(Infinity)` hangs or panics — DONE 55313c71
T-489 js: fromIndex and separator arguments of includes / lastIndexOf / join — DONE 55313c71
T-490 js: regex `replace` replacement patterns `` $` `` `$'` and literal `$n`/`$<x>` — DONE 55313c71
T-491 js interp: index arguments are not ToIntegerOrInfinity'd (NaN, fractions) — DONE 55313c71
T-521 bash: the arithmetic model is 32-bit, bash's is 64-bit — DONE 4c0f0d43
T-520 bash: INT32_MIN prints as `-`; the compiler's shift by ≥ 32 or negative is IR poison — DONE 4c0f0d43
T-522 bash: arithmetic `&&`/`||` do not short-circuit; the compiler's `?:` evaluates both arms; a variable's value is not evaluated as an expression — DONE 664a4365
T-517 bash: indexed-array subscripts are not arithmetic; array elements inside `$(( ))`; negative subscripts — DONE f33dc025
T-518 bash: sparse indexed arrays — compiler keeps insertion order; `([k]=v w)` continues from the wrong index; `${a[@]:off:len}` counts positions — DONE f33dc025
T-528 bash: `${x:?msg}` in a subshell, and silent arithmetic / substring errors — DONE 2226584d
T-524 bash: `[[ A -eq B ]]` operands are not arithmetic — DONE 51c5285d
T-523 bash: `return N` is not reduced mod 256; the compiler ignores `;&` and leaves `$?` after a case — DONE bb673058
T-525 bash: `${N-word}`/`${N+word}` on positional parameters; `${!ref}` through an array element — DONE b84a31b5
T-519 bash: quoted or escaped glob characters stay active in case patterns, `${v#p}`/`${v%p}`/`${v/p/r}`, and `=~` — DONE c3dd5e6a
T-527 bash: `${v/p/r}` details — `&` in the replacement, an empty subject, extglob without shopt; the compiler's `${v%?}` splits UTF-8 — DONE acd2ecfe
T-526 bash: printf flags, operands and escapes — DONE f6d3b150
T-529 bash: command substitution nests at most 8 deep natively — DONE 295bfad5
T-531 bash compiler: `break N` beyond the loop depth is a compile error — DONE 0bfb57fe
T-532 lua: a closure made in a `for`/`while` body captures one shared variable, not a fresh one per iteration — DONE 51964557
T-533 lua: multiple assignment evaluates an indexed target's table and key after assigning — DONE e15b50e2
T-534 lua: integer-vs-float comparison is done in doubles, wrong past 2^53 — DONE 98819ebd
T-536 lua native: `math.max`/`math.min` die when a float beats an int argument — DONE 98819ebd
T-535 lua: `t[nil] = v` and `t[NaN] = v` are accepted; `pairs` then counts an unreadable entry — DONE cacdc476
T-537 lua: numeric `for` prep — an int start with a float step is an integer on the first pass; a zero step does not raise — DONE c339f500
T-538 lua: no proper tail calls — DONE b32c7caa
T-539 lua: `tostring` of a table or function has no identity — DONE c04e8cf0
T-540 lua: string library — `%z` class missing; `%s` with width/precision accepts `\0`; `\u{...}` above 0x1FFFFF — DONE 65572cc5
T-508 swift: `rounded(_:)` ignores its rule; `.sign` is never `.minus` — DONE 2a59ce6e
T-509 swift: `init(clamping:)`/`init(exactly:)` truncate; `radix:` is ignored — DONE bf511553
T-510 swift: `for v in arr` does not iterate a copy; interp and compiler disagree — DONE 5f26f384
T-511 swift: array subscript — a negative index wraps; interp grows the array on a write past the end — DONE 0b8cc7dc
T-512 swift: `switch` patterns on `Int.min`/`Int.max` — DONE 6b97a867
T-514 swift: an unimplemented member read answers nil silently; `?? default` then disagrees between legs — DONE c8ec3a26
T-515 swift: debug-description escapes and trap wording — DONE 17073228
T-516 swift: missing features that fail loudly — DONE 5399e731
T-513 swift: grapheme clusters — ZWJ only in the interpreter; RI pairs, CR LF, emoji modifiers, Hangul never; prefix/contains/`<` not grapheme-aware — DONE f04d4e28
T-507 swift: a literal typed by context is not adopted, so Double arithmetic runs as Int — PARTIAL c29f70a2 (literals adopt Double next to a Double: ternaries, collections, ??, payloads, min/max); Any-parameter and nested existential cases are T-570
T-567 bash: line continuation, loop status, redirected assignments, echo -e escapes — DONE 629bcf04
T-568 bash: `${x@Q}` of ESC, `$'..'` inside `${v/p/r}`, `"${!r}"` with r='arr[@]', byte-wise printf %c / precision — DONE d7d6796b
T-497 java interp: a static field initialised with a prelude class (HashMap, StringBuilder) fails — DONE 196484d2
T-501 java: `StringBuilder.append(char[])` appends the array identity — DONE 196484d2
T-500 java compiler: `(char)` of a double outside the int range wraps instead of saturating first — DONE 196484d2
T-496 java: the conditional operator ignores binary numeric promotion (JLS 15.25) — DONE 196484d2
T-498 java: try-with-resources lets close()'s exception replace the body's — DONE 196484d2
T-494 java: null dereference and unboxing never raise a catchable NullPointerException — DONE 8490cccb
T-495 java: casts never throw ClassCastException — DONE 106a19c2
T-499 java: Formatter exceptions are uncatchable; `%#.0f` drops the point — DONE 8877dfda
T-502 java: varargs forwarding and a homogeneous Object[] break String.format — DONE 06e2fe3a
T-464 java: interface default overload hiding; functional-interface overload choice; Object[] element types — DONE 7f214441
T-503 java: Throwable.toString ignores an overridden getMessage(); message and Unicode details — DONE 7355360d
T-506 java: commonly hit missing API — DONE e7916619
T-504 java: Integer identity outside the -128..127 cache — DECLINED: an Integer is a plain number in every engine, so == compares values; identity outside the cache needs boxes with identity in the value model of all engines (same reason as T-256), and `Integer b = a; a == b` must stay true
T-558 python: any/all/max/min/sum materialise a generator argument; an endless source never finishes — DONE 862aad9f
T-559 python: divmod() error text and dispatch — DONE 2d349ea5
T-557 python: str.maketrans, `__import__`, translate with a non-dict table, non-ASCII isidentifier — DONE 27088c06
T-556 python: bytes and bytearray are lists — DONE cce63c26
T-564 go native: `s[i]` on a string converts the whole string to bytes per index — PARTIAL 260975a8 (ASCII strings indexed in place, a 4-entry byte-form cache for non-ASCII; 77 s -> 0.34 s for 50 passes); the cache-thrash remainder is T-577
T-562 go: `[]byte(s)[i]` has dynamic type int; `cap()` of a channel — DONE 1fdfe139
T-563 go: `fmt.Fprint*` / `os.Stdout` / io.Writer, and the `math` package — PARTIAL cf215ba0 (Fprint*, io.Writer, os.Stdout/Stderr, math constants and exact functions, Sqrt, Pow for integral/half exponents); transcendentals are T-576
T-574 java: regex errors are uncatchable; fully qualified static calls; `(Object) arr` to Object...; "Caused by:" — DONE 049e1dfd
T-573 bash native: the mod.sh benchmark regressed +26% with the 64-bit arithmetic rework — DONE a0a680b4 (bench +26.15% -> +0.14%, spread 0.19%, re-measured by the master)
T-572 bash: `let` with a redirection; unquoted `let` field splitting; NUL bytes; per-iteration stack growth — DONE 61e861d2
T-560 js/ts: a generator that re-enters itself aborts instead of throwing TypeError — DONE 1e54eac7
T-566 js: `padStart(Infinity)` hangs; a missing method aborts natively; nested-try labeled jumps; big-radix parseInt — DONE 1e54eac7 28476b59 (incl. V8's fused parseInt for odd radices, verified on this arm64 node)
T-565 js: wording and remaining low-value gaps — PARTIAL 1e54eac7 (toString(radix) = V8 DoubleToRadixCString, radix message, node-24 JSON.parse messages, String.fromCodePoint); TDZ is T-580, arguments/__ keys/normalize are T-581
T-570 swift: a ternary passed to an `Any` parameter unifies; nested existential annotations — DONE 5b5742db 72163636
T-571 swift: mutating value methods on numbers and Bools; General_Category Character properties; trap exit status — DONE d8070820 (traps exit 133 in every leg; probe.sh accepts 133 for swift, 83fd0c9f)
T-579 bash: an arithmetic error does not cancel the assignment; bytes 0x01/0x02 break field lists — DONE e57f7553
T-582 swift: missing traps — Int(huge Double), inverted range, compiled removeLast on empty; the ratchet does not compile under swiftc — DONE de9fcee2 (the whole ratchet now runs on swiftc: 769 checks, 0 failures)
T-575 python: module `__name__`, builtin class methods with keywords, `slice`, bytes method gaps — DONE 411114f9 6bd49c7f except the dict pop/re-add-at-equal-size detection — DECLINED: CPython's answer depends on its hash table's resize/compaction history (a 4-key dict ends quietly, a 3-key one raises); a deletion counter made 46 of 116 probe rows wrong vs 38 before; matching it needs CPython's entry table per dict in all engines, a cost on every insert
T-585 swift: Range `lowerBound`/`upperBound`; `contains` on Double and String ranges — DONE b3f96517 (plus isEmpty, clamped, overlaps, indices, Int8 elements, count near Int.max, variable range-pattern bounds)
T-583 bash: several assignments in one command; `$(( ))` inside `${s:off:len}`; inner stderr of `$( )` — DONE 169887c8 71363b2d (plus redirections undone on return/break/exit, stdin rewound only under a stdin redirection)
T-584 bash: a value whose first byte is 0x02 is read as a field list — DONE 9b84ee7e
T-569 lua: runtime bookkeeping keys are visible to programs; the `io` library is missing — DONE 5d41bb3a 3bcbd2fd (key types are part of the key in every engine; io.write/io.stdout:write/io.type; io.read needs a stdin no handle-IR engine has)
T-577 go: invalid UTF-8 in strings — each engine differs; byte-index cache thrash — DONE 478a79d7 (floor byteAt/byteSub/bytesStr; s[i] O(1) natively; WTF-8 surrogate-byte residual is T-592)
T-576 go: `math` transcendentals, Float64bits, NaN sign, untyped math constants, missing members — PARTIAL 0deea0ec (Exp/Log/Sin/Cos/Tan/Atan/Hypot/Cbrt/Pow/FMA bit-exact vs go on 3000 inputs in all engines, Float64bits, exact pkg constants, undefined members rejected); NaN sign/payload is T-591
T-591 go: uint64 assignment of `1 << 63`; untyped big constant × float in the interp; overflowing constant args; NaN sign — DONE 94cf36c7 b030493b for the P1 parts; the NaN sign/payload part DECLINED: goja's floatToValue collapses every NaN result to one value and the interpreter negates by multiplying, so a float-bits builtin would split llvm.Run/native from the interpreter and goja from -frozen; it needs NaN-carrying float boxes in the interpreter first
T-592 go: UTF-8-encoded surrogate bytes in strings under llvm.Run and native; interp byte cache — PARTIAL c90fdc51 (interp byte cache set-associative); surrogate bytes moved to CORE as T-594
T-463 php: $this outside object context; bind/call on non-literal closures; binding warnings; S63T var_dump id divergence — PARTIAL 30ee93ac 35841d78 (a, b, d, e, f, g done from php-src; c's refusals done); printing binding warnings is T-596 (needs a warning-channel decision)
T-580 js/ts: no temporal dead zone for let / const / class — DONE 1c0e5be6 (checks only where a reference can reach the dead zone; bench mod.js module byte-identical)
T-581 js: arguments object, `__`-prefixed user keys, `String.prototype.normalize`, BigInt callee text — PARTIAL f67bcd8d df30f42f (normalize UAX #15 checked on every code point vs node, arguments object kind, __ user keys, BigInt callee text); arguments length/mapping is T-597
T-589 lua: tables and functions as keys collide — DONE 991c9c71
T-590 lua: metatables and the missing standard library — PARTIAL 9512ba5c..57ffd73d (metatables and all metamethods at +0.05%, _G proxy, coroutine on floor generators, utf8, math.random bit-exact, collectgarbage, string.pack, table lib via metamethods, FILE* type); rest is T-598, T-599
T-595 php: `[$closure, '__invoke']` callbacks; `[&$k, 4]` in the compiler; callback-form static fallback; bindTo identity — DONE 8ee86d4a (the call_user_func('M::sp') claim refuted by php-src: __call is right, bug45180/bug45186 phpts)
T-593 go: an overflowing constant is not rejected at compound assignment, typed const, return, element, send, `_ =` — DONE 89ffe028 (plus division by constant zero, negative constant shifts, constant index/slice/conversion range, truncated float constants: 86e8cea2..5208be9f)
T-600 php compiler: arrays passed by value are not copied; array copies drop references; spread of references — PARTIAL 2393d44a 464bc456 49c99ffa (by-value array params copied when written, shared references kept through copies and spreads, Closure::bind/fromCallable via an instance, named by-ref args); rest is T-602
T-597 js: arguments length and mapping; `Object.hasOwn`; `"__#"` user keys — PARTIAL 3687b718 (Object.hasOwn/groupBy/fromEntries, Map.groupBy, Number statics, String.raw, structuredClone, sloppy mapped arguments, arguments length slot); rest is T-603
T-578 java: PatternSyntaxException message wording; class literals of unloaded JDK types — DONE 6096b47d (plus a java.util/java.lang API sweep: 46e77cc7 cd485893 caf757fb)
T-598 lua: interpreter coroutines replay side effects; `ipairs("abc")`; indexing a coroutine — DONE 3d17ef92 (a real suspension primitive in both tag hosts: goja coroutine + frozen jsGenerator.suspend)
T-599 lua: os library, load/dofile, `<close>`, a real `_G`, NaN sign in string.pack — DONE 82a5666c 117bdd13 3a1cd51e for <close>, os (time/clock/date/difftime/getenv/exit), _G raw access; load/loadfile/dofile DECLINED (a native binary has no run-time Lua compiler, and constant-only load would make load(s) depend on whether s is a literal); NaN sign in string.pack DECLINED (goja canonicalises NaN, MetaJS cannot see a double's bits); __gc/__mode DECLINED (no engine lets a program drive its collector)
T-602 php: spread into a by-ref parameter; lazy copy holder counts; Closure statics by name; caller context in library callbacks — DONE ad2d64a0 for spreads into by-ref params, by-ref variadics, Closure statics by name, caller-frame library callbacks; lazy copy holder counts DECLINED: exact copy-on-write needs decrement-on-drop (refcounting), which manual 7.8 rejects, plus a separation check at ~85 write sites — documented at phpRefHolders
T-604 java: regex dialect gaps; missing Math/Long/Double API; TreeMap views; unmodifiable view write-through — DONE bc54d07a c6b40b82 (a Java dialect flag J in the shared regex engine, additive; the a{99999999999} runaway now 0.2 s)
T-586 python: slice ordering and unhashable slice keys — DONE b7d9fd8a (plus f-string {{, exception chaining, namedtuple, exact math ints, __init_subclass__, bound-method keywords, operator, user __format__: 2fcba0c7..cb34806c)
T-603 js compiler: a method call inside an assignment target; arguments edges; `"__#"` keys — DONE 8034b969 25069eb4 (plus newline-++ ASI, function-sees-lexical scope, generator/alias mapped arguments); remaining arguments/key edges are T-615
T-587 swift: ClosedRange.Index; an element as an expression pattern — DONE f96819ea (plus a 16-commit swiftc-checked sweep: 34b3fd80..f2327f9e)
T-609 kotlin: Regex should use the Java dialect (flag J) — DONE 3992ef1c (plus radix parsing, groupBy/associateBy valueTransform, joinToString limit/named args, java's Formatter for String.format, signed narrow hashCode, stdlib IAE messages)
T-615 js: spreading a non-iterable object; arguments passed to another function; holes; `"__#@"` keys — PARTIAL 6cac2349 (non-iterable spread/for-of/yield* TypeError, symbol methods on prototypes, Symbol.asyncIterator); rest is T-619
T-607 python regex: inline flags `(?i)` `(?s)` `(?x)` `(?i:..)`; a backreference to an unset group matches empty — DONE 0a3a158e (python dialect flag P, unset-group backreference flag n, used by ruby too)
T-610 python: name mangling; legacy `__getitem__` iteration; duplicate keyword via `**`; compiled implicit `__context__` — DONE 35e0414c
T-613 python/js interpreters: generators still run by replay — DONE e0c57419 (js/ts) f0393acb (python): every interpreter generator runs on the host coroutine primitive
T-614 python: smaller gaps — math, object repr prefix, OSError errno subclass, `__hash__` attribute — DONE 81b61124 (exp2/erf/gamma left out: CPython's answers depend on the platform libm, manual 7.9)
T-611 python: metaclasses, and the `enum` module on top of them — DONE 48620009 (Flag/IntFlag and the functional API not offered)
T-612 python: set iteration order of ints differs from CPython — DONE af6e7322 for sets of ints built by adding; dummy-slot order after removal, union/copy table sizes and constant displays recorded in manual 7.9
T-616 swift: `d[f()]? += 1` evaluates the target twice; `(+)` as a value; `\.0` key paths; metatype init overloads — DONE fb23c0a5 (plus a 16-commit swiftc-checked sweep: b32fe227..b32b114c)
T-617 kotlin: `Char.isLetter` for uncased letters; UInt in format; collection class names in format errors — DONE 8af069ec (plus a stdlib sweep, ~40 fixes: text search options, split, comparators, groupingBy, Char arithmetic, Regex.split, minus, radix toString, operator-function calls)
T-601 go: typed constant folding; named constant string index; float32 constant overflow; canonical error text — DONE fa884729 (plus a sweep: sort/slices/maps/cmp/iter/sync/atomic/strconv/unicode/utf8/bytes written in Go, range-over-func, nil-receiver methods, *T type tests, generic methods, ~15 engine defects)
T-618 java/kotlin regex: inline flags `(?i)` and atomic groups `(?>..)` under the J dialect — DONE 9a2177eb (java and kotlin; plus line terminators, Unicode case folding, (?U), comments mode)
T-608 java regex: forward `\k<name>`; `\X \G \N{..} \b{g}` and \p scripts/blocks; PatternSyntaxException getters — DONE 9a2177eb 60e91955 (forward \k error, \G, \X, \b{g}, every \p family from java's own tables, PSE getters; the jxGraphNext bound from the security review fixed); \N{name} DECLINED: needs Unicode's ~35,000-name table (hundreds of KB) in every java/kotlin native binary for a rarely used escape
T-621 swift: generic type arguments are not applied; static members on a generic parameter; non-ASCII operators — DONE 6d5d390a (generic arguments per instance incl. inference from context, M.zero, non-ASCII operators, enum metatype init; plus 13 swiftc-checked fixes)
T-622 kotlin: `Char(97)`, `sequence {}` builder, implicit-receiver operator calls, split with a named limit, formatter class names — DONE b75ef2dd fc105dae (the compiled split-with-named-limit item did not reproduce; an interp-only variant was fixed)
T-620 python: `re.error` is uncatchable; compiled class-body except bindings lost — DONE 1119e4c9 (plus heapq/bisect/contextlib/io/types/textwrap/abc/typing/fractions/statistics modules and ~20 engine fixes)
T-627 kotlin: sortedMapOf unsorted; Array.copyOf(); extensions not dispatched by receiver type; `import java.util.Random` — DONE 3186c9ec
T-626 swift: generics through metatypes and nil Optionals; `Set<Double>([..])`; overflow-reporting arithmetic; utf16Offset — DONE 26488dd9 9d7706d6
T-623 go: pointers through interfaces; generic zero values and %T; argument evaluation order; forward goto; late static types — DONE fe55e098 (a pointer field's printed address DECLINED: run-dependent hex)
T-624 go: missing standard packages; append capacity growth — PARTIAL fe55e098..HEAD (io, bufio, strings.Reader, time without a clock, path, filepath, flag, log, context, os.Args/Stdin, gc's growslice); regexp/json/reflect/stdin are T-632
T-630 kotlin: sorted collections forget order on insert; copyOfRange; java.util imports; extension dispatch limits — DONE cd7fb3ad
T-631 swift: interp runs a metatype init twice; `is T` with a metatype-bound T; `Int?(5)`; generic enum type(of:); Set returns — DONE d4683967 (is T / as? T with bound T, Int?(5), generic enum type(of:), Set returns, minimumCapacity); the interp double-init P1 did not reproduce in ~40 forms nor at the filing commit (4aa5b951) — closed as not reproducible, assertion 154a pins the case
T-588 bash: `$[ expr ]`; `$'\uDFFF'` length; native stdout buffering vs stderr order — DONE 77c376cc (plus \u/\U decoding in every engine; stderr ordered against buffered stdout natively)
T-635 bash: `$'\cX'` control escapes; `$'\?'` in the interpreter; batch native stdout/stderr order — DONE a03ea5f4 (bash); batch has no program-visible stderr — its fatal-diagnostic ordering moved to T-636
T-625 java/kotlin regex: supplementary characters in interp and native; PSE constructor; Kotlin Regex.options — DONE fbc1b072 (plus a java sweep: local class captures, array patterns, EnumMap/EnumSet, java.util.function defaults, Pattern/Matcher, records, BitSet ...)
T-636 batch native: capture nesting limited to 8; rt_die not ordered after stdout; bash `${v@Q}` of a non-UTF-8 byte — DONE 8bd03604 b91f7bd2
T-633 kotlin: `+=` on a `val` mutable collection reassigns instead of plusAssign — DONE c291b1fd a3b2e827 (plusAssign on a val, String elements, TreeSet/TreeMap constructors; kotlin bench -20.6% with identical output, from one js_is_type per +); extension resolution by static type DECLINED (the grammars compute no expression types); comparator Trees and deque classes are T-639
T-619 js: `"__#@"` keys vs symbols; arguments passed on; holes; `Object.getOwnPropertySymbols` — DONE f83299c1 for "__#@" keys, getOwnPropertySymbols, the [...null] message; js-builtins.js on real Symbol.iterator DECLINED (naming a symbol turns on symbol/BigInt operator checks: +45% on a native Map loop); arguments passed on and array holes moved to CORE as T-640
T-639 kotlin: `is` on collections; MutableList deque methods; comparator TreeSet/TreeMap; LinkedList/ArrayDeque/PriorityQueue — DONE c29d57f4 11f099d7 (is on collections, deque members, comparator TreeSet/TreeMap, LinkedList/ArrayDeque, JDK-heap PriorityQueue); the empty-ArrayDeque message ("ArrayDeque is empty." vs ours "List is empty.") DECLINED: an ArrayDeque is a list here and a distinct kind would touch ~26 primitive-kind sites
T-641 js: Object.getOwnPropertyNames of a prototype — DONE 02f3aef6
T-642 kotlin: `is ArrayList` / `is HashSet` and other concrete collection classes — DONE 10f71e86 (map views tagged with their real classes; two over-answers it introduced are T-643)
T-605 lua: the string metatable's `__index` cannot be replaced; `next(_G)` skips non-string proxy keys — DONE 1729b70e
T-637 java: static field and method of one name; field hiding; lazy static init; nested type names hoisted globally — DONE 84f9d63d (all P1 parts; Cloneable, live descending views, Matcher regions; hitEnd/requireEnd are T-646)
T-643 kotlin: `sorted()` results and deques answer `is ArrayList` true; SortedSet key views; javaClass.name — DONE 5238c524
T-645 lua: string arithmetic ignores the strings' metatable — DONE 0ccab79f
T-648 lua: `tonumber("-0.0")` loses the zero's sign in the interpreter and natively — DONE a735f8fd
T-628 python: raising `__hash__` without `__eq__`; slots in `__dict__`; `Plain[int]`; builtin `__module__`; regex dialect errors — DONE 05033f1e
T-629 python: stdin and input(); datetime, decimal, pprint, array, struct; remaining math — DONE 344fa727 f8b15ac9 1c48f755 (readIn host primitive + input()/sys.stdin, datetime/decimal/struct/array/pprint, str seeds, sumprod, fdlibm exp/log1p/log10); fma, __exit__ tracebacks and libm-dependent functions DECLINED (fma needs a double's bits in MetaJS; no engine has frame data for tracebacks; libm answers differ per platform — manual 7.9)
T-634 swift: an Optional Set return does not adopt its literal — DONE c2852a7a (an `Optional<Set<E>>` spelling is not converted — unprobed, left)
T-647 kotlin: `javaClass` / `::class` names of non-collections and collections; sorted-copy class edges — DONE d16124ba; residual class-name fidelity DECLINED as rarely printed: values views of hashMap/TreeMap/single/empty maps answer LinkedHashMap$LinkedValues (a per-map values kind would touch every kind-3 check in three engines), builtin KClass qualifiedName/toString (kotlin.String / class java.lang.String), and an exception's javaClass.name package
T-638 bash: `@Q` / `printf %q` / `declare -p` of a well-formed non-printable character — DONE 8a8a72b3 (iswprint table derived from bash itself and cross-checked against iswprint)
T-646 java: static method through a subclass name; RandomAccess / Serializable markers; Matcher.hitEnd — PARTIAL 067791a2 (statics through a subclass, RandomAccess/Serializable via instanceof exclusion at no ArrayList cost, interp binary names after nested lib loads); hitEnd/requireEnd and declaration order are T-650
T-606 php: library callbacks outside libCbArg; argument-error order; named arg into a by-ref variadic — DONE 8653dda9 (cbn2's expected text corrected to php-src's argument order)
T-644 js: own-property details of prototypes and functions — DONE 1456360f (a function declaration's arguments/caller/prototype own names stay a recorded value-model limit)
T-649 python: class-body names visible inside methods; compiled runtime exceptions' `__mro__`; regex lookbehind — DONE 7e3dc3f0 (\N{name} resolves only the 14-name table — a full Unicode name table DECLINED, as for java's \N)
T-650 java: static field through an instance; StringBuffer; CharSequence; ArrayList.clone; superclass order; Matcher.hitEnd — DONE 898cfc4c b4e0e8a3 7fcc0dcb except Matcher.hitEnd/requireEnd (T-654)
T-651 php: case-insensitive dynamic function names; `\`-prefixed names; array_map type checks; positional after named in a spread — DONE 350c6dcd
T-654 java: user `super.clone()`; FQN `java.io.Serializable` without import; Matcher.hitEnd/requireEnd — PARTIAL 218ab25d (Object.clone via super.clone, Cloneable/Serializable markers without imports); hitEnd/requireEnd and the nested-name-rename defect are T-656
T-655 php: function_exists(); direct calls in another letter case; callback message; var_dump of a first-class callable — DONE 8f44cfd4 858a0fd6 (Closure handle reuse after a temporary DECLINED: object ids are never reused by design, manual 7.8 GC model)
T-652 js: for-in through the prototype chain; `in` on prototype methods; Object.prototype methods as class values; private static getters — DONE 6b4fd2a2
T-657 php: first-class callable identity; var_dump of method callables; builtins without library wrappers — DONE 14fa3618
T-653 python: compiled `type(e) is ValueError`; `__class__` in methods; regex group refs, Unicode \w, findall None; class-body comprehensions — DONE b7d3736a
T-656 java: `x.new Inner()` with a third same-named nested type; Matcher.hitEnd/requireEnd — PARTIAL 429af651 (q.new Inner() by the qualifier's declared type; multi-level qualified nested names); hitEnd/requireEnd are T-661
T-659 php: calling a library function through a name built at run time — DONE 6a69f43e (programs without a computed call compile byte-identically; with one the module grows ~3 MB of IR, the exe ~0.7 MB)
T-662 php: library functions called with too few arguments do not raise ArgumentCountError — DONE a9a5ac97
T-658 js: inherited data properties read as undefined; Object.create; Object.prototype method set; shim identity — DONE 9f4ba349
T-663 php: user functions with too few arguments; library method arity; named-argument "not passed" — DONE 4b257c30
T-665 php: generator argument count checked at first resume; get_class/get_parent_class accept too many — DONE 9e081dd3
T-664 js: replacing a constructor's prototype; getPrototypeOf/setPrototypeOf/defineProperty; toLocaleString; Object.create(null) — DONE e4c56dde (PARTIAL: replaced prototypes, get/setPrototypeOf, defineProperty/ies, create(null), toLocaleString — 4 legs = node; valueOf.call boxing → T-669)
T-667 php: ArgumentCountError unavailable unless spelled; arrow-function generators — DONE e7427422 (ArgumentCountError class without the name; arrow-fn generators; new nolib matrix pair)
T-668 php: scalar parameter and return types are never checked — DONE 2e67e859 + e3111793 (scalar/nullable/union/class param+return types with weak coercion, strict widening, generator args at the call; spec-only from php-src)
T-632 go: regexp (RE2 dialect), encoding/json and reflect, real stdin — DONE 36957e91 b805369c c0936e9d (regexp RE2 dialect + Go errors, os.Stdin via readIn, encoding/json incl. depth limit and checked ,string; reflect subset json needs — 4 legs = go run; reflect method sets/Call/MakeSlice, CompilePOSIX/Longest, MarshalJSON on non-struct types DECLINED: no program in scope needs them; follow-ups T-672..T-675)
T-671 php: promoted constructor parameters declare no typed property; typed-property TypeError prints `self` — DONE b901ad9e 6b65aa72 (promoted params declare typed properties, self/parent resolved per zend_compile_single_typename, typed trait statics in the interpreter, promoted null default is the compile error; spec-only from php-src)
T-669 js: primitive wrapper objects (`new Number(7)`, `Object(7)`, `({}).valueOf.call(7)`) — DONE 0c010f6d (wrapper objects via __bvof in all engines; Object is a real function; Boolean a builtin — 4 legs = node)
T-670 js: defineProperty writable/configurable; getOwnPropertyNames of non-enumerables; Function/Boolean.prototype; String of a null-proto object — DONE 0c010f6d (writable/enumerable/configurable defaults, Cannot redefine property, getOwnPropertyNames order, Function/Boolean.prototype, null-proto ToPrimitive TypeError — 4 legs = node; follow-ups T-676/T-677)
T-672 go: `*t = *o` through a pointer; zero embedded struct field is nil; typed package constants print as numbers — DONE 2a3ded12 (struct store through pointers, zero embedded struct instances, promoted field writes, typed package constants and folded typed constants — 4 legs = go run)
T-673 go: fmt.Sscanf / Scan family; `%T` of a func and of os.Stdin/os.Stdout; `*(v.p.(*int)) = 5` does not parse — DONE 84120661 (fmt Scan/Sscan/Fscan families ported from fmt/scan.go, %T of funcs, *os.File std streams, qualified interface assertions, *(expr) = v — 4 legs = go run; follow-up T-678)
T-660 python: Unicode `\d` / `\s` in regex; `__class__` in a class nested in a method; astral \w across engines — DONE f1ce9dea 8d6cb63c (Unicode \d/\s, P text read by code point for astral spans/classes, __class__ in nested class bodies; plus must_advance, pos/endpos, Unicode IGNORECASE — 4 legs = python3 3.14; follow-up T-679)
T-679 python: `__class__` / `super()` in a function called while its class body runs — DONE 75ee1c30 (empty __class__ cell while the body runs, CPython's super() check order, cell filled before the metaclass hooks — 4 legs = python3; follow-up T-681)
T-678 go: `&r.i` / `&xs[1]` give the value, not a pointer; anonymous struct with embedded pointer; `%T` of a host func — DONE ae70ca39 1719a6bf 5042a693 aee854d2 (pointer cells over fields/elements, slot equality and pointer map keys, anonymous embedded fields, %T of host funcs, *(a).b, MethodSfx SameLine, *f() += — 4 legs = go run; follow-up T-682)
T-681 python: `__classcell__` through a metaclass `__new__`; `super()` object's own type — DONE 96f1a30b (CPython symtable cell rule, __classcell__ via type.__new__ with its RuntimeError/TypeError, super's own type — 4 legs = python3; follow-up T-683)
T-676 js: Object.freeze / seal do nothing; read-only writes and non-configurable deletes are silent in strict code — DONE 88ec20ec (freeze/seal/preventExtensions + is*, sloppy drops and strict TypeErrors with node's texts, counter-gated — 4 legs = node, TS via tsc 6)
T-677 js: builtin globals' `.name`/`.length`; `(7).constructor === Number`; `Object(1n)`/`Object(Symbol())`; String wrapper index keys — DONE 88ec20ec (builtin name/length, primitive .constructor identity, BigInt/Symbol wrappers, String wrapper index keys — 4 legs = node; follow-up T-684)
T-682 go: anonymous struct types as declared types; embedding a package-qualified type; fmt.Append*/os.Getenv — DONE c904bd4d f64c6e84 (anonymous struct types via a registry, pkg.T embedding, fmt.Append*, os.Getenv/LookupEnv, per-slot struct zeros in make and append growth — 4 legs = go run; follow-up T-685)
T-683 python: `__name__` is undefined at module level; `type(super)` / `isinstance(super(), super)`; `__classdictcell__` — DONE aebcd94c (__name__ = __main__ / dotted module names, the name super is the type, __classdictcell__ per 3.14, comprehension cell marking, super repr — 4 legs = python3; follow-up T-686)
T-685 go: append growth pads a function-local struct type with 0 in the compiler — DONE be5b3c0f (per-declaration zero makers for local types in the compiler — 4 legs = go run; follow-up T-687)
T-687 go: a local type shadowing a top-level struct corrupts it; zero of a type-parameter-typed field is nil — DONE 1133215b (local types keyed per declaration and stripped for printing, generic-local types instantiated, generic struct elision and slice element types — 4 legs = go run; follow-up T-688)
T-686 python: `(a): int = 1` binds nothing; `type(cell)`; 3.14 namespace keys `__firstlineno__` / `__static_attributes__` / `__annotate_func__` — DONE f133e518 (parenthesized annotated targets, type(cell), 3.14 metaclass namespace order with __firstlineno__/__static_attributes__/__annotate_func__ — 4 legs = python3)
T-688 go: append growth inside a generic function ignores the instantiation (pad zero and capacity) — DONE 1b2e4da9 (instantiated generic types copy as structs, per-instantiation $esz/$ezero, string/bool pad zeros — 4 legs = go run; the nil-slice cap 4/2 is gc's stack-buffer escape analysis, already DECLINED; follow-ups T-689/T-690)
T-684 js: class `.length`; Math/JSON member names; arrow-function "use strict"; TS strict default; frozen-array message index — DONE bad3ffd0 (class/Error/Map/Set length, Math/JSON member names, anonymous class names, strict arrows and generator prologues, strict undeclared assignment, .ts runs strict as tsc emits it, spec-ordered frozen-array refusals — 4 legs = node / tsc 6 + node; follow-up T-691)
T-666 php: the preg_* functions do not exist — DONE b6352924 (preg_* family on a PCRE2 dialect of the shared engine, php_pcre.c loops in php-stdlib.php; ~6,000 cases = libpcre2 10.48 via a php_pcre.c-port harness; follow-ups T-692/T-693; warnings stay with T-596)
T-693 php: `"a{b}"` in a double-quoted string is read as interpolation — DONE a0eeb482 d6c593dd (`{` interpolates only before `$`; the `${name}` / `${name[expr]}` / `${expr}` forms per zend_language_parser.y, strings and heredocs; spec-only)
T-691 js/ts: strict write to a function's `name`/`length` is silent; TS legacy octal `010` — DONE 43c6e0bd (strict refusal of fn/class name/length writes in all engines, no legacy octal/leading-zero literals in TS and JS separators/BigInt — node + tsc 6; follow-ups T-694/T-695)
T-689 go: native append costs ~120k instructions per call — DONE 0b7d7180 (append path off layer-2 plumbing externs: 1M appends 243G→18.2G instructions, 21.1s→1.8s; output byte-identical)
T-690 go: `%T` of a dynamically typed int64 prints `int` — DONE f5f57ab0 (type-parameter statics resolve to the bound type for boxing; any(x) conversions and ...T params box — 4 legs = go run; follow-ups T-696/T-697)
T-661 java: Matcher.hitEnd / requireEnd — DONE ac76281e (hitEnd/requireEnd set at java.util.regex's own node points, Begin root and Start minLength guard, sticky across reset/region — 4 legs = java 24; follow-up T-698)
T-696 go: compiled element width by type text — float32 and defined small-int slices — DONE 60860073 (defined number types registered for typed slots, float constants rounded/truncated in float32/int slots incl. arrays and maps, interpreter sized-int vs float constant arithmetic — 4 legs = go run)
T-697 go: nil slice element prints `<nil>`; interpreter cannot convert to a named interface type — DONE 3e8c2cd6 (untyped nil in slice/map slots becomes the typed nil, conversion to a declared interface boxes like any(x), fmt.Stringer — 4 legs = go run; follow-up T-699)
T-699 go: `int / 2.0` gives a float; `z, w := nil, 5` redeclaring a slice; conversion to another package's interface — DONE 93e55138 (untyped float constant converts by the other operand's static type incl. compound assignment, := nil takes the redeclared type, pkg.Iface(x) conversion — 4 legs = go run)
T-694 js/ts: valid numeric literals rejected — `0x1_0`, `0.0_1`, `08e1` / `09.5e1` — DONE f25a0f10 (one separator rule for every digit run in all four grammars, sloppy 08e1/09.5e1, exact radix rounding — 92-form sweep = node / tsc)
T-695 js: no function source text; defineProperty / delete of a function's `name` / `length` — DONE 03feaa8f c665a638 (defineProperty/delete of fn name/length via the own-property side table; js function and class source text from recorded spans, V8 elision in messages — 4 legs = node; TS keeps [function], see Settled; follow-up T-700)
T-700 js: Object.getOwnPropertyDescriptor missing; builtin methods read as values have no name; `delete K.name` on a class — DONE 86530e0c (getOwnPropertyDescriptor(s) in all engines, builtin methods carry node's name/length, delete of a class's name/length, interpreter hasOwnProperty of literal accessors — 4 legs = node, TS via tsc 6; follow-up T-701)
T-701 js: `delete` of a class static field or method does nothing — DONE e52a2e65 (delete of class static fields/methods/accessors in all engines without touching instance methods — 4 legs = node; follow-up T-702)
T-702 js: numeric-key class statics; for-in over a subclass skips inherited statics — DONE 686732d0 (numeric static keys by their string form in interp and layer 2, for-in walks inherited enumerable statics with shadowing — 4 legs = node; follow-up T-703)
T-703 js: a class cannot extend a plain constructor function — DONE d1dbefda (a constructor function as heritage via its instance descriptor, __ctor runs F on the new this, live statics through __ctorfn — 4 legs = node, TS via tsc 6; follow-up T-704)
T-675 regex (CORE): the shared backtracking matcher recurses per SPLIT and overflows the native stack — DONE 108459ea (explicit backtrack stack in regex.js and the Go twin; ^.*$ over 70 KB natively = python3/go run; regex-heavy native −19% python, −11% ruby)
T-698 java regex: a region end cutting a surrogate pair; `\G` in matches()/lookingAt() after an earlier match — DONE 3a5c52e3 (region cut inside a pair per CharProperty/CharPropertyGreedy, Matcher first/last/oldLast for \G — 1,312-row probe = java 24)
T-680 python re: IGNORECASE pairs from CPython's `_ignorecase_fixes`; non-ASCII ranges of 256+ chars — DONE 4777026e (IN_UNI_IGNORE lowercase lookup with re._casefix extras, any range size, caseless backrefs by lowercase — 104-row probe = python3)
T-692 php: preg_* remainder — recursion/subroutines, conditionals, branch reset, verbs; `(?xx)`; `[:lower:]` under /iu; PREG_BACKTRACK_LIMIT_ERROR — DONE d79a88e5 (recursion/subroutines, conditionals, branch reset, verbs, start options, (?xx), backtrack-limit error, TypeError, PCRE2 start optimizations — = libpcre2 10.48 harness; follow-ups T-705/T-706)
T-704 js: `new.target`; a class extending Array / Map / Set and the other builtins — DONE e69952fb..74fb6177 (new.target through construction and super(), Array/Promise/RegExp/wrapper subclasses with statics, overrides and super.m(); also classes declared in try blocks in the compiled halves — 4 legs = node, TS via tsc 6; follow-up T-707)
T-706 regex (CORE): the per-attempt step cap fires on long linear inputs — DONE c336f5ba 2dd9391b (one step budget per search scaled by program × subject, bash ERE VM on an explicit stack; pathological patterns still stop — = go/python/node/ruby/java/pcre2/bash on 70 KB)
T-705 php preg: PCRE2's empty-iteration loop rule; long `\p{...}` names; PCRE2 start-optimization residue; `(*ACCEPT)` scope; `MARK` — DONE d4599b4d (PCRE2 loop rule, loose \p names per 10.48, (*ACCEPT) scope, MARK reporting, minlength; residue declined, see Settled)
T-707 js: named properties on arrays; species for builtin subclasses — DONE 438be4a4 6c1f2bdd (named array properties in a side object on the non-index path in all engines, ArraySpeciesCreate / Promise SpeciesConstructor, regex match records are arrays — 4 legs = node; other languages' benches paired-neutral; follow-up T-710)
T-709 bash: the interpreter is very slow building a long string (`printf -v s '%*s' 35000` + `${s// /ab}`) — DONE 17952f3c (fixed-length patterns test one candidate per start like bash's umatchlen, linear result building, freed candidates, one-allocation padding — 35000-char repro 0.3–1.8 s in every engine, = bash 5.3; follow-up T-711)
T-711 bash: glob `?` and pattern elements match one UTF-16 unit in the interpreter — DONE 3e4a5ea7 (glob elements step whole characters in the interpreter — 4 legs = bash 5.3; follow-up T-712)
T-712 bash: non-ASCII POSIX classes in globs; empty extglob matches in `${v//pat/rep}` — DONE 61c107ce (Unicode class table read off macOS iswctype replaces the print table in both engines, undecodable bytes in no class, pat_subst's empty-match rule — 4 legs = bash 5.3; follow-ups T-713/T-714)
T-710 js: `Sub.from` / `Sub.of` on an Array subclass; `Symbol.species` reads; `typeof [].hasOwnProperty`; tagged-template strings array; `"01"` array key — DONE 19d778a0 (Symbol.species getters, Object.prototype methods as values on primitives/arrays, tagged-template strings as frozen arrays with hidden raw; Sub.from/of already worked — 4 legs = node; the "01" key moves to T-716 CORE; follow-up T-715)
T-714 c: string literals over 4096 bytes break llvm.Run; global `char *a[N] = {...}` crashes the compiler — DONE 7c1446cd 24ac13e0 (llvm.Run's IR reader accepts llir's u0x GEP indices on all three parse paths; string-literal brace items and literal-operand initializers in the C grammars — = cc)
T-713 bash: extglob edge cases — `*` before `@(`/`!(`/`+(`, `**(a)`, halves split on `*@(a|)` — DONE abdce695 (bash 5.3 sm_loop.c rules: star skips the empty tail, top-level trailing !(, empty +(/*( occurrences, match_upattern prefilter; rt_eg empty last alternative and UTF-8 steps — 4 legs = bash 5.3, 640-pattern fuzz; follow-up T-717)
T-715 js: `K.constructor`; sparse array keys; defineProperty attributes on array named keys; template object identity — DONE f35c764b 2a48d3ba (class .constructor, attributes and accessors on array named keys incl. integrity levels, one template object per call site — 4 legs = node, TS via tsc 6; sparse-array keys moved into T-640; follow-up T-718)
T-717 bash: native binaries exhaust the 4 MiB string arena on long-running programs — DONE 9dea2e18 (growable chunked heap with mark/release per loop pass and line, a floor for every outliving store — 352,000-line ticket shape = bash 5.3 at 3.8 MB; bash bench +1.16%, ~11 instructions per minimal loop pass, accepted for unbounded run length; follow-ups T-719/T-720)
T-708 core: building a large string natively is quadratic in ar_block's free-list walk — DONE 17370194 (big blocks on size-step free lists: python "ab"*35000 native 20.3 s → 0.16 s; 35000 appends 20–77 s → 0.5–5.6 s across 8 languages)
T-674 go (CORE): native recursion capped at ~500 Go frames by the js_gotry jump-buffer stack — DONE 3c89e494 (jmp_buf pool doubles instead of capping at 512; native go recursion now bounded by the 8 MB C stack, ~16,500 frames; = go run on depth 5000 and 10000-deep json)
T-505 java (and probably every language): unbounded recursion kills the process instead of raising — DONE 4b8bcb99 (one call counter in all four engines raising each language's own error — python RecursionError at CPython's 1000, js RangeError, java/kotlin StackOverflowError, ruby SystemStackError, lua catchable stack overflow, dart StackOverflowError, go fatal exit 2, swift SIGSEGV, csharp per .NET docs; non-python limit 10,000 frames, below the reference for java/lua/go but within what the native 8 MB C stack allows; php/metajs → T-721)
T-722 ruby: an uncaught internal error lacks MRI's `file:line:in 'meth': ` prefix — DONE 3430e76e for SystemStackError (statement site register in both engines, = ruby 2.6 first line and backtrace[0]); builtin-raised errors need frame names at jsrt.go raise sites → T-724
T-718 js: a strict-mode write to a getter-only property is silently dropped — DONE efccf010 (getter-only writes raise node's TypeError naming the holder in strict code and drop without an own property in sloppy, incl. statics, destructuring, Object.assign; stale accessors after delete/redefine — 4 legs = node, TS via tsc 6; follow-ups T-725/T-726)
T-720 tooling: tests/bash-test-full.sh is at ~97M of llvm.Run's 100M step brake — DONE a1656068 (bash ratchet split into full/.2/.3/.4 parts at ~30/39/14/65M steps, wired into test.sh, clang-check, native-full, gates)
T-719 bash native: quadratic list/word building; O(n²) `for w in $list`; positional stack capped at 4096 — DONE 9936078f (in-place list growth, field cursors, one-pass word and printf building, guarded in-place s+=, doubling positionals — native probes 334 MB → 2 MB, minutes → 0.4 s, = bash 5.3; bash bench +0.56%; follow-ups T-727/T-728)
T-727 bash: assigning an element to a scalar drops the scalar — DONE 6e13d2ea (a scalar becomes element 0 on element assignment, plus local -a, unset element, index order — 4 legs = bash 5.3)
T-728 bash native: arrays scan a flat table on every scalar access, cap at 4096 rows; stored values never freed — DONE 6e13d2ea 679f55e2 (per-name array records with bisect/hash, no row cap, long stored values in reusable per-slot buffers guarded by holders — 2.54G → 28M instructions, 2.8 GB → 3.8 MB; bash bench −12.5%; follow-up T-729)
T-721 php / metajs: unbounded recursion still kills the process — DONE c0d681d6 (php: plain recursion ends with the CLI's "Allowed memory size" fatal, exit 255, per php-src's VM stack — the max_allowed_stack_size Error only applies to internal re-entry, not modelled; metajs: Maximum call stack size exceeded, re-frozen)
T-716 core: a non-canonical numeric string key (`"01"`, `"1.0"`, `"+1"`, `" 1"`, `"-0"`) is an array index — DONE 3a10c7c1 (only canonical numeric strings index arrays, in the floor, twin, interpreters and layer 2; lua/php unchanged — = node)
T-594 go (CORE): UTF-8-encoded surrogate bytes are rejoined by the shared string floor — DONE f60458b8 (go raw-string mode in the floor and twin, rune-counted width/precision, byte-preserving strings.Split; new byte fixture — = go run)
T-640 js (CORE): arguments mapped through another function; array holes — DONE f69dba89 (arguments aliased through any reference via a hidden scope slot; real array holes in all engines skipped by keys/for-in/in/iteration — = node; other languages' benches within spread)
T-729 bash: mapfile/readarray; attributes on array elements; unset of a scalar's element; `${!a[@]:0:3}` accepted — DONE 04af2aca 4600656e (mapfile/readarray with all options, attributes on element stores, unset of a scalar's element, indirection validation and ${!a[k]}, reclaimed long elements with copy-on-read safety — = bash 5.3; RSS 60.6 → 1.9 MB; follow-up T-730)
T-723 js/ts: tests/srcpos-probe.sh reports -callgraph definition lines off by one — DONE f80983ea (bisected to e4a8fdc3 hoisting; emitFnInit records the declaration position; srcpos-probe 26/26 and now a gate in tests/gates.sh)
T-725 js: accessors inherited through a prototype object; `Reflect` missing in the interpreter — DONE 4dc32dac (accessors through prototype objects and views in all engines, getter-only refusals naming the prototype's constructor, Reflect with all 13 methods — = node)
T-726 js: accessors on array indices / functions via defineProperty; Symbol-keyed properties under freeze and in messages — DONE b036333f for functions and Symbol keys (defineProperty on functions with attributes and accessors; freeze/seal cover Symbol keys; Symbol(desc) in messages — = node); array-index accessors → T-731
T-724 ruby: builtin-raised errors print no `file:line:in 'meth': ` prefix; String + Integer concatenates — DONE 557c085e (raise helpers carry the builtin's frame name, sites stamped at rescue entry, def-line arity errors, String#+ / numeric coercion TypeErrors — 252-row operator matrix and uncaught lines = ruby 2.6; a features assertion that asserted the wrong answer corrected; follow-up T-732)
T-730 bash: a pipeline's last stage runs in the current shell; scalar `-i`/`-u` ignored by read and `${v:=}`; smaller gaps — DONE ffd8ba03..751ce5c1 (pipeline stages as subshells, ${!REF OP} on its target, attribute-aware stores everywhere, two-phase declarations, subshell trap reset, repeated local, plus set -e, let, compiler EXIT traps and capture fd copies — = bash 5.3; follow-up T-733)
T-733 bash: `o=$(nosuchcmd)` halves diverge; `exec` unsupported — DONE 820fcdfe (command not found on stderr ending its own subshell in both halves, redirection-only and bare exec — = bash 5.3)
T-731 js: accessors on array indices; -callgraph names js functions `jsf_N` — DONE 389e9440 f9c16192 (-callgraph source names for js/ts with srcpos-probe checking NAME@LINE in 10 languages; index accessors honoured by reads, iteration, methods and introspection behind a counter gate — = node, all benches within spread; follow-ups T-734/T-735)
T-732 ruby: `3r % 0` hangs the interpreter; `!x` after a newline read as a method suffix; operator/arity gaps — DONE 82d93a95 (Rational zero divisors raise, glued ?/! suffixes, x.OP(y) as send, receiver-less send to top-level defs, negative array sizes, frames for format and blocks — = ruby 2.6; follow-up T-736)
T-734 js: `Object.assign` onto an array aborts compiled; data attributes on array indices — DONE d83425a1 (Object.assign onto arrays/functions through js_jsmset, per-index data attributes with spec-ordered movers in a shared js-arrmove.js — ~200-row probes = node)
T-735 tooling: -callgraph definition names for python (`jsf_N`), php (body line, not header), swift (mangled `f$1`) — DONE a130da28 (python name pass follows js_pysig/js_pyfnname, php hoisted defs at their header, swift single definitions by source name — srcpos 26/26; follow-up T-737)
T-736 ruby: bang methods on strings do not mutate; `puts` inside an expression; send with a receiver to a top-level def; `===` — DONE 942cfc07 4db06632 (strings are mutable boxes per literal, promoted in place on first mutation through any place; frozen strings, identity; puts in expressions; receiver send to top-level defs; === by name — = ruby 2.6; ruby bench unchanged; follow-up T-738)
T-737 tooling + js: swift overloads and module bodies in -callgraph; `Object.assign(5, {...})` compiled — DONE ca9a0474 6763ce45 (swift overloads by source name, module bodies as <module>, unescaped jsonl; Object.assign ToObject's a primitive target incl. host objects — srcpos 26/26, = node; follow-up T-739)
T-738 ruby: strings produced by methods are not boxes; Array#to_h; define_method compiled; frozen hash keys; frozen_string_literal — DONE bfe97792 4235b67f (method/operator results are fresh string boxes incl. arrays of them, Array#to_h, define_method by String/Symbol in the compiler, frozen hash keys and immediates, frozen_string_literal per file — = ruby 2.6; follow-up T-740)
T-739 tooling: -callgraph loses swift and python call edges — DONE 3b5a140e (name-read/call helpers of python, swift, dart, kotlin, lua, ruby recognised; overload/method second names resolve to the source name; helpers dropped from the graph — EDGE_CASES for 12 languages, srcpos 38/38; follow-up T-741)

# Settled — do not re-open

**~196 items, one section, grouped by VERDICT and by nothing else** — no dates, no
rounds, no order, no commit archaeology, because none of it changes what to do
next. Each line is a verdict plus the shortest reason that stops someone
re-opening it. The detail is in git; the durable lessons are in the manual
(ch. 4 and 7).

**DECLINED after measurement — each is a number the next person should not have to
re-derive.** lua's `load`/`loadfile`/`dofile` (a native binary has no run-time compiler; constant-only load would make `load(s)` depend on whether `s` is a literal) · lua `__gc`/`__mode` and NaN sign/payload in lua `string.pack` and go `Float64bits` (no engine lets a program drive its collector; goja canonicalises every NaN) · python's "dictionary keys changed during iteration" for a pop and re-add of an already-yielded key at equal size (CPython's answer follows its hash table's resize/compaction history: a deletion counter made **46 of 116** probe rows wrong vs 38 before; matching needs CPython's per-dict entry table, a cost on every insert) · php's `isset`/`$this`
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

**DECLINED after measurement.** php preg residue: `\\p{Dash}`-style binary properties and bidi classes (no tables in the J engine; they raise an "unsupported" Error), Script_Extensions for `\\p{Greek}`/`scx=` (read as Script), ~1% of recursion+verb fuzz cases shaped by deeper PCRE2 start optimisations · java/pcre2 stack or recursion-limit failures on huge alternation loops (`(?:a|b){2,}` over 70 KB: Java StackOverflowError, non-JIT PCRE2 RECURSION_LIMIT; we match — succeeding where the reference engine runs out of stack) · js duplicate `class` / `const` / `let` in one scope accepted instead of node's SyntaxError (an invalid program we accept; no valid program changes) · java `(?:\\G.)+` after earlier Matcher operations (GroupCurly's internal accept node moves Matcher.last: a JDK implementation quirk, not specified) · js a detached builtin `apply`/`call` (`var ap = Math.max.apply; ap(null, [3,9,4])`) answering instead of node's TypeError (all legs agree; only an error case differs) · typescript function source text (the oracle prints tsc's re-emitted JavaScript, which the engines do not reproduce; .ts functions print `[function]`) · php heredoc `{$...}` / `${...}` end found by brace counting (a `}` inside a quoted string within the interpolated expression ends it early; double-quoted strings parse the expression properly) · python `__firstlineno__` / `__static_attributes__` / `__annotate_func__` on plain (non-metaclass) classes and their `__dict__` listing (introspection only; the metaclass namespace, where programs observe them, matches 3.14) · python rejecting `(p).k: int = 7` as a SyntaxError (an invalid program we accept; no valid program changes) · js `Function("src")` compiling source at run time (no engine here compiles at run time; it raises the EvalError text node gives when code generation is disallowed) · php per-file strict_types (the probes and suites are single-file programs; the whole-program flag equals per-file there) · php handle reuse after free (needs refcounts) · swift's module-qualified enum names in collections (`[o.Dir.up]` depends on swiftc's module name) · rbUserObj / rbIsClassObj frame elimination (no gain over 15 pairs) · a real suspension for the replay generators (impossible under the pinned goja; ~3,600 lines pure-MetaJS) · CPS coroutines in the emitter · full correct rounding of the transcendentals · a shared nominal type id on boxed values (cheap to store, but every consumer needs static types the grammars do not have; split per language) · matching Apple libm's last digit (ruby atan2: no portable algorithm reproduces it, 1.6% of pairs differ even from the correctly rounded value) · js sloppy-mode primitive `this` boxing (needs wrapper objects in every engine) · a generator's `finally` at collection (python needs refcount timing; no finalizers, manual 7.8) · `i in arr` after `delete` · iterator close on
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
