<?php
// Fast feature-matrix test for the PHP interpreter (php-interpreter.abnf) and the
// LLVM-IR compiler (php-to-llvm-ir.abnf). It replaces the four algorithm-themed
// php-test-big-* stress tests: instead of large loops (sorting batteries, Roman
// numerals, expression parsers) every implemented construct is exercised with the
// SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
// recursion stays below depth 6. A failed check prints its id (so a diff pinpoints
// it) and the program ends with exit($fails); exit 0 and byte-identical output on
// all four legs (interpreter/compiler x goja/-frozen) mean everything passed.

$fails = 0;
$checks = 0;

function check($name, $got, $want) {
    global $fails;
    global $checks;
    $checks = $checks + 1;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- numbers, arithmetic, precedence (ints only; / truncates toward zero) -----
check("arith-precedence", 2 + 3 * 4, 14);
check("arith-paren", (2 + 3) * 4, 20);
check("arith-unary-minus", -3 + 5, 2);
check("arith-chain", 20 - 5 - 3, 12);
check("arith-div-trunc", 7 / 2, 3);
check("arith-div-neg", -7 / 2, -3);
check("arith-mod", 7 % 3, 1);
check("arith-mod-neg", -7 % 3, -1);
check("arith-intdiv", intdiv(9, 2), 4);
check("arith-intdiv-neg", intdiv(-7, 2), -3);
check("arith-abs", abs(-6), 6);
check("arith-max-min", max(2, 9) + min(2, 9), 11);

$ca = 5;
$ca += 3;
check("compound-plus", $ca, 8);
$ca -= 2;
check("compound-minus", $ca, 6);
$ca *= 4;
check("compound-times", $ca, 24);
$ca /= 5;
check("compound-div-trunc", $ca, 4);
$ca %= 3;
check("compound-mod", $ca, 1);

$inc = 5;
$postVal = $inc++;
check("post-inc", $postVal, 5);
check("post-inc-var", $inc, 6);
$preVal = ++$inc;
check("pre-inc", $preVal, 7);
$inc--;
--$inc;
check("dec-both", $inc, 5);

// ----- comparisons -----
check("cmp-lt-gt", (2 < 3) && (3 > 2) && (3 <= 3) && (3 >= 4) === false, true);
check("cmp-eq", 3 == 3, true);
check("cmp-strict-eq", 3 === 3, true);
check("cmp-neq", (3 != 4) && (3 !== 4), true);
check("cmp-str-eq", "abc" === "abc", true);
check("cmp-str-lt", "apple" < "banana", true);
check("cmp-str-lt-single", ("a" < "b") && !("b" < "a"), true);

// ----- boolean logic (symbol and word forms; && || yield booleans) -----
check("logic-and-or-not", (true && true) && (false || true) && !false, true);
check("logic-word-forms", (2 < 3 and 4 < 5) && (false or 1 < 2), true);
check("logic-and-is-bool", (5 && "x") === true, true);
check("logic-or-is-bool", (0 || "") === false, true);
$sideFx = 0;
function bump() {
    global $sideFx;
    $sideFx = $sideFx + 1;
    return true;
}
$noRun = false && bump();
$oneRun = true && bump();
$skipRun = true || bump();
check("logic-short-circuit", $sideFx, 1);
check("logic-short-values", $noRun === false && $oneRun === true && $skipRun === true, true);

// ----- ternary, Elvis, null coalescing -----
check("ternary", 5 > 3 ? "a" : "b", "a");
check("ternary-false", 5 < 3 ? "a" : "b", "b");
check("elvis-falsy", 0 ?: "fb", "fb");
check("elvis-truthy", 7 ?: 0, 7);
check("elvis-empty-str", "" ?: "empty", "empty");
$nul = null;
check("coalesce-null", $nul ?? 7, 7);
check("coalesce-set", 5 ?? 7, 5);
check("coalesce-chain", null ?? null ?? "third", "third");
$zeroV = 0;
check("coalesce-keeps-zero", $zeroV ?? 99, 0);
$emptyV = "";
check("coalesce-keeps-empty", $emptyV ?? "x", "");
check("coalesce-then-elvis", null ?? 0 ?: "z", "z");

// ----- strings -----
check("str-concat", "foo" . "bar", "foobar");
check("str-concat-num", "n=" . 5 . "!", "n=5!");
check("str-concat-neg", "v" . (-3), "v-3");
check("str-length", strlen("hello"), 5);
check("str-length-empty", strlen(""), 0);
check("str-index", "abcdef"[1], "b");
$idxSrc = "abcdef";
check("str-index-var", $idxSrc[4], "e");
$who = "world";
check("str-interp", "hello $who!", "hello world!");
$n7 = 7;
check("str-interp-num", "n=$n7 mid {$n7} end", "n=7 mid 7 end");
check("str-escapes", strlen("a\tb") + strlen("x\ny"), 6);
check("str-unicode-len", strlen("héllo"), 5);
$build = "a";
$build .= "b";
$build .= "c";
check("str-concat-assign", $build, "abc");

// ----- control flow: if / elseif / else -----
function grade($n) {
    if ($n > 10) {
        return "big";
    } elseif ($n > 5) {
        return "mid";
    } else {
        return "small";
    }
}
check("if-elseif-else", grade(11) . grade(7) . grade(1), "bigmidsmall");

$w0 = 0;
while ($w0 > 0) { $w0 = $w0 - 1; }          // runs zero times
check("while-zero", $w0, 0);
$w3 = 0;
while ($w3 < 3) { $w3 = $w3 + 1; }          // runs three times
check("while-three", $w3, 3);

$forSum = 0;
for ($fi = 1; $fi <= 3; $fi++) { $forSum += $fi; }
check("for-basic", $forSum, 6);

$brk = "";
for ($bi = 0; $bi < 9; $bi++) {
    if ($bi === 2) { break; }
    $brk .= $bi;
}
check("for-break", $brk, "01");

$cont = "";
for ($ci = 0; $ci < 4; $ci++) {
    if ($ci % 2 === 1) { continue; }
    $cont .= $ci;
}
check("for-continue", $cont, "02");

$nested = "";
for ($oi = 0; $oi < 2; $oi++) {
    for ($ii = 0; $ii < 3; $ii++) {
        if ($ii === 1) { break; }           // inner break must not end the outer loop
        $nested .= $oi . $ii;
    }
}
check("nested-break", $nested, "0010");

// ----- functions, recursion, closures -----
function add($a, $b) { return $a + $b; }
check("fn-args", add(20, 22), 42);

function fib($n) {
    if ($n < 2) { return $n; }
    return fib($n - 1) + fib($n - 2);
}
check("fn-recursion", fib(6), 8);

function isEven($n) { return $n === 0 ? true : isOdd($n - 1); }
function isOdd($n) { return $n === 0 ? false : isEven($n - 1); }
check("fn-mutual-recursion", isEven(4) && isOdd(5), true);

$three = 3;
$hundred = 100;
$scale = function ($v) use ($three) { return $v * $three; };
$shift = function ($v) use ($hundred) { return $v + $hundred; };
check("closure-capture", $scale(5), 15);
check("closure-independent", $shift(5), 105);

function makeAdder($base) {
    return function ($v) use ($base) { return $v + $base; };
}
$add10 = makeAdder(10);
check("closure-returned", $add10(5), 15);

$mul = 2;
$off = 3;
$combo = function ($v) use ($mul, $off) { return $v * $mul + $off; };
check("closure-two-captures", $combo(5), 13);

function applyTwice($f, $x) { return $f($f($x)); }
check("fn-higher-order", applyTwice(function ($n) { return $n * 2; }, 3), 12);

// ----- indexed arrays -----
$arr = [10, 20, 30];
check("arr-literal-index", $arr[0] + $arr[2], 40);
check("arr-count", count($arr), 3);
check("arr-count-empty", count([]), 0);
$arr[1] = 21;
check("arr-write", $arr[1], 21);
$arr[] = 40;
check("arr-append", count($arr) === 4 && $arr[3] === 40, true);
check("arr-push", array_push($arr, 50), 5);
check("arr-push-value", $arr[4], 50);
check("arr-in-array", in_array(30, $arr) && !in_array(99, $arr), true);
check("arr-nested", [[1, 2], [3]][0][1], 2);
check("arr-fn-syntax", array(1, 2, 3)[1], 2);

$fe = 0;
foreach ([10, 20, 30] as $v) { $fe += $v; }
check("foreach-values", $fe, 60);
$feIdx = "";
foreach ([5, 6, 7] as $ix => $v) { $feIdx .= $ix . $v; }
check("foreach-index", $feIdx, "051627");
$feEmpty = 0;
foreach ([] as $v) { $feEmpty += 100; }
check("foreach-empty", $feEmpty, 0);

// ----- associative arrays (ordered maps) -----
$ages = ["alice" => 30, "bob" => 25];
check("map-get", $ages["bob"], 25);
$ages["carol"] = 40;
check("map-set-new", $ages["carol"], 40);
check("map-count", count($ages), 3);
check("map-keys-values", count(array_keys($ages)) + count(array_values($ages)), 6);
$order = "";
foreach ($ages as $name => $age) { $order .= $name . ","; }
check("map-order", $order, "alice,bob,carol,");
$ksum = 0;
foreach (["k" => ["deep" => 5]] as $k => $v) { $ksum += $v["deep"]; }
check("map-nested", $ksum, 5);
check("map-nested-get", ["x" => ["y" => 5]]["x"]["y"], 5);
$dynKey = "bob";
check("map-dyn-key", $ages[$dynKey], 25);
$intKeys = 0;
foreach ([2 => "a", 5 => "b"] as $k => $v) { $intKeys += $k; }
check("map-int-keys", $intKeys, 7);
check("map-fn-syntax", array("k" => 99)["k"], 99);

// ----- destructuring -----
[$d1, $d2] = [1, 2];
check("destructure-bracket", $d1 + $d2, 3);
list($d3, $d4) = [30, 40];
check("destructure-list", $d3 + $d4, 70);
$s1 = 1;
$s2 = 2;
[$s1, $s2] = [$s2, $s1];
check("destructure-swap", $s1 === 2 && $s2 === 1, true);
$dst = [0, 0];
[$dst[0], $dst[1]] = [55, 66];
check("destructure-elements", $dst[0] + $dst[1], 121);
function makePair() { return [11, 22]; }
[$m1, $m2] = makePair();
check("destructure-call", $m1 + $m2, 33);

// ----- classes: constructor, $this, methods, chaining, dispatch -----
class Counter {
    public $value;
    public $step;
    public function __construct($start, $step) {
        $this->value = $start;
        $this->step = $step;
    }
    public function increment() {
        $this->value += $this->step;
        return $this->value;
    }
    public function doubled() {
        return $this->get() * 2;                 // method calling a method via $this
    }
    public function get() {
        return $this->value;
    }
}
$c = new Counter(10, 5);
check("class-ctor", $c->get(), 10);
check("class-method", $c->increment(), 15);
check("class-field-read", $c->value, 15);
check("class-this-method", $c->doubled(), 30);

class Point {
    public $x;
    public $y;
    public function __construct($x, $y) {
        $this->x = $x;
        $this->y = $y;
    }
    public function manhattan() {
        return abs($this->x) + abs($this->y);
    }
    public function plus($other) {
        return new Point($this->x + $other->x, $this->y + $other->y);
    }
}
$p = new Point(3, -4);
check("class-object-arg", $p->plus(new Point(1, 6))->x, 4);
check("class-chaining", $p->plus(new Point(2, 2))->manhattan(), 7);

class WhoA { public function who() { return "A"; } }
class WhoB { public function who() { return "B"; } }
function callWho($o) { return $o->who(); }
check("class-dispatch", callWho(new WhoA()) . callWho(new WhoB()), "AB");

// ----- exceptions: throw / catch / finally (incl. jumps out of finally) -----
function finOverridesReturn() {
    try {
        return "try";
    } finally {
        return "fin";
    }
}
check("return-in-finally-overrides", finOverridesReturn(), "fin");

function finBreaksLoop() {
    $r = "";
    for ($i = 0; $i < 3; $i++) {
        try {
            $r = $r . "a";
        } finally {
            if ($i == 1) { break; }
        }
    }
    return $r;
}
check("break-in-finally", finBreaksLoop(), "aa");

class Boom {
    public $code;
    public function __construct($c) {
        $this->code = $c;
    }
}
function risky($n) {
    if ($n > 3) { throw new Boom($n); }
    return $n * 2;
}

$exLog = "";
try {
    $exLog .= "t";
    throw new Boom(1);
} catch (Exception $e) {
    $exLog .= "c";
} finally {
    $exLog .= "f";
}
check("try-throw-catch-finally", $exLog, "tcf");

$noThrow = "";
try {
    $noThrow .= "t";
} catch (Exception $e) {
    $noThrow .= "!";
} finally {
    $noThrow .= "f";
}
check("try-no-throw", $noThrow, "tf");

$caught = -1;
try {
    risky(5);
    check("throw-unreachable", true, false);
} catch (Exception $e) {
    $caught = $e->code;                          // catch binds the thrown object
}
check("catch-binding", $caught, 5);
check("throw-not-taken", risky(2), 4);

$strCaught = "";
try {
    throw "plain";                               // any value can be thrown
} catch (Exception $e) {
    $strCaught = $e;
}
check("throw-string", $strCaught, "plain");

function classify($n) {
    try {
        if ($n > 0) { return $n * 10; }
        throw new Boom(0);
    } catch (Exception $e) {
        return -1;
    } finally {
    }
}
check("return-from-try", classify(4), 40);
check("return-from-catch", classify(-1), -1);

function nestedReturn() {
    try {
        try {
            return 9;
        } finally {
        }
    } finally {
    }
    return 0;
}
check("return-nested-tries", nestedReturn(), 9);

function loopBreak() {
    $sum = 0;
    for ($i = 0; $i < 9; $i++) {
        try {
            if ($i === 3) { break; }
            $sum += $i;
        } finally {
        }
    }
    return $sum;
}
check("break-out-of-try", loopBreak(), 3);

function loopContinue() {
    $sum = 0;
    for ($i = 0; $i < 5; $i++) {
        try {
            if ($i === 2) { continue; }
            $sum += $i;
        } catch (Exception $e) {
        }
    }
    return $sum;
}
check("continue-out-of-try", loopContinue(), 8);

$multi = 0;
try {
    throw new Boom(7);
} catch (Boom | Exception $e) {                  // multi-catch clause binds and runs
    $multi = $e->code;
}
check("multi-catch", $multi, 7);

$first = 0;
try {
    throw new Boom(11);
} catch (Boom $e) {                              // the FIRST catch clause wins
    $first = $e->code;
} catch (Exception $e) {
    $first = -99;
}
check("first-catch-wins", $first, 11);

$noVar = 0;
try {
    throw new Boom(3);
} catch (Exception) {                            // catch without a bound variable
    $noVar = 42;
}
check("catch-no-var", $noVar, 42);

function rethrower() {
    try {
        try {
            throw "deep";
        } catch (Exception $e) {
            throw $e . "er";                     // rethrow a derived value
        }
    } catch (Exception $e2) {
        return $e2;
    }
}
check("rethrow", rethrower(), "deeper");

$finRuns = 0;
function withFinally($doThrow) {
    global $finRuns;
    try {
        if ($doThrow) { throw new Boom(1); }
        return "ok";
    } catch (Exception $e) {
        return "caught";
    } finally {
        $finRuns = $finRuns + 1;                 // finally runs on both paths
    }
}
check("finally-both-paths", withFinally(false) . withFinally(true), "okcaught");
check("finally-run-count", $finRuns, 2);

// ----- everything combined in one small pipeline (3-element data flow) -----
function transform($list) {
    $out = "";
    foreach ($list as $n) {
        try {
            if ($n < 0) { throw new Boom($n); }
            $out .= ($n % 2 === 0 ? "e" : "o") . $n;
        } catch (Exception $e) {
            $out .= "x";
        }
    }
    return $out;
}
check("combined-pipeline", transform([1, 2, -3]), "o1e2x");

// ----- done -----
echo "features: " . $checks . " checks, " . $fails . " failures\n";
exit($fails);
