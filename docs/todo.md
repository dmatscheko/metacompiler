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

Next free ID: **T-567** · Last verified: `17e73c7`; round tickets filed since are [V] at the commit that filed them (all [V] tickets were probed
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

### T-556 python: bytes and bytearray are lists · P2 · PY · OPEN [V]
`(255).to_bytes(2, "big", signed=True)` → all legs `[0, 255]` (python `b'\x00\xff'`); `type(b"ab").__name__` → `list`,
`b"ab" == [97, 98]` → True (python False); `int(b"12")`, `float(b"1.5")` → TypeError naming 'list' (python 12, 1.5). A real
bytes type in the value model (repr `b'..'`, immutability, `==` against a list False, int()/float() parsing), all engines.

### T-557 python: str.maketrans, `__import__`, translate with a non-dict table, non-ASCII isidentifier · P3 · PY · OPEN [V]
`str.maketrans("ab","xy")` is missing: the unbound `str.X(...)` dispatch passes the first argument as the receiver, so a
static method needs dispatch work in all engines. `__import__` missing. `"ab".translate([..])` → TypeError (CPython accepts any
`__getitem__` mapping). `"😀".isidentifier()` → True (python False): past ASCII every printable code point counts as a letter;
use the XID_Start/XID_Continue tables.

### T-558 python: any/all/max/min/sum materialise a generator argument; an endless source never finishes · P1 · PY · OPEN [V]
`from itertools import count; print(any(x > 2 for x in count()))` → every leg runs until killed (python True). The
builtins turn the argument into a list before scanning, so short-circuiting never happens (also for a generator function).
Consume the iterator step by step in all engines.

### T-559 python: divmod() error text and dispatch · P3 · PY · OPEN [V]
`divmod("a", 1)` → TypeError "unsupported operand type(s) for //: 'str' and 'int'" (python "... for divmod(): 'str' and
'int'"). A class with `__floordiv__`/`__mod__` but no `__divmod__` (unprobed, from the code): divmod() uses the two, CPython
raises TypeError. Dict "keys changed during iteration" is detected only when the iterator runs out (snapshot compare); for
"pop a not-yet-reached key, then add one" CPython yields the remaining keys without an error while we raise.

---

# RB — Ruby

---

# JS — JavaScript and TypeScript

### T-565 js: wording and remaining low-value gaps · P3 · JS · OPEN [V]
TDZ says "q is not defined" (node "Cannot access 'q' before initialization"); toString radix message; JSON.parse syntax-error
text is older V8 wording. `(1e21).toString(36)` / `(2**60).toString(7)`: interp `5v1j4f4ds7c000`, compiler `...c4ks` — the
halves must agree (port V8's DoubleToRadixCString to all three engines). `arguments[0]=9` in a zero-arg call makes
`arguments.length` 1 (node 0; needs an arguments object). `JSON.parse('{"__proto__":1}')` → `{}` (`__` keys are internal slots,
manual 7.9). `String.fromCodePoint` missing everywhere; `normalize` missing in the compiler.

### T-566 js: `padStart(Infinity)` hangs; a missing method aborts natively; nested-try labeled jumps; big-radix parseInt · P2 · JS · OPEN [V]
`"abc".padStart(Infinity, "*")` → interp and native run until killed (node RangeError "Invalid string length"). Calling a
missing method natively (e.g. `String.fromCodePoint(65)` in a try) ends the binary with exit 1, interp and run throw a
catchable TypeError "... is not a function". A labeled break crossing several nested try bodies with for-of loops in between
does not close the for-of iterators opened inside the intermediate try closures (only the innermost's and the label's own).
P3: `parseInt` past 2^53 in a radix that is neither 10 nor a power of two — all legs agree, node differs in the last digit
(implementation-defined approximation in V8).

### T-560 js/ts: a generator that re-enters itself aborts instead of throwing TypeError · P2 · JS · OPEN [V]
A generator body calling `me.next()` on itself: every compiled leg aborts with "generator already running" (the floor's
gen_busy / the twin's jsGenerator.enter with no hook registered); node throws a catchable `TypeError: Generator is already
running`. Register a `js_rt_genbusy` hook in js-/typescript-to-llvm-ir.abnf the way python-to-llvm-ir.abnf does, and check
the interpreter half.

### T-478 js: a `let`/`const` in a loop head is one shared binding, not one per iteration · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`var f1=[]; for (let i=0;i<3;i++) f1.push(()=>i)` → all legs `3,3,3`, node `0,1,2`. Same for `for (let x of src)` (`9,9,9`
vs `7,8,9`), `for (let k in {a:1,b:2})` (`b,b` vs `a,b`), `for (const x of src)`, `for (let x of src) { ... }` with a block
body, `for (let [a,b] of [[1,2],[3,4]])` (`7,7` vs `3,7`). A `let` inside the body and `while`+`let` are right. Interp
also: `for (let i=0, get=()=>i; i<2; i++) fs.push(get)` gives `2,2`, node `0,0` (the head closure sees the FIRST copy). No
test uses `for (let`. Files: for / for-of / for-in lowering in js-interpreter.abnf and js-to-llvm-ir.abnf (check
typescript-*.abnf too, which shares the shape).

### T-479 assignment evaluates the RHS before the LHS reference; compound assignment reads the LHS after the RHS (js, java) · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
ECMA-262 13.15.2 and JLS 15.26 both evaluate the left-hand reference (object, key/index) first and, for `op=`, read the
old value before the RHS. All legs agree and are wrong in BOTH languages (go has the tuple-assign variant, T-466).
js vs node: `o[f("k")] = f("v")` logs `vk` (node `kv`); `a[i++] = i` → `0,2` (node `1,2`); `x += (x=5,10)` → 15 (node 11);
`ar[k++] += k` on `[10,20,30]` → unchanged (node `[11,20,30]`); `obj[key] = (key="b",5)` → `{"a":1,"b":5}` (node `{"a":5}`).
java vs javac/java 24 (operands from arrays): `j += j++ * 2` 16 (java 15); `a[k++] = k++` `[0,0,0]` (java `[1,0,0]`);
`b[k] = k = 2` `[0,0,2]` (java `[2,0,0]`); `arr(c)[f("i",1)] += f("v",5)` logs `vAi` (java `Aiv`); `y -= (y = 1)` 0 (java 2);
`s += (s = "b") + s` `bbb` (java `abb`); `e[idx] += (idx = 1) * 100` `[1, 102]` (java `[101, 2]`). Files: the assignment /
compound-assignment builders of js-*, typescript-* and java-* grammars; check lib/interp-core.js and lib/compile-core.js
first in case the order lives in a shared helper, and probe the other languages' compound assignment while there.
Shared: JV (java grammars) — a worker taking this needs both groups' files.

### T-480 js interp: a non-string computed key (`o[1]=v`, `o[true]`, `o[null]`) is stored but not tracked as an own key · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`var o={}; o[ks[i]]="v"` with `ks=[1,"s",true,null]` then `Object.keys(o).length, for-in count, JSON.stringify(o), o[ks[i]]`:
interp `key 1 0 0 {} v` (goja and -frozen), run/native/node `key 1 1 1 {"1":"v"} v`; `p[2]="x"; p["k"]="y"` keys: interp `k`,
others `2|k`. `setMember` (js-interpreter.abnf ~5812) calls `trackKey(o, k)` with the raw key and `trackKey` (~7528) returns
early on `typeof key != "string"`.

### T-481 js: own-key order ignores the array-index rule (indices ascending first, then strings in insertion order) · P2 · JS · IN PROGRESS (JS worker, round 1) [V]
Keys inserted `b,2,a,1,-1,01,4294967295,4294967294,1.5,0,10,1e3," 3",9007199254740993,3`: all legs print insertion order;
node `0|1|2|3|10|4294967294|b|a|-1|01|4294967295|1.5|1e3| 3|9007199254740993` — affects `Object.keys`, `for-in`,
`JSON.stringify`, entries/values. Literal `{z:1,5:2,y:3,0:4,"-0":5}`: ours `z|5|y|0|-0`, node `0|5|z|y|-0`.
`JSON.parse('{"b":1,"a":2,"1":3}')` re-stringified: ours `{"b":1,"a":2,"1":3}`, node `{"1":3,"b":1,"a":2}`. Manual 7.9 records
insertion-order `keysOf` but nothing exempts js from ECMA-262 OrdinaryOwnPropertyKeys. Interp only, same probe: `numkeys`
(keys from numbers/booleans/null) prints nothing — T-480. Files: interp `__keys`, `js_keys` in runtime.c / jsrt.go /
js-rt.metajs; decide whether the order is applied at enumeration (cheap) or kept at insertion.

### T-482 js compiler: labeled break/continue out of `try`/`finally` inside a loop loops forever or fails to build · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`bl: for (var i=0;i<3;i++){ try { if (i===1) break bl; lt.push("t"+i); } finally { lt.push("f"+i); } }` → interp and node
`t0,f0,f1`; llvm.Run hits the step brake (endless loop); `-exe` fails to link: unresolved `js_jsxcall` (finder also saw
`use of undefined value '%55'` in a larger module). `continue label` from an inner loop through a finally is the same.
Unlabelled break/continue through finally is fine. File: js-to-llvm-ir.abnf (labels crossing the control-signal path;
check typescript-to-llvm-ir.abnf, and why the module references an extern nothing defines).

### T-483 js compiler: `for-of` over an array mutated during iteration uses the starting length · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`a=[1,2,3,4]; for (x of a) { out.push(x); if (x===2) a.shift(); }` → run/native `1,2,4,undefined`, interp/node `1,2,4`;
pop while iterating `[1,2,3]` → `1,2,undefined` vs `1,2`; push while iterating `[1,2]` → `1,2` vs `1,2,10,20`. The array
iterator re-reads `length` every step. File: js-to-llvm-ir.abnf (array fast path of for-of), typescript likewise.

### T-484 js: string `<`/`>` and default `sort()` compare code points, not UTF-16 code units · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
s = ["", "\ud800", "\udc00x", "𝄞", "￿"]: node orders by code units, so `"𝄞" < ""`. run, native and
interp -frozen: `"𝄞" < ""` false; default sort `d800,e000,ffff,d834` (node `d800,d834,e000,ffff`). Interp under goja
gets astral right but a lone surrogate wrong (`"" < "\ud800"` true; sort `e000,d800,ffff,d834`). Files: `jsCompare` in
abnf/jsrt.go (~2333, Go string `<`), `js_compare` in runtime.c (~2881), the interp sort comparator. Check which other
languages share these comparators before changing them (java/kotlin/csharp/dart compare UTF-16 too; python/go/swift do not).

### T-485 js: the Go twin's `Math.max`/`Math.min` drop NaN next to ±Infinity · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`Math.max(NaN, Infinity)`, `Math.min(-Infinity, NaN)`, `Math.max.apply(null,[1,NaN,Infinity])`: interp (goja and -frozen)
and llvm.Run `Infinity`/`-Infinity`, native and node `NaN`. abnf/jsrt.go ~17604 folds with Go's `math.Max`/`math.Min`,
which define Max(x,+Inf)=+Inf even for NaN x. Check go's min/max (T-470) and other languages that route through it.

### T-487 js interp (goja): ToInt32/ToUint32 of |x| ≥ 2^63 saturates · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`x = 1e21`: interp `x|0 = -1`, `x>>>0 = 4294967295`, `~x = 0`, `1<<x = -2147483648`; run, native, interp -frozen and node
`-559939584 3735027712 559939583 1`. 2^63 and 2^64+2^33 also wrong. Cause: third_party/goja/runtime.go:946 and :961
`int32(int64(f))` — Go saturates on arm64. A goja patch like the ftoa one (manual ch. 4); add a cross-host pin test.

### T-488 js: TypeError/RangeError raised by runtime builtins is uncatchable; `repeat(Infinity)` hangs or panics · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`try { "a".repeat(-1) } catch(e) {...}` and `try { [].reduce((a,b)=>a+b) } catch(e)`: every leg ends the program with
`js runtime error: Invalid count value: -1` / `Reduce of empty array with no initial value`, exit 1; node catches a
RangeError / TypeError. `"ab".repeat(Infinity)` in a try: interp and native run until killed, run panics in Go
(`strings: Repeat output length overflow`); node RangeError. `new Array(-1)` is already catchable — copy its path.
Files: the builtin error raise in jsrt.go / runtime.c / js-rt.metajs / js-interpreter.abnf.

### T-489 js: fromIndex and separator arguments of includes / lastIndexOf / join · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`arr=[1,2,1]`: `arr.includes(1,5)`, `arr.includes(1,3)` → true (node false: fromIndex ignored); `arr.lastIndexOf(1,-5)` → 2
(node -1); `"abcabc".lastIndexOf("a",-1)` → 3 (node 0, also for -5); `[1,2].join(null)` → `1,2` (node `1null2`). All legs.

### T-490 js: regex `replace` replacement patterns `` $` `` `$'` and literal `$n`/`$<x>` · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`"xabcx".replace(/b/, "$\`")` → `xa$\`cx` (node `xaxacx`); `"$'"` → `xa$'cx` (node `xacxcx`); finder also: `"$<x>"` with no named
groups → `xacx` (node `xa$<x>cx`), `"$2"` with no group 2 → `xacx` (node `xa$2cx`), `"$10"` → `xa0cx` (node `xa$10cx`). The
string-pattern form is right. All legs. Files: the regex replace expander (lib/regex.js and its twins).

### T-491 js interp: index arguments are not ToIntegerOrInfinity'd (NaN, fractions) · P1 · JS · IN PROGRESS (JS worker, round 1) [V]
`"abc".at(NaN)`, `"abc".substr(NaN,1)`, `[5,6].at(NaN)` → interp `""`, `""`, `undefined`; run/native/node `a`, `a`, `5`. For
`-0.5`: interp `charAt ""`, `charCodeAt NaN`, `at "c"`, `substr "c"`, `[5,6].at → 6`; node `a 97 a a 5`.
`"abc".indexOf("c", NaN)` → NaN (node 2). File: js-interpreter.abnf string/array builtins.

---

# JV — Java

### T-464 java: interface default overload hiding; functional-interface overload choice; Object[] element types · P2 · JV · OPEN [V]
javac/java 24: (a) a class overriding one overload of an interface default hides the other overloads under dynamic
dispatch (`q.describe()` answers `square 2.25null`, java `area 2.25`) — overload candidates must walk the interfaces.
(b) overload choice treats a functional-interface parameter as accepting any reference (`kind(Shape)` vs `kind(Object)` with
a String picks Shape; java Object). (c) one-dimensional Object[] stays untagged: `new Object[]{"a"}` reports String[],
`Arrays.copyOf(s, 3)` Object[], `toArray()` String[]; needs Class.getComponentType / Array.newInstance in all engines.
(d) `thenComparing(Function)` (one-arg key form) not provided — indistinguishable from the Comparator form at run time.

### T-494 java: null dereference and unboxing never raise a catchable NullPointerException · P1 · JV · OPEN [V]
18 cases, each in `try { ... } catch (NullPointerException e)`, operands from `static Object[] nulls = {null}`; java prints
"k NPE" for all. Ours: `b.v`, `b.get()`, `a.length`, `s.length()`, `l.size()`, `b.v = 5`, `s.toCharArray()`, for-each over a
null List ABORT the program (interp "java interpreter error: field .v of null", run/native "js runtime error: member 'v' of
null"); silent wrong values: `(Integer)null + 1` → 1, switch on a null String → default, `Integer i = null; i == 3` → false,
`if ((Boolean)null)` → else, `o.hashCode()` → 0, `throw null` caught as "other"; halves disagree: `s.equals("x")` (interp
aborts, compiled false), `o.toString()` (aborts / "null"), `a[0] = 1` on a null int[] (interp aborts, compiled NPE — right),
`Double d = null; double e = d` (interp null, compiled 0.0). Also `int v = new HashMap<String,Integer>().get("q")` → null and
`"abc".compareTo(null)` → -13 (java NPE). Files: member/call/field paths in java-interpreter.abnf (core.getIndex ~6283
already raises), java-to-llvm-ir.abnf, the twin, lib/java-rt.metajs unboxing.

### T-495 java: casts never throw ClassCastException · P1 · JV · OPEN [V]
`Object o = "x"; Integer i = (Integer) o; System.out.println(i);` prints `x` in all legs; java throws `ClassCastException:
class java.lang.String cannot be cast to class java.lang.Integer (...)`; `(String)` of an Integer is silent too. Files: cast
lowering in both java grammars (the value model carries the type, manual 7.15).

### T-496 java: the conditional operator ignores binary numeric promotion (JLS 15.25) · P1 · JV · OPEN [V]
t from `boolean[]{true,false}`: `t ? cv : iv` (char/int) prints `b` (java 98); `t ? iv : 2.5` → 7 (7.0); `t ? boxed : 1.5` → 4
(4.0); `Object o = t ? iv : 2.0` → 7 (7.0); `t ? 1f : iv` → 7 (7.0); `!t ? 1.0 : iv` → 7 (7.0); `!t ? 'x' : 0` → `0` (java
char NUL); `true ? 1 : 'a'` → 1 (java char U+0001). All legs. Assignment conversion is right. Files: conditional typing in
both java grammars (java-decltypes.js).

### T-497 java interp: a static field initialised with a prelude class (HashMap, StringBuilder) fails · P1 · JV · OPEN [V]
`public class Main { static Map<String,Integer> mp = new HashMap<>(); ... }` → interp `java interpreter error: unknown class:
HashMap`, exit 1; run, native, java `[1] {k=2}`. Same for `static StringBuilder sb2 = new StringBuilder("o")` in any class;
`static List<Integer> lst = new ArrayList<>()` works. Static initialisers run before the java-util.java prelude is installed.
File: java-interpreter.abnf (static-init order vs prelude install).

### T-498 java: try-with-resources lets close()'s exception replace the body's · P1 · JV · OPEN [V]
`try (Res r1 = new Res("bad")) { throw new RuntimeException("body-ex"); } catch (RuntimeException e) { println("caught " + e); }`
with close() throwing `IllegalStateException("close-bad")`: all legs `caught java.lang.IllegalStateException: close-bad`;
java `caught java.lang.RuntimeException: body-ex` with close-bad suppressed (`getSuppressed` is also missing). Files:
try-with-resources lowering in both grammars.

### T-499 java: Formatter exceptions are uncatchable; `%#.0f` drops the point · P1 · JV · OPEN [V]
`try { String.format("%d", v) } catch (Exception e)` with v = "str" or 3.0: every leg ends the program ("java interpreter
error: IllegalFormatConversionException: d != str", compiled "js runtime error: ..."); java catches
`IllegalFormatConversionException: d != java.lang.String` (note the class name in the message). Same for
MissingFormatArgument / UnknownFormatConversion; java also has IllegalFormatPrecision (`%.2d`), MissingFormatWidth (`%-d`),
IllegalFormatCodePoint (`%c` 0x110000). `String.format("%#.0f", 2.5)` → `2` (java `2.`). Cause: `jxkFail` in
languages/lib/java-strings.js (~269, ~333) is a host failure, not a Java throw; abnf/jsrtjavastr.go and java-rt mirror it.

### T-500 java compiler: `(char)` of a double outside the int range wraps instead of saturating first · P1 · JV · OPEN [V]
`(int)(char) d` for d in {1e10, -1e10, 3e9, -3e9}: interp and java `65535 0 65535 0`; run and native `58368 7168 24064 41472`.
JLS 5.1.3 narrows double → int (saturating) → char. Files: java-to-llvm-ir.abnf ~3309 (`js_jvchar`), its twin, java-rt.metajs.

### T-501 java: `StringBuilder.append(char[])` appends the array identity · P1 · JV · OPEN [V]
`sb.append(new char[]{'x','y'})` → `...[C@1a2b3c01...`, java `...xy...` (all legs; `String.valueOf(char[])` and
`new String(char[])` are right). File: languages/lib/java-lang.java StringBuilder.append.

### T-502 java: varargs forwarding and a homogeneous Object[] break String.format · P1 · JV · OPEN [V]
Symptom of T-464(c) that raises its priority: `Object[] one = {3}; String.format("%s", one)` → `[I@1a2b3c01` (java `3`);
`static String g(String fmt, Object... xs) { return String.format(fmt, xs); }`: `g("%d", 8)` aborts
`IllegalFormatConversionException: d != obj`, `g("%s|%s", 8, 9)` aborts MissingFormatArgument (java `8`, `8|9`). Mixed element
types work. Cause: the array is tagged from its CONTENTS (Object[] of ints becomes `[I`) and `jxFormat`'s spread test
(`k1 == "L" || "["`, java-interpreter.abnf ~3681) rejects it. The forwarding idiom is very common. Fix with T-464(c).

### T-503 java: Throwable.toString ignores an overridden getMessage(); message and Unicode details · P3 · JV · OPEN [V]
`class Sub extends MyEx { public String getMessage() { return "sub-msg"; } }`, `println(e)` → `Sub` (java `Sub: sub-msg`).
`Integer.parseInt(null)` → `For input string: "null"` (java `Cannot parse null string`); `List.of(1,2).get(5)` → `Index 5 out
of bounds for length 2` (java `Index: 5 Size: 2`); `"a😀".compareToIgnoreCase("a￿")` → -10178 (java 62977, code points
since JDK 16); a lone surrogate printed → U+FFFD (java `?`). All legs.

### T-504 java: Integer identity outside the -128..127 cache · P3 · JV · OPEN [V finder only]
`Integer a3 = 128, a4 = 128; a3 == a4` → true (java false; also -129). Same value-model class as T-256 (String identity,
DECLINED): decide whether it is declined for the same reason before working it.

### T-506 java: commonly hit missing API · P3 · JV · OPEN [V finder only]
`System.out.printf`, `java.util.Collections`, `Objects`, `TreeMap`, `ArrayDeque`, `Throwable.getCause` and the (msg, cause)
constructor are what real programs hit most; also String.formatted/regionMatches/codePointCount/codePoints/chars/lines,
`%a`/`%h`, Integer.parseUnsignedInt/toUnsignedString/divideUnsigned/reverse/reverseBytes/rotateLeft,
Math.multiplyHigh/ulp/IEEEremainder, Double.hashCode/doubleToLongBits, Float.floatToIntBits, Character.toChars, ArrayList
Iterator.remove/replaceAll/subList, HashMap.computeIfPresent.

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

### T-562 go: `[]byte(s)[i]` has dynamic type int; `cap()` of a channel · P2 · GO · OPEN [V]
`b := []byte("az"); fmt.Printf("%T", b[0])` → all legs `int` (go `uint8`): `[]byte` elements are held as code points, so they
carry no sized type. `c := make(chan int, 2); fmt.Println(cap(c))` → interp `<nil>`, llvm.Run and native abort "len() of a
object" (go 2): the buffer size of `make(chan T, n)` is not recorded where `cap` looks.

### T-563 go: `fmt.Fprint*` / `os.Stdout` / io.Writer, and the `math` package · P3 · GO · OPEN [V]
`fmt.Fprintf(&sb, ...)`, `fmt.Fprintln(os.Stdout, ...)` unsupported in every engine (no io.Writer model, no os.Stdout).
`math.Sqrt` etc. unknown in every engine; needs correctly rounded primitives in all three engines, not a partial package.

### T-564 go native: `s[i]` on a string converts the whole string to bytes per index · P2 · GO · OPEN [V]
A loop reading `s[i]` over a 10,000-byte string is quadratic natively (`goBytes` of the whole string for every index): 1,430 × 50
passes took 581 s. Cache the byte form, or index the UTF-8 directly in lib/go-rt.metajs.

---

# SW — Swift  (oracle: swiftc 6.1.2)

### T-507 swift: a literal typed by context is not adopted, so Double arithmetic runs as Int · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
`let t1 = flag[1] ? 1.5 : 3; print(t1 / 2)` → all legs `1`, swiftc `1.5`. Same for `[1.5, 3][1] / 2`, `(nilDouble ?? 3) / 2`,
`enum K { case b(Double) }; K.b(3)` payload `/ 2`, `["a": 1.5, "b": 3]["b"]! / 2`. Printing shows it too: `[1.5, 2]` (swiftc
`[1.5, 2.0]`), `[2, 1.5]` (`[2.0, 1.5]`), `K.b(x: 2)` (`b(x: 2.0)`), `Double(s) ?? -999` → `-999` (`-999.0`), `max(1, v)` with v
Double → `1` (`1.0`). Already right: annotated `let`, struct field, default argument, return value, `reduce(0, +)`, typed
closure, tuple. Missing contexts: ternary arms, heterogeneous array/dictionary literals, the RHS of `??`, enum payloads,
`min`/`max` arguments. Files: swAdopt* in swift-interpreter.abnf and swift-to-llvm-ir.abnf.

### T-508 swift: `rounded(_:)` ignores its rule; `.sign` is never `.minus` · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
d from a [Double]: `d.rounded(.down/.towardZero/.toNearestOrEven/.up/.awayFromZero)` all return half-away rounding:
1.5 → down/tz 2.0 (swiftc 1.0); 2.5 → even/down/tz 3.0 (2.0); -2.5 → even/up/tz -3.0 (-2.0); 1e-05 → up/afz 0.0 (1.0);
4.35 → up/afz 4.0 (5.0). `(-0.0).sign == .minus`, `(-2.5).sign == .minus` → false (true). Files: swift-interpreter.abnf ~4071,
lib/swift-rt.metajs ~2857 (`swRound`), abnf/jsrtswift.go ~1309 — all return `swRoundHalfAway` regardless of the argument;
`sign` has no handler (the silent-nil member read, T-514).

### T-509 swift: `init(clamping:)`/`init(exactly:)` truncate; `radix:` is ignored · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
Operands from an [Int]: `UInt8(clamping: 256)` → 0 (255), `UInt8(clamping: -1)` → 255 (0), `Int8(clamping: 255)` → -1 (127),
`Int8(clamping: -129)` → 127 (-128); `UInt8(exactly: 256)` → 0 (nil), `Int8(exactly: 255)` → -1 (nil), `UInt8(exactly: -1)` → 255
(nil); finder: `Int64(exactly: 3.9)` → 3 (nil). `Int("ff", radix: 16) ?? -1` → -1 (255), `Int("-101", radix: 2)` → -101 (-5),
`Int("z", radix: 36)` → nil (35), `String(42, radix: 16)` → 42 (2a), `String(-255, radix: 2)` → -255 (-11111111); finder:
`uppercase: true` ignored. All legs. Files: integer-init path near swIntBounds (swift-interpreter.abnf ~2990),
swift-to-llvm-ir.abnf, lib/swift-rt.metajs, abnf/jsrtswift.go.

### T-510 swift: `for v in arr` does not iterate a copy; interp and compiler disagree · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
`var it = [1,2,3]; for v in it { if v == 1 { it.append(10) }; if v == 2 { it.removeAll() }; print("iter \(v) count=\(it.count)") }`
then `var d = [5,6]; for v in d { d[1] = 99; print("d \(v)") }`: swiftc `iter 1`, `iter 2`, `iter 3 count=0`, `d 5`, `d 6`;
interp `iter 1`, `iter 2`, `iter nil count=0`, `d 5`, `d 99` (original count, live elements); run/native stop after `iter 2`
and print `d 99` (live array; finder: an append is iterated). Value semantics (copy-at-store) is not applied to the for-in
sequence operand. Files: for-in lowering in both swift grammars.

### T-511 swift: array subscript — a negative index wraps; interp grows the array on a write past the end · P2 · SW · IN PROGRESS (SW worker, round 2) [V]
`var a = [1,2]; let idx = [2,5,-1]; print(a[idx[2]]); a[idx[2]] = 9; print(a)` → all legs `2`, `[1, 9]`, exit 0; swiftc
`Fatal error: Index out of range`, exit 133. Finder: `a[2] = 9` / `a[5] = 9` → interp `[1, 2, 9]` / `[1, 2, nil, nil, nil, 9]`,
run `Fatal error: Error raised at top level: IndexError`, native `list assignment index out of range` (python's setter
wording). A positive out-of-range READ is right. Files: swift subscript get/set in both grammars (the compiled write goes
through a python-style setter).

### T-512 swift: `switch` patterns on `Int.min`/`Int.max` · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
n from `[Int.min, -5, Int.max]`: `case Int.min:` never matches (all legs `other`, swiftc `min`); `case Int.max:` never matches
(swiftc `max`); `case ..<0:` with Int.min → interp `other`, run/native and swiftc `neg`; `case 0...:` after `case Int.max` →
interp `other`, run/native `nonneg`, swiftc `max`. `n == Int.min` and `n < 0` are right. Files: matchPat / emitMatchPat in the
two swift grammars (the sized-int box is not compared via the `==` arm; interp range patterns do not unbox).

### T-513 swift: grapheme clusters — ZWJ only in the interpreter; RI pairs, CR LF, emoji modifiers, Hangul never; prefix/contains/`<` not grapheme-aware · P1 · SW · IN PROGRESS (SW worker, round 2) [V]
`.count`: 👨‍👩‍👧 interp 1, run/native 5, swiftc 1 (also `first` and `reversed()` split it); 🏳️‍🌈 interp 1, compiled 3. All legs:
🇩🇪🇫🇷 4 (2), `"\r\n"` 2 (1), 👍🏽 2 (1), Hangul L+V+T 3 (1). `"e\u{301}".hasPrefix("e")` and `.contains("e")` true (false);
`"e\u{301}" < "f"` true (false), `"e\u{301}" < "\u{e9}"` true (false — they are equal; `==` already uses canonical
equivalence). The interpreter's `swGraphemeCount`/`swGraphemes` (~2456) has the 0x200D arm; the compiler's `rtGSplit`
(swift-to-llvm-ir.abnf ~3573-3720) and `sw_gcount` lack it. Implement UAX #29 once (shared), not per site.

### T-514 swift: an unimplemented member read answers nil silently; `?? default` then disagrees between legs · P2 · SW · IN PROGRESS (SW worker, round 2) [V]
`Character` `.isLetter/.isUppercase/.isNumber/.isWhitespace/.asciiValue/.wholeNumberValue/.isASCII` all print `nil` in every
leg (swiftc real values, e.g. `[a] letter=true upper=false num=false ws=false ascii=97 whole=-1 isASCII=true`);
`ch.asciiValue ?? 0` → interp 0, run/native `nil` (`undefined ?? 0` does not fire in the compiled legs). Also `Double.sign`
(T-508) and `Dir.allCases` (`nil`; `.count` → "field .count of nil"). Unknown METHODS fail loudly; unknown PROPERTIES must
too, and these Character properties should be implemented. Files: member-read fallback in both swift grammars.

### T-515 swift: debug-description escapes and trap wording · P3 · SW · IN PROGRESS (SW worker, round 2) [V finder only]
`print(["bel\u{7}", "del\u{7F}"])` prints raw control bytes (swiftc `"bel\u{07}"`, `"del\u{7F}"`); a Character `'` in an
array prints `"'"` (swiftc `"\'"`); `String(reflecting: "s")` → `s` (`"s"`); `a % 0` → `Fatal error: Division by zero` (swiftc
`... in remainder operation`).

### T-516 swift: missing features that fail loudly · P3 · SW · IN PROGRESS (SW worker, round 2) [V finder only]
`s.utf16.count` ("field .count of nil"); Range `.map/.filter/.reduce/.contains`; `Int.signum()`, `isMultiple(of:)`,
`quotientAndRemainder(dividingBy:)`; `Double.remainder(dividingBy:)`; `Optional.map` on a `firstIndex(of:)` result ("method
call 'map' on a number"); `(..<0).contains(n)` does not parse.

---

# DA — Dart  (no dart here: spec-only)

---

# PHP — PHP  (no php here: spec-only, php.net / php-src behaviour)

### T-463 php: $this outside object context; bind/call on non-literal closures; binding warnings; S63T var_dump id divergence · P2 · PHP · OPEN [V]
php-src semantics: (a) `$this` in a static method or static closure should raise Error "Using $this when not in object
context"; the interpreter stops with "unknown name: $this" or reads null, the compiler aborts "member 'x' of undefined".
(b) `bindTo`/`call` on a non-literal closure (first-class callable, fromCallable result) aborts "not modelled"
(closure_call_bind). (c) binding warnings are not printed; "Cannot unbind $this of closure using $this" is not enforced.
(d) compiled `$closure->__invoke($n)` passes a by-reference parameter by value (closure_invoke_case_insensitive).
(e) dynamic calls to an inaccessible method do not route to __call/__callStatic. (f) unverified: `call_user_func(['A','s'])`
from a compatible B instance — php-src suggests static:: is B, ours gives A. (g) the full ratchet's var_dump of S63T shows
#470 interp vs #469 compiled (object-id model; the suite does not compare it — check why the s63dump comparison misses it).

---

# LUA — Lua  (oracle: lua 5.5 binary, suite is 5.4)

### T-532 lua: a closure made in a `for`/`while` body captures one shared variable, not a fresh one per iteration · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
`for i = 1, N[1] do fs[i] = function() return i end end` → all legs `3 3 3`, lua `1 2 3`; a `local c = i * 10` in the body →
`30 30 30` (lua `10 20 30`); `for k, v in ipairs({"a","b","c"})` → `3c 3c 3c` (lua `1a 2b 3c`); a `local` in a `while` body →
`3 3 3` (lua `1 2 3`). `repeat` bodies, `do ... end` blocks and goto loops are right. The same class as js T-478. The
ratchet's SECTION 18 has no loop-closure check. Files: loop-body scoping in lua-interpreter.abnf and lua-to-llvm-ir.abnf.

### T-533 lua: multiple assignment evaluates an indexed target's table and key after assigning · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
Lua manual §3.3.3's own example: `local i, a = V[1], {}; i, a[i] = i + 1, V[2]` → all legs `a[1]=nil a[2]=20`, lua `a[1]=20`.
`cur, cur.x = u, 5` → the field lands on `u` (lua: on the old `cur`). `idx, f[idx], f[idx+1] = 3, "a", "b"` → `f[3], f[4]` set
(lua `f[1], f[2]`). Side-effect order `q[k(1)], q[k(2)] = k(3), k(4)` logs `3412` (lua `1234`; undefined by the manual, a
symptom only). The same class as go T-466. Files: the assignment emitters in both lua grammars.

### T-534 lua: integer-vs-float comparison is done in doubles, wrong past 2^53 · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
Operands from arrays: `9007199254740993 == 2^53` → true (lua false); `<`, `<=`, `>` likewise wrong (`true false true false`, lua
`false false false true`); finder: `math.maxinteger` vs `2^63` and `math.max(math.maxinteger, 2^63)` → `9223372036854775807`
(lua `9.2233720368547758e+18`). Integer-integer comparison and table keys at 2^53/2^53+1 are right. Likely `js_lueq` /
`js_lucmp` fast path in lib/lua-rt.metajs (~752-790), their twin and interpreter counterparts: need Lua's exact int/float
rule (luaV_equalobj / LTnum).

### T-535 lua: `t[nil] = v` and `t[NaN] = v` are accepted; `pairs` then counts an unreadable entry · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
`k = nil` (and `0/0`): `pcall(function() q[k] = 1 end)` → all legs `true`, `q[k]` nil, `pairs` count 1; lua `false ...: table
index is nil` (NaN: "table index is NaN"), count 0. Also `{[k] = 2}`, `rawset({}, k, 3)` and `q[nil] = nil` are accepted (lua
errors). Files: table-store paths in lib/lua-rt.metajs, the twin, the interpreter's index assignment and table constructor.

### T-536 lua native: `math.max`/`math.min` die when a float beats an int argument · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
`A = {1, 2.0, 2, 1.0, 3}`: `pcall(math.min, A[3], A[4])`, `pcall(math.max, A[2], A[1])` → interp, run and lua `true 1.0`,
`true 2.0`; native no output, exit 1, uncatchable `js runtime error: MetaJS: variable 'best' has type number and cannot hold a
object`. lib/lua-rt.metajs ~1059-1067: `var best = args[0]` pins the type (manual ch. 4 "MetaJS is typed").

### T-537 lua: numeric `for` prep — an int start with a float step is an integer on the first pass; a zero step does not raise · P1 · LUA · IN PROGRESS (LUA worker, round 2) [V]
`for i = P[1], P[2], P[3]` with 1, 3, 0.5 → first pass `1 integer` (lua `1.0 float`; later passes agree). Finder: a 0 or 0.0
step → the loop silently does not run (lua `'for' step is zero`); a table limit → `attempt to compare number with table` (lua
`bad 'for' limit (number expected, got table)`). Lua 5.5-only behaviour (string bounds converted, read-only loop variable) is
informational — the suite targets 5.4. Files: numeric-for emitters in both lua grammars.

### T-538 lua: no proper tail calls · P2 · LUA · IN PROGRESS (LUA worker, round 2) [V finder only]
`local function tc(n) if n == 0 then return "done" end return tc(n - 1) end` with n = 100000: interp, run and lua `done`; native
SIGSEGV (exit 139). n = 1,000,000: interp Go fatal "goroutine stack exceeds 1000000000-byte limit", run hits the step brake;
lua runs it. Lua guarantees unbounded tail calls (§3.4.10); the ratchet only checks depth 200. The catchable "stack
overflow" half is T-505.

### T-539 lua: `tostring` of a table or function has no identity · P2 · LUA · IN PROGRESS (LUA worker, round 2) [V finder only]
`tostring({})` → `table` (lua `table: 0x...`), so `tostring({}) ~= tostring({})` is false (lua true) and code keying on
tostring breaks. Needs an identity value (see how java's `[C@1a2b3c01` and php's object ids are produced).

### T-540 lua: string library — `%z` class missing; `%s` with width/precision accepts `\0`; `\u{...}` above 0x1FFFFF · P3 · LUA · IN PROGRESS (LUA worker, round 2) [V]
`string.find("a\0b", "%z")` → nil (lua `2 2`; `[%z]` and gsub likewise, and ours makes `%z` match a literal z): add
`case 'z': c == 0` to lsMatchClass in abnf/jsrtluastr.go (~192) and lua-strlib.js (~59). `string.format("%5.2s", "a\0b")`
formats (lua `bad argument #2 to 'format' (string contains zeros)`; also `%.0s`, `%-8s`). `("\u{200000}"):byte(1,-1)` → 4 values
(lua 5 bytes 248 136 128 128 128); `\u{4000000}`, `\u{7FFFFFFF}` → 4 values with one above 255 (lua 6 bytes). Finder:
`pcall(xpcall)` message `(value expected)` vs lua `(function expected, got no value)`.

---

# SH — bash / batch  (oracle: /opt/homebrew/bin/bash 5.3; no cmd.exe)

### T-517 bash: indexed-array subscripts are not arithmetic; array elements inside `$(( ))`; negative subscripts · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`a=(a0 a1 a2 a3 a4); i=2; echo "${a[i]} ${a[i+1]} ${a[$i+1]}"; a[i+2]=W; echo "${a[*]}"` → all legs `a0 a0 a2` / `W a1 a2 a3 a4`,
bash `a2 a3 a3` / `a0 a1 a2 a3 W` (the subscript is literal text through strToInt, though bash-interpreter.abnf:456 says
"read as an arithmetic index"; `arr[i++]=X` and `${arr[n++]}` lose the side effect). `nums=(3 4 5); echo "sum=$(( nums[0] +
nums[2] ))"; echo after` → interp `command not found: nums[0]` and the WHOLE program exits 127; run/native
`sum=<path>: command not found: nums[0]` then `after`; bash `sum=8`, `after` (also `(( m[p]++ ))`). `a=(p q r s); echo
"[${a[-1]}] [${a[-2]}]"; a[-1]=S; echo "${!a[*]} = ${a[*]}"` → `[] []`, -1 stored as a new key (interp `-1 0 1 2 3 = S p q r s`,
compiled `0 1 2 3 -1 = p q r s S`); bash `[s] [r]` / `0 1 2 3 = p q r S`. Files: IdxWord / arr* paths in
bash-interpreter.abnf, subscript normalisation and the arithmetic grammar in bash-to-llvm-ir.abnf, bash-rt.c rt_arr_*.

### T-518 bash: sparse indexed arrays — compiler keeps insertion order; `([k]=v w)` continues from the wrong index; `${a[@]:off:len}` counts positions · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`a=(); a[5]=x; a[2]=y; echo "${!a[*]} = ${a[*]}"` → interp and bash `2 5 = y x`, run/native `5 2 = x y`.
`b=([3]=c [1]=a b)` → interp `1 3 4 = a c b`, compiled `3 1 4 = c a b`, bash `1 2 3 = a b c` (an unindexed element takes the
last explicit index + 1). `sp[2]=x; sp[5]=y; sp[9]=z; echo "[${sp[*]:3}] [${sp[*]:1:1}] [${sp[*]: -4}]"` → all legs
`[] [y] []`, bash `[y z] [x] [z]` (offset = first index ≥ off; a negative offset is relative to max index + 1). Files:
bash-rt.c rt_arr_* key order, both grammars' compound assignment and array slicing.

### T-519 bash: quoted or escaped glob characters stay active in case patterns, `${v#p}`/`${v%p}`/`${v/p/r}`, and `=~` · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
For s in `x`, `*`: `case $s in '*') c=star ;; *) c=other ;; esac` → `star` for BOTH (bash `other` for x); `${s#\*}` on `*` →
`*` (bash empty); `p='?'; ${s%"$p"}` → empty (bash keeps s); `${s/'*'/R}` on `x` → `R` (bash `x`). `[[ abc =~ "a.c" ]]` and
`[[ abc =~ a"."c ]]` → rc 0 (bash 1: a quoted part of the regex is literal). `[[ x == "$p" ]]` is already right — copy its
quoting-aware word building. All legs. Files: case / pattern-operand word building in both grammars, rt_glob /
rt_strip / rt_replace, the =~ regex builder.

### T-520 bash: INT32_MIN prints as `-`; the compiler's shift by ≥ 32 or negative is IR poison · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`v=(-2147483648 2147483647); m=${v[0]}; x=${v[1]}; echo "[$(( m ))] [$(( x + 1 ))] [$(( 1 << 31 ))]"` → all legs `[-] [-] [-]`;
rt_int2str (bash-rt.c ~425) and numToStr (bash-interpreter.abnf ~943) cannot negate INT_MIN; a later `printf %d` of it prints
0. Shifts `$(( 1 << s )) $(( -8 >> s ))` for s in 32, 33, -1: interp and native wrap mod 32 (`1`, `-8`; `2`, `-4`; `-`, `-1`),
llvm.Run gives `0 -1` / `0 -1` / `0 -1` — the emitted `shl`/`ashr` (bash-to-llvm-ir.abnf ~2306-2307) is unmasked, poison per
LangRef. Bash (64-bit): `4294967296 -1`, `8589934592 -1`, `-9223372036854775808 -1` — see T-521 for the width.

### T-521 bash: the arithmetic model is 32-bit, bash's is 64-bit · P2 · SH · IN PROGRESS (SH worker, round 1) [V]
`t=10; big=3000000000; h=65536` (from an array): `$(( t ** t )) $(( big + 0 )) $(( h * h ))` → all legs
`1410065408 -1294967296 0`, bash `10000000000 3000000000 4294967296`; literals above 2^31 wrap. The source says so
(bash-interpreter.abnf ~3308) but manual 7.9 records only the `printf %x` consequence. Decide: widen to i64 in both
halves and bash-rt.c, or record the whole model as a permanent divergence in 7.9.

### T-522 bash: arithmetic `&&`/`||` do not short-circuit; the compiler's `?:` evaluates both arms; a variable's value is not evaluated as an expression · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`z=0; echo "$(( z && (z = 9) )) z=$z"` → `0 z=9` (bash `0 z=0`); `$(( 1 || (z = 9) ))` → z=9 (bash 0). All legs.
`k=1; echo "$(( k == 1 ? k++ : k-- )) k=$k"` → interp and bash `1 k=2`, run/native `1 k=1` (a select instead of branches).
`r=a; a=7; echo "$(( r + 1 ))"` → `1` (bash 8: a name's value is itself evaluated); `e="1 + 2"; echo "$(( e * 2 )) $(( $e * 2 ))"`
→ `2 2` (bash `6 5`: `$e` is substituted textually and re-parsed). All legs. The last two meet manual 7.9's "the compiler
has no arithmetic evaluator at runtime"; the interpreter can do it, and the compiler's answer should at least be an error
rather than a silent 1. Files: arithmetic evaluation in both grammars.

### T-523 bash: `return N` is not reduced mod 256; the compiler ignores `;&` and leaves `$?` after a case · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`f(){ return 300; }; f; echo $?` → 300 (bash 44); `g(){ return -1; }` → -1 (bash 255). All legs.
`case "$1" in a) echo -n "A;" ;& b) echo -n "B;" ;; esac` with `a` → interp and bash `A;B;`, run/native `A;`.
`false; case x in y) ;; esac; echo $?` and `false; case x in x) ;; esac; echo $?` → interp and bash 0 0, compiled 1 1.
Files: return and case lowering in bash-to-llvm-ir.abnf; return in both grammars.

### T-524 bash: `[[ A -eq B ]]` operands are not arithmetic · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`[[ 2+3 -eq 5 ]]`, `[[ 010 -eq 8 ]]`, `x=4; [[ x -eq 4 ]]` → rc 1 1 1 in all legs, bash 0 0 0 (the numeric tests evaluate both
sides as arithmetic expressions; octal included). Same for -ne/-lt/-le/-gt/-ge. Files: [[ ]] numeric tests in both grammars.

### T-525 bash: `${N-word}`/`${N+word}` on positional parameters; `${!ref}` through an array element · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`f(){ echo "[${2-unset}] [${3:-dflt}] [${1+set}]"; }; f; f one` → interp `[unset] [dflt] []` twice (wrong for `${1+set}` when
$1 is set), run/native `[] [dflt] [set]` twice, bash `[unset] [dflt] []` / `[unset] [dflt] [set]`. `arr=(x y z); r1='arr[1]';
r2=arr; echo "[${!r1}] [${!r2}]"` → interp `[] [x]`, compiled `[] []`, bash `[y] [x]`. Files: parameter expansion in both
grammars.

### T-526 bash: printf flags, operands and escapes · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
All legs: `%05d` of -7 → `000-7` (bash `-0007`); `%#05x` 255 → `00xff` (`0x0ff`); `%+d`, `% d` ignored (`+5`, ` 5`); `%-+4d` 3 →
`3   ` (`+3  `); `"'A"` / `"'a"` operands → 0 0 (65 97); `%*d` / `%.*s` unsupported (`<5d><42s>`, bash `<   42><ab>`);
`printf -- fmt` treats `--` as the format. Finder also: escapes in the FORMAT (`\101`, `\x41`) left literal; `%b` of `\0101`
(interp ^H1, compiled empty, bash A); `%b`'s `\c` not honoured; the compiler's `%b` ignores `\xHH`, `\NNN` and unknown escapes
(interp right); rc stays 0 on an invalid number. Files: printfBuiltin (bash-interpreter.abnf ~3305), printf lowering in
bash-to-llvm-ir.abnf, bash-rt.c rt_pad / rt_unescape.

### T-527 bash: `${v/p/r}` details — `&` in the replacement, an empty subject, extglob without shopt; the compiler's `${v%?}` splits UTF-8 · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`k=hello; echo "[${k/l/<&>}] [${k//l/&&}]"` → `[he<&>lo] [he&&&&o]` (bash 5.2+ patsub_replacement: `[he<l>lo] [hellllo]`).
`e=""; echo "[${e/*/R}] [${e//*/R}]"` → `[] []` (bash `[R] [R]`). `s=aXa; echo "[${s#@(a|X)}] [${s//@(a|X)/-}]"` → `[Xa] [---]`
(bash without `shopt -s extglob`: `[aXa] [aXa]`). These three in all legs. `v=hé; echo "[${v%?}]"` → interp and bash `[h]`,
run/native `[h\xC3]` (bash-rt.c rt_strip suffix branch; `${v#?}` is right).

### T-528 bash: `${x:?msg}` in a subshell, and silent arithmetic / substring errors · P1 · SH · IN PROGRESS (SH worker, round 1) [V]
`(: ${u:?custom}) 2>/dev/null; echo "rc=$?"; echo after` → interp prints `bash interpreter error: u: custom` and the WHOLE
program exits 2; run/native `rc=0` (no error at all); bash `rc=1`, `after`. `f(){ echo "r=$(( 7 / d ))"; echo after; }` with
d=0, `$(( 2 ** -1 ))`, `${s:1:-3}` on `abc` → all legs answer `r=0`, `pow=1`, `sub=[]` and carry on; bash reports division by 0
/ negative exponent / substring expression < 0 and aborts the command (`st=1`). Files: error paths in both grammars and
bash-rt.c.

### T-529 bash: command substitution nests at most 8 deep natively · P2 · SH · IN PROGRESS (SH worker, round 1) [V]
`d(){ if [ "$1" -le 0 ]; then echo 0; return; fi; echo $(( 1 + $(d $(( $1 - 1 ))) )); }; echo $(d 6); echo $(d 7); echo $(d 8)`
→ interp and bash `6 7 8`; native `6 7` then `bash-rt: subshell nesting exhausted ... 8 of 8`, exit 70; llvm.Run no output at
all (rt_die's `write` is undefined under llvm.Run). Ordinary recursive functions (fib 10) hit it. `SSDEPTH 8` in
bash-rt.c ~3037: grow the save stack dynamically.

### T-531 bash compiler: `break N` beyond the loop depth is a compile error · P3 · SH · IN PROGRESS (SH worker, round 1) [V]
`for i in 1 2; do for j in 1 2; do break 5; done; echo never; done; echo "done rc=$?"` → interp and bash `done rc=0`; the
compiler stops with `break used outside a loop` (bash breaks every enclosing loop). Finder also hit compiler rejects
`unset "d[$((i+1))]"` ("unset needs a literal name") and a non-literal printf format.

---

# C — C subset

---

# CORE — MetaJS, the engines, shared runtime, tooling

### T-184 dart: `IndexError` vs `RangeError` text for `[1,2,3][7]` · P3 · DA · BLOCKED - needs a Dart VM to settle [U spec-only]
A reviewer argued the VM prints `RangeError (index): Invalid value: Not in inclusive
range 0..2: 7` (a plain RangeError); the current engines print the IndexError form
`RangeError (index): Index out of range: index should be less than 3: 7`, which is
believed to be the modern VM's. Leave as is unless a real dart run settles it.

### T-505 java (and probably every language): unbounded recursion kills the process instead of raising · P2 · CORE · OPEN [V finder only]
`static void rec(int n){ rec(n+1); }` inside `try { } catch (StackOverflowError e)`: interp and run die with Go "fatal
error: stack overflow" (exit 2), native SIGSEGV (exit 139); java catches it. Needs a depth limit in the engines; probe the
other languages' equivalents (python RecursionError, js RangeError "Maximum call stack size exceeded", kotlin, csharp).
Lua (reproduced by the lua finder): `pcall(nt, 1000000)` of a non-tail recursion → lua `false ...: stack overflow`
then `after`; interp and run die with Go's fatal stack overflow (exit 2), native SIGSEGV. One mechanism for every language.

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

**DECLINED after measurement.** php handle reuse after free (needs refcounts) · swift's module-qualified enum names in collections (`[o.Dir.up]` depends on swiftc's module name) · rbUserObj / rbIsClassObj frame elimination (no gain over 15 pairs) · a real suspension for the replay generators (impossible under the pinned goja; ~3,600 lines pure-MetaJS) · CPS coroutines in the emitter · full correct rounding of the transcendentals · a shared nominal type id on boxed values (cheap to store, but every consumer needs static types the grammars do not have; split per language) · matching Apple libm's last digit (ruby atan2: no portable algorithm reproduces it, 1.6% of pairs differ even from the correctly rounded value) · js sloppy-mode primitive `this` boxing (needs wrapper objects in every engine) · a generator's `finally` at collection (python needs refcount timing; no finalizers, manual 7.8) · `i in arr` after `delete` · iterator close on
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
