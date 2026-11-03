#!/usr/bin/env bash
# A syntax error at the top level while set -e is in force: the shell leaves at the first
# line of the report, status 2, after the EXIT trap. Real bash 5.3 prints
#   tests/bash-abort-test-synerr-e.sh: line 10: syntax error near unexpected token `then'
# and no second line, after "before", then "trap 2".
trap 'echo "trap $?"' EXIT
set -e
echo before
eval 'fi' 2>/dev/null || echo "eval status $?"
if then
echo not reached
