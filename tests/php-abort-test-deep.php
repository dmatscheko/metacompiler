<?php
// ABORT PROBE. This program is SUPPOSED to die. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. A userland recursion with no
// base case runs on PHP's VM stack, not the C stack, so it is not the catchable
// "Maximum call stack size" Error (zend_call_stack_size_error is checked only
// where the engine re-enters execute_ex or zend_call_function) but the memory
// limit: an uncatchable fatal error, exit status 255, so neither the catch nor
// the finally runs. Spec-cited from php-src: no php on this machine.
function rec($n) { return rec($n + 1) + 1; }
echo "before the abort\n";
try { rec(0); } catch (Error $e) { echo "caught\n"; } finally { echo "finally\n"; }
echo "never reached\n";
