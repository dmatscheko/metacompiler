<?php
// PHP subset BIG test 3 -- Object-oriented & functional features.
//
// Theme: the language's signature constructs -- classes with $this, constructors,
// methods and fluent method chaining; a polymorphic expression tree evaluated by
// dynamic method dispatch; closures with use(); and higher-order array combinators
// (map / filter / reduce) built from those closures. The program counts failures
// in $fails and ends with exit($fails); the interpreter and the LLVM-IR compiler
// run the same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// Free function used by methods below (Euclid's algorithm).
function gcd($a, $b) {
    if ($a < 0) {
        $a = -$a;
    }
    if ($b < 0) {
        $b = -$b;
    }
    while ($b !== 0) {
        $t = $a % $b;
        $a = $b;
        $b = $t;
    }
    if ($a === 0) {
        return 1;
    }
    return $a;
}

// ----- a reduced-fraction value type -----
class Fraction {
    public $num;
    public $den;
    public function __construct($num, $den) {
        // normalise the sign onto the numerator and reduce to lowest terms
        if ($den < 0) {
            $num = -$num;
            $den = -$den;
        }
        $g = gcd($num, $den);
        $this->num = intdiv($num, $g);
        $this->den = intdiv($den, $g);
    }
    public function add($other) {
        return new Fraction(
            $this->num * $other->den + $other->num * $this->den,
            $this->den * $other->den
        );
    }
    public function sub($other) {
        return new Fraction(
            $this->num * $other->den - $other->num * $this->den,
            $this->den * $other->den
        );
    }
    public function mul($other) {
        return new Fraction($this->num * $other->num, $this->den * $other->den);
    }
    public function equals($other) {
        return $this->num === $other->num && $this->den === $other->den;
    }
    public function render() {
        return $this->num . "/" . $this->den;
    }
}

$half = new Fraction(1, 2);
$third = new Fraction(1, 3);
check("frac reduce", (new Fraction(4, 8))->render(), "1/2");
check("frac sign", (new Fraction(3, -6))->render(), "-1/2");
check("frac add", $half->add($third)->render(), "5/6");
check("frac sub", $half->sub($third)->render(), "1/6");
check("frac mul", $half->mul($third)->render(), "1/6");
check("frac equals", $half->equals(new Fraction(2, 4)), true);
check("frac not equals", $half->equals($third), false);

// 1/2 + 1/3 + 1/6 == 1
$oneSixth = new Fraction(1, 6);
$sum = $half->add($third)->add($oneSixth);
check("frac sum to one", $sum->render(), "1/1");
check("frac chain equals", $sum->equals(new Fraction(7, 7)), true);

// ----- a 2D vector with a fluent (chaining) interface -----
class Vec2 {
    public $x;
    public $y;
    public function __construct($x, $y) {
        $this->x = $x;
        $this->y = $y;
    }
    public function add($o) {
        return new Vec2($this->x + $o->x, $this->y + $o->y);
    }
    public function scale($k) {
        return new Vec2($this->x * $k, $this->y * $k);
    }
    public function dot($o) {
        return $this->x * $o->x + $this->y * $o->y;
    }
    public function normSquared() {
        return $this->dot($this);
    }
}

$v = new Vec2(1, 2);
$w = new Vec2(3, 4);
check("vec dot", $v->dot($w), 11);
$u = $v->add($w)->scale(2);
check("vec chain x", $u->x, 8);
check("vec chain y", $u->y, 12);
check("vec norm", (new Vec2(3, 4))->normSquared(), 25);
check("vec add then dot", $v->add($w)->dot(new Vec2(1, 1)), 10);

// ----- a polymorphic expression tree (composite pattern) -----
// Each node kind exposes eval(); dispatch on the stored object type does the work.
class NumNode {
    public $value;
    public function __construct($value) {
        $this->value = $value;
    }
    public function eval() {
        return $this->value;
    }
}

class AddNode {
    public $left;
    public $right;
    public function __construct($left, $right) {
        $this->left = $left;
        $this->right = $right;
    }
    public function eval() {
        return $this->left->eval() + $this->right->eval();
    }
}

class SubNode {
    public $left;
    public $right;
    public function __construct($left, $right) {
        $this->left = $left;
        $this->right = $right;
    }
    public function eval() {
        return $this->left->eval() - $this->right->eval();
    }
}

class MulNode {
    public $left;
    public $right;
    public function __construct($left, $right) {
        $this->left = $left;
        $this->right = $right;
    }
    public function eval() {
        return $this->left->eval() * $this->right->eval();
    }
}

class NegNode {
    public $child;
    public function __construct($child) {
        $this->child = $child;
    }
    public function eval() {
        return -$this->child->eval();
    }
}

// (2 + 3) * (10 - 4) = 30
$tree = new MulNode(
    new AddNode(new NumNode(2), new NumNode(3)),
    new SubNode(new NumNode(10), new NumNode(4))
);
check("ast mul", $tree->eval(), 30);

// -(7 * (1 + 1)) = -14
$tree2 = new NegNode(new MulNode(new NumNode(7), new AddNode(new NumNode(1), new NumNode(1))));
check("ast neg", $tree2->eval(), -14);

// A deeper tree: ((1+2)+(3+4)) + ((5+6)+(7+8)) = 36
$leftPair = new AddNode(new AddNode(new NumNode(1), new NumNode(2)), new AddNode(new NumNode(3), new NumNode(4)));
$rightPair = new AddNode(new AddNode(new NumNode(5), new NumNode(6)), new AddNode(new NumNode(7), new NumNode(8)));
$deep = new AddNode($leftPair, $rightPair);
check("ast deep", $deep->eval(), 36);

// ----- closures: capture with use(), and closure factories -----
$factor = 10;
$scale = function ($n) use ($factor) {
    return $n * $factor;
};
check("closure use", $scale(4), 40);

function makeAdder($base) {
    return function ($x) use ($base) {
        return $x + $base;
    };
}
$add100 = makeAdder(100);
$add5 = makeAdder(5);
check("adder 100", $add100(23), 123);
check("adder 5", $add5(23), 28);
// independent captures
check("adder still 100", $add100(0), 100);

function makeMultiplier($k) {
    return function ($x) use ($k) {
        return $x * $k;
    };
}
$triple = makeMultiplier(3);
check("triple", $triple(7), 21);

// ----- higher-order array combinators -----
function mapArr($f, $arr) {
    $out = [];
    foreach ($arr as $x) {
        $out[] = $f($x);
    }
    return $out;
}

function filterArr($f, $arr) {
    $out = [];
    foreach ($arr as $x) {
        if ($f($x)) {
            $out[] = $x;
        }
    }
    return $out;
}

function reduceArr($f, $init, $arr) {
    $acc = $init;
    foreach ($arr as $x) {
        $acc = $f($acc, $x);
    }
    return $acc;
}

function arrEq($a, $b) {
    if (count($a) !== count($b)) {
        return false;
    }
    $n = count($a);
    for ($i = 0; $i < $n; $i++) {
        if ($a[$i] !== $b[$i]) {
            return false;
        }
    }
    return true;
}

$nums = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

$squares = mapArr(function ($x) {
    return $x * $x;
}, $nums);
check("map squares", arrEq($squares, [1, 4, 9, 16, 25, 36, 49, 64, 81, 100]), true);

$evens = filterArr(function ($x) {
    return $x % 2 === 0;
}, $nums);
check("filter evens", arrEq($evens, [2, 4, 6, 8, 10]), true);

$total = reduceArr(function ($acc, $x) {
    return $acc + $x;
}, 0, $nums);
check("reduce sum", $total, 55);

$product = reduceArr(function ($acc, $x) {
    return $acc * $x;
}, 1, [1, 2, 3, 4, 5]);
check("reduce product", $product, 120);

// Compose the combinators: sum of squares of the even numbers.
$sumSqEven = reduceArr(
    function ($acc, $x) {
        return $acc + $x;
    },
    0,
    mapArr(
        function ($x) {
            return $x * $x;
        },
        filterArr(
            function ($x) {
                return $x % 2 === 0;
            },
            $nums
        )
    )
);
check("map/filter/reduce pipeline", $sumSqEven, 220);

// A closure using a captured combinator to build a running maximum.
$runningMax = reduceArr(
    function ($acc, $x) {
        return $x > $acc ? $x : $acc;
    },
    0,
    [3, 1, 4, 1, 5, 9, 2, 6]
);
check("reduce max", $runningMax, 9);

// ----- function composition -----
function compose($f, $g) {
    return function ($x) use ($f, $g) {
        return $f($g($x));
    };
}

$inc = function ($x) {
    return $x + 1;
};
$dbl = function ($x) {
    return $x * 2;
};
$incThenDouble = compose($dbl, $inc);
$doubleThenInc = compose($inc, $dbl);
check("compose 1", $incThenDouble(5), 12);
check("compose 2", $doubleThenInc(5), 11);

// Fold a list of functions into a pipeline.
function pipeline($fns, $x) {
    foreach ($fns as $f) {
        $x = $f($x);
    }
    return $x;
}
check("pipeline", pipeline([$inc, $dbl, $inc, $dbl], 1), 10);

// ----- an accumulator object combining OO state with closures -----
class Accumulator {
    public $total;
    public function __construct() {
        $this->total = 0;
    }
    // Apply a transform closure to $v, add the result to the running total,
    // and return $this so calls can be chained.
    public function absorb($f, $v) {
        $this->total += $f($v);
        return $this;
    }
    public function get() {
        return $this->total;
    }
}

$acc = new Accumulator();
$result = $acc
    ->absorb($inc, 1)
    ->absorb($dbl, 10)
    ->absorb(function ($x) {
        return $x * $x;
    }, 4)
    ->get();
// (1+1) + (10*2) + (4*4) = 2 + 20 + 16 = 38
check("accumulator chain", $result, 38);

// ----- done -----
if ($fails === 0) {
    echo "php-test-big-3 (OO & functional) passed\n";
}
exit($fails);
