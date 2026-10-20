#!/usr/bin/env bash
# Standard output and standard error written in turn reach a shared file in program
# order: bash writes each command's output as it runs. Checked from outside:
# tests/native-full.sh runs the native binary with 2>&1 into a file (stdout is then
# not a terminal) and compares with tests/bash-stderr.expected (real bash 5.3's
# output). Native only: under llvm.Run standard error is dropped.
echo out1
echo err1 >&2
printf 'out2 '
echo err2 >&2
echo out3
{ echo g1; echo g2 >&2; }
f() { echo f-out; printf 'f-err\n' >&2; }
f
for i in 1 2; do echo "o$i"; echo "e$i" >&2; done
echo end
