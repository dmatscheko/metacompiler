<?php
// PHP completion self test: the features added on top of php-test-1.php --
// the null-coalescing operator '??', the Elvis operator '?:', and positional
// destructuring assignment with list($a, $b) = ... and [$a, $b] = ....
//
// Like php-test-1.php this counts failed checks in $fails and ends with
// exit($fails), so the metacompiler run exits 0 exactly when every check passes.
// The interpreter (php-interpreter.abnf) and the LLVM-IR compiler
// (php-to-llvm-ir.abnf) run the same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- null-coalescing '??' : yields the first operand that is not null -----
$nn = null;
check("coalesce null var", $nn ?? 7, 7);
$set = 5;
check("coalesce set var", $set ?? 7, 5);
check("coalesce literal null", null ?? "fallback", "fallback");
check("coalesce chain both null", null ?? null ?? "third", "third");
check("coalesce chain short", null ?? "second" ?? "third", "second");

// '??' tests null, NOT falsiness: 0 and "" are kept (this is what sets it apart
// from Elvis below).
$zero = 0;
check("coalesce keeps zero", $zero ?? 99, 0);
$empty = "";
check("coalesce keeps empty", $empty ?? "x", "");

// '??' binds looser than '||' : the right operand is a whole ||-expression.
check("coalesce looser than or", null ?? false || true, true);

// ----- Elvis '?:' : yields the left operand when it is truthy, else the right -----
check("elvis falsy zero", 0 ?: "fb", "fb");
check("elvis truthy int", 7 ?: "fb", 7);
check("elvis empty string", "" ?: "empty", "empty");
check("elvis nonempty string", "hi" ?: "x", "hi");
$e = null;
check("elvis null", $e ?: "was null", "was null");
$g = 42;
check("elvis keeps value", $g ?: 0, 42);

// '??' binds tighter than the ternary/Elvis: (null ?? 0) ?: "z" -> 0 is falsy -> "z".
check("coalesce then elvis", null ?? 0 ?: "z", "z");

// The full ternary still parses and evaluates alongside the new operators.
$t = 3 > 1 ? "big" : "small";
check("ternary still works", $t, "big");

// ----- destructuring: [$a, $b] = ... and list($a, $b) = ... -----
[$a, $b] = [10, 20];
check("bracket destructure a", $a, 10);
check("bracket destructure b", $b, 20);

list($c, $d) = [3, 4];
check("list() destructure c", $c, 3);
check("list() destructure d", $d, 4);

// The right side may be any expression yielding an array.
$pair = [100, 200];
[$p1, $p2] = $pair;
check("destructure from var 1", $p1, 100);
check("destructure from var 2", $p2, 200);

// The whole right side is built before any target is written, so a swap works.
$s1 = 1;
$s2 = 2;
[$s1, $s2] = [$s2, $s1];
check("swap first", $s1, 2);
check("swap second", $s2, 1);

// Three targets.
[$x, $y, $z] = [7, 8, 9];
check("triple x", $x, 7);
check("triple y", $y, 8);
check("triple z", $z, 9);

// Destructure a function's return value.
function makePair() {
    return [11, 22];
}
[$m1, $m2] = makePair();
check("destructure call 1", $m1, 11);
check("destructure call 2", $m2, 22);

// Targets can be array elements, not just plain variables.
$dst = [0, 0];
[$dst[0], $dst[1]] = [55, 66];
check("destructure into index 0", $dst[0], 55);
check("destructure into index 1", $dst[1], 66);

// The destructured values are ordinary variables afterwards.
[$u, $w] = [4, 6];
check("destructure then use", $u + $w, 10);

// ----- done -----
if ($fails === 0) {
    echo "PHP completion self test passed\n";
}
exit($fails);
