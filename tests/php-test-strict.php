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
echo "strict: " . $fails . " failures\n";
exit($fails);
