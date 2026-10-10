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

Next free ID: **T-205** · Last verified: `17e73c7` (all [V] tickets were probed
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

### T-199 python: compiled complex values print as `[object Object]` · P1 · PY · IN PROGRESS (PY worker, round 2) [V]
llvm.Run and native: `x = 3j; print(x)`, `print(3j)`, `str(1+2j)`; interpreter and
CPython `3j`. `complex(1, 2)` is undefined in all engines (interpreter "cannot
construct complex", compiled "call of a non-function"). A complex format spec beyond
fill/align/width aborts. Oracle: python3.

### T-200 python: a string literal holding an astral character · P1 · PY · IN PROGRESS (PY worker, round 2) [V]
`len("𝐀")` is 2 and `"𝐀" == chr(0x1D400)` is False in all engines (CPython 1 / True);
`chr()`-built strings are right. Compare ruby T-187 (same family).

### T-201 python: setting an attribute on a function aborts · P1 · PY · IN PROGRESS (PY worker, round 2) [V]
`def f(): pass` then `f.k = 4; print(f.k)`: interpreter "AttributeError: 'k'"
(uncatchable), compiled "cannot set field 'k' on *abnf.jsClosure". CPython prints 4.

### T-202 python: class repr lacks the module prefix · P1 · PY · IN PROGRESS (PY worker, round 2) [V]
`print(B)` / `B.__mro__` show `<class 'B'>`; CPython `<class '__main__.B'>` (builtins
stay `<class 'int'>`). All engines.

### T-203 python: int/int true division with a bigint operand rounds twice · P2 · PY · IN PROGRESS (PY worker, round 2) [V]
`(3**70)/3`-style quotients: we print `2.1222685869696762e+33`, CPython
`2.122268586969676e+33`. Needs a correctly rounded bignum division in all three
engines (Go: big.Rat; the two MetaJS engines ~40 lines each). Also `"%d %d" % (1,2,3)`
and `"%d" % (1,2)` are ambiguous while a tuple is a list (T-170).

### T-204 python: `None.k = 1` does not parse · P3 · PY · IN PROGRESS (PY worker, round 2) [V]
A literal `None` (or other atom) as an attribute-assignment target is a parse error in
both grammars; CPython parses it and raises AttributeError at run time.

---

# RB — Ruby

### T-031 ruby: `respond_to?` on non-objects, `method(:x)` on block-taking names · P3 · RB · IN PROGRESS (RB worker, round 2) [V]
Round 1 measured it: `respond_to?` on a builtin aborts in every leg ("unknown Integer
method: respond_to?"). MRI's surface is ~730 names over 13 classes × 3 engines.
Decision: answer the ENGINE'S IMPLEMENTED surface (true iff the call would dispatch),
derived from the existing per-class method dispatch tables, not a copied MRI list.
`respond_to?` on non-objects needs a universal-method table (exact fix: a
`js_rselfcall(scope, name, args)` extern, ~3 engines × 25 lines). `method(:x)` for
`lambda`, `proc`, `loop`, `raise`, `require`/`exit`/`gets` answers NameError where MRI
answers a Method - identical in all legs, reasons recorded at the site.

### T-185 ruby: a negative numeric literal binds the sign before a method call · P1 · RB · IN PROGRESS (RB worker, round 2) [V]
`p -7.abs` prints `-7` in all four legs; MRI `7` (the `-` belongs to the literal).
`-7.div(2)` gives `-3`; MRI `-4`. Oracle: /usr/bin/ruby.

### T-186 ruby: `y = x -1` with a local `x` is read as a command call · P1 · RB · IN PROGRESS (RB worker, round 2) [V]
`x = 5; y = x -1` - MRI 4. Interpreter aborts "call of a non function: x", compiled
halves "call of a non function value: 5". CommandForm must not apply to a known local.

### T-187 ruby: `"\u{1F600}".length` is a halves divergence · P1 · RB · IN PROGRESS (RB worker, round 2) [V]
Interpreter 2, native 2, llvm.Run 1, MRI 1. Astral code points become a surrogate pair
in two engines (the interpreter's `\u{}` comment says astral text is U+FFFD everywhere).

### T-188 ruby: `defined?(3 + 3)` · P3 · RB · IN PROGRESS (RB worker, round 2) [V]
Interpreter nil, compiled halves `"expression"`, MRI `"method"`.

### T-189 ruby: a top-level local is visible inside a `def` · P1 · RB · IN PROGRESS (RB worker, round 2) [V]
`loc=1; def m; loc; end; m` gives 1 in all four legs; MRI raises NameError
(undefined local variable or method `loc').

### T-190 ruby: top-level `@ivar` aborts · P1 · RB · IN PROGRESS (RB worker, round 2) [V]
`@yes = 1; p @yes` - interpreter "outside of a method", compiled halves "variable not
defined: self". MRI binds it on `main` and prints 1.

### T-191 ruby: Rational/Complex exactness (rest of T-030) · P2 · RB · IN PROGRESS (RB worker, round 2) [V]
Complex parts are untyped doubles: `Complex(3, 0.0).to_i` answers 3 where MRI raises,
`5.0.to_c` prints `(5+0i)` (MRI `(5.0+0i)`). A Rational's parts are doubles, so a big
plus a Rational is approximate. `(10**400).fdiv(10**399)` is NaN (MRI 10.0; MRI's
big_fdiv). Needs Rational parts that are bigs and typed Complex parts.

---

# JS — JavaScript and TypeScript

### T-040 js/ts: identifier `\u` escapes are not resolved in the interpreter halves · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`var a = 1; print(a)` prints `1` under the compiler and throws
`ReferenceError: a is not defined` under the interpreter, in js and typescript. The
compiler has `idName`; the interpreters have ~30 `up.in` sites each, and `up.in` is
matched TEXT, so no tag inside `Id` can fix it centrally. `js-test-full.js` SECTION 52
deliberately asserts no escaped spelling until this is fixed. Oracle: node.

### T-041 js: spreading a generator answers an empty array · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`[...g()]` gives length 0 (node 2), both halves. The spread path
(`evalArgs`/`spreadInto`) admits arrays and strings only; also check `f(...g())`,
`new Set(g())`, `Array.from(g())`, destructuring from a generator.

### T-042 js: `Math.pow(±1, NaN/±Infinity)` gives 1 · P1 · JS · Shared: runtime.c, jsrt.go · IN PROGRESS (JS worker, round 1) [V]
node gives NaN; Go's `math.Pow` and C's `pow` follow C99 and give 1. The `**` operator
probably has the same gap. Fix in one place for both.

### T-043 js/ts: `JSON.stringify` / `JSON.parse` are missing · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
ReferenceError in every engine. Implement the common surface (objects, arrays,
strings with escapes, numbers incl. -0/NaN→null, nested, indent argument, `toJSON`);
then assert `JSON.stringify(new Error("m")) === "{}"`.

### T-044 js: a derived class whose `super()` returns an object does not rebind `this` · P1 · JS · IN PROGRESS (JS worker, round 1) [V]

### T-045 js: Error own-property introspection · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`e.hasOwnProperty("name")` is true (node false); `Object.getOwnPropertyNames(e)` is
`[]` (node `[stack, message]` for `new Error("m")`).

### T-046 js: "is not a function" names the callee without node's spacing · P3 · JS · IN PROGRESS (JS worker, round 1) [V]
We say `o.a+1 is not a function`; node `(o.a + 1) is not a function`. The message is
rebuilt from whitespace-stripped source text.

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

### T-050 js: sloppy-mode primitive `this` is not boxed · P3 · JS · IN PROGRESS (JS worker, round 1) [V]
`f.call(7)` in a sloppy function: node gives a Number object (`typeof` "object",
`=== 7` false). The value model has no wrapper objects. Likely DECLINED.

---

# JV — Java

### T-192 java: the interpreter prints some doubles with too many digits · P1 · JV · Shared: interp-core.js? · IN PROGRESS (JV worker, round 2) [V]
`System.out.println(1.0/16777216)`: interpreter (goja and -frozen) `5.9604644775390625E-8`;
llvm.Run, native and java `5.960464477539063E-8`. The shortest-digits path (jFloStr /
coreDecDigits). A halves divergence; probe other powers of two and boundaries.

### T-193 java: overloads are chosen by argument count only · P1 · JV · IN PROGRESS (JV worker, round 2) [V]
`pick(int,int)` and `pick(double,double)`: `pick(1,2)` prints `2.0` (java `2`); likewise
`show(long)` vs `show(String)`. All engines. JLS 15.12.2 (most specific method).

### T-194 java: missing standard-library pieces · P1 · JV · IN PROGRESS (JV worker, round 2) [V]
All engines: `String.toCharArray` ("unknown String method"); `Runnable` as a
supertype ("unknown superclass: Runnable" / "variable not defined: Runnable");
Character.isDigit/isLetter/toUpperCase/getNumericValue; Math.addExact and relatives
(ArithmeticException "integer overflow"); Double.sum/max/min. Oracle: java 24.

### T-195 java: typed functional-interface lambdas do not convert their parameters · P1 · JV · IN PROGRESS (JV worker, round 2) [V]
`DoubleUnaryOperator op = x -> x+1; op.applyAsDouble(2)` prints `3` (java `3.0`). All
engines.

### T-196 java: `Math.exp(1)` is one ulp off · P3 · JV · IN PROGRESS (JV worker, round 2) [V]
java `2.718281828459045`, every engine `2.7182818284590455`. Java's Math.exp may use
an intrinsic but StrictMath.exp is fdlibm; check whether java's Math.exp agrees with
fdlibm on a boundary-value probe and, if so, port fdlibm's exp (all engines). Else DECLINE.

### T-197 java: `Double.parseDouble` does not read hex significands · P3 · JV · IN PROGRESS (JV worker, round 2) [V]
`Double.parseDouble("0x1p3")` throws; java `8.0`.

---

# CS — C#  (no dotnet here: spec-only, ECMA-334 and the .NET reference source)

### T-110 csharp: overload resolution is not "most specific" · P2 · CS · OPEN [V spec-only]
Constructors, methods and operators pick the first declared candidate whose types pass
`is`: with `O(object)` declared before `O(string)`, `new O("s")` picks object; C#
picks string (ECMA-334 12.6.4 better function member). Implement betterness for the
common cases (identity > numeric widening > reference conversion to base > object).

### T-111 csharp: derived field initializers run after the base constructor · P2 · CS · OPEN [V spec-only]
C# runs a derived class's field initializers BEFORE calling the base constructor
(ECMA-334 15.11.3). With a virtual `Get()` called from the base constructor C# prints
the derived initializer's value; the interpreter aborts "unknown name: x" and the
compiled halves print an empty line.

### T-112 csharp: `Single` exponent-notation threshold is unverified · P3 · CS · BLOCKED - needs dotnet
`flt36`/`flt37` in tests/csharp-test-full.cs assert `1E+07`, `1E+20`,
`3.4028235E+38` for `float.ToString()`. .NET Core 3.0+ prints the shortest round-trip
digits and its switch to exponent notation may not be the fixed window `csFloStr32`
implements (16777216f may print `16777216`). Settle with a real dotnet; until then do
not change the renderer.

---

# KT — Kotlin  (no kotlinc; the real stdlib jar can be driven from java:
`~/.gradle/caches/modules-2/files-2.1/org.jetbrains.kotlin/kotlin-stdlib/2.2.21/*/kotlin-stdlib-2.2.21.jar`)

### T-080 kotlin: `Array`, `Sequence` and a Map's `values` say `List is empty.` · P2 · KT · OPEN [V]
`arrayOf<Int>().first()` → `Array is empty.`, `asSequence().first()` → `Sequence is
empty.`, `map.values.first()` → `Collection is empty.` (check each with the jar). An
Array, a Sequence, a values view and a List are the same host array here, so no
receiver carries a kind - see T-081. Also absent: `reduceOrNull`/`reduceRightOrNull`.

### T-081 boxed values know their KIND but not their declared class · P2 · KT + SW + CORE · OPEN [V]
Two boxed values of one primitive kind and different declared classes cannot be told
apart: kotlin's mixed Float/Double set key, swift's `Int` vs `Int64` (`8 as Int64 is
Int` is true where swiftc says false), kotlin's empty-range `first`, T-080. One
mechanism likely closes all: a nominal type on the value. Design, cost, then
implement or split.

### T-198 kotlin: `String.toDouble()` accepts a numeric prefix · P1 · KT · OPEN [V spec-only]
`"1.5abc".toDouble()` parses (k2ParseNum: layer 2 parseFloat, Go twin Sscanf %g); Kotlin
throws NumberFormatException. Check toInt/toLong/toFloat/toDoubleOrNull the same way.

---

# GO — Go  (oracle: `go run`)

### T-070 go: the built-in `error` interface is not modelled · P1 · GO · OPEN [V]
`x.(error)` is false even for a type with `Error() string`; `case error:` in a type
switch never matches; a recovered runtime panic is a plain string, so `r.(string)`
succeeds where go fails and `r.(error).Error()` fails where go answers
`runtime error: ...`. Model `error` structurally (method set {Error() string}) and
give runtime panics a runtime.Error value whose Error() is the text, in all engines.

### T-071 go: a scalar type assertion does not panic on mismatch · P1 · GO · OPEN [V]
`var a interface{} = "s"; _ = a.(int)` does not panic in any engine; go:
`panic: interface conversion: interface {} is string, not int`. The comma-ok form
must answer `0 false`.

### T-072 go: an unrecovered native integer divide by zero dies through the C floor · P1 · GO · Shared: runtime.c · OPEN [V]
Natively, with no `recover()` in the program, `a / b` with b == 0 prints
`js runtime error: integer divide by zero`, exit 1 (the floor's `si_arith` die), where
go prints `panic: runtime error: integer divide by zero` + goroutine trace, exit 2.
The emitter gates the checked divide (`js_gidivz`) on `usesRecover` for speed:
measure with `tests/bench.sh go` before and after.

### T-073 go: fmt bad-verb notes for composites and wide ints · P1 · GO · OPEN [V]
Composites are not walked per element: go gives `[%!s(int=1) %!s(int=2)]`, we `[1 2]`.
int64/uint64 print as int/uint in the note because the box does not keep the declared
name (may need T-081).

### T-074 go: fmt verbs `%f %g %e %x %o %b` (and widths/flags) · P1 · GO · OPEN [V]
Unimplemented; they currently render as a typed bad-verb note. Implement go's
semantics including width, precision, `-`, `+`, `0`, `#` flags and `%x` on strings.

---

# SW — Swift  (oracle: swiftc 6.1.2)

### T-090 swift: optional dictionary lookups trap on a missing key · P1 · SW · OPEN [V]
`d[k] ?? "x"` and `if let v = d[k]` on a missing key trap in every engine; swiftc
gives the default / skips the branch.

### T-091 swift: `x as? Double ?? -1` does not parse · P1 · SW · OPEN [V]

### T-092 swift: `Dictionary.first` answers a key, not a `(key, value)` tuple · P1 · SW · OPEN [V]

### T-093 swift: remaining write-adoption cases · P1 · SW · OPEN [V]
`var x: Double? = nil; x = 3` prints `3` (swiftc `3.0`). A typealias to `Any`
(`typealias JSON = [String: Any]`), an observed/computed `Any` property
(`var x: Any { didSet {...} }`), and `for var` / `if var` / closure-parameter locals
still adopt the old value's type on a write. In a generic `func gg<T>(_ a: T, _ b:
T) -> T`, `gg(1.5, 2)` gives `2.0` (swiftc infers T = Any... check). The interpreter
runs top-level statements as walked, so a protocol declared after a top-level write
to a slot of that type is unknown there - a halves split for that case.

### T-094 swift: an extension on a builtin receiver · P1 · SW · OPEN [V]
`extension String { var lenPlus: Int { return count + 1 } }` fails in all three
engines (`unknown name: count`): `getMember` does not know Swift's `count`. Needs the
interpreter, the twin and layer 2. Do NOT route through `js_kget` (its string surface
is JavaScript's).

### T-095 swift: an extension-only member through a protocol-typed variable · P3 · SW · OPEN [V]
`let d: Named = D2(); d.shout` binds to the extension in swiftc and to D2's own member
here. Needs the expression's static type: a propagation pass plus a declared-type
table touching every member-access site in both grammars. Cost first.

---

# DA — Dart  (no dart here: spec-only)

### T-100 dart: an extension on `int` · P1 · DA · OPEN [V spec-only]
`extension IX on int { bool evenq() => isEven; }` fails: `getMember` knows no Dart
`int` members. Needs a member table in abnf/jsrt.go (or jsrtdart.go) and layer 2. Do
NOT route through `js_kget` (`"abc".slice(1)` started working when dart tried, which
real Dart rejects).

### T-101 dart: function-type subtyping · P2 · DA · OPEN [V spec-only]
The features check `tdef-legacy-form` asserts `fDouble is FOldStyle` is true with
`typedef FOldStyle(bool e)` and `int fDouble(int)`; parameters are contravariant, so
`int Function(int)` is not a `dynamic Function(bool)` and Dart says false. Implement
function-type subtyping (parameter contravariance, return covariance) and correct the
check.

### T-102 dart: chained `x as T is T` · P3 · DA · OPEN [U spec-only]
The spec grammar's relationalExpression allows one `is`/`as` postfix; the features
checks `typetest-chain-as-then-is` and `typetest-chain-as-as-isnot` may be invalid
Dart. Decide from the spec; if invalid, rewrite the checks with parentheses.

---

# PHP — PHP  (no php here: spec-only, php.net / php-src behaviour)

### T-120 php: an undefined `$variable` aborts · P1 · PHP · Shared: runtime.c · OPEN [V spec-only]
PHP 8 warns and evaluates to null and continues. The floor's miss hook can only
answer a value to THROW, never one to RETURN. Options: (a) a runtime.c hook (~15
lines + Go twin + php registration; a floor change must land with its layer-2
deletions in one commit); (b) a `js_scope_has`-guarded read (doubles the chain walk
on php's hottest path - measure); (c) emitter pre-declaration of every `$name` a body
mentions (~30 lines × 2 grammars, isset/compact/global regression risk). Written at
`js_phmissname`. Must land before T-121.

### T-121 php: variable scope leaks into every function · P1 · PHP · OPEN [V spec-only]
`$g=5; function f(){ echo $g; }` prints `5`; a closure sees `$g` without `use`. PHP
has no lexical variable scope. Needs a variable boundary (`global`/`use`/`$GLOBALS`/
`static`) in both grammars on the hottest path: ONE `js_scope_has` on ONE frame, never
a walk. Depends on T-120.

### T-122 php: `empty($undef)` aborts; `$u[0] = 1` does not vivify · P1 · PHP · OPEN [V spec-only]
`empty($undef)` / `empty($undef[0])` must answer true silently; select the
non-aborting read at PARSE time (`KwEmpty "(" TgtOnly ")"` before the expression
alternative) in both grammars. `$u[0] = 1` on an undeclared variable must create the
array: `emitRefPathV`'s `viv && ref.path.length > 1` guard skips a single-step target.

### T-123 php: top-level `label:`/`goto` or a bare `{ … }` block aborts · P1 · PHP · Shared: compile-core.js · OPEN [V]
`TypeError: Not a function` from compile-core.js `makeSeq` via `buildMain`: the
top-level list does not recognise the `{lblstmt}` / `{body}` records (php's own
`makeBody` does). Works inside a function. compile-core.js feeds the frozen snapshot:
refreeze.

---

# LUA — Lua  (oracle: lua 5.5 binary, suite is 5.4)

### T-130 lua: no `error` / `pcall` / `xpcall` · P1 · LUA · OPEN [V]
Without them, "catchable" is vacuous, and the compiled halves' `member 'q' of
undefined` cannot be brought onto the interpreter's `attempt to index a nil value`.
Needs a throw/catch surface: 2 grammars + layer 2 + twin. Error values, levels and
the `file:line:` prefix: match lua.

### T-131 lua: a subnormal float key read back through `pairs()` compares unequal · P1 · LUA · OPEN [V]
Compiled halves only (the interpreter was fixed): the key comes back as a different
double, so `k == orig` is false.

---

# SH — bash / batch  (oracle: /opt/homebrew/bin/bash 5.3; no cmd.exe)

### T-140 bash: unquoted `${a[@]@A}` splits into 3 words · P1 · SH · OPEN [V]
bash 5.3: 4 words. Both halves give 3.

---

# C — C subset

### T-150 c: `(float)x` is a no-op · P1 · C · OPEN [V]
The C subset's float cast does not round to binary32. Oracle: cc.

---

# CORE — MetaJS, the engines, shared runtime, tooling

### T-151 MetaJS: the lexer reads `"\f"` / `"\v"` as letters · P1 · CORE · OPEN [V]
`"\f".charCodeAt(0)` is 102 in MetaJS. tools/metajs-lint guards known sites (it found
two native-only defects this way), but the lexer in metajs-to-llvm-ir.abnf is the
root. Fix it, refreeze (`./mec -freeze languages/metajs-to-llvm-ir.abnf && go build`,
then `tests/gates.sh --freeze` must be a fixed point), regenerate every .ll, and
decide whether the lint rule can go.

### T-152 -frozen: `indexOf(x, from)` ignores `from` · P1 · CORE · OPEN [V]
`"aXaXa".indexOf("a", 2)`: goja 2, -frozen 0 - a live goja-vs-frozen divergence. Fix
the frozen host and add a matrix assertion that runs both engines.

### T-153 goja-vs-frozen case mapping is not pinned · P2 · CORE · OPEN [V]
The ratchet rows that compared the two hosts' case mapping asserted single-character
answers real languages contradict (`"ß".upper()` is `SS`) and were removed. Re-pin the
agreement with oracle-correct full-mapping expectations (node: `"ß".toUpperCase()` is
`"SS"`) in the metajs ratchet or a Go test; fix whichever host is wrong.

### T-154 replay-based generators repeat side effects in the interpreters · P2 · CORE + JS + PY · OPEN [V]
js, typescript and python INTERPRETER halves drive generators and async functions by
replaying the body, so effects before a `yield`/`await` run again: js
`take(inf(), 3)` prints `t0 t2 t5` (node `t0 t1 t2`); an async-generator trace is
garbage. The compiled halves are correct. Manual 7.14 explains the mechanism. Find a
real suspension for the interpreter halves that works under goja and -frozen, cost it,
implement or write the costed design here and DECLINE.

### T-155 -frozen needs ~12 GB for js ratchet part 1 · P2 · CORE · OPEN [V]
tests/js-test-full.js part 1 under the -frozen interpreter needs 11,907 MB where goja
needs 99 MB; typescript's larger ratchet needs 2,398 MB, so it is js-path-specific.
First suspect: `chainItems`, which retains every folded member chain for the whole
run. This is why the js ratchet was split into three files.

### T-156 `-trace` output is not reproducible run to run · P2 · CORE · OPEN [V]
Raw Go heap addresses leak into `decl`/`read` `val` fields when a scope object is
rendered (32 of 170 lines in a python trace; ruby and php too). Fix in
abnf/trace.go's value rendering.

### T-157 `-callgraph` names a function by the minimum marker in its body · P3 · CORE · OPEN [V]
`def f` on line 8 reads `L9` in every language. Use the definition's own position.

### T-158 frame elimination on arithmetic predicates · P3 · CORE + PY + RB · OPEN [V]
`rtTag`'s −13.9% on python came from removing ONE scope frame from ONE predicate.
The same `!pyIsObj(v)`-then-field-test shape sits at `pyIsFlo`, `pyIsNumeric`,
`pyInstClassOf`, `rbUserObj`, `rbIsClassObj`. Each needs its own call-site sizing and
a measurement against a dead-code matched control (tests/bench.sh header). Never
introduce a delegating wrapper on a hot path: that cost python +41% once.

### T-159 `rtIsNil` could serve js, kotlin, php · P3 · CORE · OPEN [V]
`jvUndefOrNull` (js-rt), `ktIsNullish` (kotlin-rt), `phIsNull` (php-rt) are the same
four lines. kotlin and php already import runtime.metajs; js imports only regex.js,
so it would pay a first import plus seven `rtk*` hooks - measure first.

### T-170 a real tuple type for Python · P3 · PY + CORE · OPEN [V]
`set_member` (runtime.c) accepts a named key only on tag 6; on tag 5 (array) it
demands a numeric index, so marking an array as a tuple is not representable natively
and a tuple must become a tag-6 BOX (copy the python FLOAT box, not the tag-14 floor
box - manual 7.6). Atomic: any subset is a `--cross` divergence. Sites: python
interpreter 17 `Array.isArray` + 49 tuple refs, python-rt.metajs 42 `pyIsArr`, python
compiler 54 tuple refs, jsrt.go `*jsArray`. Unblocks T-011, `%`-formatting arity,
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

### T-180 corpus: JS reject side · P3 · JS · IN PROGRESS (JS worker, round 1) [V]
Reject 703 / 727, must-parse 2623 / 2653 (tests/reference). Always measure BOTH rows:
guards have won reject files while costing must-parse files twice. Remaining: (a)
assignment targets that are not LeftHandSideExpressions (8 files: `() => {}()`,
`({a:this}=0)`, `switch (c) { default: default: }`); (b) `import`/`export` in a
Script (6 files) - needs the three ratchet files renamed to `.mjs` in
.vscode/launch.json and test.sh, not a grammar change; (c) `export default 3 4`
(2 files) - needs anonymous class/function declaration forms in the
`ExportDefaultDecl` branch; (d) `\u{0000000061}` (>8 hex digits) is accepted by node
and rejected here.

### T-181 corpus: other languages · P3 · per language · OPEN [V]
batch 0 of 2 wine files parse; swift 80% (an 18-file lexical tail: suppressed
conformances, `@isolated(any)` in a type, `borrowing` as a bare type name, raw
multi-line strings with 3+ pound levels, parameter packs, form feed and NBSP); dart
96.6% (~53 of the remaining 117 are multitests whose surviving `//#` variants are
labelled syntax errors); php / csharp / java / c long tails. python 92.9% and ruby
98.1% are at their hard caps. Unicode identifier tables for swift/java/kotlin/csharp/
python must each come from that language's own toolchain or spec (python3 is Unicode
16.0, node 24 is 17.0).

### T-182 go: `tests/go-test-widen.go` asserts `a.(int)` on a non-interface `a` · P3 · GO · OPEN [V]
Real Go rejects a type assertion on a non-interface value at compile time. Rewrite the
row with an `interface{}` operand.

### T-184 dart: `IndexError` vs `RangeError` text for `[1,2,3][7]` · P3 · DA · OPEN [U spec-only]
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

**DECLINED after measurement.** a generator's `finally` at collection (python needs refcount timing; no finalizers, manual 7.8) · `i in arr` after `delete` · iterator close on
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
