<?php
/* A break or continue that would leave a finally body is a COMPILE-time error in
   PHP ("jump out of a finally block is disallowed", Zend/tests/try/try_finally_008
   and try_finally_009), so the program must be refused before it runs: "before"
   must never appear. A break that leaves a loop INSIDE the finally body is legal.
   This file is a SHOULD-FAIL ratchet for both halves. */

echo "before\n";
function inside() {
    try {
    } finally {
        foreach ([1, 2] as $v) { break; }
    }
}
function out() {
    for ($i = 0; $i < 3; $i++) {
        try {
        } finally {
            continue;
        }
    }
}
out();
