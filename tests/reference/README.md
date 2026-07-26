# tests/reference — third-party language reference corpora

Reference test suites for the real-world languages, pulled from each language's
official (or de-facto standard) test suite. The goal is a corpus that exercises
*all* language features and edge cases so we can find and finish the missing
features of every grammar — written by the language implementers themselves, not
by us (writing them ourselves could never reach that coverage; that is an
explicit non-goal).

## How it works

The payload is **not committed**. [fetch.sh](fetch.sh) materializes it; the
committed record is fetch.sh + [pins.lock](pins.lock) (upstream commit shas,
written on first fetch) + this README. Every fetched corpus directory contains
the upstream `LICENSE`/`COPYING` file(s) plus a generated `PROVENANCE.md`
(upstream URL, pinned commit, subset, fetch date).

```
./tests/reference/fetch.sh              # fetch everything (~0.5-1 GB, one-time)
./tests/reference/fetch.sh kotlin php   # fetch only some corpora
./tests/reference/fetch.sh --list       # show the corpus table
```

Fetches are shallow partial git clones (depth 1, blob-filtered, sparse), so only
the wanted subtrees are downloaded even from huge repos. Because the payload is
gitignored, `rg`/`git grep` skip it by default — use `rg --no-ignore` to search
inside the corpora. To upgrade one corpus to current upstream: delete its line
in pins.lock and its directory, re-run fetch.sh.

## The corpora

| Language   | Directory                    | Upstream (subset)                                              | License |
|------------|------------------------------|----------------------------------------------------------------|---------|
| JS         | `js/test262-parser-tests`    | tc39/test262-parser-tests (`pass/`, `fail/`, `early/`)         | per-file MIT/BSD/Apache (see its `licenses.md`) |
| TypeScript | `typescript/typescript-conformance` | microsoft/TypeScript (`tests/cases/conformance`)        | Apache-2.0 |
| Python     | `python/cpython-tests`       | python/cpython (~30 curated `Lib/test/test_*.py` grammar/syntax files) | PSF-2.0 |
| Kotlin     | `kotlin/kotlin-psi-testdata` | JetBrains/kotlin (`compiler/psi/psi-impl/testData` — parser + lexer tests with expected trees) | Apache-2.0 |
| Java       | `java/openjdk-javac-tests`   | openjdk/jdk (`test/langtools/tools/javac`)                     | GPL-2.0+CE — local only |
| Ruby       | `ruby/ruby-spec-language`    | ruby/spec (`language/`)                                        | MIT |
| PHP        | `php/php-src-tests`          | php/php-src (`tests/lang`, `Zend/tests` — `.phpt` format)      | PHP-3.01 |
| C#         | `csharp/mono-mcs-tests`      | mono/mono (`mcs/tests` — standalone compiler tests)            | MIT |
| Swift      | `swift/swift-parse-tests`    | swiftlang/swift (`test/Parse`)                                 | Apache-2.0 + Swift exc. |
| Dart       | `dart/dart-language-tests`   | dart-lang/sdk (`tests/language` — the language feature suite)  | BSD-3-Clause |
| Go         | `go/go-test-suite`           | golang/go (`test/` — the compiler torture suite)               | BSD-3-Clause |
| C          | `c/c-testsuite`              | c-testsuite/c-testsuite (`tests/single-exec` — exec tests + expected output) | mixed (ISC + LGPL cases) — local only |
| Lua        | `lua/lua-tests`              | lua.org official `lua-5.4.6-tests` tarball                     | MIT |
| Bash       | `bash/oils-spec`             | oils-for-unix/oils (`spec/` — bash-compatibility spec tests)   | Apache-2.0 |
| Bash       | `bash/gnu-bash-tests`        | GNU bash 5.3 release tarball (`tests/`, from ftp.gnu.org)      | GPL-3.0+ — local only |
| Batch      | `batch/wine-cmd-tests`       | wine (`programs/cmd/tests` — the only thorough cmd.exe suite in existence) | LGPL-2.1+ — local only |

Opt-in extras (commented out in fetch.sh): full tc39/test262 `test/language`
(execution semantics, ~40k files) and GCC's `gcc.c-torture` (GPL).

### Languages with no external corpus — on purpose

`metajs`, `calculator`, `tinyc`, `lisp` (a deliberately tiny Scheme-like integer
subset) and `brainfuck` are our own/toy languages; their spec is their grammar
file, so external suites would test features the languages intentionally do not
have. Their `tests/*-test-*` files remain the reference.

## Licensing

This repo is Apache-2.0. Everything in the table's permissive rows
(MIT/BSD/Apache/PSF/PHP-3.01) may be **vendored (committed) later** if we ever
want the payload in git — keep the per-directory LICENSE + PROVENANCE.md and
this README's attribution when doing so. PHP-3.01 additionally forbids using
"PHP" in a derived product's name and asks for an acknowledgment notice; both
are satisfied by keeping the license file and this table.

The **local-only** rows (OpenJDK: GPL-2.0-with-Classpath-exception, GNU bash:
GPL-3.0, Wine: LGPL-2.1, c-testsuite: cases imported from scc (ISC) and tinycc
(LGPL) as recorded in its per-case `.otags` files) are deliberately never
committed. Running our parser/interpreter over them locally or in CI creates no
obligations for our code — copyleft triggers on redistribution, and we do not
redistribute them. They are fetched anyway because they are the best (for
batch: the only) comprehensive suites for their languages.

For Java no good permissive corpus exists: ANTLR grammars-v4's Java examples
were considered and rejected (files scraped from Wikipedia/Oracle docs with
unclear licensing) — the OpenJDK javac suite is the reference, and since we
never commit it, its GPL+CE license costs us nothing.

## Format notes per corpus (how to consume)

- **test262-parser-tests**: every file in `pass/` must parse; every file in
  `fail/` must *not* parse; `early/` must parse but produce an early error.
  Tiny standalone files, no harness needed — ideal for a parse ratchet.
- **TypeScript conformance**: mostly standalone `.ts`; multi-file cases use
  `// @filename:` pragmas; compile-option pragmas (`// @strict: true`) are
  comments and can be ignored for parsing.
- **CPython tests**: pure parse targets (execution would need `unittest`).
  `test_grammar.py` alone deliberately exercises the whole grammar.
- **Kotlin PSI testData**: each `foo.kt` has a `foo.txt` with the expected
  JetBrains parse tree — great for diagnosing *how* something should parse.
  Subtrees named `_ERR`/`Recovery` contain intentionally-broken code.
- **ruby/spec**: `language/**/*_spec.rb` parse standalone; executing them needs
  the mspec harness, but parse coverage is the point here.
- **php-src `.phpt`**: sections `--TEST--`/`--FILE--`/`--EXPECT*--`; the PHP
  program is the `--FILE--` section, expected stdout is `--EXPECT--` — needs a
  trivial extractor before feeding to the interpreter.
- **mono mcs tests**: `test-*.cs`/`gtest-*.cs` are standalone known-good
  compiler tests (covers up to ~C# 7; modern C# needs another source later).
- **swift test/Parse**: lit-style `// RUN:` lines are comments; files with
  `// expected-error` markers are *intentionally invalid* at those lines.
- **dart tests/language**: one directory per feature; files containing
  `// [analyzer]`/`// [cfe]` error expectations are intentionally invalid.
- **go test/**: first comment line is the directive — `// run`, `// compile`
  (must compile), `// errorcheck` (must fail, `// ERROR "..."` marks where).
- **c-testsuite**: `tests/single-exec/*.c` with `*.c.expected` stdout files;
  self-contained, exit code 0 = pass. Very close to our own test convention.
- **oils spec**: `spec/*.test.sh`, cases delimited by `#### name` headers with
  `## STDOUT:`/`## status:` expectations (comment lines) — needs a small
  splitter; ignore `ysh-*`/`hay*` files (Oils' own language, not bash).
- **gnu bash tests**: `tests/*.tests` scripts with `*.right` expected-output
  files.
- **wine cmd tests**: `test_builtins.cmd` + `test_builtins.cmd.exp` expected
  output (`@todo_wine@`-prefixed lines are known-broken *in wine*, i.e. real
  cmd.exe behaves as written).

## Two corpora need extracting first

Two suites pack many cases per file, so they get a one-time split into plain
programs plus their expected stdout. Both extractors are idempotent and write
only into the (gitignored) payload, under `<corpus>/extracted/`:

```bash
./tests/reference/extract-phpt.py
```
```bash
./tests/reference/extract-oils.py
```

[extract-phpt.py](extract-phpt.py) turns each `.phpt` into `NAME.php` +
`NAME.expected`. Only tests with a literal `--EXPECT--` survive; `--EXPECTF--`
(pattern), `--SKIPIF--`, `--INI--`, `--EXTENSIONS--` and friends are skipped
and counted, because their expected output depends on a PHP runtime we do not
reproduce. Currently 2,753 of 5,783 tests extract.

[extract-oils.py](extract-oils.py) splits each `spec/*.test.sh` into
`NNN-slug.sh` + `.expected` (+ `.status`/`.stderr` when the case states them),
resolving per-shell overrides **for bash**: `## OK bash …` / `## BUG bash …`
replace the generic expectation, `## N-I bash` cases are skipped, and overrides
naming only other shells are ignored. YSH files are skipped wholesale.
Currently 2,562 cases from 139 spec files.

Both write a `manifest.tsv` next to the extracted cases.

## The ratchet sweep

[sweep.sh](sweep.sh) is the reference-corpus counterpart of `./test.sh --full`:
it runs every corpus file through its language's grammar and reports how much
of the real language we parse today.

```bash
./tests/reference/sweep.sh kotlin
```

```
LANGUAGE    EXPECT      FILES  PARSED           PARSE-FAIL     TIMEOUT
kotlin      parse         534     312  58.4%      220  41.2%      2
            reject?       167      27  16.2%      140  83.8%      0

Top parse-failure clusters (token at the position the parse died):
  kotlin:
       88  interface
       41  <
```

Each file is run as `mec <grammar> <file> -q -warn-unsupported -warn-imports`
and classified by what it printed: **parsed** (no parse error — it may still
fail later at runtime, which is a separate question), **PARSE-FAIL** (`Not
everything could be parsed` — a real grammar gap), or **timeout**. Expectations
come from each suite's own convention, as listed in the format notes above:
`parse` (must parse), `reject` (genuine syntax errors — test262 `fail/`, where
PARSE-FAIL is the *good* outcome), and `reject?` (upstream negative tests that
are mostly *semantic*, so a syntax-only front end may legitimately parse them —
reported but not judged).

The payoff is the cluster histogram: for every parse failure the runner records
the token sitting where the parse died, then ranks them. The top cluster names
the construct that would buy the most coverage next. Per-file results land in
`.results/sweep.tsv` (`lang, expect, result, file, position, token`) so you can
drill in:

```bash
awk -F'\t' '$1=="kotlin" && $6=="interface"' tests/reference/.results/sweep.tsv | head
```

Useful flags: `-n 200` samples a few files per language for a quick look, `-f
PATTERN` restricts to matching paths, `-j`/`-t` set jobs and the per-file
timeout, `-c to-llvm-ir` sweeps the compiler grammars instead of the
interpreters, and `-l` lists the table. Like `test.sh --full`, it is
informational and always exits 0. A full sweep is ~32k runs and takes roughly
an hour on 16 cores; use `-n` or a language name while iterating.

### Baseline, 2026-07-26 (32,117 files, interpreter grammars)

The first full sweep — the numbers to ratchet upward. `parse` files must parse;
for `reject` (test262's genuine syntax errors) the PARSE-FAIL column is the
score instead.

| Language | expect | files | parsed | parse-fail | timeout |
|---|---|--:|--:|--:|--:|
| bash | parse | 2562 | 896 (35.0%) | 1666 | 0 |
| batch | parse | 2 | 0 (0.0%) | 2 | 0 |
| c | parse | 220 | 139 (63.2%) | 81 | 0 |
| csharp | parse | 2752 | 1319 (47.9%) | 1433 | 0 |
| dart | parse | 3449 | 1228 (35.6%) | 2215 | 6 |
| | reject? | 1083 | 430 (39.7%) | 652 | 1 |
| go | parse | 2682 | 870 (32.4%) | 1809 | 3 |
| | reject? | 714 | 285 (39.9%) | 428 | 1 |
| java | parse | 4436 | 1962 (44.2%) | 2474 | 0 |
| | reject? | 1198 | 631 (52.7%) | 567 | 0 |
| js | parse | 2651 | 2045 (77.1%) | 555 | 51 |
| | reject | 729 | 347 (47.6%) | **375 (51.4%)** | 7 |
| kotlin | parse | 542 | 274 (50.6%) | 268 | 0 |
| | reject? | 159 | 23 (14.5%) | 136 | 0 |
| lua | parse | 33 | 5 (15.2%) | 28 | 0 |
| php | parse | 2753 | 1326 (48.2%) | 1427 | 0 |
| python | parse | 30 | 2 (6.7%) | 28 | 0 |
| ruby | parse | 162 | 28 (17.3%) | 134 | 0 |
| swift | parse | 100 | 29 (29.0%) | 70 | 1 |
| | reject? | 165 | 33 (20.0%) | 132 | 0 |
| typescript | parse | 5695 | 3262 (57.3%) | 2420 | 13 |

Reading the numbers: a low percentage does not always mean a weak grammar. The
Python and Lua corpora are a handful of very large files that each exercise the
whole language, so one unknown construct fails a whole file; test262's tiny
one-construct-per-file shape is why JS scores highest. The two rows worth
attention on their own terms are **js/reject** — 347 files that are genuinely
invalid JavaScript parse anyway, i.e. the grammar is too permissive — and the
51 JS timeouts, which are PEG backtracking blowups rather than missing syntax.

The cluster histograms name the next feature in each language: `do` (81 files)
dominates Ruby, `&$` and `<<<` (by-reference parameters and heredocs) dominate
PHP, and in TypeScript all 588 files that die at line 1 column 1 do so because
they open with a UTF-8 BOM — three bytes of encoding, not a syntax gap, and by
far the cheapest win in the table.
