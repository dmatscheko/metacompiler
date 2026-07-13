# MetaCompiler

A generic compiler frontend or in other words an at-runtime compiler-compiler.

This is
* a parser that parses a language specification written in annotated EBNF
* and a compiler that (at runtime) compiles the parsing result to a new parser and compiler.

The new parser and compiler are equivalent to the first ones and can replace them at runtime or can be used in parallel.

This means it is a parser and compiler with a polymorphic core.

This system should allow to define and use compiler for arbitrary computer languages, only by specifying them in plain text in annotated EBNF.

## Table of Contents
- [MetaCompiler](#metacompiler)
  - [Table of Contents](#table-of-contents)
  - [What is an annotated EBNF (ABNF)?](#what-is-an-annotated-ebnf-abnf)
    - [Small Example](#small-example)
  - [Documentation](#documentation)
    - [Build / Usage](#build--usage)
    - [Running the tests](#running-the-tests)
    - [High level overview](#high-level-overview)
      - [Default processing steps](#default-processing-steps)
    - [ABNF Syntax](#abnf-syntax)
      - [EBNF of EBNF](#ebnf-of-ebnf)
      - [EBNF of non-context-free EBNF](#ebnf-of-non-context-free-ebnf)
      - [EBNF of ABNF](#ebnf-of-abnf)
      - [Almost Common syntax](#almost-common-syntax)
    - [Parser commands](#parser-commands)
      - [Line plus inline commands](#line-plus-inline-commands)
      - [Line commands](#line-commands)
      - [Inline commands](#inline-commands)
    - [Exposed JS API](#exposed-js-api)
      - [General](#general)
      - [Console output](#console-output)
      - [Storage](#storage)
      - [Strings](#strings)
      - [Variables](#variables)
        - [Local variables](#local-variables)
        - [Global variables](#global-variables)
      - [The stacks](#the-stacks)
        - [Local stacks](#local-stacks)
        - [Global stack](#global-stack)
      - [Parser script API](#parser-script-api)
      - [Compiler API](#compiler-api)
      - [Parser and Compiler ABNF a-grammar API](#parser-and-compiler-abnf-a-grammar-api)
        - [Builder functions](#builder-functions)
        - [Serializer functions](#serializer-functions)
        - [Grammar functions](#grammar-functions)
        - [OperatorID Constants](#operatorid-constants)
        - [RangeType Constants](#rangetype-constants)
        - [CharType Constants](#chartype-constants)
        - [NumberType Constants](#numbertype-constants)
      - [LLVM IR API](#llvm-ir-api)
  - [Further Examples](#further-examples)
    - [ABNF of ABNF to a-grammar](#abnf-of-abnf-to-a-grammar)
    - [Its output, when it gets applied on itself:](#its-output-when-it-gets-applied-on-itself)
  - [Links](#links)

## What is an annotated EBNF (ABNF)?

* The EBNF defines the syntax (the grammar) of another language.
* The annotations in the EBNF define the semantic (the meaning) of the other language.
* The combined format is therefore called annotated EBNF or ABNF.

This means ABNF is a meta language. A language to describe the syntax and semantic of another language.

### Small Example

This is a fully working calculator for addition and multiplication. It can parse its input and calculate the output, all while taking into account point before line calculation and bracketing:

<details>
  <summary>Click to expand!</summary>

```javascript
:title("Calculator interpreter (global stack)") ;


:startRule(Expression) ;

Expression  =
    Term
    {
        ( "+" | "-" )           <~~pushg(up.in)~~>
        Term                    <~~
                                var t2 = popg()
                                var op = popg()
                                var t1 = popg()
                                println("calculating " + t1 + " " + op + " " + t2)
                                var res = (op=="+") ? t1 + t2 : t1 - t2
                                pushg(res)
                                ~~>
    }
    ;

Term        =
    Factor
    {
        ( "*" | "/" )           <~~pushg(up.in)~~>
        Factor                  <~~
                                var t2 = popg()
                                var op = popg()
                                var t1 = popg()
                                println("calculating " + t1 + " " + op + " " + t2)
                                var res = (op=="*") ? t1 * t2 : t1 / t2
                                pushg(res)
                                ~~>
    }
    ;

Factor      =
    (
        "("
        Expression
        ")"
    )
    |
    Neg
    |
    // The grammar allows a comma as decimal separator, parseFloat() does not.
    Number                      <~~pushg(parseFloat(up.in.replace(",", ".")))~~>
    ;

// Unary minus negates the factor behind it (and '- -x' works through the recursion).
Neg         = "-" Factor        <~~pushg(-popg())~~> ;

Number      = ( "0" | "1"..."9" { "0"..."9" } )
              [ ( "." | "," ) "0"..."9" { "0"..."9" } ] ;


:startScript(~~
    c.compile(c.asg)
    println("\nRESULT: " + popg() + "\nFormula was: " + ltr.in)
~~) ;
```

When fed the input `1 + 3 * (3 + (7 - 1) * 2)`, it outputs `46`.
</details>

## Documentation

### Build / Usage

```
go build .
./mec languages/brainfuck-interpreter.abnf tests/brainfuck-test-2.txt
```
or
```
go run . languages/abnf-of-abnf.abnf languages/abnf-of-abnf.abnf
```
or, to parse a C file with the full C99 grammar:
```
go run . languages/c99-parser.abnf tests/c99-test-1.c -q
```

### Running the tests

The test suite is `.vscode/launch.json`: every configuration with an `args` list is one
grammar/input run. `test.sh` runs the whole matrix and checks the two invariants the
project guarantees:

```
./test.sh                # run everything (parallel, about a minute with a warm script cache)
./test.sh -j 1           # any worker count; default is one worker per CPU, 1 = sequential
./test.sh --filter kotlin # run only entries whose name or args contain "kotlin"
./test.sh --list         # list the matrix entries without running them
./test.sh --help         # full option list
```

Each entry is run twice - once with the default goja tag engine and once goja free
(with `-frozen` appended, see [the frozen bootstrap](#the-frozen-bootstrap-running-without-goja))
- and the script verifies that

1. every `-q` (quiet) run produces **byte-identical** output under goja and `-frozen`, and
2. the exit status is correct: the language tests are self checking (exit 0 means the
   program's own checks passed), so ordinary entries must exit 0 on both engines, while
   the by-design failures - the entries whose name says `SHOULD FAIL` plus the
   `smaller-match-first` and `infinite-loop` grammar guards - must exit non-zero on both.

`test.sh` exits 0 iff the whole matrix is green. It needs only `go` (to build the
compiler) and a POSIX shell; `timeout`/`gtimeout`/`perl`, if present, guard each run
against hangs.

#### The second test group: full-syntax ratchets (`--full`)

The goal is to support the **full** languages, and `tests/<lang>-test-full.<ext>` is the
measure: one hand-written file per language that walks the whole practical syntax of the
modern language (organized after its official spec, in self-contained `SECTION` chunks —
see the anatomy notes in [tests/js-test-full.js](tests/js-test-full.js)). These files are
**not** part of the default matrix; today the grammars abort on them by design.

```
./test.sh --full             # per grammar: every language area that does not work yet
./test.sh --full --filter js # probe a single language
```

The probe runs each grammar over its file and, whenever the grammar chokes, removes the
section around the error and retries — so the report lists *every* unsupported area, and
distinguishes "does not parse" from "parses, but not implemented: X". A file that runs
green untouched is reported `FULL` and cross-checked goja vs `-frozen`: that language is
ready to graduate into the default matrix. The self-defined languages (metajs, tinyc,
lisp, calculator, brainfuck) have no `-full` file — their feature tests already cover
their entire spec.

#### The example languages in languages/

Every language in `languages/` has at least one interpreter and one compiler (the test
programs live in `tests/`, next to the feature-test grammars) (only the full
C99 grammar is parse only). The compilers
generate LLVM IR and immediately execute it with the built-in IR interpreter (`llvm.Run`),
and the test programs are self checking: the exit code of the run is 0 on success.

| Language     | Interpreter                                                  | Compiler (to LLVM IR)        | Test inputs                                       |
| ------------ | ------------------------------------------------------------ | ---------------------------- | ------------------------------------------------- |
| Calculator   | calculator-interpreter-1.abnf, calculator-interpreter-2.abnf | calculator-to-llvm-ir.abnf   | calculator-test-1.txt                             |
| Brainfuck    | brainfuck-interpreter.abnf                                   | brainfuck-to-llvm-ir.abnf    | brainfuck-test-1.txt, brainfuck-test-2.txt        |
| TinyC        | tinyc-interpreter.abnf                                       | tinyc-to-llvm-ir.abnf        | tinyc-test-1.txt, tinyc-test-2.txt                |
| Lisp         | lisp-interpreter.abnf                                        | lisp-to-llvm-ir.abnf         | lisp-test-1.txt                                   |
| MetaJS       | metajs-interpreter.abnf                                      | metajs-to-llvm-ir.abnf       | metajs-test-1.js, metajs-test-2.js, metajs-fail-test.js, metajs-fail-test-undeclared.js, metajs-fail-test-anytype.js |
| C (subset)   | c-interpreter.abnf                                           | c-to-llvm-ir.abnf            | c-test-1.c                                        |
| Java         | java-interpreter.abnf                                        | java-to-llvm-ir.abnf         | java-test-1.java                                  |
| Kotlin       | kotlin-interpreter.abnf                                      | kotlin-to-llvm-ir.abnf       | kotlin-test-1.kt                                  |
| Go           | go-interpreter.abnf                                          | go-to-llvm-ir.abnf           | go-test-1.go                                      |
| Python       | python-interpreter.abnf                                      | python-to-llvm-ir.abnf       | python-test-1.py                                  |
| JavaScript   | js-interpreter.abnf                                          | js-to-llvm-ir.abnf           | js-test-1.js, js-test-classes.js                  |
| TypeScript   | typescript-interpreter.abnf                                  | typescript-to-llvm-ir.abnf   | typescript-test-1.ts                              |
| Lua          | lua-interpreter.abnf                                         | lua-to-llvm-ir.abnf          | lua-test-1.lua                                    |
| Ruby         | ruby-interpreter.abnf                                        | ruby-to-llvm-ir.abnf         | ruby-test-1.rb                                    |
| PHP          | php-interpreter.abnf                                         | php-to-llvm-ir.abnf          | php-test-1.php                                    |
| C#           | csharp-interpreter.abnf                                      | csharp-to-llvm-ir.abnf       | csharp-test-1.cs                                  |
| Dart         | dart-interpreter.abnf                                        | dart-to-llvm-ir.abnf         | dart-test-1.dart                                  |
| Swift        | swift-interpreter.abnf                                       | swift-to-llvm-ir.abnf        | swift-test-1.swift                                |
| C preproc.   | c-preprocessor.abnf (source-to-source, via `-pipe`)          | -                            | c-test-preprocess.c                               |
| C99          | c99-parser.abnf (parses only)                                | -                            | c99-test-1.c                                      |

Every language is a well chosen executable subset; the exact feature list and the
deliberate deviations are documented in the :description() of each interpreter grammar.
The test programs are valid programs of the real languages: c-test-1.c compiles and
passes under clang, go-test-1.go under go run, python-test-1.py under python3,
java-test-1.java under javac/java, and metajs-test-1.js under node (with println/printf
shims) - all with exit code 0, matching both of our engines.
The C subset has real pointers, arrays, globals, structs (nested values, self
referencing pointers for linked lists, . and -> along whole paths) and a switch with
real fallthrough, and
compiles to plain integer LLVM IR (llvm.Run) - member access is a getelementptr into a
real LLVM struct type. The object and dynamic languages (Java with
classes/records/single inheritance/switch on int and String, Kotlin with when/string
templates/properties/elvis/safe calls/lambdas with map/filter/sumOf and
until/downTo/step ranges,
Go with structs/methods/multiple returns/switch/maps/function literals with
closures/defer, Python with real INDENT/DEDENT
significant indentation/f-strings/dicts/slices/list comprehensions, MetaJS with the
JS switch and prefix ++/--) compile to handle threaded
IR on the MetaJS runtime (llvm.RunJS): objects, strings, lists and closures live behind
i64 handles, methods dispatch through the shared class descriptor convention (js_mcall,
with a __super chain for Java's inheritance). Go maps and Python dicts share one handle
shape (two parallel key/value arrays, so the insertion order is deterministic in both
engines).
The same handle runtime backs the dynamic and scripting family - JavaScript (with
classes/for-in/destructuring), its type-erasing sibling TypeScript, Ruby, Lua, PHP, C#,
Dart and Swift - which all reuse the shared class descriptor, closures (js_closure) and
deterministic key enumeration (js_keys, powering for-in / pairs / range-over-map). None
of them needed a new runtime primitive.
MetaJS pins types: every variable must be declared (var/let/const) before use, and a
variable's type class is pinned by the first non-undefined, non-null value it holds -
enforced by both engines, positively tested by metajs-test-2.js and demonstrated by
metajs-fail-test.js and metajs-fail-test-undeclared.js (both SHOULD FAIL). A variable
that is meant to change its type declares it: `var v = anytype` starts as undefined and
is exempt from pinning (assigning anytype to an existing variable is an error,
metajs-fail-test-anytype.js). The rule was prototyped in a separate 'Typed MetaJS'
dialect pair that has since been folded into the base language (the js_tdecl/js_tset
extern names remember it).

The big-language grammars (Java, Kotlin, Go, Python, C and MetaJS -
interpreters and compilers) draw their common machinery from `languages/lib/`: the
startScript begins with `include("lib/interp-core.js")` (scopes, the break/continue
protocol, the expression and statement thunk builders) or `include("lib/compile-core.js")`
(the LLVM module, handle constants, on-demand externs, the loopStack and the emitter
builders). Where languages genuinely differ the behavior is a knob on the library's
`core` object - Java sets its string-concatenating `+` and array `.length`, Go the blank
identifier `_`, `nil` wording, map-aware indexing and defer frame hooks, Kotlin the
implicit-this name fallback, Python its truthiness and assignment-declares-local rule,
C its nonzero-int conditions - and anything genuinely language-specific (Java's
inheritance dispatch, Go's multi-assign, Python's elif chains, MetaJS' type boxes,
C's arena addressing) stays in the grammar file as a plain-assignment override of the
library name (never a `function` declaration: the two engines install those in opposite
order relative to the include). The freezer inlines the emitter library's `include()`
lines into the bootstrap snapshot, so `-freeze` keeps working and the snapshot stays
self contained.

The same grammars also share their token PRODUCTIONS via grammar level `:include()`
fragments in `languages/lib`: cstyle-comments.abnf (whitespace with // and /* */),
ident.abnf / ident-dollar.abnf (identifiers and the KwEnd keyword boundary, with and
without '$'), dq-strings.abnf (quoted strings with escapes), numbers.abnf (IntLit and
DecLit) and bools.abnf - each a handful of productions whose tags call the shared
makeConst/unescapeJs builders, so one fragment serves interpreters and compilers of
every language alike. Fragments reference the names Whitespace and KwEnd as
'expected names' that the including grammar (or a sibling fragment) provides; a name
defined twice is a hard error, so porting mistakes fail loudly. Note that :include()
resolves its path when the grammar is USED, relative to the parsed input file - which
works here because the test inputs live next to the grammars - and the freezer
inlines grammar includes the same way as script includes. TinyC, Lisp, Calculator and
Brainfuck share almost nothing with the libs and remain deliberately self-contained.

Besides those there are the self describing grammars (abnf-of-abnf.abnf, ebnf-of-ebnf.bnf,
ebnf-of-abnf.bnf, tiny-self-parse.bnf, brainfuck-parser.bnf and tinyc-parser.bnf as syntax
only variants), the feature
demos (tlv-test, parser-script-test, include-test, parse-and-compile-from-js, llvm-ir-tests,
negation-test for the ! and @b forms),
and two grammars that deliberately fail to demonstrate the parser limits
(smaller-match-first-test, infinite-loop).

MetaJS is special: it is the restricted JavaScript subset that the annotation scripts of all
grammars are written in (see the description in languages/metajs-interpreter.abnf for the exact
language). That closes the loop for the frozen bootstrap below: every grammar above - all
interpreters and compilers of all languages - also runs completely goja free with -frozen.
It also means the discipline rules apply to the tag scripts themselves: under -frozen they
declare and assign through the pinning externals (js_tdecl/js_tset), while goja enforces
only the declaration rule (the scripts are compiled as strict-mode JS; a JS engine cannot
pin types) - so the -frozen leg of a test sweep is the conformance check for tag scripts.
A variable that legitimately hops between types (takeAll popping mixed pushes, an
interpreter folding obj.name.length across object, string, number) declares it:
`var v = anytype` starts as undefined and is exempt from pinning. The marker is a
plain host global, so the declaration stays valid JavaScript for goja (whose anytype
is simply undefined - it cannot pin types anyway).

#### The frozen bootstrap (running without goja)

Normally the annotation scripts run on goja (a JS engine in Go). The frozen mode replaces it:

```
./mec -freeze languages/metajs-to-llvm-ir.abnf    # snapshot: abnf/jsagrammar.go + abnf/jsbootstrap.ll
go build .                                    # embed the snapshot
./mec languages/tinyc-to-llvm-ir.abnf tests/tinyc-test-1.txt -q -frozen
```

`-freeze` runs metajs-to-llvm-ir.abnf once under goja and lets it compile its *own* tag scripts
(plus its emitter library) to one LLVM IR module, keyed by tag source text. With `-frozen`,
the Go core then executes every annotation script goja free:

1. The script is parsed with the frozen a-grammar (pure Go).
2. Its ASG is walked bottom up; each tag of that walk runs a snapshotted compiler closure on
   the built-in IR machine, which emits the IR module of the script.
3. The emitted module runs on the MetaJS handle runtime (abnf/jsrt.go): every dynamic value
   is an i64 handle into a Go side table, the js_* externals implement the JS semantics, and
   a reflection bridge exposes the whole host API (up/ltr, the stacks, c.*, abnf.*, and the
   llvm.* builder objects with their methods) - including JS closures that scripts push onto
   the stacks, which survive as IR function handles.

Because a script's compiled module depends only on its source and the snapshot, step 2 is
cached on disk across runs (`<user cache dir>/mec/scripts`, about 1MB per grammar): later
`-frozen` runs reload the emitted `.ll` text instead of recompiling every tag script, which
roughly halves their startup cost. A refreeze changes the snapshot hash and with it every
cache key, so entries never go stale; `MEC_SCRIPT_CACHE=off` disables the cache,
`MEC_SCRIPT_CACHE=<dir>` relocates it.

All grammars pass their self checking runs with identical output in both modes;
frozen mode is roughly an order of magnitude slower (threaded IR on an interpreter instead
of a JS VM). goja is only needed to (re)create the snapshot after changing metajs-to-llvm-ir.abnf.

#### Tracing and diagrams (one system through all languages)

Because every dynamic language compiles to handle-IR whose semantics run through the
js_* externals of one shared runtime, a single hook there traces Java, Kotlin, Go,
Python, Lisp and MetaJS programs alike - under goja and under -frozen, with byte
identical event streams (abnf/trace.go):

```
./mec languages/java-to-llvm-ir.abnf tests/java-test-1.java -q \
      -cfg java.dot -trace java.jsonl   # run + control flow graph + event stream
./mec -render calls -trace java.jsonl > calls.dot   # dynamic call graph (counts)
./mec -render vars  -trace java.jsonl > vars.dot    # which function touches which variable
```

`-cfg` writes the basic block graph of every executed IR module as Graphviz DOT
(Mermaid when the file name ends in .mmd) - the compilers literally build the
control flow, so the diagram is the program's real branch structure, for the
integer-IR languages (TinyC, C subset) too. `-trace` streams runtime events as
JSON lines: stmt when a statement begins, decl/read/write for scope variables,
mread/mwrite for object members
(with stable value handles), call/ret with resolved names (a method closure gets
its name when the class descriptor is built). Only the program runtime is traced;
the tag scripts of the grammars stay silent, which is why the goja and -frozen
streams of one program are identical. `-render` turns a stream into DOT on stdout.

Events carry source lines: every tag sees the position of its node as up.pos, and
the compiler grammars wrap their Statement production with stmtPos()
(lib/compile-core.js), which plants a js_srcpos marker per statement - but only
while -trace, -cfg or -callgraph is active (the tag scripts read the switch as
c.tracing), so
the emitted IR is untouched otherwise. The runtime remembers the marker as the
current position and stamps it on every event; -cfg labels the blocks with their
line ranges (entry L15-16).

`-callgraph` extracts a STATIC call graph from the compiled module - the written
calls, whether they run or not. The integer-IR languages have direct call
instructions; for the handle-IR languages the names are recovered from the IR
patterns: method names and js_scope_get callee names are string constants, a
closure pairs with its source name where it is stored (js_tdecl('fib',
js_closure(N, ...)), also method slots of class descriptors), and 'new Counter'
falls out of the js_get(class, '__ctor') chain. And because the dynamic languages
never resolve names at compile time, a file that calls functions defined in OTHER
files still compiles alone - so a whole codebase graphs file by file:

```
rm -f cg.jsonl
for f in src/*.js; do
  ./mec languages/metajs-to-llvm-ir.abnf "$f" -q -callgraph cg.jsonl || true
done                                   # .jsonl appends; a run without main() may exit 1
./mec -render static -trace cg.jsonl > codebase.dot
```

The result clusters the definitions per source file, merges the nodes by name (a
cross-file call connects to the defining file's node), draws undefined callees
dashed, and labels edges with the number of call sites. A non-.jsonl -callgraph
path writes the DOT of one run directly. Same-named methods of different classes
merge into one node (class attribution is a planned refinement). The flags
default to off and change nothing when unused; interpreter-grammar runs and a
stepper are the planned next layers.

`-main NAME` names the entry-point function a compiled program calls instead of
its default. It lets a program whose entry is not named `main` run, and lets a
real-world file that has no `main()` be driven from a chosen function - handy for
rooting a `-callgraph` / `-trace` at a specific function. Grammars read it as
`c.mainName`; it is wired into Kotlin, Go, JavaScript, TypeScript, MetaJS, Dart,
C, TinyC (a top-level function), and Java and C# (a static method of `Main` /
`Program`). Each falls back to its own default entry when the flag is absent
(`main` for most, `Main` for C#), so runs without the flag are unchanged.
Languages that just run their top level (Python, Ruby, Lua, PHP, Swift, Lisp) have
no distinct entry to name.

The value may also be a **snippet** of the target language rather than a bare
name: if it contains a `(` it is parsed with the same grammar (from its `Statement`
rule, via the `c.parseFrom` / `-StartRule` override) and run in the program's
already-populated scope, so it can call the file's functions with real arguments -
`-main 'greet("one", 2, three)'`, or `-main 'exitProcess(check())'` to choose the
exit code. Implemented for Kotlin (both engines, byte-identical). The snippet is a
single statement.

#### Grammar linting (-verify)

`-verify` checks a grammar for name consistency without running it and exits
(it does not process a target file):

```
./mec languages/java-interpreter.abnf -verify
```

It reports three things, each with the source line: **undefined** names - an
identifier used with no production defining it (a typo, or a missing
`:include()`); **malformed** ranges - a rune or byte range whose bound is not
exactly one rune/byte, so `"ab"..."z"` would silently read as `"a"..."z"`; both
are errors and exit non-zero - and **unreachable**
productions - defined but never reached from the start rule through identifiers
(a warning). Reachability seeds from the start rule and from command parameters
like `:whitespace(Whitespace)`, so a whole whitespace/comment sub-grammar counts
as reached. Grammars that pull in shared `:include()` fragments are assembled
first (the fragments define more than any one language uses, so only the
grammar's own productions are reported as unreachable); startScript-driven
grammars with no `:startRule()` skip the reachability check. It is the modern
successor of an old standalone verifier, rewritten to walk the current
a-grammar by name (it resolves nothing and mutates nothing).

A companion flag, `-pretty`, compiles the first file (the grammar) and prints its serialized
a-grammar as a pretty Go literal (one brace per line, the runtime `:origin()`
stamp dropped so it matches `abnf/agrammar.go`'s form), then exits - handy for
inspecting a compiled grammar or regenerating the example dump above.

### High level overview

There are two basic components: A __parser__ and a __compiler__.

The __parser__ processes the target text top down, so from the topmost grammar rule, over its currently matching branches, to the leaves that match. The leaves are fixed strings that must match (also called token or terminal symbols). The parser thereby generates an ASG (abstract semantic graph). This is similar to an AST (abstract syntax tree), but the parser attaches code, provided by special grammar rules (`Tags`) to the group of matching strings (`Token`). Because of those special rules, the grammar is called annotated grammar or a-grammer.

The only hirarchy or grouping inside the ASG is done by the `Tag` rules. Each `Tag` can contain multiple other `Tags` and of course the strings (`Token`) that were found in the target text. The `Tags` contain those child `Tags` and `Token` in the `Sequence` of their occurence in the target text.

The __compiler__ processes the ASG, generated by the parser, bottom up. It first finds the outermost leaves and collects the data from there by executing attached code. This data is accumulated until the compiler reaches the topmost point. There the code inside the a-grammer can decide what to do with the collected data and how to represent it.

#### Default processing steps

The above parser and compiler are not only used once but twice. The first time, the ABNF of a new language is parsed and then compiled to a new a-grammar. The second time, the new a-grammar of the new language is used by the parser and compiler, to again parse and also compile files, that are already written in the new language.

The whole process can be shown a bit more formal as follows:

The files are given as positional arguments; flags (`-q`, `-v`, `-frozen`, ...) may
appear anywhere among them. The first file is the grammar, every further file a stage:

```
./mec grammar.abnf program.txt                  # one stage
./mec meta.abnf lang.abnf program.x             # two stages (meta builds lang)
./mec meta.abnf lang.abnf mid.x program.y       # and beyond
```

This is the input to the default main process:
* `initial-a-grammar` = Default annotated grammar of annotated EBNF.
* `file[1]` = The content of the first file (a grammar in ABNF).
* `file[i]` = The content of the i-th file, processed by the previous stage's grammar.
* `*-a-grammar-startScript` = The code, defined via `:startScript()` inside the grammar / the ABNF. This code starts the rest of the compile process during the compile step.

This is how that input is processed (with `grammar[0]` = `initial-a-grammar`):
1. `parse(grammar[i-1], file[i])`  = `file[i]-ASG`.
2. `compile(file[i]-ASG, grammar[i-1]-startScript)`  = `grammar[i]`.

Each stage feeds its compiled `grammar[i]` (which must be an a-grammar, except at the last stage, where it may be a final result such as an emitted module or a program's exit code) into the parser for the next file. After the last file, if its compiled grammar is *startScript-only* (has a `:startScript()` but no `:startRule()`, so it takes no input), that startScript is run on empty input - which lets the last file be a grammar that builds and runs a module entirely in JS. So `./mec llvm-ir-tests.abnf` runs on its own, and `./mec abnf-of-abnf.abnf llvm-ir-tests.abnf` builds that grammar with a meta-layer first and then runs it.

Per-stage diagnostics: `-v` shows the ASG and compiled result of every stage, `-v<n>` of stage `n` alone (1-indexed); `-vv` / `-vv<n>` add the parser+compiler trace. `-slot<n> <v>` compiles stage `n` with tag slot `<v>` (default 0), for grammars whose tags carry more than one code slot.

### Multi-file programs: project imports (`-i`)

An import that names neither a builtin (`kotlin.math`, `java.util`, Python's `math`, ...) nor an external library can name a **project file**. `-i DIR` adds an include root (repeatable, searched in order); the directory of the program itself is always searched first:

```
./mec -i tests/imports languages/kotlin-interpreter.abnf tests/kotlin-test-multifile.kt
./mec -i tests/imports languages/java-to-llvm-ir.abnf   tests/java-test-multifile.java
./mec -i tests/imports languages/python-interpreter.abnf tests/python-test-multifile.py
```

The dotted import path maps to a relative file path per language: Kotlin `import geo.Vec` tries `geo/Vec.kt`, then `geo.kt`; Java `import util.Pair;` tries `util/Pair.java`, then `util.java`; Python `from mypkg.helper import fn` tries `mypkg/helper.py`, then `mypkg.py`. The found file is parsed **with the same grammar** and walked with the same tag scripts, so its declarations register exactly like the main file's (Python module top level executes at import time, like real Python). Each file is imported once - repeats and cycles are no-ops - and warnings inside an imported file carry its own name and line numbers. Deviations: names bind flat (no module objects/namespaces, so Python's `from x import y` style is the natural fit), and a nearer stdlib prefix still wins over a project file of the same name.

Resolution order per import: builtin prefix (no-op) → project file (parsed and merged) → `-warn-imports` warning or a clean abort.

### Piping one language's output into another (`-pipe`)

The stages above chain *a-grammars*: each grammar compiles the next file. `-pipe` chains along a second, orthogonal axis - *text*. It splits the command line into independent segments; the text a segment prints becomes the program input of the next segment. So one language can transform the source that another language then consumes:

```
./mec languages/c-preprocessor.abnf prog.c -pipe languages/c-to-llvm-ir.abnf
./mec languages/c-preprocessor.abnf prog.c -pipe languages/c-interpreter.abnf
```

Here [`c-preprocessor.abnf`](languages/c-preprocessor.abnf) is just another language whose output happens to be C source: it expands object-like `#define` macros (and honors `#undef`), passing every other directive through, and prints the result. The C front end downstream then sees the macros already expanded. Because the preprocessor is an ordinary grammar, this generalizes - any language whose output is text can feed any other language, as long as their syntaxes do not collide. Each segment is a full a-grammar chain in its own right, so `meta.abnf lang.abnf prog.x -pipe other.abnf` works too.

An example of this process, done fully inside the `:startScript()` code of an ABNF:

<details>
  <summary>Click to expand!</summary>

```javascript
:title("Parse and compile from JS") ;


:startScript(~~

    // Lets build a new compiler directly from scratch:

    // For this we need an agrammar. This is the information, that the metacompiler needs to actually be a compiler.

    // To create an agrammar, we need an ABNF:
    var ABNF = ":startScript(~\~ println('found me'); c.compile(c.asg) ~\~); :startRule(X); X = ( 'A' | 'B' | 'C' ) <~\~ println('and found the letter ' + up.in) ~\~>;"

    // Now we parse the ABNF and create an ASG from it. We use c.ABNFagrammar as agrammar for this parse step, because we wrote our compiler definition in ABNF (the variable above), and the agrammar c.ABNFagrammar understands ABNF:
    var ASG = c.parse(c.ABNFagrammar, ABNF)

    // Lets print the resulting ASG:
    println("\nASG: " + abnf.serializeRules(ASG))

    // So far we have only parsed our ABNF into an ASG. Now we have to compile the ASG into something useful. We again use the agrammar c.ABNFagrammar, because this is - as stated above - the agrammar that understands ABNF.
    // The compile step of c.ABNFagrammar creates the agrammar and returns it:
    var myAgrammar = c.compileRunStartScript(ASG, c.ABNFagrammar)

    // Lets now print the resulting agrammar:
    println("\nmyAgrammar: " + abnf.serializeRules(myAgrammar))

    // We can now use our own agrammar to parse something new. We parse the letter B and again get an ASG from it:
    var myASG = c.parse(myAgrammar, "C")

    // Lets now print the resulting ASG:
    println("\nmyASG: " + abnf.serializeRules(myASG))

    // And to see a result, we have to compile our new ASG myASG with our grammar myAgrammar:
    println("\nOutput:")
    c.compileRunStartScript(myASG, myAgrammar)

    // Note that the ASG already includes almost all information for compiling. The agrammar is only used for its preamble. In our case this would be the Tag: "<~\~ println('found me'); c.compile(c.asg) ~\~>".

~~) ;
```
</details>

### ABNF Syntax

#### EBNF of EBNF

A basic EBNF syntax looks like this:

```javascript
EBNF        = { Production } ;
Production  = name "=" [ Expression ] ";" ;

Expression  = Sequence { "|" Sequence } ;
Sequence    = Term { Term } ;

Term        = name | token | Group | Option | Repetition ;
Group       = "(" Expression ")" ;
Option      = "[" Expression "]" ;
Repetition  = "{" Expression "}" ;
```

* A `token` is just a quoted string that should occour like that in the EBNF. E.g. all quoted strings inside this EBNF are `token`.
* A `name` is also a string, but without quotes. In the case of EBNF, it defines the name of a `Production` or identifies one, when it is used inside of an `Expression`.

#### EBNF of non-context-free EBNF

This system uses an EBNF syntax that is a bit more capable:

```javascript
:title("EBNF of EBNF") ;
:startRule(EBNF) ;
:whitespace(Whitespace) ;

EBNF        = { Production | LineCommand } ;
Production  = Name "=" [ Expression ] ";" ;

Expression  = Alternative ;
Alternative = Sequence { "|" Sequence } ;
Sequence    = Term { Term } ;

Term        = Name | Group | Option | Repetition | ByteRange | Range
            | NotCharsOfByte | NotCharOfByte | NotCharsOf | NotCharOf | NotToken
            | CharsOfByte | CharOfByte | CharsOf | CharOf | Times | Command ;
Group       = "(" Expression ")" ;
Option      = "[" Expression "]" ;
Repetition  = "{" Expression "}" ;
ByteRange   = Token "..b" Token ;
Range       = Token [ "..." Token ] ;
CharsOf     = "@+" Token ;
CharOf      = "@" Token ;
CharsOfByte = "@b+" Token ;
CharOfByte  = "@b" Token ;
NotCharsOf  = "!@+" Token ;
NotCharOf   = "!@" Token ;
NotCharsOfByte = "!@b+" Token ;
NotCharOfByte  = "!@b" Token ;
NotToken    = "!" Token ;
Times       = CmdNumber [ "..." ( CmdNumber | "" ) ] Group ;

CmdNumber   = Number | Command ;

LineCommand = Command ";" ;
Command     = ":" CmdName "(" [ ( Name | Token | Number ) { "," ( Name | Token | Number ) } ] ")" ;
```
The `Name`, `Token`, `Number`, and `Whitespace` definitions are in [Almost Common syntax](#almost-common-syntax)

* `:title()` is a `Command`. Those commands normally inform the parser about context, but not necessarily influence what has to be parsed in the target text (but they can). This means, the EBNF-variant that is used by this system is _not_ context free. There are commands that can be inline in an `Expression` and there are commands that have to be in their own line, terminated with semicolon (`LineCommands`). Some commands, like the `:whitespace()` command can occour either as inline command or as `LineCommand`. In the case of whitespace, this allows to change what is seen as whitespace and therefore allows to parse strings correctly.
  * The `:title()` command only describes the EBNF via a short title. There is a `:description()` command available that describes the EBNF in more detail.
  * The `:startRule()` command defines the top level EBNF rule.
  * The `:whitespace()` command defines what can be skipped in the target text as whitespace between `token` and `numbers`.
  * See [Parser commands](#parser-commands) for more.
* `number` is a new type of content in the EBNF. It stands for plain unquoted numbers.
* `ByteRange` defines that the a char between (and including) the two `token` should be in the target text. The comparison is done for exactly that single byte.
* `Range` with only one parameter is the same as a `token`. `Range` when used as two `token` with the `...` between, defines that the a char between (and including) the two `token` should be in the target text. That char can be any UTF8 symbol and therefore can use more than one byte.
* `CharOf` is not strictly necessary but shortens some EBNF quite a lot. It stands for any one of the UTF8-chars of the `token`. Exactly one of the chars has to be in the target text.
* `CharsOf` is the same as `CharOf`, but the chars contained in the `token` can occour in any order from zero to infinite times. At least one char has to be in the target text.
* `CharOfByte` (`@b`) and `CharsOfByte` (`@b+`) are the byte versions of `CharOf` and `CharsOf`: they compare single bytes instead of UTF8 chars (useful for binary formats, like the `..b` byte range).
* All four set forms can be prefixed with `!` (`!@`, `!@+`, `!@b`, `!@b+`): they then match exactly the chars (or bytes) that are NOT in the `token`. `!@"\n"` is one char of anything but a line feed, `!@+"<>"` is a whole run without angle brackets.
* `NotToken` (`!token`) is a negative lookahead: it matches _without consuming anything_ when the token does NOT match at the current position. `"if" !"fy"` accepts `if` but not the start of `iffy`.
* `Times` is a number, or a number and `...`, or a number and `...` and another number. Each of the three options followed by a `Group`.
  * __number ( Expression )__: The Expression must occur exactly _number_ times.
  * __number ... ( Expression )__: The Expression must occur _number_ to infinite times.
  * __numberA ... numberB ( Expression )__: The Expression must occur _numberA_ to _numberB_ times.

#### EBNF of ABNF

Annotated EBNF basically only adds tags to the syntax of the above EBNF:

```javascript
:title("EBNF of ABNF") ;
:startRule(ABNF) ;
:whitespace(Whitespace) ;

ABNF        = { Production | LineCommand } ;
Production  = Name [ Tag ] "=" [ Expression ] ";" ;

Expression  = Alternative ;
Alternative = Sequence { "|" Sequence } ;
Sequence    = Term { Term } ;

Term        = ( Name | Group | Option | Repetition | ByteRange | Range
              | NotCharsOfByte | NotCharOfByte | NotCharsOf | NotCharOf | NotToken
              | CharsOfByte | CharOfByte | CharsOf | CharOf | Times | Command ) [ Tag ] ;

Group       = "(" Expression ")" ;
Option      = "[" Expression "]" ;
Repetition  = "{" Expression "}" ;
ByteRange   = Token "..b" Token ;
Range       = Token [ "..." Token ] ;
CharsOf     = "@+" Token ;
CharOf      = "@" Token ;
CharsOfByte = "@b+" Token ;
CharOfByte  = "@b" Token ;
NotCharsOf  = "!@+" Token ;
NotCharOf   = "!@" Token ;
NotCharsOfByte = "!@b+" Token ;
NotCharOfByte  = "!@b" Token ;
NotToken    = "!" Token ;
Times       = CmdNumber [ "..." ( CmdNumber | "" ) ] Group ;

CmdNumber   = Number | Command ;

LineCommand = Command ";" ;
Command     = ":" CmdName "(" [ ( Name | Token | Number ) { "," ( Name | Token | Number ) } ] ")" ;

Tag         = "<" ( Name | Token ) { "," ( Name | Token ) } ">" ;
```

* The `Tag` is always responsible for the `Term` right before it.

Note: If you want to see an ABNF of an ABNF, this is here: [ABNF of ABNF to a-grammar](#abnf-of-abnf-to-a-grammar).

#### Almost Common syntax

This is the definition of `Name` and `Token`, of `Number`, and of `Whitespace`:

```javascript
Name        = Alphabet :whitespace() { Alphabet | Digit | "_" } :whitespace(Whitespace) ;
CmdName     = Alphabet :whitespace() { Alphabet | Digit | "_" } :whitespace(Whitespace) ;

Token       = Dquotetoken | Squotetoken | Code ;
// The escape pair (backslash plus any printable char) is consumed as a whole and tried
// first, so neither an escaped quote nor a \\ can end the token early.
Dquotetoken = '"' :whitespace() { TokenEsc | AsciiNoQs | "'" } '"' :whitespace(Whitespace) ;
Squotetoken = "'" :whitespace() { TokenEsc | AsciiNoQs | '"' } "'" :whitespace(Whitespace) ;
TokenEsc    = "\\" "\x20"..b"\x7e" ;
Code        = '~~' :whitespace() { [ "~" ] AllButTilde } '~~' :whitespace(Whitespace) ;

Alphabet    = "a"..."z" | "A"..."Z" ;
Digit       = "0"..."9" ;
AsciiNoQs   = "\x28"..."\x7e" | "\x23"..."\x26" | @"\t\n\r !" ; // Readable ASCII without double and single quotes.
NoLinebreak = !@"\n" ;  // Any char except the line feed (using the negated set syntax).
NoStar      = !@"*" ;   // Any char except the star.
NoStarSlash = !@"*/" ;  // Any char except star and slash.
AllButTilde = "\x00"..."}" | "\\~" | "\x7f"..."\U0010ffff" ; // All chars. Only tilde is escaped.

Number      = "0" | "1"..."9" { "0"..."9" } ;

Whitespace  = { @+"\t\n\r " | Comment } ;

Comment     = LineComment | BlockComment ;
// The body consumes stars only when they are not part of the closing */, so comments
// like /* foo **/ close correctly.
BlockComment = "/*" :whitespace() { NoStar | "*" { "*" } NoStarSlash } "*" { "*" } "/" :whitespace(Whitespace) ;
// The line feed is consumed as ordinary whitespace, so a line comment can also end at
// the end of the file.
LineComment = "//" :whitespace() { NoLinebreak } :whitespace(Whitespace) ;
```

As can be seen in the above EBNF, a `token` consists of one backslash escaped string, quoted in single or double quotes.
```
"This is an\nexample\tof a multiline string with one tab"
```
A `tag` starts with `<`, ends with `>`, and normally contains one or multiple comma separated raw strings (`Code`), quoted in `~~` (two on either side). Inside a raw tag string, `\~` is a special symbol for `~` to be able to write a literal `~~` combination. Single `~` can be written without a backslash escape.
```
< ~~This is an
example of a multiline
raw string (inside a tag)
with one tilde: ~
and then two tildes: ~\~~~, ~~This is a second string inside the Tag~~ >
```

### Parser commands

The following parser commands are available:

#### Line plus inline commands

* __:whitespace(whitespace name | token)__  
This defines the allowed whitespace between the following token and numbers.
* __:script(script name | token)__  
This defines a JS, that is executed instead of a parser rule. It can emit parser rules depending on the target text and is therefore a dynamic parser rule.

#### Line commands

* __:include(fileName name | token)__  
This includes another ABNF into the current one.
* __:title(title token)__  
The title of the ABNF.
* __:description(description token)__  
The description of the ABNF.
* __:startRule(rule name)__  
The start rule of the ABNF. This is the top level rule for the parser.
* __:startScript(script name | token {, script name | token})__  
The start script of the ABNF. The compiler runs the start script that must specify what to compile (usually `c.asg`) and what to do with the result.

#### Inline commands

* __:number(size number, type number)__  
`:number(size, type)` This reads `size` bytes from the target text, interprets is as `type` and returns it to the parser, as if it would have been written as `Number` in the ABNF. This allows to parse e.g. TLV formats. `type` can be `0` for little endian, `1` for big endian, `2` for BCD, and `3` for ASCII (see [NumberType Constants](#numbertype-constants)).


### Exposed JS API

The annotations of the ABNF can contain JS code. The ASG (abstract semantic graph) gets processed from the leaves up to the stem. If annotations are encountered on the way, their JS code gets executed.

#### General

* __exit(v int)__  
Terminates the application and returns `v`.
* __sleep(d int)__  
Sleeps for `d` milliseconds.

#### Console output

* __print(...)__ [fmt.Print](https://golang.org/pkg/fmt/#Print)
* __println(...)__ [fmt.Println](https://golang.org/pkg/fmt/#Println)
* __printf(...)__ [fmt.Printf](https://golang.org/pkg/fmt/#Printf)
* __sprintf(...)__ [fmt.Sprintf](https://golang.org/pkg/fmt/#Sprintf)

#### Storage

* __load(fileName) string__  
Loads a file from the disk.
* __store(fileName, data string)__  
Stores a file to the disk.

#### Strings

* __unescape(s string) string__  
Backslash unescapes the content of a `Dquotetoken` or a `Squotetoken` string. It is about the inverse of `printf("%q", s)` but without the quotation marks.
* __unescapeTilde(s string) string__  
Backslash unescapes a tags `Code` string. It basically only unescapes a `\~` into a `~` (tilde) and leaves everything else untouched.

#### Variables

* __append(a []object, v1 object, ...) []object__  
The function appends the objects `v1` ... `vn` to the array `a` and returns the combined array.

##### Local variables

* __up__ (for upstream)  
  All local variables. All `up.*` variables can be changed by the user. This includes `up.in`.
  * __up.in__  
  (string) The collective matched strings of all child nodes.
  * __up.\*__  
  User generated local variables. They can be arbitrary objects. Those objects are concatenated to arrays of objects when being propagated upwards.
  * __up.str\*__  
  User generated local variables. All variables that start with `str` must be strings. Those objects are concatenated as strings when being propagated upwards. up.in is an example of such string concatenation.
  * __up.arr\*__  
  User generated local variables. Those can be arrays of arbitrary objects. They are appended when being propagated upwards. If an object is not an array, it will be put into one.
  * __up.stack__  
  See [Local stacks](#local-stacks)

##### Global variables

* __ltr__ (for left to right)  
  All global variables (global JS variables can be used too). All `ltr.*` variables can be changed by the user. This includes `ltr.in`.
  * __ltr.in__  
  (string) The collective matched strings of all nodes from left to right. Only matched strings of nodes to the right (that are not processed yet), are not included.  
  * __ltr.\*__  
  User generated global variables. They can be arbitrary objects. Except for `ltr.in`, those objects are not changed by the compiler.
  * __ltr.stack__  
  See [Global stack](#global-stack)

#### The stacks

This API provides multiple local (LIFO) stacks and one global (LIFO) stack.

##### Local stacks

Each leaf starts with its own local stack. This stacks are combined hirarchically like the local variables of `up.arr`.

* __pop() object__  
Pops an arbitrary object from the local stack.
* __push(v object)__  
Pushes an arbitrary object onto the local stack.
* __up.stack__  
This stack can also be accessed via the variable `up.stack`.

##### Global stack

This one global stack is useful to e.g. bring data from one `Term` to a sibling. It is like the `ltr.*` variables:

* __popg() object__  
Pops an arbitrary object from the stack.
* __pushg(v object)__  
Pushes an arbitrary object onto the stack.
* __ltr.stack__  
This stack can also be accessed via the variable `ltr.stack`.

#### Parser script API

Inside the code of an inline `:script()` command, the JS runs while the parser is working. There, the object `c` gives access to the parser state instead of the compiler:

* __c.getSrc() string__ / __c.setSrc(src string)__  
Reads / replaces the whole target text.
* __c.getSdx() int__ / __c.setSdx(sdx int)__  
Reads / moves the current parse position.
* __c.peek(offset int) int__  
Returns the byte at the current parse position plus `offset`, or `-1` outside of the target text. Unlike `c.getSrc()` this does not copy the target text, so it is the cheap way to look ahead (see the keyword boundary check `KwEnd` in `languages/c99-parser.abnf` for a negative lookahead built with it).
* __push(v object)__ / __pop() object__  
A stack that survives between the `:script()` calls of one parse run.

If the script returns a rule (built with the `abnf.*` functions), the parser applies that rule at the current position; any other return value means there is nothing to apply.

#### Compiler API

* __c.agrammar__  
The grammar that produced the current state, that the JS is executed in.
* __c.ABNFagrammar__  
A grammar that can parse and compile ABNF. This is the default initial grammar for the tool.
* __c.parse(agrammar []Rule, target string, options Parseropts) []Rule__  
Parses the string `target` with the `agrammar` and returns an ASG. The parameter `options` can be left out.
* __c.asg__  
The whole abstract semantic graph.
* __c.localAsg__  
The local part of the abstract semantic graph.
* __c.compile(asg []Rule, slot int, traceEnabled bool) map[string]object__  
Compiles the given ASG and returns the map of the combined upstream variables.  
Normally, `c.compile()` is called as `c.compile(c.asg)`.<br/>
The parameter `slot` states the index of the code part inside the `Tags`. It is normally 0. The parameter `traceEnabled` can additionally switch on the compiler trace output.<br/>
The compiler works like this:  
```
    OUT
     ^
     |
     C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of Tag Rules.)
     |    |
     ^    v
     *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
    /|    |
   | | _  |
   T | |  |     (T) The text of an EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
   | X |  |     (X) The script of a single Tag Rule gets executed. This is after its childs came back from being splitted at (C).
   | | O  |     (O) Other Rules are ignored.
   | | |  |
   \ | /  |
    \|/   |
     *    |     (*) Childs from one Rule get splitted. The splitted path always only processe one rule (That can contain childs).
     |    |
     ^    |
     IN<-'
```
* __c.compileRunStartScript(asg []Rule, aGrammar []Rule, slot int, traceEnabled bool) map[string]object__  
Instantiates a new compiler with `asg` and `aGrammar` and starts the :startScript() code of the `aGrammar`. This start script code has to do the rest, to compile the ASG in parameter `asg`. Specifically, it has to call `c.compile(c.asg)` to compile that ASG. And it has to handle the result of the compilation if necessary.<br/>
The parameter `slot` states the index of the code part inside the `Tags`. It is normally 0.

#### Parser and Compiler ABNF a-grammar API

The a-grammar can be built from within JS. For this, some simple builder funcions are exposed:

##### Builder functions

* __abnf.arrayToRules(rules []object) []Rule__  
  Wraps a plain JS array of rules into an a-grammar (`Rules`) value.
* __abnf.newRule(Operator OperatorID, String string, Int int, Pos int, Childs []Rule, CodeChilds []Rule) Rule__  
  The generic constructor: builds any rule from its raw fields (the `new*` helpers below are shorthands for it).
* __abnf.newToken(String string, Pos int) Rule__  
  A terminal symbol: fixed text that must appear in the target text.
* __abnf.newTokenEscaped(String string, Pos int) Rule__  
  Like `newToken`, but resolves the backslash escapes of `String` on the Go side, so a byte set token like `'\xff'` keeps its raw bytes instead of being mangled into U+FFFD by the JS engine.
* __abnf.newNumber(Int int, Pos int) Rule__  
  A plain number literal (as produced by `:number()`).
* __abnf.newIdentifier(String string, Pos int) Rule__  
  A reference to a production by name.
* __abnf.newProduction(String string, Childs []Rule, Pos int) Rule__  
  A named production - the definition an `Identifier` points to.
* __abnf.newTag(CodeChilds []Rule, Childs []Rule, Pos int) Rule__  
  An annotation over `Childs`; `CodeChilds` holds the JS code that runs once they matched.
* __abnf.newCommand(String string, CodeChilds []Rule, Pos int) Rule__  
  A `:name(...)` command; `String` is the name, `CodeChilds` the parameters.
* __abnf.newRepetition(Childs []Rule, Pos int) Rule__  
  Zero or more repetitions (the `{ ... }` form).
* __abnf.newOption(Childs []Rule, Pos int) Rule__  
  An optional part (the `[ ... ]` form).
* __abnf.newGroup(Childs []Rule, Pos int) Rule__  
  A group (the `( ... )` form) that must not be broken apart.
* __abnf.newSequence(Childs []Rule, Pos int) Rule__  
  A sequence of rules matched one after another.
* __abnf.newAlternative(Childs []Rule, Pos int) Rule__  
  Alternatives (the `|` form); the first matching child wins.
* __abnf.newRange(CodeChilds []Rule, RangeType int, Pos int) Rule__  
  `CodeChilds` holds the two Token `[from, to]`, `RangeType` is one of the [RangeType Constants](#rangetype-constants).
* __abnf.newTimes(CodeChilds []Rule, Childs []Rule, Pos int) Rule__  
  A counted repetition (e.g. `3...5 ( X )`); `CodeChilds` holds the count bounds.
* __abnf.newCharOf(Token Rule, CharType int, Pos int) Rule__  
  One char of a set. `Token` is the set's Token *rule* (pass the rule, not its string, so raw bytes survive the JS engine), `CharType` is a combination of the [CharType Constants](#chartype-constants).
* __abnf.newCharsOf(Token Rule, CharType int, Pos int) Rule__  
  Like `newCharOf`, but matches a maximal run of set chars.
* __abnf.newNot(Childs []Rule, Pos int) Rule__  
  Negative lookahead: `Childs` holds exactly one rule; the Not matches with zero width when that rule does **not** match at the current position.
* __correctReferencesAndIDs(agrammar []Rule)__ (a global function, not part of `abnf.*`)  
This fills the array position of `Productions` into their `Identifier` (-1 if the production does not exist). It also identifies each different `Tag` with another UID. The array positions of the productions and the UIDs of the Tags are stored in the rules Int field. This method must be used on newly created a-grammars, if they are directly used for compilation. The parser applies this method automatically.

##### Serializer functions

* __abnf.serializeRule(rule Rule) string__  
  Serializes one rule to its Go-literal form (the form hard coded in `abnf/agrammar.go`, re-readable by compiling it as Go).
* __abnf.serializeRules(rules []Rule) string__  
  Serializes a whole a-grammar to its Go-literal form.
* __abnf.toStringRule(rule Rule) string__  
  A short, human readable dump of one rule (child rules abbreviated as `[...]`).
* __abnf.toStringRules(rules []Rule) string__  
  A short, human readable dump of a whole a-grammar.

##### Grammar functions

* __abnf.getStartRule(rules []Rule) Rule__  
Returns the start rule of the a-grammar. The start rule points to the top level production for the parser.
* __abnf.getStartScript(rules []Rule) Rule__  
Returns the :startScript() of the a-grammar. This contains the JS code that controls the compile step.
* __abnf.getTitle(rules []Rule) Rule__  
Returns the a-grammar's `:title()` Token (the title text), or nothing if it has none.
* __abnf.getDescription(rules []Rule) Rule__  
Returns the a-grammar's `:description()` Token, or nothing if it has none.

##### OperatorID Constants

The `Operator` field of a `Rule`; compare against these to inspect a rule (e.g. `rule.Operator == abnf.oid.Sequence`).

* __abnf.oid.Error__ — internal marker for an invalid rule (never appears in a valid a-grammar).
* __abnf.oid.Success__ — internal status marker (never appears in a valid a-grammar).
* __abnf.oid.Sequence__ — a sequence of rules matched in order (may be broken apart).
* __abnf.oid.Group__ — a group `( ... )` that must not be broken apart.
* __abnf.oid.Token__ — a terminal symbol (fixed text in the target).
* __abnf.oid.Number__ — a plain number (e.g. from `:number()`).
* __abnf.oid.Or__ — alternatives `|`; the first matching child wins.
* __abnf.oid.Optional__ — an optional part `[ ... ]`.
* __abnf.oid.Repeat__ — zero or more repetitions `{ ... }`.
* __abnf.oid.Range__ — a char range; `Int` holds the [RangeType](#rangetype-constants).
* __abnf.oid.Times__ — a counted repetition (e.g. `3...5 ( X )`).
* __abnf.oid.Tag__ — an annotation carrying JS code.
* __abnf.oid.Command__ — a `:name(...)` command.
* __abnf.oid.CharOf__ — exactly one char of a set; `Int` holds [CharType](#chartype-constants) flags.
* __abnf.oid.CharsOf__ — a run of chars of a set; `Int` holds [CharType](#chartype-constants) flags.
* __abnf.oid.Not__ — negative lookahead (matches with zero width when the child does not).
* __abnf.oid.Production__ — a named production (the definition an `Identifier` points to).
* __abnf.oid.Identifier__ — a reference to a production by name.

##### RangeType Constants

The `Int` field of a `Range` rule (see `newRange`).

* __abnf.rangeType.Rune__ — the bounds are single runes (the `"a"..."z"` form).
* __abnf.rangeType.Byte__ — the bounds are single bytes (the `"\x00"..b"\xff"` form).

##### CharType Constants

Flags for the `CharType` argument of `newCharOf` / `newCharsOf` (combine with `|`):

* __abnf.charType.Rune__ — match one UTF8 char of the set (the default, the `@` / `@+` forms).
* __abnf.charType.Byte__ — match a single byte instead (the `@b` / `@b+` forms).
* __abnf.charType.Negated__ — match the chars _not_ in the set (the `!@` forms).

##### NumberType Constants

The `type` parameter of `:number(size, type)` - how the `size` bytes read from the target text are decoded:

* __abnf.numberType.LittleEndian__ — an unsigned little-endian integer (the default).
* __abnf.numberType.BigEndian__ — an unsigned big-endian integer.
* __abnf.numberType.BCD__ — binary-coded decimal (an optional trailing `f` nibble is dropped).
* __abnf.numberType.ASCII__ — ASCII decimal digits.

#### LLVM IR API

This tool uses the [Go LLIR/LLVM library](https://github.com/llir/llvm) to create LLVM IR and to interact with it. The API documentation can be found here: [llir/llvm overview](https://llir.github.io/document/) and here [LLIR/LLVM library documentation](https://pkg.go.dev/github.com/llir/llvm/). For more information on LLVM IR go to the [LLVM language reference](https://llvm.org/docs/LangRef.html).

The functions and constants are exposed to JS as:

* __llvm.ir.\*__ [llvm.ir](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir)
* __llvm.constant.\*__ [llvm.constant](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant)
* __llvm.metadata.\*__ [llvm.metadata](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata)
* __llvm.types.\*__ [llvm.types](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types)
* __llvm.enum.\*__ [llvm.enum](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/enum)

* Custom functions:
  * __llvm.Callgraph(m ir.Module) string__  
  The function `llvm.Callgraph(m ir.Module) string` creates the callgraph of the given LLVM IR module in Graphviz DOT format (can be viewed e.g. online with the [Graphviz visual editor](http://magjac.com/graphviz-visual-editor/)).
  * __llvm.Eval(m ir.Module, f string) uint32__  
  The function `llvm.Eval(m ir.Module, f string)` executes the function `f` inside the IR module `m` with the built-in IR interpreter and returns the resulting uint32.
  * __llvm.Run(m ir.Module, f string, input string) {Ret uint32, Out string}__  
  Like `llvm.Eval()`, but `input` is what `getchar()` reads (can be left out), `Out` is everything the program wrote via `putchar()`/`puts()`, and `Ret` is the return value. The interpreter supports the integer subset of LLVM IR that the compiler grammars generate: alloca/load/store, getelementptr into arrays and structs (packed layout: the fields lie back to back, ints take 4 bytes and pointers 8), integer arithmetic and comparisons, zext/sext/trunc, ptrtoint/inttoptr/bitcast, select, phi, branches, calls, and the externals putchar, getchar, puts and abs.
  * __llvm.RunJS(m ir.Module, f string) {Ret uint32, Out string}__  
  Executes a MetaJS module (IR emitted by `languages/metajs-to-llvm-ir.abnf`, where every value is an i64 handle and the `js_*` externals implement the JS semantics on the Go side). `f` is the module entry, normally `jsmain`; its handle result is converted to an int32 and returned as `Ret`.

## Further Examples

### ABNF of ABNF to a-grammar

<details>
  <summary>Click to expand!</summary>

```javascript
:title("ABNF of ABNF to a-grammar") ;
:description("The annotated EBNF format, described in itself: parsing this file with the
built-in a-grammar and compiling the result yields exactly that a-grammar again (its
serialized twin is hard coded in abnf/agrammar.go and bootstraps every run).

This file is the reference for the whole syntax: productions, alternatives |, sequences,
groups ( ), options [ ], repetitions { }, counted repetitions like 3...5 ( X ), rune and
byte ranges (... and ..b), char sets (@'ab' is one char of the set, @+'ab' a run of them,
@b and @b+ the byte versions, and !@ !@+ !@b !@b+ match the chars NOT in the set),
!'token' as negative lookahead (matches without consuming when the token does not),
tokens with escapes (\\n \\t \\x41 \\u00e4 and the token's own quote), commands like
:whitespace(), and tags carrying JS code.

After changing this file, regenerate abnf/agrammar.go as described in the README.") ;


// --- main rules

:startRule(ABNF) ;
// This is a parser command that sets the possible white space.
:whitespace(Whitespace) ;

// This is the start rule.
ABNF        = { Production | LineCommand } ;

Production  = Name <~~ var prodTag=undefined; var prodExpression=undefined; pushg(pop()) ~~> [ Tag <~~ prodTag=pop() ~~> ] "=" [ Expression <~~ prodExpression=pop() ~~> ] ";" <~~  pushg(buildProduction(popg(), prodTag, prodExpression)) ~~> ;

Expression  = Alternative ;

Alternative <~~ push(simplify(abnf.newAlternative(popg(), up.pos))) ~~>
            = Sequence <~~ pushg([pop()]) ~~> { "|" Sequence <~~ pushg(append(popg(), pop())) ~~> } ;

Sequence    <~~ push(simplify(abnf.newSequence(popg(), up.pos))) ~~>
            = Term <~~ pushg([pop()]) ~~> { Term <~~ pushg(append(popg(), pop())) ~~> } ;

// The alternatives are ordered so that the longer prefixes are tried first
// ("!@b+" before "!@b" before "!@+" before "!@" before "!", and "@b+" before "@b"
// before "@+" before "@"), because the parser keeps the first match.
Term        <~~ push(popg()) ~~>
            = ( Name | ByteRange | Range | NotCharsOfByte | NotCharOfByte | NotCharsOf | NotCharOf | NotToken | CharsOfByte | CharOfByte | CharsOf | CharOf | Group | Option | Repetition | Times | Command ) <~~ pushg(simplify(pop())) ~~> [ Tag <~~ var tag=pop(); tag.Childs=simplifyToArr(popg()); pushg(tag) ~~> ] ;

Group       = "(" Expression <~~ push(abnf.newGroup(simplifyToArr(pop()), up.pos)) ~~> ")" ;
Option      = "[" Expression <~~ push(abnf.newOption(simplifyToArr(pop()), up.pos)) ~~> "]" ;
Repetition  = "{" Expression <~~ push(abnf.newRepetition(simplifyToArr(pop()), up.pos)) ~~> "}" ;
Range       <~~ push(popg()) ~~>
            = Token <~~ pushg(pop()) ~~> [ "..." Token <~~ pushg(abnf.newRange([popg(), pop()], abnf.rangeType.Rune, up.pos)) ~~> ] ;
ByteRange   = Token <~~ pushg(pop()) ~~> "..b" Token <~~ push(abnf.newRange([popg(), pop()], abnf.rangeType.Byte, up.pos)) ~~> ;

// The char set family: one char of the set (@) or a maximal run of them (@+), matching
// runes (default) or single bytes (b), of the set itself or of its complement (!).
CharsOf        = "@+" Token <~~ push(abnf.newCharsOf(pop(), abnf.charType.Rune, up.pos)) ~~> ;
CharOf         = "@" Token <~~ push(abnf.newCharOf(pop(), abnf.charType.Rune, up.pos)) ~~> ;
CharsOfByte    = "@b+" Token <~~ push(abnf.newCharsOf(pop(), abnf.charType.Byte, up.pos)) ~~> ;
CharOfByte     = "@b" Token <~~ push(abnf.newCharOf(pop(), abnf.charType.Byte, up.pos)) ~~> ;
NotCharsOf     = "!@+" Token <~~ push(abnf.newCharsOf(pop(), abnf.charType.Negated, up.pos)) ~~> ;
NotCharOf      = "!@" Token <~~ push(abnf.newCharOf(pop(), abnf.charType.Negated, up.pos)) ~~> ;
NotCharsOfByte = "!@b+" Token <~~ push(abnf.newCharsOf(pop(), abnf.charType.Byte | abnf.charType.Negated, up.pos)) ~~> ;
NotCharOfByte  = "!@b" Token <~~ push(abnf.newCharOf(pop(), abnf.charType.Byte | abnf.charType.Negated, up.pos)) ~~> ;

// Negative lookahead: !'x' matches (consuming nothing) when the token does not match here.
NotToken    = "!" Token <~~ push(abnf.newNot([pop()], up.pos)) ~~> ;

Times       = CmdNumber <~~ pushg([pop()]) ~~> [ "..." ( CmdNumber | "" <~~ push(abnf.newToken("...")) ~~> ) <~~ pushg(append(popg(), pop())) ~~> ] Group <~~ push(abnf.newTimes(popg(), simplifyToArr(pop()), up.pos)) ~~> ;

CmdNumber   = Number | Command ;

LineCommand = Command <~~ pushg(pop()) ~~> ";" ;
Command     <~~ push(abnf.newCommand(pop(), popg(), up.pos)) ~~>
            = ":" CmdName "(" <~~ pushg([]) ~~> [ ( Name | Token | Number ) <~~ pushg(append(popg(), pop())) ~~> { "," ( Name | Token | Number ) <~~ pushg(append(popg(), pop())) ~~> } ] ")" ;

Tag         <~~ push(abnf.newTag(popg(), undefined, up.pos)) ~~>
            = "<" ( Name | Token ) <~~ pushg([pop()]) ~~> { "," ( Name | Token ) <~~ pushg(append(popg(), pop())) ~~> } ">" ;

Name        <~~ push(abnf.newIdentifier(up.in, up.pos)) ~~>
            = Alphabet :whitespace() { Alphabet | Digit | "_" } :whitespace(Whitespace) ;
CmdName     <~~ push(up.in) ~~>
            = Alphabet :whitespace() { Alphabet | Digit | "_" } :whitespace(Whitespace) ;

Token       = Dquotetoken | Squotetoken | Code ;
// The escape pair (backslash plus any printable char) is consumed as a whole and tried
// first, so neither an escaped quote ends the token early nor pairs the second
// backslash of a \\ with the closing quote.
Dquotetoken = '"' :whitespace() { TokenEsc | AsciiNoQs | "'" } <~~ push(abnf.newTokenEscaped(up.in, up.pos)) ~~> '"' :whitespace(Whitespace) ;
Squotetoken = "'" :whitespace() { TokenEsc | AsciiNoQs | '"' } <~~ push(abnf.newTokenEscaped(up.in, up.pos)) ~~> "'" :whitespace(Whitespace) ;
// The backslash is written as \x5c here: this file must stay parseable by the built-in
// a-grammar of OLDER builds during regeneration, and some of those mis-parsed a "\\"
// token directly in front of another quote.
TokenEsc    = "\x5c" "\x20"..b"\x7e" ;
Code        = '~~' :whitespace() { [ "~" ] AllButTilde } <~~ push(abnf.newToken(unescapeTilde(up.in), up.pos)) ~~> '~~' :whitespace(Whitespace) ;

Alphabet    = "a"..."z" | "A"..."Z" ;
Digit       = "0"..."9" ;
AsciiNoQs   = "\x28"..."\x7e" | "\x23"..."\x26" | @"\t\n\r !" ; // Readable ASCII without double and single quotes.
NoLinebreak = !@"\n" ;  // Any char except the line feed (the file uses its own !@ syntax).
NoStar      = !@"*" ;   // Any char except the star.
NoStarSlash = !@"*/" ;  // Any char except star and slash.
AllButTilde = "\x00"..."}" | "\\~" | "\x7f"..."\U0010ffff" ; // All chars. Only tilde is escaped.

Number      <~~ push(abnf.newNumber(up.in, up.pos)) ~~>
            = "0" | "1"..."9" { "0"..."9" } ;

Whitespace  = { @+"\t\n\r " | Comment } ;

Comment     = LineComment | BlockComment ;
// The body consumes stars only when they are not part of the closing */. A failing body
// iteration is rolled back, so comments like /* foo **/ close correctly.
BlockComment = "/*" :whitespace() { NoStar | "*" { "*" } NoStarSlash } "*" { "*" } "/" :whitespace(Whitespace) ;
// The line feed is not part of the comment; it is consumed as ordinary whitespace, so a
// line comment can also end at the end of the file.
LineComment = "//" :whitespace() { NoLinebreak } :whitespace(Whitespace) ;

// ---


:startScript(~~

    function buildProduction(prodName, prodTag, prodExpression) {
        if (prodTag != undefined) {
            prodTag.Childs = simplifyToArr(prodExpression)
            return abnf.newProduction(prodName.String, [prodTag], prodName.Pos)
        } else {
            return abnf.newProduction(prodName.String, simplifyToArr(prodExpression), prodName.Pos)
        }
    }

    // This breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.
    function simplifyArr(rules) {
        if (rules.length == 1) {
            const op = rules[0].Operator
            if (op == abnf.oid.Sequence || op == abnf.oid.Group || (op == abnf.oid.Or && rules[0].Childs.length <= 1)) return simplifyArr(rules[0].Childs)
        }
        return rules
    }

    // This also breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.
    function simplifyToArr(rule) {
        if (rule == undefined) return undefined
        return simplifyArr([rule])
    }

    // Groups with only one child can be broken apart as long as down there is an unbreakable rule. Try to find one.
    function trySimplifyDown(rule) {
        if (rule.Childs == undefined) return rule
        const op = rule.Operator
        if ((rule.Childs.length == 1) && (op == abnf.oid.Sequence || op == abnf.oid.Group || op == abnf.oid.Or)) return trySimplifyDown(rule.Childs[0])
        if (op == abnf.oid.Sequence) return undefined
        return rule
    }

    function simplify(rule) {
        let ruleDown = trySimplifyDown(rule)
        if (ruleDown != undefined) return ruleDown
        if (rule.Childs.length == 1) { // Breaking up abnf.oid.Group did not work. Getting down only with Sequence and Or.
            const op = rule.Operator
            if (op == abnf.oid.Sequence || op == abnf.oid.Or) return simplify(rule.Childs[0])
        }
        return rule
    }

    c.compile(c.asg)
    let rules = ltr.stack

    // To show the initial a-grammar:
    println("=> Rules: " + abnf.serializeRules(rules))

    // To return the generated a-grammar to the next parser:
    rules

~~) ;
```
</details>

### Its output, when it gets applied on itself:

This output is exactly the data structure that controls the parser when it is used as an ABNF parser. It is what makes the parser to an ABNF parser. It is the pretty-printed serialized a-grammar (the same structure as `abnf/agrammar.go`); regenerate this dump with `./mec languages/abnf-of-abnf.abnf -pretty`:

<details>
  <summary>Click to expand!</summary>

```javascript
&r.Rules{&r.Rule{Operator:r.Command, String:"title", CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"ABNF of ABNF to a-grammar"
            }
        }
    }, &r.Rule{Operator:r.Command, String:"description", CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"The annotated EBNF format, described in itself: parsing this file with the\nbuilt-in a-grammar and compiling the result yields exactly that a-grammar again (its\nserialized twin is hard coded in abnf/agrammar.go and bootstraps every run).\n\nThis file is the reference for the whole syntax: productions, alternatives |, sequences,\ngroups ( ), options [ ], repetitions { }, counted repetitions like 3...5 ( X ), rune and\nbyte ranges (... and ..b), char sets (@'ab' is one char of the set, @+'ab' a run of them,\n@b and @b+ the byte versions, and !@ !@+ !@b !@b+ match the chars NOT in the set),\n!'token' as negative lookahead (matches without consuming when the token does not),\ntokens with escapes (\\n \\t \\x41 \\u00e4 and the token's own quote), commands like\n:whitespace(), and tags carrying JS code.\n\nAfter changing this file, regenerate abnf/agrammar.go as described in the README."
            }
        }
    }, &r.Rule{Operator:r.Command, String:"startRule", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"ABNF"
            }
        }
    }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"ABNF", Childs:&r.Rules{&r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Production"
                            }, &r.Rule{Operator:r.Identifier, String:"LineCommand"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Production", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" var prodTag=undefined; var prodExpression=undefined; pushg(pop()) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                    }
                }
            }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" prodTag=pop() "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Tag"
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"="
            }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" prodExpression=pop() "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Expression"
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"  pushg(buildProduction(popg(), prodTag, prodExpression)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:";"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Expression", Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Alternative"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Alternative", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(simplify(abnf.newAlternative(popg(), up.pos))) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Sequence"
                            }
                        }
                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"|"
                            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Sequence"
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Sequence", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(simplify(abnf.newSequence(popg(), up.pos))) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Term"
                            }
                        }
                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Term"
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Term", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(popg()) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(simplify(pop())) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                                    }, &r.Rule{Operator:r.Identifier, String:"ByteRange"
                                    }, &r.Rule{Operator:r.Identifier, String:"Range"
                                    }, &r.Rule{Operator:r.Identifier, String:"NotCharsOfByte"
                                    }, &r.Rule{Operator:r.Identifier, String:"NotCharOfByte"
                                    }, &r.Rule{Operator:r.Identifier, String:"NotCharsOf"
                                    }, &r.Rule{Operator:r.Identifier, String:"NotCharOf"
                                    }, &r.Rule{Operator:r.Identifier, String:"NotToken"
                                    }, &r.Rule{Operator:r.Identifier, String:"CharsOfByte"
                                    }, &r.Rule{Operator:r.Identifier, String:"CharOfByte"
                                    }, &r.Rule{Operator:r.Identifier, String:"CharsOf"
                                    }, &r.Rule{Operator:r.Identifier, String:"CharOf"
                                    }, &r.Rule{Operator:r.Identifier, String:"Group"
                                    }, &r.Rule{Operator:r.Identifier, String:"Option"
                                    }, &r.Rule{Operator:r.Identifier, String:"Repetition"
                                    }, &r.Rule{Operator:r.Identifier, String:"Times"
                                    }, &r.Rule{Operator:r.Identifier, String:"Command"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" var tag=pop(); tag.Childs=simplifyToArr(popg()); pushg(tag) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Tag"
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Group", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"("
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newGroup(simplifyToArr(pop()), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Expression"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:")"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Option", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"["
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newOption(simplifyToArr(pop()), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Expression"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"]"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Repetition", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"{"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newRepetition(simplifyToArr(pop()), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Expression"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"}"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Range", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(popg()) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(pop()) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                            }
                        }
                    }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"..."
                            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(abnf.newRange([popg(), pop()], abnf.rangeType.Rune, up.pos)) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"ByteRange", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(pop()) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"..b"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newRange([popg(), pop()], abnf.rangeType.Byte, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CharsOf", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"@+"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharsOf(pop(), abnf.charType.Rune, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CharOf", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"@"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharOf(pop(), abnf.charType.Rune, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CharsOfByte", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"@b+"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharsOf(pop(), abnf.charType.Byte, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CharOfByte", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"@b"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharOf(pop(), abnf.charType.Byte, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NotCharsOf", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"!@+"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharsOf(pop(), abnf.charType.Negated, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NotCharOf", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"!@"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharOf(pop(), abnf.charType.Negated, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NotCharsOfByte", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"!@b+"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharsOf(pop(), abnf.charType.Byte | abnf.charType.Negated, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NotCharOfByte", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"!@b"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCharOf(pop(), abnf.charType.Byte | abnf.charType.Negated, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NotToken", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"!"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newNot([pop()], up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Token"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Times", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg([pop()]) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"CmdNumber"
                    }
                }
            }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"..."
                    }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"CmdNumber"
                                    }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newToken(\"...\")) "
                                            }
                                        }, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:""
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newTimes(popg(), simplifyToArr(pop()), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Group"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CmdNumber", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Number"
                    }, &r.Rule{Operator:r.Identifier, String:"Command"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"LineCommand", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(pop()) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Command"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:";"
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Command", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newCommand(pop(), popg(), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:":"
                    }, &r.Rule{Operator:r.Identifier, String:"CmdName"
                    }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg([]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"("
                            }
                        }
                    }, &r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                                            }, &r.Rule{Operator:r.Identifier, String:"Token"
                                            }, &r.Rule{Operator:r.Identifier, String:"Number"
                                            }
                                        }
                                    }
                                }
                            }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:","
                                    }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                                            }
                                        }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                                                    }, &r.Rule{Operator:r.Identifier, String:"Token"
                                                    }, &r.Rule{Operator:r.Identifier, String:"Number"
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Token, String:")"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Tag", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newTag(popg(), undefined, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"<"
                    }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg([pop()]) "
                            }
                        }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                                    }, &r.Rule{Operator:r.Identifier, String:"Token"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:","
                            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" pushg(append(popg(), pop())) "
                                    }
                                }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Name"
                                            }, &r.Rule{Operator:r.Identifier, String:"Token"
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Token, String:">"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Name", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newIdentifier(up.in, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Alphabet"
                    }, &r.Rule{Operator:r.Command, String:"whitespace"
                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Alphabet"
                                    }, &r.Rule{Operator:r.Identifier, String:"Digit"
                                    }, &r.Rule{Operator:r.Token, String:"_"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"CmdName", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(up.in) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Alphabet"
                    }, &r.Rule{Operator:r.Command, String:"whitespace"
                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Alphabet"
                                    }, &r.Rule{Operator:r.Identifier, String:"Digit"
                                    }, &r.Rule{Operator:r.Token, String:"_"
                                    }
                                }
                            }
                        }
                    }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Token", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Dquotetoken"
                    }, &r.Rule{Operator:r.Identifier, String:"Squotetoken"
                    }, &r.Rule{Operator:r.Identifier, String:"Code"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Dquotetoken", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"\""
            }, &r.Rule{Operator:r.Command, String:"whitespace"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newTokenEscaped(up.in, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"TokenEsc"
                                    }, &r.Rule{Operator:r.Identifier, String:"AsciiNoQs"
                                    }, &r.Rule{Operator:r.Token, String:"'"
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"\""
            }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Squotetoken", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"'"
            }, &r.Rule{Operator:r.Command, String:"whitespace"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newTokenEscaped(up.in, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"TokenEsc"
                                    }, &r.Rule{Operator:r.Identifier, String:"AsciiNoQs"
                                    }, &r.Rule{Operator:r.Token, String:"\""
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"'"
            }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"TokenEsc", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"\\"
            }, &r.Rule{Operator:r.Range, Int:1, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" "
                    }, &r.Rule{Operator:r.Token, String:"~"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Code", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"~~"
            }, &r.Rule{Operator:r.Command, String:"whitespace"
            }, &r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newToken(unescapeTilde(up.in), up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Optional, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"~"
                                    }
                                }
                            }, &r.Rule{Operator:r.Identifier, String:"AllButTilde"
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"~~"
            }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Alphabet", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"a"
                            }, &r.Rule{Operator:r.Token, String:"z"
                            }
                        }
                    }, &r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"A"
                            }, &r.Rule{Operator:r.Token, String:"Z"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Digit", Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"0"
                    }, &r.Rule{Operator:r.Token, String:"9"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"AsciiNoQs", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"("
                            }, &r.Rule{Operator:r.Token, String:"~"
                            }
                        }
                    }, &r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"#"
                            }, &r.Rule{Operator:r.Token, String:"&"
                            }
                        }
                    }, &r.Rule{Operator:r.CharOf, String:"\t\n\r !"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NoLinebreak", Childs:&r.Rules{&r.Rule{Operator:r.CharOf, String:"\n", Int:2
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NoStar", Childs:&r.Rules{&r.Rule{Operator:r.CharOf, String:"*", Int:2
            }
        }
    }, &r.Rule{Operator:r.Production, String:"NoStarSlash", Childs:&r.Rules{&r.Rule{Operator:r.CharOf, String:"*/", Int:2
            }
        }
    }, &r.Rule{Operator:r.Production, String:"AllButTilde", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"\x00"
                            }, &r.Rule{Operator:r.Token, String:"}"
                            }
                        }
                    }, &r.Rule{Operator:r.Token, String:"\\~"
                    }, &r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"\x7f"
                            }, &r.Rule{Operator:r.Token, String:"\U0010ffff"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Number", Childs:&r.Rules{&r.Rule{Operator:r.Tag, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:" push(abnf.newNumber(up.in, up.pos)) "
                    }
                }, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"0"
                            }, &r.Rule{Operator:r.Sequence, Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"1"
                                            }, &r.Rule{Operator:r.Token, String:"9"
                                            }
                                        }
                                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Range, Int:0, CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"0"
                                                    }, &r.Rule{Operator:r.Token, String:"9"
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Whitespace", Childs:&r.Rules{&r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.CharsOf, String:"\t\n\r "
                            }, &r.Rule{Operator:r.Identifier, String:"Comment"
                            }
                        }
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"Comment", Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"LineComment"
                    }, &r.Rule{Operator:r.Identifier, String:"BlockComment"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"BlockComment", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"/*"
            }, &r.Rule{Operator:r.Command, String:"whitespace"
            }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Or, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"NoStar"
                            }, &r.Rule{Operator:r.Sequence, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"*"
                                    }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"*"
                                            }
                                        }
                                    }, &r.Rule{Operator:r.Identifier, String:"NoStarSlash"
                                    }
                                }
                            }
                        }
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"*"
            }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"*"
                    }
                }
            }, &r.Rule{Operator:r.Token, String:"/"
            }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Production, String:"LineComment", Childs:&r.Rules{&r.Rule{Operator:r.Token, String:"//"
            }, &r.Rule{Operator:r.Command, String:"whitespace"
            }, &r.Rule{Operator:r.Repeat, Childs:&r.Rules{&r.Rule{Operator:r.Identifier, String:"NoLinebreak"
                    }
                }
            }, &r.Rule{Operator:r.Command, String:"whitespace", CodeChilds:&r.Rules{&r.Rule{Operator:r.Identifier, String:"Whitespace"
                    }
                }
            }
        }
    }, &r.Rule{Operator:r.Command, String:"startScript", CodeChilds:&r.Rules{&r.Rule{Operator:r.Token, String:"\n\n    function buildProduction(prodName, prodTag, prodExpression) {\n        if (prodTag != undefined) {\n            prodTag.Childs = simplifyToArr(prodExpression)\n            return abnf.newProduction(prodName.String, [prodTag], prodName.Pos)\n        } else {\n            return abnf.newProduction(prodName.String, simplifyToArr(prodExpression), prodName.Pos)\n        }\n    }\n\n    // This breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.\n    function simplifyArr(rules) {\n        if (rules.length == 1) {\n            const op = rules[0].Operator\n            if (op == abnf.oid.Sequence || op == abnf.oid.Group || (op == abnf.oid.Or && rules[0].Childs.length <= 1)) return simplifyArr(rules[0].Childs)\n        }\n        return rules\n    }\n\n    // This also breaks up an abnf.oid.Group. Use only for childs of unbreakable rules.\n    function simplifyToArr(rule) {\n        if (rule == undefined) return undefined\n        return simplifyArr([rule])\n    }\n\n    // Groups with only one child can be broken apart as long as down there is an unbreakable rule. Try to find one.\n    function trySimplifyDown(rule) {\n        if (rule.Childs == undefined) return rule\n        const op = rule.Operator\n        if ((rule.Childs.length == 1) && (op == abnf.oid.Sequence || op == abnf.oid.Group || op == abnf.oid.Or)) return trySimplifyDown(rule.Childs[0])\n        if (op == abnf.oid.Sequence) return undefined\n        return rule\n    }\n\n    function simplify(rule) {\n        let ruleDown = trySimplifyDown(rule)\n        if (ruleDown != undefined) return ruleDown\n        if (rule.Childs.length == 1) { // Breaking up abnf.oid.Group did not work. Getting down only with Sequence and Or.\n            const op = rule.Operator\n            if (op == abnf.oid.Sequence || op == abnf.oid.Or) return simplify(rule.Childs[0])\n        }\n        return rule\n    }\n\n    c.compile(c.asg)\n    let rules = ltr.stack\n\n    // To show the initial a-grammar:\n    println(\"=> Rules: \" + abnf.serializeRules(rules))\n\n    // To return the generated a-grammar to the next parser:\n    rules\n\n"
            }
        }
    }
}
```
</details>

<br/>

More examples can be found inside the [tests directory](../main/tests).

## Links

Grammars for many languages: [Grammars written for ANTLR v4](https://github.com/antlr/grammars-v4)

## License

Copyright 2021-2025 David Matscheko

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
