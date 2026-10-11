<?php
/* A goto into a finally body from outside it is a COMPILE-time error in PHP ("jump
   into a finally block is disallowed", Zend/tests/try/finally_goto_001, 002 and
   004), wherever the goto stands - before the finally, after it, or in the try
   body - so the program must be refused before it runs: "before" must never appear.
   A goto that stays inside its finally body is legal (finally_goto_003). This file
   is a SHOULD-FAIL ratchet for both halves. */

echo "before\n";
function inside() {
    try {
    } finally {
        goto here;
        here:
    }
}
function into() {
    try {
    } finally {
        test:
    }
    goto test;
}
into();
