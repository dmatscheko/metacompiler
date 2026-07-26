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

## Suggested workflow (ratchet)

Sweep a corpus with the matching grammar in recognize/parse mode, count
pass/fail, fix the biggest cluster, repeat — same spirit as the
`tests/*-test-full.*` ratchet files, but against the language's own reference
suite. The per-file pass/fail semantics above tell the runner which files are
*supposed* to be rejected.
