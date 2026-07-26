#!/usr/bin/env bash
#
# fetch.sh - materialize third-party language reference corpora into tests/reference/.
#
# The payload directories are NOT committed (see .gitignore): this script plus
# pins.lock plus README.md are the committed, reproducible record of exactly which
# files come from where. Every fetched corpus directory gets the upstream LICENSE/
# COPYING file(s) and a generated PROVENANCE.md (upstream URL, commit, subset).
#
# Usage:
#   ./fetch.sh                 fetch every corpus
#   ./fetch.sh kotlin php      fetch only corpora whose name or language dir matches
#   ./fetch.sh --list          list corpora and exit
#
# Pinning: the first fetch of a corpus resolves the upstream default branch to a
# commit sha and records it in pins.lock (committed). Later runs reuse the locked
# sha. To upgrade one corpus: delete its pins.lock line and its payload directory,
# then re-run. To re-fetch without upgrading: just delete the payload directory.
#
# Fetches are shallow partial clones (depth 1, blob:none, sparse checkout), so only
# the wanted subtrees are downloaded even from huge repos.

set -euo pipefail
cd "$(dirname "$0")"

CACHE=.cache
LOCK=pins.lock
touch "$LOCK"

LIST=0
if [ "${1:-}" = "--list" ]; then LIST=1; shift || true; fi
SELECT="$*"

FAILED=""
FETCHED=""

# is corpus $1 (dest dir $2) selected by the command line?
sel() {
    [ -z "$SELECT" ] && return 0
    local w
    for w in $SELECT; do
        case "$1" in *"$w"*) return 0 ;; esac
        case "$2" in *"$w"*) return 0 ;; esac
    done
    return 1
}

locked_sha() {
    awk -F'\t' -v n="$1" '$1 == n { print $3; exit }' "$LOCK"
}

copy_licenses() { # $1 = cache repo dir, $2 = dest dir
    local f
    for f in LICENSE LICENSE.txt LICENSE.md LICENSE.rst COPYING COPYING.LIB license/LICENSE.txt licenses.md; do
        if [ -f "$1/$f" ]; then cp "$1/$f" "$2/$(echo "$f" | tr / -)"; fi
    done
}

provenance() { # $1 dest, $2 upstream, $3 pin, $4 license, $5 subset description
    {
        echo "# Provenance"
        echo
        echo "Fetched by tests/reference/fetch.sh - do not edit, not committed."
        echo
        echo "- Upstream: $2"
        echo "- Pin: $3"
        echo "- License: $4 (upstream license text copied into this directory)"
        echo "- Subset: $5"
        echo "- Fetched: $(date +%F)"
    } > "$1/PROVENANCE.md"
}

# git_corpus <name> <url> <dest> <license-name> <subpath>...
git_corpus() {
    local name=$1 url=$2 dest=$3 lic=$4
    shift 4
    local subs="$*"
    sel "$name" "$dest" || return 0
    if [ "$LIST" = 1 ]; then printf '%-28s %-34s %s\n' "$name" "$dest" "$url ($lic)"; return 0; fi
    if [ -e "$dest/.fetched" ]; then echo "[skip] $name (present; delete $dest to refetch)"; return 0; fi

    local sha
    sha=$(locked_sha "$name")
    if [ -z "$sha" ]; then
        sha=$(git ls-remote "$url" HEAD | head -n1 | cut -f1) || true
        if [ -z "$sha" ]; then echo "[FAIL] $name: cannot resolve HEAD of $url"; return 1; fi
        printf '%s\t%s\t%s\n' "$name" "$url" "$sha" >> "$LOCK"
    fi
    echo "[git ] $name @ ${sha:0:12}  <- $url"

    local repo=$CACHE/$name
    if [ ! -d "$repo/.git" ]; then
        git init -q "$repo"
        git -C "$repo" remote add origin "$url"
        git -C "$repo" config remote.origin.promisor true
        git -C "$repo" config remote.origin.partialclonefilter blob:none
    fi
    {
        echo "/LICENSE*"
        echo "/COPYING*"
        echo "/license/**"
        echo "/licenses.md"
        local p
        for p in "$@"; do echo "/$p"; echo "/$p/**"; done
    } | git -C "$repo" sparse-checkout set --no-cone --stdin
    git -C "$repo" fetch -q --depth 1 origin "$sha" || { echo "[FAIL] $name: fetch"; return 1; }
    git -C "$repo" checkout -q --detach "$sha" || { echo "[FAIL] $name: checkout"; return 1; }

    mkdir -p "$dest"
    local copied=0 p
    for p in "$@"; do
        if [ -e "$repo/$p" ]; then
            cp -R "$repo/$p" "$dest/$(basename "$p")"
            copied=1
        else
            echo "  [warn] $name: upstream path missing: $p"
        fi
    done
    if [ "$copied" = 0 ]; then echo "[FAIL] $name: no subset path existed"; return 1; fi
    copy_licenses "$repo" "$dest"
    provenance "$dest" "$url" "$sha" "$lic" "$subs"
    touch "$dest/.fetched"
    FETCHED="$FETCHED $name"
}

# tar_corpus <name> <url> <dest> <license-name> <license-note> [subdir]
tar_corpus() {
    local name=$1 url=$2 dest=$3 lic=$4 note=$5 subdir=${6:-}
    sel "$name" "$dest" || return 0
    if [ "$LIST" = 1 ]; then printf '%-28s %-34s %s\n' "$name" "$dest" "$url ($lic)"; return 0; fi
    if [ -e "$dest/.fetched" ]; then echo "[skip] $name (present; delete $dest to refetch)"; return 0; fi
    if [ -z "$(locked_sha "$name")" ]; then
        printf '%s\t%s\t%s\n' "$name" "$url" "tarball-url-is-the-pin" >> "$LOCK"
    fi
    echo "[tar ] $name  <- $url"
    mkdir -p "$CACHE/$name" "$dest"
    curl -fsSL "$url" -o "$CACHE/$name/src.tar.gz" || { echo "[FAIL] $name: download"; return 1; }
    tar -xzf "$CACHE/$name/src.tar.gz" -C "$CACHE/$name" || { echo "[FAIL] $name: extract"; return 1; }
    local top
    top=$(find "$CACHE/$name" -mindepth 1 -maxdepth 1 -type d | head -1)
    if [ -n "$subdir" ]; then
        cp -R "$top/$subdir"/. "$dest"/
    else
        cp -R "$top"/. "$dest"/
    fi
    copy_licenses "$top" "$dest"
    echo "$note" > "$dest/LICENSE.note"
    provenance "$dest" "$url" "$url" "$lic" "${subdir:-whole tarball}"
    touch "$dest/.fetched"
    FETCHED="$FETCHED $name"
}

run() { # tolerate one corpus failing without aborting the rest
    if ! "$@"; then FAILED="$FAILED $2"; fi
}

# ---------------------------------------------------------------------------
# Permissive-licensed corpora (safe to vendor into this Apache-2.0 repo if we
# ever decide to commit the payload; see README.md).
# ---------------------------------------------------------------------------

run git_corpus js-test262-parser-tests https://github.com/tc39/test262-parser-tests \
    js/test262-parser-tests "BSD-style (see LICENSE)" \
    pass fail early

run git_corpus typescript-conformance https://github.com/microsoft/TypeScript \
    typescript/typescript-conformance "Apache-2.0" \
    tests/cases/conformance

run git_corpus python-cpython https://github.com/python/cpython \
    python/cpython-tests "PSF-2.0" \
    Lib/test/test_grammar.py Lib/test/test_syntax.py Lib/test/test_tokenize.py \
    Lib/test/test_string_literals.py Lib/test/test_fstring.py Lib/test/test_patma.py \
    Lib/test/test_named_expressions.py Lib/test/test_genexps.py Lib/test/test_listcomps.py \
    Lib/test/test_dictcomps.py Lib/test/test_setcomps.py Lib/test/test_decorators.py \
    Lib/test/test_global.py Lib/test/test_scope.py \
    Lib/test/test_with.py Lib/test/test_yield_from.py Lib/test/test_coroutines.py \
    Lib/test/test_asyncgen.py Lib/test/test_positional_only_arg.py \
    Lib/test/test_keywordonlyarg.py Lib/test/test_unpack.py Lib/test/test_unpack_ex.py \
    Lib/test/test_augassign.py Lib/test/test_type_params.py Lib/test/test_raise.py \
    Lib/test/test_exceptions.py Lib/test/test_generators.py Lib/test/test_compile.py \
    Lib/test/test_super.py Lib/test/test_descr.py

run git_corpus kotlin-psi https://github.com/JetBrains/kotlin \
    kotlin/kotlin-psi-testdata "Apache-2.0" \
    compiler/psi/psi-impl/testData

run git_corpus ruby-spec https://github.com/ruby/spec \
    ruby/ruby-spec-language "MIT" \
    language

run git_corpus php-src https://github.com/php/php-src \
    php/php-src-tests "PHP-3.01" \
    tests/lang Zend/tests

run git_corpus csharp-mono https://github.com/mono/mono \
    csharp/mono-mcs-tests "MIT" \
    mcs/tests

run git_corpus swift-parse https://github.com/swiftlang/swift \
    swift/swift-parse-tests "Apache-2.0 WITH Swift-exception" \
    test/Parse

run git_corpus dart-language https://github.com/dart-lang/sdk \
    dart/dart-language-tests "BSD-3-Clause" \
    tests/language

run git_corpus go-test https://github.com/golang/go \
    go/go-test-suite "BSD-3-Clause" \
    test

run git_corpus bash-oils https://github.com/oils-for-unix/oils \
    bash/oils-spec "Apache-2.0" \
    spec

run tar_corpus lua-tests https://www.lua.org/tests/lua-5.4.6-tests.tar.gz \
    lua/lua-tests "MIT" \
    "Lua and its official test suite are MIT licensed: https://www.lua.org/license.html"

# ---------------------------------------------------------------------------
# Copyleft-licensed corpora - kept fetch-only ON PURPOSE, never commit these.
# Using them locally (running our parser/interpreter over them) creates no
# obligations for our Apache-2.0 code; committing them would put GPL/LGPL
# material in the repo. See README.md.
# ---------------------------------------------------------------------------

run git_corpus c-testsuite https://github.com/c-testsuite/c-testsuite \
    c/c-testsuite "mixed: MIT runner; cases per .otags files - scc (ISC) and tinycc (LGPL); do not commit" \
    tests

run git_corpus java-openjdk-javac https://github.com/openjdk/jdk \
    java/openjdk-javac-tests "GPL-2.0 WITH Classpath-exception-2.0 (do not commit)" \
    test/langtools/tools/javac

run tar_corpus bash-gnu https://ftp.gnu.org/gnu/bash/bash-5.3.tar.gz \
    bash/gnu-bash-tests "GPL-3.0-or-later (do not commit)" \
    "GNU bash and its test suite are GPL-3.0-or-later; see COPYING copied alongside." \
    tests

run git_corpus batch-wine https://github.com/wine-mirror/wine \
    batch/wine-cmd-tests "LGPL-2.1-or-later (do not commit)" \
    programs/cmd/tests

# ---------------------------------------------------------------------------
# Opt-in extras - uncomment to fetch (big and/or only needed later):
#
# Full ECMAScript conformance suite (execution semantics, ~40k files; needs its
# harness/*.js includes to run):
# run git_corpus js-test262 https://github.com/tc39/test262 \
#     js/test262 "BSD-3-Clause (Ecma)" test/language harness
#
# GCC C torture tests (GPL-3.0, huge repo - slow even as partial clone):
# run git_corpus c-gcc-torture https://github.com/gcc-mirror/gcc \
#     c/gcc-torture "GPL-3.0 (do not commit)" gcc/testsuite/gcc.c-torture
# ---------------------------------------------------------------------------

[ "$LIST" = 1 ] && exit 0
echo
if [ -n "$FETCHED" ]; then echo "fetched:$FETCHED"; fi
if [ -n "$FAILED" ]; then
    echo "FAILED:$FAILED"
    exit 1
fi
echo "done."
