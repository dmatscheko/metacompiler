<?php
// PHP subset self test.
// The program counts failed checks in $fails and ends with exit($fails), so the
// metacompiler run exits 0 exactly when every check passes. The interpreter
// (php-interpreter.abnf) and the LLVM-IR compiler (php-to-llvm-ir.abnf) run the
// same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- arithmetic and precedence -----
check("precedence", 1 + 2 * 3, 7);
check("parens", (1 + 2) * 3, 9);
check("division", 7 / 2, 3);
check("modulo", 7 % 3, 1);
check("unary minus", -5 + 2, -3);
check("nested", 2 * (3 + 4) - 1, 13);

// ----- comparisons -----
check("lt", 2 < 3, true);
check("gt", 5 > 4, true);
check("le", 3 <= 3, true);
check("ge", 3 >= 4, false);
check("loose eq", 3 == 3, true);
check("strict eq", 3 === 3, true);
check("neq", 3 != 4, true);
check("strict neq", 3 !== 4, true);
check("str eq", "abc" === "abc", true);

// ----- boolean logic (symbol and word forms) -----
check("and", true && true, true);
check("or", false || true, true);
check("and false", true && false, false);
check("not", !false, true);
check("word and", (2 < 3 and 4 < 5), true);
check("word or", (false or 1 < 2), true);

// ----- ternary -----
$big = 7 > 3 ? "yes" : "no";
check("ternary", $big, "yes");
$m = 4 > 9 ? 4 : 9;
check("ternary max", $m, 9);

// ----- strings: concatenation and interpolation -----
$who = "world";
check("concat", "a" . "b" . "c", "abc");
check("concat num", "n=" . 5, "n=5");
check("interp", "hello $who!", "hello world!");
$n = 42;
check("interp num", "n=$n", "n=42");
check("brace interp", "v={$n}", "v=42");
check("strlen", strlen("hello"), 5);

// ----- compound assignment -----
$x = 10;
$x += 5;
check("plus assign", $x, 15);
$x -= 3;
check("minus assign", $x, 12);
$x *= 2;
check("times assign", $x, 24);
$s = "a";
$s .= "b";
$s .= "c";
check("concat assign", $s, "abc");

// ----- increment / decrement -----
$i = 5;
$j = $i++;
check("post inc value", $j, 5);
check("post inc var", $i, 6);
$k = ++$i;
check("pre inc", $k, 7);
$i--;
check("post dec", $i, 6);

// ----- if / elseif / else -----
function classify($v) {
    if ($v < 0) {
        return "negative";
    } elseif ($v === 0) {
        return "zero";
    } else {
        return "positive";
    }
}
check("if neg", classify(-3), "negative");
check("if zero", classify(0), "zero");
check("if pos", classify(8), "positive");

// ----- while with break and continue -----
$sum = 0;
$i = 1;
while ($i <= 10) {
    $sum += $i;
    $i++;
}
check("while sum", $sum, 55);

$evens = 0;
$j = 0;
while ($j < 20) {
    $j++;
    if ($j % 2 === 1) {
        continue;
    }
    if ($j > 10) {
        break;
    }
    $evens += $j;
}
check("break continue", $evens, 30);

// ----- for loop -----
$fact = 1;
for ($f = 1; $f <= 5; $f++) {
    $fact *= $f;
}
check("for factorial", $fact, 120);

// ----- functions: recursion -----
function fib($n) {
    if ($n < 2) {
        return $n;
    }
    return fib($n - 1) + fib($n - 2);
}
check("recursion", fib(10), 55);

function add($a, $b) {
    return $a + $b;
}
check("call", add(20, 22), 42);

// ----- closures with use (real capture) -----
$factor = 3;
$scale = function ($v) use ($factor) {
    return $v * $factor;
};
check("closure", $scale(5), 15);

$base = 100;
$adder = function ($v) use ($base) {
    return $v + $base;
};
check("closure2", $adder(23), 123);

// ----- indexed arrays -----
$arr = [3, 1, 4, 1, 5];
check("index", $arr[2], 4);
check("count", count($arr), 5);
$arr[0] = 9;
check("set", $arr[0], 9);
$arr[] = 2;
check("append count", count($arr), 6);
check("append value", $arr[5], 2);
check("in_array hit", in_array(4, $arr), true);
check("in_array miss", in_array(7, $arr), false);
check("array_push", array_push($arr, 8), 7);
check("pushed value", $arr[6], 8);

// ----- foreach over indexed array -----
$total = 0;
foreach ([10, 20, 30] as $v) {
    $total += $v;
}
check("foreach values", $total, 60);

$idxsum = 0;
foreach ([5, 6, 7] as $ix => $v) {
    $idxsum += $ix;
}
check("foreach index", $idxsum, 3);

// ----- associative arrays (ordered maps) -----
$ages = ["alice" => 30, "bob" => 25, "carol" => 35];
check("map get", $ages["bob"], 25);
$ages["dave"] = 40;
check("map set", $ages["dave"], 40);
check("map count", count($ages), 4);
check("array_keys", count(array_keys($ages)), 4);
check("array_values", count(array_values($ages)), 4);

$agesum = 0;
foreach ($ages as $name => $age) {
    $agesum += $age;
}
check("foreach map", $agesum, 130);

$namelen = 0;
foreach (array_keys($ages) as $name) {
    $namelen += strlen($name);
}
check("keys iterate", $namelen, 17);

// ----- array() syntax -----
$viaFn = array(1, 2, 3);
check("array() fn", $viaFn[1], 2);
$viaMap = array("k" => 99);
check("array() map", $viaMap["k"], 99);

// ----- classes: $this, methods, constructor, new -----
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
    public function get() {
        return $this->value;
    }
}

$c = new Counter(10, 5);
check("ctor", $c->get(), 10);
check("method", $c->increment(), 15);
check("method again", $c->increment(), 20);
check("field read", $c->value, 20);

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
check("manhattan", $p->manhattan(), 7);
$q = $p->plus(new Point(1, 6));
check("point add x", $q->x, 4);
check("point add y", $q->y, 2);
check("point method chain", $p->plus(new Point(2, 2))->manhattan(), 7);

// ----- done -----
if ($fails === 0) {
    echo "PHP subset self test passed\n";
}
exit($fails);
