# ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
# thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
# abort_norm() in test.sh.
# The diagnostic is written on standard error, which llvm.Run has none of (see rt_putc
# in languages/lib/bash-rt.c), so the shell's standard error is made a copy of its
# standard output first, and every engine shows the same text.
exec 2>&1
echo "before the abort"
nosuchcommand_abort_probe
echo "never reached"
