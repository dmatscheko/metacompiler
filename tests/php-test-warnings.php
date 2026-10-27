<?php
// The diagnostics test: warnings, notices and deprecations, which the php grammars
// raise only when compiled with -cf2 display_errors=1. It runs that way, displaying
// each diagnostic on stdout as php-cli does, and again with lines=0 as well, which
// drops the location. Every diagnostic is also checked through error_get_last():
// its type, message, file (__FILE__) and line (__LINE__), or "Unknown" and 0 with
// lines=0. A failed check prints its id and the program ends with exit($fails).
// The messages are php-src's (Zend/zend_execute.c, zend_operators.c,
// zend_closures.c, zend_object_handlers.c, zend_API.c).

$fails = 0;
$checks = 0;

function check($name, $got, $want) {
    global $fails;
    global $checks;
    $checks = $checks + 1;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        var_dump($got);
        $fails = $fails + 1;
    }
}

// The last diagnostic as [type, message, file, line], then forgotten.
function lastErr() {
    $e = error_get_last();
    error_clear_last();
    return $e === null ? null : [$e["type"], $e["message"], $e["file"], $e["line"]];
}

// lines=0 is told apart by a probe: its diagnostic has no line.
$probe = @$diagProbe;
$withLines = error_get_last()["line"] !== 0;
error_clear_last();

// The record a diagnostic raised on line $line leaves.
function want($type, $message, $line) {
    global $withLines;
    return [$type, $message, $withLines ? __FILE__ : "Unknown", $withLines ? $line : 0];
}

// ----- the level: error_reporting() and @ -----
check("level-default", error_reporting(), E_ALL);
check("no-error-yet", error_get_last(), null);
function level() { return error_reporting(); }
check("silence-level", @level(), 4437);
check("silence-restored", error_reporting(), E_ALL);
$v = @$silenced; check("silence-recorded", lastErr(), want(E_WARNING, 'Undefined variable $silenced', __LINE__));
check("level-set", error_reporting(E_ALL & ~E_WARNING), E_ALL);
$v = $unshown; check("level-hidden-recorded", lastErr(), want(E_WARNING, 'Undefined variable $unshown', __LINE__));
check("level-read", error_reporting(), E_ALL & ~E_WARNING);
// A level the operand turns back on is kept (ZEND_END_SILENCE).
function allOn() { error_reporting(E_ALL); return 1; }
@allOn();
check("silence-keeps-raised", error_reporting(), E_ALL);
// An exception leaving the operand restores the level too.
function thrower() { throw new Exception("out"); }
try { @thrower(); } catch (Exception $ex) { check("silence-catch-level", error_reporting(), E_ALL); }
check("silence-exception-restored", error_reporting(), E_ALL);
error_reporting(0);
$v = $quiet; check("level-zero-recorded", lastErr(), want(E_WARNING, 'Undefined variable $quiet', __LINE__));
error_reporting(E_ALL);

// ----- variables -----
$v = $undefA; check("undef-var", lastErr(), want(E_WARNING, 'Undefined variable $undefA', __LINE__));
$v = $undefB ?? 1; check("undef-var-coalesce", lastErr(), null);
check("undef-var-isset", [isset($undefC), lastErr()], [false, null]);
$vn = "undefD";
$v = $$vn; check("undef-varvar", lastErr(), want(E_WARNING, 'Undefined variable $undefD', __LINE__));
$undefE++; check("undef-var-inc", lastErr(), want(E_WARNING, 'Undefined variable $undefE', __LINE__));
$undefF .= "x"; check("undef-var-concat-assign", lastErr(), want(E_WARNING, 'Undefined variable $undefF', __LINE__));
$undefG ??= 1; check("undef-var-coalesce-assign", lastErr(), null);

// ----- array elements and string offsets -----
$a = ["x" => 1, 5 => 2];
$v = $a["y"]; check("undef-key-string", lastErr(), want(E_WARNING, 'Undefined array key "y"', __LINE__));
$v = $a[7]; check("undef-key-int", lastErr(), want(E_WARNING, 'Undefined array key 7', __LINE__));
$v = $a["8"]; check("undef-key-numeric-string", lastErr(), want(E_WARNING, 'Undefined array key 8', __LINE__));
$v = $a["08"]; check("undef-key-non-canonical", lastErr(), want(E_WARNING, 'Undefined array key "08"', __LINE__));
$v = $a[null]; check("undef-key-null", lastErr(), want(E_WARNING, 'Undefined array key ""', __LINE__));
$v = $a[6.5]; check("key-float-then-undef", lastErr(), want(E_WARNING, 'Undefined array key 6', __LINE__));
$v = $a[5.5]; check("key-float-value", $v, 2);
$a2 = [1 => "one"];
$v = $a2[1.5]; check("key-float-deprecated", [$v, lastErr()], ["one", want(E_DEPRECATED, 'Implicit conversion from float 1.5 to int loses precision', __LINE__)]);
$v = $a["x"] ?? 0; check("undef-key-coalesce", lastErr(), null);
$v = isset($a["q"]); check("undef-key-isset", lastErr(), null);
$a["k"]++; check("undef-key-inc", [$a["k"], lastErr()], [1, want(E_WARNING, 'Undefined array key "k"', __LINE__)]);
$a["j"] .= "z"; check("undef-key-concat-assign", lastErr(), want(E_WARNING, 'Undefined array key "j"', __LINE__));
$a["n"][] = 1; check("append-no-diagnostic", lastErr(), null);
$s = "abc";
$v = $s[5]; check("string-offset", [$v, lastErr()], ["", want(E_WARNING, 'Uninitialized string offset 5', __LINE__)]);
$v = $s[-4]; check("string-offset-negative", lastErr(), want(E_WARNING, 'Uninitialized string offset -4', __LINE__));
$v = $s["1x"]; check("string-offset-illegal", lastErr(), want(E_WARNING, 'Illegal string offset "1x"', __LINE__));
$n = null;
$v = $n["k"]; check("offset-on-null", lastErr(), want(E_WARNING, 'Trying to access array offset on null', __LINE__));
$t = true;
$v = $t[0]; check("offset-on-true", lastErr(), want(E_WARNING, 'Trying to access array offset on true', __LINE__));
$i = 3;
$v = $i[0]; check("offset-on-int", lastErr(), want(E_WARNING, 'Trying to access array offset on int', __LINE__));

// ----- properties -----
class WarnP { public $p = 1; public function __construct() {} }
class WarnMagic { public function __get($n) { return "magic"; } }
$o = new WarnP();
$v = $o->q; check("undef-prop", [$v, lastErr()], [null, want(E_WARNING, 'Undefined property: WarnP::$q', __LINE__)]);
$pn = "r";
$v = $o->$pn; check("undef-prop-dynamic", lastErr(), want(E_WARNING, 'Undefined property: WarnP::$r', __LINE__));
$v = (new WarnMagic())->q; check("magic-get-no-diagnostic", [$v, lastErr()], ["magic", null]);
$v = $n->p; check("prop-on-null", [$v, lastErr()], [null, want(E_WARNING, 'Attempt to read property "p" on null', __LINE__)]);
$v = $i->p; check("prop-on-int", lastErr(), want(E_WARNING, 'Attempt to read property "p" on int', __LINE__));
$v = $n->p ?? "d"; check("prop-on-null-coalesce", [$v, lastErr()], ["d", null]);
$o->c++; check("undef-prop-inc", [$o->c, lastErr()], [1, want(E_WARNING, 'Undefined property: WarnP::$c', __LINE__)]);

// ----- ++ and -- -----
$b = true; $b++; check("inc-bool", lastErr(), want(E_WARNING, 'Increment on type bool has no effect, this will change in the next major version of PHP', __LINE__));
$z = null; $z--; check("dec-null", [$z, lastErr()], [null, want(E_WARNING, 'Decrement on type null has no effect, this will change in the next major version of PHP', __LINE__)]);
$z = null; $z++; check("inc-null", [$z, lastErr()], [1, null]);
$w = "a-b"; $w++; check("inc-non-alnum", [$w, lastErr()], ["a-c", want(E_DEPRECATED, 'Increment on non-alphanumeric string is deprecated', __LINE__)]);
$w = ""; $w--; check("dec-empty", [$w, lastErr()], [-1, want(E_DEPRECATED, 'Decrement on empty string is deprecated as non-numeric', __LINE__)]);
$w = "abc"; $w--; check("dec-non-numeric", [$w, lastErr()], ["abc", want(E_DEPRECATED, 'Decrement on non-numeric string has no effect and is deprecated', __LINE__)]);
$w = "Az"; $w++; check("inc-alnum", [$w, lastErr()], ["Ba", null]);

// ----- operators -----
$ops = [1.5, "5 apples", "1.5", 3, [1]];
$v = $ops[0] % 2; check("mod-float", [$v, lastErr()], [1, want(E_DEPRECATED, 'Implicit conversion from float 1.5 to int loses precision', __LINE__)]);
$v = $ops[0] | 4; check("bitor-float", [$v, lastErr()], [5, want(E_DEPRECATED, 'Implicit conversion from float 1.5 to int loses precision', __LINE__)]);
$v = 1 << $ops[0]; check("shift-float", [$v, lastErr()], [2, want(E_DEPRECATED, 'Implicit conversion from float 1.5 to int loses precision', __LINE__)]);
$v = ~$ops[0]; check("bitnot-float", [$v, lastErr()], [-2, want(E_DEPRECATED, 'Implicit conversion from float 1.5 to int loses precision', __LINE__)]);
$v = $ops[2] % 2; check("mod-float-string", lastErr(), want(E_DEPRECATED, 'Implicit conversion from float-string "1.5" to int loses precision', __LINE__));
$v = $ops[1] + 1; check("add-leading-numeric", [$v, lastErr()], [6, want(E_WARNING, 'A non-numeric value encountered', __LINE__)]);
$v = $ops[1] % 3; check("mod-leading-numeric", [$v, lastErr()], [2, want(E_WARNING, 'A non-numeric value encountered', __LINE__)]);
$v = $ops[3] * 2.0; check("mul-clean", lastErr(), null);
$v = "x" . $ops[4]; check("concat-array", [$v, lastErr()], ["xArray", want(E_WARNING, 'Array to string conversion', __LINE__)]);
$cv = 7.5; $cv %= 2; check("mod-assign-float", [$cv, lastErr()], [1, want(E_DEPRECATED, 'Implicit conversion from float 7.5 to int loses precision', __LINE__)]);
$v = "ab" | "  "; check("bitor-strings-clean", lastErr(), null);

// ----- call arguments: read by value, not by reference -----
function byVal($x) { return $x; }
function byRef(&$x) { return 0; }
class WarnArgs { public function val($x) { return $x; } public function ref(&$x) { return 0; } }
$v = byVal($argA); check("arg-by-value", lastErr(), want(E_WARNING, 'Undefined variable $argA', __LINE__));
$v = byRef($argB); check("arg-by-ref", [$argB, lastErr()], [null, null]);
$v = strlen($argC); check("arg-native", lastErr(), want(E_WARNING, 'Undefined variable $argC', __LINE__));
$v = byVal($a["missing"]); check("arg-by-value-key", lastErr(), want(E_WARNING, 'Undefined array key "missing"', __LINE__));
preg_match('/a/', "a", $argD); check("arg-native-by-ref", [count($argD), lastErr()], [1, null]);
$wa = new WarnArgs();
$v = $wa->val($argE); check("arg-method-by-value", lastErr(), want(E_WARNING, 'Undefined variable $argE', __LINE__));
$v = $wa->ref($argF); check("arg-method-by-ref", lastErr(), null);
$v = byVal(x: $argG); check("arg-named-by-value", lastErr(), want(E_WARNING, 'Undefined variable $argG', __LINE__));

// ----- closure bindings refused (zend_valid_closure_binding) -----
class WarnBind {
    public $v = 1;
    public function get() { return function () { return $this->v; }; }
    public function m() { return 2; }
}
class WarnOther {}
$wb = new WarnBind();
$v = $wb->get()->bindTo(null); check("unbind-this", [$v, lastErr()], [null, want(E_WARNING, 'Cannot unbind $this of closure using $this', __LINE__)]);
$sf = static function () { return 1; };
$v = $sf->bindTo($wb); check("bind-static", [$v, lastErr()], [null, want(E_WARNING, 'Cannot bind an instance to a static closure', __LINE__)]);
$v = Closure::bind($sf, $wb); check("bind-static-static-call", lastErr(), want(E_WARNING, 'Cannot bind an instance to a static closure', __LINE__));
$mc = Closure::fromCallable([$wb, 'm']);
$v = $mc->bindTo(new WarnOther()); check("bind-method-other", lastErr(), want(E_WARNING, 'Cannot bind method WarnBind::m() to object of class WarnOther', __LINE__));
$v = $mc->bindTo(null); check("unbind-method", lastErr(), want(E_WARNING, 'Cannot unbind $this of method', __LINE__));
$v = $mc->bindTo($wb, WarnOther::class); check("rebind-method-scope", lastErr(), want(E_WARNING, 'Cannot rebind scope of closure created from method', __LINE__));
$v = Closure::fromCallable('strlen')->bindTo(null, WarnBind::class); check("rebind-function-scope", lastErr(), want(E_WARNING, 'Cannot rebind scope of closure created from function', __LINE__));
$v = $wb->get()->bindTo(new WarnBind()); check("bind-ok", [$v(), lastErr()], [1, null]);

// ----- done -----
echo "warnings: " . $checks . " checks, " . $fails . " failures\n";
exit($fails);
