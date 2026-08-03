#!/usr/bin/env python3
"""coro-floor.py - hand tests/coro-poc/build.sh a COPY of the C floor.

This script used to PATCH a copy of languages/lib/runtime.c with the coroutine
proof of concept, because the floor did not have one.  It does now: tags 15/16,
the CORO/RES registry, the per-coroutine jmp_buf pool and the throw barrier are
all in languages/lib/runtime.c itself, so the patch is empty and
`build.sh --diff` prints nothing.

What remains is the one thing build.sh still needs: a COPY of the floor in a
temp directory that `--break` can mutate without touching the real file.  The
`--check` mode is what keeps this honest - it fails loudly if the coroutine
block ever disappears from the floor, so a silently-empty PoC is impossible.
"""
import subprocess
import sys

# Every one of these must be present in the floor, or the PoC below is testing
# nothing.  They are the pieces docs/runtime-next-plan.md lists.
REQUIRED = [
    ("libc prototypes", "long pthread_cond_broadcast(void *c);"),
    ("registry globals", "long CORO;           /* long *: the live coroutines' control blocks */"),
    ("gc_trace tag 15", "\tif (t == 15) { gc_try(w[1], 1); gc_try(w[2], 1); "
                        "gc_try(w[4], 1); gc_try(w[6], 1); return; }"),
    ("gc_trace tag 16", "\tif (t == 16) { gc_try(w[1], 1); return; }"),
    ("gc_roots coroutine stacks", "\t\t\tgc_scan_range(cb[4], cb[5]);"),
    ("growable registries", "long coro_grow(long arr, long cap, long newcap) {"),
    ("gc_roots resumer stacks", "\t\tgc_scan_range(lo[i], hi[i]);"),
    ("per-coroutine jmp_buf pool", "\t\t\tgc_scan_jb(cb[9], cb[10]);"),
    ("is_callable tag 16", "t == 9 || t == 16; }"),
    ("get_member generator.next", 'str_eq_c(key, "next")) { return mk_bound(obj, 60); }'),
    ("js_call tag 16", "\tif (t == 16) { return gen_create(callee, args); }"),
    ("the throw barrier", "\tbarrier = jb_at(0);"),
    ("the re-raise on the resumer", "\t\tjs_throw(ff(g));"),
    ("builtin_method mid 60", "\tif (mid == 60) { return gen_next(recv, args); }"),
]


def main() -> int:
    args = sys.argv[1:]
    out = None
    src_path = "languages/lib/runtime.c"
    from_head = False
    check = False
    i = 0
    while i < len(args):
        if args[i] == "-o":
            i += 1
            out = args[i]
        elif args[i] == "--head":
            from_head = True
        elif args[i] == "--check":
            check = True
        else:
            src_path = args[i]
        i += 1

    if from_head:
        text = subprocess.check_output(
            ["git", "show", "HEAD:" + src_path], text=True)
    else:
        with open(src_path, encoding="utf-8") as f:
            text = f.read()

    missing = [name for name, needle in REQUIRED if text.count(needle) != 1]
    if missing:
        sys.stderr.write("coro-floor.py: the C floor is missing %d of the %d "
                         "coroutine pieces: %s\n"
                         % (len(missing), len(REQUIRED), ", ".join(missing)))
        return 2
    if check:
        sys.stderr.write("coro-floor.py: all %d coroutine pieces present in %s\n"
                         % (len(REQUIRED), src_path))
        return 0

    if out:
        with open(out, "w", encoding="utf-8") as f:
            f.write(text)
    else:
        sys.stdout.write(text)
    return 0


if __name__ == "__main__":
    sys.exit(main())
