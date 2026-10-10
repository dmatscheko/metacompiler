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

Next free ID: **T-269** · Last verified: `17e73c7` (all [V] tickets were probed
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

### T-259 python: `range` is a materialised list · P2 · PY · IN PROGRESS (PY worker, round 4) [V]
`print(range(3))` prints `[0, 1, 2]` (CPython `range(0, 3)`); `range(3) == [0,1,2]` is
True (CPython False); type, slicing to a range. A lazy range touches ~51 pyIsArr sites
in python-rt.metajs, 28 Array.isArray in the interpreter, 29 js_py* externs and the
emitter's for-loop: 400-600 lines. Cost against the value; implement or DECLINE.

### T-260 python: `"a" + 1` concatenates · P1 · PY · IN PROGRESS (PY worker, round 4) [V]
Gives `'a1'` in all engines (also `y[0] += 1` on a str); CPython `TypeError: can only
concatenate str (not "int") to str`. Check `1 + "a"` (unsupported operand type(s)).

### T-261 python: `print(..., end=, sep=)` · P1 · PY · IN PROGRESS (PY worker, round 4) [V]
Compiled halves print the kwargs dict (`{'end': ' | '}`), the interpreter ignores
`end`. Also `sep`, `file=` (ignore or error), `flush`.

### T-262 python: `dict(enumerate(s))` · P1 · PY · IN PROGRESS (PY worker, round 4) [V]
Prints `{}` compiled, `<enumerate object>` interpreter; CPython `{0: 'a', ...}`. Check
`dict(zip(...))`, `list(enumerate(...))`, `dict(generator of pairs)`.

### T-263 python: TypeErrors for objects that are not iterable / callable / sized · P1 · PY · IN PROGRESS (PY worker, round 4) [V]
`class C: pass`: `iter(C())` aborts the interpreter; `C()()` aborts "call of a non
function value" in every leg; `len(C())` leaves no output line. CPython: `'C' object is
not iterable` / `'C' object is not callable` / `object of type 'C' has no len()`.

### T-264 python: `b"\N{BULLET}"` · P3 · PY · IN PROGRESS (PY worker, round 4) [V]
Prints `[34]` in all engines; CPython `b'\\N{BULLET}'` (`\N` is not a bytes escape).

### T-265 python: math functions and complex powers vs Apple libm last digit · P3 · PY · IN PROGRESS (PY worker, round 4) [V]
Non-integral complex powers and the `math.*` transcendentals differ from CPython in
the last digit on ~half of rows (Go/C pow, atan2, exp, log, cos, sin vs Apple's libm);
native differs from llvm.Run on 14 rows of a power probe. CPython calls the platform
libm: decide whether matching it is possible (port Apple's algorithms? no source) or
DECLINE with the measurement.

---

# RB — Ruby

### T-031 ruby: `respond_to?` on non-objects, `method(:x)` on block-taking names · P3 · RB · IN PROGRESS (RB worker, round 3) [V]
Round 1 measured it: `respond_to?` on a builtin aborts in every leg ("unknown Integer
method: respond_to?"). MRI's surface is ~730 names over 13 classes × 3 engines.
Decision: answer the ENGINE'S IMPLEMENTED surface (true iff the call would dispatch).
There are no per-class tables to derive it from (dispatch is if-chains, ~300 names in
the interpreter alone). Design: ONE shared per-class name table included by the
interpreter and layer 2 (the regex.js way), a Go copy with a `go test` comparing the
two, and a probe checking every listed name dispatches in all three engines.
`respond_to?` on non-objects needs a universal-method table (exact fix: a
`js_rselfcall(scope, name, args)` extern, ~3 engines × 25 lines). `method(:x)` for
`lambda`, `proc`, `loop`, `raise`, `require`/`exit`/`gets` answers NameError where MRI
answers a Method - identical in all legs, reasons recorded at the site.

### T-223 ruby: `Integer#[]` (bit reference) answers nil · P1 · RB · IN PROGRESS (RB worker, round 3) [V]
`5[0]` and `(-3)[0]` must be 1 (MRI); nil in every leg. Also `5[1..2]` is Ruby 2.7+,
skip that.

### T-224 ruby: `(m -1)` inside parentheses is a subtraction · P3 · RB · IN PROGRESS (RB worker, round 3) [V]
A method `m` (no local `m`): `(m -1)` calls m(-1) in MRI; every leg subtracts. The
parenthesised Expression path never tries the command form.

### T-225 ruby: top-level `p self` · P1 · RB · IN PROGRESS (RB worker, round 3) [V]
Interpreter prints nil, compiler aborts; MRI prints `main` (and `self.to_s` is "main").

### T-226 ruby: the interpreter ignores singleton `to_s`/`inspect`; no `instance_variable_set` · P1 · RB · IN PROGRESS (RB worker, round 3) [V]
`def o.to_s; "X"; end; puts o` / `p o` with a singleton inspect: the compiled halves
honour it, the interpreter does not (halves divergence). `instance_variable_set` is
missing in the interpreter.

### T-227 ruby: `sym.length` aborts compiled · P1 · RB · IN PROGRESS (RB worker, round 3) [V]
`:abc.length` / `.size`: compiled halves "len() of a symbol" (the lowered `js_pylen`
has no Symbol arm; the Go `js_pylen` is shared - prefer a ruby-specific lowering).

### T-228 ruby: top-level locals and top-level defs share one namespace · P1 · RB · IN PROGRESS (RB worker, round 3) [V]
`def w; 10; end; w = 4; p w -1` - compiler calls 4 ("call of a non function value: 4")
and the def is overwritten; MRI prints 3, and `w()` still calls the method.

### T-229 ruby: `Math.atan2` is one ulp from MRI · P3 · RB · IN PROGRESS (RB worker, round 3) [V]
e.g. `Math.atan2(3, 0.5)`, all three engines agree with each other. Check whether MRI
uses the platform libm (then compare against C atan2 on this machine) before porting.

### T-231 ruby: `NotLocalCmd` reads scopes from indentation · P3 · RB · IN PROGRESS (RB worker, round 3) [V]
The parse guard that keeps `x -1` (local x) a subtraction rebuilds the local table from
the source by indentation; unusual indentation misleads it. Replace with a walk-time
local table (the grammar sees assignments before uses) if the PEG allows it; else
document the limit at the site and close.

### T-232 ruby: `defined?` residual gaps · P3 · RB · IN PROGRESS (RB worker, round 3) [V]
`defined?(nil + 1)` and `defined?(x.nope)` give "method" (MRI nil - the receiver's
respond-check); `defined?(Math::PI)` nil (MRI "constant"); `defined?(not x)` nil (MRI
"method").

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

### T-250 ts: no Set, Map, Array.from · P1 · JS · OPEN [V]
`new Set()` is a ReferenceError in both TS legs. Port js's jsbLoad
(languages/lib/js-builtins.js), the "@@iterator" iteration probes and the Array
function to the two TS grammars.

### T-251 js: `Math.hasOwnProperty("abs")` · P3 · JS · OPEN [V]
"TypeError: Math.hasOwnProperty is not a function" in all legs; node true. (No
Object.prototype chain; at least the builtin objects should answer it.)

### T-252 js: a class-instance iterator's `return()` is not called compiled · P1 · JS · OPEN [V]
On a for-of `break` or array destructuring the run and native legs skip `return()`;
the interpreter calls it (= node). The close sites (js-to-llvm-ir ~3697, 4290, 4376,
5488 and js_jsiterclose) read with memGetExt / rt.getMember, which do not find class
methods - use js_jsvget as the iteration probes do.

### T-253 ts: `delete (Math as any).sqrt` aborts in the interpreter · P3 · JS · OPEN [V]
"delete operator not implemented" for delete through a parenthesised cast.

### T-254 js: "is not a function" for a comma/arrow callee · P3 · JS · OPEN [V]
`(index => 1, "intro" in o)()` prints the raw callee; node
`((intermediate value) , ("intro" in o))`. cpPrint declines on `=>`, function and
object literals. Also `Object.keys(Array)` is `["from","of"]` (node `[]`).

---

# JV — Java

### T-255 java: `instanceof` an interface · P1 · JV · Shared: interp-core.js · OPEN [V]
`class Impl implements Lst, Gen<String>` (Lst extends Col): java `true true true true`
for Lst/Col/Gen/Impl; interpreter `false false false true` (rtIsType in interp-core.js
walks only `__class`/`__super` and treats "List"/"Collection" as JS arrays); compiled
`true true false true` (a second interface is missed). ArrayList.equals in
java-util.java therefore cannot accept a user List.

### T-256 java: String identity · P2 · JV · OPEN [V]
`new String("ab") == "ab"` and `==` on a computed concatenation are true; java false.
A String has no identity in the value model. Cost first; likely DECLINE unless cheap.

### T-257 java: missing String API and core classes · P1 · JV · OPEN [V]
`String.valueOf`, `String.join`, `String.format` ("unknown name: String" interpreter,
"unknown method 'valueOf'" compiled); the interpreter's String methods stop at length,
charAt, equals, toString, substring, indexOf, isEmpty, toCharArray (`trim` fails);
`getClass()`, `new Object()`, `StringBuilder` unknown. Oracle: java 24. Consider
java-util.java (compiled with the program) for StringBuilder.

### T-258 java: overload residuals · P3 · JV · OPEN [V]
Where a static type is a type variable, a library type, or an ambiguous method result,
a call still picks by the runtime value; an anonymous class's forwarding to its super
constructor picks at runtime.

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
Int` is true where swiftc says false), kotlin's empty-range `first`, T-080, go's `%!s(uint64=...)` note (T-073) and T-240. One
mechanism likely closes all: a nominal type on the value. Design, cost, then
implement or split.

### T-198 kotlin: `String.toDouble()` accepts a numeric prefix · P1 · KT · OPEN [V spec-only]
`"1.5abc".toDouble()` parses (k2ParseNum: layer 2 parseFloat, Go twin Sscanf %g); Kotlin
throws NumberFormatException. Check toInt/toLong/toFloat/toDoubleOrNull the same way.

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

### T-267 go: `panic(any(nil))`, `any`, `fmt.Errorf` · P1 · GO · OPEN [V]
"unknown name: any", "unknown method 'Errorf'" in both halves. go prints `panic: nil`
(go 1.21+ `panic called with nil argument`; check go run) and supports Errorf (with
`%w`, errors.Is/As/Unwrap if cheap).

### T-268 go: `gofmt -l abnf/` lists abnf/jsrtgolang.go · P3 · GO · OPEN [V]
One line in goConvFail; gofmt it.

---

# SW — Swift  (oracle: swiftc 6.1.2)

### T-233 swift: a Dictionary as a Sequence · P1 · SW · IN PROGRESS (SW worker, round 3) [V]
`for e in d` with one binder binds the key (swiftc `(key: "a", value: 1)`); `d.map`,
`d.filter`, `first(where:)`, `min(by:)`, `sorted(by:)` abort ("unknown Dictionary
method"). All engines.

### T-234 swift: `hasPrefix` / `contains` on String abort compiled · P1 · SW · IN PROGRESS (SW worker, round 3) [V]
`"abc".hasPrefix("ab")`, `"abc".contains("b")`: llvm.Run and native "unknown String
method"; interpreter `true` (= swiftc). Audit the compiled String surface against the
interpreter's.

### T-235 swift: `Optional(2.5)` and `Optional.map` · P1 · SW · IN PROGRESS (SW worker, round 3) [V]
"unknown name: Optional" in all engines; `first.map { String($0) } ?? "-"`: interpreter
"method .map on nil", compiler "method call 'map' on a object"; swiftc `h` / `-`. Also
`flatMap`.

### T-236 swift: a bare trailing closure call in expression position, compiler only · P1 · SW · IN PROGRESS (SW worker, round 3) [V]
`return filter { $0 > 1 }` inside an Array extension: compiler "variable not defined:
filter"; interpreter `[2, 3]` (= swiftc).

### T-237 swift: extension-only members through a protocol type with no declared receiver · P3 · SW · IN PROGRESS (SW worker, round 3) [V]
`f().shout` / `a.b.shout` where the static type is a protocol still dispatch to the
conforming type's own member; swiftc binds the extension. Needs an expression typer.

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

### T-242 php: `$GLOBALS` is not modelled · P1 · PHP · OPEN [V spec-only]
Neither half supports `$GLOBALS['x']` read/write. A parse-time rewrite of
`$GLOBALS[e]` needs a frame selector on every target reader in both grammars (the
main frame is reachable through the hidden "\x00globals" name).

### T-243 php: a function declared inside a function is not global · P1 · PHP · OPEN [V spec-only]
`function outer(){ function inner(){} } outer(); inner();` - "Call to undefined
function inner()" in both halves; PHP: inner is global once outer() has run.

### T-244 php: an arrow fn captures a reference variable's cell · P1 · PHP · OPEN [V spec-only]
`$x=1; $r=&$x; $f=fn()=>$x; $x=2; echo $f();` prints 2 in all engines; PHP captures by
value at creation: 1. emitCapture / makeArrowFn in both grammars (the comment
justifying the cell contradicts the spec).

---

# LUA — Lua  (oracle: lua 5.5 binary, suite is 5.4)

### T-245 lua: the string library lacks `find`, `gsub` (and check `match`, `gmatch`, `format`, `byte`, `char`, `reverse`) · P1 · LUA · OPEN [V]
`print(string.find("abc","b",1,true), string.gsub("aa","a","b"))`: "attempt to call a
nil value" in all engines; lua `2 2 bb 2`. Lua patterns (not regex) - implement once,
shared by the interpreter and layer 2 if possible. Also `select` (and `select('#')`).

### T-246 lua: errors lack lua 5.5's variable-name suffix · P2 · LUA · OPEN [V]
lua says `attempt to index a nil value (local 't')` / `(global 'x')` / `(upvalue 'u')` /
`(field 'a')` / `(method 'm')` / `(constant '3')`; no engine does. Needs declared-name
tracking in the emitter, the failing operand from js_luarith, and a top-level frame in
the interpreter to tell top-level locals from globals.

### T-247 lua: "bad argument" wording and argument checks · P1 · LUA · OPEN [V]
Ours `bad argument to 'floor' (number expected)`; lua `bad argument #1 to 'floor'
(number expected, got string)` ('math.floor' when pcall calls it directly).
`math.sqrt("x")` answers nan compiled where the interpreter raises.

### T-248 lua: smaller error-semantics gaps · P2 · LUA · OPEN [V]
(a) a function that runs off its end returns nil instead of no value (`print(f())`
prints "nil", lua an empty line; `pcall(function() end)` gives "true nil") - assertable
once `select('#')` exists (T-245); (b) indexing a float holding an integer (`2.0`)
answers nil compiled, lua raises; (c) errors are positioned at the statement's first
line, lua uses the failing operation's line; (d) an error inside an xpcall handler
propagates, lua answers "error in error handling". Also: the Lua native SHOULD ABORT
matrix row could now compare all three engines.

### T-249 lua: numeric table keys are very slow natively · P2 · LUA · OPEN [V]
`t[i%100] = i; s = s + t[i%50]` for 3,000,000 passes retires ~422e9 instructions
(~140k per pass). Profile (`sample`) and fix; add a bench row if it is a common shape.

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

### T-211 the C floor's case table is an older Unicode · P1 · CORE · Shared: runtime.c · IN PROGRESS (CORE worker, round 3) [V]
Native only: `Character.toUpperCase` of U+019B, U+0264, U+1C89, U+1C8A differs from
llvm.Run and java (U+019B → U+A7DC is Unicode 16). Regenerate the floor's table from
the Unicode version the oracles use; check every language's upper/lower on the delta.

### T-230 -frozen big-integer arithmetic costs millions of steps · P2 · CORE · IN PROGRESS (CORE worker, round 3) [V]
The `-frozen` interpreter spends ~2-8M steps per operation on ~300-digit numbers; the
ruby ratchet needs ~50M of its 1e8 per-call budget. Profile the frozen host's path for
the shared bignum helpers (interp-core.js / runtime.metajs) and cut it.

### T-238 tests/coro-poc/build.sh fails on main · P2 · CORE · IN PROGRESS (CORE worker, round 3) [V]
`coro-floor.py: the C floor is missing 1 of the 17 coroutine pieces: gc_trace tag 16`.
Bring the proof-of-concept's checker in line with the floor, or delete the PoC if it
no longer earns its keep (it is referenced by T-172/T-173).

### T-151 MetaJS: the lexer reads `"\f"` / `"\v"` as letters · P1 · CORE · IN PROGRESS (CORE worker, round 3) [V]
`"\f".charCodeAt(0)` is 102 in MetaJS. tools/metajs-lint guards known sites (it found
two native-only defects this way), but the lexer in metajs-to-llvm-ir.abnf is the
root. Fix it, refreeze (`./mec -freeze languages/metajs-to-llvm-ir.abnf && go build`,
then `tests/gates.sh --freeze` must be a fixed point), regenerate every .ll, and
decide whether the lint rule can go.

### T-152 -frozen: `indexOf(x, from)` ignores `from` · P1 · CORE · IN PROGRESS (CORE worker, round 3) [V]
`"aXaXa".indexOf("a", 2)`: goja 2, -frozen 0 - a live goja-vs-frozen divergence. Fix
the frozen host and add a matrix assertion that runs both engines.

### T-153 goja-vs-frozen case mapping is not pinned · P2 · CORE · IN PROGRESS (CORE worker, round 3) [V]
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

### T-155 -frozen needs ~12 GB for js ratchet part 1 · P2 · CORE · IN PROGRESS (CORE worker, round 3) [V]
tests/js-test-full.js part 1 under the -frozen interpreter needs 11,907 MB where goja
needs 99 MB; typescript's larger ratchet needs 2,398 MB, so it is js-path-specific.
First suspect: `chainItems`, which retains every folded member chain for the whole
run. This is why the js ratchet was split into three files.

### T-156 `-trace` output is not reproducible run to run · P2 · CORE · IN PROGRESS (CORE worker, round 3) [V]
Raw Go heap addresses leak into `decl`/`read` `val` fields when a scope object is
rendered (32 of 170 lines in a python trace; ruby and php too). Fix in
abnf/trace.go's value rendering.

### T-157 `-callgraph` names a function by the minimum marker in its body · P3 · CORE · IN PROGRESS (CORE worker, round 3) [V]
`def f` on line 8 reads `L9` in every language. Use the definition's own position.

### T-158 frame elimination on arithmetic predicates · P3 · CORE + PY + RB · OPEN [V]
`rtTag`'s −13.9% on python came from removing ONE scope frame from ONE predicate.
The same `!pyIsObj(v)`-then-field-test shape sits at `pyIsFlo`, `pyIsNumeric`,
`pyInstClassOf`, `rbUserObj`, `rbIsClassObj`. Each needs its own call-site sizing and
a measurement against a dead-code matched control (tests/bench.sh header). Never
introduce a delegating wrapper on a hot path: that cost python +41% once.

### T-159 `rtIsNil` could serve js, kotlin, php · P3 · CORE · IN PROGRESS (CORE worker, round 3) [V]
`jvUndefOrNull` (js-rt), `ktIsNullish` (kotlin-rt), `phIsNull` (php-rt) are the same
four lines. kotlin and php already import runtime.metajs; js imports only regex.js,
so it would pay a first import plus seven `rtk*` hooks - measure first.

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

**DECLINED after measurement.** js sloppy-mode primitive `this` boxing (needs wrapper objects in every engine) · a generator's `finally` at collection (python needs refcount timing; no finalizers, manual 7.8) · `i in arr` after `delete` · iterator close on
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
