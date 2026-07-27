# ABNF dialect gotchas

Traps in this project's annotated-ABNF dialect and in the two engines that run
it (goja, and the frozen MetaJS bootstrap). Every entry below cost somebody
real debugging time, and most of them share one shape: **the run stays green,
or the error points somewhere else entirely.** That is what makes them worth a
document rather than a comment at the site.

Written while widening the language grammars toward full recognition; see
`README.md` for the dialect itself and `docs/interpreter-concept-consolidation.md`
/ `docs/compiler-concept-consolidation.md` for how the tags compose.


## Parsing

**Ordered choice never retries a committed alternative.** This is a PEG: put the
longest / most specific alternative FIRST. A shorter one that matches a prefix
wins, and then the rest of the sequence fails without the parser ever going back
to try the longer one. Real bugs of this shape found here: `AnnTarget` listing
`set` before `setparam`, so `@setparam:x` matched `set` and then died on the `p`;
`ObjectExotic` after `ObjectReal`; and `Primary` listing `FString | StringLit`,
so an f-string matched only the FIRST piece of `f"a{1}" "b"` and the adjacent
plain piece was never reached.

**A literal suffix can swallow the next identifier.** Adding an unsigned integer
suffix made `0 until n` parse as `0u` followed by a method call `ntil`. Where a
suffix could run into a following name, guard it with a `KwEnd`.

**A Command immediately followed by a group is read as a repetition.** Writing
`:whitespace() ( "?" | "!" )` fails the grammar with `Only Command :number() can
be used for Times`. Worse, it fails only when that rule is first REACHED, so it
hides behind an otherwise green run. Hoist the group into a named production.

**`!Rule` is not supported — only `!"literal"`.** There is no negative lookahead
on a rule reference anywhere in this tree. When you need one, write a `:script`
guard instead. The failure mode is unusually confusing: writing
`Annotation = "@" !KwInterface Id …` does not report a grammar error, it makes
the whole grammar unparseable, and the loader then dumps a parse tree of the
GRAMMAR FILE itself — which looks nothing like the mistake. The working pattern
is a small `:script` that checks the identifier and its boundary explicitly.

**Whitespace is NOT skipped before a `:script` guard.** The dialect skips spaces
in front of each *token*, so at the moment a guard runs, any newline or run of
spaces is still AHEAD of the position. A guard that wants to know "did a line
break just happen" must scan FORWARD, not backward. (A first attempt that
scanned backward silently did nothing.)

**`c.peek` answers `-1` out of range, not `0` or `undefined`** — identically in
`abnf/parserscript.go` and `abnf/frozen.go`. Guards that tested only for
`=== 0 || === undefined` never detected the file boundary and spun to their step
cap on every call, which is quadratic over a file. Always test `< 0`. Negative
offsets are legal and useful: `c.peek(-n)` reads backwards.

**`c.peek` returns BYTES, not runes.** A `KwEnd` that compared against code
points made `1<U+2028>` unparseable. Decode UTF-8 inline if you need a rune.

**Guards must be PURE functions of (source, position).** Mutable parser state is
useless here: a rolled-back alternative never undoes the mutation, and nothing
else will either, so the state is corrupt for everything that follows. The
working pattern is to re-derive what you need on every call by walking with
`c.peek`. Strict-mode tracking in `js-interpreter.abnf` does exactly this — it
walks backwards to the start of file and recomputes strictness from scratch,
which is O(prefix) per call but correct under any amount of backtracking.

**`:script` guards run in the parser VM** and therefore cannot call helpers
defined in the grammar's start script.

**Catastrophic backtracking is real.** Nested `((((…))))` or `[[[[…]]]]` can hit
2^depth when two independent descents parse the same left side (an assignment
form and a conditional form, or a function type and a parenthesized type). The
fix that works is a cheap fast-fail lookahead that scans ahead for the deciding
token — `HasTopLevelAssign`, `AssignAhead`, `ArrowAhead`, `ParenAhead`,
`FnTypeAhead` are all instances. Cap every scan loop; an uncapped one that runs
past the start or end of the file hangs.


## Scripts and tags

**Two top-level functions sharing a name silently rebind TAGS.** Declaring
`function makeCtorCall` twice in a `:startScript` raises no JS error and leaves
the parse tree correct — but a production bound to the first declaration keeps
calling it, so (real example) a `Method` node executed the `Ctor` tag and every
class reported "constructor name does not match class". Nothing points at the
duplicated name. `mec <grammar> -verify` now reports this as an error; run it
after editing a script.

**The `:description(...)` block is PARSED, not inert prose.** A literal `\u` or
`\x` in it is read as an escape and the whole grammar fails to load with
`newTokenEscaped: invalid syntax`. This has caught several people, always while
documenting a newly added escape feature. Describe escapes without literal
backslashes, and re-run the matrix after editing a description — not only after
editing rules.


## The frozen MetaJS engine

`./test.sh` runs everything twice, once under goja and once with `-frozen`, and
requires byte-identical stdout. The frozen engine is a smaller language, so
these fail ONLY under `-frozen` — behind an otherwise green goja run.

- **No `for…in`.** Anything enumerating object keys (a class body, an instance
  dictionary, pattern captures) needs an explicit key list carried alongside.
- **No `array.splice`, and `array.length` is not assignable.** Removal and
  truncation have to be written with index shifting plus `pop()`.
- **A reassigned function parameter keeps the type it was first called with.**
  `function f(v) { v = v + "" }` fails with "variable 'v' has type number and
  cannot hold a string". Copy the parameter into a fresh `anytype` local before
  mutating it.
- **No `string.split` / `array.join`, no `string.lastIndexOf`, and `indexOf`
  IGNORES its from-index argument.** The last one is the nasty one: it does not
  raise, it returns the same match every time, so a `while (i < s.length) { i =
  s.indexOf("\n", i) + 1 }` line walker spins until the IR step cap — under
  `-frozen` only. Write a small `for` scan instead. (All four found while adding
  the Ruby heredoc lexer.)
- **A `var` declared in a `for`-init is scoped to that loop.** A second
  `for (i = 0; …)` reusing the name dies with "assignment to undeclared
  variable". Declare it standalone first.
- **`arguments` is not resolved from inside a `try` block.** Capture it into a
  local before the `try`.
- **`{k: anytype}` inside an object literal** behaves differently from
  `var x = anytype`.
- **A record that has travelled through the tag stack returns a FRESH array on
  every property read**, so `rec.arr.push(x)` silently does nothing. Read into a
  local, mutate it, store it back.

**`./test.sh` alone does NOT run the full-syntax ratchet under `-frozen`; only
`./test.sh --full` does.** A frozen-only bug in shared thunk machinery will hide
behind a green matrix until someone runs the ratchet.


## Measuring

**A ratchet that exits 0 has not necessarily run.** `tests/python-test-full.py`
only DEFINED `main()`, so both Python grammars ran the module top to bottom and
exited 0 having executed nothing — reported as full support for weeks.
`test.sh --full` now demands the `full: N checks, M failures` summary line and
prints `VACUOUS` otherwise.

**The reference sweep RUNS each corpus file**, so a correct parse of a
non-terminating program (`for(;;);`, `a: while (true) { continue a }`) is
indistinguishable from a hang and lands in the TIMEOUT column. Improving
coverage can therefore INCREASE the timeout count. Check `parsed + timeout` for
conservation before concluding anything, and only treat a timeout as a
backtracking bug after reducing it to a minimal input.

**Check what a corpus actually claims before believing a gap.** Kotlin's
`testData/psi/` ships a parse TREE (a `PsiErrorElement` marks a syntax error)
while `testData/lexer/` ships a TOKEN DUMP (`BAD_CHARACTER`, `DANGLING_NEWLINE`)
where the words `PsiErrorElement` never appear — so a classifier keyed only on
the tree demanded that we parse every lexer file, including the ones kotlinc
refuses. Chasing that number made one grammar accept vertical tab, NEL, LS and
PS as whitespace, which Kotlin does not. Only the form feed is real Kotlin
whitespace.

**`// COMPILATION_ERRORS` is not a syntax-error marker.** Of the 450 Kotlin
files carrying it, 174 have a clean parse tree: their errors are SEMANTIC, and a
syntax-only front end is required to parse them.

## Tags that return runtime objects

**A tag's completion value crosses into Go, and the bridge does not handle
cycles.** `toGoNatural` (`abnf/jsrt.go`, reached from `exportCompletion` in
`abnf/frozen.go`) walks the value a tag returns. Returning a runtime object
graph with a back-reference — a class descriptor whose constants point at
`__class`, which points back at the class — overflows the stack, and **only
under `-frozen`**. Have the tag return nothing, or a plain summary, and keep the
graph on the interpreter side.

**Put the specific case-label forms before the general one.** `case S -> 1`
parses as the lambda `S -> 1` if a general expression alternative is tried
first — the same ordered-choice trap as everywhere else, but easy to miss
because both readings are syntactically valid.

## `:script` productions

**A guard that scans FORWARD for a token on "this line" must skip the leading
whitespace itself — including newlines.** `SameLine`/`LineCont` treat a newline
as significant because they ask "did a line break just happen"; a lookahead that
asks "is there a `=>` ahead on this statement" starts at a position where the
blank lines and comments separating it from the PREVIOUS statement are still
ahead, so bailing on the first newline makes it answer "no" everywhere. Skip
whitespace and `#` comments first, then start the real scan. (Ruby's
`FatArrowAhead`; the symptom was that the rightward-assignment statement never
matched, and the parse error pointed inside the expression instead.)

**A `:script` returns a RULE, and a returned `newToken(text, …)` must match the
input literally** — the text cannot be a rewritten or synthesized value. What a
scanner CAN do is return a token longer than one lexeme: Ruby's heredoc lexer
returns the marker, the body lines and the terminator line as one token and
splits them apart again in the tag. Building that token from `c.peek` means
decoding UTF-8 by hand, since a string of raw bytes is re-encoded on the way to
Go and then no longer matches.

**Returning `undefined` from a `:script` means "match empty and SUCCEED"**, not
"fail". To make a script FAIL, return an impossible token — the convention in
this tree is `abnf.newToken("\x01<reason>", 0)`. A long-string scanner that
returned `undefined` on no-match made every expression match empty, which looks
nothing like the actual mistake.

**A `:script` sees the RAW input** — no whitespace is skipped before it — but
the token it returns IS matched after whitespace skipping. So a scanner has to
step over the prefix itself to find its literal, and then return a token
covering only the literal, without that prefix.

**Override a shared helper by ASSIGNMENT, not by declaration.** `makeSeq` and
friends already exist in `languages/lib/interp-core.js`, so a grammar that wants
its own must write `makeSeq = function (…) {…}`. Writing `function makeSeq(…)`
instead is hoisted under goja but position-dependent in the frozen engine, so
the two engines disagree about which definition is live. (See also the duplicate
top-level function trap above, which `mec -verify` now catches.)

## A recurring shape: object creation must SEED the member chain

Three languages independently lost 5-35 points of corpus coverage to the same
mistake, so check for it before theorising about a big `.` cluster.

`new Foo().bar()`, `new C ().test ()`, `new Outer().new Inner()` — if the
object-creation production is written as an ALTERNATIVE of the postfix or unary
level, it takes no suffix chain, and every one of those parses fails. In Java's
compiler this was a 744-file cluster that looked like "qualified names" and was
nothing of the kind; moving `NewExpr` into the primary/seed position was worth
six points on its own, and the identical fix moved C# 48.0% -> 54.1%.

The working shape is: object creation is a SEED of the member chain (like an
identifier or a parenthesized expression), not a form that sits above it. A
qualified `new` (`outer.new Inner()`) then wants to be a SUFFIX of that chain
rather than a form of its own — writing it as its own postfix form makes the
primary parse twice per nesting level, which is a 2^depth blowup that OpenJDK
ships a regression test for.

Related ordering trap from the same work: keyword-shaped call forms —
`typeof(T)`, `nameof(x)`, `sizeof(T)`, `default(T)` — look exactly like a bare
call, so they must be tried BEFORE the general call seed. Otherwise
`typeof(IShape)` parses as a call of a variable named `typeof`.
