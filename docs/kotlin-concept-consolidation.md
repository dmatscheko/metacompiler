# Kotlin grammars: equivalent concepts expressed inconsistently

Analysis of `languages/kotlin-interpreter.abnf` and `languages/kotlin-to-llvm-ir.abnf`
("the Kotlin languages"). Each finding is a single concept that is written out more
than once, in slightly different ways, and can be collapsed to one canonical form.

The two files share their **syntax productions byte-for-byte** but deliberately keep
**separate tag code** (one walks/interprets, the other emits IR). So a within-file
duplication is a true smell; a cross-file duplication is expected for the *grammar*
but not for *identical Kotlin-local helper functions* that could live in one place.

Ground rules honoured by every collapse below:

- **Byte-identity is sacred.** `mec <grammar> <test> -q` must produce identical stdout
  under goja (default) and `-frozen`, and the interpreter's observable output must keep
  matching the compiler's. Every collapse is therefore either pure walk-time data
  restructuring (no change to emission order / thunk behaviour) or is validated against
  the 37-entry Kotlin matrix on **both** engines.
- **Adopt the best variant, don't pick arbitrarily.** Where the variants differ, the
  canonical form takes the most complete/correct behaviour (e.g. the loop context-save
  that also tracks `curFTag`).
- **No refreeze needed.** The frozen snapshot freezes only `metajs-to-llvm-ir.abnf`'s
  own scripts; the Kotlin grammars' tag scripts are compiled on the fly by the frozen
  MetaJS compiler, so editing them (or `interp-core.js`) is refreeze-free.

Legend: **[interp]** = kotlin-interpreter.abnf, **[comp]** = kotlin-to-llvm-ir.abnf.
Line numbers are pre-refactor.

---

## F1 — Member-list partitioning (props / methods / companion)

**Concept:** given the walked members of a class / object / object-expression /
companion, split them into property-initializers, methods, and (for a class) the
companion.

**Written 8 times, each slightly different:**

| site | file | start index | handles companion? | var names |
|---|---|---|---|---|
| `makeObjectExpr` :1330 | interp | 0 | no | props / methods |
| `buildClass` :1695 | interp | after cparams | **yes** | props / methods |
| `attachCompanion` :1736 | interp | 0 | no | cprops / cmethods |
| `buildObject` :1765 | interp | 1 (after name) | no | props / methods |
| `makeObjectExpr` :1342 | comp | 0 | no | props / methods |
| `makeClassStmt` :1877 | comp | after cparams | **yes** | props / methods |
| `emitCompanion` :1912 | comp | 0 | no | cprops / cmethods |
| `makeObjectStmt` :1945 | comp | 1 (after name) | no | props / methods |

Every copy carries the same predicate — `items[i].prop != undefined && items[i].init
!= undefined` → property, `items[i].method != undefined` → method — plus, in two of
them, `items[i].companion != undefined` → companion. The differences (start index,
whether companion is collected, `props` vs `cprops` naming, brace style) are incidental,
not semantic.

**Canonical form:** one `splitMembers(items, start)` per file returning
`{props, methods, companion}`. `companion` is `null` for the callers that never see one
(object / object-expression / companion bodies do not nest a companion), so returning
the field unconditionally is harmless. Pure data grouping done at walk time before any
side effect, so the resulting arrays are identical in content and order → byte-identical.

**Status: collapsed.**

---

## F2 — Consume the pending loop label

**Concept:** a loop emitter/runner takes the label a `Labeled` wrapper parked in
`pendingLabel`, then clears it so an inner loop does not re-consume it.

**Written 10 times, verbatim** — `var lbl = pendingLabel; pendingLabel = null` — in
`makeWhile`, `makeDo`, `makeForRange`, `makeForList`, `makeForDestruct`, in **both**
files ([interp] :1478 :1504 :1528 :1556 :1586; [comp] :1577 :1594 :1619 :1664 :1708).

**Canonical form:** `takeLabel()` (reads `pendingLabel`, nulls it, returns the value).
Identical values, identical timing → byte-identical.

**Status: collapsed.**

---

## F3 — Loop control-signal classification (interpreter)

**Concept:** after running a loop body once, decide what its return value means —
`BRK` → stop, `CONT` → next iteration, a `{contL}` for *this* loop's label → next
iteration, anything else → propagate out.

**Written 5 times** ([interp] `makeWhile` :1482, `makeDo` :1507, `makeForRange` :1541,
`makeForList` :1564, `makeForDestruct` :1593):

```js
if (r != undefined) {
    if (r === BRK) break
    if (r === CONT) continue
    if (lbl != null && r.contL === lbl) continue
    <propagate>   // while/do: `return r`  |  for-loops: `res = r; break`
}
```

The classification is identical; only the *propagate* tail differs (the `for` loops must
`scopes.pop()` before returning, so they set `res` and `break`). This is real
duplication, but the `break`/`continue` keywords act on the caller's own loop and cannot
be moved into a plain helper.

**Canonical form:** a `loopSignal(r, lbl)` helper that returns `"brk" | "cont" |
"stop"`; each caller keeps its two-line `if`/`if` for the keywords and its own
propagate tail. Reduces the 4-line classification to one call, and removes the risk of
the four copies drifting.

**Status: collapsed (interpreter only — the compiler routes break/continue through
`emitBreak`/`emitContinue` in compile-core and does not have this shape).**

---

## F4 — Frame invocation for closures (interpreter)

**Concept:** run a function/method/lambda body in a fresh scope frame: install the base
scope chain, optionally bind `this`, bind the parameters, run the frame hooks, run the
body, restore scopes, and unwrap an `{isRet}` result.

The shared `invokeBody` (interp-core.js:438) is the canonical version, but **three**
Kotlin-local helpers hand-roll their own copy because they each need a different base
chain or `this` policy:

| helper | base chain | `this` | arg offset |
|---|---|---|---|
| `invokeBody` (interp-core) | `[globalScope, {}]` | if `self != undefined` | 0 |
| `registerExtFun` :994 | `[globalScope, {}]` | **forced** (may be null) | 1 |
| `makeCapturedMethod` :1304 | `captured.slice(0)` + `{}` | forced | 1 |
| `makeLocalFunStmt` :1637 | `captured.slice(0)` + `{}` | none | 0 |

All four end identically: `framePush?` → `body()` → `framePop?` → restore →
`return (r && r.isRet) ? r.v : undefined`.

**Canonical form:** one Kotlin-local `runFrame(base, self, bindThis, params, args,
argOff, body)` that the three variants call with their base/`this`/offset. `invokeBody`
stays in the shared lib for the generic global-rooted case (`makeFunClosure` /
`makeMethodClosure` already delegate to it — they are not part of the smell). Removes
~40 lines of near-identical frame juggling and the risk of the copies drifting apart on
`framePush`/`isRet`.

**Status: collapsed (interpreter only — the compiler's equivalent is `emitFunc`, see
F9).**

---

## F5 — Object-descriptor + instance construction

**Concept:** build a class-style descriptor object (`__isclass`, `__name`, the method
closures), make an instance whose `__class` is that descriptor, then run the property
initializers with the instance bound to `this`.

**[comp]** — `makeObjectExpr` :1339 and `makeObjectStmt` :1941 are identical emission
except for three things: the `__name` string (`"<object>"` vs the real name), a trailing
`js_scope_decl name` (statement only), and the return (`{b, v: objV}` vs `b`).

**Canonical form:** a shared `emitObjectInstance(b, name, methods, props)` → `{b, v:
objV}`; `makeObjectExpr` returns it directly, `makeObjectStmt` calls it then declares the
name and returns the block. Identical emission order → byte-identical.

**[interp]** — `makeObjectExpr` :1327 and `buildObject` :1761 share the same shape but
diverge meaningfully: the object-*expression* captures the evaluation scope and rebuilds
the descriptor per evaluation (`makeCapturedMethod`), while the object-*declaration* is a
one-time singleton built at global scope (`makeMethodClosure`). The shared skeleton is
real but the divergence is semantic, so the interpreter pair is **documented, not
merged** (merging would need a scope-policy + method-maker parameter that leaks the
difference back out).

**Status: compiler pair collapsed; interpreter pair documented.**

---

## F6 — `extResolves` + builtin-method-name table (cross-file)

Both files build the **identical** `bmn` list and `builtinMethodNames` set and an
`extResolves(name)` with the same three-test body ([interp] :983-1018, [comp]
:845-857). The only real difference is the extension table consulted (`extFuns` holds
closures in the interpreter, `extFunNames` holds booleans in the compiler), so the first
test reads `extFuns["$"+name] == undefined` vs `extFunNames["$"+name] != true`.

**Canonical form:** the identical `bmn`/`builtinMethodNames` construction could be
shared, but `extResolves` genuinely differs (it reads a different table), so only a
fragment is common. **Left duplicated — see F11 for why a Kotlin-only shared file is
declined.**

---

## F7 — Labelled-return walk stack (cross-file, identical)

`var retLabels = []` + `retLblPush` / `retLblPop` / `retLblTop` are **byte-identical**
in both files and depend on nothing engine-specific.

**Canonical form:** could move to a shared file. **Left duplicated — see F11.**

---

## F8 — `kCharCode` (cross-file, identical)

The char-literal decoder `kCharCode(t)` is **byte-identical** in both files and is pure.

**Canonical form:** could move to a shared file. **Left duplicated — see F11.**

---

## F9 — `emitFunc` vs `emitCtor` (compiler)

**Concept:** open a new IR function `jsf_N`, save/replace the emit context (`curF`,
`curScopeV`, `loopStack`, and — the key point — `curFTag`), bind `this`/params from the
`args` handle, emit the body, restore the context.

`emitFunc` :1765 and `emitCtor` :1835 share this whole preamble/postamble, but
**`emitCtor` omits the `curFTag` save/set/restore** that `emitFunc` performs. `curFTag`
drives `inCtlBody()` (compile-core), which decides whether a `return`/`break`/`continue`
leaving a try-body emits a control signal instead of a `ret`/`br`. A constructor's
property initializer can contain a try-expression, so the omission is a latent
inconsistency: inside a ctor body `inCtlBody()` reads a stale tag.

A full merge is not clean — `emitCtor` interleaves `js_scope_decl` with `js_set` for
property-carrying ctor params and needs `f.Params[1]` (the args handle) that the generic
`body(b)` signature does not expose, so reusing `emitFunc`'s param loop would reorder the
emitted instructions and break byte-identity.

**Canonical form:** keep the two functions, but **align `emitCtor`'s context-save with
`emitFunc`'s** (add the `curFTag` save/set/restore — the better variant). This makes the
two consistent and fixes the latent staleness, and is output-neutral for all current
tests (no test puts a `try` in a ctor property initializer; verified by the matrix
staying byte-identical).

**Status: aligned (context-save), not merged.**

---

## F10 — The four zero-width peek guards (shared grammar)

`NotArrow`, `SameLine`, `NotParen`, `NotColon` ([interp] :379-426, [comp] :399-446) are
four `:script` productions sharing one shape: skip whitespace via `c.peek`, then return
an impossible token (forcing backtracking) when a specific next token is present.

- `NotParen` / `NotColon` are identical except the landing byte (`40` vs `58`) and the
  token name.
- `NotArrow` adds a two-byte check for `->`.
- `SameLine` is genuinely different: it skips **only** spaces/tabs and *fires* on a
  newline/`//`, because it is the one guard that cares about line boundaries.

Since 2026-07-10 the dialect has a native `!"token"` zero-width negative lookahead, so
`NotArrow`/`NotParen`/`NotColon` (pure "next token isn't X") are candidates to become
`!"->"`, `!"("`, `!":"` — eliminating three hand-written parse-time scripts. `SameLine`
must stay a `:script` (native lookahead skips the very newlines it inspects).

**Risk:** this changes the *shared* grammar and depends on `!"token"` matching the
guards' whitespace-skipping exactly; a mismatch would silently change what the grammar
accepts. **Documented as a recommended, separately-validated change; not bundled with
the tag-code collapses.** The two-VM architecture (parse-time `:script` runs in a
separate goja runtime from the walk-time start script) means the four scripts cannot
share a JS helper, so native lookahead is the only real collapse available here.

---

## F11 — A shared `lib/kotlin-common.js`? Considered and **declined**

After F1–F5, several Kotlin-local helpers are byte-identical across the two files and
pure: `splitMembers` (F1), `kCharCode` (F8), and the `retLabels` machinery (F7). They
*could* live once in a new `lib/kotlin-common.js` that each grammar `include()`s.

**Declined, deliberately**, because it would break the codebase's own convention: no
language pair here (java, python, go, c, …) has a `*-common.js` — each interpreter/
compiler pair keeps its language-specific helpers **duplicated and self-contained**, and
shares only the *generic* `interp-core.js` / `compile-core.js` across all languages.
These Kotlin helpers are Kotlin-specific (a companion-aware member split, Kotlin's
labelled-return stack, Kotlin char escapes), so they do not belong in the generic core
either. Introducing a Kotlin-only shared file to remove byte-identical duplication would
trade a small, intentional, harmless duplication for an architectural inconsistency with
every other language — a net loss by this task's own "consistency" yardstick.

The within-file collapses (F1–F5, F9) already removed the genuinely *inconsistent*
expressions. The remaining cross-file copies are identical and idiomatic; they are left
as-is on purpose.

---

## Summary

| # | concept | sites | scope | action |
|---|---|---|---|---|
| F1 | member partitioning | 8 | within-file ×2 | collapsed → `splitMembers` |
| F2 | consume pending label | 10 | within-file ×2 | collapsed → `takeLabel` |
| F3 | loop signal classification | 5 | within-file (interp) | collapsed → `loopSignal` |
| F4 | frame invocation | 3 (+1 lib) | within-file (interp) | collapsed → `runFrame` |
| F5 | object construction | 2+2 | within-file | comp merged; interp documented |
| F6 | `extResolves`/`bmn` | 2 | cross-file | documented (left duplicated, see F11) |
| F7 | `retLabels` stack | 2 | cross-file | documented (left duplicated, see F11) |
| F8 | `kCharCode` | 2 | cross-file | documented (left duplicated, see F11) |
| F9 | `emitFunc`/`emitCtor` | 2 | within-file (comp) | context-save aligned |
| F10 | peek guards | 4 | shared grammar | documented (native `!"token"`, own pass) |
| F11 | shared `kotlin-common.js` | — | cross-file | **declined** (breaks per-pair convention) |

**Collapsed this pass:** F1, F2, F3, F4, F5 (compiler pair), F9 — every genuinely
*inconsistent* within-file expression. Validated byte-identical on both engines (goja
and `-frozen`) after each step; the 37-entry Kotlin matrix stays 37/37 green.
