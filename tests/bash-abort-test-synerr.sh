#!/usr/bin/env bash
# A syntax error at the top level. The lines before it run, the commands on its own line do
# not, and the script leaves with status 2 after the EXIT trap. Real bash 5.3 prints
#   tests/bash-abort-test-synerr.sh: line 13: syntax error near unexpected token `fi'
#   tests/bash-abort-test-synerr.sh: line 13: `echo dropped; fi'
# after "before", "inside", "same line" and "eval status 2", then "trap 2".
trap 'echo "trap $?"' EXIT
echo before
if true; then
  echo inside
fi; echo same line
eval 'x=y() { :; }' 2>/dev/null || echo "eval status $?"
echo dropped; fi
echo not reached
