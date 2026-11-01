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
// The library's functions: the calling file's strict_types decides their
// arguments too (zend_parse_arg_*_slow), an int still widening to a float; a
// callback an internal function calls is called coercively.
class StrictStr { public function __toString(): string { return "s"; } }
$sv = ["3", 5, null, 2, 1.5];
check("strict-lib-int", strictErr(fn() => str_repeat("a", $sv[0])), "str_repeat(): Argument #2 (\$times) must be of type int, string given");
check("strict-lib-string", strictErr(fn() => str_repeat($sv[1], 2)), "str_repeat(): Argument #1 (\$string) must be of type string, int given");
check("strict-lib-null", strictErr(fn() => trim($sv[2])), "trim(): Argument #1 (\$string) must be of type string, null given");
check("strict-lib-stringable", strictErr(fn() => ucfirst(new StrictStr)), "ucfirst(): Argument #1 (\$string) must be of type string, StrictStr given");
check("strict-lib-float", strictErr(fn() => str_repeat("a", $sv[4])), "str_repeat(): Argument #2 (\$times) must be of type int, float given");
check("strict-lib-bool", strictErr(fn() => strstr("abc", "b", 1)), "strstr(): Argument #3 (\$before_needle) must be of type bool, int given");
check("strict-lib-ok", [str_repeat("a", $sv[3]), substr_count("aaa", "a", 0, $sv[2]), number_format(1.5, 1, $sv[2]), number_format($sv[1])], ["aa", 3, "1.5", "5"]);
check("strict-lib-float-param", strictErr(fn() => number_format($sv[0])), "number_format(): Argument #1 (\$num) must be of type float, string given");
check("strict-native", strictErr(fn() => strlen($sv[1])), "strlen(): Argument #1 (\$string) must be of type string, int given");
check("strict-native-int", strictErr(fn() => substr("abc", $sv[0])), "substr(): Argument #2 (\$offset) must be of type int, string given");
$sf = "strlen";
check("strict-native-dynamic", strictErr(fn() => $sf($sv[1])), "strlen(): Argument #1 (\$string) must be of type string, int given");
check("strict-callback", [array_map("str_repeat", ["a"], ["2"]), array_map("strlen", [12])], [["aa"], [2]]);
echo "strict: " . $fails . " failures\n";
exit($fails);
