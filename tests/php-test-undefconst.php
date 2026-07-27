<?php
/* Reading an undefined bareword constant is a FATAL error in PHP 8 ("Undefined
   constant NOPE"), not a null. Both halves must stop here, and stop at the same
   statement: the interpreter always did, while the compiler read the name with
   js_phclslookup - which answers null for an unknown name so an interface list can
   carry a forward reference - and quietly evaluated NOPE to null, printed a blank
   line and ran on. This file is a SHOULD-FAIL ratchet for that: the two lines before
   the fault must both appear (a declared constant still reads, and a constant
   declared AS null still reads null rather than being mistaken for an undefined one),
   and "after" must never appear. */

const KNOWN = 7;
const NULLC = null;

echo KNOWN, "\n";
echo (NULLC === null) ? "null-const-ok\n" : "null-const-BROKEN\n";
echo NOPE, "\n";
echo "after\n";
