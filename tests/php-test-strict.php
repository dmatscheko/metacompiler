<?php
declare(strict_types=1);
// declare(strict_types=1) for typed properties: a value must already be of one of
// the declared types, except that an int widens to a float; nothing else is
// coerced (zend_verify_scalar_type_hint). A file of its own, because the
// declaration governs the whole file. Exits with the failure count.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

class StrictP {
    public int $i = 0;
    public float $f = 0.0;
    public string $s = "";
    public static float $sf = 1.5;
}
function strictErr($f) {
    try { return $f(); } catch (TypeError $e) { return $e->getMessage(); }
}

$p = new StrictP;
$p->f = 5;
StrictP::$sf = 2;
check("strict-widen", [$p->f, StrictP::$sf], [5.0, 2.0]);
check("strict-int", strictErr(function () use ($p) { $p->i = "5"; }), "Cannot assign string to property StrictP::\$i of type int");
check("strict-float", strictErr(function () use ($p) { $p->i = 5.0; }), "Cannot assign float to property StrictP::\$i of type int");
check("strict-string", strictErr(function () use ($p) { $p->s = 5; }), "Cannot assign int to property StrictP::\$s of type string");
check("strict-static", strictErr(function () { StrictP::$sf = "2"; }), "Cannot assign string to property StrictP::\$sf of type float");
// Parameter and return types: the caller's strict_types refuses an argument's
// coercion, the callee's a return value's; an int still widens to a float.
function strictArg(int $i) { return $i; }
function strictFlo(float $f): float { return $f; }
function strictRet($v): int { return $v; }
check("strict-arg-widen", [strictFlo(5), strictArg(7)], [5.0, 7]);
$at = __FILE__ . " on line " . __LINE__; $msg = strictErr(fn() => strictArg("5"));
check("strict-arg", $msg, "strictArg(): Argument #1 (\$i) must be of type int, string given, called in " . $at);
$at = __FILE__ . " on line " . __LINE__; $msg = strictErr(fn() => strictArg(true));
check("strict-arg-bool", $msg, "strictArg(): Argument #1 (\$i) must be of type int, true given, called in " . $at);
check("strict-ret", strictErr(fn() => strictRet(5.0)), "strictRet(): Return value must be of type int, float returned");
echo "strict: " . $fails . " failures\n";
exit($fails);
