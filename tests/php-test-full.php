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
// beyond what the feature-matrix file already uses (strlen, count, ...),
// define(), superglobals, include/require/eval, fibers, Generator methods
// (->send() and friends live on the engine's Generator class), references
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
}
function s19nums() { yield 1; yield 2; yield from [3, 4]; yield 5; }
function s19keyed() { yield "a" => 1; yield "b" => 2; }
function s19bounded($limit) { $i = 0; while ($i < $limit) { yield $i; $i++; } }

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
    echo "full: " . $checks . " checks, " . $failures . " failures\n";
    return $failures;
}
exit(main());
