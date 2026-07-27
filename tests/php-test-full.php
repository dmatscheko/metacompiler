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
// strlen() measures PHP's string form of its argument. php-to-llvm-ir.abnf measured
// the JAVASCRIPT form, so strlen(true) was 4 and strlen(false) 5 while
// php-interpreter.abnf (and PHP 8) answer 1 and 0.
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
// can observe - so these lines put it in the transcript, where the matrix's
// byte-identity check and ./test.sh --cross both see it. Every expected line is
// quoted from the corpus: int(9223372036854775807) and
// float(9.223372036854776E+18) from lang/operators/add_basiclong_64bit.phpt,
// float(1.9119287772983036E+25) from lang/integer_literals/binary_64bit.phpt,
// float(INF) from the same file.
function s29dump() {
    var_dump(PHP_INT_MAX, PHP_INT_MAX + 1, PHP_INT_MIN);
    var_dump(0.1 + 0.2, 1 / 3, 3.0, 1.0E+25);
    var_dump(0b111111010000101010101010101010111111111111111111111111111111111111111111111111111111);
    var_dump(true, null, "ab", [1 => "x"]);
}

// ===== SECTION 30: the widened parse surface =====
// The constructs php-src's own test suite uses that this grammar could not read
// before 2026-07-27. Each line here unblocked real corpus files, and the count of
// files it unblocked is the reason it is in the ratchet rather than in a comment.
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
// Multi-level array auto-vivification ('$a[0]["k"] = 1') was trimmed out of the '@'
// assertion below while the compiler half could not do it. It can now (js_phviv /
// js_phvivpush / js_phvivroot in abnf/jsrtphp.go answer the walk emitRefPathV makes),
// so the assertion is back in its full form and the av1..av8 block beside it covers
// the rest of the step shapes - including the two that must create NOTHING, isset()
// and unset().
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
    // The target is deliberately a MULTI-LEVEL one: the compiler half did not
    // auto-vivify intermediate levels, so this assertion used to be trimmed to a
    // single step. It is pinned to lang/engine_assignExecutionOrder_002.phpt, whose
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
    echo "full: " . $checks . " checks, " . $failures . " failures\n";
    return $failures;
}
exit(main());
