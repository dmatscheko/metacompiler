<?php
// Multi-file PHP test: the free function geom_area() and the class Rect live in
// tests/imports/geomlib.php and are pulled in with `require 'geomlib.php';`, found via the
// -i include root (mec -i tests/imports ...). The imported file is parsed with the same
// grammar; its function and class register globally, like the main file's own
// declarations. The program counts failed checks in $fails and ends with exit($fails), so
// the run exits 0 exactly when every check passes. The interpreter (php-interpreter.abnf)
// and the LLVM-IR compiler (php-to-llvm-ir.abnf) run the same file and must agree.
require 'geomlib.php';

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// imported free function
check("imported function", geom_area(6, 7), 42);

// imported class: constructor + instance method
$r = new Rect(3, 4);
check("imported method", $r->area(), 12);

// imported class: a method that returns a new instance, then a call on it
$big = $r->scale(2);
check("imported chain", $big->area(), 48);

if ($fails === 0) {
    echo "php multifile test passed\n";
}
exit($fails);
