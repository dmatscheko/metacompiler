<?php
// Full-syntax test: PHP (PHP 8.3 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the php grammars
// are. It walks the whole practical PHP 8.3 syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count and exit(main()) applies it
//     (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// namespaces and use-imports (single-file harness), declare(strict_types=1)
// (it would change the semantics of every section), the standard library
// beyond the array, string and math functions (sections 38, 39 and 42), define(),
// superglobals other than $GLOBALS, include/require/eval, fibers, Generator
// methods (->send() and friends live on the engine's Generator class), references
// returned from functions, reflection (attributes are applied, never read
// back), and the "${var}" interpolation form (deprecated since PHP 8.2).
// Expected values follow real PHP 8.3 (e.g. 7 / 2 === 3.5 and strlen counts
// bytes); validated against the manual by hand - no local php binary.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the PHP manual / language specification (PHP 8.3)
// with the ANTLR grammars-v4 PHP grammar as a coverage checklist.

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

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
function s01() {
    $n = 0;
    for ($i = 1; $i <= 3; $i++) { $n = $n + $i; }
    check("bas1", $n === 6);
    $m = ["a" => 1, "b" => 2];
    $m["c"] = $m["a"] + $m["b"];
    check("bas2", $m["c"] === 3 && count($m) === 3);
    $s = "";
    foreach ([1, 2] as $ix => $v) { $s .= $ix . ":" . $v . ";"; }
    check("bas3", $s === "0:1;1:2;");
    $inc = function ($x) use ($n) { return $x + $n; };
    check("bas4", $inc(4) === 10);
    check("bas5", (5 > 3 ? "y" : "n") === "y" && (0 ?: 8) === 8 && (null ?? 9) === 9);
}

// ===== SECTION 02: numeric literal forms =====
function s02() {
    check("num1", 0xFF === 255 && 0xff === 255);
    check("num2", 0b1010 === 10);
    check("num3", 0o17 === 15 && 017 === 15);
    check("num4", 1_000_000 === 1000000);
    check("num5", 1.5e3 === 1500.0 && 2.5e-2 === 0.025);
    check("num6", .5 === 0.5 && 5. === 5.0 && 1_0.2_5 === 10.25);
}

// ===== SECTION 03: string quoting and escapes =====
function s03() {
    check("sq1", strlen('a\nb') === 4 && strlen("a\nb") === 3 && strlen("\q") === 2);
    check("sq2", '\'' === "'" && strlen('\\') === 1 && "\"" === '"');
    check("sq3", strlen('$v') === 2 && "\$v" === '$v');
    check("sq4", "\x41\101" === "AA");
    check("sq5", strlen("\u{48}") === 1 && strlen("\u{2764}") === 3);
}

// ===== SECTION 04: string interpolation =====
// The "${var}" form is deprecated since PHP 8.2 and deliberately skipped.
function s04() {
    $n = 6;
    $arr = [10, 20];
    $map = ["key" => "V"];
    $o = new S04Obj();
    check("itp1", "n=$n!" === "n=6!");
    check("itp2", "{$n}7" === "67");
    check("itp3", "$arr[0]-$arr[1]" === "10-20");
    check("itp4", "$map[key]" === "V");
    check("itp5", "{$map['key']}{$arr[1]}" === "V20");
    check("itp6", "$o->val" === "7" && "{$o->twice()}" === "14");
}
class S04Obj { public $val = 7; public function twice() { return $this->val * 2; } }

// ===== SECTION 05: heredoc and nowdoc =====
function s05() {
    $x = 8;
    $h = <<<EOT
    val $x
    line2
    EOT;
    $w = <<<'EOT'
    raw $x\n
    EOT;
    $i = <<<END
      a
       b
      END;
    $e = <<<X
    X;
    check("hd1", $h === "val 8\nline2");
    check("hd2", $w === 'raw $x\n');
    check("hd3", $i === "a\n b");
    check("hd4", $e === "");
}

// ===== SECTION 06: array literals and spread =====
function s06() {
    $a = array(1, 2, 3);
    $b = [1, 2, 3,];
    check("arr1", $a == $b && $a === $b);
    $m = [5 => "x"];
    $m[] = "y";
    check("arr2", $m[6] === "y" && count($m) === 2);
    $mix = ["s" => 1, 7 => 2, "t" => 3];
    $neg = [-3 => "n"];
    check("arr3", count($mix) === 3 && $mix[7] === 2 && $neg[-3] === "n");
    $nest = [[1, [2, 3]], ["k" => [4]]];
    check("arr4", $nest[0][1][1] === 3 && $nest[1]["k"][0] === 4);
    $s1 = [1, 2];
    $s2 = [0, ...$s1, ...[3]];
    check("arr5", $s2 === [0, 1, 2, 3]);
    $k1 = ["a" => 1, "b" => 2];
    $k2 = [...$k1, "b" => 9, "c" => 3];
    check("arr6", $k2["a"] === 1 && $k2["b"] === 9 && $k2["c"] === 3);
    check("arr7", (["x" => 1, "y" => 2] == ["y" => 2, "x" => 1]) && (["x" => 1, "y" => 2] === ["y" => 2, "x" => 1]) === false);
}

// ===== SECTION 07: destructuring =====
function s07() {
    [$a, $b] = [1, 2];
    [, $second] = [10, 20];
    check("des1", $a === 1 && $b === 2 && $second === 20);
    ["y" => $py, "x" => $px] = s07pair();
    check("des2", $px === 1 && $py === 2);
    [[$m, $n], [$o]] = [[1, 2], [3]];
    check("des3", $m + $n + $o === 6);
    list($c, list($d, $e)) = [4, [5, 6]];
    check("des4", $c + $d + $e === 15);
    $sum = 0;
    foreach ([[1, 10], [2, 20]] as [$k, $v]) { $sum += $k * $v; }
    check("des5", $sum === 50);
    $names = "";
    foreach ([["id" => 1, "nm" => "a"], ["id" => 2, "nm" => "b"]] as ["nm" => $nm]) { $names .= $nm; }
    check("des6", $names === "ab");
    $p = 1;
    $q = 2;
    [$p, $q] = [$q, $p];
    check("des7", $p === 2 && $q === 1);
}
function s07pair() { return ["x" => 1, "y" => 2]; }

// ===== SECTION 08: null handling =====
function s08() {
    $u = null;
    check("nul1", ($u ?? "d") === "d" && (0 ?? 9) === 0 && ("" ?? 9) === "");
    $u ??= 7;
    $arr = ["a" => 1];
    $arr["b"] ??= 2;
    $arr["a"] ??= 99;
    check("nul2", $u === 7 && $arr["b"] === 2 && $arr["a"] === 1);
    check("nul3", ($arr["missing"] ?? 42) === 42);
    $tmp = 3;
    unset($tmp);
    check("nul4", isset($arr["a"], $arr["b"]) && !isset($arr["zz"]) && !isset($tmp));
    check("nul5", empty("") && empty("0") && empty([]) && !empty("x") && !empty(" "));
    $node = new S08Node();
    check("nul6", $node?->val === 5 && $node?->get() === 5);
    check("nul7", $node->next?->val === null && $node->next?->get() === null);
    $cnt = 0;
    $bump = function () use (&$cnt) { $cnt++; return 1; };
    $r = $node->next?->get($bump());
    check("nul8", $r === null && $cnt === 0); // ?-> short-circuits args too
}
class S08Node { public $next = null; public $val = 5; public function get() { return $this->val; } }

// ===== SECTION 09: comparison and type juggling =====
function s09() {
    check("cmp1", ("10" == "1e1") && ("1" == "01") && ("1" === "01") === false);
    check("cmp2", (0 == "a") === false && ("abc" == 0) === false && (0 == "") === false); // PHP 8 rules
    check("cmp3", (null == "") && (null == false) && (null == 0) && !(null === false));
    check("cmp4", (1 <=> 2) === -1 && (2 <=> 2) === 0 && (5 <=> 2) === 1);
    check("cmp5", ("b" <=> "a") === 1 && ([1, 2] <=> [1, 3]) === -1);
    check("cmp6", !"0" && !!"0.0" && !!" " && !"" && (bool)[] === false && (bool)[0] === true && (bool)0.0 === false);
    check("cmp7", (int)"12" === 12 && (int)3.9 === 3 && (int)-3.9 === -3 && (int)true === 1);
    check("cmp8", (float)5 === 5.0 && (string)42 === "42" && (string)3.0 === "3" && (string)true === "1" && (string)false === "" && (array)5 === [5]);
    check("cmp9", "5" + 3 === 8 && "5" . 3 === "53" && "2.5" + 1 === 3.5 && "" . 7 === "7");
    $pw = 3;
    $pw **= 2;
    check("cmp10", 6 / 2 === 3 && 7 / 2 === 3.5 && -7 % 3 === -1 && 2 ** 10 === 1024 && 2 ** -1 === 0.5 && 2 ** 3 ** 2 === 512 && $pw === 9);
}

// ===== SECTION 10: match expressions =====
function s10() {
    $r = "";
    foreach ([1, 2, 3, 9] as $v) {
        $r .= match ($v) { 1, 2 => "lo", 3 => "three", default => "hi" };
    }
    check("mat1", $r === "lolothreehi");
    $m = match ("1") { 1 => "int", "1" => "str", default => "no" };
    check("mat2", $m === "str"); // match compares with ===
    $n = 7;
    $size = match (true) { $n < 5 => "small", $n < 10 => "mid", default => "big" };
    check("mat3", $size === "mid");
    $d = match (99) { 1 => "one", default => "dflt" };
    check("mat4", $d === "dflt");
    $f = match (3.0) { 3 => "int3", 3.0 => "float3", default => "no" };
    check("mat5", $f === "float3");
}

// ===== SECTION 11: arrow functions and closures =====
function s11() {
    $inc = fn($x) => $x + 1;
    check("fn1", $inc(4) === 5);
    $m = 10;
    $times = fn($x) => $x * $m; // captures by value at creation
    $v = 1;
    $byVal = function () use ($v) { return $v; };
    $m = 0;
    $v = 99;
    check("fn2", $times(3) === 30 && $byVal() === 1);
    $mk = fn($a) => fn($b) => $a + $b;
    check("fn3", $mk(1)(2) === 3);
    $t = fn($x): int => $x * 2;
    check("fn4", $t(21) === 42);
    $cnt = 0;
    $byRef = function () use (&$cnt) { $cnt = $cnt + 1; return $cnt; };
    $byRef();
    $byRef();
    check("fn5", $cnt === 2);
    $st = static function () { return 7; };
    $stf = static fn() => 8;
    check("fn6", $st() + $stf() === 15);
    $imm = (function ($x) { return $x + 1; })(41);
    check("fn7", $imm === 42);
    check("fn8", s11global() === "30|30"); // fn captures by value at EVERY scope
}
$s11m = 10;
$s11times = fn($x) => $x * $s11m;
$s11m = 0;
function s11global() {
    global $s11times;
    $m = 10;
    $local = fn($x) => $x * $m;
    $m = 0;
    return $s11times(3) . "|" . $local(3);
}

// ===== SECTION 12: named arguments and callables =====
function s12() {
    check("cal1", s12area(3) === 6 && s12area(3, 4) === 12);
    check("cal2", s12area(h: 5, w: 2) === 10 && s12area(2, scale: 10) === 40);
    check("cal3", s12area(...["w" => 1, "h" => 3]) === 3 && s12join("-", ...["a", "b", "c"]) === "a-b-c");
    $f = s12area(...); // first-class callable syntax (8.1)
    check("cal4", $f(3, 3) === 9);
    $o = new S12M(10);
    $g = $o->add(...);
    $s = S12M::neg(...);
    check("cal5", $g(5) === 15 && $s(4) === -4);
    $name = "s12area";
    check("cal6", $name(2, 2) === 4); // variable function
    $meth = "add";
    check("cal7", $o->$meth(1) === 11 && $o->{"add"}(2) === 12);
}
function s12area($w, $h = 2, $scale = 1) { return $w * $h * $scale; }
function s12join($sep, ...$parts) { $out = ""; foreach ($parts as $i => $p) { $out .= ($i > 0 ? $sep : "") . $p; } return $out; }
class S12M { public $base; public function __construct($b) { $this->base = $b; } public function add($x) { return $this->base + $x; } public static function neg($x) { return -$x; } }

// ===== SECTION 13: function signatures =====
function s13() {
    check("sig1", s13nul(4) === 5 && s13nul(null) === null && s13void() === null);
    check("sig2", s13uni(1) === "int" && s13uni("s") === "s" && s13mix(7) === 7);
    $n = 5;
    s13ref($n);
    check("sig3", $n === 6);
    check("sig4", s13var(1, 2, 3) === 6 && s13var() === 0);
    check("sig5", s13both(new S13AB()) === "ab");
    $hit = false;
    try { s13nvr(); } catch (S13Bang $e) { $hit = true; }
    check("sig6", $hit);
    $trail = function ($a, $b,) { return $a . $b; };
    check("sig7", $trail("x", "y",) === "xy");
}
function s13nul(?int $n): ?int { return $n === null ? null : $n + 1; }
function s13uni(int|string $v): string { return $v === 1 ? "int" : $v; }
function s13mix(mixed $m): mixed { return $m; }
function s13void(): void { }
function s13ref(int &$n): void { $n = $n + 1; }
function s13var(int ...$nums): int { $s = 0; foreach ($nums as $x) { $s += $x; } return $s; }
interface S13A { public function a(): string; }
interface S13B { public function b(): string; }
class S13AB implements S13A, S13B { public function a(): string { return "a"; } public function b(): string { return "b"; } }
function s13both(S13A&S13B $x): string { return $x->a() . $x->b(); }
class S13Bang extends Exception {}
function s13nvr(): never { throw new S13Bang(); }

// ===== SECTION 14: class members =====
function s14() {
    $c = new S14Conf(4);
    check("cls1", $c->x === 4 && $c->promoted === 7 && $c->sum() === 12);
    check("cls2", $c->ro === 8 && $c->tagIs() === "t");
    check("cls3", S14Conf::GREET === "hi" && S14Conf::MAX === 10 && S14Conf::SIZE === 4 && S14Conf::TWICE === 8 && $c::GREET === "hi");
    $d = S14Conf::make(1);
    check("cls4", S14Conf::$made === 2 && $d->sum() === 9);
    $f = new S14Frozen(2, 3);
    check("cls5", $f->sum() === 5 && $f->a === 2);
    $anon = new class(6) { public $v; public function __construct($v) { $this->v = $v; } public function twice() { return $this->v * 2; } };
    check("cls6", $anon->twice() === 12);
    check("cls7", S14Conf::class === "S14Conf" && $f::class === "S14Frozen");
}
class S14Conf {
    public const GREET = "hi";
    final public const MAX = 10;
    public const int SIZE = 4; // typed class constant (8.3)
    public const TWICE = self::SIZE * 2;
    public static int $made = 0;
    public int $x;
    protected string $tag = "t";
    private $raw = 5;
    public readonly int $ro;
    public function __construct(int $x, public int $promoted = 7, private int $hidden = 3) { $this->x = $x; $this->ro = $x * 2; self::$made = self::$made + 1; }
    public function sum(): int { return $this->x + $this->raw + $this->hidden; }
    public function tagIs(): string { return $this->tag; }
    public static function make(int $x): S14Conf { return new S14Conf($x); }
}
readonly class S14Frozen {
    public function __construct(public int $a, public int $b) {}
    public function sum(): int { return $this->a + $this->b; }
}

// ===== SECTION 15: inheritance and interfaces =====
function s15() {
    $s = new S15Sq();
    check("inh1", $s->area() === 9 && $s->describe() === "S:A:9");
    check("inh2", $s instanceof S15Sq && $s instanceof S15Base && $s instanceof S15Shape);
    check("inh3", S15Sq::viaSelf() === "base" && S15Sq::viaStatic() === "sq"); // late static binding
    $t = S15Sq::create();
    check("inh4", $t instanceof S15Sq && $t->area() === 9);
    check("inh5", S15Shape::KIND === "shape" && S15Sq::KIND === "shape");
    $cn = "S15Sq";
    $dyn = new $cn();
    check("inh6", $dyn instanceof S15Sq && $s instanceof $cn);
}
interface S15Shape { const KIND = "shape"; public function area(): int; }
abstract class S15Base implements S15Shape {
    abstract public function area(): int;
    public function describe(): string { return "A:" . $this->area(); }
    public static function who(): string { return "base"; }
    public static function viaSelf(): string { return self::who(); }
    public static function viaStatic(): string { return static::who(); }
    public static function create(): static { return new static(); }
}
final class S15Sq extends S15Base {
    public $side = 3;
    public function area(): int { return $this->side * $this->side; }
    public static function who(): string { return "sq"; }
    public function describe(): string { return "S:" . parent::describe(); }
}

// ===== SECTION 16: traits =====
function s16() {
    $b = new S16Both();
    check("trt1", $b->hello() === "hello" && $b->greeted === 1);
    check("trt2", $b->welsh() === "wello");
    check("trt3", $b->all() === "hello,wello,world");
    check("trt4", S16Both::shout() === "HELLO");
}
trait S16Hello { public $greeted = 0; public function hello(): string { $this->greeted++; return "hello"; } public static function shout(): string { return "HELLO"; } }
trait S16World { public function world(): string { return "world"; } public function hello(): string { return "wello"; } }
class S16Both {
    use S16Hello, S16World { S16Hello::hello insteadof S16World; S16World::hello as welsh; world as protected innerWorld; }
    public function all(): string { return $this->hello() . "," . $this->welsh() . "," . $this->innerWorld(); }
}

// ===== SECTION 17: magic methods =====
function s17() {
    $b = new S17Bag();
    $b->size = 5; // __set stores doubled
    check("mag1", $b->size === 10);
    check("mag2", $b->nope === "?nope");
    check("mag3", isset($b->size) && !isset($b->nope));
    check("mag4", $b->anything(1, 2, 3) === "anything:3" && S17Bag::missing() === "st-missing");
    check("mag5", $b(1) === 101);
    check("mag6", "$b" === "bag[1]" && ("x" . $b) === "xbag[1]");
    $c = clone $b;
    check("mag7", $c->copies === 1 && $b->copies === 0); // __clone runs on the copy
}
class S17Bag {
    private $data = [];
    public $copies = 0;
    public function __get($name) { return $this->data[$name] ?? "?" . $name; }
    public function __set($name, $value) { $this->data[$name] = $value * 2; }
    public function __isset($name) { return isset($this->data[$name]); }
    public function __call($name, $args) { return $name . ":" . count($args); }
    public static function __callStatic($name, $args) { return "st-" . $name; }
    public function __invoke($x) { return $x + 100; }
    public function __toString(): string { return "bag[" . count($this->data) . "]"; }
    public function __clone() { $this->copies = $this->copies + 1; }
}

// ===== SECTION 18: enums =====
function s18() {
    check("enu1", S18Suit::Hearts->value === "h" && S18Suit::Hearts->name === "Hearts");
    check("enu2", S18Suit::Hearts->color() === "red" && S18Suit::Spades->color() === "black");
    check("enu3", S18Suit::Hearts === S18Suit::Hearts && S18Suit::Hearts !== S18Suit::Spades);
    check("enu4", S18Suit::Hearts instanceof S18Suit && S18Suit::Hearts instanceof S18HasCode);
    check("enu5", S18Suit::Hearts->code() === 1 && S18Suit::fallback() === S18Suit::Spades && S18Suit::WILD === "w");
    $cases = S18Dir::cases();
    check("enu6", count($cases) === 2 && $cases[0] === S18Dir::Up && $cases[1]->name === "Down");
    $pick = match (S18Dir::Down) { S18Dir::Up => "u", S18Dir::Down => "d" };
    check("enu7", $pick === "d");
}
interface S18HasCode { public function code(): int; }
enum S18Suit: string implements S18HasCode {
    case Hearts = "h";
    case Spades = "s";
    const WILD = "w";
    public function color(): string { return match ($this) { S18Suit::Hearts => "red", S18Suit::Spades => "black" }; }
    public function code(): int { return $this === S18Suit::Hearts ? 1 : 2; }
    public static function fallback(): S18Suit { return S18Suit::Spades; }
}
enum S18Dir { case Up; case Down; }

// ===== SECTION 19: generators =====
function s19() {
    $sum = 0;
    foreach (s19nums() as $v) { $sum += $v; }
    check("gen1", $sum === 15);
    $ks = "";
    foreach (s19keyed() as $k => $v) { $ks .= $k . $v; }
    check("gen2", $ks === "a1b2");
    $auto = "";
    foreach (s19bounded(3) as $k => $v) { $auto .= $k . ":" . $v . ";"; }
    check("gen3", $auto === "0:0;1:1;2:2;");
    $lazyLog = "";
    $mk = function () use (&$lazyLog) { $lazyLog .= "run;"; yield 9; };
    $g = $mk();
    $before = $lazyLog;
    foreach ($g as $v) { $lazyLog .= "got" . $v . ";"; }
    check("gen4", $before === "" && $lazyLog === "run;got9;"); // body runs lazily
    $collected = "";
    foreach (s19bounded(100) as $v) { if ($v === 3) { break; } $collected .= $v; }
    check("gen5", $collected === "012");
    $dele = "";
    foreach (s19delegate() as $k => $v) { $dele .= $k . $v . ";"; }
    check("gen6", $dele === "z0;a1;b2;"); // yield from forwards the inner KEYS
    // The Generator object's own cursor. (send() needs a suspendable body and is
    // deliberately out of scope - see the header.)
    $it = s19keyed();
    $seen = $it->current() . $it->key() . ($it->valid() ? "y" : "n");
    $it->next();
    $seen .= $it->current() . $it->key();
    $it->next();
    check("gen7", $seen === "1ay2b" && !$it->valid() && $it->getReturn() === 7);
    // An INFINITE generator is PULLED, one value at a time, so a 'break' ends it.
    // Pinned to php-src's own tests/generators/fibonacci.phpt, whose --EXPECT--
    // block runs int(1) .. int(987) and stops at the first value over 1000.
    $fib = "";
    foreach (s19fib() as $n) { if ($n > 1000) { break; } $fib .= $n . ","; }
    check("gen8", $fib === "1,2,3,5,8,13,21,34,55,89,144,233,377,610,987,");
    // and the generator's side effects INTERLEAVE with the consumer's, which is the
    // observable half of laziness. tests/generators/bug63066.phpt pins the order: its
    // --EXPECTF-- prints the consumer's "foo" BEFORE the fatal error raised by the
    // statement that follows the yield, so the consumer runs between the two.
    $order = "";
    $og = function () use (&$order) { $order .= "y1;"; yield 1; $order .= "y2;"; yield 2; };
    foreach ($og() as $v) { $order .= "c" . $v . ";"; }
    check("gen9", $order === "y1;c1;y2;c2;");
    // 'yield from' delegates lazily too, so an infinite outer generator that
    // delegates in a loop still streams: tests/generators/bug71297.phpt is exactly
    // this shape (break at the fourth value) and its --EXPECT-- block is "012".
    $del = "";
    foreach (s19infdel() as $v) { if ($del === "012") { break; } $del .= $v; }
    check("gen10", $del === "012");
}
function s19fib() { $a = 1; $b = 1; while (true) { yield $b; $t = $a + $b; $a = $b; $b = $t; } }
function s19one($i) { yield $i; }
function s19infdel() { $i = 0; while (true) { yield from s19one($i); $i++; } }
function s19nums() { yield 1; yield 2; yield from [3, 4]; yield 5; }
function s19keyed() { yield "a" => 1; yield "b" => 2; return 7; }
function s19bounded($limit) { $i = 0; while ($i < $limit) { yield $i; $i++; } }
function s19delegate() { yield "z" => 0; yield from s19keyed(); }

// ===== SECTION 20: references and copy semantics =====
function s20() {
    $a = 1;
    $b = &$a;
    $b = 5;
    $up = $a === 5;
    $a = 7;
    check("ref1", $up && $b === 7);
    $arr = [1, 2, 3];
    foreach ($arr as &$v) { $v = $v * 10; }
    unset($v);
    $el = &$arr[0];
    $el = 99;
    check("ref2", $arr === [99, 20, 30]);
    $src = [1, 2];
    $copy = $src;
    $copy[0] = 9;
    check("ref3", $src[0] === 1 && $copy[0] === 9); // arrays assign by value
    $o1 = new S20Box();
    $o2 = $o1;
    $o2->v = 9;
    check("ref4", $o1->v === 9); // objects assign by handle
    $o3 = clone $o1;
    $o3->v = 1;
    check("ref5", $o1->v === 9 && $o3->v === 1);
    $p1 = new S20Box();
    $p2 = new S20Box();
    check("ref6", $p1 == $p2 && !($p1 === $p2) && $o1 === $o2);
}
class S20Box { public $v = 0; }

// ===== SECTION 21: exceptions =====
function s21() {
    $log = "";
    try { $log .= "t"; throw new S21Err("x"); } catch (S21Err $e) { $log .= "c" . $e->tag; } finally { $log .= "f"; }
    check("exc1", $log === "tcxf");
    $r = "";
    try { s21boom(1); } catch (S21Err $e) { $r = "sub:" . $e->tag; } // catch by parent class
    check("exc2", $r === "sub:pos");
    $m = "";
    try { s21boom(0); } catch (S21Sub|S21Other) { $m = "multi"; } // union catch, no variable
    check("exc3", $m === "multi");
    $n = "";
    try { s21boom(0); } catch (S21Sub $e) { $n = "sub"; } catch (Exception $e) { $n = "base"; }
    check("exc4", $n === "base"); // first MATCHING clause wins
    $rethrown = "";
    try {
        try { throw new S21Err("deep"); } catch (S21Err $e) { throw new S21Err($e->tag . "er"); }
    } catch (S21Err $e2) {
        $rethrown = $e2->tag;
    }
    check("exc5", $rethrown === "deeper");
    $u = null;
    $got = "";
    try { $w = $u ?? throw new S21Err("np"); } catch (S21Err $e) { $got = $e->tag; } // throw expression
    check("exc6", $got === "np");
    $short = fn($x) => $x >= 0 ? $x : throw new S21Err("neg");
    $sg = "";
    try { $short(-1); } catch (S21Err $e) { $sg = $e->tag; }
    check("exc7", $short(3) === 3 && $sg === "neg");
}
class S21Err extends Exception { public $tag; public function __construct($tag) { $this->tag = $tag; } }
class S21Sub extends S21Err {}
class S21Other extends Exception {}
function s21boom($n) { if ($n > 0) { throw new S21Sub("pos"); } throw new S21Other(); }

// ===== SECTION 22: attributes =====
// Apply-only: reading attributes back needs reflection (out of scope).
function s22() {
    $t = new S22Target();
    check("att1", $t->get(2) === 3);
    check("att2", s22fn(21) === 42);
    check("att3", S22Target::OK === true);
    $c = #[S22Extra] function () { return 5; };
    check("att4", $c() === 5);
    $a = #[S22Extra] fn() => 6;
    check("att5", $a() === 6);
}
#[Attribute]
class S22Mark { public $note; public function __construct($note = "") { $this->note = $note; } }
#[Attribute]
class S22Extra {}
#[S22Mark("on-class")]
class S22Target {
    #[S22Mark(note: "const")]
    public const OK = true;
    #[S22Mark]
    public $field = 1;
    #[S22Mark("m"), S22Extra]
    public function get(#[S22Mark] $p): int { return $this->field + $p; }
}
#[S22Mark("fn"), S22Extra]
function s22fn(#[S22Extra] $x) { return $x * 2; }

// ===== SECTION 23: control statements =====
function s23() {
    check("ctl1", s23switch(1) . s23switch(2) . s23switch(3) . s23switch(9) === "lolothreehi");
    $fall = "";
    switch (2) { case 2: $fall .= "a"; case 3: $fall .= "b"; break; default: $fall .= "c"; } // fallthrough
    check("ctl2", $fall === "ab");
    $n = 0;
    do { $n++; } while ($n < 3);
    $once = 0;
    do { $once++; } while (false);
    check("ctl3", $n === 3 && $once === 1);
    $alt = "";
    if ($n === 3): $alt .= "i"; elseif ($n === 4): $alt .= "e"; else: $alt .= "l"; endif;
    for ($i = 0; $i < 2; $i++): $alt .= "f"; endfor;
    foreach (["x", "y"] as $ch): $alt .= $ch; endforeach;
    $w = 0;
    while ($w < 1): $alt .= "w"; $w++; endwhile;
    check("ctl4", $alt === "iffxyw");
    $lvl = "";
    for ($a = 0; $a < 3; $a++) { for ($b = 0; $b < 3; $b++) { if ($b === 1) { continue 2; } $lvl .= $a . $b; } }
    $brk = "";
    foreach ([1, 2] as $x) { foreach ([1, 2] as $y) { if ($x === 2) { break 2; } $brk .= $x . $y; } }
    check("ctl5", $lvl === "001020" && $brk === "1112");
    $g = 1;
    goto s23end;
    $g = 99;
    s23end:
    check("ctl6", $g === 1);
}
function s23switch($v) { switch ($v) { case 1: case 2: return "lo"; case 3: return "three"; default: return "hi"; } }

// ===== SECTION 24: constants, statics, misc =====
function s24() {
    check("msc1", S24_TOP === 40 && S24_CALC === 41 && TRUE === true && NULL === null);
    check("msc2", s24counter() === 1 && s24counter() === 2 && s24name() === "s24name");
    $x = 5;
    $name = "x";
    $ok1 = $$name === 5; // variable variables
    $$name = 6;
    check("msc3", $ok1 && $x === 6);
    $h = "hello";
    $h[0] = "H";
    check("msc4", "abc"[-1] === "c" && "abc"[0] === "a" && $h === "Hello");
    check("msc5", (5 & 3) === 1 && (5 | 2) === 7 && (5 ^ 1) === 4 && (~5) === -6 && (1 << 4) === 16 && (32 >> 2) === 8);
    $bit = 6;
    $bit &= 3;
    $bit |= 8;
    $bit ^= 2;
    $bit <<= 1;
    $bit >>= 2;
    check("msc6", $bit === 4);
}
const S24_TOP = 40;
const S24_CALC = S24_TOP + 1;
function s24counter() { static $n = 0; $n++; return $n; }
function s24name() { return __FUNCTION__; }

// ===== SECTION 25: reference cells, capture and late binding =====
// The fine print of the three mechanisms sections 08/11/20 introduce: WHICH storage
// cell a reference denotes, WHEN a closure freezes what it captures, and when a
// static local's initializer runs. Every expectation below is the same under both
// php grammars, which is the property this file exists to hold.
function s25() {
    // Two references to the SAME element share one cell, and an ordinary write to
    // that element goes through it.
    $a = [1, 2];
    $x = &$a[0];
    $y = &$a[0];
    $y = 9;
    check("cel1", $x === 9 && $a[0] === 9);
    $b = [5];
    $p = &$b[0];
    $b[0] = 7;
    check("cel2", $p === 7);
    // A by-reference PARAMETER binds any assignable argument, not only a plain
    // variable: an array element and an object PROPERTY are storage locations too.
    $arr = [3, 4];
    s25bump($arr[1]);
    $nest = ["k" => [1, 2]];
    s25bump($nest["k"][0]);
    $o = new S25Box();
    s25bump($o->v);
    check("cel3", $arr[1] === 8 && $nest["k"][0] === 2 && $o->v === 12);
    // A reference to a property aliases it in both directions, and a second
    // reference to the same property shares the one cell.
    $pr = &$o->v;
    $pr = 4;
    $o->v = 7;
    $pr2 = &$o->v;
    $pr2 = 1;
    check("cel4", $pr === 1 && $o->v === 1);
    // 'use ($k)' freezes the VALUE where the closure is created, even when another
    // closure holds the same variable by reference; 'use (&$k)' stays live.
    $k = 1;
    $live = function () use (&$k) { $k = $k + 10; return $k; };
    $frozen = function () use ($k) { return $k; };
    $live();
    check("cap1", $k === 11 && $frozen() === 1);
    // A by-reference use survives being captured again by an inner closure.
    $n = 0;
    $mk = function () use (&$n) { return function () use (&$n) { $n = $n + 1; return $n; }; };
    $f = $mk();
    $f();
    $f();
    check("cap2", $n === 2);
    // An arrow function captures the VALUE of a variable that is a reference, and a
    // by-reference use inside it binds the arrow function's own copy.
    $rv = 1;
    $rr = &$rv;
    $af = fn() => $rv;
    $rv = 2;
    $an = fn() => function () use (&$rv) { $rv = $rv + 1; return $rv; };
    $inner = $an();
    check("cap3", $af() === 1 && $inner() === 3 && $inner() === 4 && $rv === 2);
    // A static local's initializer runs on the FIRST CALL, not where the program
    // starts, and each declaration site has storage of its own.
    check("sta1", s25order() === "" && s25count() === 6 && s25order() === "init;");
    check("sta2", s25count() === 7 && s25other() === 101 && s25count() === 8);
    // A static explicitly set to null stays null; it is not re-initialized.
    check("sta3", s25nulled() && s25nulled());
    // Named arguments name the parameters of a METHOD and of a constructor.
    $q = new S25Pair(c: 9, a: 1);
    check("nam1", $q->s === "1-2-9");
    check("nam2", $o->span(hi: 5, lo: 1) === 4 && $o->span(2, step: 3) === 24);
}
class S25Box { public $v = 6; public function span($lo, $hi = 10, $step = 1) { return ($hi - $lo) * $step; } }
class S25Pair { public $s; public function __construct($a, $b = 2, $c = 3) { $this->s = $a . "-" . $b . "-" . $c; } }
function s25bump(&$v) { $v = $v * 2; }
$s25log = "";
function s25order() { global $s25log; return $s25log; }
function s25seed() { global $s25log; $s25log .= "init;"; return 5; }
function s25count() { static $n = s25seed(); $n++; return $n; }
function s25other() { static $n = 100; $n++; return $n; }
function s25nulled() { static $z = 3; $z = null; return $z === null; }

// ===== SECTION 26: property hooks and declare =====
// PHP 8.4 property hooks, and the declare() statement. A hook makes a property read
// or write run code; the property itself stays the backing store, which is what
// '$this->x' means INSIDE x's own hook (anywhere else it would re-enter the hook).
function s26() {
    $t = new S26Temp(20);
    check("hok1", $t->celsius === 20 && $t->fahrenheit === 68);
    $t->fahrenheit = 212;
    check("hok2", $t->celsius === 100 && $t->fahrenheit === 212);
    $t->celsius = -300;
    check("hok3", $t->celsius === -273); // the set hook clamps
    $c = new S26Counted();
    check("hok4", $c->n === 0 && $c->reads === 1 && $c->n === 0 && $c->reads === 2);
    check("hok5", $c->plain === "p"); // an unhooked property keeps the plain path
    check("hok6", s26declared() === 6);
}
class S26Temp {
    public $celsius = 0 { set { $this->celsius = $value < -273 ? -273 : $value; } }
    public $fahrenheit { get => $this->celsius * 9 / 5 + 32;
                         set { $this->celsius = ($value - 32) * 5 / 9; } }
    public function __construct($c) { $this->celsius = $c; }
}
class S26Counted {
    public $reads = 0;
    public $plain = "p";
    public $n = 0 { get { $this->reads = $this->reads + 1; return $this->n; } }
}
// declare() carries no directive this subset models, and the braced form is an
// ordinary block. (Under strict_types real PHP raises a TypeError for a mismatched
// argument; no grammar here checks parameter types, so the call is accepted.)
declare(ticks=1);
function s26declared(): int {
    declare(ticks=1) { $r = 6; }
    return $r;
}

// ===== SECTION 27: interpreter/compiler agreement ratchet =====
// strlen() measures PHP's string form of its argument: strlen(true) is 1 and
// strlen(false) 0 (the JavaScript form would give 4 and 5).
function s27() {
    check("agr1", strlen(true) === 1);
    check("agr2", strlen(false) === 0);
    check("agr3", strlen(null) === 0);
    check("agr4", strlen("abc") === 3);
    check("agr5", strlen(12) === 2);
    $t = true; $f = false; $n = null;
    check("agr6", strlen($t) === 1 && strlen($f) === 0 && strlen($n) === 0);
}
// ===== SECTION 28: 64-bit integers =====
// PHP's int is 64 bit two's complement, and its overflow rule is the OPPOSITE of
// C's: an operation whose exact result leaves 64 bits does not wrap, it silently
// becomes a float. Every expectation below is quoted from PHP's own test suite,
// vendored under tests/reference/php/php-src-tests/ - the .phpt named on each
// line is the one whose --EXPECT-- block specifies it.
function s28() {
    // lang/operators/add_basiclong_64bit.phpt defines MAX_64Bit / MIN_64Bit
    // exactly this way and var_dumps int(9223372036854775807).
    check("i64a", PHP_INT_MAX === 9223372036854775807);
    check("i64b", PHP_INT_MIN === -9223372036854775807 - 1);
    check("i64c", PHP_INT_SIZE === 8);
    check("i64d", (string)1234567890123456789 === "1234567890123456789");
    // add_basiclong_64bit.phpt: "9223372036854775807 + 1" is
    // float(9.223372036854776E+18), "+ -1" is int(9223372036854775806).
    check("i64e", is_float(PHP_INT_MAX + 1) && is_int(PHP_INT_MAX + -1));
    check("i64f", PHP_INT_MAX + -1 === 9223372036854775806);
    check("i64g", is_int(PHP_INT_MAX) && !is_float(PHP_INT_MAX));
    // multiply_basiclong_64bit.phpt: the product that still fits stays an int.
    check("i64h", is_float(PHP_INT_MAX * 2) && 4611686018427387903 * 2 === 9223372036854775806);
    // subtract_basiclong_64bit.phpt: MIN_64Bit - 1 is a float.
    check("i64i", is_float(PHP_INT_MIN - 1) && is_int(PHP_INT_MIN + 1));
    // negate_basiclong_64bit.phpt: the one int that cannot be negated is MIN.
    check("i64j", is_float(-PHP_INT_MIN) && -PHP_INT_MAX === PHP_INT_MIN + 1);
    // postinc_basiclong_64bit.phpt / predec_basiclong_64bit.phpt.
    $x = PHP_INT_MAX; $x++;
    $y = PHP_INT_MIN; $y--;
    check("i64k", is_float($x) && is_float($y));
    // divide_basiclong_64bit.phpt: '/' is an int ONLY when it comes out exact.
    check("i64l", 6 / 3 === 2 && is_int(6 / 3));
    check("i64m", 7 / 2 === 3.5 && is_float(7 / 2));
    check("i64n", intdiv(7, 2) === 3 && intdiv(-7, 2) === -3);
    check("i64o", 9223372036854775807 / 7 === 1317624576693539401);
    // modulus_basiclong_64bit.phpt: '%' casts both operands to int, and
    // "9223372036854775807 % -1" is int(0).
    check("i64p", 7 % 3 === 1 && -7 % 3 === -1 && 7.9 % 3 === 1);
    check("i64q", PHP_INT_MAX % -1 === 0 && PHP_INT_MIN % -1 === 0);
    // bitwiseShiftLeft_basiclong_64bit.phpt: '<<' WRAPS - "9223372036854775807
    // << 1" is int(-2), not a float - and a count at or above 64 shifts out.
    check("i64r", PHP_INT_MAX << 1 === -2 && is_int(PHP_INT_MAX << 1));
    check("i64s", 1 << 62 === 4611686018427387904 && 1 << 63 === PHP_INT_MIN);
    check("i64t", PHP_INT_MAX << 65 === 0 && PHP_INT_MAX << 9223372036854775807 === 0);
    // bitwiseShiftRight_basiclong_64bit.phpt: ">> 65" is int(0) for a positive
    // value and int(-1) for a negative one.
    check("i64u", PHP_INT_MAX >> 65 === 0 && PHP_INT_MIN >> 65 === -1);
    check("i64v", PHP_INT_MIN >> 1 === -4611686018427387904);
    // bitwiseNot / bitwiseAnd / bitwiseOr / bitwiseXor _basiclong_64bit.phpt.
    check("i64w", ~PHP_INT_MAX === PHP_INT_MIN && ~0 === -1 && ~PHP_INT_MIN === PHP_INT_MAX);
    check("i64x", (PHP_INT_MAX & PHP_INT_MIN) === 0 && (PHP_INT_MAX | PHP_INT_MIN) === -1);
    check("i64y", (PHP_INT_MAX ^ PHP_INT_MIN) === -1 && (PHP_INT_MAX & 255) === 255);
    // The two arithmetic errors PHP 8 made catchable. Messages quoted from
    // divide_basiclong_64bit.phpt ("DivisionByZeroError: Division by zero"),
    // modulus_basiclong_64bit.phpt ("Modulo by zero") and
    // bitwiseShiftLeft_basiclong_64bit.phpt ("Bit shift by negative number").
    $m = "";
    try { $z = 1 / 0; } catch (DivisionByZeroError $e) { $m = $e->getMessage(); }
    check("i64z", $m === "Division by zero");
    $m = "";
    try { $z = 1 % 0; } catch (DivisionByZeroError $e) { $m = $e->getMessage(); }
    check("i64A", $m === "Modulo by zero");
    $m = "";
    try { $z = 1 << -1; } catch (ArithmeticError $e) { $m = $e->getMessage(); }
    check("i64B", $m === "Bit shift by negative number");
    $m = "";
    try { $z = intdiv(1, 0); } catch (DivisionByZeroError $e) { $m = $e->getMessage(); }
    check("i64C", $m === "Division by zero");
    // A DivisionByZeroError IS an ArithmeticError (PHP's real hierarchy).
    $ok = false;
    try { $z = 1 / 0; } catch (ArithmeticError $e) { $ok = true; }
    check("i64D", $ok);
    // lang/integer_literals/{hexadecimal,binary,octal}_64bit.phpt: a literal that
    // does not FIT is a float, not a wrapped int.
    check("i64E", is_int(0x7FFFFFFFFFFFFFFF) && 0x7FFFFFFFFFFFFFFF === PHP_INT_MAX);
    check("i64F", is_float(0xFFFFFFFFFFFFFFFF) && is_float(0x45FFFABCDE0000000));
    check("i64G", is_int(0b111111111111111111111111111111111111111111111111111111111111111));
    check("i64H", 0o777777777777777777777 === PHP_INT_MAX && is_float(0o1000000000000000000000));
    check("i64I", 0_16 === 14 && 00_016 === 14 && 016 === 14);
    check("i64J", is_int(9223372036854775807) && is_float(9223372036854775808));
    // tests/int_overflow_64bit.phpt: (int) of an out-of-range float WRAPS modulo
    // 2^64 - (int)(PHP_INT_MAX * 2 + 4) is int(0), i.e. 2^64 mod 2^64.
    check("i64K", (int)(PHP_INT_MAX + 1) === PHP_INT_MIN);
    check("i64L", (int)(PHP_INT_MAX * 2 + 4) === 0);
    check("i64M", (int)(PHP_INT_MAX + 1000) === PHP_INT_MIN);
    // tests/int_underflow_64bit.phpt: every value below MIN casts to MIN.
    check("i64N", (int)(-9223372036854775809) === PHP_INT_MIN);
    // A numeric STRING that overflows is a float too (is_numeric_string reports
    // IS_DOUBLE), which is why the sum below changes type.
    check("i64O", "9223372036854775807" + 0 === PHP_INT_MAX);
    check("i64P", is_float("9223372036854775808" + 0));
    // Comparison is exact at 64 bits: a double would collapse these onto one value.
    check("i64Q", 9223372036854775807 > 9223372036854775806);
    check("i64R", !(PHP_INT_MAX == PHP_INT_MAX - 1) && (PHP_INT_MAX <=> PHP_INT_MAX - 1) === 1);
    // Array keys are 64 bit too, and a string key that does not fit stays a string.
    $a = [];
    $a[PHP_INT_MAX] = "hi";
    $a[PHP_INT_MIN] = "lo";
    check("i64S", $a[PHP_INT_MAX] === "hi" && $a[PHP_INT_MIN] === "lo" && count($a) === 2);
    // '**' overflows to a float like every other arithmetic operator.
    check("i64T", 2 ** 62 === 4611686018427387904 && is_int(2 ** 62) && is_float(2 ** 63));
    check("i64U", (-2) ** 3 === -8 && is_float(2 ** -1));
    check("i64V", abs(PHP_INT_MIN) === 9.2233720368547758E+18 && abs(-5) === 5);
}

// ===== SECTION 29: the type predicates and PHP's two float renderers =====
// PHP renders a float TWO ways: echo / (string) / interpolation use
// precision=14, while var_dump and var_export use serialize_precision=-1, the
// shortest round-tripping digits. The same value is 9.2233720368548E+18 to the
// first and 9.223372036854776E+18 to the second - confusing them is the classic
// trap, and both spellings appear in the SAME corpus file
// (lang/operators/add_basiclong_64bit.phpt: the "--- testing: ..." echo lines
// against the var_dump lines under them).
function s29() {
    check("typ1", gettype(5) === "integer" && gettype(5.0) === "double");
    check("typ2", gettype("s") === "string" && gettype(true) === "boolean" && gettype(null) === "NULL");
    check("typ3", gettype([]) === "array" && gettype(new S29Box()) === "object");
    check("typ4", is_int(5) && !is_int(5.0) && is_float(5.0) && !is_float(5));
    check("typ5", is_string("a") && is_bool(false) && is_array([]) && is_null(null));
    check("typ6", is_numeric("12") && is_numeric(1.5) && !is_numeric("x"));
    check("typ7", get_debug_type(5) === "int" && get_debug_type(5.0) === "float");
    check("typ8", get_debug_type(null) === "null" && get_debug_type(new S29Box()) === "S29Box");
    check("typ9", get_class(new S29Box()) === "S29Box");
    // precision=14 for the string form.
    check("flo1", (string)(PHP_INT_MAX + 1) === "9.2233720368548E+18");
    check("flo2", (string)(0.1 + 0.2) === "0.3");
    check("flo3", (string)(1 / 3) === "0.33333333333333");
    check("flo4", (string)3.0 === "3" && (string)(-0.0) === "-0");
    check("flo5", "" . 1.0E+25 === "1.0E+25" && "" . 1.0E-5 === "1.0E-5");
    check("flo6", "" . 0.0001 === "0.0001" && "" . 100000000000000.0 === "1.0E+14");
    // INF / NAN print as those three letters in both renderers (the corpus's
    // float(INF) line in lang/integer_literals/binary_64bit.phpt).
    check("flo7", "" . INF === "INF" && "" . -INF === "-INF" && "" . NAN === "NAN");
    $big = PHP_INT_MAX + 1;
    check("flo9", "v=$big" === "v=9.2233720368548E+18");
}
class S29Box { public $v = 1; }
// var_dump's OWN renderer is serialize_precision=-1, which no expression above
// can observe - so these lines put it in the transcript, where --full's goja vs
// -frozen check sees it (--cross skips this file). Every expected line is
// quoted from the corpus: int(9223372036854775807) and
// float(9.223372036854776E+18) from lang/operators/add_basiclong_64bit.phpt,
// float(1.9119287772983036E+25) from lang/integer_literals/binary_64bit.phpt,
// float(INF) from the same file.
function s29dump() {
    var_dump(PHP_INT_MAX, PHP_INT_MAX + 1, PHP_INT_MIN);
    var_dump(0.1 + 0.2, 1 / 3, 3.0, 1.0E+25);
    // Shortest round-tripping digits that need all seventeen.
    $sd = [0.0015425378683206609, 1.7385704740834339E+92];
    var_dump($sd[0], $sd[1]);
    var_dump(0b111111010000101010101010101010111111111111111111111111111111111111111111111111111111);
    var_dump(true, null, "ab", [1 => "x"]);
    // var_dump takes any number of values.
    var_dump(1, 2, 3, 4, 5, 6, 7, 8, 9, "ten");
}

// ===== SECTION 30: the widened parse surface =====
// Constructs php-src's own test suite uses; each is in the ratchet because it
// unblocks real corpus files.
//
// The three PHP 8.5 forms below are pinned to php-src's own --EXPECT-- blocks under
// tests/reference/php/php-src-tests/extracted/tests/ (there is no php binary on this
// machine, so the corpus IS the specification):
//   pipe_operator/simple_userland_call  "5 |> '_test'"                  -> int(6)
//   pipe_operator/wrapped_chains        "$x |> '_test1' |> '_test2'", 5 -> int(12)
//                                       so '|>' is LEFT associative
//   pipe_operator/precedence_addition   "5 + 2 |> '_test1'"             -> int(14)
//                                       so '|>' binds LOOSER than '+'
//   pipe_operator/precedence_coalesce   "5 |> get_username(...) ?? 'd'" -> "5"
//                                       so '??' binds LOOSER than '|>'
// All four of those corpus files reproduce byte-for-byte against this grammar.
//
// Multi-level array auto-vivification ('$a[0]["k"] = 1') is covered by the '@'
// assertion and the av1..av8 block beside it - including the two step shapes that
// must create NOTHING, isset() and unset().
class S30Box { public $a, $b; public $p = 1; }
class S30Vis { public private(set) int $n = 4; }
function s30dbl($x) { return $x * 2; }
function s30inc($x) { return $x + 1; }
function &s30ref() { static $v = 7; return $v; }
function s30dnf(): (S30Box&S30Vis)|int { return 5; }
function s30id($x) { return $x; }
class S30Dyn {
    const K = "kv";
    public static $sp = 5;
    public static function dyn() { return "D"; }
}
function s30four($a, $b, $c, $d) { return $a . "-" . $b . "-" . $c . "-" . $d; }
function s30named($a = 1, $b = 2, $c = 3) { return $a . "/" . $b . "/" . $c; }
function s30rest($a, ...$r) {
    $s = "";
    foreach ($r as $v) { $s = $s === "" ? ("" . $v) : ($s . "," . $v); }
    return $a . "[" . $s . "]";
}
class S30PA {
    public function m($x, $y) { return "m" . $x . $y; }
    public static function s($x, $y) { return "s" . $x . $y; }
}

function s30() {
    // --- unbraced single-statement bodies ---
    $t = "";
    if (1) $t = $t . "a";
    if (0) $t = $t . "z"; else $t = $t . "b";
    if (0) $t = $t . "z"; else if (1) $t = $t . "c"; else $t = $t . "z";
    for ($i = 0; $i < 2; $i++) $t = $t . "d";
    foreach ([1, 2] as $v) $t = $t . "e";
    $n = 0;
    while ($n < 2) $n = $n + 1;
    do $n = $n + 1; while (0);
    check("u1", $t === "abcddee");
    check("u2", $n === 3);
    // an empty body is a body: 'foreach (...);' and 'while (0);'
    $m = 0;
    foreach ([1, 2, 3] as $q);
    while ($m > 0);
    check("u3", $q === 3 && $m === 0);
    // dangling else binds to the NEAREST if
    $d = "";
    if (1) if (0) $d = "x"; else $d = "y";
    check("u4", $d === "y");

    // --- '@' error suppression is a pass-through, and covers a whole assignment ---
    // The target is deliberately a MULTI-LEVEL one. It is pinned to
    // lang/engine_assignExecutionOrder_002.phpt, whose
    // '$ee["array entry created after f()"][f()] = "hello";' is exactly this shape and
    // whose --EXPECTF-- print_r shows the intermediate array created.
    $ar = [];
    @$ar[0]["k"] = "v";
    check("s1", @$ar[0]["k"] === "v" && $ar[0]["k"] === "v");
    check("s2", @s30dbl(4) === 8);
    // --- the rest of auto-vivification, one assertion per step shape ---
    $av = [];
    $av[0]["k"] = 1; $av[0]["j"] = 2;          // an existing level is REUSED, not replaced
    check("av1", $av[0]["k"] === 1 && $av[0]["j"] === 2);
    $av2 = null;                                // the ROOT becomes an array too
    $av2["x"]["y"]["z"] = 5;
    check("av2", $av2["x"]["y"]["z"] === 5 && count($av2) === 1);
    $av3 = [];
    $av3[]["p"] = 7;                            // a '[]' step appends the new level
    check("av3", $av3[0]["p"] === 7);
    $av4 = [];
    $av4["u"][] = 8;
    check("av4", $av4["u"][0] === 8);
    $av5 = [];
    $av5["a"]["b"] += 3; $av5["a"]["c"]++;      // compound assignment and ++ vivify
    check("av5", $av5["a"]["b"] === 3 && $av5["a"]["c"] === 1);
    $av6 = [];
    $av6r = &$av6["p"]["q"]; $av6r = 11;        // so does taking a reference
    check("av6", $av6["p"]["q"] === 11);
    $av7 = [];
    [$av7["m"]["n"]] = [42];                    // and a destructuring slot
    check("av7", $av7["m"]["n"] === 42);
    // isset() and unset() walk the same path and must create NOTHING.
    $av8 = [];
    $av8seen = isset($av8["z"]["w"]);
    unset($av8["z"]["w"]);
    check("av8", $av8seen === false && count($av8) === 0);

    // --- the pipe operator (PHP 8.5), left associative ---
    check("p1", (5 |> s30dbl(...)) === 10);
    check("p2", (5 |> s30dbl(...) |> s30inc(...)) === 11);
    check("p3", (5 |> "s30dbl") === 10);
    // '|>' binds tighter than '&&' and looser than '.', and single '|' still works
    check("p4", (5 | 2) === 7 && (5 & 3) === 1);

    // --- PHP 8.5 partial function application ---
    // Pinned to php-src's tests/partial_application/. rfc_examples_overview.phpt
    // enumerates the five forms and asserts each is equivalent to the arrow function
    // written beside it, which is exactly what pa1..pa5 check:
    //   foo(1, ?, 3, 4)   ==  fn($b)         => foo(1, $b, 3, 4)
    //   foo(1, ?, 3, ?)   ==  fn($b, $d)     => foo(1, $b, 3, $d)
    //   foo(1, ...)       ==  fn($b, $c, $d) => foo(1, $b, $c, $d)
    //   foo(1, 2, ...)    ==  fn($c, $d)     => foo(1, 2, $c, $d)
    //   foo(1, ?, 3, ...) ==  fn($b, $d)     => foo(1, $b, 3, $d)
    check("pa1", (s30four(1, ?, 3, 4))(2) === "1-2-3-4");
    check("pa2", (s30four(1, ?, 3, ?))(2, 4) === "1-2-3-4");
    check("pa3", (s30four(1, ...))(2, 3, 4) === "1-2-3-4");
    check("pa4", (s30four(1, 2, ...))(3, 4) === "1-2-3-4");
    check("pa5", (s30four(1, ?, 3, ...))(2, 4) === "1-2-3-4");
    // A NAMED placeholder stands for the parameter of that name, and the parameters
    // it does not mention keep their defaults (named_placeholders.phpt, Case 1:
    // 'foo(b: ?)' called with one argument prints int(1) / that argument / int(3)).
    check("pa6", (s30named(b: ?))(9) === "1/9/3");
    // Superfluous arguments are forwarded IFF the trailing '...' asked for them -
    // superfluous_args_are_forwarded.phpt calls 'f(?, ...)' and 'f(?)' with three
    // arguments each and its --EXPECT-- shows three, then one.
    check("pa7", (s30rest(?, ...))(1, 2, 3) === "1[2,3]");
    check("pa8", (s30rest(?))(1, 2, 3) === "1[]");
    // A method, a static method and a callable held in a variable all partial.
    $pk = new S30PA();
    check("pa9", ($pk->m(?, 2))(1) === "m12");
    check("pa10", (S30PA::s(1, ?))(2) === "s12");
    $pf = "s30four";
    check("pa11", ($pf(1, ?, 3, 4))(2) === "1-2-3-4");
    // The PHP 8.1 first-class callable is untouched: a BARE '(...)' is the function
    // itself, and it still pipes.
    check("pa12", (s30four(...))(1, 2, 3, 4) === "1-2-3-4" && (5 |> s30dbl(...)) === 10);

    // --- a '::' member may be COMPUTED ---
    // tests/bug55247.phpt writes "Test::{'method'}();" and its --EXPECT-- is the
    // method's own output; Zend/tests/int_static_prop_name.phpt writes
    // 'var_dump(Foo::${42});'; tests/varSyntax/staticMember.phpt writes
    // 'var_dump(A::$$b_str);'; tests/dynamic_call/dynamic_method_calls.phpt writes
    // 'foo::$$b();'. The constant form is Zend/tests/dynamic_class_const_fetch_*.
    $dm = "dyn";
    check("ds1", S30Dyn::{$dm}() === "D" && S30Dyn::{'dyn'}() === "D");
    $dc = "K";
    check("ds2", S30Dyn::{$dc} === "kv" && S30Dyn::{'K'} === "kv");
    $dp = "sp";
    check("ds3", S30Dyn::$$dp === 5 && S30Dyn::${'sp'} === 5);
    S30Dyn::$$dp = 11;
    check("ds4", S30Dyn::$sp === 11);
    S30Dyn::${'sp'} += 1;
    check("ds5", S30Dyn::$sp === 12 && isset(S30Dyn::$$dp));

    // --- a foreach slot is any assignable TARGET, not just a variable ---
    // tests/foreach/bug34467.phpt writes 'foreach (array (1,2,3) as $abc->k => $abc->v)'
    // and its --EXPECT-- prints "0 1 / 1 2 / 2 3"; lang/040.phpt writes
    // 'foreach($a as $b[0])'; tests/foreach/bug34873.phpt uses
    // '$this->var["key"] => $this->var["value"]'; and tests/lang/foreach_list_001.phpt
    // puts a nested list() on the value side of a '=>'.
    $fo = new S30Box();
    $ft = "";
    foreach ([1, 2, 3] as $fo->a => $fo->b) { $ft = $ft . $fo->a . $fo->b . ";"; }
    check("fe1", $ft === "01;12;23;");
    $fa = [];
    foreach ([5, 6] as $fa[0]) {}
    check("fe2", $fa[0] === 6);
    $fk = new S30Box();
    $fk->a = [];
    foreach (["p" => "q"] as $fk->a["key"] => $fk->a["value"]) {}
    check("fe3", $fk->a["key"] === "p" && $fk->a["value"] === "q");
    $fs = "";
    foreach ([[[1, 2], [3, 4]]] as $fi => list(list($p1, $p2), list($p3, $p4))) {
        $fs = $fs . $fi . $p1 . $p2 . $p3 . $p4;
    }
    check("fe4", $fs === "01234");
    // the plain and by-reference forms are untouched
    $fb = [1, 2];
    foreach ($fb as &$fv) { $fv = $fv * 10; }
    unset($fv);
    check("fe5", $fb[0] === 10 && $fb[1] === 20);

    // --- a CALL as the head of an assignment target ---
    // tests/bug37144.phpt writes 'foo()->bar[1] = "123";', 'foo()->bar[0]++;' and
    // 'unset(foo()->bar[0]);' and expects the file to run to its "ok";
    // tests/bug70332.phpt writes 'test($arg)->name[1] = "xxxx";' and its --EXPECT--
    // print_r shows the CALLER's object carrying the new element - which is what
    // makes the returned object, not a copy, the thing written into.
    $ch = new S30Box();
    $ch->a = [7];
    s30id($ch)->a[1] = "123";
    check("ch1", $ch->a[0] === 7 && $ch->a[1] === "123");
    s30id($ch)->a[0]++;
    check("ch2", $ch->a[0] === 8);
    check("ch3", isset(s30id($ch)->a[1]) && !isset(s30id($ch)->a[9]));
    unset(s30id($ch)->a[0]);
    check("ch4", !isset($ch->a[0]) && $ch->a[1] === "123");
    s30id($ch)->a["k"] ??= 42;
    check("ch5", $ch->a["k"] === 42);
    s30id($ch)->b = "p";
    check("ch6", $ch->b === "p");
    // the guard that keeps this from re-parsing every call statement must not eat
    // an ordinary one, nor a call used as an argument or a condition
    $cnt = 0;
    s30dbl(2);
    if (s30id($ch)->a[1] === "123") { $cnt = $cnt + s30dbl(s30dbl(1)); }
    check("ch7", $cnt === 4);

    // --- one declaration, several properties ---
    $bx = new S30Box();
    $bx->a = 1; $bx->b = 2;
    check("m1", $bx->a + $bx->b === 3);

    // --- computed property names, on the write side AND the read side ---
    $nm = "p";
    $bx->$nm = 9;
    check("m2", $bx->p === 9 && $bx->$nm === 9);
    $bx->{"q" . "r"} = 3;
    check("m3", $bx->qr === 3 && $bx->{"qr"} === 3);
    $bx->{1234} = "N";
    check("m4", $bx->{1234} === "N");

    // --- variable variables nest ---
    $one = "two"; $two = "three";
    $$$one = "deep";
    check("v1", $three === "deep");

    // --- '=&' from a reference-returning call degrades to a value assignment ---
    $h =& s30ref();
    check("r1", $h === 7);
    // the real reference form is untouched
    $x = 3; $y =& $x; $y = 9;
    check("r2", $x === 9);

    // --- a '&' slot in a destructuring pattern ---
    list(&$la, $lb) = [4, 5];
    check("r3", $la === 4 && $lb === 5);
    foreach ([[6, 7]] as [&$fa, $fb]) { check("r4", $fa === 6 && $fb === 7); }

    // --- an attribute group on an anonymous class ---
    $anon = new #[S30Attr(7)] class () { public $z = 8; };
    check("a1", $anon->z === 8);

    // --- clone-with (PHP 8.5) overwrites named properties on the COPY ---
    $c1 = new S30Box(); $c1->p = 1;
    $c2 = clone($c1, ["p" => 5]);
    check("c1", $c2->p === 5 && $c1->p === 1);
    $c3 = clone($c1);
    check("c2", $c3->p === 1);

    // --- PHP 8.4 asymmetric visibility, and PHP 8.2 DNF types ---
    check("d1", (new S30Vis())->n === 4);
    check("d2", s30dnf() === 5);

    // --- a declaration is a statement: class and function inside a function body ---
    if (1) {
        class S30Inner { public $w = 6; }
        function s30nested() { return 11; }
    }
    check("n1", (new S30Inner())->w === 6 && s30nested() === 11);
}

// ===== SECTION 31: isset() diagnoses NOTHING =====
// isset() and unset() are the two forms PHP 8 guarantees never to warn and never
// to raise, whatever the operand: an undeclared variable, a path through one, a
// property of null. This file declares __isset and __get (SECTION 12), which is
// the shape where the compiled halves' magic fallback must not probe a null.
//
// The string-offset rows: a NEGATIVE offset counts from the end (PHP 7.1+), so
// isset($s[-1]) is true and agrees with the read; isset($s[3]) and isset($s["x"])
// are false in PHP 8.
class S31M {
    private $d = ["k" => 3];
    public function __isset($n) { return isset($this->d[$n]); }
    public function __get($n) { return isset($this->d[$n]) ? $this->d[$n] : "none"; }
}
class S31P { public $p = 3; public $n = null; }
function s31() {
    check("is01", !isset($s31undef));
    check("is02", !isset($s31undef[0]) && !isset($s31undef["k"]["j"]));
    check("is03", !isset($s31undef->p) && !isset($s31undef->a->b));
    $o = new S31P();
    check("is04", isset($o->p) && !isset($o->n) && !isset($o->zz));
    check("is05", !isset($o->n->b) && !isset($o->zz->b));
    $m = new S31M();
    check("is06", isset($m->k) && !isset($m->nope) && $m->k === 3);
    // A reference is dereferenced before the path is walked: the root is the
    // array, not the reference CELL.
    $arr = [1, 2];
    $r = &$arr;
    check("is07", isset($r[0]) && !isset($r[9]));
    $obj = new S31P();
    $ro = &$obj;
    check("is08", isset($ro->p) && !isset($ro->zz));
    // Negative string offsets.
    $vals = ["abc", -1, -3, 1];
    $s = $vals[0];
    check("is09", isset($s[$vals[1]]) && $s[$vals[1]] === "c");
    check("is10", isset($s[$vals[2]]) && $s[$vals[2]] === "a");
    check("is11", isset($s[$vals[3]]) && $s[$vals[3]] === "b");
    // And the offsets that are NOT set: past the end, past the start, and one that
    // is not a number at all. isset() answers false for each and diagnoses nothing
    // (reading such an offset yields "", which is not null, so the walk alone is
    // not enough).
    $bad = ["abc", 3, -4, "x", 0];
    $t = $bad[0];
    check("is12", !isset($t[$bad[1]]));
    check("is13", !isset($t[$bad[2]]));
    check("is14", !isset($t[$bad[3]]) && isset($t[$bad[4]]));
    // unset() through a name that was never assigned is a no-op, not a diagnostic.
    // Nothing here may raise, and the array beside it must survive untouched.
    $keep = ["x" => 1];
    unset($s31gone[0]);
    unset($s31gone->p);
    unset($s31gone["k"]["j"]);
    unset($o->n->b);
    unset($o->zz->b);
    check("is15", count($keep) === 1 && !isset($s31gone) && isset($o->p));
    unset($keep["x"]);
    check("is16", count($keep) === 0);
}

// ===== SECTION 32: variable scope =====
// PHP has no lexical variable scope: a function sees only its own variables and
// the ones it declares `global`, a closure only what it `use`s, and a try or catch
// body shares its function's frame. An undefined variable reads null (PHP 8
// warns and continues).
$s32g = 5;
function s32local() { return isset($s32g) ? "leak" : "local"; }
function s32shadow() { $s32g = 9; return $s32g; }
function s32global() { global $s32g; $s32g = $s32g + 1; return $s32g; }
function s32undef() { return $s32nothing; }
class S32K {
    public $v = 3;
    function m() {
        $f = function () { return $this->v; };
        $a = fn() => $this->v + 1;
        return $f() * 10 + $a();
    }
}
function s32() {
    global $s32g;
    check("vs1", s32local() === "local");
    check("vs2", s32shadow() === 9 && $s32g === 5);
    check("vs3", s32global() === 6 && $s32g === 6);
    check("vs4", s32undef() === null);
    $x = 1;
    $c = function () { return isset($x) ? "leak" : "none"; };
    check("vs5", $c() === "none");
    $d = function () use ($x) { $x = $x * 2; return $x; };
    check("vs6", $d() === 2 && $d() === 2 && $x === 1);
    $e = function () use (&$x) { $x = $x + 10; return $x; };
    check("vs7", $e() === 11 && $x === 11);
    try { $t = 1; throw new Exception("e"); } catch (Exception $ex) { $k = 2; }
    check("vs8", $t === 1 && $k === 2 && $ex->getMessage() === "e");
    $s = $s32undefA . "a";
    $s32undefB++;
    $s32undefC .= "c";
    check("vs9", $s === "a" && $s32undefB === 1 && $s32undefC === "c");
    check("vs10", (new S32K())->m() === 34);
}

// ===== SECTION 33: empty() never diagnoses; an index write vivifies =====
class S33M {
    public $log = "";
    function __isset($n) { $this->log .= "i"; return $n === "yes"; }
    function __get($n) { $this->log .= "g"; return 1; }
}
function s33() {
    check("em1", empty($s33undef) && empty($s33undef[0]) && empty($s33undef[0][1]));
    $n = null;
    check("em2", empty($n->p) && empty($s33undef->p));
    $s = "ab";
    check("em3", empty($s[5]) && !empty($s[1]));
    // empty() asks __isset first and reads (__get) only what is set.
    $m = new S33M();
    check("em4", empty($m->no) && $m->log === "i");
    check("em5", !empty($m->yes) && $m->log === "iig");
    $i = 0;
    $a = [0, 5];
    check("em6", !empty($a[++$i]) && $i === 1);
    $u[0] = 1;
    $u[] = 2;
    $w = null;
    $w[] = 7;
    check("viv1", $u === [1, 2] && $w === [7]);
}

// ===== SECTION 34: a bare block, and labels at the top level =====
$s34n = 0;
{ $s34n = 10; }
s34again:
$s34n++;
if ($s34n < 13) goto s34again;
goto s34skip;
$s34n = -1;
s34skip:
function s34() {
    global $s34n;
    $i = 0;
    { $i = 5; }
    check("gt1", $s34n === 13 && $i === 5);
}

// ===== SECTION 35: a function declared inside a function is global =====
// A named function is registered in the global function table when control reaches
// its declaration, wherever that stands: inside a function, a method or an if body.
function s35outer() { function s35inner($x) { return $x * 2; } return "o"; }
class S35K { function declare() { function s35fromMethod() { return "m"; } return 0; } }
function s35() {
    $o = s35outer();
    if (true) { function s35cond() { return "c"; } }
    (new S35K())->declare();
    check("nf1", $o === "o" && s35inner(21) === 42);
    check("nf2", s35cond() === "c" && s35fromMethod() === "m");
}

// ===== SECTION 36: $GLOBALS =====
// $GLOBALS[e] is the main program's variable named e, read and written from any
// frame: a write creates it, a compound assignment and ++ go through it, an index
// path below it vivifies, isset/empty/unset see the main frame, and a by-reference
// parameter binds it. A function's own variable of the same name is untouched.
$s36a = 1;
$s36arr = [1, 2];
function s36set() {
    $GLOBALS['s36a'] = $GLOBALS['s36a'] + 10;
    $GLOBALS['s36new'] = "n";
    $GLOBALS['s36arr'][] = 3;
    $k = 's36a';
    $GLOBALS[$k]++;
    $GLOBALS['s36cat'] .= "x";
    $GLOBALS['s36deep']['k']['j'] = 7;
    return $GLOBALS['s36arr'][0] . "|" . count($GLOBALS['s36arr']);
}
function s36probe() {
    $r = isset($GLOBALS['s36a']) ? "set" : "unset";
    $r .= isset($GLOBALS['s36none']) ? ",set" : ",unset";
    $r .= empty($GLOBALS['s36none']) ? ",empty" : ",full";
    unset($GLOBALS['s36new']);
    return $r;
}
function s36inc(&$v) { $v = $v + 1; }
function s36byref() { s36inc($GLOBALS['s36a']); }
function s36local() { $s36a = "local"; return $s36a . "/" . $GLOBALS['s36a']; }
function s36() {
    global $s36a, $s36arr, $s36new, $s36cat, $s36deep;
    check("gl1", s36set() === "1|3");
    check("gl2", $s36a === 12 && $s36new === "n" && $s36cat === "x" && count($s36arr) === 3);
    check("gl3", $s36deep['k']['j'] === 7);
    check("gl4", s36probe() === "set,unset,empty" && !isset($GLOBALS['s36new']));
    s36byref();
    check("gl5", $s36a === 13 && s36local() === "local/13" && "{$GLOBALS['s36a']}" === "13");
}

// ===== SECTION 37: $GLOBALS as an array; partial application across classes =====
// A bare $GLOBALS is PHP 8.1's read-only copy of the global symbol table: an array
// of the main frame's variables, by value. A method partially applied where other
// classes declare the same method name with other parameter lists binds the
// receiver's own parameters.
class S37A { function m($x, $y) { return "m" . $x . $y; } }
class S37B { function m() { return "b"; } }
class S37C { function m() { return "c"; } }
function s37() {
    $g = $GLOBALS;
    check("glob1", isset($g["checks"]) && $g["checks"] === $GLOBALS["checks"]);
    check("glob2", in_array("failures", array_keys($g)) && !in_array("g", array_keys($g)));
    $g["checks"] = -1;
    check("glob3", $GLOBALS["checks"] !== -1);
    $n = 0;
    foreach ($GLOBALS as $k => $v) { if ($k === "checks" || $k === "failures") { $n++; } }
    check("glob4", $n == 2);
    // Each bare $GLOBALS rebuilds the copy from every name the program spells: the
    // cost must stay linear in the program's size, or ten reads exhaust the brake.
    $n = 0;
    for ($i = 0; $i < 10; $i++) { $n += count($GLOBALS) > 2 ? 1 : 0; }
    check("glob5", $n === 10);
    $a = new S37A();
    $f = $a->m(?, 2);
    check("pfa1", $f(1) === "m12" && ($a->m(?, 2))(1) === "m12" && ($a->m(1, ?))(5) === "m15");
    check("pfa2", (new S37B())->m() . (new S37C())->m() === "bc");
}

// ===== SECTION 38: the array function library =====
// array_key_exists, implode and the array_* family as the PHP 8.3 manual specifies
// them: key preservation and renumbering, the sort family (stable, flags), the
// callback functions, and the ValueError / TypeError texts. A library function is
// also a callable value ('strtoupper', strlen(...)).
function s38() {
    $a = [3, 1, 2];
    $m = ["b" => 2, "a" => 3, "c" => 1];
    check("arr1", array_key_exists(1, $a) && array_key_exists("1", $a) && !array_key_exists(5, $a)
        && array_key_exists("x", ["x" => null]) && key_exists("b", $m));
    check("arr2", implode(",", $a) === "3,1,2" && implode($a) === "312" && join("-", [1, 2]) === "1-2"
        && implode(", ", ["a", 1, 2.5, true, null]) === "a, 1, 2.5, 1, ");
    $s = $a;
    sort($s);
    $r = $a;
    rsort($r);
    check("arr3", $s === [1, 2, 3] && $r === [3, 2, 1]);
    $as = $m; asort($as);
    $ar = $m; arsort($ar);
    $ks = $m; ksort($ks);
    $kr = $m; krsort($kr);
    check("arr4", $as === ["c" => 1, "b" => 2, "a" => 3] && $ar === ["a" => 3, "b" => 2, "c" => 1]
        && $ks === ["a" => 3, "b" => 2, "c" => 1] && $kr === ["c" => 1, "b" => 2, "a" => 3]);
    $u = [5, 3, 9];
    usort($u, fn($x, $y) => $x <=> $y);
    $st = [[1, "b"], [0, "a"], [1, "a"], [0, "b"]];
    usort($st, fn($x, $y) => $x[0] <=> $y[0]);
    $ss = ["b", "B", "a", "10", "9"];
    sort($ss, SORT_STRING);
    check("arr5", $u === [3, 5, 9] && $st === [[0, "a"], [0, "b"], [1, "b"], [1, "a"]]
        && $ss === ["10", "9", "B", "a", "b"]);
    check("arr6", array_map(fn($x) => $x * 2, $a) === [6, 2, 4]
        && array_map(null, [1, 2], ["a", "b"]) === [[1, "a"], [2, "b"]]
        && array_map(fn($x) => $x, ["k" => 1]) === ["k" => 1]
        && array_map('strtoupper', ["a", "b"]) === ["A", "B"] && array_map('strlen', ["ab", "c"]) === [2, 1]);
    check("arr7", array_filter([1, 0, 2, null, 3]) === [0 => 1, 2 => 2, 4 => 3]
        && array_filter(["a" => 1, "b" => 2], fn($k) => $k == "b", ARRAY_FILTER_USE_KEY) === ["b" => 2]
        && array_filter([1, 2, 3], fn($v, $k) => $v + $k > 2, ARRAY_FILTER_USE_BOTH) === [1 => 2, 2 => 3]);
    check("arr8", array_reduce([1, 2, 3], fn($c, $x) => $c + $x, 10) === 16 && array_sum([1, 2, 3.5]) === 6.5
        && array_product([2, 3]) === 6 && array_sum([]) === 0 && array_product([]) === 1);
    check("arr9", array_merge([1, 2], ["a" => 1], [3]) === [1, 2, "a" => 1, 3]
        && array_merge(["a" => 1], ["a" => 2]) === ["a" => 2]
        && array_merge_recursive(["a" => [1], "b" => 2], ["a" => [3], "b" => 4]) === ["a" => [1, 3], "b" => [2, 4]]
        && array_replace([1, 2, 3], [1 => "b"], [3 => "d"]) === [1, "b", 3, "d"]);
    check("arr10", array_slice([1, 2, 3, 4], 1, 2) === [2, 3] && array_slice([1, 2, 3], -2) === [2, 3]
        && array_slice(["a" => 1, 5 => 2, 3], 1, 2, true) === [5 => 2, 6 => 3]
        && array_slice(["x" => 1, 2], 0) === ["x" => 1, 2]);
    $sp = [1, 2, 3, 4, 5];
    $rm = array_splice($sp, 1, 2, ["x", "y", "z"]);
    $sp2 = [1, 2, 3];
    array_splice($sp2, -1, 1);
    check("arr11", $rm === [2, 3] && $sp === [1, "x", "y", "z", 4, 5] && $sp2 === [1, 2]);
    check("arr12", array_reverse([1, 2, 3]) === [3, 2, 1] && array_reverse(["x" => 1, 2], true) === [0 => 2, "x" => 1]
        && array_reverse([5 => "a", 6 => "b"]) === ["b", "a"]);
    check("arr13", array_unique([1, "1", 2, 2.0, "a", "a"]) === [0 => 1, 2 => 2, 4 => "a"]
        && array_count_values(["a", "b", "a", 1, "1"]) === ["a" => 2, "b" => 1, 1 => 2]);
    check("arr14", array_fill(5, 3, "v") === [5 => "v", 6 => "v", 7 => "v"] && array_fill_keys(["a", "b"], 0) === ["a" => 0, "b" => 0]
        && array_combine(["a", "b"], [1, 2]) === ["a" => 1, "b" => 2] && array_flip(["a" => 1, "b" => 2]) === [1 => "a", 2 => "b"]);
    check("arr15", array_search(2, [1, 2, 3]) === 1 && array_search("2", [1, 2, 3], true) === false
        && in_array("1e1", ["10"]) && !in_array(0, ["a"]) && !in_array("1", [1], true)
        && array_keys(["a" => 1, "b" => 2, "c" => 1], 1) === ["a", "c"] && array_keys([5 => 0, 7 => 0]) === [5, 7]);
    check("arr16", array_key_first(["x" => 1, "y" => 2]) === "x" && array_key_last(["x" => 1, "y" => 2]) === "y"
        && array_key_first([]) === null && array_key_last([]) === null);
    $stk = [1, 2, 3];
    $top = array_pop($stk);
    $stk[] = 9;
    $sh = [5 => "a", "k" => "b", 9 => "c"];
    $first = array_shift($sh);
    check("arr17", $top === 3 && $stk === [1, 2, 9] && $first === "a" && $sh === ["k" => "b", 0 => "c"]);
    $us = [1, 2];
    $n1 = array_unshift($us, "x", "y");
    $pu = [1];
    $n2 = array_push($pu, 2, 3);
    check("arr18", $n1 === 4 && $us === ["x", "y", 1, 2] && $n2 === 3 && $pu === [1, 2, 3]);
    check("arr19", array_chunk([1, 2, 3, 4, 5], 2) === [[1, 2], [3, 4], [5]]
        && array_chunk(["a" => 1, "b" => 2, "c" => 3], 2, true) === [["a" => 1, "b" => 2], ["c" => 3]]
        && array_pad([1, 2], 4, 0) === [1, 2, 0, 0] && array_pad([1, 2], -4, 0) === [0, 0, 1, 2]);
    check("arr20", array_column([["id" => 1, "n" => "a"], ["id" => 2, "n" => "b"]], "n", "id") === [1 => "a", 2 => "b"]
        && array_column([["n" => "a"], ["m" => 1], ["n" => "c"]], "n") === ["a", "c"]);
    check("arr21", array_diff([1, 2, 3, 4], [2, 4]) === [0 => 1, 2 => 3]
        && array_intersect_key(["a" => 1, "b" => 2, "c" => 3], ["a" => 0, "c" => 0]) === ["a" => 1, "c" => 3]
        && array_diff_assoc(["a" => 1, "b" => 2], ["a" => 1, "b" => 3]) === ["b" => 2]
        && array_intersect([1, 2, 3], [2, 3, 4], [3, 2]) === [1 => 2, 2 => 3]
        && array_udiff([1, 5, 3], [3], fn($x, $y) => $x <=> $y) === [0 => 1, 1 => 5]);
    check("arr22", array_is_list([1, 2]) && !array_is_list([1 => 1]) && array_is_list([])
        && range(1, 5) === [1, 2, 3, 4, 5] && range(5, 1, 2) === [5, 3, 1] && range("a", "e") === ["a", "b", "c", "d", "e"]
        && range(0, 1, 0.25) === [0.0, 0.25, 0.5, 0.75, 1.0]);
    check("arr23", max(1, 5, 3) === 5 && max([1, 9, 3]) === 9 && min(4, 2, 8) === 2 && max("apple", "banana") === "banana"
        && max(1, "1") === 1 && min([0, -0.5]) === -0.5);
    check("arr24", array_change_key_case(["A" => 1, "b" => 2], CASE_UPPER) === ["A" => 1, "B" => 2]
        && reset($a) === 3 && end($a) === 2 && count($a) === 3 && sizeof($a) === 3);
    check("arr25", array_find([1, 2, 3, 4], fn($v) => $v > 2) === 3 && array_find_key(["a" => 1, "b" => 5], fn($v) => $v > 2) === "b"
        && array_any([1, 2], fn($v) => $v > 1) && !array_all([1, 2], fn($v) => $v > 1));
    $len = strlen(...);
    $ak = 'array_key_exists';
    check("arr26", $len("abc") === 3 && $ak("a", ["a" => 1]) && call_twice('count', [1, 2]) === 4);
    $e1 = "";
    try { array_combine([1], [1, 2]); } catch (ValueError $e) { $e1 = $e->getMessage(); }
    $e2 = "";
    try { array_key_exists(1, "x"); } catch (TypeError $e) { $e2 = $e->getMessage(); }
    $e3 = "";
    try { max([]); } catch (ValueError $e) { $e3 = $e->getMessage(); }
    $e4 = "";
    try { array_chunk([1], 0); } catch (ValueError $e) { $e4 = $e->getMessage(); }
    check("arr27", $e1 === "array_combine(): Argument #1 (\$keys) and argument #2 (\$values) must have the same number of elements"
        && $e2 === "array_key_exists(): Argument #2 (\$array) must be of type array, string given"
        && $e3 === "max(): Argument #1 (\$value) must contain at least one element"
        && $e4 === "array_chunk(): Argument #2 (\$length) must be greater than 0");
    // A by-reference parameter followed by a variadic one (array_push's shape).
    $rv = [1, 2];
    check("arr28", s38refvar($rv, 3, 4) === 4 && $rv === [1, 2, 3, 4] && s38refvar($rv) === 4);
    // The internal pointer: next() past the end leaves it invalid (prev() cannot
    // come back), a copy carries it, foreach never moves it, unset() of the current
    // element moves it to the successor, array_pop() resets it, and an element
    // appended at an invalid pointer becomes current.
    $p = ["x" => 1, "y" => 2, 5 => 3];
    check("ptr1", current($p) === 1 && key($p) === "x" && next($p) === 2 && key($p) === "y"
        && next($p) === 3 && key($p) === 5 && next($p) === false && key($p) === null
        && prev($p) === false && current($p) === false);
    check("ptr2", end($p) === 3 && prev($p) === 2 && reset($p) === 1 && pos($p) === 1);
    next($p);
    $q = $p;
    foreach ($p as $v) { }
    check("ptr3", next($q) === 3 && key($p) === "y" && $p == $q && $p === $q);
    unset($p["x"]);
    $k1 = key($p);
    unset($p["y"]);
    check("ptr4", $k1 === "y" && key($p) === 5 && current($p) === 3);
    $e = [];
    check("ptr5", current($e) === false && key($e) === null && end($e) === false && next($e) === false);
    $c = [1, 2, 3];
    end($c);
    array_pop($c);
    $k2 = key($c);
    next($c); next($c);
    $c[] = 9;
    check("ptr6", $k2 === 0 && key($c) === 2 && current($c) === 9);
}
function s38refvar(array &$a, ...$xs) { foreach ($xs as $x) { $a[] = $x; } return count($a); }
function call_twice($f, $v) { return $f($v) + $f($v); }

// ===== SECTION 39: the string function library and sprintf =====
// substr / strpos and the str_* family over BYTE offsets, the trim character
// lists, str_replace with arrays and its count, the comparison functions' -1/0/1
// (natural order too), levenshtein and similar_text, sprintf's flags, widths,
// precisions and conversions, and number_format - with non-ASCII text kept whole.
function s39() {
    $s = "Hello World";
    check("str1", substr($s, 0, 5) === "Hello" && substr($s, -5) === "World" && substr($s, 3, -3) === "lo Wo"
        && substr($s, 20) === "" && substr($s, -20, 3) === "Hel" && substr("abc", 1, null) === "bc" && substr("héllo", 3) === "llo");
    check("str2", strpos($s, "o") === 4 && strpos($s, "o", 5) === 7 && strpos($s, "z") === false && strpos($s, "o", -3) === false
        && strrpos($s, "o") === 7 && strrpos($s, "o", -5) === 4 && stripos($s, "WORLD") === 6 && strpos("héllo", "l") === 3);
    check("str3", str_contains($s, "lo W") && str_starts_with($s, "Hell") && str_ends_with($s, "rld") && str_ends_with($s, "")
        && !str_contains($s, "x"));
    check("str4", strtoupper("abc-é") === "ABC-é" && strtolower("ABC") === "abc" && ucfirst("hello") === "Hello"
        && lcfirst("Hello") === "hello" && ucwords("hello big-world foo_bar") === "Hello Big-world Foo_bar"
        && ucwords("hello big-world", "-") === "Hello big-World");
    check("str5", trim("  x y \n") === "x y" && trim("xxhixx", "x") === "hi" && rtrim("1.5000", "0") === "1.5"
        && ltrim("0012", "0") === "12" && trim("abcHIcba", "a..c") === "HI" && chop("ab  ") === "ab");
    check("str6", str_repeat("ab", 3) === "ababab" && str_repeat("x", 0) === "" && str_pad("5", 3, "0", STR_PAD_LEFT) === "005"
        && str_pad("ab", 7, "-=", STR_PAD_BOTH) === "-=ab-=-" && str_pad("ab", 6, "xy") === "abxyxy" && str_pad("abc", 2) === "abc");
    $cnt = 0;
    $rep = str_replace("l", "L", $s, $cnt);
    check("str7", str_replace("o", "0", $s) === "Hell0 W0rld" && str_replace(["a", "b"], ["b", "c"], "ab") === "cc"
        && str_replace(["a", "b"], "x", "abc") === "xxc" && $rep === "HeLLo WorLd" && $cnt === 3
        && str_ireplace("WORLD", "There", $s) === "Hello There" && str_replace("a", "b", ["k" => "a", "c"]) === ["k" => "b", "c"]);
    check("str8", str_split("abcde", 2) === ["ab", "cd", "e"] && strrev("abc") === "cba"
        && explode(",", "a,b,,c") === ["a", "b", "", "c"] && explode(",", "a,b,c", 2) === ["a", "b,c"]
        && explode(",", "a,b,c", -1) === ["a", "b"] && explode(",", "abc", -1) === []);
    check("str9", strcmp("a", "b") === -1 && strcmp("b", "a") === 1 && strcmp("a", "a") === 0 && strcmp("abc", "ab") === 1
        && strcasecmp("HELLO", "hello") === 0 && strncmp("abcd", "abef", 2) === 0 && strncasecmp("ABx", "aby", 3) === -1);
    check("str10", strstr("user@example.com", "@") === "@example.com" && strstr("user@example.com", "@", true) === "user"
        && strrchr("a/b/c", "/") === "/c" && stristr("HayStack", "st") === "Stack" && strstr("abc", "z") === false);
    check("str11", substr_count("hello hello", "ll") === 2 && substr_count("aaa", "aa") === 1
        && substr_replace("Hello", "J", 0, 1) === "Jello" && substr_replace("abc", "X", -1) === "abX");
    check("str12", strtr("Hi all", "ai", "eo") === "Ho ell" && strtr("Hi all", ["Hi" => "Hello", "all" => "world", "a" => "A"]) === "Hello world"
        && nl2br("a\nb\r\nc") === "a<br />\nb<br />\r\nc"
        && htmlspecialchars("<a href='x'>T&C</a>") === "&lt;a href=&#039;x&#039;&gt;T&amp;C&lt;/a&gt;"
        && addslashes("O'Re\"il") === "O\\'Re\\\"il" && stripslashes("O\\'Re") === "O'Re");
    check("str13", str_word_count("Hello fri3nd, you're looking good today!") === 7
        && str_word_count("Hello World again", 1) === ["Hello", "World", "again"]
        && wordwrap("The quick brown fox", 10, "\n", true) === "The quick\nbrown fox"
        && wordwrap("A very long woooooooooooord.", 8, "\n", true) === "A very\nlong\nwooooooo\nooooord."
        && chunk_split("abcd", 2, "|") === "ab|cd|");
    check("str14", ord("A") === 65 && ord("") === 0 && chr(97) === "a" && chr(321) === "A");
    check("fmt1", sprintf("%d items", 5) === "5 items" && sprintf("%05.2f", 3.14159) === "03.14" && sprintf("%-6s|", "ab") === "ab    |"
        && sprintf("%6s|", "ab") === "    ab|" && sprintf("%'*8s", "ab") === "******ab" && sprintf("%.3s", "abcdef") === "abc");
    check("fmt2", sprintf("%x %X %o %b", 255, 255, 8, 5) === "ff FF 10 101" && sprintf("%+d %+d", 5, -5) === "+5 -5"
        && sprintf("%05d", -42) === "-0042" && sprintf("%x", -1) === "ffffffffffffffff" && sprintf("%u", -1) === "18446744073709551615");
    check("fmt3", sprintf("%e", 1234.5) === "1.234500e+3" && sprintf("%.2e", 0.000123) === "1.23e-4" && sprintf("%f", 1.5) === "1.500000"
        && sprintf("%.1f", 0.26) === "0.3" && sprintf("%g", 0.00001234) === "1.234e-5" && sprintf("%G", 123456789.0) === "1.23457E+8");
    check("fmt4", sprintf("%2\$s %1\$s", "a", "b") === "b a" && sprintf("%c", 65) === "A" && sprintf("%%") === "%"
        && sprintf("%s", true) === "1" && vsprintf("%s-%s", ["a", "b"]) === "a-b" && sprintf("%5.1f", -2.26) === " -2.3");
    check("fmt5", number_format(1234567.891) === "1,234,568" && number_format(1234567.891, 2) === "1,234,567.89"
        && number_format(1234567.891, 2, ",", ".") === "1.234.567,89" && number_format(0.5) === "1"
        && number_format(-1234.567, 1) === "-1,234.6" && number_format(-0.4) === "0" && number_format(2.675, 2) === "2.68");
    $nat = ["img12.png", "img10.png", "IMG2.png", "img1.png"];
    natsort($nat);
    $natc = $nat;
    natcasesort($natc);
    $ns = ["10", "9", "2", "1"];
    sort($ns, SORT_NATURAL);
    check("str16", strnatcmp("img12.png", "img10.png") === 1 && strnatcmp("img2", "img10") === -1 && strcmp("img2", "img10") === 1
        && strnatcasecmp("A1", "a2") === -1 && strnatcmp("x01", "x1") === -1 && strnatcmp("a  b", "a b") === 0
        && $nat === [2 => "IMG2.png", 3 => "img1.png", 1 => "img10.png", 0 => "img12.png"]
        && $natc === [3 => "img1.png", 2 => "IMG2.png", 1 => "img10.png", 0 => "img12.png"] && $ns === ["1", "2", "9", "10"]);
    $pct = 0;
    $sim = similar_text("Hello", "World", $pct);
    check("str17", levenshtein("kitten", "sitting") === 3 && levenshtein("", "abc") === 3 && levenshtein("flaw", "lawn", 2, 3, 4) === 6
        && similar_text("bafoobar", "barfoo") === 5 && similar_text("barfoo", "bafoobar") === 3 && similar_text("World", "Word") === 4
        && $sim === 1 && $pct === 20.0);
    // A non-ASCII string keeps its characters through every byte-walking function.
    check("str18", ucwords("héllo wörld") === "Héllo Wörld" && nl2br("é\nü") === "é<br />\nü" && stripslashes("é\\'x") === "é'x"
        && strtr("héllo", "l", "L") === "héLLo" && strtr("héllo", ["é" => "e", "ll" => "LL"]) === "heLLo"
        && sprintf("é %d ü %s", 5, "ö") === "é 5 ü ö" && sprintf("%5s|", "é") === "   é|" && trim("  héllo  ") === "héllo"
        && wordwrap("héllo wörld foo", 6) === "héllo\nwörld\nfoo" && str_pad("é", 4, "*") === "é**");
    check("str19", strcmp("é", "è") === 1 && strcmp("a", "é") === -1 && strtoupper("héllo") === "HéLLO"
        && substr_count("ééé", "é") === 3 && str_replace("é", "e", "café é") === "cafe e" && explode("é", "aébéc") === ["a", "b", "c"]
        && str_word_count("héllo wörld") === 4 && strpos("aéb", "b") === 3);
    $e1 = "";
    try { explode("", "x"); } catch (ValueError $e) { $e1 = $e->getMessage(); }
    $e2 = "";
    try { str_repeat("x", -1); } catch (ValueError $e) { $e2 = $e->getMessage(); }
    $e3 = "";
    try { strpos("abc", "a", 9); } catch (ValueError $e) { $e3 = $e->getMessage(); }
    $e4 = "";
    try { sprintf("%d %d", 1); } catch (ArgumentCountError $e) { $e4 = get_class($e) . ": " . $e->getMessage(); }
    check("str15", $e1 === "explode(): Argument #1 (\$separator) cannot be empty"
        && $e2 === "str_repeat(): Argument #2 (\$times) must be greater than or equal to 0"
        && $e3 === "strpos(): Argument #3 (\$offset) must be contained in argument #1 (\$haystack)"
        && $e4 === "ArgumentCountError: 3 arguments are required, 2 given");
}

// ===== SECTION 40: early binding and references through a variable call =====
// A function declared at the top level of a file is bound before any of the
// file's statements run, so a call may stand above the declaration. A call
// through a variable (a closure, a string naming a function, a callback the
// array library makes) passes a variable BY REFERENCE exactly where the callee's
// parameter is declared '&', and by value everywhere else.
$s40_early = s40_twice(21);
function s40_twice($x) { return $x * 2; }
function s40_inc(&$n) { $n++; }
function s40() {
    global $s40_early;
    check("bind1", $s40_early === 42);
    $f = function (&$x) { $x++; };
    $a = 5;
    $f($a);
    $byVal = function ($x) { $x++; return $x; };
    $b = 5;
    $r = $byVal($b);
    check("bind2", $a === 6 && $b === 5 && $r === 6);
    $g = "s40_inc";
    $c = 1;
    $g($c);
    $arrow = fn(&$x) => $x = $x * 3;
    $arrow($c);
    check("bind3", $c === 6);
    $arr = [1, 2, 3];
    array_walk($arr, function (&$v, $k) { $v = $v * 10 + $k; });
    check("bind4", $arr === [10, 21, 32]);
}

// ===== SECTION 41: a string is a string of BYTES =====
// strlen, offsets, substr and comparison count bytes; a byte that is not part of
// a character (chr(200), "\xC8", one byte cut out of "é") stays one byte, and
// bytes put back next to each other spell the character again.
function s41() {
    $s = "h\u{e9}llo \u{20ac}!";
    check("byte1", strlen($s) === 11 && ord($s[1]) === 195 && ord($s[2]) === 169 && $s[3] === "l"
        && ord($s[-2]) === 172 && $s[1] . $s[2] === "\u{e9}");
    $n = [200, 0xe2, 0x82, 0xac];
    check("byte2", ord(chr($n[0])) === 200 && strlen(chr($n[0])) === 1 && strlen("\xc8") === 1
        && "\xc8" === chr($n[0]) && strlen("\xc3\xa9") === 2 && "\xc3\xa9" === "\u{e9}");
    $acc = "";
    for ($i = 1; $i < 4; $i++) { $acc .= chr($n[$i]); }
    $b1 = chr($n[1]);
    $b2 = chr($n[2]);
    check("byte3", $acc === "\u{20ac}" && "$b1$b2\xac" === "\u{20ac}" && strrev(strrev($s)) === $s
        && implode("", str_split($s)) === $s && strpos($s, chr($n[2])) === 8);
    $w = "h\u{e9}";
    $w[2] = "X";
    check("byte4", $w === "h" . chr(195) . "X" && strlen($w) === 3 && strcmp(chr($n[0]), "\u{e9}") === 1
        && isset($s[10]) && !isset($s[11]));
}

// ===== SECTION 42: the math library =====
// round() is PHP 8.4's: the scaled value's integral part is corrected before
// the half-way decision, so round(1.955, 2) is 1.96; ceil, floor and round
// always answer a float. intval() with a base reads like C's strtol, the
// bindec family skips what is not a digit, and decbin renders 64 bits unsigned.
function s42() {
    $v = [3.7, 5, 2.5, -2.5, 1.955, 1234.5678, -0.4, 1.005, 0.285];
    check("math1", floor($v[0]) === 3.0 && floor($v[1]) === 5.0 && ceil($v[0]) === 4.0 && ceil(-0.5) === -0.0
        && round($v[2]) === 3.0 && round($v[3]) === -3.0 && round($v[4], 2) === 1.96 && round($v[5], -2) === 1200.0
        && round($v[7], 2) === 1.01 && round($v[8], 2) === 0.29 && (string)round($v[6]) === "-0" && round(3, -1) === 0.0);
    check("math2", round($v[2], 0, PHP_ROUND_HALF_EVEN) === 2.0 && round(3.5, 0, PHP_ROUND_HALF_EVEN) === 4.0
        && round($v[2], 0, PHP_ROUND_HALF_ODD) === 3.0 && round($v[2], 0, PHP_ROUND_HALF_DOWN) === 2.0);
    check("math3", sqrt(16) === 4.0 && pow(2, 10) === 1024 && pow(2, -1) === 0.5 && fmod(-10, 3) === -1.0
        && fdiv(1, 0) === INF && is_nan(fdiv(0, 0)) && log(8, 2) === 3.0 && log10(1000) === 3.0
        && log(M_E) === 1.0 && hypot(3, 4) === 5.0 && deg2rad(180) === M_PI && is_finite(1.5) && !is_finite(INF));
    check("math4", intval("42abc") === 42 && intval("0x1A", 16) === 26 && intval("012", 0) === 10
        && intval("0b11", 0) === 3 && intval("  -42", 8) === -34 && intval("zzzzzzzzzzzzzzz", 36) === PHP_INT_MAX
        && intval(42.99) === 42);
    check("math5", bindec("111") === 7 && hexdec("ff") === 255 && octdec("777") === 511 && bindec("1 0 1") === 5
        && decbin(10) === "1010" && dechex(-1) === "ffffffffffffffff" && base_convert("zz", 36, 10) === "1295"
        && hexdec("ffffffffffffffff") === 1.8446744073709552E+19);
    $e1 = "";
    $e2 = "";
    try { sqrt("abc"); } catch (TypeError $e) { $e1 = $e->getMessage(); }
    try { log(1, 0); } catch (ValueError $e) { $e2 = $e->getMessage(); }
    check("math6", $e1 === "sqrt(): Argument #1 (\$num) must be of type float, string given"
        && $e2 === "log(): Argument #2 (\$base) must be greater than 0");
    // ** and pow() on floats are C's pow, the double nearest the exact power; no php
    // here, so the right-hand sides are this machine's libm pow (python3's float **).
    $px = [1.9687173381259235, 2.0, 1.0000005275492378, 0.9303244752705773];
    $py = [596.2128067674018, -0.5, -48986194.85211566, 5961.696604822106];
    check("math7", $px[0] ** $py[0] === 2.488229202381568E+175 && pow($px[1], $py[1]) === 0.7071067811865476
        && $px[2] ** $py[2] === 5.979866080580326E-12 && pow($px[3], $py[3]) === 1.018795851248988E-187);
}

// ===== SECTION 43: keywords in any letter case =====
// PHP's keywords are case-insensitive (manual, "List of Keywords"): `Class`,
// `FUNCTION`, `ForEach ... AS`, `RETURN`, `NEW`, `If/ElseIf/Else` are the
// keywords themselves. No PHP toolchain here: the values follow from that rule.
Abstract Class S43Base { abstract Function f(); }
CLASS S43Impl Extends S43Base { public FUNCTION f() { RETURN 7; } }
function s43() {
    $c = NEW S43Impl();
    $sum = 0;
    ForEach ([1, 2] AS $v) { $sum += $v * $c->f(); }
    $w = "";
    If ($sum > 100) { $w = "big"; } ElseIf ($sum == 21) { $w = "ok"; } Else { $w = "other"; }
    check("kwci1", $sum === 21 && $w === "ok");
}

// ===== SECTION 44: comparison edges =====
// zend_compare: a NAN against a number or a string is uncomparable (1 in either
// order), `a > b` is `b < a`, and an array is greater than any scalar that is
// not a bool or null (Zend/tests/type_coercion/nan_comp_op.phpt,
// string_to_number_comparison.phpt).
function s44() {
    $nan = fdiv(0, 0);
    $r = [0, "", "NAN", [], null, 1.5];
    check("cmp1", !($nan == $r[2]) && !($r[2] == $nan) && ($nan <=> $r[0]) === 1 && ($r[0] <=> $nan) === 1
        && !($nan < $r[0]) && !($nan <= $r[0]) && !($nan > $r[0]) && !($nan >= $r[0])
        && ($nan <=> $r[1]) === 1 && ($r[1] <=> $nan) === 1 && !($nan >= $r[1]) && !($r[5] >= $nan));
    check("cmp2", ($nan <=> $r[3]) === -1 && $nan < $r[3] && !($nan > $r[3]) && ($r[3] <=> $nan) === 1
        && ($nan <=> $r[4]) === 1 && $nan > $r[4] && $r[4] < $nan && (1 <=> $r[3]) === -1 && "a" < [1]
        && [1] > "a" && (1.75 <=> "1.75abc") === -1 && INF == "INF");
}

// ===== SECTION 45: runtime settings, handles, and the names a class answers to =====
// The precision ini setting is the ndigit of zend_gcvt for echo, (string) and
// interpolation: the exact binary value correctly rounded (dtoa mode 2), 0 counting
// as 1 and -1 meaning the shortest digits. A subnormal shows that the exact value is
// what is rounded: 1.7e-319 is 1.6999810742105611E-319, and lang/bug24640.phpt
// echoes it as 1.69998107421E-319 at precision 12. bug68797.phpt pins precision=17;
// string_to_number_comparison.phpt pins precision=0's "2".
//
// A constant expression is evaluated on first use with self:: bound to the
// declaring class (constants/bug69676_3.phpt), so it may name a constant declared
// below it, or a class declared later. Method names are case-insensitive. An
// anonymous class is named after its parent, else its first interface, else
// "class", plus "@anonymous" - all an error message shows of it (anon/012.phpt,
// anon_class_name.phpt). spl_object_id is the handle var_dump prints as #id.
class S45A { const SELFN = self::N; const N = 'A'; }
class S45B extends S45A { const N = 'B'; const IND = self::SELFN; }
class S45C { const X = S45D::Y * 2; }
class S45D { const Y = 5; }
class S45P {}
interface S45I {}
class S45M {
    function getName() { return "n"; }
    static function Make() { return "m"; }
}
// An override in another letter case still overrides: the method table is keyed
// case-insensitively, so getName() on a child that declares GETNAME() runs the
// child's, and a class's hello() beats its trait's Hello().
class S45G { function getName() { return "G"; } function viaThis() { return $this->GETname(); } }
class S45H extends S45G { function GETNAME() { return "H>" . parent::getname(); } }
trait S45T { function Hello() { return "trait"; } }
class S45K extends S45H { use S45T; function hello() { return "class"; } }
// One anonymous class per declaration SITE: two evaluations share its statics.
function s45anon() { return new class { public static $n = 0; function inc() { return ++static::$n; } }; }
function s45miss($o) {
    try { $o->nope(); } catch (Error $e) { return $e->getMessage(); }
    return "";
}
function s45() {
    check("prc1", ini_get("precision") === "14" && (string) 1.7e-319 === "1.6999810742106E-319"
        && (string) (1 / 3) === "0.33333333333333");
    check("prc2", ini_set("precision", "17") === "14" && (string) 0.1 === "0.10000000000000001"
        && "" . (1 / 3) === "0.33333333333333331" && (string) 2.2250738585072012e-308 === "2.2250738585072014E-308");
    check("prc3", ini_set("precision", 20) === "17" && (string) 0.1 === "0.10000000000000000555");
    check("prc4", ini_set("precision", 0) === "20" && (string) 1.75 === "2" && "" . 1.7e300 === "2.0E+300"
        && (1.75 <=> "1.75abc") === 1);
    check("prc5", ini_set("precision", -1) === "0" && (string) 0.1 === "0.1" && (string) (1 / 3) === "0.3333333333333333");
    check("prc6", ini_set("precision", -2) === false && ini_get("precision") === "-1"
        && ini_set("precision", 14) === "-1" && (string) 0.1 === "0.1"
        && ini_get("no.such.option") === false && ini_set("no.such.option", "1") === false);
    check("lzc1", S45A::SELFN === "A" && S45B::SELFN === "A" && S45B::IND === "A" && S45C::X === 10);
    $o = new S45M();
    check("mcs1", $o->GETNAME() === "n" && $o->getname() === "n" && S45M::make() === "m"
        && S45M::MAKE() === "m" && ($o->GetName(...))() === "n");
    check("anc1", s45miss(new class {}) === "Call to undefined method class@anonymous::nope()"
        && s45miss(new class extends S45P {}) === "Call to undefined method S45P@anonymous::nope()"
        && s45miss(new class implements S45I {}) === "Call to undefined method S45I@anonymous::nope()");
    $h = new S45H();
    $k = new S45K();
    check("mcs2", $h->getName() === "H>G" && $h->viaThis() === "H>G" && $k->GetName() === "H>G"
        && $k->Hello() === "class" && $k->HELLO() === "class");
    $x = s45anon();
    $y = s45anon();
    $n = get_class($x);
    check("anc2", $x->inc() === 1 && $y->inc() === 2 && $n === get_class($y) && $n !== get_class(new class {})
        && strstr($n, "\0", true) === "class@anonymous" && strlen($n) > 16 && $x::class === $n);
    $a = new S45M();
    $b = new S45M();
    $c = clone $a;
    check("oid1", is_int(spl_object_id($a)) && spl_object_id($a) === spl_object_id($a)
        && spl_object_id($a) !== spl_object_id($b) && spl_object_id($c) !== spl_object_id($a)
        && spl_object_id($c) !== spl_object_id($b));
}

// ===== SECTION 46: print_r, call_user_func, slot order, constant cycles =====
// print_r is zend_print_zval_r: "Array\n(" blocks, entries indented four spaces
// as "[key] => value", nested blocks eight further, visibility shown on mangled
// property names, and an object already being printed is " *RECURSION*"
// (enum/print_r.phpt for enums). A promoted constructor property takes its slot
// where the constructor stands in the class body. A constant read inside
// another constant's initializer is marked while it runs, so a cycle is
// "Cannot declare self-referencing constant" naming the reference that found a
// mark (constants/bug41633_3.phpt, constexpr/constant_expressions_self_referencing_array.phpt).
// sprintf's %g is zend_gcvt at the requested precision, the exact binary value
// correctly rounded (dtoa mode 2, half to even on an exact tie), as echo is.
// A numeric string past the double range is INF.
class S46A { public $a = 1; protected $b = [true, null]; private $c = 1.5; public $me; }
enum S46E: string { case X = "x"; }
class S46P { public $z = 0; function __construct(public $x, public $y = 2) {} public $w = 9; }
class S46C { const A = S46C::B; const B = S46C::A; const OK = self::V . "!"; const V = "v"; }
class S46D { const FOO = [self::BAR]; const BAR = [self::FOO]; }
function s46cyc($f) {
    try { return $f(); } catch (Error $e) { return $e->getMessage(); }
}
function s46() {
    $o = new S46A();
    $o->me = $o;
    $want = "S46A Object\n(\n    [a] => 1\n    [b:protected] => Array\n        (\n            [0] => 1\n            [1] => \n        )\n\n"
        . "    [c:S46A:private] => 1.5\n    [me] => S46A Object\n *RECURSION*\n)\n";
    check("prr1", print_r($o, true) === $want);
    check("prr2", print_r([1, ["k" => S46E::X]], true) === "Array\n(\n    [0] => 1\n    [1] => Array\n        (\n"
        . "            [k] => S46E Enum:string\n                (\n                    [name] => X\n"
        . "                    [value] => x\n                )\n\n        )\n\n)\n"
        && print_r(0.1 + 0.2, true) === "0.3" && print_r(false, true) === "");
    check("cuf1", call_user_func("strtoupper", "ab") === "AB" && call_user_func(fn($a, $b) => $a . $b, 1, 2) === "12"
        && call_user_func_array("str_repeat", ["x", 3]) === "xxx");
    check("ord1", array_keys((array) new S46P(1)) === ["z", "x", "y", "w"]);
    check("cyc1", s46cyc(fn() => S46C::A) === "Cannot declare self-referencing constant S46C::B"
        && s46cyc(fn() => S46C::B) === "Cannot declare self-referencing constant S46C::A"
        && s46cyc(fn() => S46D::FOO) === "Cannot declare self-referencing constant self::BAR"
        && S46C::OK === "v!");
    $big = ["1e1000", "-1e1000"];
    check("inf1", INF == $big[0] && -INF == $big[1] && $big[0] + 0 === INF && is_numeric($big[0]));
    $g = [0.1, 1 / 3, 5e-324, 2.5, 0.15];
    check("fmtg1", sprintf("%.17g", $g[0]) === "0.10000000000000001" && sprintf("%.20g", $g[1]) === "0.33333333333333331483"
        && sprintf("%g", $g[2]) === "4.94066e-324" && sprintf("%.1g|%.1g", $g[3], $g[4]) === "2|0.1");
}

// ===== SECTION 47: expression-position forms =====
// A member chain that ends in a call is an assignable head: the write lands in
// what the call returned, so it is seen through an object and lost in a by-value
// array or a null (both temporaries). print is an expression worth 1, a bare yield
// is an operand, an assignment is an operand ('1 + $w = 2'), each part of a for
// header is a comma list, 'global' takes a computed name, b"" is a plain string,
// '(void)' discards a statement's value, a match condition list may end in a
// comma, instanceof takes a parenthesized class expression, and an enum case
// takes attributes. PHP manual: Operator Precedence, Generators, for, Variable
// variables, match, Type Operators.
#[Attribute]
class S47Attr {}
class S47Box {
    public $p = 0;
    public $arr = [1];
    function me() { return $this; }
    function arr() { return $this->arr; }
    function nul() { return null; }
}
enum S47E {
    #[S47Attr]
    case A;
}
function s47gen() {
    $v = 3 + yield 5;
    yield $v;
}
function s47glob() {
    $n = "s47gv";
    global $$n;
    global ${"s47" . "gw"};
    return $s47gv + $s47gw;
}
function s47val() { return 4; }
function s47set(&$r) { $r = 5; }
function s47() {
    $b = new S47Box();
    $b->me()->p = 7;
    $b->me()->arr[] = 2;
    $b->arr()[0] = 9;
    $b->nul()[0] = 1;
    check("ex1", $b->p === 7 && count($b->arr) === 2 && $b->arr[0] === 1);
    $b->me()->p ??= 5;
    $b->me()->p += 1;
    check("ex2", $b->p === 8);
    $r = print "";
    $t = false ?: print "";
    check("ex3", $r === 1 && $t === 1);
    $z = 1 + $w = 2;
    $k = null;
    check("ex4", $z === 3 && $w === 2 && (null !== $k = 5) && $k === 5);
    $n = 0;
    for ($i = 0, $j = 3; $i < $j; $i++, $j--) { $n++; }
    $q = 0;
    for ($m = 0; $q = $m, $m < 2; $m++) { }
    check("ex5", $n === 2 && $i === 2 && $j === 1 && $q === 2);
    $GLOBALS["s47gv"] = 11;
    $GLOBALS["s47gw"] = 20;
    check("ex6", s47glob() === 31);
    check("ex7", b"x" . b'y' === "xy");
    $cnt = 0;
    (void) $cnt++;
    check("ex8", $cnt === 1);
    $mv = match (0) { 1, 0, => "z", default, => "d" };
    $md = match (5) { 1, 0, => "z", default, => "d" };
    check("ex9", $mv === "z" && $md === "d");
    $cls = "S47";
    check("ex10", $b instanceof ($cls . "Box") && !($b instanceof ("S47E")));
    check("ex11", S47E::A->name === "A");
    $got = [];
    foreach (s47gen() as $y) { $got[] = $y; }
    check("ex12", $got === [5, 3]);
    $rv =& s47val();
    check("ex13", $rv === 4);
    // An append target binds a by-reference parameter to the NEW element.
    $ap = [1];
    s47set($ap[]);
    check("ex14", $ap === [1, 5]);
}

// ===== SECTION 48: callable forms, calls through a value, numeric prefixes =====
// A callable is a closure, a function name, 'C::m', [$obj, 'm'] or ['C', 'm']
// (call_user_func / array_map take all of them). A call through a value places
// named arguments and a variadic tail as a direct call does; the variadic
// parameter also collects the named arguments no parameter took. '$cn::m()' calls
// through a class NAME in a string, and 'C::$m()' calls the static METHOD named by
// $m. A leading-numeric string contributes its whole numeric prefix, leading
// whitespace and exponent included: "1.5e3abc" + 0 is 1500.0.
class S48K {
    public $v = 7;
    static function s($a = 0) { return "s" . $a; }
    function m($a = 1, $b = 2) { return "m" . $this->v . $a . $b; }
    static function sum(...$xs) { return array_sum($xs); }
    function isum(...$xs) { return array_sum($xs); }
}
function s48v($a, ...$r) { return $a . ":" . implode(",", $r) . ":" . implode(",", array_keys($r)); }
function s48() {
    $o = new S48K();
    $f = ["S48K", "s"];
    $g = [$o, "m"];
    $h = "S48K::s";
    check("dcl1", $f(1) === "s1" && $g() === "m712" && $h(3) === "s3");
    check("dcl2", call_user_func($f, 4) === "s4" && call_user_func_array($g, [5]) === "m752" && call_user_func($h, 6) === "s6");
    check("dcl3", array_map($f, [1, 2]) === ["s1", "s2"] && array_map($g, [1, 2]) === ["m712", "m722"]);
    check("dcl4", call_user_func([$o, "isum"], 1, 2, 3) === 6 && call_user_func(["S48K", "sum"], 1, 2, 3) === 6
        && call_user_func($g, b: 5) === "m715");
    $c = function($x, $y = 7) { return $x * 10 + $y; };
    check("dna1", $c(y: 5, x: 4) === 45 && $c(x: 4) === 47 && $c(4, y: 1) === 41);
    $kw = ["y" => 2, "x" => 1];
    check("dna2", $c(...$kw) === 12 && $c(...[3, "y" => 2]) === 32 && call_user_func($c, y: 6, x: 1) === 16);
    $v = function($a, ...$r) { return $a . ":" . implode(",", $r) . ":" . implode(",", array_keys($r)); };
    $n = "s48v";
    check("dna3", $v(1, 2, 3) === "1:2,3:0,1" && $n(1, 2, 3) === "1:2,3:0,1" && $v(a: 9) === "9::");
    check("dna4", s48v(1, ...["k" => 2]) === "1:2:k" && $v(1, ...["k" => 2]) === "1:2:k");
    $fi = $o->isum(...);
    check("dna5", S48K::sum(1, 2, 3) === 6 && S48K::s(a: 5) === "s5" && $fi(1, 2) === 3);
    $t = "S48K";
    $mn = "s";
    check("dst1", $t::s(9) === "s9" && S48K::$mn(8) === "s8" && $t::sum(1, 2, 3) === 6);
    $xs = ["1.5e3abc", "12e2x", " 5e1 z", "\t-2.5E-1q", "7e", ".5e1y"];
    check("npf1", @($xs[0] + 0) === 1500.0 && @($xs[1] + 0) === 1200.0 && @($xs[2] + 0) === 50.0 && @($xs[3] + 0) === -0.25);
    check("npf2", @($xs[4] + 0) === 7 && @($xs[5] + 0) === 5.0 && (int)$xs[0] === 1500 && (int)$xs[2] === 50);
    // The edges of the int32 window the native arithmetic answers inline: a result
    // past it is still an exact int.
    $w = [-4, 2, 2147483647, 1, -2147483648];
    check("fpi1", $w[0] % $w[1] === 0 && $w[2] + $w[3] === 2147483648 && $w[4] - $w[3] === -2147483649
        && $w[4] < $w[2] && !($w[2] <= $w[4]) && $w[4] % -1 === 0);
}

// ===== SECTION 49: a class is initialized at its first use =====
// A class's constants and static property defaults are evaluated when it is first
// used - a static property access or `new` - not where it is declared
// (zend_update_class_constants): its constants first, in declaration order, each
// as an outermost read, then its statics. So a constant cycle reached from a
// static default names the link the constant pass finds, a static may read a
// constant define()d after the class, and a failed initialization runs again on
// the next use.
class S49F { const A = self::B; const B = self::A; public static $x = self::B; }
class S49L { public static $x = S49_LATE; const K = S49_LATE * 2; }
class S49B { const A = S49B::B; const B = S49B::A; public $p = 1; }
function s49try($f) {
    try { return $f(); } catch (Error $e) { return $e->getMessage(); }
}
function s49() {
    check("cin1", s49try(fn() => S49F::$x) === "Cannot declare self-referencing constant self::B"
        && s49try(fn() => S49F::$x) === "Cannot declare self-referencing constant self::B");
    define("S49_LATE", 21);
    check("cin2", S49L::$x === 21 && S49L::K === 42);
    check("cin3", s49try(fn() => new S49B()) === "Cannot declare self-referencing constant S49B::B");
}

// ===== SECTION 50: error cases and the loosest operators =====
// Writing a property of null is a catchable Error (Zend/tests/bug52614); an append
// '$a[]' read by value is "Cannot use [] for reading" (func_arg_fetch_optimization);
// a bare $GLOBALS cannot bind a by-reference parameter
// (restrict_globals/invalid_pass_by_ref). The word operators 'or' / 'and' bind
// looser than assignment and yield (generators/yield_precedence), and a keyless
// yield takes the next integer after the largest integer key used so far.
class S50N { public $o; function nul() { return $this->o; } }
function s50val($v) { return 1; }
function s50ref(&$r) { $r = 1; }
function s50noop() { return 0; }
function s50gen() { yield "a" or s50noop(); }
function s50keys() { yield 1; yield "k" => 2; yield 3; yield 10 => 4; yield 5; yield 3 => 6; yield 7; yield -5 => 8; yield 9; }
function s50err($f) {
    try { $f(); return "none"; } catch (Error $e) { return $e->getMessage(); }
}
function s50() {
    $n = new S50N();
    check("err1", s50err(function() use ($n) { $n->nul()->a = 1; }) === "Attempt to assign property \"a\" on null");
    check("err2", s50err(function() { s50val($arr[]); }) === "Cannot use [] for reading");
    check("err3", s50err(function() { s50ref($GLOBALS); }) === "s50ref(): Argument #1 (\$r) could not be passed by reference");
    $a = false or true;
    $b = true and false;
    check("lop1", $a === false && $b === true);
    $vs = [];
    foreach (s50gen() as $v) { $vs[] = $v; }
    check("lop2", $vs === ["a"]);
    $ks = [];
    foreach (s50keys() as $k => $v) { $ks[] = $k; }
    check("gky1", $ks === [0, "k", 1, 10, 11, 3, 12, -5, 13]);
}

// ===== SECTION 51: named-argument errors, class-relative callables, scalar receivers =====
// A named argument no parameter declares raises "Unknown named parameter $x", one
// naming a filled parameter "Named parameter $x overwrites previous argument"
// (Zend/tests/named_params). A callable naming a missing class raises
// 'Class "X" not found' (dynamic_call/bug68475), and 'parent::m' / 'self::m' resolve
// in the class scope of the caller, through call_user_func too. Writing a property
// of an int, string, float, bool or array is a catchable Error, and a write through
// a variable bound by reference reaches the object it holds.
function s51f($a, $b = 2) { return $a + $b; }
function s51v($a, ...$rest) { return count($rest); }
class S51T {}
class S51I { function __invoke($a = "a", $b = "b") { return $a . $b; } }
class S51A { static function hi() { return "A"; } }
class S51B extends S51A {
    static function hi() { return "B"; }
    static function viaVar() { $f = 'parent::hi'; return $f(); }
    static function viaArr() { $f = ['self', 'hi']; return $f(); }
    static function viaCuf() { return call_user_func('parent::hi'); }
    static function viaCufa() { return call_user_func_array(['self', 'hi'], []); }
}
function s51err($f) {
    try { return $f(); } catch (Error $e) { return $e->getMessage(); }
}
function s51() {
    check("nmd1", s51err(function() { return s51f(1, c: 3); }) === "Unknown named parameter \$c");
    check("nmd2", s51err(function() { return s51f(b: 5, a: 1); }) === 6);
    check("nmd3", s51err(function() { return s51f(1, a: 3); }) === "Named parameter \$a overwrites previous argument");
    check("nmd4", s51err(function() { return s51f(a: 1, a: 3); }) === "Named parameter \$a overwrites previous argument");
    check("nmd5", s51err(function() { return s51f(...[1, 'b' => 2], b: 3); }) === "Named parameter \$b overwrites previous argument");
    check("nmd6", s51err(function() { return s51v(1, z: 3); }) === 1);
    check("nmd7", s51err(function() { return new S51T(x: 1); }) === "Unknown named parameter \$x");
    $inv = new S51I();
    check("nmd8", s51err(function() use ($inv) { return $inv(b: "B"); }) === "aB");
    check("nmd9", s51err(function() use ($inv) { return $inv(c: "C"); }) === "Unknown named parameter \$c");
    $g = function ($p) { return $p; };
    check("nmd10", s51err(function() use ($g) { return $g(q: 1); }) === "Unknown named parameter \$q");
    check("cls1", s51err(function() { $c = 'S51Nope::m'; return $c(); }) === "Class \"S51Nope\" not found");
    check("cls2", s51err(function() { $c = ['S51Nope', 'm']; return $c(); }) === "Class \"S51Nope\" not found");
    check("cls3", s51err(function() { $c = 'parent::hi'; return $c(); }) === "Cannot use \"parent\" when no class scope is active");
    check("cls4", S51B::viaVar() === "A" && S51B::viaArr() === "B");
    check("cls5", S51B::viaCuf() === "A" && S51B::viaCufa() === "B");
    $vals = [5, "s", 1.5, true, [1]];
    $msgs = [];
    foreach ($vals as $v) { $msgs[] = s51err(function() use ($v) { $v->p = 1; }); }
    check("prp1", $msgs === ["Attempt to assign property \"p\" on int", "Attempt to assign property \"p\" on string",
                             "Attempt to assign property \"p\" on float", "Attempt to assign property \"p\" on bool",
                             "Attempt to assign property \"p\" on array"]);
    $o = new S51T();
    $r = &$o;
    $r->x = 1;
    $o->y = 2;
    check("prp2", $o->x === 1 && $r->y === 2);
}

// ===== SECTION 52: goto out of try, catch and nested try bodies =====
// A goto may leave a try or catch body (never enter one): every finally on the way
// out runs first, innermost first, and the jump lands on the label in whichever
// enclosing body declares it - a loop it leaves is simply left.
function s52a() {
    $log = "";
    try { $log .= "t"; goto out; } finally { $log .= "f"; }
    $log .= "X";
    out:
    return $log . "o";
}
function s52b($n) {
    $log = "";
    for ($i = 0; $i < $n; $i++) {
        try {
            try { if ($i == 2) { goto done; } $log .= $i; } finally { $log .= "f"; }
        } catch (Exception $e) { $log .= "c"; }
    }
    done:
    return $log . "d";
}
function s52c() {
    $log = "";
    $k = 0;
    again:
    $k++;
    try { throw new Exception("e" . $k); }
    catch (Exception $e) { $log .= $e->getMessage(); if ($k < 3) goto again; }
    return $log;
}
function s52d() {
    $log = "";
    try {
        $log .= "a";
        try { goto mid; } finally { $log .= "1"; }
        $log .= "X";
        mid:
        $log .= "m";
        goto end;
    } finally { $log .= "2"; }
    $log .= "Y";
    end:
    return $log;
}
function s52e() {
    $log = "";
    for ($i = 0; $i < 3; $i++) {
        try {
            try { if ($i == 1) goto skip; } finally { $log .= "a" . $i; }
        } finally { $log .= "b" . $i; }
        $log .= "n" . $i;
        skip:
    }
    return $log;
}
function s52() {
    check("gto1", s52a() === "tfo");
    check("gto2", s52b(5) === "0f1ffd");
    check("gto3", s52c() === "e1e2e3");
    check("gto4", s52d() === "a1m2");
    check("gto5", s52e() === "a0b0n0a1b1a2b2n2");
}

// ===== SECTION 53: keyed lookup in long arrays =====
// An array of more than a handful of keys is looked up through a hashed index;
// these walk the paths that must keep it right: removal, pop then append, a sort,
// a copy, and keys whose spellings collide only before folding ("01", "-0", "1").
function s53() {
    $a = [];
    for ($i = 0; $i < 40; $i++) { $a["k" . $i] = $i; $a[$i * 3] = "n" . $i; }
    unset($a["k5"], $a[9]);
    check("lix1", count($a) === 78 && $a["k6"] === 6 && $a[12] === "n4" && !isset($a["k5"]) && !array_key_exists(9, $a));
    $a["k5"] = "back";
    check("lix2", $a["k5"] === "back" && array_key_last($a) === "k5");
    ksort($a);
    arsort($a);
    check("lix3", $a["k8"] === 8 && $a[33] === "n11" && $a["k5"] === "back");
    $b = $a;
    $b["k10"] = "copy";
    check("lix4", $a["k10"] === 10 && $b["k10"] === "copy");
    $st = [];
    for ($i = 0; $i < 100; $i++) { $st[] = $i; }
    for ($i = 0; $i < 50; $i++) { array_pop($st); $st[] = -$i; }
    check("lix5", count($st) === 100 && $st[99] === -49 && $st[48] === 48);
    $m = ["1" => "a", "01" => "d", "-0" => "e", 0 => "f"];
    for ($i = 0; $i < 20; $i++) { $m["x" . $i] = $i; }
    check("lix6", $m[1] === "a" && $m["01"] === "d" && $m["-0"] === "e" && $m["0"] === "f" && count($m) === 24);
    $neg = [];
    for ($i = -20; $i < 20; $i++) { $neg[$i] = $i * 2; }
    check("lix7", $neg[-7] === -14 && $neg["-7"] === -14 && $neg[19] === 38);
}

// ===== SECTION 54: callable and dispatch error tails =====
// Calling a value that is no callable is an Error naming what it is
// (dynamic_call/dynamic_call_freeing, first_class_callable_errors). A class name is
// case-insensitive, in a callable too (dynamic_call/bug68475), and a missing class
// under '::' is 'Class "X" not found' whatever follows (static_method_non_existing_class).
// A builtin's name is a callable value. A self:: / parent:: callable forwards the
// caller's late-static-binding class, as the direct call does. An internal function
// rejects an invalid callback argument with a TypeError (bug70895,
// methods-on-non-objects-call-user-func, call_user_func_002, bug48770_2). The simple
// interpolation "$a[k]" takes a numeral key as its text (numeric_strings/neg_num_string).
class S54P {}
class S54A {
    public static function who() { return static::class; }
    public static function sm($x = "-") { return "sm" . $x; }
}
class S54B extends S54A {
    public static function fwd() {
        return call_user_func('parent::who') . " " . call_user_func(['parent', 'who']) . " "
            . call_user_func('self::who') . " " . parent::who() . " " . call_user_func('S54A::who');
    }
}
class S54C extends S54B {}
function s54err($f) {
    try { return $f(); } catch (Error $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s54() {
    $vals = [new S54P, 5, 1.5, null, [1, 2, 3]];
    $msgs = [];
    foreach ($vals as $v) { $msgs[] = s54err(function() use ($v) { return $v(); }); }
    check("cbl1", $msgs === ["Error: Object of type S54P is not callable", "Error: Value of type int is not callable",
                             "Error: Value of type float is not callable", "Error: Value of type null is not callable",
                             "Error: Array callback must have exactly two elements"]);
    $lc = 's54a::sm';
    $up = ['S54A', 'SM'];
    $cn = 's54a';
    check("cbl2", $lc(1) === "sm1" && $up(2) === "sm2" && call_user_func('S54a::sm', 3) === "sm3"
        && get_class(new $cn) === "S54A" && $cn::sm(4) === "sm4");
    $x = "m";
    check("cbl3", s54err(function() use ($x) { return S54Nope::{$x}(); }) === "Error: Class \"S54Nope\" not found"
        && s54err(function() { return S54Nope::m(); }) === "Error: Class \"S54Nope\" not found"
        && s54err(function() { return S54A::nope(); }) === "Error: Call to undefined method S54A::nope()"
        && s54err(function() { $f = 'S54A::nope'; return $f(); }) === "Error: Call to undefined method S54A::nope()");
    check("cbl4", array_filter([1, "2", 3.5, 4], 'is_int') === [0 => 1, 3 => 4]
        && array_map('gettype', [1, "a"]) === ["integer", "string"]
        && array_map('is_numeric', ["1e3", "x"]) === [true, false]);
    check("cbl5", S54B::fwd() === "S54B S54B S54B S54B S54A" && S54C::fwd() === "S54C S54C S54C S54C S54A");
    $cuf = 'call_user_func(): Argument #1 ($callback) must be a valid callback, ';
    $arr = [3, 1];
    check("cbl6", s54err(function() { return call_user_func('s54_nope'); })
            === "TypeError: " . $cuf . "function \"s54_nope\" not found or invalid function name"
        && s54err(function() { return call_user_func([null, 'm']); })
            === "TypeError: " . $cuf . "first array member is not a valid class name or object"
        && s54err(function() { return call_user_func(['S54Nope', 'm']); }) === "TypeError: " . $cuf . "class \"S54Nope\" not found"
        && s54err(function() { return call_user_func_array('S54A::nope', []); })
            === "TypeError: call_user_func_array(): Argument #1 (\$callback) must be a valid callback, class S54A does not have a method \"nope\""
        && s54err(function() { return array_map('%n', 0); })
            === "TypeError: array_map(): Argument #1 (\$callback) must be a valid callback or null, function \"%n\" not found or invalid function name"
        && s54err(function() use ($arr) { return usort($arr, 'S54A::nope'); })
            === "TypeError: usort(): Argument #2 (\$callback) must be a valid callback, class S54A does not have a method \"nope\"");
    $a = ["0x0" => 5, "-0x0" => 6, "9223372036854775808" => 9, "-9223372036854775808" => 10, "00" => 7, "-1" => 4];
    check("cbl7", "$a[0x0]|$a[-0x0]|$a[9223372036854775808]|$a[-9223372036854775808]|$a[00]|$a[-1]" === "5|6|9|10|7|4");
}

// ===== SECTION 55: array union, class names in any case, early binding, argument types =====
// '+' of two arrays is the union - the left array, then each key of the right one it
// lacks - and an array beside a non-array is a TypeError naming both types
// (Zend/tests/add_002). A class name is case-insensitive in catch, instanceof,
// method_exists, class_exists and an implements list; 'Name::class' is resolved at
// compile time, so it names an undeclared class too. A top-level class with no
// interface and no trait, whose parent is known, is bound before the file runs
// (early binding); an interface-implementing class and an enum are not.
// Closure::fromCallable checks its argument as a callback (closures/closure_from_callable_*).
// count(), abs() and array_values() refuse an argument of the wrong type with a
// TypeError, and count() asks a Countable. A 'self::' / 'static::' callback handed to
// array_map & co resolves in the caller's class scope.
$s55early = new S55Early();
$s55late = interface_exists('S55Iface') . "|" . class_exists('S55Impl') . "|" . class_exists('S55Enum') . "|"
    . class_exists('S55Kid');
class S55Early { function hi() { return "early"; } }
class S55Kid extends s55early {}
interface S55Iface {}
class S55Impl implements S55Iface {}
enum S55Enum { case A; }
class S55Cnt implements countable { function count(): int { return 7; } }
class S55Cb {
    static function inc($x) { return $x + 1; }
    static function even($x) { return $x % 2 == 0; }
    static function tag($x) { return static::class . $x; }
    static function run() {
        return implode(",", array_map('self::inc', [1, 2])) . "|" . implode(",", array_map('static::tag', [1]))
            . "|" . implode(",", array_filter([1, 2, 3, 4], ['self', 'even']));
    }
    static function bad() { return array_map('self::nope', [1]); }
    function priv() { return Closure::fromCallable([$this, 'hidden']); }
    private function hidden() { return "hid"; }
}
class S55Cb2 extends S55Cb {}
function s55dbl($x) { return $x * 2; }
function s55err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s55() {
    global $s55early, $s55late;
    $ops = [[1, 2, 'x' => 'a'], [5, 6, 7, 'x' => 'b', 'y' => 'c'], 1, "s"];
    $c = [5 => 'p'];
    $c += [5 => 'q', 9 => 'r'];
    $c[] = 's';
    check("aun1", $ops[0] + $ops[1] === [0 => 1, 1 => 2, 'x' => 'a', 2 => 7, 'y' => 'c']
        && $c === [5 => 'p', 9 => 'r', 10 => 's']);
    check("aun2", s55err(fn() => $ops[0] + $ops[2]) === "TypeError: Unsupported operand types: array + int"
        && s55err(fn() => $ops[3] - $ops[0]) === "TypeError: Unsupported operand types: string - array");
    $caught = "";
    try { throw new InvalidArgumentException("x"); } catch (logicexception $e) { $caught = "caught"; }
    check("cic1", $caught === "caught" && (new S55Kid) instanceof S55EARLY && (new S55Cnt) instanceof Countable);
    check("cic2", method_exists('s55early', 'HI') && class_exists('S55KID') && !class_exists('s55iface')
        && interface_exists('s55IFACE') && !class_exists('S55Nope'));
    check("ccl1", S55Nope::class === "S55Nope" && s55kid::class === "s55kid" && \S55A\B::class === "S55A\\B");
    check("ebd1", $s55early->hi() === "early" && $s55late === "1|||1");
    $f = Closure::fromCallable('s55dbl');
    $g = Closure::fromCallable('S55Cb::inc');
    $h = (new S55Cb)->priv();
    check("fcb1", $f(4) === 8 && $f instanceof Closure && $g(1) === 2 && $h() === "hid"
        && s55err(fn() => Closure::fromCallable('s55_nope'))
            === "TypeError: Failed to create closure from callable: function \"s55_nope\" not found or invalid function name");
    $vals = [" Ab", null, "-5"];
    check("bta1", s55err(fn() => count($vals[0])) === "TypeError: count(): Argument #1 (\$value) must be of type Countable|array, string given"
        && s55err(fn() => count($vals[1])) === "TypeError: count(): Argument #1 (\$value) must be of type Countable|array, null given"
        && count(new S55Cnt) === 7);
    check("bta2", s55err(fn() => abs($vals[0])) === "TypeError: abs(): Argument #1 (\$num) must be of type int|float, string given"
        && abs($vals[2]) === 5
        && s55err(fn() => array_values($vals[0])) === "TypeError: array_values(): Argument #1 (\$array) must be of type array, string given");
    check("rcb1", S55Cb::run() === "2,3|S55Cb1|2,4" && S55Cb2::run() === "2,3|S55Cb21|2,4"
        && s55err(fn() => S55Cb::bad())
            === "TypeError: array_map(): Argument #1 (\$callback) must be a valid callback or null, class S55Cb does not have a method \"nope\"");
}

// ===== SECTION 56: class relations, operands with no number, relative names in any case =====
// is_a() and is_subclass_of() test an object, or with allow_string a class name, as
// instanceof against a class named in any case with one leading "\" dropped;
// is_subclass_of leaves the class itself out and takes a string by default
// (inheritance/bug53727, class_alias/class_alias_018); Exception and Error implement
// Throwable. get_parent_class() answers the
// parent as declared, false for none, and refuses a name that denotes no class with a
// TypeError (lang/bug21961). An object, a closure or an array in an arithmetic or
// bitwise operator is a TypeError naming both operand types, and -$x fails as
// $x * -1 (operator_unsupported_types). A closure's debug type is Closure. self,
// parent and static are case-insensitive, in the source and in a callable.
interface S56I {}
class S56A implements S56I {
    static function m() { return "A"; }
    static function viaCuf($c) { return call_user_func($c); }
}
class S56B extends S56A {
    static function m() { return "B"; }
    static function rel() { return SELF::m() . Parent::m() . STATIC::m(); }
    static function cufs() { return call_user_func('PARENT::m') . call_user_func(['Self', 'm']) . call_user_func('sTaTiC::m'); }
}
class S56C extends S56B { static function m() { return "C"; } }
function s56err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s56() {
    $vals = [new S56C, 'S56C', 'Nope', 5, function() {}];
    check("isa1", is_a($vals[0], 'S56A') && is_a($vals[0], 's56i') && is_a($vals[0], '\S56C') && !is_a($vals[0], 'Nope')
        && !is_a($vals[1], 'S56A') && is_a($vals[1], 'S56A', true) && !is_a($vals[2], 'S56A', true) && !is_a($vals[3], 'S56A')
        && is_a($vals[4], 'Closure'));
    check("isa2", is_subclass_of($vals[0], 'S56B') && !is_subclass_of($vals[0], 'S56C') && is_subclass_of($vals[1], 'S56I')
        && !is_subclass_of($vals[1], 's56c') && !is_subclass_of($vals[1], 'S56A', false) && !is_subclass_of($vals[4], 'Closure'));
    $errs = [new TypeError("x"), 'Exception'];
    check("isa3", is_a($errs[0], 'Throwable') && $errs[0] instanceof Throwable && is_subclass_of('TypeError', 'Error')
        && is_a($errs[1], 'Throwable', true));
    check("gpc1", get_parent_class($vals[0]) === "S56B" && get_parent_class($vals[1]) === "S56B" && get_parent_class('s56b') === "S56A"
        && get_parent_class(new S56A) === false && get_parent_class($vals[4]) === false);
    check("gpc2", s56err(fn() => get_parent_class($vals[2]))
            === "TypeError: get_parent_class(): Argument #1 (\$object_or_class) must be an object or a valid class name, string given"
        && s56err(fn() => get_parent_class($vals[3]))
            === "TypeError: get_parent_class(): Argument #1 (\$object_or_class) must be an object or a valid class name, int given");
    $ops = [new stdClass, 1, 2.5, function() {}, [1]];
    check("ona1", s56err(fn() => $ops[0] + $ops[1]) === "TypeError: Unsupported operand types: stdClass + int"
        && s56err(fn() => $ops[2] * $ops[0]) === "TypeError: Unsupported operand types: float * stdClass"
        && s56err(fn() => $ops[0] % $ops[1]) === "TypeError: Unsupported operand types: stdClass % int"
        && s56err(fn() => $ops[3] - $ops[1]) === "TypeError: Unsupported operand types: Closure - int"
        && s56err(fn() => $ops[1] / $ops[0]) === "TypeError: Unsupported operand types: int / stdClass"
        && s56err(fn() => $ops[0] ** $ops[2]) === "TypeError: Unsupported operand types: stdClass ** float");
    check("ona2", s56err(fn() => $ops[0] | $ops[1]) === "TypeError: Unsupported operand types: stdClass | int"
        && s56err(fn() => $ops[1] << $ops[4]) === "TypeError: Unsupported operand types: int << array"
        && s56err(fn() => -$ops[0]) === "TypeError: Unsupported operand types: stdClass * int"
        && s56err(fn() => -$ops[4]) === "TypeError: Unsupported operand types: array * int"
        && s56err(function() use ($ops) { $x = $ops[0]; $x += 1; return $x; }) === "TypeError: Unsupported operand types: stdClass + int");
    check("dbt1", get_debug_type($ops[3]) === "Closure" && gettype($ops[3]) === "object");
    check("cxl1", class_exists('\S56A') && interface_exists('\S56I') && !class_exists('\S56I') && method_exists('\S56B', 'rel'));
    check("rel1", S56B::rel() === "BAB" && S56C::rel() === "BAC" && S56B::cufs() === "ABB" && S56C::cufs() === "ABC"
        && S56A::viaCuf('SELF::m') === "A" && S56A::viaCuf(['Self', 'm']) === "A");
}

// ===== SECTION 57: non-numeric strings, increments, closures as objects, value names =====
// A string with no numeric prefix ("abc", "") in an arithmetic or shift operator is a
// TypeError naming both operand types, where "5 apples" reads its prefix
// (lang/bug28800, lang/operators/*_variationStr); +$x is $x * 1. Two strings under &,
// | and ^ combine byte by byte ('|' to the longer one) and ~ inverts a string's bytes.
// ++ and -- follow increment_function: null++ is 1, null-- stays null, a bool does not
// change, "" becomes "1" or -1, a non-numeric string increments Perl-style and does
// not decrement, and an object or an array is a TypeError, as is ~ of one
// (tests/in-de-crement/object_cannot_incdec). A closure and a generator are objects
// of the classes Closure and Generator; get_class() with no argument names the
// caller's class. An argument TypeError names a bool by its value ("true given").
// interface_exists() knows the engine's own interfaces, and get_class, is_a,
// is_subclass_of, get_parent_class, class_exists and interface_exists are callable
// by name.
class S57K { function me() { return get_class(); } }
class S57P {}
class S57Q extends S57P {}
function s57gen() { yield 1; }
function s57err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s57() {
    $s = ["abc", "5 apples", "", " 12", "ab", "cd", "a5.9", 3];
    check("nns1", s57err(fn() => $s[0] + 1) === "TypeError: Unsupported operand types: string + int"
        && s57err(fn() => 1 - $s[2]) === "TypeError: Unsupported operand types: int - string"
        && s57err(fn() => $s[6] * 1.5) === "TypeError: Unsupported operand types: string * float"
        && s57err(fn() => $s[0] % $s[7]) === "TypeError: Unsupported operand types: string % int"
        && s57err(fn() => $s[0] << 1) === "TypeError: Unsupported operand types: string << int"
        && s57err(fn() => $s[0] & 1) === "TypeError: Unsupported operand types: string & int"
        && s57err(fn() => -$s[0]) === "TypeError: Unsupported operand types: string * int"
        && s57err(fn() => +$s[2]) === "TypeError: Unsupported operand types: string * int"
        && s57err(function() use ($s) { $x = $s[0]; $x += 1; return $x; }) === "TypeError: Unsupported operand types: string + int");
    check("nns2", $s[1] + 1 === 6 && $s[1] * 2 === 10 && $s[3] + 1 === 13 && +"5" === 5 && +"1.5" === 1.5 && +$s[1] === 5);
    check("sbw1", ($s[4] & $s[5]) === "a`" && ($s[4] | "  x") === "abx" && ($s[4] ^ "  ") === "AB"
        && ("12" & "3") === "1" && bin2hex(~$s[4]) === "9e9d" && ~5 === -6);
    $v = [null, true, "", "Az", "zz", "a9", "5", "5.5", "a!", 2147483647];
    $r = [];
    foreach ($v as $x) { $y = $x; $y++; $r[] = $y; }
    check("inc1", $r === [1, true, "1", "Ba", "aaa", "b0", 6, 6.5, "a!", 2147483648]);
    $r = [];
    foreach ($v as $x) { $y = $x; $y--; $r[] = $y; }
    check("inc2", $r === [null, true, -1, "Az", "zz", "a9", 4, 4.5, "a!", 2147483646]);
    $old = $v[6];
    $was = $old++;
    check("inc3", $was === "5" && $old === 6);
    $o = [new S57K, [1], function() {}];
    check("inc4", s57err(function() use ($o) { $x = $o[0]; $x++; }) === "TypeError: Cannot increment S57K"
        && s57err(function() use ($o) { $x = $o[0]; --$x; }) === "TypeError: Cannot decrement S57K"
        && s57err(function() use ($o) { $x = $o[1]; $x--; }) === "TypeError: Cannot decrement array"
        && s57err(function() use ($o) { $x = $o[2]; ++$x; }) === "TypeError: Cannot increment Closure"
        && s57err(fn() => ~$o[0]) === "TypeError: Cannot perform bitwise not on S57K"
        && s57err(fn() => ~$o[1]) === "TypeError: Cannot perform bitwise not on array");
    $g = s57gen();
    check("clo1", is_object($o[2]) && is_object($g) && is_object(strlen(...)) && get_class($o[2]) === "Closure"
        && get_class($g) === "Generator" && (array)$o[2] === [] && (object)$o[2] === $o[2]);
    check("clo2", s57err(fn() => get_class($v[1])) === "TypeError: get_class(): Argument #1 (\$object) must be of type object, true given"
        && s57err(fn() => get_class($v[0])) === "TypeError: get_class(): Argument #1 (\$object) must be of type object, null given"
        && (new S57K)->me() === "S57K");
    check("vnm1", s57err(fn() => count($v[1])) === "TypeError: count(): Argument #1 (\$value) must be of type Countable|array, true given"
        && s57err(fn() => array_values(false)) === "TypeError: array_values(): Argument #1 (\$array) must be of type array, false given"
        && s57err(fn() => array_key_exists(0, true)) === "TypeError: array_key_exists(): Argument #2 (\$array) must be of type array, true given"
        && s57err(fn() => implode(",", false)) === "TypeError: implode(): Argument #2 (\$array) must be of type ?array, false given");
    check("ifx1", interface_exists('Throwable') && interface_exists('\Countable') && interface_exists('arrayaccess')
        && interface_exists('Traversable') && interface_exists('Stringable') && !class_exists('Throwable')
        && !interface_exists('Exception') && !interface_exists('S57Nope'));
    check("cbn1", array_map('get_class', [new S57K, new S57Q]) === ["S57K", "S57Q"]
        && array_map('get_parent_class', [new S57Q]) === ["S57P"]
        && array_map('is_a', [new S57Q, new S57K], ['S57P', 'S57P']) === [true, false]
        && array_map('is_subclass_of', ['S57Q'], ['S57P']) === [true]
        && array_map('class_exists', ['S57K', 'S57Nope']) === [true, false]
        && call_user_func('interface_exists', 'Throwable') === true);
}

// ===== SECTION 58: Traversable, the engine's own classes, ~ of a bool or null, closures dumped =====
// A class implementing Iterator or IteratorAggregate (or an interface extending
// one) is a Traversable, a backed enum a UnitEnum and an Exception a Stringable:
// the engine's own interfaces extend those (zend_interfaces.c, zend_enum.c,
// zend_exceptions.c). Closure and Generator are classes class_exists() knows and
// is_a() / is_subclass_of() test by name. ~ of a bool or null is a TypeError
// naming the value as zend_zval_value_name does (bitwise_not_function). A
// closure literal takes a handle where it is made, and var_dump shows
// zend_closure_get_debug_info's record - name "{closure:<scope>:<line>}", file,
// line, static, this, parameter - which s58dump puts in the transcript, with an
// object reached again inside its own dump printed as *RECURSION*.
class S58It implements Iterator {
    private $i = 0;
    function current(): mixed { return $this->i; }
    function key(): mixed { return $this->i; }
    function next(): void { $this->i++; }
    function rewind(): void { $this->i = 0; }
    function valid(): bool { return $this->i < 2; }
}
class S58Ag implements IteratorAggregate {
    function getIterator(): Iterator { return new S58It; }
}
interface S58Seq extends Iterator {}
abstract class S58Sub implements S58Seq {}
enum S58E: string { case A = "a"; }
function s58err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s58() {
    check("trv1", new S58It instanceof Traversable && new S58Ag instanceof \Traversable
        && is_subclass_of('S58It', 'Traversable') && is_subclass_of(new S58Ag, 'traversable')
        && is_subclass_of('S58Sub', 'Traversable') && is_a('S58Sub', 'Iterator', true)
        && !(new S58It instanceof IteratorAggregate));
    check("trv2", S58E::A instanceof UnitEnum && S58E::A instanceof BackedEnum
        && new Exception("m") instanceof Stringable && !(new S58It instanceof Stringable));
    check("cls1", class_exists('Closure') && class_exists('\Generator') && class_exists('closure')
        && !interface_exists('Closure') && !class_exists('Traversable'));
    check("cls2", is_a('Generator', 'Traversable', true) && is_subclass_of('Generator', 'Iterator')
        && !is_subclass_of('Generator', 'Generator') && is_a('Closure', 'Closure', true)
        && !is_a('Closure', 'Closure') && !is_subclass_of('Closure', 'Iterator')
        && function() {} instanceof \Closure);
    $v = [true, false, null];
    check("bnt1", s58err(fn() => ~$v[0]) === "TypeError: Cannot perform bitwise not on true"
        && s58err(fn() => ~$v[1]) === "TypeError: Cannot perform bitwise not on false"
        && s58err(fn() => ~$v[2]) === "TypeError: Cannot perform bitwise not on null");
    $c = function() {};
    $o = new stdClass;
    check("cid1", spl_object_id($c) > 0 && spl_object_id($o) === spl_object_id($c) + 1);
}
class S58Box {
    public $n = 1;
    function bound() { return function() {}; }
    function unbound() { return static fn() => 1; }
}
function s58inner() { return function() { return fn() => 2; }; }
class S58Self {
    public $me;
    public $cb;
    function arm() { $this->cb = function() {}; $this->me = $this; return $this; }
}
function s58dump() {
    $k = 5;
    $m = [1];
    var_dump(function($a, &$b, $c = 2, ...$rest) use ($k, &$m) {});
    $m = [2];
    var_dump(fn($x) => $x + $k);
    $b = new S58Box;
    var_dump($b->bound(), $b->unbound());
    var_dump(s58inner(), s58inner()());
    // An object reached again inside its own dump is *RECURSION*.
    var_dump((new S58Self)->arm());
}

// ===== SECTION 59: magic names in closures, SPL's classes and iterator functions =====
// __FUNCTION__ and __METHOD__ are compile-time constants: in a closure both are
// the closure's own name "{closure:<scope>:<line>}", in a method __METHOD__ is
// "Class::name" (zend_try_ct_eval_magic_const). SPL is always loaded: its data
// structures, iterator_to_array / iterator_count / iterator_apply over them, and
// class_implements, whose order is ce->interfaces (zend_do_implement_interfaces,
// zend_do_inherit_interfaces: the parent's, the ones named, then each one's own
// from the end; a class naming none takes its parent's from the end).
function s59err($f) {
    try { return $f(); } catch (Throwable $e) { return get_class($e) . ": " . $e->getMessage(); }
}
function s59f() {
    $c = function () { return __FUNCTION__ . "|" . __METHOD__; };
    $d = fn() => __FUNCTION__;
    $e = function () { $g = fn() => __FUNCTION__; return $g(); };
    return [__FUNCTION__, $c(), $d(), $e(), __METHOD__];
}
class S59C {
    function m() {
        $c = function () { return __FUNCTION__ . "|" . __METHOD__ . "|" . __CLASS__; };
        return [$c(), __FUNCTION__, __METHOD__];
    }
}
class S59H {
    public $p {
        get => __FUNCTION__ . "|" . __METHOD__;
    }
}
interface S59I {}
interface S59J extends S59I {}
class S59A implements S59J, Countable { function count(): int { return 3; } }
class S59B extends S59A {}
function s59g() {
    yield 1;
    yield from new ArrayIterator(["a" => 2, "b" => 3]);
    yield from new ArrayObject([4]);
}
class S59Log implements ArrayAccess {
    public $d = [];
    public $log = "";
    function offsetExists($k): bool { $this->log .= "e" . $k; return isset($this->d[$k]); }
    function offsetGet($k): mixed { $this->log .= "g" . $k; return $this->d[$k] ?? null; }
    function offsetSet($k, $v): void {
        $this->log .= "s" . ($k ?? "_");
        if ($k === null) { $this->d[] = $v; } else { $this->d[$k] = $v; }
    }
    function offsetUnset($k): void { $this->log .= "u" . $k; unset($this->d[$k]); }
}
class S59It extends ArrayIterator {
    function current(): mixed { return parent::current() * 10; }
}
function s59() {
    $f = s59f();
    check("mag1", $f[0] === "s59f" && $f[1] === "{closure:s59f():2570}|{closure:s59f():2570}"
        && $f[2] === "{closure:s59f():2571}" && $f[3] === "{closure:{closure:s59f():2572}:2572}"
        && $f[4] === "s59f");
    $m = (new S59C)->m();
    check("mag2", $m[0] === "{closure:S59C::m():2577}|{closure:S59C::m():2577}|S59C"
        && $m[1] === "m" && $m[2] === "S59C::m" && (new S59H)->p === "\$p::get|S59H::\$p::get");
    check("spl1", class_exists('ArrayIterator') && class_exists('arrayobject') && class_exists("SplStack")
        && class_exists('SplObjectStorage') && interface_exists('SeekableIterator')
        && !class_exists('SeekableIterator') && is_subclass_of('SplQueue', 'SplDoublyLinkedList'));
    $it = new ArrayIterator(["a" => 1, "b" => 2, 7 => 3]);
    $s = "";
    foreach ($it as $k => $v) { $s .= $k . "=" . $v . ";"; }
    check("spl2", $s === "a=1;b=2;7=3;" && count($it) === 3 && $it instanceof Traversable
        && iterator_to_array($it) === ["a" => 1, "b" => 2, 7 => 3]
        && iterator_to_array($it, false) === [1, 2, 3] && iterator_count($it) === 3);
    $it->seek(1);
    check("spl3", $it->key() === "b" && $it->current() === 2 && $it->offsetExists(7)
        && $it->offsetGet("a") === 1 && s59err(fn() => $it->seek(3)) === "OutOfBoundsException: Seek position 3 is out of range");
    $ao = new ArrayObject([3, 1]);
    $ao->append(2);
    $ao->offsetSet("k", 9);
    $ao->offsetUnset(0);
    check("spl4", count($ao) === 3 && $ao->getArrayCopy() === [1 => 1, 2 => 2, "k" => 9]
        && iterator_to_array($ao) === [1 => 1, 2 => 2, "k" => 9]
        && $ao->getIterator() instanceof ArrayIterator && $ao->exchangeArray([5]) === [1 => 1, 2 => 2, "k" => 9]
        && $ao->getArrayCopy() === [5]);
    check("spl5", iterator_to_array(new S59It([1, 2])) === [10, 20]);
    $s = "";
    foreach (s59g() as $k => $v) { $s .= $k . "=" . $v . ";"; }
    check("spl11", $s === "0=1;a=2;b=3;0=4;");
    // ArrayAccess: a read is offsetGet, a write offsetSet (null key for []), a
    // compound write both, isset() offsetExists alone, empty() offsetExists then
    // offsetGet, unset() offsetUnset (zend_std_*_dimension).
    $l = new S59Log;
    $l["a"] = 1;
    $l[] = 2;
    $l["a"] += 5;
    $r = [$l["a"], isset($l["a"]), isset($l["z"]), empty($l["a"])];
    unset($l["a"]);
    [$x] = $l;
    check("arr1", $r === [6, true, false, false] && $x === 2 && $l->log === "sas_gasagaeaezeagauag0");
    $ao = new ArrayObject([]);
    $ao["x"] = 1;
    $ao[] = 5;
    $os = new SplObjectStorage;
    $o = new stdClass;
    $os[$o] = "data";
    check("arr2", $ao["x"] === 1 && $ao[0] === 5 && isset($ao["x"]) && !isset($ao["y"]) && count($ao) === 2
        && $os[$o] === "data" && isset($os[$o]) && count($os) === 1);
    unset($ao["x"], $os[$o]);
    $st = new SplStack;
    $st[] = "p";
    $st[] = "q";
    check("arr3", $ao->getArrayCopy() === [0 => 5] && count($os) === 0 && $st[1] === "q" && $st->top() === "q");
    $st = new SplStack();
    $st->push(1);
    $st->push(2);
    $st->push(3);
    check("spl6", iterator_to_array($st) === [2 => 3, 1 => 2, 0 => 1] && $st->top() === 3
        && $st->pop() === 3 && count($st) === 2 && $st->getIteratorMode() === 6
        && s59err(fn() => $st->setIteratorMode(SplDoublyLinkedList::IT_MODE_FIFO))
            === "RuntimeException: Iterators' LIFO/FIFO modes for SplStack/SplQueue objects are frozen"
        && s59err(fn() => (new SplStack)->pop()) === "RuntimeException: Can't pop from an empty datastructure"
        && s59err(fn() => $st->offsetGet(5)) === "OutOfRangeException: SplDoublyLinkedList::offsetGet(): Argument #1 (\$index) is out of range");
    $q = new SplQueue();
    $q->enqueue("x");
    $q->enqueue("y");
    $q->setIteratorMode(SplDoublyLinkedList::IT_MODE_DELETE);
    $s = "";
    foreach ($q as $k => $v) { $s .= $k . "=" . $v . ";"; }
    check("spl7", $s === "0=x;0=y;" && count($q) === 0 && $q->isEmpty());
    $os = new SplObjectStorage();
    $o1 = new stdClass;
    $o2 = new stdClass;
    $os->attach($o1, "one");
    $os->attach($o2);
    $os->attach($o1, "uno");
    $s = "";
    foreach ($os as $i => $o) { $s .= $i . ":" . ($o === $o1 ? "o1" : "o2") . ":" . var_export($os->getInfo(), true) . ";"; }
    check("spl8", count($os) === 2 && $os->contains($o2) && $os->offsetGet($o1) === "uno"
        && $s === "0:o1:'uno';1:o2:NULL;" && iterator_count($os) === 2);
    $os->detach($o1);
    check("spl9", count($os) === 1 && !$os->contains($o1)
        && s59err(fn() => $os->offsetGet($o1)) === "UnexpectedValueException: Object not found");
    check("spl10", iterator_apply(new ArrayIterator([1, 2, 3]), fn() => false) === 1
        && iterator_apply(new ArrayIterator([1, 2, 3]), fn() => true) === 3);
    check("imp1", array_keys(class_implements('S59A')) === ["S59J", "Countable", "S59I"]
        && array_keys(class_implements(new S59B)) === ["S59I", "Countable", "S59J"]
        && class_implements('S59I') === [] && class_implements('S59J') === ["S59I" => "S59I"]
        && class_implements('NoSuchClass') === false
        && array_keys(class_implements('ArrayIterator'))
            === ["SeekableIterator", "Traversable", "Iterator", "ArrayAccess", "Serializable", "Countable"]
        && array_keys(class_implements(new SplStack))
            === ["Serializable", "ArrayAccess", "Countable", "Traversable", "Iterator"]
        && class_parents(new SplStack) === ["SplDoublyLinkedList" => "SplDoublyLinkedList"]);
}
function s59dump() {
    $st = new SplStack();
    $st->push("a");
    var_dump($st, new ArrayIterator([1]));
    $os = new SplObjectStorage();
    $os->attach(new stdClass, 5);
    var_dump($os);
}

// ===== SECTION 60: magic file constants, automatic Stringable, fetches for isset =====
// __LINE__ is the line of its token, __FILE__ the script's absolute path and __DIR__
// that path's directory (zend_try_ct_eval_magic_const). A class declaring
// __toString implements Stringable after the interfaces it names
// (add_stringable_interface), and so does one taking it from a trait. A value
// 'yield from' delegates keeps its own key and leaves the generator's auto key
// alone. '??' fetches its operand for isset (BP_VAR_IS): an ArrayAccess element
// asks offsetExists before offsetGet, a string offset out of range is null.
// isset() on an ArrayObject / ArrayIterator element wants a non-null value unless
// offsetExists is overridden, while SplObjectStorage's asks only for the object
// (spl_array_has_dimension_ex, spl_object_storage_has_dimension). An engine
// exception class that extends another takes its parent's interfaces from the end
// (zend_do_inheritance_ex). An ArrayAccess element passed by reference is
// offsetGet's answer, read once.
interface S60I {}
class S60A { function __toString(): string { return "a"; } }
class S60B extends S60A { function __toString(): string { return "b"; } }
class S60C implements S60I { function __toString(): string { return "c"; } }
trait S60T { function __toString(): string { return "t"; } }
class S60D implements S60I { use S60T; }
function s60g() { yield 1; yield from [7, 8]; yield 5; }
function s60h() { yield 0; yield from s60g(); yield 3; }
class S60L implements ArrayAccess {
    public $d = ["a" => 1, "n" => null];
    public $log = "";
    function offsetExists($k): bool { $this->log .= "e" . $k; return array_key_exists($k, $this->d); }
    function offsetGet($k): mixed { $this->log .= "g" . $k; return $this->d[$k] ?? null; }
    function offsetSet($k, $v): void { $this->d[$k] = $v; }
    function offsetUnset($k): void { unset($this->d[$k]); }
}
class S60O extends ArrayObject { function offsetExists($k): bool { return true; } }
class S60P extends ArrayObject {}
class S60E extends RuntimeException {}
function s60same(&$a, &$b) { return $a === $b; }
function s60() {
    $ls = [__LINE__,
        __LINE__];
    check("fil1", $ls[1] - $ls[0] === 1 && is_int($ls[0]) && __DIR__[0] === "/"
        && __DIR__ . "/" . substr(__FILE__, strlen(__DIR__) + 1) === __FILE__ && str_ends_with(__FILE__, ".php"));
    check("str1", new S60A instanceof Stringable && new S60B instanceof Stringable && new S60D instanceof Stringable
        && !(new stdClass instanceof Stringable)
        && array_keys(class_implements('S60A')) === ["Stringable"] && array_keys(class_implements('S60B')) === ["Stringable"]
        && array_keys(class_implements('S60C')) === ["S60I", "Stringable"]
        && array_keys(class_implements('S60D')) === ["S60I", "Stringable"]);
    $s = "";
    foreach (s60h() as $k => $v) { $s .= $k . "=" . $v . ";"; }
    check("gen1", $s === "0=0;0=1;0=7;1=8;1=5;1=3;");
    $l = new S60L;
    $r = [$l["a"] ?? "d", $l["z"] ?? "d", $l["n"] ?? "d"];
    $str = "abc";
    check("coa1", $r === [1, "d", "d"] && $l->log === "eagaezengn"
        && ($str[5] ?? "d") === "d" && ($str[1] ?? "d") === "b" && ($str[-9] ?? "d") === "d");
    $n = new S60L;
    $n->d["s"] = new S60L;
    $q = [isset($n["z"]["a"]), isset($n["s"]["a"]), $n["z"]["a"] ?? 4];
    check("coa2", $q === [false, true, 4] && $n->log === "ezesgsez" && $n->d["s"]->log === "ea");
    $ao = new ArrayObject(["a" => 1, "n" => null]);
    $ai = new ArrayIterator(["n" => null]);
    $ov = new S60O(["n" => null]);
    $sub = new S60P(["n" => null, "z" => 0]);
    $os = new SplObjectStorage;
    $o = new stdClass;
    $os[$o] = null;
    check("ais1", isset($ao["a"]) && !isset($ao["n"]) && !isset($ao["z"]) && $ao->offsetExists("n") && empty($ao["n"])
        && ($ao["n"] ?? "d") === "d" && !isset($ai["n"]) && $ai->offsetExists("n")
        && isset($ov["n"]) && isset($ov["q"]) && !isset($sub["n"]) && isset($sub["z"])
        && isset($os[$o]) && empty($os[$o]));
    check("imp2", array_keys(class_implements('Exception')) === ["Throwable", "Stringable"]
        && array_keys(class_implements('RuntimeException')) === ["Stringable", "Throwable"]
        && array_keys(class_implements('InvalidArgumentException')) === ["Throwable", "Stringable"]
        && array_keys(class_implements(new TypeError)) === ["Stringable", "Throwable"]
        && array_keys(class_implements('S60E')) === ["Throwable", "Stringable"]);
    $l->log = "";
    $l["k"] = $o;
    check("aar1", s60same($l["k"], $o) && $l->log === "gk");
}

// ===== SECTION 61: magic methods in any case, ??= reads for isset, paths =====
// Method names are case-insensitive, magic ones included.
// '$t ??= e' fetches $t for isset (BP_VAR_IS: offsetExists before offsetGet at
// every level), and only when that is null does e run and $t get walked again for
// writing; every index on the path is evaluated once, before either walk. basename / dirname / pathinfo are php_basename and zend_dirname:
// '/' is the separator and trailing slashes do not count.
class S61S { public static $log = ""; public static $q; }
class S61U { function __TOSTRING() { return "u"; } }
class S61V { static function __CALLSTATIC($n, $a) { return $n . count($a); } }
class S61W { function __GET($n) { return "g" . $n; } }
class S61L implements ArrayAccess {
    public $d = ["a" => 1, "n" => null];
    public $log = "";
    function offsetExists($k): bool { $this->log .= "e" . $k; return array_key_exists($k, $this->d); }
    function offsetGet($k): mixed { $this->log .= "g" . $k; return $this->d[$k] ?? null; }
    function offsetSet($k, $v): void { $this->log .= "s" . $k; $this->d[$k] = $v; }
    function offsetUnset($k): void { unset($this->d[$k]); }
}
function s61k($k) { S61S::$log .= "k" . $k; return $k; }
function s61() {
    check("cim1", (string) new S61U === "u" && "x" . new S61U === "xu" && S61V::zz(1, 2) === "zz2"
        && (new S61W)->foo === "gfoo");
    $l = new S61L;
    $r = [$l["a"] ??= 5, $l["z"] ??= 7, $l["n"] ??= 8];
    check("caa1", $r === [1, 7, 8] && $l->log === "eagaezszengnsn" && $l->d["z"] === 7);
    $l2 = new S61L;
    $l2->d["s"] = new S61L;
    $l2["s"]["k"] ??= 9;
    $l2["s"]["k"] ??= 10;
    check("caa3", $l2->log === "esgsgsesgs" && $l2->d["s"]->log === "ekskekgk" && $l2->d["s"]->d["k"] === 9);
    $a = ["x" => 1];
    $a[s61k("x")] ??= s61k("v");
    $a[s61k("y")] ??= s61k("w");
    $a[s61k("y")] .= "t";
    $u ??= s61k("u");
    $u ??= s61k("U");
    S61S::${s61k("q")} ??= "z";
    check("caa2", S61S::$log === "kxkykwkykukq" && $a === ["x" => 1, "y" => "wt"] && $u === "u"
        && S61S::$q === "z");
    check("pth1", basename("/etc/sudoers.d", ".d") === "sudoers" && basename("/etc/") === "etc"
        && basename("/") === "" && basename(".d", ".d") === ".d" && basename("a/b.php/", ".php") === "b"
        && dirname("/etc/passwd") === "/etc" && dirname("/etc/") === "/" && dirname("a") === "."
        && dirname("") === "" && dirname("//a//b//") === "//a" && dirname("/usr/local/lib", 2) === "/usr"
        && dirname("a/b/c", 9) === ".");
    check("pth2", pathinfo("/www/inc/lib.inc.php") === ["dirname" => "/www/inc", "basename" => "lib.inc.php",
            "extension" => "php", "filename" => "lib.inc"]
        && pathinfo("") === ["basename" => "", "filename" => ""]
        && pathinfo("/p/.test") === ["dirname" => "/p", "basename" => ".test", "extension" => "test", "filename" => ""]
        && pathinfo("a.b", PATHINFO_EXTENSION) === "b" && pathinfo("ab", PATHINFO_EXTENSION) === ""
        && pathinfo("/x/a.b", PATHINFO_FILENAME | PATHINFO_EXTENSION) === "b");
    $e = "";
    try { dirname("a", 0); } catch (ValueError $x) { $e = $x->getMessage(); }
    check("pth3", $e === "dirname(): Argument #2 (\$levels) must be greater than or equal to 1");
}

// ===== SECTION 62: declared properties before magic, __unset, realpath =====
// The magic property methods run only for a property the object does not have
// (zend_std_read_property / write_property / has_property / unset_property): a
// declared `public $p;` holds null and is read, written, isset() and unset() as
// itself. Once unset() it is gone, and the magic applies to it. isset() of a magic
// property is __isset's answer alone; __get is asked only by empty() and '??'.
// realpath() resolves ".", ".." and symbolic links against the working directory,
// and answers false for a path that does not exist or that continues past a
// file (tsrm_realpath_r).
class S62M {
    public $p;
    public $k = 1;
    public $log = "";
    private $d = [];
    function __get($n) { $this->log .= "g" . $n; return $this->d[$n] ?? null; }
    function __set($n, $v) { $this->log .= "s" . $n; $this->d[$n] = $v; }
    function __isset($n) { $this->log .= "i" . $n; return isset($this->d[$n]); }
    function __unset($n) { $this->log .= "u" . $n; unset($this->d[$n]); }
}
function s62() {
    $m = new S62M;
    $r = [isset($m->p), $m->p];
    $m->p = 1;
    $r[] = isset($m->p);
    $r[] = $m->p ?? 0;
    check("mgd1", $r === [false, null, true, 1] && $m->log === "");
    unset($m->p);
    $r = [isset($m->p)];
    $m->p = 2;
    $r[] = $m->p;
    $r[] = isset($m->p);
    check("mgd2", $r === [false, 2, true] && $m->log === "ipspgpip");
    $m->log = "";
    unset($m->p);
    unset($m->q);
    unset($m->k);
    $r = [isset($m->k), $m->k];
    check("mgu1", $r === [false, null] && $m->log === "upuqikgk");
    $m2 = new S62M;
    $m2->x = 5;
    $r = [isset($m2->x), isset($m2->y), empty($m2->x)];
    check("mgi1", $r === [true, false, false] && $m2->log === "sxixiyixgx");
    $f = realpath(__FILE__);
    check("rp1", $f === __FILE__ && realpath(__DIR__) === __DIR__ && realpath(__DIR__ . "//.././" . basename(__DIR__) . "/") === __DIR__
        && realpath("/") === "/" && is_string(realpath("")) && realpath("") === realpath("."));
    check("rp2", realpath(__DIR__ . "/no-such-file") === false && realpath(__FILE__ . "/") === false
        && realpath(__FILE__ . "/..") === false && realpath(__DIR__ . "/no-such-dir/..") === false);
}

// ===== SECTION 63: visibility, uninitialised typed properties, magic guards =====
// A private property is reachable only from its declaring class, a protected one
// from any class related to the one that first declared it (zend_get_property_offset,
// is_protected_compatible_scope). Out of reach, a read goes to __get and a write to
// __set when the class has one, else both are "Cannot access ... property" Errors;
// isset() answers false, or __isset's answer. A private of a base class is invisible
// to a subclass scope, which sees the name as undeclared. Static properties are
// checked the same way, and isset() / '??' answer false / null for one out of reach.
// A typed property without a default starts UNINITIALISED: reading it is an Error
// that skips __get, isset() is false without __isset, '??' takes its right side,
// listings leave it out and var_dump prints uninitialized(type); a write
// initialises it without __set. unset() puts a typed property back in that state,
// and then the magic methods are offered it. While __get / __set / __isset /
// __unset runs for one property of one object, the same access inside it reaches
// the plain property (zend_get_property_guard). A function declared to return by
// reference hands its variable to a '??=' written through its call.
class S63A {
    private $a = 1;
    protected $b = 2;
    public $c = 3;
    private static $sa = 4;
    protected static $sb = 5;
    function getA() { return $this->a; }
    function readA($o) { return $o->a; }
    static function sa() { return self::$sa; }
}
class S63B extends S63A {
    function getB() { return $this->b; }
    function setB($v) { $this->b = $v; }
    function seeA() { return isset($this->a); }
    static function sb() { return static::$sb; }
}
class S63C extends S63A { function peekB($o) { return $o->b; } }
class S63M {
    private $p = 5;
    public $log = "";
    function __get($n) { $this->log .= "g" . $n; return "m" . $n; }
    function __set($n, $v) { $this->log .= "s" . $n; }
}
class S63T {
    public int $x;
    public ?string $s;
    public $u;
    public int $d = 3;
}
class S63G {
    public int $t;
    public $log = "";
    function __get($n) { $this->log .= "g" . $n; return $this->$n ?? "-"; }
    function __set($n, $v) { $this->log .= "s" . $n; $this->$n = $v; }
    function __isset($n) { $this->log .= "i" . $n; return isset($this->$n); }
    function __unset($n) { $this->log .= "u" . $n; unset($this->$n); }
}
function s63err($f) {
    try { $f(); } catch (Error $e) { return $e->getMessage(); }
    return "none";
}
function &s63ref() { global $s63v; return $s63v; }
function s63val() { global $s63v; return $s63v; }
function s63() {
    $o = new S63A;
    $b = new S63B;
    check("vis1", $o->c === 3 && $o->getA() === 1 && $o->readA(new S63A) === 1 && $b->getB() === 2
        && (new S63C)->peekB($b) === 2 && !$b->seeA());
    check("vis2", s63err(function() use ($o) { return $o->a; }) === "Cannot access private property S63A::\$a"
        && s63err(function() use ($o) { return $o->b; }) === "Cannot access protected property S63A::\$b"
        && s63err(function() use ($o) { $o->a = 9; }) === "Cannot access private property S63A::\$a"
        && s63err(function() use ($o) { unset($o->b); }) === "Cannot access protected property S63A::\$b"
        && !isset($o->a) && isset($o->c) && $o->getA() === 1);
    $n = "a";
    $b->setB(7);
    check("vis3", s63err(function() use ($o, $n) { return $o->$n; }) === "Cannot access private property S63A::\$a"
        && $b->getB() === 7);
    $m = new S63M;
    $r = [$m->p];
    $m->p = 1;
    check("vis4", $r === ["mp"] && $m->log === "gpsp");
    check("vis5", S63A::sa() === 4 && S63B::sb() === 5
        && s63err(function() { return S63A::$sa; }) === "Cannot access private property S63A::\$sa"
        && s63err(function() { S63A::$sb = 1; }) === "Cannot access protected property S63A::\$sb"
        && !isset(S63A::$sa) && (S63A::$sb ?? "n") === "n");
    // isset() of a static property out of reach raises nothing, so no Error takes
    // an object handle (zend_std_get_static_property_with_info under BP_VAR_IS).
    // This engine never recycles a handle; PHP reuses freed ones.
    $h = new stdClass;
    $q = isset(S63A::$sa) || isset(S63A::$sb);
    check("vis6", !$q && spl_object_id(new stdClass) === spl_object_id($h) + 1);
    $t = new S63T;
    $r = [s63err(function() use ($t) { return $t->x; }), isset($t->x), $t->x ?? "d", count((array) $t)];
    $keys = [];
    foreach ($t as $k => $v) { $keys[] = $k; }
    check("uni1", $r === ["Typed property S63T::\$x must not be accessed before initialization", false, "d", 2]
        && $keys === ["u", "d"] && print_r($t, true) === "S63T Object\n(\n    [u] => \n    [d] => 3\n)\n");
    $t->x = 5;
    $r = [$t->x, isset($t->x)];
    unset($t->x, $t->u);
    $r[] = s63err(function() use ($t) { return $t->x; });
    $c = clone $t;
    $c->x = 9;
    check("uni2", $r === [5, true, "Typed property S63T::\$x must not be accessed before initialization"]
        && $c->x === 9 && array_keys((array) $t) === ["d"] && array_keys((array) $c) === ["x", "d"]);
    $g = new S63G;
    $r = [s63err(function() use ($g) { return $g->t; }), isset($g->t)];
    $g->t = 1;
    $r[] = $g->t;
    check("grd1", $r === ["Typed property S63G::\$t must not be accessed before initialization", false, 1]
        && $g->log === "");
    unset($g->t);
    $r = [$g->t, isset($g->t)];
    $g->t = 2;
    check("grd2", $r === ["-", false] && $g->t === 2 && $g->log === "gtititst");
    $g->log = "";
    $g->dyn = 3;
    $r = [$g->dyn, isset($g->dyn), isset($g->zz), $g->zz];
    unset($g->dyn);
    unset($g->dyn);
    check("grd3", $r === [3, true, false, "-"] && $g->log === "sdynizzgzzizzudyn" && !property_exists($g, "dyn"));
    check("pex1", property_exists($g, "t") && property_exists("S63A", "a") && property_exists("S63B", "b")
        && property_exists("S63A", "sa") && !property_exists("S63A", "zz") && property_exists($m, "log"));
    global $s63v;
    s63ref()["k"] ??= 1;
    s63ref()["k"] ??= 2;
    s63val()["v"] ??= 3;
    check("rrt1", $s63v === ["k" => 1]);
}
function s63dump() {
    $t = new S63T;
    $t->s = null;
    var_dump($t);
    unset($t->d);
    var_dump($t);
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

// ===== END SECTIONS =====

function main() {
    global $checks;
    global $failures;
    s01(); // SECTION-CALL 01
    s02(); // SECTION-CALL 02
    s03(); // SECTION-CALL 03
    s04(); // SECTION-CALL 04
    s05(); // SECTION-CALL 05
    s06(); // SECTION-CALL 06
    s07(); // SECTION-CALL 07
    s08(); // SECTION-CALL 08
    s09(); // SECTION-CALL 09
    s10(); // SECTION-CALL 10
    s11(); // SECTION-CALL 11
    s12(); // SECTION-CALL 12
    s13(); // SECTION-CALL 13
    s14(); // SECTION-CALL 14
    s15(); // SECTION-CALL 15
    s16(); // SECTION-CALL 16
    s17(); // SECTION-CALL 17
    s18(); // SECTION-CALL 18
    s19(); // SECTION-CALL 19
    s20(); // SECTION-CALL 20
    s21(); // SECTION-CALL 21
    s22(); // SECTION-CALL 22
    s23(); // SECTION-CALL 23
    s24(); // SECTION-CALL 24
    s25(); // SECTION-CALL 25
    s26(); // SECTION-CALL 26
    s27(); // SECTION-CALL 27
    s28(); // SECTION-CALL 28
    s29(); // SECTION-CALL 29
    s29dump(); // SECTION-CALL 29
    s30(); // SECTION-CALL 30
    s31(); // SECTION-CALL 31
    s32(); // SECTION-CALL 32
    s33(); // SECTION-CALL 33
    s34(); // SECTION-CALL 34
    s35(); // SECTION-CALL 35
    s36(); // SECTION-CALL 36
    s37(); // SECTION-CALL 37
    s38(); // SECTION-CALL 38
    s39(); // SECTION-CALL 39
    s40(); // SECTION-CALL 40
    s41(); // SECTION-CALL 41
    s42(); // SECTION-CALL 42
    s43(); // SECTION-CALL 43
    s44(); // SECTION-CALL 44
    s45(); // SECTION-CALL 45
    s46(); // SECTION-CALL 46
    s47(); // SECTION-CALL 47
    s48(); // SECTION-CALL 48
    s49(); // SECTION-CALL 49
    s50(); // SECTION-CALL 50
    s51(); // SECTION-CALL 51
    s52(); // SECTION-CALL 52
    s53(); // SECTION-CALL 53
    s54(); // SECTION-CALL 54
    s55(); // SECTION-CALL 55
    s56(); // SECTION-CALL 56
    s57(); // SECTION-CALL 57
    s58(); // SECTION-CALL 58
    s58dump(); // SECTION-CALL 58
    s59(); // SECTION-CALL 59
    s59dump(); // SECTION-CALL 59
    s60(); // SECTION-CALL 60
    s61(); // SECTION-CALL 61
    s62(); // SECTION-CALL 62
    s63(); // SECTION-CALL 63
    s63dump(); // SECTION-CALL 63
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
    echo "full: " . $checks . " checks, " . $failures . " failures\n";
    return $failures;
}
exit(main());
