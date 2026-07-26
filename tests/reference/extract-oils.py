#!/usr/bin/env python3
"""extract-oils.py - split Oils spec files into runnable .sh + .expected pairs.

The Oils bash-compatibility suite (tests/reference/bash/oils-spec, see
README.md) packs many cases per file: a case starts at a `#### title` line, its
shell code runs to the first `## ` metadata line, and the metadata states what
bash should print and exit with. This walks the suite and writes, per usable
case, files under <corpus>/extracted/<specfile>/:

    NNN-slug.sh          the shell code
    NNN-slug.expected    the stdout bash should produce
    NNN-slug.status      the exit status, only when it is not 0
    NNN-slug.stderr      the expected stderr, only when the case states one

plus a manifest.tsv (sh path, expected path, status, spec file, title).

Expectation forms handled: `## STDOUT:` ... `## END` blocks, `## stdout: LINE`,
`## stdout-json: "..."`, `## status: N`, and the same for stderr. Per-shell
overrides are resolved FOR BASH: `## OK bash ...` and `## BUG bash ...` state
what bash itself does, so they replace the generic expectation; a case marked
`## N-I bash` (not implemented in bash) is skipped, as are overrides naming
only other shells. Files that test Oils' own YSH language rather than bash
(ysh-*/hay-*, or `## our_shell: ysh`) are skipped wholesale.

Idempotent: rewrites the extracted/ tree in place; nothing outside it is
touched. extracted/ lives inside the gitignored payload, so nothing here is
ever committed.

Usage:
    ./extract-oils.py [-q] [CORPUS_DIR]      (default: bash/oils-spec)
"""

import json
import os
import re
import sys

CASE = re.compile(r'^####\s*(.*)$')
META = re.compile(r'^##\s+(.*)$')
# "OK bash/mksh stdout:", "BUG zsh status:", "N-I dash stdout-json:"
OVERRIDE = re.compile(r'^(OK|BUG|N-I)(?:-\d+)?\s+(\S+)\s+(.*)$')

TARGET_SHELL = 'bash'
# File-level keys that say nothing about a single case.
FILE_KEYS = ('compare_shells:', 'our_shell:', 'suite:', 'tags:',
             'oils_failures_allowed:', 'oils_cpp_failures_allowed:',
             'legacy_tmp_dir:')


def slug(title, n):
    s = re.sub(r'[^A-Za-z0-9]+', '-', title).strip('-').lower()
    return '%03d-%s' % (n, (s or 'case')[:60])


def split_cases(lines):
    """Yield (title, body_lines, meta_lines) per `#### case` in one spec file."""
    cur = None
    for line in lines:
        m = CASE.match(line)
        if m:
            if cur:
                yield cur
            cur = (m.group(1).strip(), [], [])
            continue
        if cur is None:
            continue          # file preamble
        title, body, meta = cur
        if META.match(line) or meta:
            # Once metadata starts, everything to the next case belongs to it -
            # STDOUT: block content is not `##`-prefixed, so it must be kept.
            meta.append(line)
        else:
            body.append(line)
    if cur:
        yield cur


def parse_meta(meta_lines):
    """Reduce a case's metadata to bash's expectation.

    Returns (expect, skip_reason). expect = {stdout, stderr, status}, values
    absent when the case does not state them.
    """
    generic = {}
    bash_specific = {}
    skip_reason = None
    i = 0
    while i < len(meta_lines):
        m = META.match(meta_lines[i])
        if not m:
            i += 1
            continue
        rest = m.group(1).rstrip('\n')
        i += 1

        target = generic
        ov = OVERRIDE.match(rest)
        if ov:
            kind, shells, rest = ov.group(1), ov.group(2), ov.group(3)
            if TARGET_SHELL not in shells.split('/'):
                continue                       # about some other shell
            if kind == 'N-I':
                skip_reason = 'not implemented in bash (## N-I bash)'
                continue
            target = bash_specific              # OK/BUG bash: what bash does

        if rest.startswith(('STDOUT:', 'STDERR:')):
            key = 'stdout' if rest.startswith('STDOUT') else 'stderr'
            block = []
            while i < len(meta_lines):
                line = meta_lines[i]
                i += 1
                if line.startswith('## END'):
                    break
                block.append(line)
            target[key] = ''.join(block)
        elif rest.startswith(('stdout-json:', 'stderr-json:')):
            key = 'stdout' if rest.startswith('stdout') else 'stderr'
            raw = rest.split(':', 1)[1].strip()
            try:
                target[key] = json.loads(raw) if raw else ''
            except ValueError:
                skip_reason = 'unparsable %s-json expectation' % key
        elif rest.startswith(('stdout:', 'stderr:')):
            key = 'stdout' if rest.startswith('stdout') else 'stderr'
            target[key] = rest.split(':', 1)[1].strip() + '\n'
        elif rest.startswith('status:'):
            target['status'] = rest.split(':', 1)[1].strip()
        elif rest.startswith('code:') or rest.startswith(FILE_KEYS):
            pass
        # anything else: a prose comment inside the metadata block, ignore

    expect = dict(generic)
    expect.update(bash_specific)       # bash's own behavior wins
    return expect, skip_reason


def main():
    argv = sys.argv[1:]
    quiet = '-q' in argv
    args = [a for a in argv if a != '-q']
    root = os.path.dirname(os.path.abspath(__file__))
    corpus = os.path.join(root, args[0] if args else 'bash/oils-spec')

    spec_dir = os.path.join(corpus, 'spec')
    if not os.path.isdir(spec_dir):
        sys.exit("extract-oils: no corpus at %s (run ./fetch.sh bash-oils first)" % spec_dir)

    out_root = os.path.join(corpus, 'extracted')
    manifest = []
    skipped = {}
    files_used = files_skipped = 0

    def skip(reason):
        skipped[reason] = skipped.get(reason, 0) + 1

    for fn in sorted(os.listdir(spec_dir)):
        if not fn.endswith('.test.sh'):
            continue
        stem = fn[:-len('.test.sh')]
        path = os.path.join(spec_dir, fn)
        lines = open(path, encoding='utf-8', errors='replace').readlines()

        if stem.startswith(('ysh-', 'hay')) or any(
                l.startswith('## our_shell:') and 'ysh' in l for l in lines[:20]):
            files_skipped += 1
            continue
        files_used += 1

        n = 0
        for title, body, meta in split_cases(lines):
            n += 1
            expect, reason = parse_meta(meta)
            if reason:
                skip(reason)
                continue
            if 'stdout' not in expect:
                skip('case states no stdout expectation')
                continue
            if not ''.join(body).strip():
                skip('case has no shell code')
                continue

            base = os.path.join(out_root, stem, slug(title, n))
            os.makedirs(os.path.dirname(base), exist_ok=True)
            with open(base + '.sh', 'w', encoding='utf-8') as f:
                f.write(''.join(body).rstrip('\n') + '\n')
            with open(base + '.expected', 'w', encoding='utf-8') as f:
                f.write(expect['stdout'])
            status = expect.get('status', '0')
            if status != '0':
                with open(base + '.status', 'w', encoding='utf-8') as f:
                    f.write(status + '\n')
            if expect.get('stderr'):
                with open(base + '.stderr', 'w', encoding='utf-8') as f:
                    f.write(expect['stderr'])
            manifest.append((os.path.relpath(base + '.sh', corpus),
                             os.path.relpath(base + '.expected', corpus),
                             status, fn, title))

    os.makedirs(out_root, exist_ok=True)
    with open(os.path.join(out_root, 'manifest.tsv'), 'w', encoding='utf-8') as f:
        f.write('#sh\texpected\tstatus\tspec\ttitle\n')
        for row in manifest:
            f.write('\t'.join(r.replace('\t', ' ') for r in row) + '\n')

    if not quiet:
        print("extract-oils: %d spec files (%d ysh-only skipped), %d cases extracted -> %s"
              % (files_used, files_skipped, len(manifest),
                 os.path.relpath(out_root, root)))
        for reason, cnt in sorted(skipped.items(), key=lambda kv: -kv[1]):
            print("  skipped %5d  %s" % (cnt, reason))


if __name__ == '__main__':
    main()
