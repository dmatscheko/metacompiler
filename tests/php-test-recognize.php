<?php
// PHP recognition test: a real-world-looking file that exercises syntax the grammar
// now RECOGNIZES but does not yet compile. Every unsupported construct is routed to
// notImplemented, so the whole file PARSES end to end.
//
// Because those constructs are notImplemented, this file is a SHOULD-FAIL by default:
// a normal run aborts at the first one (`declare` below) with a nonzero exit. Run with
// -warn-unsupported and each becomes a warned no-op, the file compiles, runs, and ends
// with exit(0). The interpreter (php-interpreter.abnf) and the LLVM-IR compiler
// (php-to-llvm-ir.abnf) accept it identically, and the -warn-unsupported compiler
// output is byte-identical under the goja and the frozen engines.

declare(strict_types=1);
require_once "vendor/autoload.php";

const GREETING = "hi";

// ----- enum / interface / trait / abstract class: declarations recognized, skipped -----
enum Direction: int {
    case North = 0;
    case South = 1;
}

interface Greeter {
    public function greet(): string;
}

trait HasName {
    public function name(): string { return "x"; }
}

abstract class AbstractBase {
    public const KIND = "base";
    protected int $level = 1;
    abstract public function describe(): string;
}

// ----- a class using modifiers, typed/readonly props, promotion, variadics, & default
// values: the surface is recognized (types/modifiers/promotion dropped, defaults parsed) -----
final class Greeting extends AbstractBase implements Greeter {
    use HasName;
    public readonly string $who;
    private static int $made = 0;

    public function __construct(private string $target = "world", int ...$rest) {
        $this->who = $target;
    }
    public function describe(): string { return "greeting"; }
    public function greet(): string { return "hello " . $this->who; }
}

// ----- genuine, already-supported code, mixed in the same file -----
function total(array $xs): int {
    $sum = 0;
    foreach ($xs as $x) {
        $sum = $sum + $x;
    }
    return $sum;
}

$greeter = new Greeting("php");
$hello   = $greeter?->greet();          // nullsafe operator (compiled as ->)
$sum     = total([1, 2, 3, 4]);

// ----- newly recognized (notImplemented) expression forms, assigned but not relied on -----
$asInt     = (int) "10";                    // type cast
$asFloat   = (float) $sum;                  // type cast
$copy      = clone $greeter;                // clone
$twice     = fn($n) => $n + $n;             // arrow function
$label     = match ($sum) {                 // match expression
    10      => "ten",
    default => "many",
};
$isGreeter = $greeter instanceof Greeter;   // instanceof
$kind      = AbstractBase::KIND;            // static / :: access
$eol       = PHP_EOL;                       // bareword constant reference
$anon      = new class implements Greeter { // anonymous class
    public function greet(): string { return "anon"; }
};

// ----- newly recognized (notImplemented) statement forms -----
switch ($sum) {
    case 10:
        $sum = $sum + 0;
        break;
    default:
        break;
}

$n = 0;
do {
    $n = $n + 1;
} while ($n < 3);

try {
    $risky = total([]);
} catch (\InvalidArgumentException | \TypeError $e) {
    $sum = -1;
} finally {
    $done = true;
}

if ($sum < 0) {
    throw new RuntimeException("unreachable");
}

echo "recognized ok\n";
exit(0);
