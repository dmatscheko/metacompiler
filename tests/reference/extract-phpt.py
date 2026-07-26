#!/usr/bin/env python3
"""extract-phpt.py - split php-src .phpt files into runnable .php + .expected pairs.

The php-src corpus (tests/reference/php/php-src-tests, see README.md) ships its
tests in the .phpt container format: named sections introduced by lines like
--TEST--, --FILE--, --EXPECT--. The metacompiler wants a plain PHP program and
the stdout it should produce, so this walks the corpus and writes, per usable
test, two files under <corpus>/extracted/:

    <relative/path/name>.php        the --FILE-- section verbatim
    <relative/path/name>.expected   the --EXPECT-- section verbatim

plus a manifest.tsv (relative php path, expected path, test title).

Tests that need a PHP runtime feature we cannot reproduce are skipped and
counted by reason - only --FILE-- plus a literal --EXPECT-- survives. In
particular --EXPECTF--/--EXPECTREGEX-- are pattern matches (%s, %d, ...), and
--SKIPIF--/--INI--/--EXTENSIONS--/--ENV--/... make the expected output depend on
the interpreter's configuration.

Idempotent: rewrites the extracted/ tree in place; nothing outside it is
touched. extracted/ lives inside the gitignored payload, so nothing here is
ever committed.

Usage:
    ./extract-phpt.py [-q] [CORPUS_DIR]      (default: php/php-src-tests)
"""

import os
import re
import sys

SECTION = re.compile(r'^--([A-Z_]+)--\s*$')

# Sections whose presence means the expected output is not a plain literal we
# can reproduce by running the --FILE-- program on its own.
DISQUALIFYING = {
    'SKIPIF': 'needs a runtime capability probe (--SKIPIF--)',
    'INI': 'depends on php.ini settings (--INI--)',
    'EXTENSIONS': 'needs a PHP extension (--EXTENSIONS--)',
    'ENV': 'depends on the environment (--ENV--)',
    'GET': 'needs a web request context (--GET--)',
    'POST': 'needs a web request context (--POST--)',
    'POST_RAW': 'needs a web request context (--POST_RAW--)',
    'COOKIE': 'needs a web request context (--COOKIE--)',
    'STDIN': 'needs stdin (--STDIN--)',
    'FILE_EXTERNAL': 'program lives in another file (--FILE_EXTERNAL--)',
    'EXPECTHEADERS': 'checks HTTP headers (--EXPECTHEADERS--)',
    'PHPDBG': 'needs the phpdbg driver (--PHPDBG--)',
    'XFAIL': 'known-failing upstream (--XFAIL--)',
    'FLAKY': 'flaky upstream (--FLAKY--)',
    'CLEAN': 'has a cleanup phase (--CLEAN--)',
}


def parse_sections(text):
    """Split .phpt text into an ordered dict-ish list of (name, body)."""
    sections = {}
    name = None
    body = []
    for line in text.splitlines(keepends=True):
        m = SECTION.match(line)
        if m:
            if name is not None:
                sections[name] = ''.join(body)
            name = m.group(1)
            body = []
        elif name is not None:
            body.append(line)
    if name is not None:
        sections[name] = ''.join(body)
    return sections


def main():
    args = [a for a in sys.argv[1:] if a != '-q']
    quiet = '-q' in sys.argv[1:]
    root = os.path.dirname(os.path.abspath(__file__))
    corpus = os.path.join(root, args[0] if args else 'php/php-src-tests')

    if not os.path.isdir(corpus):
        sys.exit("extract-phpt: no corpus at %s (run ./fetch.sh php first)" % corpus)

    out_root = os.path.join(corpus, 'extracted')
    manifest = []
    skipped = {}
    total = 0

    for dirpath, dirnames, filenames in os.walk(corpus):
        dirnames[:] = [d for d in dirnames if d != 'extracted']
        for fn in sorted(filenames):
            if not fn.endswith('.phpt'):
                continue
            total += 1
            path = os.path.join(dirpath, fn)
            rel = os.path.relpath(path, corpus)[:-len('.phpt')]

            def skip(reason):
                skipped[reason] = skipped.get(reason, 0) + 1

            try:
                text = open(path, encoding='utf-8', errors='replace').read()
            except OSError as e:
                skip('unreadable (%s)' % e.__class__.__name__)
                continue

            sec = parse_sections(text)
            bad = [DISQUALIFYING[k] for k in sec if k in DISQUALIFYING]
            if bad:
                skip(sorted(bad)[0])
                continue
            if 'FILE' not in sec:
                skip('no --FILE-- section')
                continue
            if 'EXPECT' not in sec:
                if 'EXPECTF' in sec:
                    skip('expected output is a pattern (--EXPECTF--)')
                elif 'EXPECTREGEX' in sec:
                    skip('expected output is a regex (--EXPECTREGEX--)')
                else:
                    skip('no --EXPECT-- section')
                continue

            php_path = os.path.join(out_root, rel + '.php')
            exp_path = os.path.join(out_root, rel + '.expected')
            os.makedirs(os.path.dirname(php_path), exist_ok=True)
            # .phpt bodies carry the newline that preceded the next section
            # header; strip exactly that one so the program/expectation are the
            # bytes upstream's runner would use.
            with open(php_path, 'w', encoding='utf-8') as f:
                f.write(sec['FILE'].rstrip('\n') + '\n')
            with open(exp_path, 'w', encoding='utf-8') as f:
                f.write(sec['EXPECT'].rstrip('\n') + '\n')

            title = sec.get('TEST', '').strip().splitlines()
            manifest.append((os.path.relpath(php_path, corpus),
                             os.path.relpath(exp_path, corpus),
                             title[0] if title else ''))

    os.makedirs(out_root, exist_ok=True)
    with open(os.path.join(out_root, 'manifest.tsv'), 'w', encoding='utf-8') as f:
        f.write('#php\texpected\ttitle\n')
        for row in manifest:
            f.write('\t'.join(row) + '\n')

    if not quiet:
        print("extract-phpt: %d .phpt files, %d extracted -> %s"
              % (total, len(manifest), os.path.relpath(out_root, root)))
        for reason, n in sorted(skipped.items(), key=lambda kv: -kv[1]):
            print("  skipped %5d  %s" % (n, reason))


if __name__ == '__main__':
    main()
