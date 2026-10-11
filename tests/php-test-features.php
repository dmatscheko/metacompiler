<?php
// Fast feature-matrix test for the PHP interpreter (php-interpreter.abnf) and the
// LLVM-IR compiler (php-to-llvm-ir.abnf). Every implemented construct is exercised
// with the SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
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

// ----- numbers, arithmetic, precedence -----
check("arith-precedence", 2 + 3 * 4, 14);
check("arith-paren", (2 + 3) * 4, 20);
check("arith-unary-minus", -3 + 5, 2);
check("arith-chain", 20 - 5 - 3, 12);
// / is float division in PHP; intdiv() truncates.
check("arith-div", 7 / 2, 3.5);
check("arith-div-int", intdiv(7, 2), 3);
check("arith-div-neg", -7 / 2, -3.5);
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
$ca /= 5;                     // 24 / 5 is 4.8, not 4
check("compound-div", $ca, 4.8);
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
// strlen counts BYTES: the e-acute is two of them in UTF-8, so 6.
check("str-unicode-len", strlen("héllo"), 6);
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

// A break that would leave a finally body is a compile error in PHP ("jump out of
// a finally block is disallowed", Zend/tests/try/try_finally_008); one that leaves
// a loop inside the finally body is an ordinary jump.
function finBreaksLoop() {
    $r = "";
    try {
        $r = $r . "a";
    } finally {
        for ($i = 0; $i < 3; $i++) {
            if ($i == 1) { break; }
            $r = $r . "a";
        }
    }
    return $r;
}
check("break-in-finally", finBreaksLoop(), "aa");

// Must extend Exception: real PHP 8 refuses to throw an object that does not
// implement Throwable ("Cannot throw objects that do not implement Throwable").
class Boom extends Exception {
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

// PHP can only throw Throwable objects - 'throw "plain";' is a fatal error, not
// a value throw. The message is what carries the payload.
$strCaught = "";
try {
    throw new Exception("plain");
} catch (Exception $e) {
    $strCaught = $e->getMessage();
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

// Again: only Throwable objects can be thrown, so the derived value rides in a
// new exception's message rather than being thrown as a bare string.
function rethrower() {
    try {
        try {
            throw new Exception("deep");
        } catch (Exception $e) {
            throw new Exception($e->getMessage() . "er");
        }
    } catch (Exception $e2) {
        return $e2->getMessage();
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

// ----- the recoverable name / method errors ---------------------------------
//
// PHP 8 raises a catchable \Error for each of these, and these lines prove the
// program CONTINUES. get_class + getMessage pin the class AND the exact text, which
// all three engines must agree on.
class MissTarget { public $p = 1; function ok() { return 7; } }
function missMsg($fn) {
    try { $fn(); return "no-throw"; }
    catch (Error $e) { return get_class($e) . "|" . $e->getMessage(); }
}
$missObj = new MissTarget();
$missStr = "ab";
$missInt = 7;
$missNull = null;
$missArr = [1, 2];
check("miss-method-instance", missMsg(function () use ($missObj) { $missObj->nope(); }),
      "Error|Call to undefined method MissTarget::nope()");
check("miss-method-string", missMsg(function () use ($missStr) { $missStr->nope(); }),
      "Error|Call to a member function nope() on string");
check("miss-method-int", missMsg(function () use ($missInt) { $missInt->nope(); }),
      "Error|Call to a member function nope() on int");
check("miss-method-null", missMsg(function () use ($missNull) { $missNull->nope(); }),
      "Error|Call to a member function nope() on null");
check("miss-method-array", missMsg(function () use ($missArr) { $missArr->nope(); }),
      "Error|Call to a member function nope() on array");
check("miss-function", missMsg(function () { zzz_no_such_function(); }),
      "Error|Call to undefined function zzz_no_such_function()");
check("miss-constant", missMsg(function () { echo ZZZ_NO_SUCH_CONSTANT; }),
      "Error|Undefined constant \"ZZZ_NO_SUCH_CONSTANT\"");
// An Error is NOT an Exception in PHP 8 - the two hierarchies only meet at
// Throwable - so a plain `catch (Exception)` must not see any of the above.
$missEsc = "not-caught";
try {
    try { $missObj->nope(); } catch (Exception $e) { $missEsc = "wrongly-caught"; }
} catch (Error $e) { $missEsc = "escaped-to-error"; }
check("miss-error-is-not-exception", $missEsc, "escaped-to-error");
// A generator, a closure stored in a PROPERTY (which PHP does not make callable
// as a method) and a string callable naming nothing. With `__call` declared
// anywhere in the module (MissMagic), the miss must still name the method asked for.
function missGen() { yield 1; }
class MissMagic { function __call($n, $a) { return "magic:" . $n; } }
class MissProp { public $f; }
$missG = missGen();
$missP = new MissProp();
$missP->f = function () { return 5; };
$missCallable = "zzz_no_such_callable";
check("miss-method-generator", missMsg(function () use ($missG) { $missG->nope(); }),
      "Error|Call to undefined method Generator::nope()");
check("miss-method-closure-prop", missMsg(function () use ($missP) { $missP->f(); }),
      "Error|Call to undefined method MissProp::f()");
check("miss-string-callable", missMsg(function () use ($missCallable) { $missCallable(); }),
      "Error|Call to undefined function zzz_no_such_callable()");
// __call still wins where the class actually declares one.
$missM = new MissMagic();
check("miss-magic-call-still-wins", $missM->anything(), "magic:anything");
// The method that IS declared still resolves, and a live class still wins over
// the miss path - the guard against "fixed the error, broke the dispatch".
check("miss-live-method", $missObj->ok(), 7);
check("miss-live-prop", $missObj->p, 1);

// ----- stdClass, var_export and JSON -----
// var_export is php_var_export_ex: floats at serialize_precision=-1 with a forced
// fraction, a nested array or object on its own line after "=> ", object
// properties three spaces in, objects rebuilt through \C::__set_state or
// (object) array(...). json_encode escapes '/' and non-ASCII (\u, lowercase hex,
// surrogate pairs) by default; json_decode answers null and records the error.
class JxP { public $a = 1; public $b = [1.5, "x'y"]; }
class JxS implements JsonSerializable {
    public $q = 3;
    public function jsonSerialize(): mixed { return ["q" => $this->q * 2]; }
}
enum JxPure { case A; }
enum JxBacked: string { case B = "b"; }
$jo = (object)["x" => 1, "5" => "five"];
$jn = new stdClass;
$jn->k = [2];
check("std-object-cast", [get_class($jo), $jo->x, (array)$jo], ["stdClass", 1, ["x" => 1, 5 => "five"]]);
check("std-new", [$jn instanceof stdClass, (array)(object)null, ((object)7)->scalar], [true, [], 7]);
$jv = [1.0, -0.0, 1e100, 0.1, PHP_INT_MIN, "a'b\\c", "n\0x", INF];
$jx = [];
foreach ($jv as $v) { $jx[] = var_export($v, true); }
check("vexport-scalars", implode(" ", $jx), "1.0 -0.0 1.0E+100 0.1 -9223372036854775807-1 'a\\'b\\\\c' 'n' . \"\\0\" . 'x' INF");
check("vexport-array", var_export([3 => "x", "k" => [true, null]], true),
      "array (\n  3 => 'x',\n  'k' => \n  array (\n    0 => true,\n    1 => NULL,\n  ),\n)");
check("vexport-object", var_export([new JxP()], true),
      "array (\n  0 => \n  \\JxP::__set_state(array(\n     'a' => 1,\n     'b' => \n    array (\n      0 => 1.5,\n      1 => 'x\\'y',\n    ),\n  )),\n)");
check("vexport-stdclass", var_export($jo, true), "(object) array(\n   'x' => 1,\n   '5' => 'five',\n)");
// An enum case exports as \E::Case, encodes as its value, and a pure one has none.
check("enum-json-export", [var_export([JxPure::A], true), json_encode([JxBacked::B]), json_encode(JxPure::A), json_last_error(),
      JxBacked::B instanceof BackedEnum, JxPure::A instanceof BackedEnum, JxPure::A instanceof UnitEnum],
      ["array (\n  0 => \n  \\JxPure::A,\n)", "[\"b\"]", false, 11, true, false, true]);
check("json-encode-scalars", json_encode([null, true, 0, -7, $jv[0], $jv[1], $jv[2], $jv[3], "é/€ 😀\"\x01\t"]),
      "[null,true,0,-7,1,-0,1.0e+100,0.1,\"\\u00e9\\/\\u20ac \\ud83d\\ude00\\\"\\u0001\\t\"]");
check("json-encode-objects", json_encode(["a" => 1, "b" => [], "c" => new stdClass(), 1 => [1, 2]]) . json_encode(new JxP()) . json_encode(new JxS()),
      "{\"a\":1,\"b\":[],\"c\":{},\"1\":[1,2]}{\"a\":1,\"b\":[1.5,\"x'y\"]}{\"q\":6}");
check("json-encode-flags", json_encode(["é/<'", [1.0]], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_HEX_TAG
      | JSON_HEX_APOS | JSON_PRESERVE_ZERO_FRACTION) . json_encode([1, 2], JSON_FORCE_OBJECT)
      . json_encode(["12", "1.5", "x"], JSON_NUMERIC_CHECK),
      "[\"é/\\u003C\\u0027\",[1.0]]{\"0\":1,\"1\":2}[12,1.5,\"x\"]");
check("json-encode-pretty", json_encode(["a" => [1, []]], JSON_PRETTY_PRINT), "{\n    \"a\": [\n        1,\n        []\n    ]\n}");
$je = [json_encode($jv[7]), json_last_error(), json_encode("a\xC8"), json_last_error_msg(),
       json_encode([[1]], 0, 1), json_last_error(), json_encode([1, $jv[7]], JSON_PARTIAL_OUTPUT_ON_ERROR)];
try { json_encode($jv[7], JSON_THROW_ON_ERROR); } catch (JsonException $e) { $je[] = $e->getMessage() . "|" . $e->getCode(); }
$je[] = json_last_error();
check("json-encode-errors", $je, [false, 7, false, "Malformed UTF-8 characters, possibly incorrectly encoded", false, 1, "[1,0]",
      "Inf and NaN cannot be JSON encoded|7", 7]);
$jd = json_decode('{"a":1,"b":[1,2.5,"x"],"c":{"d":null},"0":true}');
check("json-decode-object", [$jd instanceof stdClass, $jd->a, $jd->b, $jd->c->d, $jd->{"0"}, json_last_error()],
      [true, 1, [1, 2.5, "x"], null, true, 0]);
check("json-decode-assoc", json_decode('{"a":{"b":[1]},"5":2}', true), ["a" => ["b" => [1]], 5 => 2]);
$jdocs = [" 12 ", "-0", "1E2", "\"\\u00e9\\ud83d\\ude00\\n\"", "9223372036854775807", "9223372036854775808"];
$jr = [];
foreach ($jdocs as $d) { $jr[] = json_decode($d); }
$jr[] = json_decode("12345678901234567890", false, 512, JSON_BIGINT_AS_STRING);
check("json-decode-scalars", $jr, [12, 0, 100.0, "é😀\n", PHP_INT_MAX, 9.2233720368547758E+18, "12345678901234567890"]);
$jcodes = [];
foreach (["[1,2", "", "01", "[1,]", "TRUE", "\"\\ud800\"", "\"a\x01\"", "\"abc", "[[1]]"] as $d) {
    $jcodes[] = json_decode($d, false, 1) === null ? json_last_error() : -1;
}
try { json_decode("{", false, 512, JSON_THROW_ON_ERROR); } catch (JsonException $e) { $jcodes[] = $e->getMessage(); }
$jcodes[] = json_last_error_msg();
check("json-decode-errors", $jcodes, [4, 4, 4, 4, 4, 10, 3, 3, 1, "Syntax error", "Maximum stack depth exceeded"]);

// ----- visibility, foreach over an object, method_exists -----
// A non-public property is mangled by (array) ("\0*\0p", "\0Class\0q"), left out
// of json_encode, printed with :protected / :"Class":private by var_dump and
// iterated by foreach only where the class scope can read it; slots are the
// class's own properties in body order (a promoted one where the constructor
// stands), then a trait's. method_exists
// is case-insensitive and ignores visibility. A property initializer is a
// constant expression in which self:: names the class (tests/bug66015.phpt).
trait VisT { private $tp = "t"; }
class VisA {
    const K = 2;
    public $x = 1;
    protected $p = 3;
    private $q = 4;
    public static $st = [self::K => "two"];
    public $in = [self::K];
    public function __construct(protected $pr = 5, public $pu = 6) {}
    function keysHere() { $o = []; foreach ($this as $k => $v) { $o[] = $k; } return implode(",", $o); }
    private function hidden() {}
}
class VisB extends VisA {
    use VisT;
    function keysHere() { $o = []; foreach ($this as $k => $v) { $o[] = $k; } return implode(",", $o); }
}
$va = new VisA();
$vb = new VisB();
check("vis-array-cast", array_keys((array)$va), ["x", "\0*\0p", "\0VisA\0q", "in", "\0*\0pr", "pu"]);
check("vis-json", json_encode($vb), '{"x":1,"in":[2],"pu":6}');
$vo = [];
foreach ($vb as $k => $v) { $vo[] = $k; }
check("vis-foreach", [implode(",", $vo), $va->keysHere(), $vb->keysHere()], ["x,in,pu", "x,p,q,in,pr,pu", "x,p,in,pr,pu,tp"]);
var_dump($vb);
$fo = new stdClass;
$fo->a = 1;
$fo->b = 2;
$fs = 0;
foreach ($fo as $k => $v) { $fs += $v; }
foreach ($fo as $k => &$v) { $v = $v * 10; }
unset($v);
check("foreach-object", [$fs, $fo->a, $fo->b], [3, 10, 20]);
check("method-exists", [method_exists($va, "keysHere"), method_exists($va, "KEYSHERE"), method_exists("VisB", "hidden"),
      method_exists($vb, "__construct"), method_exists($va, "x"), method_exists("NoSuchClass", "f"),
      method_exists(new stdClass, "__construct"), method_exists(new Exception("e"), "getMessage")],
      [true, true, true, true, false, false, false, true]);
check("self-in-initializer", [VisA::$st, $va->in], [[2 => "two"], [2]]);

// foreach over an Iterator calls rewind, then valid / current / key (only when
// the loop binds one) / body / next; an IteratorAggregate hands its iterator over.
class FeIt implements Iterator {
    private $i = 0;
    public $log = "";
    public function __construct(private array $a) {}
    public function rewind(): void { $this->log .= "r"; $this->i = 0; }
    public function valid(): bool { $this->log .= "v"; return $this->i < count($this->a); }
    public function current(): mixed { $this->log .= "c"; return $this->a[$this->i]; }
    public function key(): mixed { $this->log .= "k"; return "k" . $this->i; }
    public function next(): void { $this->log .= "n"; $this->i++; }
}
class FeAgg implements IteratorAggregate {
    public function getIterator(): Iterator { return new FeIt([7, 8]); }
}
class FeGenAgg implements IteratorAggregate {
    public function getIterator(): Generator { yield "g" => 9; }
}
$fi = new FeIt([1, 2, 3]);
$fr = "";
foreach ($fi as $k => $v) { $fr .= "$k=$v "; }
$fl = $fi->log;
$fi->log = "";
foreach ($fi as $v) { if ($v == 2) { continue; } $fr .= "$v "; if ($v == 3) { break; } }
foreach (new FeAgg() as $k => $v) { $fr .= "$k:$v "; }
foreach (new FeGenAgg() as $k => $v) { $fr .= "$k:$v"; }
check("foreach-iterator", [$fr, $fl, $fi->log], ["k0=1 k1=2 k2=3 1 3 k0:7 k1:8 g:9", "rvcknvcknvcknv", "rvcnvcnvc"]);

// foreach by value walks the array as it was when the loop started; by
// reference it walks the live one (tests/foreach/bug68215.phpt).
$fsa = ["x" => 1, "y" => 2];
$fsi = 1;
foreach ($fsa as $k => $v) { $fsa[$fsi] = $v; unset($fsa[$k]); $fsi++; }
$fsb = [1, 2];
$fsn = "";
foreach ($fsb as $v) { $fsb[] = $v; $fsb[1] = 9; $fsn .= $v; }
check("foreach-snapshot", [$fsa, $fsn, count($fsb)], [[1 => 1, 2 => 2], "12", 4]);

// A property whose name starts "__" is an ordinary property: json_decode's
// "__class" does not replace the object's class, and unset() removes one.
$jdu = json_decode('{"__class":"C","__refcell":2,"a":1}');
$jdu->__x = 5;
$jdk = "__dict";
$jdu->$jdk = true;
unset($jdu->a);
check("dunder-props", [get_class($jdu), $jdu->__class, $jdu->__refcell + $jdu->__x, array_keys((array)$jdu), json_encode($jdu)],
      ["stdClass", "C", 7, ["__class", "__refcell", "__x", "__dict"], '{"__class":"C","__refcell":2,"__x":5,"__dict":true}']);

// An enum case's value may read self:: (Zend/tests/enum/gh7821.phpt).
interface EnSelfI { public const A = "a"; }
enum EnSelf: string implements EnSelfI {
    const B = "b";
    case C = self::B;
    case D = EnSelfI::A;
}
check("enum-case-self", [EnSelf::C->value, EnSelf::D->value], ["b", "a"]);

// ----- done -----
echo "features: " . $checks . " checks, " . $fails . " failures\n";
exit($fails);
