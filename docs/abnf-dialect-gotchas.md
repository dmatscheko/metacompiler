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

**A character-range endpoint below `\x7f` works; `\x80` does not.** Widening a Ruby
Symbol name to accept accented letters with `"\x80"..."\U0010ffff"` matched nothing at
all — `:il_était` still died on the `é` — while `"\x7f"..."\U0010ffff"` works. `\x7f`
is a whole rune on its own and `\x80` is a UTF-8 continuation byte, so only the first
survives the decode. Start such a range at `\x7f`; the existing `TplPlain` in every
string rule here already does.

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

## Emitting a runtime dispatch (compiler grammars)

**`js_has` ABORTS on a number, string, boolean or null** — its switch only knows
objects, arrays and strings, and falls through to `'in' needs an object on the
right side`. So it cannot be used as an "is this a class instance" guard around
an operand of unknown type. `js_is_type(v, "Name")` is the safe probe: it answers
false for every value it does not recognize. The working shape for operator
overloading in `csharp-to-llvm-ir.abnf` is therefore to collect, at WALK time,
which classes declare which operator (`__op2_+ -> ["Vec"]`) and emit one
`js_is_type` probe per declaring class — which also means a program that declares
no operator emits exactly the IR it did before.

**`js_defprop` accessors never see `this` in a C-style compiler grammar.** The
runtime calls them as `rt.call(acc.get, o, nil)`, and `this` only reaches a
closure through `rt.thisStack`, which is filled only when `trackThis` is on (the
JS/TS grammars). A grammar whose `emitFunc` reads `this` out of `args[0]` — every
C-family one here — must emit a property accessor as an ORDINARY method
(`__get_X` / `__set_X`) and call it with `js_mcall`, which does pass the receiver
as the first argument. Kotlin gets away with `js_defprop` because it resolves
`this` through `js_kget` instead.

## An LLVM function's PARAMETER NAMES and its BLOCK LABELS share one namespace

In LLVM's textual IR, `%name` is one flat per-function namespace: a parameter, an
instruction result and a basic-block label all live in it. So

    m.NewFunc("rt_subst", …, llvm.ir.NewParam("star", i32))
    rtSubst.NewBlock("star")

emits a module that our built-in IR interpreter **runs perfectly**, and that
clang refuses:

    error: '%star' is not a basic block
      br i1 %3, label %star, label %plain

Read that failure mode carefully, because it is the reason this entry exists.
`./test.sh` runs every grammar twice and demands byte-identical stdout, and a
name collision is byte-identical under both engines — the module text is
deterministic, `llvm.Run` resolves the branch by handle rather than by name, and
the program prints the right answer. **The project's central invariant cannot see
this class of defect at all.** Only `-exe PATH`, which hands the printed module
to clang, catches it.

Every `-to-llvm-ir` grammar is exposed, and the exposure grows with the hand-written
runtime: the more `m.NewFunc(…, NewParam("s"), NewParam("to"), NewParam("star"))`
helpers a grammar emits, the likelier some block wants one of those words too.
The collision is easy to make because the two names are written far apart — the
parameter in the `NewFunc` line, the label thirty lines down — and because the
obvious block names (`star`, `plain`, `name`, `rest`, `from`, `to`, `end`) are
exactly the obvious parameter names.

Two habits avoid it. Give blocks a prefix no parameter would use (`bstar`,
`bplain`, or the `nb()` counter style), and **run `-exe` on the language's own
test file after touching an emitter**, not only `./test.sh`:

    mec languages/<lang>-to-llvm-ir.abnf tests/<lang>-test-full.<ext> -q -exe /tmp/x && /tmp/x

(Found in `batch-to-llvm-ir.abnf`, where `rt_subst`'s `star` parameter collided
with its `star` block. Both grammars were green, both engines agreed, and the
ratchet reported FULL.)

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

**A scanner whose text BEGINS with whitespace needs `:whitespace()` around it.**
The corollary of the rule above: PHP's inline-HTML run after `?>` normally
starts with the newline that ends the tag line, and that newline is *output*.
Returning it inside the token is correct, but the engine skips whitespace before
matching the token, so the match starts one character too late and fails. Wrap
the scan — `HtmlText = :whitespace() HtmlScan :whitespace(Whitespace)` — the way
`SkipDq` does, and restore the ambient rule at the end of the production.

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

## A one-character operator that is the prefix of a two-character one

Adding a unary `&` (address-of) to a grammar that already has `&&` breaks `a && b`,
and the failure is not the obvious one. `BitAnd`'s `"&"` matches the FIRST `&` of
`&&`, and the right-hand side then *succeeds* as the unary `&b` — so the sequence
never fails, never rolls back, and the expression quietly parses as
`a & (&b)`. Spell the binary form as `( "&" !"&" )` so it declines the
double form outright. The same shape applies to `|` vs `||`, `<` vs `<<`, and
`?` vs `??`/`?.`.

This is a specific case of the ordered-choice rule at the top of this file, but
it is worth its own entry because the usual symptom of a bad alternative — a
parse failure — does not appear. You get a wrong parse tree instead.

## A unary operator that is also an expansion's OPENING DELIMITER

Batch has a logical-not `!` in `set /a`, and cmd.exe's delayed expansion writes a
variable as `!VAR!`. The natural unary level

    ArithUnary = ArithNot | ArithBNot | ArithNeg | ArithAtom ;
    ArithNot   = "!" ArithUnary ;

reads `set /a counter=!counter! + 1` as `!(counter)` — which *succeeds* — and
then dies on the closing `!`. The reported position is that closing bang, so the
error points at the END of a construct the parser never considered, and the
obvious suspects (the `!VAR!` rule, the `+`, the assignment) are all innocent.

The fix is the ordered-choice one, and it needs no lookahead: put the expansion
alternative FIRST.

    ArithUnary = ArithExp | ArithNot | ArithBNot | ArithNeg | ArithAtom ;

That is safe precisely *because* the expansion form requires its closing
delimiter. `!counter!` matches it; a genuine unary `!x` does not, so it falls
through to `ArithNot` on its own. No `!"literal"` guard and no `:script` is
involved — the delimiter does the disambiguating.

The same shape appears wherever a language wraps names in a character that is
also an operator: `!x!` vs `!x`, `%x%` vs `x % y` (batch again — the modulo
operator and the expansion delimiter are the same byte), and `#{…}` vs `#` in a
language that has both interpolation and a comment or length sigil. The rule of
thumb: the DELIMITED form is the more specific one, so it goes first.

## `!"literal"` skips leading whitespace

A negative lookahead is matched the same way a token is, which means the
whitespace in front of it is skipped first. So `ForceSfx = NoSpace "!" !"="`
silently rejected `maybe! == 7`: the lookahead stepped over the space and found
the `=` of `==`. The neighbouring `maybe! != 7` and `maybe! + 1` both worked,
which makes it present as an operator-precedence bug somewhere else entirely.

When the lookahead has to be anchored to the very next byte, test it with a
`:script` on `c.peek(0)` rather than with `!"…"`.

The same trap sat on six `":" !":"` sites in the Ruby grammars, where the lookahead
is there to keep a label's `:` from being the first half of a `::`. It also rejected
the perfectly good

    { |a, b: :b, c: :c| ... }

because the lookahead stepped over the space and found the `:` of the symbol VALUE —
so a block could not declare a keyword parameter whose default is a symbol. One
`NotColonGlued` script on `c.peek(0)` replaced all six.

## An operator that is a longer spelling of an ASSIGNMENT operator

Ruby's `=~` broke `AssignOp = ... | ( "=" !"=" )` in a way that never failed: the
`=` matched, the `!"="` lookahead was happy (the next character is `~`), and the
right-hand side then parsed **successfully** as the unary `~/re/`. So

    s =~ /b+/

quietly became `s = ~(/b+/)` — an assignment of a bitwise-not, no parse error
anywhere. This is the `&` vs `&&` trap above, but with the extra twist that the
obvious fix does not work: `( "=" !"=" !"~" )` **also rejects the perfectly good
`x = ~5`**, because a negative lookahead is matched like a token and therefore
steps over the space in front of the `~` first (see the `!"literal"` entry
above).

When the lookahead has to be anchored to the very next BYTE, it has to be a
`:script` on `c.peek(0)`. Ruby's is:

    AssignOp = ... | ( "=" !"=" NotTilde ) ;
    NotTilde = :script(~~
        (function() {
            if (c.peek(0) === 126) { return abnf.newToken("\x01tilde", 0) }
        })()
    ~~) ;

(Returning `undefined` means "match empty and SUCCEED"; returning an impossible
token is how a `:script` fails.) The same shape is needed for any operator whose
first character is an assignment operator: `=~`, `!~`, and `=>` next to `=`.

## An operator whose operand is really the START of a different lexical form

The same shape once more, and the nastiest instance of it found so far. Ruby's

    evaluate <<-ruby do
      @a = -> ((a)) { a }
    ruby

is a command call whose argument is a HEREDOC. But `AppendStmt` is spelled

    AppendStmt = Target "<<" !"<" AddExpr { "<<" AddExpr } ;

and `-ruby` is a perfectly good `AddExpr` — a unary minus applied to the variable
`ruby`. So the statement parses as `evaluate << (-ruby)`, **succeeds**, and there is
no parse error anywhere near the mistake. The damage surfaces two lines later, as
the heredoc BODY being offered to the parser as ordinary code:

    ruby interpreter error: unknown name: @a

which sends you looking at lambdas and destructuring parameters. The same reading
hides `<<~A`, `<<'A'` and `<<"A"`, and `ShiftExpr` has it too (`x = a <<-B`).

The fix is a zero-width guard directly after the `<<`, refusing the operator when a
heredoc marker is GLUED to it:

    AppendStmt = Target "<<" !"<" NotHeredoc AddExpr { "<<" NotHeredoc AddExpr } ;
    ShiftExpr  = AddExpr { ShiftOp <~~ push(up.in) ~~> NotHeredoc AddExpr } ;

`NotHeredoc` answers "this is a heredoc" for `~`, `-`, `'`, `"` or an upper-case /
underscore initial glued to the `<<`, and lets everything else through — which is
exactly Ruby's own rule, and why `a << b`, `a << -1` and `arr << Foo` keep working
(they all have a space, so nothing is glued).

It has to be a `:script` on `c.peek(0)`, not `!"~"` / `!"-"`: a negative lookahead
is matched like a token and therefore SKIPS the whitespace in front of it (see the
`!"literal"` entry above), so `!"-"` would also reject the perfectly good
`a << -1`. Two entries in this file now share that conclusion; treat "the fix is a
`!"…"` lookahead" as wrong by default whenever the distinction is *gluing*.

## The Times trap also fires on a group of single-character alternatives

The "a Command immediately followed by a group is read as a repetition" entry
above is usually met while writing a lookahead. It fires just as readily on an
ordinary character choice. Adding Ruby's `$1..$9` / `$~` / `$&` globals as

    GvarNum = "$" :whitespace() ( "1"..."9" | "~" | "&" ) <~~ ... ~~> :whitespace(Whitespace) ;

failed the whole grammar with

    Only Command :number() can be used for Times. Command is: times{or{range "~&"}}

— and, as the original entry warns, only when that rule is first REACHED, so it
hid behind an otherwise green run and surfaced as a mid-file parse failure. The
fix is the same one: hoist the group into a named production.

    GvarNum   = "$" :whitespace() GvarNumCh <~~ ... ~~> :whitespace(Whitespace) ;
    GvarNumCh = "1"..."9" | "~" | "&" ;

Worth its own note because the group here is not a lookahead and looks nothing
like a repetition — it is the ordinary way every other alternative in the file
is spelled, just in the one position where it cannot be.

## A greedy modifier list eats the declaration keyword

Adding `class` to a `Modifier` production (to allow Swift's `class var`) broke
`@objc class Cache {…}` at top level: `Modifiers` consumed the `class`, and
`ClassDecl` then had nothing left to match. Keep a keyword that also *starts* a
declaration out of the shared modifier list, and give members their own
modifier production instead.

## A top level that EXECUTES as it walks cannot host a conditional branch

Conditional compilation (`#if` / `#else` / `#endif`, and any construct that parses
several alternative bodies and keeps one) needs every branch to be *collected*,
not run. In an interpreter grammar whose top level is written as

    TopItem <~~ var t = pop(); t() ~~> = Statement ;

the thunk fires from the TAG, during the walk — so a `#if` whose branches are
`{ TopItem }` executes EVERY branch, and the symptom is not a parse error but a
program that runs its `#else` body as well as its `#if` body. The fix is one
level of indirection: make each item PUSH its thunk (a top-level function
declaration pushes `function () { registerFun(f) }` rather than registering on
the spot), and give the item list one wrapper production that runs what it
finds. The branch rule can then collect `takeAll()` and the selecting tag splices
only the live branch's items back with repeated `push()`.

Compiler grammars in this tree are already in that shape — `Program <~~
buildMain(takeAll()) ~~>` — so the same feature is a pure addition there. The
splice-by-repeated-push trick works wherever the enclosing rule reads its items
with `takeAll()` (block, class body, switch body, closure body); it does NOT work
under a rule that pops exactly one, such as the compiler's
`Statement <~~ push(stmtPos(pop())) ~~>`, which is why a statement-level `#if`
belongs in the ITEM LIST of `Block` and `Switch` rather than among the
alternatives of `Statement`.

Two smaller traps from the same work: `"#else"` matches the front of `"#elseif"`
(order `#elseif` first, or the leftover `if` is read as an if statement), and the
condition has to be consumed under `:whitespace()` — with the normal whitespace
rule in force, a `{ AnyCharButNewline }` run still skips the newline in front of
each character and swallows the rest of the file.

## Tagging a sub-rule changes how many items its PARENT pops

A rule that contains an untagged sub-rule is quietly relying on that sub-rule
pushing nothing. Give the sub-rule a tag later and the parent now finds an extra
item on the stack: `CatchPat = [ Pattern ] …` began pushing TWO items the moment
`Pattern` got a tag, so `makeCatch` took a string where it expected the catch
body. The failure surfaced a long way off, as `TypeError: Value is not an
object` inside `emitFunc`.

When you tag a rule that other rules already reference, check every reference.
The cheap fix at a site that must stay item-neutral is a dropping wrapper:

    DropPattern <~~ pop() ~~> = Pattern ;

Related, in the compiler grammars: `Statement` wraps every thunk in `stmtPos`
for `-trace`/`-cfgraph`. A statement production that pushes a RECORD rather than
a function must override `stmtPos` to pass non-functions through, or the record
is wrapped into a function and the feature silently stops working.

## A bare `return` swallows the next line (the missing ASI restricted production)

`if (cond) return` followed by a statement on the next line does not do what it
says. In real JavaScript `return` is a RESTRICTED PRODUCTION: no line terminator
may sit between it and its expression, so a newline forces the semicolon. The
grammars here spell it `Return = KwReturn [ Expression ] [ ";" ]` with no such
restriction, so

    if (x) return
    log = "after"

parses as `return (log = "after")` — the assignment is performed AND returned.

Measured against node, which prints `undefined|after`:

    js-interpreter          after|
    typescript-interpreter  after|
    metajs-interpreter      after|

This is why it presents as a FROZEN-ONLY bug in tag scripts. Under goja a tag
script is parsed by real JavaScript and gets correct ASI; under `-frozen` it is
parsed by the a-grammar frozen from metajs, which has this rule — so the two
engines genuinely disagree. It surfaced in `c-to-llvm-ir.abnf` as a
`hasOwn(fields, name)` answering TRUE for a name never added, and a two-field
struct failing with "duplicate field x", behind a green goja run.

Until the rule carries a no-line-break guard, brace every value-less early
return: `if (cond) { return }`. `if (cond) return value` is unaffected, which is
why the valued form is used all over this tree without trouble.

## A thunk crossing the tag stack loses its identity — under goja only

A build-time side table keyed by a thunk's function identity works perfectly
under `-frozen` and silently does nothing under goja. `push(v interface{})` in
`abnf/compilerscript.go` exports the goja function value to a Go `interface{}`
and re-imports it on the next `pop()`, so the parent tag receives a FRESH
function object; the frozen engine passes a handle and preserves identity. A
`table.indexOf(items[0])` therefore answers `-1` under goja and the right index
under `-frozen`.

The failure mode is the nasty one: no error, and the two engines print DIFFERENT
answers. It was found in `go-interpreter.abnf` while making `float64(i)/2`
divide as a float — goja kept printing `2` and `-241` while `-frozen` already
printed `2.5` and `15`, which reads like a frozen bug and is the opposite.

Key such a table by the END SOURCE POSITION of the node instead. `up.pos` is a
node's end offset (not its start), so a rule that wants to recognize "this
operand IS a conversion" tags every operand with its own `up.pos` and matches
the conversion's recorded position exactly:

    MulExpr <~~ push(goFold(takeAll())) ~~>
            = UnaryExpr <~~ push({at: up.pos}) ~~>
              { SameLine MulOp <~~ push(up.in) ~~> UnaryExpr <~~ push({at: up.pos}) ~~> } ;

Positions are plain numbers, so they survive the round trip unchanged. Do not
reach for a property on the thunk either: the frozen runtime rejects a non-index
property on an array (`invalid array index goarr`), and a function property does
not survive the goja round trip at all.

## Go-style semicolon insertion applies at the BINARY OPERATOR level

Go inserts a semicolon after a line whose last token is an identifier, a
literal, `++`, `--`, `)`, `]` or `}`. A grammar that spells its arithmetic as
`MulExpr = UnaryExpr { MulOp UnaryExpr }` has no such rule, so

    p := 1
    *p = 2

parses as `p := 1 * p` — the short declaration swallows the `*p` on the NEXT
line and the parse then dies on the `=`. The reported position is the `=`, which
points at the assignment rather than at the expression that ate its left side,
and `Target = { "*" } Id …` looks like the culprit even though it is correct.
The same shape hides `&x`, `-x`, `+x`, `^x` and `<-ch` at the start of a line.

The fix is one guard: the operator must be on the same line as the operand
BEFORE it, which is exactly what `SameLine` already tests.

    AddExpr = MulExpr { SameLine AddOp MulExpr } ;
    MulExpr = UnaryExpr { SameLine MulOp UnaryExpr } ;

This is not a restriction on multi-line expressions: Go puts the operator at the
END of the continued line (`x := a +` / `b`), and there the `+` still sits on
`a`'s line. `a +`, `b*`, `len(arr)-` and a parenthesized continuation all keep
working; only an operator that OPENS a line is refused, which is what Go does.
Worth two points of Go corpus coverage on its own.

## Give the RARER kind the special shape, not the common one

Go arrays and Go slices were both a plain host array, so nothing at runtime told
them apart — and the two things that needed telling apart pull in opposite
directions: an array must be COPIED on assignment, a slice must SHARE its
backing storage. A marker property is not available (the frozen runtime answers
`invalid array index goarr` for any non-index key on an array), and a side table
keyed by array identity grows without bound.

What works is to move only ONE of the two off the plain representation. Making
the SLICE a header `{a, o, n, c}` leaves `Array.isArray(v)` meaning exactly "v is
a Go array" — a free discriminator, no marker, no table — and gives `cap`,
`s[1:3]` sharing, `s[:2:3]` and an append that writes into shared storage all at
once. Making the array the special one would have cost the same work and bought
none of the slice semantics.

The cost lands on whoever holds the primitives. In the interpreter it is a
dozen call sites. In a `-to-llvm-ir` grammar the handle runtime's externs
(`js_golen`, `js_goslice`, `js_goappend`, `js_gorange`, `js_pyset`, …) only know
plain arrays, so each operation becomes a GENERATED function that reaches
through the header and falls back to the extern — `declFn` in `go-to-llvm-ir.abnf`
builds them. Two shapes make that affordable: `emitSelect` (a value-level
if/else joined with a phi) and `emitCountLoop`, which keeps its counter in a
scope slot rather than a phi so the loop needs no phi placement at all.

One trap when the header is a class instance: `js_gotypeis` then reports the
class name, so a type switch stops seeing `slice` (and `any`). Answer the probe
from the BACKING array instead of the header and both come back for free.

## `up.in` hands over the matched TOKENS, with the separators removed

A production-level tag's `up.in` is not the source SLICE the production covered:
it is the concatenation of the tokens it matched. So a declaration specifier run
spelled `TypeInt = TypeElem { TypeElem }` gives

    long double   ->   "longdouble"
    unsigned int  ->   "unsignedint"

A reader that recovers the base type by splitting that text on whitespace sees
one unknown word, matches no keyword, and falls back to plain `int`. Nothing
fails: the declaration parses, the variable is declared, the program runs.

The failure mode is what makes this worth a section. It presented as **"`double
x` works but `long double x` does not"** — in BOTH C grammars at once, which
rules out the usual suspicion of a one-sided emitter bug — and the compiler only
showed it because an `alloca i32` turned up where an `alloca i64` belonged. The
interpreter, whose memory holds JS numbers, gave the RIGHT ANSWER for the wrong
reason: its full-syntax section passed by luck, because the only `long double`
value the test stores is `1.0L`, and 1.0 truncated to the integer 1 still
compares equal to 1.0 and still divides to 0.25. A fractional value would have
been silently wrong.

Two ways out. Scan the run for the keywords themselves rather than for words:

    var baseKws = ["unsigned", "signed", "double", "float", "short", "_Bool",
                   "char", "void", "long", "int"]     // longest-first where they overlap

or, if the tokens must stay separate, give the sub-rule its own tag and collect
them: `BaseElem = TypeTok <~~ push(up.in) ~~> | Attr ;` — a single token's
`up.in` IS its own text. The same trap applies to any `raw`-text test over a
multi-token run; `isUnsignedTy(raw)` survives only because `indexOf("unsigned")`
does not care about the missing space.

## Testing a PRODUCT for zero overflows; test the factors

In emitted integer IR (and anywhere else 64-bit arithmetic wraps), the guard

    isZero = (ma * mb) == 0

is not "either factor is zero". `2^52 * 2^52` is `2^104`, which is 0 modulo
2^64 — so a soft-float multiply that used it returned 0.0 for every product of
two powers of two, which is exactly the set of values a floating-point test
suite is built from. `3.0 * 2.0` was 0.0 while `3.0 - 1.0` was fine, which sends
you looking at the mantissa arithmetic instead of at the zero guard next to it.

Test the factors, and remember that a `NewOr` of two `i1` comparisons is not
always available — a select chain is:

    az     = ICmp(EQ, ma, 0)
    isZero = ICmp(EQ, Select(az, 0, mb), 0)      // ma == 0 || mb == 0


## A trailing closure and a statement block look identical

`if case .p = E.c(3) { body }` did not parse in either Swift grammar, and the
error pointed inside the condition. The reason is that `{ body }` is a perfectly
good TRAILING CLOSURE of `E.c(3)`: `MArgs` ends with `[ TrailArg ]`, the closure
matches, the `if` then has no block left and the whole statement fails. The same
shape had been quietly costing the grammar elsewhere — `switch words.count { ... }`
only worked because `case 1, 2:` is not a valid closure body, so `MethodSfx` failed
on the closure and fell through to `FieldSfx`. Change one case label and it breaks.

Real parsers resolve this with a FLAG: while reading the condition of an
`if`/`while`/`switch`/`for`, trailing closures are switched off. A `:script` guard
cannot carry a flag — guards must be pure functions of (source, position), because a
rolled-back alternative never undoes a mutation (see "Guards must be PURE" above).

The fix is a zero-width guard that re-derives the flag from the source every time.
`NoTrailCtx` in `languages/swift-*.abnf`:

1. If the next non-space character is not `{`, answer "allowed" immediately — the
   question does not arise and the common path stays cheap.
2. Walk BACKWARDS, tracking bracket depth. On an unmatched `(` or `[` answer
   "allowed" at once: we are inside a call or a subscript, so
   `if xs.contains(where: { $0 > 1 }) { ... }` keeps working, and so does
   `check(runBoth { 30 } cleanup: { 12 })`. Stop at an unmatched `{`, at a `}`,
   at a `;`, at a newline at depth 0, or at the start of the file — that is the
   start of the enclosing statement.
3. Read the first word there, skipping a leading `else` (`} else if x.f() { ... }`).
   `if`, `while`, `switch`, `for` and `guard` mean the brace ahead is a BODY, so the
   guard fails and the trailing-closure alternative is declined.

Two details matter. The backward walk must be capped, like every scan in this tree.
And stopping at a newline is what keeps it cheap — a statement is usually one line —
while being conservative in the safe direction: a condition split across lines finds
a non-keyword first word and allows the trailing closure, which is the behaviour
there was before the guard.

This generalises to any language with trailing closures, or with a brace that can
open either a block or a literal: Kotlin's `if (c) f { }`, Ruby's `do`/`{` blocks,
and any grammar where a statement keyword is followed by an expression and then a
brace.

## A keyword-rejecting guard must skip COMMENTS, not just whitespace

The forward-scan entry above is about a lookahead that answers "is there a `=>` on
this statement". The same omission has a second, quieter shape: a guard whose job
is to REFUSE something. `NotCKw` rejects an identifier that is one of C's keywords,
so that a typedef use recognized by shape (`Name name;`) cannot swallow
`return x;`. It skipped spaces, tabs and newlines — but not comments. After

    int putchar(int c);             /* Prototypes are parsed and ignored. */

    int nfail = 0;

the guard's own position is the byte after the `;`, the scan stops at the `/` of
the comment, finds no keyword there, and SUCCEEDS. The `Id` that follows is matched
after real whitespace skipping, so it is `int` — and the declaration `int nfail = 0;`
was read as a typedef use of a type named `int`.

A refusing guard that skips too little says yes, which is the unsafe direction. It
presented as "the whole file fails, but the same two lines in a probe file work",
because the probe had no comment. Give any guard that scans forward the full skip:
spaces, tabs, CR, LF, form feed, `//` to end of line, and `/* */`.

## The matrix compares each engine against ITSELF, never the two halves

`./test.sh` runs every entry twice — goja and `-frozen` — and demands byte-identical
stdout. That is a strong invariant and it is blind to one whole class of defect: the
**interpreter grammar and the compiler grammar of the same language giving different
answers.** Both halves are self-consistent across both script hosts, both report FULL,
and the matrix is green while `nil.to_s` is `""` in one and `"null"` in the other.

Nothing will ever flag that on its own. The only way to find it is to run the same
probe through `languages/<lang>-interpreter.abnf` and `languages/<lang>-to-llvm-ir.abnf`
and diff, with the real toolchain as the tie-breaker. A harness that strips the emitted
IR is worth keeping around:

    strip() { awk 'BEGIN{n=0} {l[n++]=$0}
      END{last=-1; for(i=0;i<n;i++) if(l[i]=="}"||l[i]~/^declare /||l[i]~/^@/||l[i]~/^define /) last=i;
          for(i=last+1;i<n;i++) print l[i]}'; }
    mec languages/$L-interpreter.abnf p -q | strip
    mec languages/$L-to-llvm-ir.abnf  p -q | strip

Two notes on the flags: `-q` prints the module AND the program output for a
`-to-llvm-ir` grammar (the module is its default output, not noise), and `-qq` is not a
uniform substitute — for a grammar that runs through the JS runtime it still shows
program output, for one that runs through `llvm.Run` it does not.

### Two ways the differ itself lies

That strip is where the differ's own first two bugs lived, and both made a green half
look broken:

**The strip ate the compiler's warnings.** A compiler emits a `warning:` line while it
is BUILDING the module, so the line lands *before* the IR — and "drop everything up to
the last `}`/`declare`/`@`/`define`" drops it. Thirteen languages then read as
"interpreter warns, compiler silent" when in fact both warned identically. Lift the
`warning:` lines out first and re-emit them ahead of the stripped body. Relative order
between a warning and program output is not comparable across the halves anyway (one
warns at compile time, the other while running); PRESENCE is the thing to compare.

**The differ dropped only one of the two abort spellings.** `<lang> interpreter error:`
was filtered and `IR interpreter: call to undefined external function @f` was not, so
two halves aborting on the SAME cause read as a divergence. Filter both, or neither; an
abort is still visible as the output that stops.

And one on the input side: the `*-test-multifile.*` programs need `-i tests/imports`,
exactly as the matrix gives them. Without it the import does not resolve, `-warn-imports`
skips it, and each half then fails its own way on the missing symbol — so the comparison
is between two RECOVERY paths, not between the feature. Give the cross runner the same
include root the matrix uses.

### The recurring shapes

Every divergence found in the 2026-07-27 sweep was one of five shapes. Check these
first in a new grammar pair.

**1. The compiler hands a RAW value to the shared `println` host function.** The
interpreter half stringifies first (`rstr`, `jstr`, `swstr`, …), the compiler does not,
so Go's `%v` reaches stdout: `<nil>` for null, `[1 2]` for a list, and a full
`map[__class:map[__isclass:true …]]` dump for an object. Each language wants a different
answer here (`null` in Java/Kotlin/Dart, `nil` in Swift, `""` in C# and Ruby, `<nil>` in
Go), so the fix is a per-language print extern, the way `js_rputs` / `js_rprint` and
`js_luaprint` are — not a change to `printArgs`.

**2. A per-language operator falls through to the JavaScript one.** Python's `*` did
numeric multiplication, so `[0] * 3` was `NaN` while the interpreter repeated the list.
Lua's `<<` used `js_shl`, which is 32-bit with a 5-bit count mask, so `1 << 32` was `1`
while the interpreter (and Lua) said `4294967296`. PHP's `strlen` measured the
JavaScript string form, so `strlen(true)` was `4`.

**3. A field-access FAST PATH in the compiler skips the dispatcher.** `ruby-to-llvm-ir`
short-circuited `.to_s` to `js_jadd("", v)` (JavaScript's rule: `nil` became `"null"`)
and `.to_a` to the identity (so `MatchData#to_a` aborted downstream). The fast path
exists for speed and quietly implements different semantics from the `js_*mcall` it
bypasses; whenever you add one, check it against the general path.

**4. A variadic builtin implemented as unary.** `Array#push(a, b, c)` kept only `a` in
the shared runtime, Swift's `print(1, 2, 3)` printed `1` in the interpreter. Neither
raises. Grep for `argAt(args, 0)` / `function (x)` in anything whose language spelling
is variadic.

**5. The two halves carry DIFFERENT builtin tables.** `python-to-llvm-ir` had `max`/`min`
and lacked `bool`/`float`/`repr`/`set`/`sum`/`tuple`; `python-interpreter` had exactly the
complement. One half raises "variable not defined" for a program the other runs.

### Counting characters is three different questions

`${#s}` in bash disagreed three ways on `a<emoji>b`: real bash 5.3 counts CHARACTERS (3),
`bash-interpreter.abnf` counted UTF-16 code units (4, because the script string type is
UTF-16) and `bash-to-llvm-ir.abnf` counted bytes (6, because its runtime string is a
NUL-terminated `i8*`). All three are defensible readings of "length" and only one is the
language's.

On the interpreter side the fix has to go through `chunkAt` — a char-by-char rebuild
drops surrogate pairs, because the script string type re-encodes on every concatenation
and a lone surrogate becomes U+FFFD (this is the same trap the existing `chunkAt` was
written for). On the IR side, counting characters in UTF-8 is one byte test —
`(b & 0xC0) != 0x80` starts a character — but a SLICE needs the byte offset of a
character index, so it wants its own `rt_charoff` / `rt_csubstr` beside the byte-based
`rt_substr` that every pattern helper drives. Do not convert `rt_substr` itself: the glob
and `${v#pat}` machinery walks it in byte positions.

Note also that the C locale changes the answer: `/opt/homebrew/bin/bash` counts bytes
under `LANG=C` and characters under a UTF-8 locale. Settle such a question with the
locale set explicitly.

## The block terminator is an ordinary identifier, so an OPTIONAL tail eats it

Lua's `Return` was spelled

    Return = KwReturn [ ExprList ] ;

and `end`, `else`, `elseif` and `until` are perfectly good `Id`s. So

    if n == 0 then return end

parsed as "return the variable named `end`", the `if` then consumed the NEXT `end`,
and the whole file slid one keyword out of step. **Nothing failed at the mistake.**
The parse ran on to the end of the file and reported `450:1 (EOF)` — a position with
no relation to the cause, in a file whose first four hundred lines are correct. Four
of the thirty-three files in Lua's own test suite died this way, and the reported
positions (three EOFs and one stray `)`) looked like four unrelated bugs.

The fix is the zero-width guard the block rule already had:

    Return = KwReturn [ NotEnders ExprList ] ;

The shape generalizes to any language where a block is closed by a WORD rather than a
punctuation mark, and where some construct takes an optional trailing expression:
`return`, `break value`, `yield`, a bare `raise`. Python has the same exposure through
`in`: `for x, in [(1,)]` had its `{ "," TargetItem }` repetition match the comma and
then read `in` as a second target name. Note that a `!"in"` lookahead fixes neither
case — it skips whitespace (see the `!"literal"` entry above) and it matches a mere
PREFIX, so it would also reject the perfectly good name `information`. Both guards
here read the whole word and compare it, and both deliberately omit the SOFT keywords
(`match`, `case`, `type`, `_`), which are real names.

## Two brackets that look like a target list and are not

`(1).__class__ = MyInt` and `[list][0]: type` open with the same bracket a tuple
target does, and `TupleTarget` has to be tried BEFORE the general target (otherwise
`(a, b) = t` parses as one grouped target and never unpacks). So `TupleTarget` matched
`(1)`, committed, and the `.__class__` was left with nothing to attach to — PEG never
re-enters a rule that succeeded. A guard that declines the tuple reading when a suffix
is glued to the closing bracket lets the general target read the whole thing:

    TupleTarget = "(" … ")" NoTgtSuffix | "[" … "]" NoTgtSuffix ;

The scan must stop at a NEWLINE: the next statement may perfectly well open with `(`
or `[`, and treating that as a suffix of the previous one is the same class of bug in
the other direction.

The two brackets are also not interchangeable once you get there. `[g] = [4]` unpacks
and binds `g` to `4`; the parenthesized `(g) = [4]` binds the whole list. A single
`makeTargetList` for both gave the wrong answer for one of them with no parse error.
And the trailing comma has to be RECORDED, not merely tolerated: `x, = t` and
`for x, in [(1,)]` are ONE-element tuple targets that unpack, and a target list that
dropped the comma degraded them to plain targets — again, a wrong answer, no error.

## The reference corpus is not always valid in the version you have

Two of the thirty python files carry code the interpreter on this machine refuses:

    $ python3 -c "compile(open('test_listcomps.py').read(), 'x', 'exec')"
    SyntaxError: iterable unpacking cannot be used in comprehension   # [*i for i in …]
    $ python3 -c "compile(open('test_patma.py').read(), 'x', 'exec')"
    SyntaxError: invalid syntax                                       # case +0:

Both are deliberate negative tests sitting in a file the sweep marks `parse`, and
python's corpus carries a blanket "must parse" expectation with no negative-test
convention encoded — so the ceiling for that corpus is 28/30, not 30/30. Settle this
with the interpreter before contorting a grammar: it took one `compile()` call, and it
is the difference between a real gap and an impossible one.

The traffic runs the other way too. `except EOFError, TypeError, ZeroDivisionError:`
reads like Python 2 and is rejected on sight by anyone who remembers it — but PEP 758
made it legal in 3.14, which is what the corpus targets, and `python3` on this machine
accepts it. Check the version the corpus tracks, not the version you learned.

## A dropped sub-production must be item-NEUTRAL, tag included

An f-string's nested format spec — the `:0` of `f'{value:{width:0}}'` — is parsed and
thrown away, so the obvious spelling reuses the chunk production that already exists:

    FSpecSub = ":" :whitespace() { FSpecSubField | FSpecChunk } … ;

`FSpecChunk` carries `push({chunk: up.in})`. The discarded sub-spec therefore pushed
its text onto the value stack, where the ENCLOSING field's `takeAll()` collected it and
appended it to the width. `f'{value:{width:0}.{prec:1}}'` rendered one hundred columns
wide instead of ten, with no error anywhere and both engines agreeing. A production
whose whole purpose is to be dropped needs its own untagged chunk rule — this is the
"tagging a sub-rule changes how many items its PARENT pops" trap met from the other
side, where the sub-rule was tagged all along and the new parent is the one that
cannot afford it.

## A GROUP of alternatives is committed exactly like an ordered choice

The ordered-choice rule at the top of this file is usually met between the
alternatives of a named production. It applies just as hard to an INLINE group,
and there the mistake is much easier to make because the group looks like a
convenience rather than a decision. Dart spelled a late declaration

    VarLate = KwLate ( KwFinal Type | KwFinal | KwVar | Type ) DeclTail ;

and `late final y = 1;` did not parse. The group matched `KwFinal Type` with
`Type` = the identifier `y`, `DeclTail` then found the `=` where it wanted a
name, and the sequence failed — without the group ever being asked to try its
second alternative. The neighbouring `late final int y = 1;` works, which
presents as "the type is somehow required".

Give each shape its own full alternative, exactly as the `FieldMember` line
above it already did:

    VarLate           = VarLateFinalTyped | VarLateFinal | VarLateVar | VarLateTyped ;
    VarLateFinalTyped = KwLate KwFinal Type DeclTail ;
    VarLateFinal      = KwLate KwFinal      DeclTail ;

The same shape hides in a greedy OPTIONAL, which is the one-alternative case of
it. `ExtensionOn = KwExtension [ EXName ] [ SkipAngle ] KwOn …` cannot parse
Dart's anonymous `extension on Object { … }`: `EXName` is an `Id`, `on` is a
perfectly good identifier, so the optional consumes the keyword and `KwOn` then
fails on the type. Two alternatives (`KwExtension EXName … KwOn …` and
`KwExtension … KwOn …`) is again the fix. Roughly twenty Dart corpus files sat
behind that single pair of brackets.

## An "empty" alternative added to a shared body rule swallows ordinary statements

Dart's `external void f(Object? o);` has no body, so the obvious move is to add
the no-body form to the rule every function body goes through:

    FnBodyKind = FBlock | FArrow | FSemi ;
    FSemi <~~ push({ebody: makeConst(null)}) ~~> = ";" ;

That is wrong in a way no parse error reports. `FuncPlain = FName ParamList
FnBody`, so the ordinary statement

    c1();

now parses as a body-less DECLARATION of a function named `c1` — and since
`FuncDecl` is tried before `ExprStmt`, every call statement of a nil-ary
function silently stopped calling anything. The symptom was a closure test
reporting the wrong counter value, which sends you into the closure machinery.

Confine the permissive form to the context that needs it. Here that is a
production reached only under the `external` / `abstract` modifier:

    TopModDecl = TopMod ( ExternDecl | StatementBody ) ;

The general rule: an alternative whose right-hand side is a strict PREFIX of
what some other statement form already matches must not live in a rule that
form goes through.

## A speculative Target is 2^depth when its seeds are expensive

`Assignment = Target AssignOp Expression` and `IncDec = Target ("++"|"--")` both
parse a whole `Target` before they can tell whether the operator they need is
even there. That is harmless while `Target`'s seed is an identifier. The moment
object creation and a parenthesized expression become seeds — which is what
makes `new A().field = 42`, `(mock as dynamic).mood = 'x'` and `(variable)[0] = 0`
assignable, worth about a dozen corpus files — every nested creation is parsed
TWICE: once by the speculative `Target`, once by the `CallMember` that follows
it when the speculation fails. On `language/async_nested/*`, a corpus of
deliberately deep literals, that is 2^depth: one 59-line file went from 0.13s to
9.4s and started hitting the sweep's timeout.

The documented fix applies — a cheap zero-width lookahead in front of both
productions, answering "is there an assignment operator or a `++`/`--` ahead, at
bracket depth 0, before this expression ends". Three details cost measurements:

- **Two caps, not one.** A single counter that charges COMMENTS to the same
  budget as code is a 44-file regression: a statement preceded by nine lines of
  commentary spends the whole budget before reaching its own `=`, the guard says
  "no assignment", and a perfectly good statement stops parsing. Bound the
  scanned CODE with one counter and the raw iterations with another.
- **Stopping at a depth-0 `,` is what makes it cheap** — without it the two
  20 000-element `big_set_literal` tests took 25s each — but the comma of a TYPE
  ARGUMENT list is not the end of the expression. `Indexable<String, Object?>('')[0]
  ??= e` is one target; stopping at its comma lost eight files. Track an angle
  depth (`<` up, `>` down, floored at zero) and only honour the comma outside it.
- String literals must be skipped, or their contents are read as brackets and
  operators.

A false POSITIVE only costs the parse that would have happened anyway, so when
in doubt the guard should say yes. A false negative is a parse failure.

## The corpus is the specification when there is no toolchain

Two Dart questions that look unanswerable without `dart` are answered outright
by the vendored test suite, in prose, inside the files themselves.

`language/constructor/explicit_instantiation_syntax_test.dart` states the
disambiguation rule for `id<T, T>` — is it a type literal or two comparisons? —
and then enumerates every token:

    <continuationToken> ::= `(' | `.' | `==' | `!='
    <stopToken> ::= `)' | `]' | `}' | `;' | `:' | `,'

so `Map<int, bool>` is a value before a `,` and `x == List<int> && y` is a
syntax error in real Dart too. Implementing exactly that rule as an `InstEnd`
guard is worth about thirty files and cannot over-accept.

`language/anonymous_methods/` settles what `receiver.{ … }` means with an
assertion rather than a comment: `final v1 = buffer.{ return foo; };` followed by
`v1.expectStaticType<Exactly<int>>` says the construct is invoked IMMEDIATELY
with `this` bound to the receiver, not that it builds a closure.

Read the corpus before filing a semantics question as unanswerable.

## `var x = 0` types the slot under -frozen, not just a parameter

The entry above records that a reassigned function PARAMETER keeps the type it
was first called with. A plain local has the same rule: `var oneV = 0` followed
by `oneV = callExt(…)` fails with `variable 'oneV' has type number and cannot
hold a object` — under `-frozen` only, behind a green goja run and a green
`--full` ratchet for the interpreter half. Start any slot that will hold a
handle as `anytype`.

## A grammar script has no exponent literals: `1e308` kills `-frozen` only

Write `1.7976931348623157e308` in a tag script and goja stays green while the
**whole grammar becomes unparseable under `-frozen`**, with the unhelpful
"cannot parse script". The number rule the script dialect uses is
`lib/numbers.abnf`'s `DecLit`, which is MetaJS's, and it **has no exponent
form** — so `1e308` lexes as the number `1` followed by the name `e308`, and the
script that was fine a moment ago is now a syntax error.

Spell it as a runtime call instead: `parseFloat("1.7976931348623157e308")`, or
`Math.pow(2, -52)` for the small ones.

Two agents hit this independently on 2026-07-27, one writing `Double.MAX_VALUE`
for Kotlin and one writing PHP's float epsilon, which is why it is recorded here.
It belongs to the "green under goja, dead under `-frozen`" class along with the
`anytype` slot rule above — and it is the more confusing member, because the
failure is not at the offending line and does not name a value at all.

## Frozen type-pinning bites LOCALS too, not just parameters

The documented rule is that a reassigned parameter keeps the type it was first
called with. The same applies to a plain `var`, and it is easier to hit by
accident because nothing about the call site warns you:

    var b = toPrimitive(x)     // a string on the first call
    b = parseFloat(b)          // -frozen: "variable 'b' has type string and
                               //           cannot hold a number"

Green under goja, dead under `-frozen`, and the error names a variable rather
than the construct that caused it. Found in a BigInt relational helper, where
`1n < [2n]` took a string out of ToPrimitive and then wanted a number.

Declare any slot that will hold more than one type as `anytype`, or split the
two meanings into two names.
