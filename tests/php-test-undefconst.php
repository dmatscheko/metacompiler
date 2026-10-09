<?php
/* Reading an undefined bareword constant is an uncaught Error in PHP 8 ("Undefined
   constant NOPE"), not a null. Both halves must stop here, at the same statement -
   the compiler's js_phclslookup answers null for an unknown name (so an interface
   list can carry a forward reference), so it has to tell "undefined" apart from
   "declared as null". This file is a SHOULD-FAIL ratchet: the two lines before
   the fault must both appear (a declared constant still reads, and a constant
   declared AS null still reads null rather than being mistaken for an undefined one),
   and "after" must never appear. */

const KNOWN = 7;
const NULLC = null;

echo KNOWN, "\n";
echo (NULLC === null) ? "null-const-ok\n" : "null-const-BROKEN\n";
echo NOPE, "\n";
echo "after\n";
