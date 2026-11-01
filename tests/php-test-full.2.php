<?php
// Full-syntax test: PHP (PHP 8.3 core grammar) - part 2 of 2, sections 64 on.
//
// THE PHP RATCHET IS TWO FILES, so that no -frozen interpreter leg comes near the
// IR interpreter's step brake (-max-steps, 100M per call):
//
//   tests/php-test-full.php     sections 01 to 63
//   tests/php-test-full.2.php   sections 64 on, and every newer section
//
// The section NUMBERS are global, and ./test.sh --full reports the files as the
// units php and php2. The header of tests/php-test-full.php holds the
// conventions, the scope and the sources.


$failures = 0;
$checks = 0;

function check($id, $cond) {
    global $failures;
    global $checks;
    $checks = $checks + 1;
    if (!$cond) {
        echo "FAIL " . $id . "\n";
        $failures = $failures + 1;
    }
}

// ===== SECTION 64: method visibility, typed-property coercion, private slots =====
// A private method is callable only from its declaring class, a protected one from
// any class related to its root - the first declaration up the chain that is not
// private (zend_std_get_method, zend_check_protected). Code in a class reaches its
// own private method even on an instance of a subclass that declares the name
// again. Out of reach, __call / __callStatic stand in, else the call is an Error;
// a callback is checked from the scope that hands it over, and a constructor out
// of reach makes `new` an Error. A typed property takes a value of one of its
// types; any other scalar (or a __toString object, for string) is coerced to the
// first of int, float, string and bool the type has that accepts it, a numeric
// string choosing int or float by its own form, else the write is a TypeError and
// leaves the property as it was (zend_verify_property_type, union types RFC). A
// typed static property without a default starts uninitialised. A base class's
// private property is invisible from outside it, so a write there makes a
// separate dynamic property of the same name.
class S64A {
    private function f() { return "A::f"; }
    protected function p() { return "A::p"; }
    function test() { return $this->f() . " " . $this->p(); }
    function viaMap() { return implode(",", array_map([$this, 'f'], [1])); }
    function fcc() { return $this->f(...); }
    private static function ps() { return "A::ps"; }
    static function callPs() { return static::ps(); }
    static function viaStatic() { return implode(",", array_map([self::class, 'ps'], [1])); }
}
class S64B extends S64A {
    public function f() { return "B::f"; }
    protected function p() { return "B::p"; }
}
class S64Sib extends S64A { function poke(S64A $o) { return $o->p(); } }
class S64M {
    private function hid() { return "hid"; }
    function __call($n, $a) { return "call:" . $n . count($a); }
    private static function shid() { return "shid"; }
    static function __callStatic($n, $a) { return "static:" . $n; }
}
trait S64T { function tm() { return "tm"; } }
class S64U { use S64T { tm as protected; } function viaT() { return $this->tm(); } }
class S64Ctor {
    protected function __construct() {}
    static function make() { return new static(); }
}
class S64P {
    public int $i;
    public ?float $f;
    public string|int $si;
    public int|float $n;
    public ?S64P $self;
    public static int $st;
    public static ?string $ss = null;
}
class S64Pro {
    function __construct(public int $n, protected ?string $s = null) {}
    function setS($v) { $this->s = $v; }
}
class S64Self extends S64P { public self $me; public static ?parent $sp = null; }
trait S64ST { public self $ts; public static int $tsi = 0; public static ?self $tss = null; }
class S64SU { use S64ST; }
class S64Priv {
    private $x = 1;
    function getX() { return $this->x; }
}
class S64Kid extends S64Priv {}
function s64err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s64() {
    $b = new S64B;
    $g = $b->fcc();
    check("mvis1", $b->test() === "A::f B::p" && $b->f() === "B::f" && $b->viaMap() === "A::f" && $g() === "A::f"
        && (new S64Sib)->poke($b) === "B::p" && S64B::callPs() === "A::ps" && S64A::viaStatic() === "A::ps");
    check("mvis2", s64err(fn() => $b->p()) === "Error: Call to protected method S64B::p() from global scope"
        && s64err(fn() => S64A::ps()) === "Error: Call to private method S64A::ps() from global scope"
        && s64err(fn() => (new S64U)->tm()) === "Error: Call to protected method S64U::tm() from global scope"
        && (new S64U)->viaT() === "tm");
    check("mvis3", (new S64M)->hid(1, 2) === "call:hid2" && S64M::shid() === "static:shid"
        && s64err(fn() => new S64Ctor) === "Error: Call to protected S64Ctor::__construct() from global scope"
        && S64Ctor::make() instanceof S64Ctor);
    check("mvis4", s64err(fn() => $b->p(...)) === "Error: Call to protected method S64B::p() from global scope"
        && s64err(fn() => call_user_func([$b, 'p']))
            === "TypeError: call_user_func(): Argument #1 (\$callback) must be a valid callback, cannot access protected method S64B::p()"
        && s64err(fn() => array_map([$b, 'p'], [1]))
            === "TypeError: array_map(): Argument #1 (\$callback) must be a valid callback or null, cannot access protected method S64B::p()");
    $o = new S64P;
    $o->i = "42";
    $o->f = 3;
    $o->si = 1e100;
    $o->n = "1e3";
    check("tpc1", $o->i === 42 && $o->f === 3.0 && $o->si === "1.0E+100" && $o->n === 1000.0);
    $p = new S64P;
    check("tpc2", s64err(function () use ($p) { $p->i = "4x"; }) === "TypeError: Cannot assign string to property S64P::\$i of type int"
        && s64err(fn() => $p->i) === "Error: Typed property S64P::\$i must not be accessed before initialization"
        && s64err(function () use ($p) { $p->self = 5; }) === "TypeError: Cannot assign int to property S64P::\$self of type ?S64P"
        && s64err(function () use ($p) { $p->si = []; }) === "TypeError: Cannot assign array to property S64P::\$si of type string|int");
    $pro = new S64Pro(1);
    $pro->n = "5";
    check("tpc3", s64err(function () use ($pro) { $pro->n = "z"; }) === "TypeError: Cannot assign string to property S64Pro::\$n of type int"
        && $pro->n === 5
        && s64err(fn() => $pro->setS([])) === "TypeError: Cannot assign array to property S64Pro::\$s of type ?string"
        && s64err(function () { (new S64Self)->me = 1; }) === "TypeError: Cannot assign int to property S64Self::\$me of type S64Self"
        && s64err(function () { S64Self::$sp = 1; }) === "TypeError: Cannot assign int to property S64Self::\$sp of type ?S64P"
        && s64err(function () { (new S64SU)->ts = 1; }) === "TypeError: Cannot assign int to property S64SU::\$ts of type self");
    S64SU::$tsi = "8";
    check("tst3", S64SU::$tsi === 8
        && s64err(function () { S64SU::$tsi = "x"; }) === "TypeError: Cannot assign string to property S64SU::\$tsi of type int"
        && s64err(function () { S64SU::$tss = 1; }) === "TypeError: Cannot assign int to property S64SU::\$tss of type ?self");
    $r = "9";
    $q = new S64P;
    $q->i =& $r;
    check("tpr1", $q->i === 9 && $r === 9);
    check("tst1", s64err(fn() => S64P::$st) === "Error: Typed static property S64P::\$st must not be accessed before initialization"
        && s64err(function () { S64P::$st = "x"; }) === "TypeError: Cannot assign string to property S64P::\$st of type int"
        && !isset(S64P::$st));
    S64P::$st = "7";
    S64P::$ss = 5;
    check("tst2", S64P::$st === 7 && S64P::$ss === "5");
    $k = new S64Kid;
    $k->x = 2;
    check("pslot1", $k->getX() === 1 && $k->x === 2 && count((array) $k) === 2);
    unset($k->x);
    check("pslot2", $k->getX() === 1 && !isset($k->x));
}

// ===== SECTION 65: static calls of instance methods, callables made where they stand, is_callable =====
// A '::' call of a non-static method runs on $this, which must be an instance of
// the class named, else it is an Error naming the declaring class and the method
// as declared (zend_non_static_method_call); a callback naming one through its
// class needs the same object, from the frame that hands it over
// (zend_is_callable_check_func). 'm(...)' resolves its method where it stands: a
// missing one is the Error the call would raise. is_callable() answers whether
// that callback check passes in the calling frame.
trait S65T { function tm() { return "tm"; } }
class S65A {
    use S65T;
    function Foo() { return "foo:" . get_class($this); }
    static function sfoo() { return "sfoo"; }
    private function priv() { return "priv"; }
    function inside() {
        return [is_callable([$this, 'priv']), is_callable(['S65A', 'foo']), is_callable('S65A::foo'), is_callable('nope')];
    }
}
class S65B extends S65A {
    function viaA() { return S65A::foo(); }
    function viaFcc() { $f = parent::foo(...); return $f(); }
}
class S65C { function other() { return S65A::foo(); } }
class S65M { static function __callStatic($n, $a) { return "static:" . $n; } function __call($n, $a) { return "call:" . $n; } }
function s65err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s65() {
    $ns = "Error: Non-static method S65A::Foo() cannot be called statically";
    check("nsc1", s65err(fn() => S65A::foo()) === $ns && s65err(fn() => S65A::FOO()) === $ns
        && s65err(fn() => (new S65C)->other()) === $ns && s65err(fn() => S65A::foo(...)) === $ns
        && s65err(fn() => S65B::tm()) === "Error: Non-static method S65A::tm() cannot be called statically"
        && S65A::sfoo() === "sfoo" && S65A::sfoo(...)() === "sfoo");
    check("nsc2", s65err(fn() => call_user_func(['S65A', 'foo']))
            === "TypeError: call_user_func(): Argument #1 (\$callback) must be a valid callback, non-static method S65A::Foo() cannot be called statically"
        && s65err(fn() => Closure::fromCallable('S65A::foo'))
            === "TypeError: Failed to create closure from callable: non-static method S65A::Foo() cannot be called statically");
    $b = new S65B;
    check("nsc3", $b->viaA() === "foo:S65B" && $b->viaFcc() === "foo:S65B");
    $o = new S65A;
    $n = null;
    check("fcc1", s65err(fn() => $o->nope(...)) === "Error: Call to undefined method S65A::nope()"
        && s65err(fn() => [$o, 'nope'](...)) === "Error: Call to undefined method S65A::nope()"
        && s65err(fn() => 'S65A::nope'(...)) === "Error: Call to undefined method S65A::nope()"
        && s65err(fn() => $n->m(...)) === "Error: Call to a member function m() on null"
        && s65err(fn() => (new stdClass)->nope(...)) === "Error: Call to undefined method stdClass::nope()");
    $m = new S65M;
    check("fcc2", [$o, 'foo'](...)() === "foo:S65A" && 'strtoupper'(...)('x') === "X" && 'S65A::sfoo'(...)() === "sfoo"
        && S65M::zz(...)() === "static:zz" && $m->zz(...) instanceof Closure);
    check("isc1", [is_callable('S65A::foo'), is_callable('S65A::sfoo'), is_callable([$o, 'foo']), is_callable('strlen'),
                   is_callable('\strlen'), is_callable('S65A'), is_callable('nope'), is_callable(fn() => 1),
                   is_callable(42), is_callable(['S65A', 'nope']), is_callable([$o, 'priv']), is_callable([$m, 'any'])]
        === [false, true, true, true, true, false, false, true, false, false, false, true]);
    check("isc2", $o->inside() === [true, true, true, false]);
}

// ===== SECTION 66: callbacks and dynamic calls of instance methods, static closures, Closure's methods =====
// A callback naming a non-static method through its class binds the $this of the
// frame it is handed over in, when that frame's class is the method's or a
// subclass (zend_is_callable_check_class), and is a TypeError anywhere else. A
// DYNAMIC call ('C::m'(), ['C', 'm']()) binds none, so it is always an Error, even
// when __call would take the name for the frame's $this
// (zend_init_dynamic_call_array). [$o, 'm'] and $o->m(...) reach __call. A static
// closure has no $this. bindTo / call / Closure::bind replace a closure's $this and
// class scope (zend_create_closure_ex); no instance binds to a static closure.
class S66A {
    function foo() { return "foo:" . get_class($this); }
    static function sfoo() { return "sfoo"; }
    function stat() {
        $f = static function() { return isset($this); };
        $g = static fn() => isset($this);
        $h = function() { return isset($this); };
        $k = fn() => isset($this);
        return [$f(), $g(), $h(), $k(), s66err(static fn() => S66A::foo()), s66err(fn() => S66A::foo())];
    }
}
class S66B extends S66A {
    function twice($x) { return $x * 2 . ":" . get_class($this); }
    function cbs() {
        return [call_user_func(['S66A', 'foo']), call_user_func('S66A::foo'), array_map(['S66B', 'twice'], [1, 2]),
                call_user_func_array(['S66A', 'foo'], []), Closure::fromCallable(['S66A', 'foo'])()];
    }
    function dyn() {
        try { ['S66A', 'foo'](); } catch (Error $e) { return $e->getMessage(); }
        return "none";
    }
}
class S66M {
    function __call($n, $a) { return "call:" . $n . ":" . implode(",", $a); }
    function dyn() {
        try { ['S66M', 'bar'](); } catch (Error $e) { return $e->getMessage(); }
        return "none";
    }
}
class S66P {
    private $v = 7;
    public $n = 0;
    function getter() { return function() { return $this->v; }; }
    function counter() { return function() { $this->n += 2; return ++$this->n; }; }
}
class S66Q { private $v = 9; }
function s66err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s66() {
    $b = new S66B;
    check("cbn1", $b->cbs() === ["foo:S66B", "foo:S66B", ["2:S66B", "4:S66B"], "foo:S66B", "foo:S66B"]
        && call_user_func(['S66A', 'sfoo']) === "sfoo");
    check("cbn2", s66err(fn() => array_map(['S66A', 'foo'], [1]))
            === "TypeError: array_map(): Argument #1 (\$callback) must be a valid callback or null, non-static method S66A::foo() cannot be called statically"
        && s66err(fn() => usort($b, 'S66A::foo'))
            === "TypeError: usort(): Argument #1 (\$array) must be of type array, S66B given");
    $ns = "Error: Non-static method S66A::foo() cannot be called statically";
    $fs = 'S66A::foo';
    check("dyn1", s66err(fn() => ['S66A', 'foo']()) === $ns && s66err(fn() => $fs()) === $ns
        && $b->dyn() === "Non-static method S66A::foo() cannot be called statically"
        && ['S66A', 'sfoo']() === "sfoo" && 'S66A::sfoo'() === "sfoo");
    $m = new S66M;
    check("dyn2", $m->dyn() === "Non-static method S66M::bar() cannot be called statically"
        && s66err(fn() => ['S66M', 'bar']()) === "Error: Call to undefined method S66M::bar()");
    $s = 'w';
    check("mcl1", [$m, 'x'](1, 2) === "call:x:1,2" && $m->y(...)(3) === "call:y:3" && $m->$s(4) === "call:w:4"
        && call_user_func([$m, 'z']) === "call:z:");
    check("stc1", (new S66A)->stat() === [false, false, true, true, $ns, "foo:S66A"]);
    $get = function() { return $this->v; };
    $add = function($a, $b) { return $a + $b; };
    $cls = function() { return static::class; };
    $u = 3;
    $uf = function() use ($u) { return $u; };
    $bq = $get->bindTo(new S66Q, 'S66Q');
    check("clo1", Closure::bind($get, new S66P, S66P::class)() === 7 && $bq() === 9 && $bq !== $get
        && $get->call(new S66Q) === 9 && (new S66P)->getter()->bindTo(new S66Q, 'S66Q')() === 9
        && (new S66P)->getter()->bindTo(new S66P)() === 7 && $uf->bindTo(null, null)() === 3);
    check("clo2", $add->__invoke(2, 3) === 5 && $add->call(new S66P, 4, 5) === 9
        && Closure::bind($cls, null, 'S66Q')() === "S66Q" && $cls->call(new S66P) === "S66P"
        && (static function() { return 1; })->bindTo(new S66P) === null);
    check("clo3", s66err(fn() => (new S66P)->getter()->bindTo(new S66Q)()) === "Error: Cannot access private property S66Q::\$v"
        && s66err(fn() => $add->nope()) === "Error: Call to undefined method Closure::nope()");
    $cnt = (new S66P)->counter();
    check("clo4", $cnt() === 3 && $cnt() === 6);
}

// ===== SECTION 67: $this outside an object, the called scope of '::', __call for '::' =====
// Only an instance method's body is sure to have a $this (this_guaranteed_exists);
// anywhere else - a static method, a function, a closure without one - every use
// of it but isset($this) / empty($this) fetches it (ZEND_FETCH_THIS), and the
// fetch is Error "Using $this when not in object context". A static method never
// binds a $this, whatever receiver the call went through. A '::' call naming its
// class runs a static method in that class, self:: / parent:: / static::
// forward the caller's called scope (ZEND_INIT_STATIC_METHOD_CALL), and a
// callback naming a class binds the frame's $this when the frame's class is an
// instance of it and a subclass of the named one, whose class is then the called
// scope even of a static method (zend_is_callable_check_class). A '::' call of a
// method the class lacks or the scope may not call goes to the __call of a
// $this that is an instance of the class, else to __callStatic
// (get_static_method_fallback); a dynamic call cannot take __call, so it is an
// Error there (zend_init_dynamic_call_array). A method name is case-insensitive,
// a by-reference parameter included, and $c->__invoke($v) is the call $c($v).
// $o->$m(...) and C::$m(...) are first-class callables like the literal forms.
class S67A {
    public $x = 1;
    static function who() { return static::class; }
    static function s() { return static::who() . self::who(); }
    static function run() { return (new S67B)->s(); }
    static function rd() { return $this->x; }
    static function bare() { return $this; }
    static function call() { return $this->who(); }
    static function wr() { $this->x = 5; return 1; }
    static function prop() { return isset($this->x); }
    static function probe() { return [isset($this), empty($this)]; }
    function closures() {
        return [s67err(static function() { return $this->x; }), s67err(static fn() => $this ?? 0),
                (static fn() => isset($this))()];
    }
    function calls() {
        return [S67A::who(), self::who(), static::who(), $this->who(), call_user_func(['S67A', 'who']),
                call_user_func('S67A::who'), self::s(), S67A::s()];
    }
}
class S67B extends S67A {}
class S67C { function j() { return [call_user_func(['S67A', 'who']), S67A::who()]; } }
function s67fn() { return $this; }
function s67err($f) {
    try { return $f(); } catch (Error $e) { return get_class($e) . ": " . $e->getMessage(); }
}
class S67H { function __invoke(&$x) { $x .= "!"; } function Tail(&$x) { $x .= "?"; } }
class S67M {
    private static function sp($x) { return "sp$x"; }
    function __call($n, $a) { return "call:$n:" . implode(",", $a); }
    static function __callStatic($n, $a) { return "cs:$n:" . implode(",", $a); }
}
class S67N extends S67M {
    function t() {
        $s = "sp";
        return [S67M::sp(1), S67M::$s(2), parent::sp(3), static::sp(4), S67M::nope(5), (S67M::sp(...))(6)];
    }
    function dyn() {
        return [s67err(fn() => ['S67M', 'sp'](1)), s67err(fn() => 'S67M::nope'(2))];
    }
    static function st() { return [S67M::sp(1), parent::nope(2), ['S67M', 'sp'](3)]; }
}
class S67X { function t() { return [S67M::sp(1), 'S67M::sp'(2), S67M::nope(3)]; } }
class S67K { function __call($n, $a) { return "K:" . $n; } }
class S67L extends S67K { function t() { return [S67K::miss(), parent::miss2(), self::miss3()]; } }
function s67() {
    $nt = "Error: Using \$this when not in object context";
    check("ths1", s67err(fn() => S67A::rd()) === $nt && s67err(fn() => S67A::bare()) === $nt
        && s67err(fn() => S67A::call()) === $nt && s67err(fn() => S67A::wr()) === $nt
        && s67err(fn() => S67A::prop()) === $nt && s67err('s67fn') === $nt
        && s67err(function() { return $this; }) === $nt && s67err(fn() => (new S67B)->rd()) === $nt);
    check("ths2", S67A::probe() === [false, true] && (new S67A)->closures() === [$nt, $nt, false]);
    check("lsb1", (new S67B)->calls() === ["S67A", "S67B", "S67B", "S67B", "S67B", "S67B", "S67BS67B", "S67AS67A"]
        && (new S67B)->s() === "S67BS67B" && S67A::run() === "S67BS67B" && S67B::s() === "S67BS67B"
        && (new S67C)->j() === ["S67A", "S67A"]);
    $inc = function(&$n) { $n++; };
    $n = 1;
    $inc->__INVOKE($n);
    $inc->__invoke($n);
    $h = new S67H;
    $s = "a";
    $h($s);
    $h->__invoke($s);
    $h->tail($s);
    check("inv1", $n === 3 && $s === "a!!?");
    check("cfb1", (new S67N)->t() === ["call:sp:1", "call:sp:2", "call:sp:3", "call:sp:4", "call:nope:5", "call:sp:6"]
        && S67N::st() === ["cs:sp:1", "cs:nope:2", "cs:sp:3"] && (new S67X)->t() === ["cs:sp:1", "cs:sp:2", "cs:nope:3"]
        && (new S67L)->t() === ["K:miss", "K:miss2", "K:miss3"]);
    check("cfb2", (new S67N)->dyn() === ["Error: Non-static method S67M::sp() cannot be called statically",
                                        "Error: Non-static method S67M::nope() cannot be called statically"]);
    $o = new S67M;
    $m = "sp";
    $w = "who";
    check("fcc1", ($o->$m(...))(7) === "call:sp:7" && (S67M::$m(...))(8) === "cs:sp:8" && (S67B::$w(...))() === "S67B");
}

// ===== SECTION 68: rebinding closures made from callables, and closures using $this =====
// A closure made from a callable ($o->m(...), C::m(...), f(...),
// Closure::fromCallable) keeps the method's class as its scope: bindTo / call give
// it an instance of that class unless the method is static, none unless it is,
// and no other scope; one made from a function takes no scope at all. A closure
// whose own body uses $this (isset($this) included, a nested function's use not)
// cannot lose the $this it has. A refused binding answers null with a warning
// (zend_valid_closure_binding); the new closure runs the same method, called in
// the new $this's class, or in the method's class for a static one
// (do_closure_bind). A __call or __callStatic standing in for a method rebinds
// like one (Zend/tests/closures/closure_call_bind.phpt).
class S68P {
    public $v = 1;
    function m() { return "P::m" . $this->v . static::class; }
    function inc(&$x) { $x += $this->v; }
    static function sm() { return "sm" . static::class; }
    function arrowUse() { return fn() => $this->v; }
    function nested() { return function() { return fn() => 2; }; }
    function nestedThis() { return function() { return function() { return $this; }; }; }
    function viaIsset() { return function() { return isset($this); }; }
    function plainUse() { return function() { return 5; }; }
    static function __callStatic($n, $a) { return "cs:" . $n . ":" . static::class; }
}
class S68Q extends S68P { public $v = 2; function m() { return "Q::m"; } }
class S68R { public $v = 9; }
class S68F { function __call($n, $a) { return "call:" . $n . ":" . get_class($this); } }
class S68G extends S68F {}
function s68plain() { return "plain"; }
function s68() {
    $p = new S68P;
    $fm = $p->m(...);
    check("fkb1", $fm->call(new S68P) === "P::m1S68P" && $fm->call(new S68Q) === null
        && $fm->bindTo(new S68Q)() === "P::m2S68Q" && $fm->bindTo(new S68Q)->bindTo(new S68P)() === "P::m1S68P"
        && $fm->bindTo(null) === null && $fm->bindTo(new S68R) === null && $fm->bindTo(new S68P, 'S68R') === null
        && Closure::fromCallable([$p, 'm'])->bindTo(new S68Q)() === "P::m2S68Q");
    $i = $p->inc(...);
    $n = 5;
    $i($n);
    $i->bindTo(new S68Q)($n);
    $sq = S68Q::sm(...);
    check("fkb2", $n === 8 && $sq() === "smS68Q" && $sq->bindTo(null)() === "smS68P"
        && $sq->bindTo(null, 'S68P')() === "smS68P" && $sq->bindTo(new S68P) === null && $sq->bindTo(null, 'S68Q') === null);
    $cs = S68P::nope(...);
    $pl = s68plain(...);
    $up = Closure::fromCallable('strtoupper');
    check("fkb3", $cs() === "cs:nope:S68P" && $cs->bindTo(new S68P) === null && $cs->bindTo(null)() === "cs:nope:S68P"
        && $pl->bindTo(new S68P)() === "plain" && $pl->call(new S68P) === null && $pl->bindTo(null, 'S68P') === null
        && $up->bindTo(null)("x") === "X" && $up->bindTo(null, 'S68P') === null);
    $f = new S68F;
    $nm = "foo";
    check("fkb4", Closure::fromCallable([$f, $nm . "bar"])->bindTo(new S68G)() === "call:foobar:S68G"
        && $f->{$nm . "bar"}(...)->bindTo(new S68F)() === "call:foobar:S68F");
    check("ubt1", $p->arrowUse()->bindTo(null) === null && Closure::bind($p->arrowUse(), null) === null
        && $p->viaIsset()->bindTo(null) === null && $p->nested()->bindTo(null)()() === 2
        && $p->nestedThis()->bindTo(null) !== null && $p->plainUse()->bindTo(null)() === 5
        && $p->arrowUse()->bindTo(new S68Q)() === 2);
    // A rebound closure still uses $this (ZEND_ACC_USES_THIS is the function's).
    check("ubt2", $p->arrowUse()->bindTo(new S68Q)->bindTo(null) === null
        && Closure::bind($p->arrowUse(), new S68Q, 'S68Q')->bindTo(null) === null
        && $p->plainUse()->bindTo(new S68Q)->bindTo(null)() === 5);
}

// ===== SECTION 69: callbacks naming Closure methods and missing methods, array references =====
// A Closure is an object: [$closure, 'm'] is a valid callback for its methods
// (__invoke, bind, bindTo, call, fromCallable), and $c->__invoke(...) or
// Closure::fromCallable([$c, '__invoke']) is $c itself (zend_closure_from_frame).
// A callback naming a method its class lacks goes where
// zend_is_callable_check_func sends it: ['C', 'm'] with an object bound takes
// C's own __call, and nothing else; 'C::m' takes the __call of the frame's
// $this (call_user_func's frame is passed over) or C's __callStatic, and checked
// from Closure::fromCallable, a method of an internal class, only the latter
// (Zend/tests/magic_methods/bug45186_2.phpt). [&$v] / [$k => &$v] makes the
// element a reference. Binding a closure always makes a new Closure. A method's
// by-reference parameter binds the variable even when another class declares
// the same name by value.
class S69M {
    function __call($n, $a) { return "M:" . $n . ":" . get_class($this); }
    static function __callStatic($n, $a) { return "Ms:" . $n; }
}
class S69S extends S69M {
    function __call($n, $a) { return "S:" . $n; }
    function t() {
        return [call_user_func('S69M::sp'), call_user_func(['S69M', 'sp']), call_user_func_array(['S69M', 'sp'], []),
                array_map(['S69M', 'sp'], [1])[0], array_map('S69M::sp', [1])[0],
                call_user_func(['parent', 'pp']), call_user_func('parent::pp'), call_user_func(['self', 'ss']),
                Closure::fromCallable('S69M::fc')(), Closure::fromCallable(['S69M', 'fc'])()];
    }
}
class S69N { static function __callStatic($n, $a) { return "Ns:" . $n; } }
class S69P extends S69N {
    function __call($n, $a) { return "P:" . $n; }
    function t() {
        $r = [call_user_func('S69N::q'), (is_callable(['S69N', 'q']) ? "y" : "n") . (is_callable('S69N::q') ? "y" : "n")];
        try { call_user_func(['S69N', 'q']); } catch (TypeError $e) { $r[] = $e->getMessage(); }
        return $r;
    }
}
class S69Q { function __call($n, $a) { return "Q:" . $n; } }
class S69R extends S69Q {
    function t() {
        $r = [call_user_func('S69Q::w')];
        try { Closure::fromCallable('S69Q::w'); } catch (TypeError $e) { $r[] = $e->getMessage(); }
        return $r;
    }
}
class S69V { public $v = 3; }
class S69A { function m(&$x) { $x = "A"; } static function s(&$x) { $x = "As"; } }
class S69B { function m($x, $y = 0) { $x = "B"; } static function s($x, $y = 0) { $x = "Bs"; } }
function s69twice($x) { return $x * 2; }
function s69() {
    $m = function($a, $b) { return $a + $b; };
    $iv = [$m, '__invoke'];
    $bad = "";
    try { call_user_func([$m, 'nope']); } catch (TypeError $e) { $bad = $e->getMessage(); }
    check("cli1", call_user_func([$m, '__invoke'], 3, 4) === 7 && call_user_func_array([$m, '__INVOKE'], [5, 6]) === 11
        && $iv(10, 20) === 30 && array_map([$m, '__invoke'], [1, 2], [3, 4]) === [4, 6]
        && is_callable([$m, 'bindTo']) && !is_callable([$m, 'nope'])
        && $bad === 'call_user_func(): Argument #1 ($callback) must be a valid callback, class Closure does not have a method "nope"');
    $g = function($k) { return $this->v * $k; };
    check("cli2", call_user_func([$g, 'bindTo'], new S69V, 'S69V')(2) === 6 && call_user_func([$g, 'call'], new S69V, 4) === 12
        && Closure::fromCallable([$m, '__invoke']) === $m && $m->__invoke(...) === $m);
    $p = 1;
    $q = 2;
    foreach ([&$p, &$q] as &$x) { $x = $x * 10; }
    unset($x);
    $arr = [1, 2];
    $o = new S69V;
    foreach (['a' => &$arr[1], 'b' => &$fresh, 'c' => &$o->v] as $k => &$y) { $y = $k; }
    unset($y);
    check("arf1", $p === 10 && $q === 20 && $arr[1] === "a" && $fresh === "b" && $o->v === "c"
        && count([&$p, 'k' => &$q, 5]) === 3);
    $c = s69twice(...);
    $u = Closure::fromCallable('strtoupper');
    check("fbi1", $c->bindTo(null) !== $c && $c->bindTo(null)(4) === 8 && Closure::bind($c, new S69V) !== $c
        && $u->bindTo(null) !== $u && $u->bindTo(null)("x") === "X");
    check("cbf1", (new S69S)->t() === ["S:sp", "M:sp:S69S", "M:sp:S69S", "M:sp:S69S", "S:sp",
                                      "M:pp:S69S", "S:pp", "S:ss", "Ms:fc", "M:fc:S69S"]);
    check("cbf2", (new S69P)->t() === ["Ns:q", "ny",
            'call_user_func(): Argument #1 ($callback) must be a valid callback, class S69N does not have a method "q"']
        && (new S69R)->t() === ["Q:w", 'Failed to create closure from callable: class S69Q does not have a method "w"']);
    $v1 = 1; (new S69A)->m($v1);
    $v2 = 1; S69A::s($v2);
    $v3 = 1; (new S69B)->m($v3);
    $seen = "";
    foreach ([new S69A, new S69B] as $ob) { $v = 1; $ob->m($v); $seen .= $v; }
    $mn = "m";
    $v4 = 1; (new S69A)->$mn($v4);
    check("mbr1", $v1 === "A" && $v2 === "As" && $v3 === 1 && $seen === "A1" && $v4 === "A");
}

// ===== SECTION 70: arrays passed by value, and references inside array copies =====
function s70set($xs) { $xs[0] = 5; return $xs[0]; }
function s70push($xs) { $xs[] = 3; array_push($xs, 4); return count($xs); }
function s70deep($xs) { $xs['k'][0] = 9; unset($xs['j']); return $xs['k'][0]; }
function s70refs($xs) { foreach ($xs as &$v) { $v = 8; } unset($v); $r = &$xs[1]; $r = 7; return $xs[0] + $xs[1]; }
function s70rest(...$xs) { $xs[0][0] = 6; return $xs[0][0]; }
function s70read($xs) { return $xs[0] + count($xs); }
function s70gen($xs) { yield $xs[0]; yield $xs[0]; }
class S70M { public $items = [1]; function m($xs) { $xs[] = 2; return count($xs); } static function s($xs) { $xs[0]++; return $xs[0]; }
    function inc(&$x) { $x += 10; } }
class S70V { public $v = 3; }
class S70A { function m(&$x, $y = 0) { $x = "A"; } }
class S70B { function m($x, $y = 0) { $x = "B"; } }
function s70named(&$x, $y = 0) { $x = "F"; }
function s70inc(&$x) { $x++; }
function s70set2(&$z) { $z = 3; }
function s70() {
    $a = [1, 2];
    $n = ['k' => [1], 'j' => 2];
    $f = function($xs) { $xs[0] = 4; return $xs[0]; };
    $k = new S70M;
    check("abv1", s70set($a) === 5 && s70push($a) === 4 && s70deep($n) === 9 && s70refs($a) === 15
        && s70rest($a) === 6 && $f($a) === 4 && call_user_func($f, $a) === 4 && S70M::s($a) === 2
        && $k->m($k->items) === 2 && $a === [1, 2] && $n === ['k' => [1], 'j' => 2] && $k->items === [1]);
    $g = s70gen($a);
    $first = $g->current();
    $a[0] = 5;
    $g->next();
    check("abv2", s70read($a) === 7 && $first === 1 && $g->current() === 1);
    $n = 1; $r1 = []; $r1[0] = &$n; $r2 = $r1; $r2[0] = 7;
    $k = 1; $r3 = [&$k, 4]; $r3[0] = 9;
    $x = 1; $r4 = []; $r4[0] = &$x; unset($x); $r5 = $r4; $r5[0] = 5;
    $p = [1, 2]; s70set2($p[0]); $q = $p; $q[0] = 8;
    check("arc1", $n === 7 && $k === 9 && $r4[0] === 1 && $p[0] === 3);
    $l1 = [1, 2, 3]; foreach ($l1 as &$v) { $v *= 2; } unset($v); $c1 = $l1; $c1[2] = 0;
    $l2 = [1, 2, 3]; foreach ($l2 as &$v) { $v *= 2; } $c2 = $l2; $c2[2] = 0;
    $l3 = [1, 2, 3]; foreach ($l3 as &$w) { } foreach ($l3 as $w) { }
    $l4 = [1, 2]; foreach ($l4 as &$v4) { } unset($v4); foreach ($l4 as $x4) { $x4 = 9; }
    check("arc2", $l1 === [2, 4, 6] && $l2 === [2, 4, 0] && $l3 === [1, 2, 2] && $l4 === [1, 2]);
    $m = 1; $sp = []; $sp[0] = &$m;
    call_user_func_array('s70inc', $sp);
    s70inc(...$sp);
    s70inc(...[&$m]);
    (new S70M)->inc(...$sp);
    $plain = [1]; call_user_func_array('s70inc', $plain);
    check("spr1", $m === 14 && $plain === [1]);
    $th = function() { return $this->v; };
    $d = function() { return 1; };
    $cb = [$d, 'bind'];
    $bad = "";
    try { $d->bind(1, null); } catch (TypeError $e) { $bad = $e->getMessage(); }
    $f1 = 1; s70named(y: 2, x: $f1);
    $f2 = 1; s70named($f2, y: 2);
    $seen = "";
    foreach ([new S70A, new S70B] as $ob) {
        $v1 = 1; $ob->m(y: 2, x: $v1);
        $v2 = 1; $ob->m($v2);
        $v3 = 1; $ra = [&$v3]; $ob->m(...$ra);
        $seen .= $v1 . $v2 . $v3;
    }
    check("nbr1", $f1 === "F" && $f2 === "F" && $seen === "AAA111");
    check("cst1", $d->bind($th, new S70V)() === 3 && $cb($th, new S70V)() === 3
        && call_user_func([$d, 'bind'], $th, new S70V)() === 3 && $d->fromCallable('strtoupper')("x") === "X"
        && $bad === 'Closure::bind(): Argument #1 ($closure) must be of type Closure, int given');
}

// ===== SECTION 71: spreads into by-reference parameters, Closure's statics by name, library callbacks =====
// A spread of a variable makes each element a by-reference parameter takes a
// reference in that variable's array (zend_vm_def.h ZEND_SEND_UNPACK); a
// by-reference variadic takes every position from its own. call_user_func_array
// passes a non-reference element as a temporary one and leaves its list alone
// (zend_call_function). 'Closure::bind' / ['Closure', 'fromCallable'] name the
// static methods (zend_closures.c). A library function's callback is judged at
// the first user frame (zend_API.c zend_is_callable_ex), with its $this and class
// scope: a method that scope may not call goes to __call (with an object bound)
// or __callStatic (without) when the class has it, an explicit ['C', 'm'] with an
// object bound reaches only C's __call (zend_is_callable_check_func), and a
// trampoline keeps the called scope of the bound $this.
function s71inc(&$x) { $x++; }
function s71two($y, &$x) { $x .= "!"; }
function s71named(&$x, &$z = null) { $x += 100; }
function s71all(&...$xs) { foreach ($xs as &$x) { $x *= 2; } }
function s71keep(&...$xs) { return $xs; }
class S71K { function m(&$x) { $x .= "m"; } static function s(&$x) { $x .= "s"; } }
class S71P { private $v = 42; }
class S71N { static function __callStatic($n, $a) { return "N:" . $n; } }
class S71NK extends S71N { function t() {
    try { return array_map(['S71N', 'q'], [1]); } catch (TypeError $e) { return $e->getMessage(); } } }
class S71S { function m($x) { return "m" . $x; } }
class S71SK extends S71S { static function t() {
    try { return array_map(['S71S', 'm'], [1]); } catch (TypeError $e) { return $e->getMessage(); } } }
class S71C { private function p($x = 0) { return "p"; } function __call($n, $a) { return "call:" . static::class . ":" . $n; } }
class S71CK extends S71C { function t() {
    return [array_map(['S71C', 'p'], [1])[0], call_user_func('S71C::p'), call_user_func([$this, 'p'])]; } }
class S71Q { private function p() { return "p"; } static function __callStatic($n, $a) { return "cs:" . static::class . ":" . $n; } }
class S71QK extends S71Q {
    static function t() { return call_user_func(['S71Q', 'p']); }
    function u() { return [call_user_func('S71Q::zz'), Closure::fromCallable('S71Q::zz')()]; } }
function s71() {
    $a = [1]; s71inc(...$a);
    $b = [1, "a"]; s71two(...$b);
    $c = ['x' => 5]; s71named(...$c);
    $d = [7]; s71inc(...[...$d]);
    $f = 's71inc'; $e = [3]; $f(...$e);
    $g = function (&$x) { $x *= 3; }; $h = [4]; $g(...$h);
    $k = new S71K; $i = ["k"]; $k->m(...$i); $j = ["t"]; S71K::s(...$j);
    $l = [1]; $l2 = $l; s71inc(...$l);
    $n = [1]; s71inc(...$n); $n2 = $n; $n2[0] = 9;
    check("sbr1", $a === [2] && $b === [1, "a!"] && $c === ['x' => 105] && $d === [7] && $e === [4]
        && $h === [12] && $i === ["km"] && $j === ["ts"] && $l === [2] && $l2 === [1] && $n === [2]);
    $p = 1; $q = 2; s71all($p, $q);
    $r = [3, 4]; s71all(...$r);
    $s = 1; $t = s71keep($s); $t[0] = 5;
    $cf = [1]; call_user_func_array('s71inc', $cf);
    check("sbr2", $p === 2 && $q === 4 && $r === [6, 8] && $s === 5 && $cf === [1]);
    $th = function () { return $this->v; };
    $bad = "";
    try { call_user_func(['Closure', 'nope']); } catch (TypeError $e) { $bad = $e->getMessage(); }
    $ns = "";
    try { call_user_func('Closure::bindTo'); } catch (TypeError $e) { $ns = $e->getMessage(); }
    check("cls1", call_user_func(['Closure', 'bind'], $th, new S71P, S71P::class)() === 42
        && call_user_func('closure::BIND', $th, new S71P, S71P::class)() === 42
        && call_user_func_array(['\Closure', 'bind'], [$th, new S71P, S71P::class])() === 42
        && call_user_func(['Closure', 'fromCallable'], 'strtoupper')("ab") === "AB"
        && call_user_func_array([$th, 'fromCallable'], ['strrev'])("ab") === "ba"
        && array_map(['Closure', 'fromCallable'], ['strlen'])[0]("four") === 4
        && $bad === 'call_user_func(): Argument #1 ($callback) must be a valid callback, class Closure does not have a method "nope"'
        && $ns === 'call_user_func(): Argument #1 ($callback) must be a valid callback, non-static method Closure::bindTo() cannot be called statically');
    check("lcb1", (new S71NK)->t() === 'array_map(): Argument #1 ($callback) must be a valid callback or null, class S71N does not have a method "q"'
        && S71SK::t() === 'array_map(): Argument #1 ($callback) must be a valid callback or null, non-static method S71S::m() cannot be called statically'
        && (new S71CK)->t() === ["call:S71CK:p", "call:S71CK:p", "call:S71CK:p"]
        && S71QK::t() === "cs:S71Q:p" && (new S71QK)->u() === ["cs:S71QK:zz", "cs:S71QK:zz"]);
}

// ===== SECTION 72: library callbacks by name and of the u* set operations, argument order, named by-reference variadics =====
// Every library function's callback is judged at the first user frame, also when
// the function is called through its name or is array_walk_recursive or a u* set
// operation (zend_API.c zend_is_callable_at_frame). zpp judges the arguments in
// order: usort's array $array before its callback, but the u* variants'
// trailing callback ("+f") before any of their arrays, and that position has no
// parameter name (ext/standard/array.c php_array_diff; Zend
// get_function_arg_name). A named argument a by-reference variadic collects is
// by reference too (Zend/tests/named_params/variadic.phpt).
class S72A {
    private function cmp($a, $b) { return $a <=> $b; }
    private static function rcmp($a, $b) { return $b <=> $a; }
    private function walk(&$v, $k, $x = 0) { $v = $v + $x + 1; }
    function t() {
        $a = [3, 1, 2];
        $out = [];
        $c = $a; $f = 'usort'; $f($c, [$this, 'cmp']); $out[] = implode(',', $c);
        $c = $a; $f = 'uasort'; $f($c, 'self::rcmp'); $out[] = implode(',', $c);
        $out[] = implode(',', array_udiff($a, [2, 5], [$this, 'cmp']));
        $out[] = implode(',', array_uintersect($a, [2, 5], 'self::rcmp'));
        $out[] = implode(',', array_keys(array_diff_ukey(['a' => 1, 'b' => 2], ['a' => 3], [$this, 'cmp'])));
        $u = 'array_intersect_uassoc';
        $out[] = implode(',', array_keys($u(['a' => 1, 'b' => 2], ['a' => 1], [$this, 'cmp'])));
        $w = [1, [2, 3]]; array_walk_recursive($w, [$this, 'walk'], 10);
        $f = 'array_walk'; $v = [1, 2]; $f($v, [$this, 'walk']);
        $out[] = json_encode($w) . json_encode($v);
        return implode(' ', $out);
    }
    function errs() {
        $x = 5;
        $a = [1];
        $out = [];
        $calls = [function () use (&$x) { usort($x, 'self::nope'); },
                  function () use ($a) { $f = 'array_reduce'; $f($a, [$this, 'nope']); },
                  function () use ($x) { array_filter($x, [$this, 'nope']); },
                  function () use (&$x) { array_walk_recursive($x, [$this, 'nope']); },
                  function () use ($x, $a) { array_udiff($x, $a, [$this, 'nope']); },
                  function () use ($x, $a) { $f = 'array_uintersect_assoc'; $f($a, $a, $x, [$this, 'cmp']); },
                  function () use ($a) { array_diff_ukey($a); }];
        foreach ($calls as $cl) {
            try { $cl(); $out[] = "none"; } catch (TypeError | ArgumentCountError $e) { $out[] = $e->getMessage(); }
        }
        return $out;
    }
}
function s72all(&...$refs) { foreach ($refs as $k => &$r) { $r = $r . $k; } }
class S72M { function m(&...$v) { foreach ($v as &$w) { $w = -$w; } } }
function s72() {
    $cl = Closure::bind(function () { $f = 'usort'; $c = [2, 1]; $f($c, [$this, 'cmp']); return implode(',', $c); },
                        new S72A, S72A::class);
    check("lcb2", (new S72A)->t() === "1,2,3 3,2,1 3,1 2 b a [12,[13,14]][2,3]" && $cl() === "1,2");
    check("lae1", (new S72A)->errs() === [
        'usort(): Argument #1 ($array) must be of type array, int given',
        'array_reduce(): Argument #2 ($callback) must be a valid callback, class S72A does not have a method "nope"',
        'array_filter(): Argument #1 ($array) must be of type array, int given',
        'array_walk_recursive(): Argument #1 ($array) must be of type array, int given',
        'array_udiff(): Argument #3 must be a valid callback, class S72A does not have a method "nope"',
        'array_uintersect_assoc(): Argument #3 must be of type array, int given',
        'array_diff_ukey() expects at least 2 arguments, 1 given']);
    $p = "p"; $q = "q"; $r = "r";
    s72all($p, s: $q, t: $r);
    $f = 's72all'; $d = "d"; $f(e: $d);
    $g = function (&...$v) { foreach ($v as &$w) { $w = $w * 10; } };
    $n = 1; $g(k: $n);
    $o = 2; (new S72M)->m(u: $o);
    $arr = ['a' => 'x']; s72all(a: $arr['a']);
    $sp = ['y', 'k' => 'z']; s72all(...$sp);
    check("nbv1", $p === "p0" && $q === "qs" && $r === "rt" && $d === "de" && $n === 10 && $o === -2
        && $arr === ['a' => 'xa'] && $sp === ['y0', 'k' => 'zk']);
}

// ===== SECTION 73: function names in any case, array_map's arrays, unpacking order, callbacks judged by the caller =====
// A function name given at run time is looked up lower-cased with one leading
// "\" dropped, and a miss names it as given (Zend/zend_execute.c
// zend_init_dynamic_call_string). array_map's arrays are TypeErrors, only #2
// having a parameter name (ext/standard/array.c; Zend get_function_arg_name).
// A spread's integer key after one of its string keys is an Error
// (Zend/tests/named_params/unpack.phpt). A library function judges its callback
// at the first user frame, also when it is reached through call_user_func, as a
// first-class callable, or with the callback passed by name; the "+ff" pair of
// array_udiff_uassoc / array_uintersect_uassoc is judged in argument order
// before any array.
function S73Cmp($a, $b) { return $a <=> $b; }
function s73f($x, $k = 0) { return "$x/$k"; }
class S73A {
    private function cmp($a, $b) { return $b <=> $a; }
    private function kcmp($a, $b) { return strcmp($a, $b); }
    private function neg($x) { return -$x; }
    function t() {
        $a = [3, 1, 2];
        $out = [];
        $c = $a; usort($c, callback: [$this, 'cmp']); $out[] = implode(',', $c);
        $out[] = implode(',', array_map(array: [1, 2], callback: [$this, 'neg']));
        $out[] = implode(',', call_user_func('array_map', [$this, 'neg'], $a));
        $u = usort(...); $c = $a; $u($c, [$this, 'cmp']); $out[] = implode(',', $c);
        $f = 'USORT'; $c = $a; $f($c, [$this, 'cmp']); $out[] = implode(',', $c);
        $x = ['a' => 1, 'b' => 2, 'c' => 3];
        $y = ['a' => 1, 'b' => 5];
        $out[] = json_encode(array_udiff_uassoc($x, $y, [$this, 'cmp'], [$this, 'kcmp']));
        $out[] = json_encode(array_uintersect_uassoc($x, $y, [$this, 'cmp'], [$this, 'kcmp']));
        return implode(' ', $out);
    }
    function errs() {
        $out = [];
        foreach ([function () { array_udiff_uassoc([1], [$this, 'cmp']); },
                  function () { array_udiff_uassoc([1], [1], [$this, 'nope'], [$this, 'nope2']); },
                  function () { array_uintersect_uassoc([1], 5, [$this, 'cmp'], [$this, 'nope2']); },
                  function () { array_uintersect_uassoc([1], 5, [$this, 'cmp'], [$this, 'kcmp']); }] as $cl) {
            try { $cl(); $out[] = "none"; } catch (TypeError | ArgumentCountError $e) { $out[] = $e->getMessage(); }
        }
        return $out;
    }
}
function s73err($f) {
    try { $f(); return "none"; } catch (Error $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s73() {
    $a = [3, 1, 2];
    $f = 'USORT'; $f($a, 'S73CMP');
    $g = 'StrToUpper';
    $h = '\UKSORT'; $b = ['y' => 1, 'x' => 3]; $h($b, '\s73cmp');
    $k = '\S73CMP';
    check("fnc1", implode(',', $a) === "1,2,3" && $g("abc") === "ABC" && implode(',', array_keys($b)) === "x,y"
        && $k(1, 2) === -1 && implode(',', array_map('STRTOUPPER', ['q'])) === "Q");
    check("fnc2", s73err(function () { $f = '\S73None'; $f(); }) === "Error: Call to undefined function \\S73None()"
        && s73err(function () { $f = 'S73None'; $f(); }) === "Error: Call to undefined function S73None()");
    check("amt1", s73err(function () { array_map('strlen', 5); }) === "TypeError: array_map(): Argument #2 (\$array) must be of type array, int given"
        && s73err(function () { array_map(null, true); }) === "TypeError: array_map(): Argument #2 (\$array) must be of type array, true given"
        && s73err(function () { array_map('strlen', ['a'], 'x'); }) === "TypeError: array_map(): Argument #3 must be of type array, string given");
    $sp = ['k' => 5, 6];
    check("upk1", s73err(function () use ($sp) { s73f(1, ...$sp); }) === "Error: Cannot use positional argument after named argument during unpacking"
        && s73err(function () { $f = 's73f'; $f(...['k' => 1, 2]); }) === "Error: Cannot use positional argument after named argument during unpacking"
        && s73f(...[1, 'k' => 2]) === "1/2");
    check("ccb1", (new S73A)->t() === '3,2,1 -1,-2 -3,-1,-2 3,2,1 3,2,1 {"b":2,"c":3} {"a":1}'
        && s73err(function () { $u = usort(...); $c = [1, 2]; $u($c, [new S73A, 'cmp']); })
        === "TypeError: usort(): Argument #2 (\$callback) must be a valid callback, cannot access private method S73A::cmp()");
    check("ccb2", (new S73A)->errs() === [
        'array_udiff_uassoc() expects at least 3 arguments, 2 given',
        'array_udiff_uassoc(): Argument #3 must be a valid callback, class S73A does not have a method "nope"',
        'array_uintersect_uassoc(): Argument #4 must be a valid callback, class S73A does not have a method "nope2"',
        'array_uintersect_uassoc(): Argument #2 must be of type array, int given']);
}

// ===== SECTION 74: function_exists, direct calls in any case, call_user_func_array's library callbacks =====
// Function names are case-insensitive in a direct call as in a dynamic one, and
// a miss names the function as written (Zend/zend_compile.c lower-cases the
// lookup key, zend_execute.c ZEND_INIT_FCALL_BY_NAME words the miss).
// function_exists looks the name up lower-cased with one leading "\" dropped
// (Zend/zend_builtin_functions.c). An array callback of another size is "array
// callback must have exactly two members" (Zend/zend_API.c
// zend_is_callable_at_frame). A library function reached through
// call_user_func_array judges its callback at the first user frame, as through
// call_user_func (zend_API.c get_scope skips internal frames). A named function
// taken as a value is a Closure with a handle, which var_dump shows by
// "function" and "parameter" (Zend/zend_closures.c zend_closure_get_debug_info
// of a fake closure); s74dump puts that in the transcript.
function S74Inc(&$x) { $x++; return $x; }
function N() { return "n"; }
class S74A {
    private function neg($x) { return -$x; }
    private function odd($x) { return $x % 2 == 1; }
    private function cmp($a, $b) { return $a <=> $b; }
    function t() {
        $a = [1, 2, 3];
        $args = [[$this, 'neg'], $a];
        $out = [implode(',', call_user_func_array('array_map', $args)), is_array($args[0]) ? "kept" : "changed"];
        $out[] = implode(',', call_user_func_array('Array_Filter', ['array' => $a, 'callback' => [$this, 'odd']]));
        $out[] = implode(',', call_user_func_array('array_udiff', [$a, [2], [$this, 'cmp']]));
        $f = function () use ($a) { return call_user_func_array('array_map', [[$this, 'neg'], $a]); };
        $out[] = implode(',', $f());
        try { call_user_func_array('array_udiff', [$a, [2], [$this]]); } catch (TypeError $e) { $out[] = $e->getMessage(); }
        return $out;
    }
}
function s74err($f) {
    try { $f(); return "none"; } catch (Error $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s74() {
    check("fex1", function_exists('strlen') && function_exists('S74Inc') && function_exists('s74inc')
        && function_exists('\S74INC') && !function_exists('s74none') && function_exists('USORT')
        && Function_Exists('array_map'));
    $v = 1;
    S74INC($v);
    s74inc($v);
    $g = S74INC(...);
    $g($v);
    check("fnc3", $v === 4 && STRTOUPPER("x") === "X" && StrLen("abc") === 3 && n() === "n"
        && implode(',', Array_Map(fn($x) => $x * 3, [1, 2])) === "3,6" && \StrToLower("Q") === "q");
    check("fnc4", s74err(function () { S74Missing(1); }) === "Error: Call to undefined function S74Missing()"
        && s74err(function () { \S74Missing2(); }) === "Error: Call to undefined function S74Missing2()");
    check("fnc5", (function () { $f = 'N'; return $f(); })() === "n");
    check("cbm1", s74err(function () { call_user_func([1], 2); })
        === "TypeError: call_user_func(): Argument #1 (\$callback) must be a valid callback, array callback must have exactly two members"
        && s74err(function () { array_map([1, 2, 3], [1]); })
        === "TypeError: array_map(): Argument #1 (\$callback) must be a valid callback or null, array callback must have exactly two members");
    check("fcd1", spl_object_id(S74Inc(...)) > 0 && spl_object_id(strlen(...)) > 0 && spl_object_id(usort(...)) > 0);
    check("cfa1", (new S74A)->t() === ['-1,-2,-3', 'kept', '1,3', '1,3', '-1,-2,-3',
        'array_udiff(): Argument #3 must be a valid callback, array callback must have exactly two members']);
}

function s74dump() {
    var_dump(usort(...), S74INC(...), N(...));
    $f = 'StrLen';
    var_dump($f(...));
}

// ===== SECTION 75: first-class callable identity, closures made from methods, natives by name =====
// Every f(...), $name(...) and Closure::fromCallable('f') makes a NEW Closure
// object (Zend/zend_closures.c zend_create_fake_closure), with its own handle;
// a closure's own $c(...) and fromCallable($c) are $c itself
// (zend_closure_from_frame). A closure made from a method is shown by
// "function" "<class>::<method>", its bound "this" and "parameter"
// (zend_closure_get_debug_info); one standing in for __call / __callStatic is
// named after the method asked for, in the class declaring the magic method,
// with the one optional parameter $arguments (zend_get_call_trampoline_func).
// s75dump puts those in the transcript. define, defined, constant,
// error_reporting and realpath are functions like any other: function_exists
// knows them and a name calls them; function_exists knows every library
// function, whether or not the program names it literally.
function s75f($x = 1) { return $x; }
class S75P { function __call($n, $a) { return $n; } static function __callStatic($n, $a) { return $n; } }
class S75C extends S75P {
    public $own;
    function m($a, &$b = null) { return $a; }
    static function s() { return 7; }
}
function s75() {
    $a = s75f(...);
    $b = s75f(...);
    $n = 's75' . 'f';
    check("fci1", $a !== $b && $a === $a && Closure::fromCallable('s75f') !== $a && $n(...) !== $n(...)
        && spl_object_id($a) !== spl_object_id($b) && spl_object_id($a) === spl_object_id($a)
        && spl_object_id($a) !== spl_object_id(Closure::fromCallable($n)) && $a(4) === 4 && $n(...)(5) === 5);
    $o = new S75C;
    $m1 = $o->m(...);
    $m2 = $o->m(...);
    $s1 = S75C::s(...);
    $u = $o->other(...);
    check("fci2", $m1 !== $m2 && spl_object_id($m1) > 0 && spl_object_id($m1) !== spl_object_id($m2)
        && spl_object_id($s1) > 0 && spl_object_id($u) > 0 && spl_object_id($u) !== spl_object_id($s1)
        && $m1(3) === 3 && $s1() === 7 && $u() === "other");
    $c = function () { return 1; };
    check("fci3", $c(...) === $c && Closure::fromCallable($c) === $c);
    $d = 'define';
    $d('S75_K', 9);
    $df = 'defined';
    $cf = 'constant';
    $er = 'error_reporting';
    check("nat1", function_exists('define') && function_exists('defined') && function_exists('constant')
        && function_exists('error_reporting') && function_exists('realpath')
        && $df('S75_K') && !$df('S75_NONE') && $cf('S75_K') === 9 && is_int($er()));
    $k = ['diff_key', 'ripos'];
    check("fex2", function_exists('array_' . $k[0]) && function_exists('str' . $k[1])
        && !function_exists('array_' . $k[1]) && !function_exists('str' . $k[1] . ',strlen'));
}
function s75dump() {
    $o = new S75C;
    var_dump($o->m(...));
    var_dump(S75C::s(...), $o->other(...), S75C::other(...));
    $o->own = $o->m(...);
    var_dump($o);
}

// ===== SECTION 76: internal functions called with a wrong argument count =====
// An internal function checks its argument count against its declaration before
// it runs, whether it is called directly, by name or as a callback, and raises
// the ArgumentCountError zend_wrong_parameters_count_error words:
// "<f>() expects exactly|at least|at most <n> argument(s), <given> given".
// "exactly" when min == max, else "at least" for too few and "at most" for too
// many; a variadic function has no maximum. The counts are the stubs' (strtr
// takes 2-3, strrchr 2-3, str_word_count 1-3, pi none).
class S76K { function m() { return get_class(); } }
function s76ace($f, $args) {
    try { call_user_func_array($f, $args); return "none"; } catch (ArgumentCountError $e) { return $e->getMessage(); }
}
function s76() {
    $m = [];
    try { strrev(); } catch (ArgumentCountError $e) { $m[] = $e->getMessage(); }
    $f = 'strrev';
    try { $f(); } catch (ArgumentCountError $e) { $m[] = $e->getMessage(); }
    try { call_user_func('strrev'); } catch (ArgumentCountError $e) { $m[] = get_class($e) . ": " . $e->getMessage(); }
    check("aca1", $m === ["strrev() expects exactly 1 argument, 0 given", "strrev() expects exactly 1 argument, 0 given",
        "ArgumentCountError: strrev() expects exactly 1 argument, 0 given"]);
    check("aca2", s76ace('str_pad', ['a']) === "str_pad() expects at least 2 arguments, 1 given"
        && s76ace('str_pad', ['a', 3, '-', 0, 9]) === "str_pad() expects at most 4 arguments, 5 given"
        && s76ace('implode', [',', [1], 3]) === "implode() expects at most 2 arguments, 3 given"
        && s76ace('max', []) === "max() expects at least 1 argument, 0 given"
        && s76ace('pi', [1]) === "pi() expects exactly 0 arguments, 1 given"
        && s76ace('array_key_exists', [1]) === "array_key_exists() expects exactly 2 arguments, 1 given"
        && s76ace('array_merge', []) === "none" && s76ace('str_pad', ['a', 3]) === "none");
    $n = [];
    try { strlen(); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    try { strlen("a", "b"); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    $o = 'ord';
    try { $o("a", "b"); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    $w = ['x', 'y'];
    try { array_walk($w, 'strtoupper'); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    check("aca3", $n === ["strlen() expects exactly 1 argument, 0 given", "strlen() expects exactly 1 argument, 2 given",
        "ord() expects exactly 1 argument, 2 given", "strtoupper() expects exactly 1 argument, 2 given"]);
    check("aca4", s76ace('strtr', ['abc']) === "strtr() expects at least 2 arguments, 1 given"
        && s76ace('strtr', ['abc', 'a', 'b', 'c']) === "strtr() expects at most 3 arguments, 4 given"
        && strtr("abc", "ab", "xy") === "xyc" && strrchr("a/b/c", "/", true) === "a/b" && strrchr("a/b/c", "/") === "/c"
        && str_word_count("it's a-b x1y", 1, "1") === ["it's", "a-b", "x1y"]
        && str_word_count("-ab- cd", 2, "-") === [0 => "-ab-", 5 => "cd"] && (new S76K)->m() === "S76K");
}

// ===== SECTION 77: user functions called with too few arguments =====
// A user function given too few arguments raises zend_missing_arg_error's
// ArgumentCountError: "Too few arguments to function <f>(), <n> passed in <file>
// on line <l> and exactly|at least <r> expected" - the file and line the CALLER's,
// <f> "Class::m" for a method (the class it is declared in), "exactly" when every
// non-variadic parameter is required. A call an internal function makes (a
// callback) names no site: "<n> passed and exactly <r> expected". A named argument
// that skips a required parameter raises "<f>(): Argument #<i> ($<p>) not passed"
// (zend_handle_undef_args) instead, for an internal function too. A library
// method counts like an internal function (SplStack::push is
// SplDoublyLinkedList's), and array_keys / array_walk / array_walk_recursive take
// at most 3 arguments: their third parameter is an UNKNOWN default, no variadic.
function s77one($a) { return $a; }
function s77opt($a, $b = 2, ...$c) { return $a; }
function s77two($a, $b) { return $a . $b; }
class S77K {
    function __construct($a = 0, $b = 0) { }
    function m($x) { return $x; }
    static function s($x, $y) { return $x; }
    function clo() { return function ($q) { return $q; }; }
}
class S77C { function __construct($a, $b) { } }
// The message with this file's path as F and line $l as L.
function s77t($e, $l) {
    return str_replace(" on line " . $l . " ", " on line L ", str_replace(__FILE__, "F", $e->getMessage()));
}
function s77() {
    $m = [];
    try { s77one(); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    try { s77opt(); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    try { s77two(1); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    try { (new S77K)->m(); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    try { S77K::s(1); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    try { new S77C(1); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    $f = 's77one';
    try { $f(); } catch (ArgumentCountError $e) { $m[] = s77t($e, __LINE__); }
    check("tfa1", $m === [
        "Too few arguments to function s77one(), 0 passed in F on line L and exactly 1 expected",
        "Too few arguments to function s77opt(), 0 passed in F on line L and at least 1 expected",
        "Too few arguments to function s77two(), 1 passed in F on line L and exactly 2 expected",
        "Too few arguments to function S77K::m(), 0 passed in F on line L and exactly 1 expected",
        "Too few arguments to function S77K::s(), 1 passed in F on line L and exactly 2 expected",
        "Too few arguments to function S77C::__construct(), 1 passed in F on line L and exactly 2 expected",
        "Too few arguments to function s77one(), 0 passed in F on line L and exactly 1 expected"]);
    $c = (new S77K)->clo();
    $cm = "";
    try { $c(); } catch (ArgumentCountError $e) { $cm = s77t($e, __LINE__); }
    $tail = "}(), 0 passed in F on line L and exactly 1 expected";
    check("tfa2", substr($cm, 0, 44) === "Too few arguments to function S77K::{closure"
        && substr($cm, strlen($cm) - strlen($tail)) === $tail);
    $n = [];
    try { array_map('s77two', [1]); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    try { call_user_func('s77one'); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    try { s77two(b: 1); } catch (ArgumentCountError $e) { $n[] = $e->getMessage(); }
    try { s77two(a: 1); } catch (ArgumentCountError $e) { $n[] = s77t($e, __LINE__); }
    try { s77opt(x: 1); } catch (ArgumentCountError $e) { $n[] = s77t($e, __LINE__); }
    check("tfa3", $n === ["Too few arguments to function s77two(), 1 passed and exactly 2 expected",
        "Too few arguments to function s77one(), 0 passed and exactly 1 expected",
        "s77two(): Argument #1 (\$a) not passed",
        "Too few arguments to function s77two(), 1 passed in F on line L and exactly 2 expected",
        "Too few arguments to function s77opt(), 0 passed in F on line L and at least 1 expected"]);
    $w = [1];
    $k = [];
    try { (new SplStack)->push(); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { str_repeat(times: 3); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { str_pad(string: "a"); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { array_fill(start_index: 0, value: 1); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { array_keys([1], 1, true, 4); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { array_walk($w, 'strval', 1, 2); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    try { array_walk_recursive($w, 'strval', 1, 2); } catch (ArgumentCountError $e) { $k[] = $e->getMessage(); }
    check("tfa4", $k === ["SplDoublyLinkedList::push() expects exactly 1 argument, 0 given",
        "str_repeat(): Argument #1 (\$string) not passed", "str_pad() expects at least 2 arguments, 1 given",
        "array_fill(): Argument #2 (\$count) not passed", "array_keys() expects at most 3 arguments, 4 given",
        "array_walk() expects at most 3 arguments, 4 given", "array_walk_recursive() expects at most 3 arguments, 4 given"]);
    check("tfa5", str_pad(length: 3, pad_string: "-", string: "a") === "a--" && strlen(string: "abc") === 3
        && array_keys([1, 2, 1], 1) === [0, 2]
        && str_repeat(times: 2, string: "ab") === "abab" && s77opt(1) === 1 && (new S77K(1))->m(5) === 5);
    // A generator function checks its arguments at the CALL, before the generator
    // exists: php-src compiles the ZEND_RECV ops ahead of ZEND_GENERATOR_CREATE.
    $g = [];
    try { s77gen(); $g[] = "made"; } catch (ArgumentCountError $e) { $g[] = s77t($e, __LINE__); }
    try { S77G::pair(1); $g[] = "made"; } catch (ArgumentCountError $e) { $g[] = s77t($e, __LINE__); }
    try { (new S77G)->one(); $g[] = "made"; } catch (ArgumentCountError $e) { $g[] = s77t($e, __LINE__); }
    try { s77gen(c: 1); $g[] = "made"; } catch (Error $e) { $g[] = $e->getMessage(); }
    check("tfa6", $g === ["Too few arguments to function s77gen(), 0 passed in F on line L and exactly 1 expected",
        "Too few arguments to function S77G::pair(), 1 passed in F on line L and exactly 2 expected",
        "Too few arguments to function S77G::one(), 0 passed in F on line L and exactly 1 expected",
        "Unknown named parameter \$c"]);
    // get_class and get_parent_class take at most one argument (ZEND_PARSE_PARAMETERS_START(0, 1)).
    $o = [new S77G, 2];
    $x = [];
    try { get_class($o[0], $o[1]); } catch (ArgumentCountError $e) { $x[] = $e->getMessage(); }
    try { get_parent_class($o[0], $o[1]); } catch (ArgumentCountError $e) { $x[] = $e->getMessage(); }
    check("tfa7", $x === ["get_class() expects at most 1 argument, 2 given",
        "get_parent_class() expects at most 1 argument, 2 given"]);
}
function s77gen($a) { yield $a; }
class S77G {
    static function pair($a, $b) { yield $a => $b; }
    function one($x) { yield $x; }
}

// ===== SECTION 78: scalar parameter and return types =====
// A typed parameter checks its argument (zend_verify_arg_type), a declared return
// type the value returned (zend_verify_return_type). Without strict_types a
// scalar coerces as for a typed property: a numeric string or a bool to int or
// float, a whole float to int, a number or a bool to string, a scalar to bool,
// an int|string union keeping a string and taking a bool as int; null and arrays
// never coerce. The TypeError reads "f(): Argument #n ($p) must be of type T, U
// given, called in F on line L" and "f(): Return value must be of type T, U
// returned", U being "none" for a function that ends without a return; self and
// parent show as class names. A default spelled null makes the type nullable. A
// generator checks its arguments at the call, and its return value is no
// declared type's. A void function is not checked. A callable takes a closure, a
// function's name or an object with __invoke.
function s78int(int $a) { return $a; }
function s78flo(float $f) { return $f; }
function s78str(string $s) { return $s; }
function s78bool(bool $b) { return $b; }
function s78dn(int $x = null) { return $x; }
function s78uni(int|string $x) { return $x; }
function s78ret($x): int { return $x; }
function s78none(): int { }
function s78nret(): ?string { return null; }
function s78void(): void { return; }
function s78var(int ...$xs) { return $xs; }
function s78gvar(int ...$xs) { yield 1; }
function s78ref(int &$r) { $r = $r + 1; }
function s78gen(int $n): Generator { yield $n; return "r"; }
function s78cb(callable $c) { return $c(); }
class S78Inv { function __invoke() { return 8; } }
class S78P {
    function same(self $o): static { return $o; }
}
class S78K extends S78P {
    function up(parent $o): int { return 1; }
}
// The message with this file's path as F and line $l as L.
function s78e($f, $l) {
    try { $f(); return "none"; } catch (TypeError $e) {
        return str_replace(" on line " . $l, " on line L", str_replace(__FILE__, "F", $e->getMessage()));
    }
}
function s78() {
    check("sty1", s78int("5") === 5 && s78int(" 7") === 7 && s78int(5.0) === 5 && s78int(true) === 1 && s78int("1e3") === 1000
        && s78flo(3) === 3.0 && s78flo("2.5") === 2.5 && s78str(12) === "12" && s78str(1.5) === "1.5" && s78str(false) === ""
        && s78bool(0) === false && s78bool("a") === true && s78dn(null) === null && s78uni(1.0) === 1 && s78uni("1.5") === "1.5"
        && s78uni(true) === 1 && s78ret("7") === 7 && s78nret() === null && s78void() === null && s78var(1, "2", 3.0) === [1, 2, 3]);
    $m = [];
    $m[] = s78e(fn() => s78int("x"), __LINE__);
    $m[] = s78e(fn() => s78int(null), __LINE__);
    $m[] = s78e(fn() => s78int([]), __LINE__);
    $m[] = s78e(fn() => s78dn("x"), __LINE__);
    $m[] = s78e(fn() => s78var(1, "z"), __LINE__);
    $m[] = s78e(fn() => s78ret("q"), __LINE__);
    $m[] = s78e(fn() => s78none(), __LINE__);
    $m[] = s78e(fn() => s78gvar(1, "z"), __LINE__);
    check("sty2", $m === [
        "s78int(): Argument #1 (\$a) must be of type int, string given, called in F on line L",
        "s78int(): Argument #1 (\$a) must be of type int, null given, called in F on line L",
        "s78int(): Argument #1 (\$a) must be of type int, array given, called in F on line L",
        "s78dn(): Argument #1 (\$x) must be of type ?int, string given, called in F on line L",
        "s78var(): Argument #2 (\$xs) must be of type int, string given, called in F on line L",
        "s78ret(): Return value must be of type int, string returned",
        "s78none(): Return value must be of type int, none returned",
        "s78gvar(): Argument #2 (\$xs) must be of type int, string given, called in F on line L"]);
    $n = [];
    $n[] = s78e(fn() => (new S78P)->same(5), __LINE__);
    $n[] = s78e(fn() => (new S78K)->up(5), __LINE__);
    $n[] = s78e(fn() => s78gen("x"), __LINE__);
    $n[] = s78e(fn() => s78cb(5), __LINE__);
    $cl = __LINE__; $c = function (int $q): string { return $q; };
    $n[] = s78e(fn() => $c("w"), __LINE__);
    check("sty3", $n === [
        "S78P::same(): Argument #1 (\$o) must be of type S78P, int given, called in F on line L",
        "S78K::up(): Argument #1 (\$o) must be of type S78P, int given, called in F on line L",
        "s78gen(): Argument #1 (\$n) must be of type int, string given, called in F on line L",
        "s78cb(): Argument #1 (\$c) must be of type callable, int given, called in F on line L",
        "{closure:s78():" . $cl . "}(): Argument #1 (\$q) must be of type int, string given, called in F on line L"]
        && $c("4") === "4");
    $r = "5";
    s78ref($r);
    $g = s78gen("3");
    $gv = $g->current();
    $g->next();
    check("sty4", $r === 6 && $gv === 3 && $g->getReturn() === "r" && (new S78K)->same(new S78K) instanceof S78K
        && (new S78K)->up(new S78K) === 1 && s78cb(fn() => 3) === 3 && s78cb(new S78Inv) === 8 && s78cb('s78nret') === null);
}

// ===== SECTION 79: preg_* (PCRE) =====
// php-src ext/pcre/php_pcre.c over PCRE2 (NEWLINE LF; /u is UTF and UCP):
// the delimiters and modifiers, the matches array (groups up to the last that
// took part, a named group under its name too, PREG_OFFSET_CAPTURE pairs,
// PREG_UNMATCHED_AS_NULL), the global loops' retry after an empty match,
// $n / ${n} / \n references, PREG_SPLIT_* and the error codes. Expected values
// from libpcre2-8 driven with php_pcre.c's options and loops.
function s79d($v) {
    if ($v === null) { return "N"; }
    if ($v === false) { return "F"; }
    if (is_int($v)) { return "i:" . $v; }
    if (is_string($v)) {
        $o = "";
        $n = strlen($v);
        for ($i = 0; $i < $n; $i++) {
            $b = ord($v[$i]);
            $o .= ($b < 32 || $b >= 127) ? "%" . strtoupper(dechex($b)) : chr($b);
        }
        return "'" . $o . "'";
    }
    $parts = [];
    foreach ($v as $k => $x) { $parts[] = (is_int($k) ? $k : "'" . $k . "'") . "=>" . s79d($x); }
    return "[" . implode(",", $parts) . "]";
}
function s79cb($m) { return "<" . strtoupper($m[0]) . ">"; }
function s79() {
    check("pg1", preg_match('/(\d+)-(\d+)/', 'ab 12-34 x', $m) === 1 && s79d($m) === "[0=>'12-34',1=>'12',2=>'34']");
    check("pg2", preg_match('/(a)(b)?/', 'a', $m) === 1 && s79d($m) === "[0=>'a',1=>'a']");
    check("pg3", preg_match('/(a)(b)?/', 'a', $m, PREG_UNMATCHED_AS_NULL) === 1 && s79d($m) === "[0=>'a',1=>'a',2=>N]");
    check("pg4", preg_match('/(a)(b)?(c)/', 'ac', $m, PREG_OFFSET_CAPTURE) === 1
        && s79d($m) === "[0=>[0=>'ac',1=>i:0],1=>[0=>'a',1=>i:0],2=>[0=>'',1=>i:-1],3=>[0=>'c',1=>i:1]]");
    check("pg5", preg_match('/(?<y>\d{4})-(?<m>\d\d)/', 'on 2024-10-20', $m) === 1
        && s79d($m) === "[0=>'2024-10','y'=>'2024',1=>'2024','m'=>'10',2=>'10']");
    check("pg6", preg_match('/(?<y>\d{4})-(?<m>\d\d)?/', 'on 2024-', $m, PREG_UNMATCHED_AS_NULL) === 1
        && s79d($m) === "[0=>'2024-','y'=>'2024',1=>'2024','m'=>N,2=>N]");
    check("pg7", preg_match('/(?P<w>\w+)(?\'z\'!)?/', 'hi', $m) === 1 && s79d($m) === "[0=>'hi','w'=>'hi',1=>'hi']");
    check("pg8", preg_match('/x/', 'abc', $m) === 0 && $m === []);
    check("pg9", preg_match('/b/', 'abcb', $m, PREG_OFFSET_CAPTURE, 2) === 1 && $m[0][1] === 3
        && preg_match('/b/', 'abcb', $m, PREG_OFFSET_CAPTURE, -1) === 1 && $m[0][1] === 3
        && preg_match('/^b/', 'abcb', $m, 0, 1) === 0 && preg_match('/\Gb/', 'abcb', $m, 0, 1) === 1
        && preg_match('/(?<=a)b/', 'abcb', $m, 0, 1) === 1);
    check("pg10", preg_match('/b/', 'abcb', $m, 0, 5) === false && preg_last_error() === PREG_INTERNAL_ERROR);
    // Without u a subject is bytes; with u it is UTF-8 characters, and offsets
    // stay byte offsets.
    check("pg11", preg_match('/^.$/', "\xC3\xA9") === 0 && preg_match('/^.$/u', "\xC3\xA9") === 1
        && preg_match('/^..$/', "\xC3\xA9") === 1);
    check("pg12", preg_match('/\w+/u', "h\xC3\xA9llo w", $m) === 1 && $m[0] === "h\xC3\xA9llo"
        && preg_match('/\w+/', "h\xC3\xA9llo w", $m) === 1 && $m[0] === "h");
    check("pg13", preg_match('/l+/u', "h\xC3\xA9llo", $m, PREG_OFFSET_CAPTURE) === 1 && $m[0][1] === 3);
    check("pg14", preg_match('/./u', "a\xE9b") === false && preg_last_error() === PREG_BAD_UTF8_ERROR
        && preg_last_error_msg() === "Malformed UTF-8 characters, possibly incorrectly encoded");
    check("pg15", preg_match('/./u', "\xC3\xA9a", $m, 0, 1) === false && preg_last_error() === PREG_BAD_UTF8_OFFSET_ERROR
        && preg_match('/./u', "\xC3\xA9a", $m, 0, 2) === 1 && $m[0] === "a" && preg_last_error() === PREG_NO_ERROR);
    check("pg16", preg_match('/\p{Lu}+/u', "abC\xC3\x89Dx", $m) === 1 && $m[0] === "C\xC3\x89D"
        && preg_match('/[[:alpha:]]+/', "12ab\xC3\xA9", $m) === 1 && $m[0] === "ab"
        && preg_match('/[[:alpha:]]+/u', "12ab\xC3\xA9", $m) === 1 && $m[0] === "ab\xC3\xA9");
    check("pg17", preg_match('/\x{e9}/iu', "\xC3\x89") === 1 && preg_match('/\xe9/i', "\xC9") === 0);
    // Delimiters and modifiers; a pattern PHP refuses is false and PREG_INTERNAL_ERROR.
    check("pg18", preg_match('{a}', 'a') === 1 && preg_match('(a(b))', 'ab') === 1 && preg_match('[a]i', 'A') === 1
        && preg_match('<a<b>>', 'a<b>') === 1 && preg_match('#a\#b#', 'a#b') === 1 && preg_match("  /a/  ", 'a') === 1);
    check("pg19", preg_match('abc', 'x') === false && preg_last_error() === PREG_INTERNAL_ERROR
        && preg_last_error_msg() === "Internal error" && preg_match('/abc', 'x') === false
        && preg_match('/abc/k', 'x') === false && preg_match('', 'x') === false && preg_match('/a/e', 'a') === false);
    check("pg20", preg_match('/(abc/', 'x') === false && preg_match('/[a/', 'x') === false
        && preg_match('/a**/', 'x') === false && preg_match('/a{2,1}/', 'x') === false
        && preg_match('/(?<n>a)(?<n>b)/', 'ab') === false && preg_match('/(a)\2/', 'aa') === false
        && preg_match('/[a-\d]/', 'x') === false && preg_match('/(?<=a+)b/', 'ab') === false);
    check("pg21", preg_match('/a$/', "a\n") === 1 && preg_match('/a$/D', "a\n") === 0 && preg_match('/a$/m', "a\nb") === 1
        && preg_match('/^b/m', "a\nb") === 1 && preg_match('/a.b/', "a\nb") === 0 && preg_match('/a.b/s', "a\nb") === 1
        && preg_match('/a.b/', "a\rb") === 1 && preg_match('/a\Rb/', "a\r\nb") === 1);
    check("pg22", preg_match('/a+/U', 'aaa', $m) === 1 && $m[0] === "a" && preg_match('/a+?/U', 'aaa', $m) === 1 && $m[0] === "aaa");
    check("pg23", preg_match("/a b # c\n c/x", 'abc') === 1 && preg_match('/(a)(?:b)(c)/n', 'abc', $m) === 1 && $m === ["abc"]
        && preg_match('/b/A', 'abc') === 0 && preg_match('/b/A', 'abc', $m, 0, 1) === 1);
    check("pg24", preg_match('/(?i:a)b/', 'Ab') === 1 && preg_match('/(?i:a)b/', 'AB') === 0 && preg_match('/(?^i:A)/', 'a') === 1
        && preg_match('/(a(?i)b)c/', 'aBc') === 1 && preg_match('/(a(?i)b)c/', 'aBC') === 0);
    check("pg25", preg_match('/(?<n>a)|(?<n>b)/J', 'b', $m) === 1 && s79d($m) === "[0=>'b','n'=>'b',1=>'',2=>'b']");
    check("pg26", preg_match('/(?<x>a)\k<x>\k\'x\'\k{x}(?P=x)\g{x}\g1\g{-1}/', 'aaaaaaaa') === 1
        && preg_match('/(a)?\1/', 'b') === 0 && preg_match('/\101\x42\o{103}\cD/', "ABC\x04") === 1);
    check("pg27", preg_match('/a{,2}b/', 'aaab', $m) === 1 && $m[0] === "aab" && preg_match('/x{2,a}/', 'x{2,a}') === 1
        && preg_match('/[]a]+/', ']a]b', $m) === 1 && $m[0] === "]a]" && preg_match('/[[]+/', '[[a') === 1
        && preg_match('/\Qa.b\E+/', 'a.bbb', $m) === 1 && $m[0] === "a.bbb");
    check("pg28", preg_match('/a\Kb/', 'ab', $m, PREG_OFFSET_CAPTURE) === 1 && s79d($m) === "[0=>[0=>'b',1=>i:1]]");
    // preg_match_all: PREG_PATTERN_ORDER pads every group, PREG_SET_ORDER
    // is preg_match's shape per match; an empty match is retried non-empty.
    check("pg29", preg_match_all('/(?<d>\d)(x)?/', 'a1b2x', $m) === 2
        && s79d($m) === "[0=>[0=>'1',1=>'2x'],'d'=>[0=>'1',1=>'2'],1=>[0=>'1',1=>'2'],2=>[0=>'',1=>'x']]");
    check("pg30", preg_match_all('/(?<d>\d)(x)?/', 'a1b2x', $m, PREG_SET_ORDER) === 2
        && s79d($m) === "[0=>[0=>'1','d'=>'1',1=>'1'],1=>[0=>'2x','d'=>'2',1=>'2',2=>'x']]");
    check("pg31", preg_match_all('/a*/', 'baaa', $m, PREG_OFFSET_CAPTURE) === 3
        && s79d($m) === "[0=>[0=>[0=>'',1=>i:0],1=>[0=>'aaa',1=>i:1],2=>[0=>'',1=>i:4]]]");
    check("pg32", preg_match_all('/a*?/', 'aa', $m) === 5 && preg_match_all('//u', "\xC3\xA9\xF0\x9F\x98\x80", $m, PREG_OFFSET_CAPTURE) === 3
        && $m[0][2][1] === 6 && preg_match_all('//', "\xC3\xA9") === 3);
    check("pg33", preg_match_all('/\d/', 'a1b2', $m, 0, 2) === 1 && preg_match_all('/(?<n>z)/', 'abc', $m) === 0
        && s79d($m) === "[0=>[],'n'=>[],1=>[]]" && preg_match_all('/\Gb/', 'bbab') === 2);
    check("pg34", preg_match_all('/(a)|(b)/', 'ab', $m, PREG_UNMATCHED_AS_NULL) === 2
        && s79d($m) === "[0=>[0=>'a',1=>'b'],1=>[0=>'a',1=>N],2=>[0=>N,1=>'b']]");
    // preg_replace and its references; a backslash before \ or $ replaces it.
    check("pg35", preg_replace('/a/', 'b', 'aa') === "bb" && preg_replace('/a/', 'b', 'aa', 1) === "ba"
        && preg_replace('/a/', 'b', 'aa', 0) === "aa");
    check("pg36", preg_replace('/(\w+) (\w+)/', '$2 ${1}!', 'hello world') === "world hello!"
        && preg_replace('/(\w+) (\w+)/', '\2 \1 \0 $0 ${0}', 'hello world') === "world hello hello world hello world hello world");
    check("pg37", preg_replace('/(a)(b)?/', '[$2|\2|${2}|$3|$10|$1x]', 'a') === "[|||||ax]"
        && preg_replace('/(a)/', '\\\\1 \\\\\1 $$1 \$1 \\\\$1 $ \ ${1 ${x}', 'a') === '\1 \a $a $1 \a $ \ ${1 ${x}');
    check("pg38", preg_replace('/(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)/', '$11-$10-${11}-$111', 'abcdefghijk') === "k-j-k-k1");
    check("pg39", preg_replace('/x*/', '-', 'abc') === "-a-b-c-" && preg_replace('/x*/', '-', 'abc', 2) === "-a-bc"
        && preg_replace('/a|/', 'X', 'bab') === "XbXXbX" && preg_replace('//u', '|', "\xC3\xA9") === "|\xC3\xA9|"
        && preg_replace('//', '|', "ab") === "|a|b|");
    check("pg40", preg_replace('/(\d)(?=(\d{3})+$)/', '$1,', '1234567') === "1,234,567"
        && preg_replace('/^/m', '> ', "a\nb") === "> a\n> b" && preg_replace('/a\Kb/', 'X', 'abab') === "aXaX");
    check("pg41", preg_replace(['/a/', '/b/'], ['b', 'c'], 'ab', -1, $n) === "cc" && $n === 3
        && preg_replace(['/a/', '/b/'], 'x', ['k' => 'ab', 'cd']) === ['k' => 'xx', 0 => 'cd']
        && preg_replace(['/a/', '/b/'], ['1'], 'ab') === "1" && preg_replace('/[/', 'x', 'abc') === null);
    check("pg42", preg_filter('/\d/', '#', ['a1', 'b', 'c2']) === [0 => 'a#', 2 => 'c#'] && preg_filter('/\d/', '#', 'b') === null);
    check("pg43", preg_replace_callback('/\d+/', fn($m) => $m[0] * 2, 'a1 b22', -1, $n) === "a2 b44" && $n === 2
        && preg_replace_callback('/x*/', 's79cb', 'ab') === "<>a<>b<>" && preg_replace_callback('/\d/', 's79cb', '1234', 2) === "<1><2>34");
    $seen = [];
    preg_replace_callback('/(?<n>\d)(?<m>x)?/', function ($m) use (&$seen) { $seen[] = s79d($m); return ""; }, 'a1x2', -1, $n,
        PREG_OFFSET_CAPTURE | PREG_UNMATCHED_AS_NULL);
    check("pg44", $seen === ["[0=>[0=>'1x',1=>i:1],'n'=>[0=>'1',1=>i:1],1=>[0=>'1',1=>i:1],'m'=>[0=>'x',1=>i:2],2=>[0=>'x',1=>i:2]]",
        "[0=>[0=>'2',1=>i:3],'n'=>[0=>'2',1=>i:3],1=>[0=>'2',1=>i:3],'m'=>[0=>N,1=>i:-1],2=>[0=>N,1=>i:-1]]"]);
    check("pg45", preg_replace_callback_array(['/\d+/' => fn($m) => 'N', '/[a-z]+/' => fn($m) => 'w'], 'ab 12 c', -1, $n) === "w N w"
        && $n === 3);
    // preg_split
    check("pg46", preg_split('/[\s,]+/', "hypertext language, programming") === ["hypertext", "language", "programming"]
        && preg_split('//', 'abc', -1, PREG_SPLIT_NO_EMPTY) === ["a", "b", "c"] && preg_split('//', 'ab') === ["", "a", "b", ""]);
    check("pg47", s79d(preg_split('/(-)|(\+)/', 'a-b+c', -1, PREG_SPLIT_DELIM_CAPTURE)) === "[0=>'a',1=>'-',2=>'b',3=>'',4=>'+',5=>'c']"
        && s79d(preg_split('/(-)|(\+)/', 'a-b+c', -1, PREG_SPLIT_DELIM_CAPTURE | PREG_SPLIT_OFFSET_CAPTURE))
            === "[0=>[0=>'a',1=>i:0],1=>[0=>'-',1=>i:1],2=>[0=>'b',1=>i:2],3=>[0=>'',1=>i:-1],4=>[0=>'+',1=>i:3],5=>[0=>'c',1=>i:4]]");
    check("pg48", preg_split('/-/', 'a-b-c', 2) === ["a", "b-c"] && preg_split('/-/', 'a-b-c', 1) === ["a-b-c"]
        && preg_split('/-/', 'a-b-c', 0) === ["a", "b", "c"] && preg_split('/-/', 'a-b-c', -2) === ["a-b-c"]
        && preg_split('/-/', '-a--b-', 2, PREG_SPLIT_NO_EMPTY) === ["a", "-b-"]);
    check("pg49", preg_split('/\d*/', 'a1b') === ["", "a", "", "b", ""] && preg_split('/\d*/', 'a1b', 2) === ["", "a1b"]
        && preg_split('//u', "\xC3\xA9\xE2\x82\xAC", -1, PREG_SPLIT_NO_EMPTY) === ["\xC3\xA9", "\xE2\x82\xAC"]
        && preg_split('/,/', '') === [""] && preg_split('/,/', '', -1, PREG_SPLIT_NO_EMPTY) === []
        && preg_split('/[/', 'a') === false && preg_split('/,/u', "a,\xFF") === false);
    // preg_quote, preg_grep
    check("pg50", preg_quote('Hello.World?(x) {a} [b] <c> | : - # = !') === 'Hello\.World\?\(x\) \{a\} \[b\] \<c\> \| \: \- \# \= \!'
        && preg_quote("a/b#c", "/") === "a\\/b\\#c" && preg_quote("a\0b") === "a\\000b" && preg_quote("h\xC3\xA9llo.") === "h\xC3\xA9llo\\.");
    check("pg51", preg_grep('/^\d+$/', ['1', 'a', 'x' => '22']) === [0 => '1', 'x' => '22']
        && preg_grep('/^\d+$/', ['1', 'a', 'x' => '22'], PREG_GREP_INVERT) === [1 => 'a']);
    // Recursion and subroutine calls: a call's captures are undone when it
    // returns; DEFINE groups are matched only through calls.
    check("pg52", preg_match('/\((?:[^()]|(?R))*\)/', 'x(a(b)c)y', $m) === 1 && $m[0] === '(a(b)c)'
        && preg_match('/^((.)(?1)\2|.?)$/', 'abcba', $m) === 1 && s79d($m) === "[0=>'abcba',1=>'abcba',2=>'a']"
        && preg_match('/(?<p>\[(?:[^\[\]]|(?&p))*\])/', 'z[a[b]]', $m) === 1 && $m[0] === '[a[b]]'
        && preg_match('/(a|b\g<1>)/', 'bba', $m) === 1 && $m[1] === 'bba'
        && preg_match('/(\d)(?1)/', '12', $m) === 1 && s79d($m) === "[0=>'12',1=>'1']"
        && preg_match('/(?(DEFINE)(?<d>\d+)(?<op>[+-]))^(?&d)(?:(?&op)(?&d))*$/', '12+3-4') === 1);
    // Conditionals on a group, a name, an assertion and recursion.
    check("pg53", preg_match('/(a)?(?(1)b|c)/', 'ab', $m) === 1 && s79d($m) === "[0=>'ab',1=>'a']"
        && preg_match('/(a)?(?(1)b|c)/', 'c', $m) === 1 && $m === ['c'] && preg_match('/^(a)?(?(1)b|c)$/', 'ac') === 0
        && preg_match('/(?<q>")?\w+(?(q)")/', '"ab"', $m) === 1 && $m[0] === '"ab"'
        && preg_match('/(?(?=\d)\d\d|[a-z])/', '5x', $m) === 1 && $m[0] === 'x'
        && preg_match('/(?(R)a|b)/', 'b') === 1 && preg_match('/(?(1)a|b|c)/', 'a') === false);
    // A branch reset numbers each branch's groups from the same point.
    check("pg54", preg_match('/(?|(a)(x)|(b))(y)/', 'by', $m) === 1 && s79d($m) === "[0=>'by',1=>'b',2=>'',3=>'y']"
        && preg_match('/(?|(?<n>a)|(?<n>b))/', 'b', $m) === 1 && s79d($m) === "[0=>'b','n'=>'b',1=>'b']");
    // The backtracking verbs, with PCRE2's first code unit optimization.
    check("pg55", preg_match('/a+(*COMMIT)b/', 'aaac aaab') === 0 && preg_match('/aa(*SKIP)b|a+c/', 'aaac', $m) === 1
        && $m[0] === 'ac' && preg_match('/(a(*THEN)b|ac)/', 'ac', $m) === 1 && $m[0] === 'ac'
        && preg_match('/a(*ACCEPT)b/', 'ac', $m) === 1 && $m[0] === 'a' && preg_match('/(*COMMIT)abc/', 'xyzabc') === 1
        && preg_match('/(*COMMIT)[ab]c/', 'xbc') === 0 && preg_replace('/(*COMMIT)/', '<$0>', 'a1b2') === "<>a<>1<>b<>2<>");
    // Pattern start options, callouts, (?xx), and caseless \p{Lu} against
    // [[:lower:]].
    check("pg56", preg_match('/(*UTF)^.$/', "\xC3\xA9") === 1 && preg_match('/(*CRLF)a$/', "a\r\n") === 1
        && preg_match('/a(?C1)b/', 'ab') === 1 && preg_match('/(?xx)[a b]+/', 'a b', $m) === 1 && $m[0] === 'a'
        && preg_match('/\p{Lu}/iu', 'a') === 1 && preg_match('/[[:lower:]]/iu', 'B') === 0);
    // The step limit is PREG_BACKTRACK_LIMIT_ERROR; a recursion that cannot
    // advance PCRE2's internal error.
    check("pg57", preg_match('/(?:\D+|<\d+>)*[!?]/', 'foobar foobar foobar') === false
        && preg_last_error() === PREG_BACKTRACK_LIMIT_ERROR && preg_last_error_msg() === "Backtrack limit exhausted"
        && preg_match('/(?R)/', 'a') === false && preg_last_error() === PREG_INTERNAL_ERROR);
    // An array where a string parameter stands is a TypeError.
    $te = "";
    try { preg_match('/a/', ['x']); } catch (TypeError $e) { $te = $e->getMessage(); }
    check("pg58", $te === 'preg_match(): Argument #2 ($subject) must be of type string, array given');
    // Linear work on a long subject is never cut off by the step limit.
    $lng = str_repeat("ab", 20000);
    check("pg59", preg_match('/(?:t|u|v|w|x|y|a|b){2,}/', $lng, $m) === 1 && strlen($m[0]) === 40000);
    // A pathological pattern on a long subject still stops at the step limit.
    $as = str_repeat("a", 2000) . "!";
    check("pg60", preg_match('/(a+)+$/', $as) === false && preg_last_error() === PREG_BACKTRACK_LIMIT_ERROR
        && preg_match('/(a|aa)+$/', $as) === false && preg_last_error() === PREG_BACKTRACK_LIMIT_ERROR);
    // PCRE2 compiles x+ as one loop whose iterations stop at an empty one, and
    // x{n,m} as copies of x, which do not.
    check("pg61", preg_match('/((?(1)a|))+/', 'aaa', $m) === 1 && $m === ['', '']
        && preg_match('/((?(1)a|)){1,3}/', 'aaa', $m) === 1 && $m === ['aa', 'a']
        && preg_match('/((?(1)a|)){2,}/', 'aaa', $m) === 1 && $m === ['aaa', 'a']
        && preg_match('/((?(1)a|)){2,}?b/', 'aaab', $m) === 1 && $m === ['aaab', 'a']);
    // (*ACCEPT) in an atomic group ends the match; in a subroutine called from
    // an assertion it returns from the call.
    check("pg62", preg_match('/(?>a(*ACCEPT)b)c/', 'abc', $m) === 1 && $m === ['a']
        && preg_match('/(?!(?>a(*ACCEPT)b))a/', 'ax') === 0
        && preg_match('/(?=(?1)c)(a(*ACCEPT)b)/', 'ac', $m) === 1 && $m === ['a', 'a']
        && preg_match('/x(?>(?1))y|(a(*ACCEPT)b)/', 'xay', $m) === 1 && $m === ['xay']);
    // The latest (*MARK) on the matching path is $matches['MARK'].
    $mk = "";
    preg_replace_callback('/(*MARK:x)a|b/', function ($m) use (&$mk) { $mk .= $m['MARK'] ?? "-"; return ""; }, 'ab');
    check("pg63", preg_match('/a(*:x)b|c(*:y)d/', 'cd', $m) === 1 && $m === ['cd', 'MARK' => 'y']
        && preg_match_all('/(*MARK:x)a|(*MARK:y)b|c/', 'abc', $m) === 3 && $m['MARK'] === ['x', 'y']
        && preg_match_all('/(*MARK:x)a|c/', 'ac', $m, PREG_SET_ORDER) === 2 && $m === [['a', 'MARK' => 'x'], ['c']]
        && preg_match('/(?=(*MARK:x)a)a/', 'a', $m) === 1 && $m['MARK'] === 'x'
        && preg_match('/(?!(*MARK:x)b)a/', 'a', $m) === 1 && $m === ['a']
        && preg_match('/(*MARK:x)(?:a(*MARK:y)|b)c/', 'bc', $m) === 1 && $m['MARK'] === 'x' && $mk === "x-");
    // \p names as PCRE2 10.48 reads them: case, spaces, hyphens and
    // underscores ignored; no long general category names; scripts by any
    // name; binary properties, which /i leaves alone.
    check("pg64", preg_match('/\p{Lowercase_Letter}/u', 'a') === false && preg_last_error() === PREG_INTERNAL_ERROR
        && preg_match('/\p{gc=Lu}/u', 'A') === false && preg_match('/\p{Lowercase}/iu', 'A') === 0
        && preg_match('/\p{l o-w_e r}\p{LU}/u', 'aB') === 1 && preg_match('/\p{Ital}\p{sc=Grek}/u', "\u{10300}\u{3B1}") === 1
        && preg_match('/\p{olditalic}\p{Script=Greek}/u', "\u{10300}\u{3B1}") === 1 && preg_match('/\p{ahex}+/u', 'aF9g', $m) === 1
        && $m[0] === 'aF9' && preg_match('/\p{White_Space}\p{ascii}/u', " a") === 1);
    // pcre2_match tries no start that leaves fewer units than the shortest
    // match, where a branch recursing into its own group counts only if it is
    // the first; and no first-character test when an (*ACCEPT) may come first.
    check("pg65", preg_match('/xyz|(?R)ab/', 'ab') === 0 && preg_last_error() === PREG_NO_ERROR
        && preg_match('/aaa|(?R)/', 'b') === false && preg_match_all('/(*ACCEPT)(?>(*PRUNE)x)/', 'abab') === 5);
}

// ===== SECTION 80: word operators, promoted hooks, nullsafe dynamic names, recursive var_dump =====
// 'and' binds tighter than 'xor', which binds tighter than 'or' (all three looser
// than '='), each answering a bool by PHP truthiness: "0", 0.0 and [] are false.
// php-src's constexpr/constant_expressions EXPECT gives T_15 = 1 xor 1 as
// bool(false) and T_16 = 1 xor 0 as bool(true).
const S80X1 = 1 xor 1;
const S80X2 = 1 xor 0;
const S80X3 = 1 xor 1 or 1;
const S80X4 = 1 xor 1 and 0;
trait S80TA { function hi() { return "A"; } }
trait S80TB { function hi() { return "B"; } }
// A trait adaptation may name the trait fully qualified (grammar/semi_reserved_006).
class S80U { use S80TA, S80TB { \S80TA::hi insteadof S80TB; \S80TB::hi as bhi; } }
// A promoted constructor parameter may carry hooks; promotion writes through
// the set hook (property_hooks/property_promotion: "set(42)" then "get").
class S80H {
    public function __construct(
        public int $x = 3 { set => $value * 2; get => $this->x + 1; },
        public $log = "",
    ) {}
}
function s80() {
    $vs = ["0", 0.0, [], "", "a", 1];
    $got = "";
    foreach ($vs as $v) {
        $got .= ($v || false ? "1" : "0") . ($v && true ? "1" : "0") . (($v or false) ? "1" : "0") . (($v and true) ? "1" : "0");
    }
    check("wo1", $got === "0000000000000000" . "11111111");
    check("wo2", S80X1 === false && S80X2 === true && S80X3 === true && S80X4 === true);
    $a = true xor true;
    check("wo3", $a === true && (true xor true) === false && (0 xor "a") === true);
    $n = 0;
    $r = (false xor ($n = 5));
    check("wo4", $r === true && $n === 5);
    $u = new S80U();
    check("tq1", $u->hi() === "A" && $u->bhi() === "B");
    $h = new S80H(5);
    check("ph1", $h->x === 11);
    $h->x = 1;
    check("ph2", $h->x === 3 && (new S80H())->x === 7);
    $o = new stdClass;
    $o->foobar = 7;
    $s = "foo";
    $z = null;
    check("ns1", $o?->{$s . "bar"} === 7 && $z?->{$s . "bar"} === null && $z?->{$s}() === null);
}
// var_dump of an array that holds a reference to itself prints *RECURSION* where
// it re-enters that array, and a reference slot held twice prints &: the lines
// below are php-src's tests/bug35163, bug35163_2 and bug35163_3 EXPECT blocks.
function s80dump() {
    $a = array(array(1));
    $a[0][] =& $a[0];
    $a[0][] =& $a[0];
    $a[0][0] = 2;
    var_dump($a);
    $b = array(1);
    $b[] =& $b;
    $b[] =& $b;
    $b[0] = 2;
    var_dump($b);
    $c = new stdClass;
    $c->b = array(1);
    $c->b[] =& $c->b;
    $c->b[] =& $c->b;
    $c->b[0] = 2;
    var_dump($c);
}

// ===== SECTION 81: count(COUNT_RECURSIVE) and print_r of a self-referencing array =====
// COUNT_RECURSIVE adds the count of every nested array (php.net count); an array
// reached again through a reference while it is counted adds nothing
// (php_count_recursive), and print_r prints " *RECURSION*" after its header
// (zend_print_zval_r_to_buf). A Countable's count() takes no mode.
class S81C implements Countable { public function count(): int { return 42; } }
function s81() {
    check("cr1", count([1, [2, 3]], COUNT_RECURSIVE) === 4);
    check("cr2", count([1, [2, [4, 5]], 3], COUNT_RECURSIVE) === 7 && sizeof([[1], [2, [3]]], 1) === 6
        && count([1, [2, 3]], COUNT_NORMAL) === 2);
    $msg = "";
    try { count([1], 2); } catch (ValueError $e) { $msg = $e->getMessage(); }
    check("cr3", $msg === "count(): Argument #2 (\$mode) must be either COUNT_NORMAL or COUNT_RECURSIVE");
    check("cr4", count(new S81C(), COUNT_RECURSIVE) === 42 && count([new S81C(), [1]], COUNT_RECURSIVE) === 3);
    $x = [1, [2]];
    $x[1][] = &$x[1];
    $y = [1];
    $y[] = &$y;
    check("cr5", count($x, COUNT_RECURSIVE) === 4 && count($y, COUNT_RECURSIVE) === 2);
    check("pr1", print_r($x, true) === "Array\n(\n    [0] => 1\n    [1] => Array\n        (\n            [0] => 2\n"
        . "            [1] => Array\n *RECURSION*\n        )\n\n)\n");
}

// ===== SECTION 82: a reference's holders, counted down =====
// An array copy keeps an element that is a reference only while another slot
// holds it, and makes it a plain value otherwise (zend_array_dup_element). A
// holder goes when a frame returns (its variables), when a by-reference
// parameter, a captured variable or a static is bound elsewhere with '=&', and
// when an array literal is copied into a variable.
function s82local() { $x = 1; $a = [&$x]; return $a; }
function s82rebind(&$p) { $o = 0; $p = &$o; }
function s82static($n) {
    static $s = 1;
    if ($n == 1) {
        $o = 0;
        $s = &$o;
        return null;
    }
    $k = [];
    $k[0] = &$s;
    $c = $k;
    $c[0] = 8;
    return $s;
}
function s82() {
    $r = s82local();
    $r2 = $r;
    $r2[0] = 9;
    check("rc1", $r[0] === 1);
    $x = 1;
    $arr = [];
    $arr[0] = &$x;
    s82rebind($x);
    $b = $arr;
    $b[0] = 9;
    check("rc2", $x === 9);
    $y = 1;
    $ay = [];
    $ay[0] = &$y;
    $fy = function () use (&$y) { $o = 0; $y = &$o; };
    $fy();
    $fy();
    $by = $ay;
    $by[0] = 7;
    check("rc3", $y === 7);
    $z = 1;
    $az = [&$z];
    unset($z);
    $bz = $az;
    $bz[0] = 5;
    check("rc4", $az[0] === 1);
    s82static(1);
    check("rc5", s82static(2) === 8);
}

// ===== SECTION 83: return by reference; a static property takes a copy =====
// A function or method declared 'function &f()' returns the variable its return
// names, and '$x = &f()' binds $x to it; a returned element or property is
// fetched for writing, so it is created when missing. Any other right side of
// '= &', a call to a function not returning by reference included, is assigned
// by value (ZEND_ASSIGN_REF of a function result, ZEND_RETURN_BY_REF). Assigning
// an array to a static property copies it, as any assignment does.
function &s83counter() { static $c = 0; $c++; return $c; }
function &s83global() { global $s83g; return $s83g; }
function s83plain() { global $s83g; return $s83g; }
function &s83slot(array &$a, $k) { return $a[$k]["in"]; }
class S83 {
    public static $p;
    public $items = ["a" => 1];
    public static function &shared() { static $v = 0; return $v; }
    public function &item($k) { return $this->items[$k]; }
    public function &all() { return $this->items; }
}
function s83() {
    global $s83g;
    $c1 = &s83counter();
    $c1 += 100;
    check("rr1", s83counter() === 102);
    $s83g = 1;
    $h = &s83global();
    $h = 3;
    check("rr2", $s83g === 3);
    $p = &s83plain();
    $p = 99;
    check("rr3", $s83g === 3);
    $v = &S83::shared();
    $v = 10;
    check("rr4", S83::shared() === 10);
    $o = new S83;
    $it = &$o->item("b");
    $it = "B";
    check("rr5", $o->items["b"] === "B");
    $o->item("c");
    check("rr6", array_key_exists("c", $o->items) && $o->items["c"] === null);
    $all = &$o->all();
    $all["z"] = 26;
    $cp = $o->all();
    $cp["y"] = 25;
    check("rr7", count($o->items) === 4 && !isset($o->items["y"]));
    $arr = [];
    $d = &s83slot($arr, "k");
    $d = "deep";
    check("rr8", $arr["k"]["in"] === "deep");
    $a = [1];
    S83::$p = $a;
    $a[0] = 2;
    S83::$p[] = 5;
    check("rr9", S83::$p[0] === 1 && count($a) === 1);
    $n = "dyn";
    $x = 1;
    $$n = &$x;
    $dyn = 5;
    check("rr10", $x === 5);
}

// ===== SECTION 84: reference holders released by unwinding and by computed names =====
// A frame an exception unwinds is destroyed like one that returns: its
// variables stop holding their references, also when a finally's return then
// drops the exception. So does a variable whose name is computed ('$$n = &$x',
// 'global $$k'). An array copy then makes an element held by nothing else a
// plain value again (zend_array_dup_element, i_free_compiled_variables).
$s84 = [1, 1, 1, 1];
function s84deep($i) { global $s84; $q = &$s84[$i]; throw new Exception("u"); }
function s84mid($i) { s84deep($i); }
function s84swallow($i) { try { s84deep($i); } finally { return 1; } }
function s84catch() {
    global $s84;
    $own = &$s84[3];
    try { s84deep(3); } catch (Exception $e) { }
    $copy = $s84;
    $copy[3] = 2;
    return $s84[3];
}
function s84dyn() { global $s84; $n = "e"; $$n = &$s84[1]; }
function s84glob() { $k = "s84g"; global $$k; }
function s84() {
    global $s84;
    try { s84mid(0); } catch (Exception $e) { }
    s84swallow(2);
    s84dyn();
    $c = $s84;
    $c[0] = 5;
    $c[1] = 6;
    $c[2] = 7;
    check("rc6", $s84[0] === 1 && $s84[1] === 1 && $s84[2] === 1);
    check("rc7", s84catch() === 2);
    $GLOBALS["s84g"] = 1;
    s84glob();
    $a = [];
    $a[0] = &$GLOBALS["s84g"];
    unset($GLOBALS["s84g"]);
    $b = $a;
    $b[0] = 9;
    check("rc8", $a[0] === 1);
}

// ===== SECTION 85: generators suspend, step by their methods, and refuse to rewind =====
// A yield inside a try, catch or finally body makes its function a generator
// like any other. The body is SUSPENDED at each yield: its frame keeps its
// variables, and a reference one of them holds, while the generator is parked.
// next() and send() first run a fresh body to its first yield, so next() lands
// on the second and send(v) gives v to the first (Generator::next, ::send). A
// foreach takes the element a generator is parked at, and throws when it is
// finished ("Cannot traverse an already closed generator") or was resumed past
// its first yield ("Cannot rewind a generator that was already run",
// zend_generator_get_iterator, zend_generator_rewind); getReturn() of a body
// that has not returned throws as well (Generator::getReturn).
function s85thrower() { throw new Exception("t"); }
function s85incatch() { try { s85thrower(); } catch (Exception $e) { yield 5; } }
function s85intry() { try { yield 1; } finally { echo ""; } }
function s85three() { $x = yield 1; $y = yield 2; yield $x . "," . $y; }
function s85ref() { global $s85z; $r = &$s85z[0]; yield 1; yield 2; }
function s85err($g, $how) {
    try {
        if ($how == 0) { foreach ($g as $v) { } }
        else if ($how == 1) { $g->rewind(); }
        else { $g->getReturn(); }
    } catch (Exception $e) { return $e->getMessage(); }
    return "";
}
function s85() {
    global $s85z;
    $a = "";
    foreach (s85incatch() as $k => $v) { $a .= $k . "=" . $v; }
    foreach (s85intry() as $v) { $a .= ";" . $v; }
    check("gs1", $a === "0=5;1");
    $g = s85three();
    $g->next();
    $n = $g->current();
    $s = s85three();
    $first = $s->send("a");
    $s->next();
    check("gs2", $n === 2 && $first === 2 && $s->current() === "a,");
    $p = s85three();
    $p->current();
    $seen = "";
    foreach ($p as $k => $v) { $seen .= $k . $v; break; }
    foreach ($p as $k => $v) { $seen .= $k . (is_string($v) ? "s" : $v); }
    check("gs3", $seen === "0101122s");
    check("gs4", s85err($p, 0) === "Cannot traverse an already closed generator"
        && s85err($g, 1) === "Cannot rewind a generator that was already run"
        && s85err($g, 0) === "Cannot rewind a generator that was already run"
        && s85err(s85three(), 2) === "Cannot get return value of a generator that hasn't returned");
    $s85z = [1, 2];
    $gen = s85ref();
    $gen->current();
    $w = $s85z;
    $w[0] = 11;
    check("gs5", $s85z[0] === 11);
}

// ===== SECTION 86: variable variables passed by reference =====
// '$$n' and '${expr}' name a variable like '$name' does, so a by-reference
// parameter binds that variable, creating it when missing (ZEND_FETCH_W of a
// computed name, ZEND_SEND_REF).
function s86inc(&$x) { $x++; }
function s86() {
    $a = 1;
    $n = "a";
    s86inc($$n);
    s86inc(${"a"});
    $m = "fresh";
    s86inc($$m);
    $arr = [5];
    $an = "arr";
    s86inc(${$an}[0]);
    check("vr1", $a === 3 && $fresh === 1 && $arr[0] === 6);
}

// ===== SECTION 87: a by-value array parameter holds no reference after the call =====
// A by-value array parameter shares the caller's array until the callee writes
// it, and the copy that write separates is freed with the frame, so once the
// call has returned the references in the array have the holders they had
// before it; an array copy then makes one only its own slot holds a plain value
// (zend_array_dup_element). A copy the call returns, or that a closure it
// returns captured, lives on and keeps holding them.
function s87read($a) { return count($a); }
function s87write($a) { $a[] = 2; $a["n"][] = 3; return 0; }
function s87swap($a) { $a = [7]; return 0; }
function s87ret($a) { $a[] = 2; return $a; }
function s87clo($a) { $a[] = 1; return function () use ($a) { return count($a); }; }
function s87() {
    $x = 1;
    $ax = [&$x];
    s87read($ax);
    s87swap($ax);
    unset($x);
    $cx = $ax;
    $cx[0] = 9;
    $y = 1;
    $ay = [];
    $ay["n"][0] = &$y;
    s87write($ay);
    unset($y);
    $cy = $ay;
    $cy["n"][0] = 9;
    check("pc1", $ax[0] === 1 && $ay["n"][0] === 1);
    $z = 1;
    $az = [&$z];
    $kept = s87ret($az);
    $w = 1;
    $aw = [&$w];
    $k = s87clo($aw);
    unset($z, $w);
    $cz = $az;
    $cz[0] = 9;
    $cw = $aw;
    $cw[0] = 9;
    check("pc2", $az[0] === 9 && $aw[0] === 9);
}

// ===== SECTION 88: an array literal nested in another is no holder of its own =====
// A literal nested as an element of another is built straight into it, so its
// '&' elements are held by its slots and the referenced variables and nothing
// else; an array copy then makes one only its own slot holds a plain value
// (zend_array_dup_element).
function s88() {
    $a = 1;
    $b = 2;
    $m = [[[&$a]], "k" => [&$b, 5]];
    unset($a);
    $c = $m;
    $c[0][0][0] = 9;
    $c["k"][0] = 8;
    check("nl1", $m[0][0][0] === 1 && $m["k"][0] === 8 && $b === 8);
}

// ===== SECTION 89: Generator::throw, and what 'yield from' answers and forwards =====
// throw(e) first runs a fresh body to its first yield, raises e at the yield
// the body is parked at and answers the next current(); a finished generator
// throws e at the caller (Generator::throw, php-src
// Zend/tests/generators/throw_caught.phpt). Parked inside a 'yield from' over
// a generator, the body hands sends and throws on to it (zend_generator_get_current),
// and the expression's value is that generator's return value - null over an
// array (ZEND_YIELD_FROM).
function s89thr() { try { yield 1; } catch (Exception $e) { yield "c:" . $e->getMessage(); } yield 3; }
function s89fin() { global $s89log; try { yield 1; yield 2; } finally { $s89log .= "F"; } }
function s89inner() { $x = yield 1; $y = yield 2; return "r" . $x . "|" . $y; }
function s89outer() { $r = yield from s89inner(); yield $r; }
function s89catcher() { try { yield 1; yield 2; } catch (Exception $e) { yield "in:" . $e->getMessage(); return 5; } }
function s89deleg() { $r = yield from s89catcher(); yield "ret" . $r; }
function s89plain() { yield 1; yield 2; }
function s89deleg2() { try { yield from s89plain(); } catch (Exception $e) { yield "out:" . $e->getMessage(); } }
function s89arr() { $r = yield from [1, 2]; yield $r === null ? "n" : "v"; }
function s89() {
    global $s89log;
    $t = s89thr();
    $t->current();
    $a = $t->throw(new Exception("boom"));
    $t->next();
    $fresh = s89thr();
    check("gt1", $a === "c:boom" && $t->current() === 3 && $fresh->throw(new Exception("x")) === "c:x");
    $s89log = "";
    $f = s89fin();
    $f->current();
    $m = "";
    try { $f->throw(new Exception("u")); } catch (Exception $e) { $m .= $e->getMessage(); }
    try { $f->throw(new Exception("d")); } catch (Exception $e) { $m .= $e->getMessage(); }
    check("gt2", $m === "ud" && $s89log === "F" && !$f->valid());
    $s = "";
    foreach (s89outer() as $k => $v) { $s .= $k . "=" . $v . ";"; }
    foreach (s89arr() as $v) { $s .= $v; }
    check("gt3", $s === "0=1;1=2;0=r|;12n");
    $o = s89outer();
    $o->current();
    $sent = $o->send("a") . "," . $o->send("b");
    check("gt4", $sent === "2,ra|b");
    $d = s89deleg();
    $d->current();
    $in = $d->throw(new Exception("e"));
    $d->next();
    $d2 = s89deleg2();
    $d2->current();
    check("gt5", $in === "in:e" && $d->current() === "ret5" && $d2->throw(new Exception("q")) === "out:q");
}

// ===== SECTION 90: (array) of an object keeps its references; a running generator =====
// (array) copies an object's property table, and a property that some other
// slot also references stays that reference in the copy, while one nothing else
// references becomes its value (zend_proptable_to_symtable, zend_array_dup_element).
// unset() of a property, typed or not, stops it holding its reference.
// A generator whose body is running - it resumed itself, directly or through
// another generator - cannot be resumed: next() and send() throw an Error at the
// caller, while current() and key() answer the yield it last stopped at
// (zend_generator_resume).
class S90P { public $x = 1; private $y = 2; function refY() { $r = &$this->y; return (array)$this; } }
class S90U { public int $t = 1; public $d = 2; }
function s90self() { global $s90g; yield 1; try { $s90g->next(); } catch (Error $e) { yield get_class($e) . ":" . $e->getMessage(); } yield $s90g->current() . "@" . $s90g->key(); }
function s90send() { global $s90h; yield 1; $s90h->send(5); yield 2; }
function s90a() { global $s90b; yield 1; $s90b->next(); yield 2; }
function s90b() { global $s90a; yield 10; try { $s90a->next(); } catch (Error $e) { yield "b:" . $e->getMessage(); } }
function s90() {
    global $s90g, $s90h, $s90a, $s90b;
    $o = new stdClass;
    $o->a = 1;
    $r = &$o->a;
    $arr = (array)$o;
    $r = 5;
    $p = new stdClass;
    $p->arr = [1];
    $p->arr[] = &$p->arr;
    $h = (array)$p;
    $h["arr"][0] = 7;
    $q = new stdClass;
    $q->b = 1;
    $qa = (array)$q;
    $qa["b"] = 9;
    check("oac1", $arr["a"] === 5 && $p->arr[0] === 7 && $q->b === 1);
    $s = new stdClass;
    $s->c = 1;
    $rs = &$s->c;
    unset($rs);
    $sa = (array)$s;
    $sa["c"] = 4;
    $t = new S90P;
    $ta = $t->refY();
    $ta["\0S90P\0y"] = 8;
    $tb = (array)$t;
    check("oac2", $s->c === 1 && $ta["x"] === 1 && $tb["\0S90P\0y"] === 8);
    $u = new stdClass;
    $u->a = 1;
    $ua = [];
    $ua[0] = &$u->a;
    unset($u->a);
    $ub = $ua;
    $ub[0] = 9;
    $w = new S90U;
    $wa = [];
    $wa[0] = &$w->t;
    $wa[1] = &$w->d;
    unset($w->t, $w->d);
    $wb = $wa;
    $wb[0] = 7;
    $wb[1] = 6;
    check("uref1", $ua[0] === 1 && $wa === [1, 2]);
    $s90g = s90self();
    $seen = [];
    foreach ($s90g as $v) { $seen[] = $v; }
    check("grun1", $seen === [1, "Error:Cannot resume an already running generator",
        "Error:Cannot resume an already running generator@1"]);
    $s90h = s90send();
    $s90h->current();
    $m = "";
    try { $s90h->next(); } catch (Error $e) { $m = $e->getMessage(); }
    $s90a = s90a();
    $s90b = s90b();
    $s90a->current();
    $s90b->current();
    $s90a->next();
    check("grun2", $m === "Cannot resume an already running generator" && !$s90h->valid()
        && $s90a->current() === 2 && $s90b->current() === "b:Cannot resume an already running generator");
}

// ===== SECTION 91: Generator::throw on a running generator =====
// throw(e) on a generator whose body (or the body it delegates to) is running
// discards e and throws only the Error "Cannot resume an already running
// generator" at the caller; the body is not resumed (zend_generator_throw_exception
// fails when ZEND_GENERATOR_CURRENTLY_RUNNING is set, php-src
// Zend/tests/gh19326.phpt), so the Error has no previous. foreach by reference over an (array) cast writes
// the copy, not the object's property.
function s91self() {
    global $s91g;
    yield 1;
    $log = "";
    try { $s91g->throw(new Exception("x")); $log = "none"; }
    catch (Exception $e) { $log = "Exception:" . $e->getMessage(); }
    catch (Error $e) { $log = get_class($e) . ":" . $e->getMessage() . ($e->getPrevious() === null ? "" : "+prev"); }
    yield $log;
    yield 3;
}
function s91inner() { global $s91o; yield 10; try { $s91o->throw(new LogicException("y")); } catch (Error $e) { yield "in:" . $e->getMessage() . ($e->getPrevious() === null ? "" : "+prev"); } yield 12; }
function s91outer() { yield from s91inner(); yield 20; }
function s91esc() { global $s91u; yield 1; $s91u->throw(new Exception("z")); yield 2; }
function s91() {
    global $s91g, $s91o, $s91u;
    $s91g = s91self();
    $seen = [];
    foreach ($s91g as $k => $v) { $seen[] = $k . "=" . $v; }
    check("grun3", $seen === ["0=1", "1=Error:Cannot resume an already running generator", "2=3"]);
    $s91o = s91outer();
    $seen = [];
    foreach ($s91o as $k => $v) { $seen[] = $k . "=" . $v; }
    check("grun4", $seen === ["0=10", "1=in:Cannot resume an already running generator", "2=12", "0=20"]);
    $s91u = s91esc();
    $s91u->current();
    $m = "";
    try { $s91u->next(); } catch (Throwable $e) { $m = get_class($e) . ":" . $e->getMessage() . ($e->getPrevious() === null ? "" : "+prev"); }
    check("grun5", $m === "Error:Cannot resume an already running generator" && !$s91u->valid());
    $t = new stdClass;
    $t->arr = [1];
    foreach ((array)$t as &$v) { $v[] = 5; }
    unset($v);
    check("oac3", count($t->arr) === 1);
}

// ===== SECTION 92: the exception data model =====
// Exception and Error declare message:protected, string:private, code:protected,
// file:protected, line:protected, trace:private, previous:private in that order
// (Zend/zend_exceptions.stub.php); (array) mangles the non-public names and a
// foreach from outside sees none of them. __construct's third argument is
// getPrevious(); a subclass constructor that never calls the parent's still has
// the defaults. An exception escaping a finally body while another is pending
// gets that one appended to its previous chain, unless the two chains already
// meet (zend_dispatch_try_catch_finally_helper, zend_exception_set_previous).
function s92keys($o) {
    $r = [];
    foreach (array_keys((array)$o) as $k) { $r[] = str_replace("\0", "|", $k); }
    return implode(",", $r);
}
class S92Quiet extends Exception { function __construct() {} }
class S92Pub extends Exception { public $code; function __construct($c) { $this->code = $c; } }
class S92Dflt extends RuntimeException { protected $message = "dflt"; public $extra = 1; }
function s92fin() { try { throw new Exception("a"); } finally { throw new Exception("b"); } }
function s92catch() {
    try { throw new Exception("a"); }
    catch (Exception $x) { throw new Exception("b"); }
    finally { throw new Exception("c", 0, new Exception("p")); }
}
function s92inner() { try { throw new Exception("a"); } finally { try { throw new Exception("b"); } catch (Exception $z) {} } }
function s92none() { try { } finally { throw new Exception("n"); } }
function s92nest() { try { try { throw new Exception("i1"); } finally { throw new Exception("i2"); } } finally { throw new Exception("o"); } }
function s92ret() { try { throw new Exception("r"); } finally { return "ret"; } }
function s92chain($e) {
    $s = "";
    for ($p = $e; $p !== null; $p = $p->getPrevious()) { $s .= $p->getMessage() . ","; }
    return $s;
}
function s92() {
    $a = new Exception("a");
    $b = new Exception("b", 7, $a);
    check("exm1", $b->getPrevious() === $a && $a->getPrevious() === null && $b->getCode() === 7
        && method_exists($a, "getPrevious"));
    check("exm2", s92keys($b) === "|*|message,|Exception|string,|*|code,|*|file,|*|line,|Exception|trace,|Exception|previous");
    check("exm3", s92keys(new DivisionByZeroError("z")) === "|*|message,|Error|string,|*|code,|*|file,|*|line,|Error|trace,|Error|previous");
    $n = 0;
    foreach ($b as $k => $v) { $n++; }
    check("exm4", $n === 0);
    $q = new S92Quiet();
    check("exm5", $q->getMessage() === "" && $q->getCode() === 0 && $q->getPrevious() === null);
    check("exm6", s92keys(new S92Pub(4)) === "|*|message,|Exception|string,code,|*|file,|*|line,|Exception|trace,|Exception|previous");
    $e = null;
    try { s92fin(); } catch (Exception $x) { $e = $x; }
    check("exm7", s92chain($e) === "b,a,");
    try { s92catch(); } catch (Exception $x) { $e = $x; }
    check("exm8", s92chain($e) === "c,p,b,");
    try { s92inner(); } catch (Exception $x) { $e = $x; }
    check("exm9", s92chain($e) === "a,");
    try { s92none(); } catch (Exception $x) { $e = $x; }
    check("exm10", s92chain($e) === "n,");
    try { s92nest(); } catch (Exception $x) { $e = $x; }
    check("exm11", s92chain($e) === "o,i2,i1," && s92ret() === "ret");
    $ca = new Exception("ca");
    $cb = new Exception("cb", 0, $ca);
    try { try { throw $cb; } finally { throw $ca; } } catch (Exception $x) { $e = $x; }
    check("exm12", s92chain($e) === "ca,");
    try { try { throw $ca; } finally { throw $cb; } } catch (Exception $x) { $e = $x; }
    check("exm13", s92chain($e) === "cb,ca,");
    try { try { throw $ca; } finally { throw $ca; } } catch (Exception $x) { $e = $x; }
    check("exm14", s92chain($e) === "ca,");
    $z = 0;
    try { try { throw new Exception("q"); } finally { $z = 1 % $z; } } catch (Error $x) { $e = $x; }
    check("exm15", get_class($e) === "DivisionByZeroError" && s92chain($e) === "Modulo by zero,q,");
    $err = new Error("x", 0, $a);
    check("exm16", $err->getPrevious() === $a);
    $d = new S92Dflt();
    $m = "";
    try { $m = $d->message; } catch (Error $x) { $m = $x->getMessage(); }
    check("exm17", $d->getMessage() === "dflt" && $m === "Cannot access protected property S92Dflt::\$message"
        && s92keys($d) === "|*|message,|Exception|string,|*|code,|*|file,|*|line,|Exception|trace,|Exception|previous,extra");
}

// ===== END SECTIONS =====

function main() {
    global $checks;
    global $failures;
    s64(); // SECTION-CALL 64
    s65(); // SECTION-CALL 65
    s66(); // SECTION-CALL 66
    s67(); // SECTION-CALL 67
    s68(); // SECTION-CALL 68
    s69(); // SECTION-CALL 69
    s70(); // SECTION-CALL 70
    s71(); // SECTION-CALL 71
    s72(); // SECTION-CALL 72
    s73(); // SECTION-CALL 73
    s74(); // SECTION-CALL 74
    s74dump(); // SECTION-CALL 74
    s75(); // SECTION-CALL 75
    s75dump(); // SECTION-CALL 75
    s76(); // SECTION-CALL 76
    s77(); // SECTION-CALL 77
    s78(); // SECTION-CALL 78
    s79(); // SECTION-CALL 79
    s80(); // SECTION-CALL 80
    s80dump(); // SECTION-CALL 80
    s81(); // SECTION-CALL 81
    s82(); // SECTION-CALL 82
    s83(); // SECTION-CALL 83
    s84(); // SECTION-CALL 84
    s85(); // SECTION-CALL 85
    s86(); // SECTION-CALL 86
    s87(); // SECTION-CALL 87
    s88(); // SECTION-CALL 88
    s89(); // SECTION-CALL 89
    s90(); // SECTION-CALL 90
    s91(); // SECTION-CALL 91
    s92(); // SECTION-CALL 92
    echo "full: " . $checks . " checks, " . $failures . " failures\n";
    return $failures;
}
exit(main());
