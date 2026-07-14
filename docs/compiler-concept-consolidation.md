# Compiler (`-to-llvm-ir`) grammars: equivalent concepts across languages

Cross-language pass over the 14 big-language LLVM-IR compilers that share
`lib/compile-core.js` (`c, csharp, dart, go, java, js, kotlin, lua, metajs, php, python,
ruby, swift, typescript`). Same method and goal as the interpreter pass
(`interpreter-concept-consolidation.md`): find one concept lowered more than once, across
different languages, in inconsistent ways, and collapse it into a single canonical
definition in the existing shared home (`compile-core.js`).

## The one structural difference from the interpreter pass — the frozen bootstrap

`compile-core.js` is **inlined into the frozen bootstrap**: `metajs-to-llvm-ir.abnf` is
the grammar that `-freeze` snapshots (`abnf/jsbootstrap.ll` + `abnf/jsagrammar.go`), and
its start-script lib pulls in `compile-core.js`. So — unlike `interp-core.js`, which is
not in the bootstrap — **any edit to `compile-core.js` (or to `metajs-to-llvm-ir.abnf`
itself) must be followed by a refreeze**:

```
./mec -freeze languages/metajs-to-llvm-ir.abnf && go build .
```

The refreeze regenerates the snapshot from current source (verified stable to a
fixpoint: a second freeze is byte-identical). The move is still behavior-preserving —
the `-frozen` leg produces the same IR as goja — but the committed snapshot has to be
regenerated so it corresponds to the committed source. (The visible cost is a large,
mechanical `jsbootstrap.ll` diff: SSA registers and block labels renumber when a lib
function relocates.)

---

## Collapsed — `makeCatch` → `compile-core.js`

Nine compilers lower `try { } catch (e) { }` and each defined the **byte-identical**
catch-clause packager:

```js
function makeCatch(items) {
    if (items.length > 1) { return {catchbody: items[1], catchname: items[0]} }
    return {catchbody: items[0], catchname: undefined}
}
```

Present in `csharp, dart, java, js, kotlin, metajs, php, swift, typescript` — a broad,
cross-family group. It is exactly the compiler twin of `interp-core.js`'s `excCatch`; the
interpreter side already lives in the core, but the compiler side had drifted into nine
copies. The other five compilers (`c, go, lua, python, ruby`) never defined it, so there
is no clash.

**Canonical form** added once to `compile-core.js` (with the same comment tying it to
`excCatch`); the nine copies removed. Each language's own `makeTry` — which *does* differ
per language (see below) — keeps consuming the shared `{catchbody, catchname}`. **9
definitions → 1.**

---

## Documented, not collapsed

### `makeThrow` — so close, but dart blocks it

Eight compilers share the same `makeThrow` *shape* (`evaluate e → js_throw → NewRet →
deadBlock`), differing only cosmetically (param named `e` vs `t`, comment "unwinds" vs
"aborts"). But **dart's is a different function**: Dart's `throw` is an *expression*, so
`makeThrow(t, pos)` returns `{b, v: hUndef}` instead of terminating the block with
`deadBlock()`. A single core `makeThrow` (statement form) would clash with dart's
expression form (and the goja-vs-frozen winner rule would make them diverge). Renaming to
dodge the clash would mean editing the `Throw` production tags in eight grammars — more
surface than the duplication is worth. Left per-language.

### `makeTry` — genuinely language-variant

`makeTry` looks shared but isn't: the languages differ in how many catch clauses they
keep, whether the try body is a `ctl` closure, how `excDispatch` is wired, and whether
try is even lowered vs `notImpl`. No single canonical body. Correctly per-language.

### The js / typescript / metajs cluster

As on the interpreter side, these three near-identical languages share a large cluster of
byte-identical builders (`makeUnary`, `makeTry`, `makeProp`, `makeObject`,
`makeHandleConst`, `makeFunctionExpr`, `makeFor`, `exprToClause`, `emitRefPath`, …). Not
hoistable: several names (`makeFor`, `makeDo`, …) collide with *different* bodies in other
languages, and a family-only `lib/js-core.js` is declined for the same
convention reason as `lib/kotlin-common.js` (see `kotlin-concept-consolidation.md`, F11).

### Language-variant builders

`makeReturn`, `makeArray`, `makeBitNot`, `makeDo`, `makeCond`, … appear in many compilers
with materially different bodies (control-signal protocol, handle constants, extern
names). No cross-family majority to canonicalize. Left per-language.

---

## Incidental fix surfaced by the comparison

`js-to-llvm-ir.abnf` carried a stale comment — *"try/catch/finally is not implemented;
under -warn the try block runs"* — directly above a `makeTry` that **does** fully lower
try/catch/finally (via `js_try` + closures + `excDispatch`), and directly above an
accurate description of that lowering. The false line was removed. (Not a missing
feature, so no task chip — the feature is present; only the comment was wrong.)

---

## Summary

| concept | languages | verdict |
|---|---|---|
| `makeCatch` | csharp, dart, java, js, kotlin, metajs, php, swift, typescript | **collapsed → compile-core.js** (refreeze) |
| `makeThrow` | 8 + dart | documented — dart's expression form would clash |
| js-family builders | js, typescript, metajs | documented — can't hoist / no family file |
| `makeTry`, `makeReturn`, `makeArray`, … | most | documented — language-variant |

`makeCatch` is the single clean *universal* cross-compiler collapse, mirroring the
interpreter side's `excCatch`; the rest is language-variant or family-local by the
codebase's deliberate per-language convention.
