<?php
// A program that names nothing of lib/php-stdlib.php, so none of it is compiled
// unless the engine itself needs a piece. The class a failing argument count check
// raises is one such piece: it must exist although the program never spells its
// name, which is why the expected class name below is built from two halves.
// Exits with the failure count.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

function nlOne($a) { return $a; }
class NlK { function m($x) { return $x; } }

$got = [];
try { nlOne(); } catch (Error $e) { $got[] = get_class($e); $got[] = $e instanceof TypeError; }
try { (new NlK)->m(); } catch (Error $e) { $got[] = get_class($e); }
$want = "Argument" . "CountError";
check("nolib-arity-class", $got, [$want, true, $want]);
echo "nolib: " . $fails . " failures\n";
exit($fails);
