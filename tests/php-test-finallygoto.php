<?php
/* A goto out of a finally body is a COMPILE-time error in PHP ("jump out of a
   finally block is disallowed", Zend/tests/try/try_finally_005.phpt), so the
   program must be refused before it runs: "before" must never appear. A goto that
   stays inside its finally body is legal (finally_goto_003.phpt). This file is a
   SHOULD-FAIL ratchet for both halves. */

echo "before\n";
function inside() {
    try {
    } finally {
        goto here;
        here:
    }
}
function out() {
    try {
    } finally {
        goto label;
    }
label:
    return 1;
}
out();
