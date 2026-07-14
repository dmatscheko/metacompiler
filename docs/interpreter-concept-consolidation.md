# Interpreter grammars: equivalent concepts across languages

Cross-language pass over the 14 big-language interpreters that share
`lib/interp-core.js` (`c, csharp, dart, go, java, js, kotlin, lua, metajs, php, python,
ruby, swift, typescript`). Goal: find one concept implemented more than once, across
different languages, in inconsistent ways, and collapse it into a single canonical
definition — in the **existing** shared home (`interp-core.js`), the way every other
cross-language helper already lives there.

Method: extract every top-level `function name(...)` from each interpreter's start
script, normalize whitespace/comments, and group by identical body. A helper whose body
is byte-identical across several *different-family* languages, and which is **not**
already in `interp-core.js`, is a hoist candidate.

Same ground rules as before: byte-identical `-q` output under goja and `-frozen` after
every change (`interp-core.js` is **not** in the frozen bootstrap — the bootstrap freezes
only `metajs-to-llvm-ir.abnf` — so each interpreter recompiles the edited core on the
fly; the edit is refreeze-free and validated on both engines by the full 289-entry
matrix).

---

## Collapsed — the dictionary helper trio → `interp-core.js`

Six languages carry a distinct dict/map value type modelled as an insertion-ordered
`{__dict, keys, vals}` box, and each re-implements the same three primitives:

| helper | csharp | dart | php | python | ruby | swift |
|---|---|---|---|---|---|---|
| `isDict` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `dictFind` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `dictSet` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

They are logically identical everywhere, with **cosmetic drift** that is exactly the
"expressed inconsistently" smell:

- `isDict`: five write `v !== null && v !== undefined && …`; **python swaps the operands**
  (`v !== undefined && v !== null`) and spreads it over three lines.
- `dictFind`: four write the loop body as `{ return i }`; **csharp drops the braces**
  (`return i`); **python** multi-lines the loop.
- `dictSet`: five put the `if/else` on one line; **python** breaks the `else` onto its
  own line.

**Canonical form** (adopting the majority spelling), added once to `interp-core.js`:

```js
function isDict(v) { return v !== null && v !== undefined && typeof v == "object" && v.__dict === true }
function dictFind(d, k) {
    for (var i = 0; i < d.keys.length; i++) { if (d.keys[i] === k) { return i } }
    return -1
}
function dictSet(d, k, v) {
    var i = dictFind(d, k)
    if (i >= 0) { d.vals[i] = v } else { d.keys.push(k); d.vals.push(v) }
}
```

The six copies are removed (each left with a one-line pointer comment). The other eight
interpreters never defined these names, so there is no clash; they simply gain three
unused helpers. Each language keeps its own *literal/constructor* builders on top
(`newDict` in python, `makeMap` in dart, `makeDict` in swift, …) — those are genuinely
language-shaped and stay put. **18 function definitions → 3.**

---

## Documented, not collapsed — and why

### The js / typescript / metajs cluster (big, but can't hoist)

`js`, `typescript` and `metajs` are near-identical languages (TypeScript = JS with types
parsed-and-ignored; MetaJS = a JS dialect), so their interpreters share **byte-identical**
bodies for a large cluster: `makeAnd`, `makeOr`, `makeCond`, `makeSwitch`, `makeDo`,
`makeFor`, `makeDecl`, `makeFuncDecl`, `makeFunctionExpr`, and more.

This is the largest duplication in the set, but it **cannot** be hoisted into
`interp-core.js`:

- Several of these names (`makeSwitch`, `makeDo`, `makeFor`, `makeDecl`, `makeCond`) are
  **also** defined — with *different* bodies — by `c`, `csharp`, `java`, `go`, `swift`,
  `tinyc`, etc. A single core definition would clash with those per-language variants
  (and, via the goja-hoist-vs-frozen-textual-position rule, the two engines would pick
  opposite winners). So core is not a legal home for them.
- The only alternative — a shared `lib/js-core.js` for the three siblings — is declined
  for the **same reason a `lib/kotlin-common.js` was declined** (see
  `kotlin-concept-consolidation.md`, F11): no language pair/family here has a `*-common.js`;
  each language stays self-contained beyond the generic core, and introducing a
  family-only shared file for one family would itself be an architectural inconsistency.

`makeOr`/`makeAnd` are the one sub-case defined *only* by the three js-family languages
(everyone else uses core's `makeOrAnd`), so they *could* sit in core without a clash —
but hoisting two helpers used by exactly one family into the universal core just to trim
three copies pollutes core with js-isms for no real gain. Left as-is.

### Language-variant builders (look shared, aren't)

`makeReturn` (14 langs), `makeAssign` (14), `makeIncDec`, `makeCall`, `makeTemplate`,
`mcall`, `makeFor`, `makeSwitch`, … appear in many interpreters but with **materially
different bodies** — they encode each language's own return-signal protocol, assignment
targets, string coercion and method dispatch (e.g. kotlin's `makeReturn` threads a
`blockSignalOf`; `mcall` is explicitly the one name `interp-core.js` asks each language
to provide). A normalized-body grouping puts them in singletons or small same-family
clusters, never a clean cross-family majority, so there is no single "best variant" to
adopt. Correctly left per-language.

---

## Summary

| concept | languages | verdict |
|---|---|---|
| `isDict` / `dictFind` / `dictSet` | csharp, dart, php, python, ruby, swift | **collapsed → interp-core.js** |
| js-family syntax builders (`makeSwitch`, `makeOr`, …) | js, typescript, metajs | documented — can't hoist (name clashes) / no family file |
| `makeReturn`, `makeAssign`, `mcall`, `makeIncDec`, … | most/all | documented — genuinely language-variant |

The dict trio is the single clean *universal* cross-interpreter collapse available; the
remaining duplication is either language-variant (not one concept) or family-local in a
way the codebase's per-language convention deliberately keeps separate.
