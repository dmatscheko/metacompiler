# ABORT PROBE. This program is SUPPOSED to die: set -e at the TOP LEVEL of a script ends
# it, with that command's status, at the first failing command it does not exempt. See
# the "SHOULD ABORT" rows in .vscode/launch.json and abort_norm() in test.sh.
set -e
false && true
! true
if true; then false && true; fi
echo "the exempt failures went on"
true && false
echo "never reached"
