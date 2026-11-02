<?php
// php-stdlib.php -- PHP's array and string library and SPL, written in PHP.
//
// Both php grammars compile this file into a program that names one of its
// functions or classes (phpLibRef), so the interpreter half, llvm.Run and the native binary
// all run this one text and agree by construction. It is written on the engine
// natives - count, strlen, array_push, array_values, array_key_exists,
// array_key_last, array_pop, substr, strpos, strrpos, strcmp, ord, chr,
// strtolower, strtoupper (lib/php-strings.js, abnf/jsrtphpstr.go) and eleven
// intrinsics: __mec_fmtfloat for sprintf's float conversions ("r" is the
// shortest digits serialize_precision=-1 lays out), __mec_math for
// the math library's double operations, __mec_aptr for the internal array
// pointer, __mec_precision for the precision setting, __mec_oid for an
// object's handle, __mec_callable for the TypeError of an invalid callback
// argument, __mec_ifaces for the interfaces a class names, __mec_clsname for
// the declared name of the class a name denotes (false for none), __mec_clock
// for the wall clock in microseconds and __mec_pregc /
// __mec_pregx for preg_* (lib/php-preg.js). A native also gets a
// wrapper here of the same name, which is what a string callable or f(...)
// reaches: a direct call never does, since the call is lowered to the native
// first.
//
// A string is walked by BYTE offsets, but rebuilt from substr() RUNS between the
// ASCII bytes a function acts on: a one-byte piece of a multi-byte character is
// a stand-in (php-strings.js), which compares right but must not be copied out.
//
// Parameter names are the manual's, so named arguments bind. Errors are PHP 8's
// own ValueError / TypeError / ArgumentCountError texts. Deliberately absent:
// anything random (shuffle, array_rand), and the functions that reach the
// caller's variable scope (compact, extract).

const COUNT_NORMAL = 0;
const COUNT_RECURSIVE = 1;
const SORT_REGULAR = 0;
const SORT_NUMERIC = 1;
const SORT_STRING = 2;
const SORT_NATURAL = 6;
const SORT_FLAG_CASE = 8;
const ARRAY_FILTER_USE_BOTH = 1;
const ARRAY_FILTER_USE_KEY = 2;
const CASE_LOWER = 0;
const CASE_UPPER = 1;
const STR_PAD_LEFT = 0;
const STR_PAD_RIGHT = 1;
const STR_PAD_BOTH = 2;
const PATHINFO_DIRNAME = 1;
const PATHINFO_BASENAME = 2;
const PATHINFO_EXTENSION = 4;
const PATHINFO_FILENAME = 8;
const JSON_HEX_TAG = 1;
const JSON_HEX_AMP = 2;
const JSON_HEX_APOS = 4;
const JSON_HEX_QUOT = 8;
const JSON_FORCE_OBJECT = 16;
const JSON_NUMERIC_CHECK = 32;
const JSON_UNESCAPED_SLASHES = 64;
const JSON_PRETTY_PRINT = 128;
const JSON_UNESCAPED_UNICODE = 256;
const JSON_PARTIAL_OUTPUT_ON_ERROR = 512;
const JSON_PRESERVE_ZERO_FRACTION = 1024;
const JSON_UNESCAPED_LINE_TERMINATORS = 2048;
const JSON_OBJECT_AS_ARRAY = 1;
const JSON_BIGINT_AS_STRING = 2;
const JSON_INVALID_UTF8_IGNORE = 1048576;
const JSON_INVALID_UTF8_SUBSTITUTE = 2097152;
const JSON_THROW_ON_ERROR = 4194304;
const JSON_ERROR_NONE = 0;
const JSON_ERROR_DEPTH = 1;
const JSON_ERROR_STATE_MISMATCH = 2;
const JSON_ERROR_CTRL_CHAR = 3;
const JSON_ERROR_SYNTAX = 4;
const JSON_ERROR_UTF8 = 5;
const JSON_ERROR_RECURSION = 6;
const JSON_ERROR_INF_OR_NAN = 7;
const JSON_ERROR_UNSUPPORTED_TYPE = 8;
const JSON_ERROR_INVALID_PROPERTY_NAME = 9;
const JSON_ERROR_UTF16 = 10;
const JSON_ERROR_NON_BACKED_ENUM = 11;
const ENT_NOQUOTES = 0;
const ENT_COMPAT = 2;
const ENT_QUOTES = 3;
const ENT_IGNORE = 4;
const ENT_SUBSTITUTE = 8;
const ENT_HTML401 = 0;
const ENT_XML1 = 16;
const ENT_XHTML = 32;
const ENT_HTML5 = 48;
const ENT_DISALLOWED = 128;

class ArgumentCountError extends TypeError {}
class JsonException extends Exception {}
interface JsonSerializable { public function jsonSerialize(): mixed; }
// ErrorException (Zend/zend_exceptions.c): message, code and previous as
// Exception takes them; a filename replaces the file the exception was created
// in, and the line it was created on is replaced by $line, else by 0 when a
// filename was given. $severity defaults to E_ERROR.
class ErrorException extends Exception {
    protected int $severity = 1;
    public function __construct(string $message = "", int $code = 0, int $severity = 1,
                                ?string $filename = null, ?int $line = null, ?Throwable $previous = null) {
        parent::__construct($message, $code, $previous);
        $this->severity = $severity;
        if ($filename !== null) { $this->file = $filename; }
        if ($line !== null) { $this->line = $line; }
        elseif ($filename !== null) { $this->line = 0; }
    }
    final public function getSeverity(): int { return $this->severity; }
}

// ----- the natives, as callable values ---------------------------------------
function count($value, $mode = 0) { return count($value, $mode); }
function sizeof($value, $mode = 0) { return count($value, $mode); }
function strlen(string $string) { return strlen($string); }
function array_push(&$array, ...$values) {
    foreach ($values as $v) { $array[] = $v; }
    return count($array);
}
function array_values($array) { return array_values($array); }
function abs($num) { return abs($num); }
function intdiv(int $num1, int $num2) { return intdiv($num1, $num2); }
function array_key_exists($key, $array) { return __mec_akeyexists($key, $array); }
function key_exists($key, $array) { return __mec_akeyexists($key, $array); }
// PHP_FUNCTION(array_key_exists), reached by a call that is not its opcode: by a
// callable, with named arguments or as key_exists. A float key that is not an
// integer is deprecated (zend_dval_to_lval_safe) at the program's call.
function __mec_akeyexists($key, $array) {
    if (is_float($key) && is_array($array) && ($at = __mec_here()) !== null) { __mec_f2ichk($key, $at[0], $at[1]); }
    return array_key_exists($key, $array);
}
function array_key_last($array) { return array_key_last($array); }
function array_pop(&$array) { return array_pop($array); }
function substr(string $string, int $offset, ?int $length = null) { return substr($string, $offset, $length); }
function strpos(string $haystack, string $needle, int $offset = 0) { return strpos($haystack, $needle, $offset); }
function strrpos(string $haystack, string $needle, int $offset = 0) { return strrpos($haystack, $needle, $offset); }
function strcmp(string $string1, string $string2) { return strcmp($string1, $string2); }
function ord(string $character) { return ord($character); }
function chr(int $codepoint) { return chr($codepoint); }
function strtolower(string $string) { return strtolower($string); }
function strtoupper(string $string) { return strtoupper($string); }
function is_int($value) { return is_int($value); }
function is_integer($value) { return is_int($value); }
function is_long($value) { return is_int($value); }
function is_float($value) { return is_float($value); }
function is_double($value) { return is_float($value); }
function is_string($value) { return is_string($value); }
function is_bool($value) { return is_bool($value); }
function is_array($value) { return is_array($value); }
function is_object($value) { return is_object($value); }
function is_null($value) { return is_null($value); }
function is_numeric($value) { return is_numeric($value); }
function gettype($value) { return gettype($value); }
function get_debug_type($value) { return get_debug_type($value); }
// get_class() and get_parent_class() may be called without their argument.
function get_class(...$object /* at most 1 */) { return count($object) == 0 ? get_class() : get_class($object[0]); }
function get_parent_class(...$object_or_class /* at most 1 */) {
    return count($object_or_class) == 0 ? get_parent_class() : get_parent_class($object_or_class[0]);
}
function is_a($object_or_class, $class, $allow_string = false) { return is_a($object_or_class, $class, $allow_string); }
function is_subclass_of($object_or_class, $class, $allow_string = true) { return is_subclass_of($object_or_class, $class, $allow_string); }
function class_exists($class, $autoload = true) { return class_exists($class); }
function interface_exists($interface, $autoload = true) { return interface_exists($interface); }
function method_exists($object_or_class, $method) { return method_exists($object_or_class, $method); }
function property_exists($object_or_class, $property) { return property_exists($object_or_class, $property); }
function var_dump($value, ...$values) {
    var_dump($value);
    foreach ($values as $v) { var_dump($v); }
}
function function_exists($function) { return function_exists($function); }
function define($constant_name, $value, $case_insensitive = false) { return define($constant_name, $value); }
function defined($constant_name) { return defined($constant_name); }
function constant($name) { return constant($name); }
function error_reporting($error_level = null) { return error_reporting($error_level); }
function realpath($path) { return realpath($path); }

// zend_zval_value_name, the "X given" of PHP 8.3's argument TypeErrors: a bool is
// named by its value ("true given"), anything else as get_debug_type does.
function __mec_vname($v) { return $v === true ? "true" : ($v === false ? "false" : get_debug_type($v)); }
// The TypeError of an internal function's first parameter, array $array, which
// zpp judges before the callback that follows it. $obj: an object is accepted too
// (Z_PARAM_ARRAY_OR_OBJECT).
function __mec_arrarg($v, $fname, $obj) {
    if (!is_array($v) && !($obj && is_object($v))) {
        throw new TypeError($fname . "(): Argument #1 (\$array) must be of type array, " . __mec_vname($v) . " given");
    }
}
// Argument $n of library function $fn, whose parameter $name declares $type
// (string, int, float or bool, "?" in front if nullable), as zpp reads one the
// prologue's exact-type test refused (zend_parse_arg_{str,long,double,bool}_slow):
// [the value, the text of its Deprecated or ""]. A strict call ($strict) takes
// only an int for a float; a coercive one reads null as the type's zero, a
// scalar or a Stringable object as the type, and a float that is not an integer
// as an int, the first and the last deprecated. Anything else is the TypeError.
function __mec_zppr($v, $type, $fn, $n, $name, $strict) {
    $t = $type[0] === "?" ? substr($type, 1) : $type;
    if ($v === null && !$strict) {
        return [$t === "string" ? "" : ($t === "int" ? 0 : ($t === "float" ? 0.0 : false)),
                $fn . "(): Passing null to parameter #" . $n . " (\$" . $name . ") of type " . $type . " is deprecated"];
    }
    if ($t === "float" && is_int($v)) { return [(float) $v, ""]; }
    if (!$strict) {
        if ($t === "string" && (is_int($v) || is_float($v) || is_bool($v) || (is_object($v) && method_exists($v, "__toString")))) { return [(string) $v, ""]; }
        if ($t === "bool" && (is_int($v) || is_float($v) || is_string($v))) { return [(bool) $v, ""]; }
        if ($t === "float" && (is_bool($v) || (is_string($v) && is_numeric($v)))) { return [(float) $v, ""]; }
        if ($t === "int" && is_bool($v)) { return [(int) $v, ""]; }
        if ($t === "int" && (is_float($v) || (is_string($v) && is_numeric($v)))) {
            $d = is_float($v) ? $v : $v + 0;
            if (is_int($d)) { return [$d, ""]; }
            if ($d == $d && $d >= -9.2233720368547758E18 && $d < 9.2233720368547758E18) {
                if ((float) (int) $d === $d) { return [(int) $d, ""]; }
                return [(int) $d, is_float($v) ? "Implicit conversion from float " . __mec_serfloat($v, "E") . " to int loses precision"
                    : "Implicit conversion from float-string \"" . $v . "\" to int loses precision"];
            }
        }
    }
    throw new TypeError($fn . "(): Argument #" . $n . " (\$" . $name . ") must be of type " . $type . ", " . __mec_vname($v) . " given");
}
function __mec_zpp($v, $type, $fn, $n, $name, $strict) { return __mec_zppr($v, $type, $fn, $n, $name, $strict)[0]; }
// __mec_zpp with display_errors=1: the Deprecated is displayed at the call's
// $file and $line.
function __mec_zppd($v, $type, $fn, $n, $name, $strict, $file = null, $line = 0) {
    $r = __mec_zppr($v, $type, $fn, $n, $name, $strict);
    if ($r[1] !== "") { __mec_diag(8192, $r[1], $file, $line); }
    return $r[0];
}

// ----- comparison helpers ----------------------------------------------------
// The sort flags' orderings: SORT_REGULAR is PHP's standard comparison (<=>),
// SORT_NUMERIC compares the numeric values, SORT_STRING the bytes, and
// SORT_FLAG_CASE makes SORT_STRING case-insensitive.
function __mec_flagcmp($a, $b, $flags) {
    $f = $flags & ~SORT_FLAG_CASE;
    if ($f == SORT_NUMERIC) { return ((float)$a) <=> ((float)$b); }
    if ($f == SORT_STRING) {
        if ($flags & SORT_FLAG_CASE) { return strcasecmp((string)$a, (string)$b); }
        return strcmp((string)$a, (string)$b);
    }
    if ($f == SORT_NATURAL) { return __mec_natcmp($a, $b, ($flags & SORT_FLAG_CASE) != 0); }
    return $a <=> $b;
}
// A user comparison callback's answer as an ordering. A bool is PHP 8's
// deprecated form: false is retried the other way round.
function __mec_usercmp($callback, $a, $b) {
    $r = $callback($a, $b);
    if ($r === false) { return $callback($b, $a) ? -1 : 0; }
    if ($r === true) { return 1; }
    return (int)$r;
}
// A STABLE merge sort of [key, value] pairs (PHP's sort is stable since 8.0).
// $by: 0 compares values, 1 compares keys; $cmp is a comparison callback or null
// for the flag ordering; $dir is 1 or -1.
function __mec_sortpairs($pairs, $by, $cmp, $flags, $dir) {
    $n = count($pairs);
    if ($n < 2) { return $pairs; }
    $mid = intdiv($n, 2);
    $left = __mec_sortpairs(array_slice($pairs, 0, $mid), $by, $cmp, $flags, $dir);
    $right = __mec_sortpairs(array_slice($pairs, $mid), $by, $cmp, $flags, $dir);
    $out = [];
    $i = 0;
    $j = 0;
    $nl = count($left);
    $nr = count($right);
    while ($i < $nl && $j < $nr) {
        $x = $left[$i][$by];
        $y = $right[$j][$by];
        $c = $cmp === null ? __mec_flagcmp($x, $y, $flags) : __mec_usercmp($cmp, $x, $y);
        if ($c * $dir <= 0) { $out[] = $left[$i]; $i++; } else { $out[] = $right[$j]; $j++; }
    }
    while ($i < $nl) { $out[] = $left[$i]; $i++; }
    while ($j < $nr) { $out[] = $right[$j]; $j++; }
    return $out;
}
function __mec_pairs($array) {
    $pairs = [];
    foreach ($array as $k => $v) { $pairs[] = [$k, $v]; }
    return $pairs;
}
function __mec_sort(&$array, $by, $cmp, $flags, $dir, $keep) {
    $sorted = __mec_sortpairs(__mec_pairs($array), $by, $cmp, $flags, $dir);
    $out = [];
    foreach ($sorted as $p) {
        if ($keep) { $out[$p[0]] = $p[1]; } else { $out[] = $p[1]; }
    }
    $array = $out;
    return true;
}

// ----- sorting ---------------------------------------------------------------
function sort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, 1, false); }
function rsort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, -1, false); }
function usort(&$array, $callback) {
    __mec_arrarg($array, 'usort', false);
    __mec_callable($callback, 'usort(): Argument #2 ($callback) must be a valid callback');
    return __mec_sort($array, 1, $callback, 0, 1, false);
}
function asort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, 1, true); }
function arsort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, -1, true); }
function uasort(&$array, $callback) {
    __mec_arrarg($array, 'uasort', false);
    __mec_callable($callback, 'uasort(): Argument #2 ($callback) must be a valid callback');
    return __mec_sort($array, 1, $callback, 0, 1, true);
}
function natsort(&$array) { return __mec_sort($array, 1, null, SORT_NATURAL, 1, true); }
function natcasesort(&$array) { return __mec_sort($array, 1, null, SORT_NATURAL | SORT_FLAG_CASE, 1, true); }
function ksort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 0, null, $flags, 1, true); }
function krsort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 0, null, $flags, -1, true); }
function uksort(&$array, $callback) {
    __mec_arrarg($array, 'uksort', false);
    __mec_callable($callback, 'uksort(): Argument #2 ($callback) must be a valid callback');
    return __mec_sort($array, 0, $callback, 0, 1, true);
}

// ----- keys, values, searching -----------------------------------------------
function array_keys($array, ...$filter /* at most 3 */) {
    $out = [];
    $search = count($filter) > 0;
    $strict = count($filter) > 1 && $filter[1];
    foreach ($array as $k => $v) {
        if (!$search || ($strict ? $v === $filter[0] : $v == $filter[0])) { $out[] = $k; }
    }
    return $out;
}
function array_key_first($array) {
    foreach ($array as $k => $v) { return $k; }
    return null;
}
function in_array($needle, $haystack, $strict = false) {
    foreach ($haystack as $v) {
        if ($strict ? $v === $needle : $v == $needle) { return true; }
    }
    return false;
}
function array_search($needle, $haystack, $strict = false) {
    foreach ($haystack as $k => $v) {
        if ($strict ? $v === $needle : $v == $needle) { return $k; }
    }
    return false;
}
function array_is_list($array) {
    $i = 0;
    foreach ($array as $k => $v) {
        if ($k !== $i) { return false; }
        $i++;
    }
    return true;
}
function array_flip($array) {
    $out = [];
    foreach ($array as $k => $v) {
        if (is_int($v) || is_string($v)) { $out[$v] = $k; }
    }
    return $out;
}
function array_count_values($array) {
    $out = [];
    foreach ($array as $v) {
        if (is_int($v) || is_string($v)) { $out[$v] = ($out[$v] ?? 0) + 1; }
    }
    return $out;
}
function array_find($array, $callback) {
    foreach ($array as $k => $v) { if ($callback($v, $k)) { return $v; } }
    return null;
}
function array_find_key($array, $callback) {
    foreach ($array as $k => $v) { if ($callback($v, $k)) { return $k; } }
    return null;
}
function array_any($array, $callback) {
    foreach ($array as $k => $v) { if ($callback($v, $k)) { return true; } }
    return false;
}
function array_all($array, $callback) {
    foreach ($array as $k => $v) { if (!$callback($v, $k)) { return false; } }
    return true;
}
// The internal pointer. current() and key() take the array by value, and a copy
// carries the pointer along, so they read the caller's position.
function current($array) { return __mec_aptr($array, "current"); }
function pos($array) { return __mec_aptr($array, "current"); }
function key($array) { return __mec_aptr($array, "key"); }
function next(&$array) { return __mec_aptr($array, "next"); }
function prev(&$array) { return __mec_aptr($array, "prev"); }
function reset(&$array) { return __mec_aptr($array, "reset"); }
function end(&$array) { return __mec_aptr($array, "end"); }

// ----- building and reshaping ------------------------------------------------
function array_fill($start_index, $count, $value) {
    if ($count < 0) { throw new ValueError("array_fill(): Argument #2 (\$count) must be greater than or equal to 0"); }
    $out = [];
    for ($i = 0; $i < $count; $i++) { $out[$start_index + $i] = $value; }
    return $out;
}
function array_fill_keys($keys, $value) {
    $out = [];
    foreach ($keys as $k) { $out[$k] = $value; }
    return $out;
}
function array_combine($keys, $values) {
    if (count($keys) != count($values)) {
        throw new ValueError("array_combine(): Argument #1 (\$keys) and argument #2 (\$values) must have the same number of elements");
    }
    $out = [];
    $vs = array_values($values);
    $i = 0;
    foreach ($keys as $k) { $out[$k] = $vs[$i]; $i++; }
    return $out;
}
function range($start, $end, $step = 1) {
    if ($step == 0) { throw new ValueError("range(): Argument #3 (\$step) cannot be 0"); }
    $step = abs($step);
    $out = [];
    if (is_string($start) && is_string($end) && strlen($start) >= 1 && strlen($end) >= 1
        && !is_numeric($start) && !is_numeric($end)) {
        $a = ord($start);
        $b = ord($end);
        $st = (int)$step;
        if ($a <= $b) { for ($c = $a; $c <= $b; $c += $st) { $out[] = chr($c); } }
        else { for ($c = $a; $c >= $b; $c -= $st) { $out[] = chr($c); } }
        return $out;
    }
    $isF = is_float($start) || is_float($end) || is_float($step)
        || (is_string($start) && is_float($start + 0)) || (is_string($end) && is_float($end + 0));
    $a = $start + 0;
    $b = $end + 0;
    if ($isF) {
        $a = (float)$a;
        $b = (float)$b;
        $step = (float)$step;
        $n = __mec_steps(abs($b - $a), $step);
        for ($i = 0; $i <= $n; $i++) { $out[] = $a <= $b ? $a + $i * $step : $a - $i * $step; }
        return $out;
    }
    if ($a <= $b) { for ($x = $a; $x <= $b; $x += $step) { $out[] = $x; } }
    else { for ($x = $a; $x >= $b; $x -= $step) { $out[] = $x; } }
    return $out;
}
// The whole number of steps that fit in a float span.
function __mec_steps($span, $step) {
    $n = (int)($span / $step);
    if ($n * $step > $span) { $n--; }
    return $n;
}
function array_slice($array, $offset, $length = null, $preserve_keys = false) {
    $n = count($array);
    if ($offset > $n) { return []; }
    if ($offset < 0) { $offset = $n + $offset < 0 ? 0 : $n + $offset; }
    if ($length === null) { $length = $n - $offset; }
    elseif ($length < 0) { $length = $n - $offset + $length; }
    elseif ($offset + $length > $n) { $length = $n - $offset; }
    $out = [];
    if ($length <= 0) { return $out; }
    $i = 0;
    foreach ($array as $k => $v) {
        if ($i >= $offset + $length) { break; }
        if ($i >= $offset) {
            if (is_string($k) || $preserve_keys) { $out[$k] = $v; } else { $out[] = $v; }
        }
        $i++;
    }
    return $out;
}
function array_splice(&$array, $offset, $length = null, $replacement = []) {
    $n = count($array);
    if ($offset > $n) { $offset = $n; }
    if ($offset < 0) { $offset = $n + $offset < 0 ? 0 : $n + $offset; }
    if ($length === null) { $length = $n - $offset; }
    elseif ($length < 0) { $length = $n - $offset + $length < 0 ? 0 : $n - $offset + $length; }
    elseif ($offset + $length > $n) { $length = $n - $offset; }
    if (!is_array($replacement)) { $replacement = [$replacement]; }
    $out = [];
    $removed = [];
    $i = 0;
    foreach ($array as $k => $v) {
        if ($i == $offset) { foreach ($replacement as $r) { $out[] = $r; } }
        if ($i >= $offset && $i < $offset + $length) { $removed[] = $v; }
        elseif (is_string($k)) { $out[$k] = $v; }
        else { $out[] = $v; }
        $i++;
    }
    if ($offset >= $n) { foreach ($replacement as $r) { $out[] = $r; } }
    $array = $out;
    return $removed;
}
function array_shift(&$array) {
    if (count($array) == 0) { return null; }
    $first = true;
    $value = null;
    $out = [];
    foreach ($array as $k => $v) {
        if ($first) { $value = $v; $first = false; continue; }
        if (is_string($k)) { $out[$k] = $v; } else { $out[] = $v; }
    }
    $array = $out;
    return $value;
}
function array_unshift(&$array, ...$values) {
    $out = [];
    foreach ($values as $v) { $out[] = $v; }
    foreach ($array as $k => $v) {
        if (is_string($k)) { $out[$k] = $v; } else { $out[] = $v; }
    }
    $array = $out;
    return count($array);
}
function array_reverse($array, $preserve_keys = false) {
    $pairs = __mec_pairs($array);
    $out = [];
    for ($i = count($pairs) - 1; $i >= 0; $i--) {
        $k = $pairs[$i][0];
        if (is_string($k) || $preserve_keys) { $out[$k] = $pairs[$i][1]; } else { $out[] = $pairs[$i][1]; }
    }
    return $out;
}
function array_pad($array, $length, $value) {
    $n = count($array);
    $pad = abs($length) - $n;
    if ($pad <= 0) { return $array; }
    $out = [];
    if ($length < 0) { for ($i = 0; $i < $pad; $i++) { $out[] = $value; } }
    foreach ($array as $k => $v) {
        if (is_string($k)) { $out[$k] = $v; } else { $out[] = $v; }
    }
    if ($length > 0) { for ($i = 0; $i < $pad; $i++) { $out[] = $value; } }
    return $out;
}
function array_chunk($array, $length, $preserve_keys = false) {
    if ($length < 1) { throw new ValueError("array_chunk(): Argument #2 (\$length) must be greater than 0"); }
    $out = [];
    $cur = [];
    foreach ($array as $k => $v) {
        if ($preserve_keys) { $cur[$k] = $v; } else { $cur[] = $v; }
        if (count($cur) == $length) { $out[] = $cur; $cur = []; }
    }
    if (count($cur) > 0) { $out[] = $cur; }
    return $out;
}
function array_column($array, $column_key, $index_key = null) {
    $out = [];
    foreach ($array as $row) {
        if (!is_array($row)) { continue; }
        if ($column_key === null) { $v = $row; }
        elseif (array_key_exists($column_key, $row)) { $v = $row[$column_key]; }
        else { continue; }
        if ($index_key !== null && array_key_exists($index_key, $row)) { $out[$row[$index_key]] = $v; }
        else { $out[] = $v; }
    }
    return $out;
}
function array_change_key_case($array, $case = CASE_LOWER) {
    $out = [];
    foreach ($array as $k => $v) {
        if (is_string($k)) { $k = $case == CASE_UPPER ? strtoupper($k) : strtolower($k); }
        $out[$k] = $v;
    }
    return $out;
}
// The first of each run of equal values stays, under its key: equal as strings
// by default, by == for SORT_REGULAR, as numbers for SORT_NUMERIC.
function array_unique($array, $flags = SORT_STRING) {
    $out = [];
    $seen = [];
    foreach ($array as $k => $v) {
        $dup = false;
        foreach ($seen as $s) {
            if ($flags == SORT_REGULAR) { $eq = $s == $v; }
            elseif ($flags == SORT_NUMERIC) { $eq = (float)$s == (float)$v; }
            else { $eq = (string)$s === (string)$v; }
            if ($eq) { $dup = true; break; }
        }
        if (!$dup) { $seen[] = $v; $out[$k] = $v; }
    }
    return $out;
}

// ----- merging and replacing -------------------------------------------------
function array_merge(...$arrays) {
    $out = [];
    foreach ($arrays as $a) {
        foreach ($a as $k => $v) {
            if (is_string($k)) { $out[$k] = $v; } else { $out[] = $v; }
        }
    }
    return $out;
}
function array_merge_recursive(...$arrays) {
    $out = [];
    foreach ($arrays as $a) {
        foreach ($a as $k => $v) {
            if (is_string($k) && array_key_exists($k, $out)) {
                $cur = is_array($out[$k]) ? $out[$k] : [$out[$k]];
                if (is_array($v)) { $out[$k] = array_merge_recursive($cur, $v); }
                else { $cur[] = $v; $out[$k] = $cur; }
            } elseif (is_string($k)) {
                $out[$k] = $v;
            } else {
                $out[] = $v;
            }
        }
    }
    return $out;
}
function array_replace($array, ...$replacements) {
    foreach ($replacements as $r) {
        foreach ($r as $k => $v) { $array[$k] = $v; }
    }
    return $array;
}
function array_replace_recursive($array, ...$replacements) {
    foreach ($replacements as $r) {
        foreach ($r as $k => $v) {
            if (is_array($v) && array_key_exists($k, $array) && is_array($array[$k])) {
                $array[$k] = array_replace_recursive($array[$k], $v);
            } else {
                $array[$k] = $v;
            }
        }
    }
    return $array;
}

// ----- set operations --------------------------------------------------------
// $mode: 0 compares values as strings, 1 compares keys, 2 both. $want is true
// for an intersection (kept when found in EVERY other array) and false for a
// difference (kept when found in NONE). $vcmp / $kcmp are user callbacks or null.
function __mec_setop($array, $others, $mode, $want, $vcmp, $kcmp) {
    $out = [];
    foreach ($array as $k => $v) {
        $keep = true;
        foreach ($others as $o) {
            $found = false;
            foreach ($o as $ok => $ov) {
                $keq = $kcmp === null ? $ok === $k : __mec_usercmp($kcmp, $k, $ok) == 0;
                $veq = $vcmp === null ? (string)$ov === (string)$v : __mec_usercmp($vcmp, $v, $ov) == 0;
                if (($mode == 0 && $veq) || ($mode == 1 && $keq) || ($mode == 2 && $keq && $veq)) {
                    $found = true;
                    break;
                }
            }
            if ($found != $want) { $keep = false; break; }
        }
        if ($keep) { $out[$k] = $v; }
    }
    return $out;
}
// The $n trailing callbacks of the u* variants, split off the array list, in
// argument order. zpp's "+f" / "+ff" judges them before any array, and their
// positions have no parameter name.
function __mec_takecb($fname, $array, &$arrays, $n) {
    if (count($arrays) < $n) {
        throw new ArgumentCountError($fname . "() expects at least " . ($n + 1) . " arguments, " . (count($arrays) + 1) . " given");
    }
    $cbs = array_slice($arrays, count($arrays) - $n);
    $arrays = array_slice($arrays, 0, count($arrays) - $n);
    foreach ($cbs as $i => $cb) {
        __mec_callable($cb, $fname . '(): Argument #' . (count($arrays) + 2 + $i) . ' must be a valid callback');
    }
    __mec_arrarg($array, $fname, false);
    foreach ($arrays as $i => $a) {
        if (!is_array($a)) { throw new TypeError($fname . "(): Argument #" . ($i + 2) . " must be of type array, " . __mec_vname($a) . " given"); }
    }
    return $cbs;
}
function array_diff($array, ...$arrays) { return __mec_setop($array, $arrays, 0, false, null, null); }
function array_diff_key($array, ...$arrays) { return __mec_setop($array, $arrays, 1, false, null, null); }
function array_diff_assoc($array, ...$arrays) { return __mec_setop($array, $arrays, 2, false, null, null); }
function array_intersect($array, ...$arrays) { return __mec_setop($array, $arrays, 0, true, null, null); }
function array_intersect_key($array, ...$arrays) { return __mec_setop($array, $arrays, 1, true, null, null); }
function array_intersect_assoc($array, ...$arrays) { return __mec_setop($array, $arrays, 2, true, null, null); }
function array_udiff($array, ...$rest) { [$cb] = __mec_takecb('array_udiff', $array, $rest, 1); return __mec_setop($array, $rest, 0, false, $cb, null); }
function array_uintersect($array, ...$rest) { [$cb] = __mec_takecb('array_uintersect', $array, $rest, 1); return __mec_setop($array, $rest, 0, true, $cb, null); }
function array_diff_ukey($array, ...$rest) { [$cb] = __mec_takecb('array_diff_ukey', $array, $rest, 1); return __mec_setop($array, $rest, 1, false, null, $cb); }
function array_intersect_ukey($array, ...$rest) { [$cb] = __mec_takecb('array_intersect_ukey', $array, $rest, 1); return __mec_setop($array, $rest, 1, true, null, $cb); }
function array_udiff_assoc($array, ...$rest) { [$cb] = __mec_takecb('array_udiff_assoc', $array, $rest, 1); return __mec_setop($array, $rest, 2, false, $cb, null); }
function array_uintersect_assoc($array, ...$rest) { [$cb] = __mec_takecb('array_uintersect_assoc', $array, $rest, 1); return __mec_setop($array, $rest, 2, true, $cb, null); }
function array_diff_uassoc($array, ...$rest) { [$cb] = __mec_takecb('array_diff_uassoc', $array, $rest, 1); return __mec_setop($array, $rest, 2, false, null, $cb); }
function array_intersect_uassoc($array, ...$rest) { [$cb] = __mec_takecb('array_intersect_uassoc', $array, $rest, 1); return __mec_setop($array, $rest, 2, true, null, $cb); }
function array_udiff_uassoc($array, ...$rest) { [$cb, $kcb] = __mec_takecb('array_udiff_uassoc', $array, $rest, 2); return __mec_setop($array, $rest, 2, false, $cb, $kcb); }
function array_uintersect_uassoc($array, ...$rest) { [$cb, $kcb] = __mec_takecb('array_uintersect_uassoc', $array, $rest, 2); return __mec_setop($array, $rest, 2, true, $cb, $kcb); }

// ----- callbacks -------------------------------------------------------------
// A string key of call_user_func_array's list is a named argument, as in a spread.
// Its list is spread as a value, not as the variable: an element that is no
// reference reaches a by-reference parameter as a temporary one, and the list
// keeps it (zend_call_function).
function call_user_func($callback, ...$args) { return $callback(...$args); }
function call_user_func_array($callback, $args) { return $callback(...($args)); }
// The class a callable's class part $name denotes, checked in class scope $scope
// with called scope $static (names, or null): [its name, the "self", "parent" or
// "static" that names it relatively], or null for none
// (zend_is_callable_check_class).
function __mec_cbclass($name, $scope, $static) {
    $lc = strtolower($name);
    if ($lc === "self") { return $scope === null ? null : [$scope, "self"]; }
    if ($lc === "parent") {
        $p = $scope === null ? false : get_parent_class($scope);
        return $p === false ? null : [$p, "parent"];
    }
    if ($lc === "static") { return $static === null ? null : [$static, "static"]; }
    $c = __mec_clsname($name);
    return $c === false ? null : [$c, null];
}
// The byte offset of the "::" splitting a callable's method part "K::m" (the
// last one), or false (zend_is_callable_check_func).
function __mec_cbsep($m) {
    $at = strrpos($m, ":");
    return $at !== false && $at > 0 && $m[$at - 1] === ":" ? $at - 1 : false;
}
// The deprecations checking callable $cb in a frame of class scope $scope and
// called scope $static raises (zend_is_callable_at_frame): "self::m" or
// ["self", "m"] (or parent, static) whose class it reaches, then
// [$object_or_class, "K::m"] whose K the array's own class is an instance of.
function __mec_cbdepr($cb, $scope, $static, $file = null, $line = 0) {
    if (is_string($cb)) {
        $org = null;
        $m = $cb;
    } elseif (is_array($cb) && count($cb) == 2 && array_key_exists(0, $cb) && array_key_exists(1, $cb) && is_string($cb[1])) {
        $m = $cb[1];
        if (is_object($cb[0])) {
            $org = get_class($cb[0]);
        } elseif (is_string($cb[0])) {
            $k = __mec_cbclass($cb[0], $scope, $static);
            if ($k === null) { return; }
            if ($k[1] !== null) { __mec_diag(8192, "Use of \"" . $k[1] . "\" in callables is deprecated", $file, $line); }
            $org = $k[0];
        } else { return; }
    } else { return; }
    $at = __mec_cbsep($m);
    if ($at === false || $at == 0) { return; }
    $k = __mec_cbclass(substr($m, 0, $at), $org ?? $scope, $static);
    if ($k === null) { return; }
    if ($org === null) {
        if ($k[1] !== null) { __mec_diag(8192, "Use of \"" . $k[1] . "\" in callables is deprecated", $file, $line); }
    } elseif (is_a($org, $k[0], true)) {
        __mec_diag(8192, "Callables of the form [\"" . $org . "\", \"" . $m . "\"] are deprecated", $file, $line);
    }
}
// The TypeError of an invalid callback: $where, then why.
function __mec_cbfail($where, $why) {
    throw new TypeError($where . (substr($where, -1) === ":" ? " " : ", ") . $why);
}
// The callable [$object_or_class, "K::m"] as a closure: K is checked against the
// array's own class (an instance of it) and the method is K's, called without
// virtual dispatch on the object, or for a class name on the frame's $this ($this_)
// when that is an instance (zend_is_callable_check_func); a non-static method with
// neither is invalid (__mec_cbns). An invalid one is the TypeError $where begins;
// any other callable is answered as it is.
function __mec_cbcompound($cb, $where, $this_, $scope, $static) {
    if (!is_array($cb) || count($cb) != 2 || !array_key_exists(0, $cb) || !array_key_exists(1, $cb) || !is_string($cb[1])) { return $cb; }
    $m = $cb[1];
    $at = __mec_cbsep($m);
    if ($at === false) { return $cb; }
    if (is_object($cb[0])) {
        $org = get_class($cb[0]);
        $obj = $cb[0];
    } elseif (is_string($cb[0])) {
        $k = __mec_cbclass($cb[0], $scope, $static);
        if ($k === null) { return $cb; }
        $org = $k[0];
        $obj = is_object($this_) && $this_ instanceof $org ? $this_ : null;
    } else { return $cb; }
    if ($at == 0) { __mec_cbfail($where, "invalid function name"); }
    $cname = substr($m, 0, $at);
    $k = __mec_cbclass($cname, $org, $static);
    if ($k === null) {
        $lc = strtolower($cname);
        __mec_cbfail($where, $lc === "parent" ? "cannot access \"parent\" when current class scope has no parent"
            : ($lc === "static" ? "cannot access \"static\" when no class scope is active" : "class \"" . $cname . "\" not found"));
    }
    $cls = $k[0];
    if (!is_a($org, $cls, true)) { __mec_cbfail($where, "class " . $org . " is not a subclass of " . $cls); }
    $mn = substr($m, $at + 2);
    if (!method_exists($cls, $mn) && !method_exists($cls, "__call") && !method_exists($cls, "__callStatic")) {
        __mec_cbfail($where, "class " . $cls . " does not have a method \"" . $mn . "\"");
    }
    if ($obj === null) { __mec_cbns($where, $cls, $mn); }
    return Closure::bind(function (...$args) use ($mn) { return self::$mn(...$args); }, $obj, $cls);
}
function array_map($callback, $array, ...$arrays) {
    if ($callback !== null) { __mec_callable($callback, 'array_map(): Argument #1 ($callback) must be a valid callback or null'); }
    if (!is_array($array)) {
        throw new TypeError('array_map(): Argument #2 ($array) must be of type array, ' . __mec_vname($array) . ' given');
    }
    foreach ($arrays as $i => $a) {
        if (!is_array($a)) { throw new TypeError('array_map(): Argument #' . ($i + 3) . ' must be of type array, ' . __mec_vname($a) . ' given'); }
    }
    $out = [];
    if (count($arrays) == 0) {
        if ($callback === null) { return $array; }
        foreach ($array as $k => $v) { $out[$k] = $callback($v); }
        return $out;
    }
    $lists = [array_values($array)];
    $n = count($array);
    foreach ($arrays as $a) {
        $lists[] = array_values($a);
        if (count($a) > $n) { $n = count($a); }
    }
    for ($i = 0; $i < $n; $i++) {
        $args = [];
        foreach ($lists as $l) { $args[] = $l[$i] ?? null; }
        $out[] = $callback === null ? $args : $callback(...$args);
    }
    return $out;
}
function array_filter($array, $callback = null, $mode = 0) {
    __mec_arrarg($array, 'array_filter', false);
    if ($callback !== null) { __mec_callable($callback, 'array_filter(): Argument #2 ($callback) must be a valid callback or null'); }
    $out = [];
    foreach ($array as $k => $v) {
        if ($callback === null) { $ok = (bool)$v; }
        elseif ($mode == ARRAY_FILTER_USE_KEY) { $ok = $callback($k); }
        elseif ($mode == ARRAY_FILTER_USE_BOTH) { $ok = $callback($v, $k); }
        else { $ok = $callback($v); }
        if ($ok) { $out[$k] = $v; }
    }
    return $out;
}
function array_reduce($array, $callback, $initial = null) {
    __mec_arrarg($array, 'array_reduce', false);
    __mec_callable($callback, 'array_reduce(): Argument #2 ($callback) must be a valid callback');
    $acc = $initial;
    foreach ($array as $v) { $acc = $callback($acc, $v); }
    return $acc;
}
// The callback's by-reference parameter writes back only where a call of a
// callable VALUE passes a reference: the interpreter half does, the compiler
// half's dynamic call passes the value.
function array_walk(&$array, $callback, ...$arg /* at most 3 */) {
    __mec_arrarg($array, 'array_walk', true);
    __mec_callable($callback, 'array_walk(): Argument #2 ($callback) must be a valid callback');
    foreach ($array as $k => &$v) {
        if (count($arg) > 0) { $callback($v, $k, $arg[0]); } else { $callback($v, $k); }
    }
    unset($v);
    return true;
}
function array_walk_recursive(&$array, $callback, ...$arg /* at most 3 */) {
    __mec_arrarg($array, 'array_walk_recursive', true);
    __mec_callable($callback, 'array_walk_recursive(): Argument #2 ($callback) must be a valid callback');
    __mec_walkrec($array, $callback, $arg);
    return true;
}
function __mec_walkrec(&$array, $callback, $arg) {
    foreach ($array as $k => &$v) {
        if (is_array($v)) { __mec_walkrec($v, $callback, $arg); }
        elseif (count($arg) > 0) { $callback($v, $k, $arg[0]); }
        else { $callback($v, $k); }
    }
    unset($v);
}
function array_sum($array) {
    $sum = 0;
    foreach ($array as $v) { if (!is_array($v)) { $sum = $sum + $v; } }
    return $sum;
}
function array_product($array) {
    $p = 1;
    foreach ($array as $v) { if (!is_array($v)) { $p = $p * $v; } }
    return $p;
}
function max($value, ...$values) {
    if (count($values) == 0) {
        if (!is_array($value)) {
            throw new TypeError("max(): Argument #1 (\$value) must be of type array, " . __mec_vname($value) . " given");
        }
        if (count($value) == 0) { throw new ValueError("max(): Argument #1 (\$value) must contain at least one element"); }
        $values = $value;
    } else {
        array_unshift($values, $value);
    }
    $best = null;
    $first = true;
    foreach ($values as $v) {
        if ($first || $v > $best) { $best = $v; }
        $first = false;
    }
    return $best;
}
function min($value, ...$values) {
    if (count($values) == 0) {
        if (!is_array($value)) {
            throw new TypeError("min(): Argument #1 (\$value) must be of type array, " . __mec_vname($value) . " given");
        }
        if (count($value) == 0) { throw new ValueError("min(): Argument #1 (\$value) must contain at least one element"); }
        $values = $value;
    } else {
        array_unshift($values, $value);
    }
    $best = null;
    $first = true;
    foreach ($values as $v) {
        if ($first || $v < $best) { $best = $v; }
        $first = false;
    }
    return $best;
}

// ----- strings: joining and splitting ----------------------------------------
function implode($separator, $array = null) {
    if ($array === null) {
        if (!is_array($separator)) {
            throw new TypeError("implode(): Argument #2 (\$array) must be of type ?array, " . __mec_vname($array) . " given");
        }
        $array = $separator;
        $separator = "";
    } elseif (!is_array($array)) {
        throw new TypeError("implode(): Argument #2 (\$array) must be of type ?array, " . __mec_vname($array) . " given");
    }
    $out = "";
    $first = true;
    foreach ($array as $v) {
        if (!$first) { $out .= $separator; }
        $out .= is_array($v) ? "Array" : (string)$v;
        $first = false;
    }
    return $out;
}
function join($separator, $array = null) { return implode($separator, $array); }
function explode(string $separator, string $string, int $limit = PHP_INT_MAX) {
    if ($separator === "") { throw new ValueError("explode(): Argument #1 (\$separator) cannot be empty"); }
    $parts = [];
    $at = 0;
    $n = strlen($separator);
    while (($p = strpos($string, $separator, $at)) !== false) {
        $parts[] = substr($string, $at, $p - $at);
        $at = $p + $n;
    }
    $parts[] = substr($string, $at);
    if ($limit > 0) {
        if (count($parts) <= $limit) { return $parts; }
        $out = array_slice($parts, 0, $limit - 1);
        $out[] = implode($separator, array_slice($parts, $limit - 1));
        return $out;
    }
    if ($limit == 0) { return [implode($separator, $parts)]; }
    return array_slice($parts, 0, $limit);
}
function str_split(string $string, int $length = 1) {
    if ($length < 1) { throw new ValueError("str_split(): Argument #2 (\$length) must be greater than 0"); }
    $out = [];
    $n = strlen($string);
    for ($i = 0; $i < $n; $i += $length) { $out[] = substr($string, $i, $length); }
    return $out;
}
function str_repeat(string $string, int $times) {
    if ($times < 0) { throw new ValueError("str_repeat(): Argument #2 (\$times) must be greater than or equal to 0"); }
    $out = "";
    $piece = $string;
    while ($times > 0) {
        if ($times % 2 == 1) { $out .= $piece; }
        $piece .= $piece;
        $times = intdiv($times, 2);
    }
    return $out;
}
function str_pad(string $string, int $length, string $pad_string = " ", int $pad_type = STR_PAD_RIGHT) {
    if ($pad_string === "") { throw new ValueError("str_pad(): Argument #3 (\$pad_string) must be a non-empty string"); }
    if ($pad_type != STR_PAD_LEFT && $pad_type != STR_PAD_RIGHT && $pad_type != STR_PAD_BOTH) {
        throw new ValueError("str_pad(): Argument #4 (\$pad_type) must be STR_PAD_LEFT, STR_PAD_RIGHT, or STR_PAD_BOTH");
    }
    $n = $length - strlen($string);
    if ($n <= 0) { return $string; }
    $left = 0;
    if ($pad_type == STR_PAD_LEFT) { $left = $n; }
    elseif ($pad_type == STR_PAD_BOTH) { $left = intdiv($n, 2); }
    $fill = str_repeat($pad_string, intdiv($n, strlen($pad_string)) + 1);
    return substr($fill, 0, $left) . $string . substr($fill, 0, $n - $left);
}
function chunk_split(string $string, int $length = 76, string $separator = "\r\n") {
    if ($length < 1) { throw new ValueError("chunk_split(): Argument #2 (\$length) must be greater than 0"); }
    $out = "";
    foreach (str_split($string, $length) as $p) { $out .= $p . $separator; }
    return $out;
}
function strrev(string $string) {
    $out = "";
    for ($i = strlen($string) - 1; $i >= 0; $i--) { $out .= substr($string, $i, 1); }
    return $out;
}
function bin2hex(string $string) {
    $hex = "0123456789abcdef";
    $out = "";
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $b = ord(substr($string, $i, 1));
        $out .= $hex[$b >> 4] . $hex[$b & 15];
    }
    return $out;
}

// ----- strings: searching ----------------------------------------------------
function str_contains(string $haystack, string $needle) { return strpos($haystack, $needle) !== false; }
function str_starts_with(string $haystack, string $needle) { return substr($haystack, 0, strlen($needle)) === $needle; }
function str_ends_with(string $haystack, string $needle) {
    $n = strlen($needle);
    return $n == 0 || substr($haystack, -$n) === $needle;
}
function stripos(string $haystack, string $needle, int $offset = 0) {
    try {
        return strpos(strtolower($haystack), strtolower($needle), $offset);
    } catch (ValueError $e) {
        throw new ValueError("stripos(): Argument #3 (\$offset) must be contained in argument #1 (\$haystack)");
    }
}
function strripos(string $haystack, string $needle, int $offset = 0) {
    try {
        return strrpos(strtolower($haystack), strtolower($needle), $offset);
    } catch (ValueError $e) {
        throw new ValueError("strripos(): Argument #3 (\$offset) must be contained in argument #1 (\$haystack)");
    }
}
function strstr(string $haystack, string $needle, bool $before_needle = false) {
    $p = strpos($haystack, $needle);
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function strchr(string $haystack, string $needle, bool $before_needle = false) { return strstr($haystack, $needle, $before_needle); }
function stristr(string $haystack, string $needle, bool $before_needle = false) {
    $p = stripos($haystack, $needle);
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function strrchr(string $haystack, string $needle, bool $before_needle = false) {
    $p = strrpos($haystack, substr($needle, 0, 1));
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function substr_count(string $haystack, string $needle, int $offset = 0, ?int $length = null) {
    if ($needle === "") { throw new ValueError("substr_count(): Argument #2 (\$needle) cannot be empty"); }
    $s = substr($haystack, $offset, $length);
    $n = 0;
    $at = 0;
    $w = strlen($needle);
    while (($p = strpos($s, $needle, $at)) !== false) {
        $n++;
        $at = $p + $w;
    }
    return $n;
}

// ----- strings: comparing ----------------------------------------------------
// strcmp is the native (byte order, -1 / 0 / 1); the rest are written on it.
function strcasecmp(string $string1, string $string2) { return strcmp(strtolower($string1), strtolower($string2)); }
function strncmp(string $string1, string $string2, int $length) {
    if ($length < 0) { throw new ValueError("strncmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(substr($string1, 0, $length), substr($string2, 0, $length));
}
function strncasecmp(string $string1, string $string2, int $length) {
    if ($length < 0) { throw new ValueError("strncasecmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(strtolower(substr($string1, 0, $length)), strtolower(substr($string2, 0, $length)));
}
// strnatcmp_ex: leading zeros skipped once, whitespace skipped, a run of digits
// compared by value (or left-aligned when either starts with 0), anything else
// byte by byte - folding ASCII case for strnatcasecmp.
function strnatcmp(string $string1, string $string2) { return __mec_natcmp($string1, $string2, false); }
function strnatcasecmp(string $string1, string $string2) { return __mec_natcmp($string1, $string2, true); }
// The one-byte piece at $i, "" past the end (C's terminating NUL).
function __mec_byte($s, $i, $n) { return $i < $n ? substr($s, $i, 1) : ""; }
function __mec_isdigit($c) { return $c !== "" && strpos("0123456789", $c) !== false; }
function __mec_isspace($c) { return $c !== "" && strpos(" \t\n\v\f\r", $c) !== false; }
function __mec_natcmp($string1, $string2, $ci) {
    $a = (string)$string1;
    $b = (string)$string2;
    $na = strlen($a);
    $nb = strlen($b);
    if ($na == 0 || $nb == 0) { return $na == $nb ? 0 : ($na > $nb ? 1 : -1); }
    $ap = 0;
    $bp = 0;
    $leading = true;
    while (true) {
        $ca = __mec_byte($a, $ap, $na);
        $cb = __mec_byte($b, $bp, $nb);
        while ($leading && $ca === "0" && $ap + 1 < $na && __mec_isdigit(__mec_byte($a, $ap + 1, $na))) { $ap++; $ca = __mec_byte($a, $ap, $na); }
        while ($leading && $cb === "0" && $bp + 1 < $nb && __mec_isdigit(__mec_byte($b, $bp + 1, $nb))) { $bp++; $cb = __mec_byte($b, $bp, $nb); }
        $leading = false;
        while (__mec_isspace($ca)) { $ap++; $ca = __mec_byte($a, $ap, $na); }
        while (__mec_isspace($cb)) { $bp++; $cb = __mec_byte($b, $bp, $nb); }
        if (__mec_isdigit($ca) && __mec_isdigit($cb)) {
            $r = __mec_natrun($a, $ap, $na, $b, $bp, $nb, $ca === "0" || $cb === "0");
            if ($r != 0) { return $r; }
            if ($ap == $na && $bp == $nb) { return 0; }
            if ($ap == $na) { return -1; }
            if ($bp == $nb) { return 1; }
            $ca = __mec_byte($a, $ap, $na);
            $cb = __mec_byte($b, $bp, $nb);
        }
        if ($ci) {
            $ca = strtoupper($ca);
            $cb = strtoupper($cb);
        }
        $c = strcmp($ca, $cb);
        if ($c != 0) { return $c; }
        $ap++;
        $bp++;
        if ($ap >= $na && $bp >= $nb) { return 0; }
        if ($ap >= $na) { return -1; }
        if ($bp >= $nb) { return 1; }
    }
}
// compare_left (a fractional run, $left: the first differing digit wins) and
// compare_right (the longer run wins, else the first difference), advancing both
// positions past the digits they read.
function __mec_natrun($a, &$ap, $na, $b, &$bp, $nb, $left) {
    $bias = 0;
    while (true) {
        $da = __mec_isdigit(__mec_byte($a, $ap, $na));
        $db = __mec_isdigit(__mec_byte($b, $bp, $nb));
        if (!$da && !$db) { return $bias; }
        if (!$da) { return -1; }
        if (!$db) { return 1; }
        $c = strcmp(__mec_byte($a, $ap, $na), __mec_byte($b, $bp, $nb));
        if ($left && $c != 0) { return $c; }
        if ($bias == 0) { $bias = $c; }
        $ap++;
        $bp++;
    }
}
function levenshtein($string1, $string2, $insertion_cost = 1, $replacement_cost = 1, $deletion_cost = 1) {
    $a = str_split((string)$string1);
    $b = str_split((string)$string2);
    $na = strlen($string1);
    $nb = strlen($string2);
    if ($na == 0) { return $nb * $insertion_cost; }
    if ($nb == 0) { return $na * $deletion_cost; }
    $p1 = [];
    for ($j = 0; $j <= $nb; $j++) { $p1[] = $j * $insertion_cost; }
    for ($i = 0; $i < $na; $i++) {
        $p2 = [$p1[0] + $deletion_cost];
        for ($j = 0; $j < $nb; $j++) {
            $c0 = $p1[$j] + ($a[$i] === $b[$j] ? 0 : $replacement_cost);
            $c1 = $p1[$j + 1] + $deletion_cost;
            if ($c1 < $c0) { $c0 = $c1; }
            $c2 = $p2[$j] + $insertion_cost;
            if ($c2 < $c0) { $c0 = $c2; }
            $p2[] = $c0;
        }
        $p1 = $p2;
    }
    return $p1[$nb];
}
// php_similar_char: the longest common run, then the same on what lies left of
// it (only when it was not the first run found) and right of it.
function __mec_similar($a, $a0, $la, $b, $b0, $lb) {
    $max = 0;
    $count = 0;
    $pa = 0;
    $pb = 0;
    for ($i = 0; $i < $la; $i++) {
        for ($j = 0; $j < $lb; $j++) {
            $l = 0;
            while ($i + $l < $la && $j + $l < $lb && $a[$a0 + $i + $l] === $b[$b0 + $j + $l]) { $l++; }
            if ($l > $max) {
                $max = $l;
                $count++;
                $pa = $i;
                $pb = $j;
            }
        }
    }
    $sum = $max;
    if ($sum > 0) {
        if ($pa > 0 && $pb > 0 && $count > 1) { $sum += __mec_similar($a, $a0, $pa, $b, $b0, $pb); }
        if ($pa + $max < $la && $pb + $max < $lb) {
            $sum += __mec_similar($a, $a0 + $pa + $max, $la - $pa - $max, $b, $b0 + $pb + $max, $lb - $pb - $max);
        }
    }
    return $sum;
}
function similar_text($string1, $string2, &$percent = null) {
    $la = strlen($string1);
    $lb = strlen($string2);
    if ($la + $lb == 0) {
        $percent = 0.0;
        return 0;
    }
    $sim = __mec_similar(str_split((string)$string1), 0, $la, str_split((string)$string2), 0, $lb);
    $percent = $sim * 200.0 / ($la + $lb);
    return $sim;
}

// ----- strings: case and whitespace ------------------------------------------
function ucfirst(string $string) { return strtoupper(substr($string, 0, 1)) . substr($string, 1); }
function lcfirst(string $string) { return strtolower(substr($string, 0, 1)) . substr($string, 1); }
function ucwords(string $string, string $separators = " \t\r\n\f\v") {
    $out = "";
    $at = 0;
    $up = true;
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($string, $i, 1);
        if ($up) {
            $u = strtoupper($c);
            if ($u !== $c) {
                $out .= substr($string, $at, $i - $at) . $u;
                $at = $i + 1;
            }
        }
        $up = strpos($separators, $c) !== false;
    }
    return $out . substr($string, $at);
}
// The byte set a trim character list names, keyed by the one-byte piece: "a..z"
// is a range.
function __mec_charset($characters) {
    $set = [];
    $n = strlen($characters);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($characters, $i, 1);
        if ($i + 3 < $n && substr($characters, $i + 1, 2) == "..") {
            $e = ord(substr($characters, $i + 3, 1));
            for ($x = ord($c); $x <= $e; $x++) { $set[chr($x)] = true; }
            $i += 3;
        } else {
            $set[$c] = true;
        }
    }
    return $set;
}
function __mec_trim($string, $characters, $left, $right) {
    $set = __mec_charset($characters);
    $string = (string)$string;
    $a = 0;
    $b = strlen($string);
    if ($left) { while ($a < $b && isset($set[substr($string, $a, 1)])) { $a++; } }
    if ($right) { while ($b > $a && isset($set[substr($string, $b - 1, 1)])) { $b--; } }
    return substr($string, $a, $b - $a);
}
function trim(string $string, string $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, true, true); }
function ltrim(string $string, string $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, true, false); }
function rtrim(string $string, string $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, false, true); }
function chop(string $string, string $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, false, true); }

// ----- strings: paths ---------------------------------------------------------
// php_basename and zend_dirname on a POSIX system: '/' is the only separator,
// trailing slashes do not count, and the file system is never consulted.
function basename($path, $suffix = "") {
    $path = (string)$path;
    $end = strlen($path);
    while ($end > 0 && substr($path, $end - 1, 1) === "/") { $end--; }
    $start = $end;
    while ($start > 0 && substr($path, $start - 1, 1) !== "/") { $start--; }
    $n = strlen($suffix);
    if ($n > 0 && $n < $end - $start && substr($path, $end - $n, $n) === (string)$suffix) { $end -= $n; }
    return substr($path, $start, $end - $start);
}
// One level up: "" stays "", a path of slashes is "/", a bare name is ".".
function __mec_dirname($path) {
    if ($path === "") { return ""; }
    $end = strlen($path);
    while ($end > 0 && substr($path, $end - 1, 1) === "/") { $end--; }
    if ($end == 0) { return "/"; }
    while ($end > 0 && substr($path, $end - 1, 1) !== "/") { $end--; }
    if ($end == 0) { return "."; }
    while ($end > 0 && substr($path, $end - 1, 1) === "/") { $end--; }
    if ($end == 0) { return "/"; }
    return substr($path, 0, $end);
}
// More levels stop early once a level no longer shortens the path.
function dirname($path, $levels = 1) {
    $path = (string)$path;
    if ($levels < 1) { throw new ValueError("dirname(): Argument #2 (\$levels) must be greater than or equal to 1"); }
    do {
        $n = strlen($path);
        $path = __mec_dirname($path);
        $levels--;
    } while ($levels > 0 && strlen($path) < $n);
    return $path;
}
// The extension follows the basename's last '.', and the filename precedes it.
// "dirname" is left out only for the empty path. With one flag the answer is
// that element, or "" when it is absent; with several, the first one present.
function pathinfo($path, $flags = 15) {
    $path = (string)$path;
    $out = [];
    if (($flags & PATHINFO_DIRNAME) != 0 && $path !== "") { $out["dirname"] = __mec_dirname($path); }
    $base = basename($path);
    if (($flags & PATHINFO_BASENAME) != 0) { $out["basename"] = $base; }
    $dot = strrpos($base, ".");
    if (($flags & PATHINFO_EXTENSION) != 0 && $dot !== false) { $out["extension"] = substr($base, $dot + 1); }
    if (($flags & PATHINFO_FILENAME) != 0) { $out["filename"] = $dot === false ? $base : substr($base, 0, $dot); }
    if ($flags == 15) { return $out; }
    foreach ($out as $v) { return $v; }
    return "";
}

// ----- strings: replacing ----------------------------------------------------
// One search string, every non-overlapping occurrence left to right; $ci folds
// ASCII case for the search. $count is incremented per replacement.
function __mec_replace1($search, $replace, $subject, $ci, &$count) {
    $search = (string)$search;
    if ($search === "") { return $subject; }
    $hay = $ci ? strtolower($subject) : $subject;
    $needle = $ci ? strtolower($search) : $search;
    $out = "";
    $at = 0;
    $n = strlen($search);
    while (($p = strpos($hay, $needle, $at)) !== false) {
        $out .= substr($subject, $at, $p - $at) . $replace;
        $at = $p + $n;
        $count++;
    }
    return $out . substr($subject, $at);
}
function __mec_replace($search, $replace, $subject, $ci, &$count, $fname) {
    if (is_array($subject)) {
        $out = [];
        foreach ($subject as $k => $v) { $out[$k] = __mec_replace($search, $replace, $v, $ci, $count, $fname); }
        return $out;
    }
    $subject = (string)$subject;
    if (!is_array($search)) {
        if (is_array($replace)) {
            throw new TypeError($fname . "(): Argument #2 (\$replace) must be of type string when argument #1 (\$search) is a string");
        }
        return __mec_replace1($search, (string)$replace, $subject, $ci, $count);
    }
    $reps = is_array($replace) ? array_values($replace) : null;
    $i = 0;
    foreach ($search as $s) {
        $r = $reps === null ? (string)$replace : (string)($reps[$i] ?? "");
        $subject = __mec_replace1($s, $r, $subject, $ci, $count);
        $i++;
    }
    return $subject;
}
function str_replace($search, $replace, $subject, &$count = null) {
    $count = 0;
    return __mec_replace($search, $replace, $subject, false, $count, "str_replace");
}
function str_ireplace($search, $replace, $subject, &$count = null) {
    $count = 0;
    return __mec_replace($search, $replace, $subject, true, $count, "str_ireplace");
}
function substr_replace($string, $replace, $offset, $length = null) {
    $n = strlen($string);
    if ($offset < 0) { $offset = $n + $offset < 0 ? 0 : $n + $offset; }
    if ($offset > $n) { $offset = $n; }
    if ($length === null) { $length = $n; }
    if ($length < 0) { $length = $n - $offset + $length < 0 ? 0 : $n - $offset + $length; }
    return substr($string, 0, $offset) . $replace . substr($string, $offset + $length);
}
function strtr($string, $from, $to = null) {
    $string = (string)$string;
    $n = strlen($string);
    $out = "";
    $at = 0;
    if ($to !== null) {
        $from = (string)$from;
        $to = (string)$to;
        $m = strlen($from) < strlen($to) ? strlen($from) : strlen($to);
        $map = [];
        for ($i = 0; $i < $m; $i++) { $map[substr($from, $i, 1)] = substr($to, $i, 1); }
        if (count($map) == 0) { return $string; }
        for ($i = 0; $i < $n; $i++) {
            $c = substr($string, $i, 1);
            if (isset($map[$c])) {
                $out .= substr($string, $at, $i - $at) . $map[$c];
                $at = $i + 1;
            }
        }
        return $out . substr($string, $at);
    }
    $pairs = [];
    foreach ($from as $k => $v) { if ((string)$k !== "") { $pairs[(string)$k] = (string)$v; } }
    $i = 0;
    while ($i < $n) {
        $best = null;
        foreach ($pairs as $k => $v) {
            $k = (string)$k;
            if (($best === null || strlen($k) > strlen($best)) && substr($string, $i, strlen($k)) === $k) { $best = $k; }
        }
        if ($best === null) {
            $i++;
        } else {
            $out .= substr($string, $at, $i - $at) . $pairs[$best];
            $i += strlen($best);
            $at = $i;
        }
    }
    return $out . substr($string, $at);
}
function nl2br(string $string, bool $use_xhtml = true) {
    $br = $use_xhtml ? "<br />" : "<br>";
    $out = "";
    $at = 0;
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($string, $i, 1);
        if ($c === "\r" || $c === "\n") {
            $d = substr($string, $i + 1, 1);
            $end = ($d === "\r" || $d === "\n") && $d !== $c ? $i + 2 : $i + 1;
            $out .= substr($string, $at, $i - $at) . $br . substr($string, $i, $end - $i);
            $at = $end;
            $i = $end - 1;
        }
    }
    return $out . substr($string, $at);
}
// php_escape_html_entities_ex with the basic table, UTF-8. The flags' bit 2 encodes
// ", bit 1 encodes ' (&#039; under ENT_HTML401, &apos; under the other doctypes). An
// invalid UTF-8 sequence (get_next_char) is dropped under ENT_IGNORE, becomes U+FFFD
// under ENT_SUBSTITUTE and empties the result otherwise; under ENT_DISALLOWED a
// character the doctype does not admit becomes U+FFFD. Without $double_encode an
// entity the doctype's table knows, or a numeric one, is copied as it is.
function htmlspecialchars(string $string, int $flags = 11, ?string $encoding = null, bool $double_encode = true) {
    $map = ["&" => "&amp;", "<" => "&lt;", ">" => "&gt;"];
    if ($flags & 2) { $map["\""] = "&quot;"; }
    if ($flags & 1) { $map["'"] = ($flags & 48) == 0 ? "&#039;" : "&apos;"; }
    $doc = $flags & 48;
    $out = "";
    $at = 0;
    $n = strlen($string);
    $i = 0;
    while ($i < $n) {
        $c = $string[$i];
        $cp = ord($c);
        $len = 1;
        $rep = null;
        if ($cp >= 0x80) {
            $u = __mec_utf8at($string, $i, $n);
            if ($u === null) {
                if (!($flags & 12)) { return ""; }
                $rep = $flags & 4 ? "" : "\u{FFFD}";
                $len = __mec_utf8bad($string, $i, $n);
            } else {
                [$cp, $len] = $u;
            }
        } elseif ($c === "&" && !$double_encode) {
            $e = __mec_entat($string, $i + 1, $n, $doc, true);
            if ($e !== null && ($e[2] || !($flags & 128) || __mec_numentallowed($e[0], $doc))) {
                $i = $e[1] + 1;
                continue;
            }
            $rep = "&amp;";
        } else {
            $rep = $map[$c] ?? null;
        }
        if ($rep === null && ($flags & 128) && !__mec_cpallowed($cp, $doc)) { $rep = "\u{FFFD}"; }
        if ($rep !== null) {
            $out .= substr($string, $at, $i - $at) . $rep;
            $at = $i + $len;
        }
        $i += $len;
    }
    return $out . substr($string, $at);
}
// How many bytes get_next_char's UTF-8 decoder skips over at the malformed sequence
// at byte $i: never a later byte that is a valid single byte or lead byte.
function __mec_utf8bad($s, $i, $n) {
    $c = ord($s[$i]);
    $need = $c < 0xC2 ? 1 : ($c < 0xE0 ? 2 : ($c < 0xF0 ? 3 : ($c < 0xF5 ? 4 : 1)));
    for ($k = 1; $k < $need; $k++) {
        if ($i + $k >= $n) { return $k; }
        $b = ord($s[$i + $k]);
        if ($b < 0x80 || ($b >= 0xC2 && $b <= 0xF4)) { return $k; }
    }
    return $need;
}
// The entity whose text starts at byte $q of $s (after the &), as traverse_for_entities
// reads one: [code point or [cp1, cp2], the position of its ;, whether it is a name]
// for &#digits; / &#xhex; up to U+10FFFF or for an &name; __mec_entname knows, else
// null.
function __mec_entat($s, $q, $n, $doc, $all) {
    $alnum = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ";
    if ($q < $n && $s[$q] === "#") {
        $q++;
        $base = $q < $n && ($s[$q] === "x" || $s[$q] === "X") ? 16 : 10;
        if ($base == 16) { $q++; }
        $start = $q;
        $v = 0;
        while ($q < $n && ($d = strpos($alnum, strtolower($s[$q]))) !== false && $d < $base) {
            $v = min($v * $base + $d, 0x110000);
            $q++;
        }
        return $q > $start && $q < $n && $s[$q] === ";" && $v <= 0x10FFFF ? [$v, $q, false] : null;
    }
    $p = $q;
    while ($q < $n && strpos($alnum, $s[$q]) !== false) { $q++; }
    if ($q == $p || $q >= $n || $s[$q] !== ";") { return null; }
    $code = __mec_entname(substr($s, $p, $q - $p), $doc, $all);
    return $code === null ? null : [$code, $q, true];
}
function htmlspecialchars_decode(string $string, int $flags = 11) { return __mec_entdecode($string, $flags, false); }
function html_entity_decode(string $string, int $flags = 11, ?string $encoding = null) { return __mec_entdecode($string, $flags, true); }
// traverse_for_entities: an entity is &name; or &#digits; / &#xhex; with at least
// three bytes after the &. Without $all only the five basic characters decode,
// named or numeric; with $all every named entity of the doctype's table and every
// code point the doctype allows (UTF-8). A quote stays encoded unless its flag bit
// is set.
function __mec_entdecode($string, $flags, $all) {
    $doc = $flags & 48;
    $out = "";
    $at = 0;
    $n = strlen($string);
    $p = strpos($string, "&");
    while ($p !== false && $p + 3 < $n) {
        $e = __mec_entat($string, $p + 1, $n, $doc, $all);
        $code = $e === null ? -1 : $e[0];
        $code2 = -1;
        if (is_array($code)) { [$code, $code2] = $code; }
        elseif ($e !== null && !$e[2] && !(($all || $code == 34 || $code == 38 || $code == 39 || $code == 60 || $code == 62)
                && __mec_cpallowed($code, $doc) && !($doc == 48 && $code == 13))) {
            $code = -1;
        }
        if (($code == 39 && !($flags & 1)) || ($code == 34 && !($flags & 2))) { $code = -1; }
        if ($code >= 0) {
            $out .= substr($string, $at, $p - $at) . __mec_utf8chr($code) . ($code2 >= 0 ? __mec_utf8chr($code2) : "");
            $at = $e[1] + 1;
        }
        $p = strpos($string, "&", $code >= 0 ? $at : $p + 1);
    }
    return $out . substr($string, $at);
}
// The code point(s) of the named entity $name in the table a doctype decodes with
// (unescape_inverse_map), or null: with $all the HTML 4.01 table, HTML5's own under
// ENT_HTML5 and only the basic ones under ENT_XML1; &apos; is known to every doctype
// but HTML 4.01. The big tables are built once.
function __mec_entname($name, $doc, $all) {
    static $h4 = null;
    static $h5 = null;
    if ($all && $doc == 48) {
        $h5 ??= __mec_html5ents();
        return $h5[$name] ?? null;
    }
    if ($name === "apos") { return $doc != 0 ? 39 : null; }
    if ($all && $doc != 16) {
        $h4 ??= __mec_html4ents();
        return $h4[$name] ?? null;
    }
    return ["quot" => 34, "amp" => 38, "lt" => 60, "gt" => 62][$name] ?? null;
}
// unicode_cp_is_allowed: the code points a document type admits.
function __mec_cpallowed($cp, $doc) {
    if ($doc == 0) {
        return ($cp >= 0x20 && $cp <= 0x7E) || $cp == 9 || $cp == 10 || $cp == 13 || ($cp >= 0xA0 && $cp <= 0xD7FF) || $cp >= 0xE000;
    }
    if ($doc == 48) {
        return ($cp >= 0x20 && $cp <= 0x7E) || ($cp >= 9 && $cp <= 13 && $cp != 11) || ($cp >= 0xA0 && $cp <= 0xD7FF)
            || ($cp >= 0xE000 && ($cp & 0xFFFF) < 0xFFFE && ($cp < 0xFDD0 || $cp > 0xFDEF));
    }
    return ($cp >= 0x20 && $cp <= 0xD7FF) || $cp == 9 || $cp == 10 || $cp == 13 || ($cp >= 0xE000 && $cp != 0xFFFE && $cp != 0xFFFF);
}
// numeric_entity_is_allowed: the code points a numeric entity may name, which
// HTML 4.01 and HTML5 admit more of than the characters themselves.
function __mec_numentallowed($cp, $doc) {
    if ($doc == 0) { return true; }
    if ($doc == 48) {
        return ($cp >= 0x20 && $cp <= 0x7E) || $cp == 9 || $cp == 10 || $cp == 12
            || ($cp >= 0xA0 && ($cp & 0xFFFF) < 0xFFFE && ($cp < 0xFDD0 || $cp > 0xFDEF));
    }
    return __mec_cpallowed($cp, $doc);
}
// The HTML 4.01 named entities and their code points (ext/standard/html_tables.h).
function __mec_html4ents() {
    return [
    "quot" => 34, "amp" => 38, "lt" => 60, "gt" => 62, "nbsp" => 160, "iexcl" => 161,
    "cent" => 162, "pound" => 163, "curren" => 164, "yen" => 165, "brvbar" => 166,
    "sect" => 167, "uml" => 168, "copy" => 169, "ordf" => 170, "laquo" => 171, "not" => 172,
    "shy" => 173, "reg" => 174, "macr" => 175, "deg" => 176, "plusmn" => 177, "sup2" => 178,
    "sup3" => 179, "acute" => 180, "micro" => 181, "para" => 182, "middot" => 183,
    "cedil" => 184, "sup1" => 185, "ordm" => 186, "raquo" => 187, "frac14" => 188,
    "frac12" => 189, "frac34" => 190, "iquest" => 191, "Agrave" => 192, "Aacute" => 193,
    "Acirc" => 194, "Atilde" => 195, "Auml" => 196, "Aring" => 197, "AElig" => 198,
    "Ccedil" => 199, "Egrave" => 200, "Eacute" => 201, "Ecirc" => 202, "Euml" => 203,
    "Igrave" => 204, "Iacute" => 205, "Icirc" => 206, "Iuml" => 207, "ETH" => 208,
    "Ntilde" => 209, "Ograve" => 210, "Oacute" => 211, "Ocirc" => 212, "Otilde" => 213,
    "Ouml" => 214, "times" => 215, "Oslash" => 216, "Ugrave" => 217, "Uacute" => 218,
    "Ucirc" => 219, "Uuml" => 220, "Yacute" => 221, "THORN" => 222, "szlig" => 223,
    "agrave" => 224, "aacute" => 225, "acirc" => 226, "atilde" => 227, "auml" => 228,
    "aring" => 229, "aelig" => 230, "ccedil" => 231, "egrave" => 232, "eacute" => 233,
    "ecirc" => 234, "euml" => 235, "igrave" => 236, "iacute" => 237, "icirc" => 238,
    "iuml" => 239, "eth" => 240, "ntilde" => 241, "ograve" => 242, "oacute" => 243,
    "ocirc" => 244, "otilde" => 245, "ouml" => 246, "divide" => 247, "oslash" => 248,
    "ugrave" => 249, "uacute" => 250, "ucirc" => 251, "uuml" => 252, "yacute" => 253,
    "thorn" => 254, "yuml" => 255, "OElig" => 338, "oelig" => 339, "Scaron" => 352,
    "scaron" => 353, "Yuml" => 376, "fnof" => 402, "circ" => 710, "tilde" => 732,
    "Alpha" => 913, "Beta" => 914, "Gamma" => 915, "Delta" => 916, "Epsilon" => 917,
    "Zeta" => 918, "Eta" => 919, "Theta" => 920, "Iota" => 921, "Kappa" => 922, "Lambda" => 923,
    "Mu" => 924, "Nu" => 925, "Xi" => 926, "Omicron" => 927, "Pi" => 928, "Rho" => 929,
    "Sigma" => 931, "Tau" => 932, "Upsilon" => 933, "Phi" => 934, "Chi" => 935, "Psi" => 936,
    "Omega" => 937, "alpha" => 945, "beta" => 946, "gamma" => 947, "delta" => 948,
    "epsilon" => 949, "zeta" => 950, "eta" => 951, "theta" => 952, "iota" => 953,
    "kappa" => 954, "lambda" => 955, "mu" => 956, "nu" => 957, "xi" => 958, "omicron" => 959,
    "pi" => 960, "rho" => 961, "sigmaf" => 962, "sigma" => 963, "tau" => 964, "upsilon" => 965,
    "phi" => 966, "chi" => 967, "psi" => 968, "omega" => 969, "thetasym" => 977, "upsih" => 978,
    "piv" => 982, "ensp" => 8194, "emsp" => 8195, "thinsp" => 8201, "zwnj" => 8204,
    "zwj" => 8205, "lrm" => 8206, "rlm" => 8207, "ndash" => 8211, "mdash" => 8212,
    "lsquo" => 8216, "rsquo" => 8217, "sbquo" => 8218, "ldquo" => 8220, "rdquo" => 8221,
    "bdquo" => 8222, "dagger" => 8224, "Dagger" => 8225, "bull" => 8226, "hellip" => 8230,
    "permil" => 8240, "prime" => 8242, "Prime" => 8243, "lsaquo" => 8249, "rsaquo" => 8250,
    "oline" => 8254, "frasl" => 8260, "euro" => 8364, "image" => 8465, "weierp" => 8472,
    "real" => 8476, "trade" => 8482, "alefsym" => 8501, "larr" => 8592, "uarr" => 8593,
    "rarr" => 8594, "darr" => 8595, "harr" => 8596, "crarr" => 8629, "lArr" => 8656,
    "uArr" => 8657, "rArr" => 8658, "dArr" => 8659, "hArr" => 8660, "forall" => 8704,
    "part" => 8706, "exist" => 8707, "empty" => 8709, "nabla" => 8711, "isin" => 8712,
    "notin" => 8713, "ni" => 8715, "prod" => 8719, "sum" => 8721, "minus" => 8722,
    "lowast" => 8727, "radic" => 8730, "prop" => 8733, "infin" => 8734, "ang" => 8736,
    "and" => 8743, "or" => 8744, "cap" => 8745, "cup" => 8746, "int" => 8747, "there4" => 8756,
    "sim" => 8764, "cong" => 8773, "asymp" => 8776, "ne" => 8800, "equiv" => 8801, "le" => 8804,
    "ge" => 8805, "sub" => 8834, "sup" => 8835, "nsub" => 8836, "sube" => 8838, "supe" => 8839,
    "oplus" => 8853, "otimes" => 8855, "perp" => 8869, "sdot" => 8901, "lceil" => 8968,
    "rceil" => 8969, "lfloor" => 8970, "rfloor" => 8971, "lang" => 9001, "rang" => 9002,
    "loz" => 9674, "spades" => 9824, "clubs" => 9827, "hearts" => 9829, "diams" => 9830
    ];
}
// The HTML5 named entities (ext/standard/html_tables.h, ent_ht_html5): a code
// point, or two for the names that stand for a pair.
function __mec_html5ents() {
    return [
    "AElig" => 198, "AMP" => 38, "Aacute" => 193, "Abreve" => 258, "Acirc" => 194,
    "Acy" => 1040, "Afr" => 120068, "Agrave" => 192, "Alpha" => 913, "Amacr" => 256,
    "And" => 10835, "Aogon" => 260, "Aopf" => 120120, "ApplyFunction" => 8289,
    "Aring" => 197, "Ascr" => 119964, "Assign" => 8788, "Atilde" => 195, "Auml" => 196,
    "Backslash" => 8726, "Barv" => 10983, "Barwed" => 8966, "Bcy" => 1041,
    "Because" => 8757, "Bernoullis" => 8492, "Beta" => 914, "Bfr" => 120069,
    "Bopf" => 120121, "Breve" => 728, "Bscr" => 8492, "Bumpeq" => 8782, "CHcy" => 1063,
    "COPY" => 169, "Cacute" => 262, "Cap" => 8914, "CapitalDifferentialD" => 8517,
    "Cayleys" => 8493, "Ccaron" => 268, "Ccedil" => 199, "Ccirc" => 264, "Cconint" => 8752,
    "Cdot" => 266, "Cedilla" => 184, "CenterDot" => 183, "Cfr" => 8493, "Chi" => 935,
    "CircleDot" => 8857, "CircleMinus" => 8854, "CirclePlus" => 8853,
    "CircleTimes" => 8855, "ClockwiseContourIntegral" => 8754,
    "CloseCurlyDoubleQuote" => 8221, "CloseCurlyQuote" => 8217, "Colon" => 8759,
    "Colone" => 10868, "Congruent" => 8801, "Conint" => 8751, "ContourIntegral" => 8750,
    "Copf" => 8450, "Coproduct" => 8720, "CounterClockwiseContourIntegral" => 8755,
    "Cross" => 10799, "Cscr" => 119966, "Cup" => 8915, "CupCap" => 8781, "DD" => 8517,
    "DDotrahd" => 10513, "DJcy" => 1026, "DScy" => 1029, "DZcy" => 1039, "Dagger" => 8225,
    "Darr" => 8609, "Dashv" => 10980, "Dcaron" => 270, "Dcy" => 1044, "Del" => 8711,
    "Delta" => 916, "Dfr" => 120071, "DiacriticalAcute" => 180, "DiacriticalDot" => 729,
    "DiacriticalDoubleAcute" => 733, "DiacriticalGrave" => 96, "DiacriticalTilde" => 732,
    "Diamond" => 8900, "DifferentialD" => 8518, "Dopf" => 120123, "Dot" => 168,
    "DotDot" => 8412, "DotEqual" => 8784, "DoubleContourIntegral" => 8751,
    "DoubleDot" => 168, "DoubleDownArrow" => 8659, "DoubleLeftArrow" => 8656,
    "DoubleLeftRightArrow" => 8660, "DoubleLeftTee" => 10980,
    "DoubleLongLeftArrow" => 10232, "DoubleLongLeftRightArrow" => 10234,
    "DoubleLongRightArrow" => 10233, "DoubleRightArrow" => 8658, "DoubleRightTee" => 8872,
    "DoubleUpArrow" => 8657, "DoubleUpDownArrow" => 8661, "DoubleVerticalBar" => 8741,
    "DownArrow" => 8595, "DownArrowBar" => 10515, "DownArrowUpArrow" => 8693,
    "DownBreve" => 785, "DownLeftRightVector" => 10576, "DownLeftTeeVector" => 10590,
    "DownLeftVector" => 8637, "DownLeftVectorBar" => 10582, "DownRightTeeVector" => 10591,
    "DownRightVector" => 8641, "DownRightVectorBar" => 10583, "DownTee" => 8868,
    "DownTeeArrow" => 8615, "Downarrow" => 8659, "Dscr" => 119967, "Dstrok" => 272,
    "ENG" => 330, "ETH" => 208, "Eacute" => 201, "Ecaron" => 282, "Ecirc" => 202,
    "Ecy" => 1069, "Edot" => 278, "Efr" => 120072, "Egrave" => 200, "Element" => 8712,
    "Emacr" => 274, "EmptySmallSquare" => 9723, "EmptyVerySmallSquare" => 9643,
    "Eogon" => 280, "Eopf" => 120124, "Epsilon" => 917, "Equal" => 10869,
    "EqualTilde" => 8770, "Equilibrium" => 8652, "Escr" => 8496, "Esim" => 10867,
    "Eta" => 919, "Euml" => 203, "Exists" => 8707, "ExponentialE" => 8519, "Fcy" => 1060,
    "Ffr" => 120073, "FilledSmallSquare" => 9724, "FilledVerySmallSquare" => 9642,
    "Fopf" => 120125, "ForAll" => 8704, "Fouriertrf" => 8497, "Fscr" => 8497,
    "GJcy" => 1027, "GT" => 62, "Gamma" => 915, "Gammad" => 988, "Gbreve" => 286,
    "Gcedil" => 290, "Gcirc" => 284, "Gcy" => 1043, "Gdot" => 288, "Gfr" => 120074,
    "Gg" => 8921, "Gopf" => 120126, "GreaterEqual" => 8805, "GreaterEqualLess" => 8923,
    "GreaterFullEqual" => 8807, "GreaterGreater" => 10914, "GreaterLess" => 8823,
    "GreaterSlantEqual" => 10878, "GreaterTilde" => 8819, "Gscr" => 119970, "Gt" => 8811,
    "HARDcy" => 1066, "Hacek" => 711, "Hat" => 94, "Hcirc" => 292, "Hfr" => 8460,
    "HilbertSpace" => 8459, "Hopf" => 8461, "HorizontalLine" => 9472, "Hscr" => 8459,
    "Hstrok" => 294, "HumpDownHump" => 8782, "HumpEqual" => 8783, "IEcy" => 1045,
    "IJlig" => 306, "IOcy" => 1025, "Iacute" => 205, "Icirc" => 206, "Icy" => 1048,
    "Idot" => 304, "Ifr" => 8465, "Igrave" => 204, "Im" => 8465, "Imacr" => 298,
    "ImaginaryI" => 8520, "Implies" => 8658, "Int" => 8748, "Integral" => 8747,
    "Intersection" => 8898, "InvisibleComma" => 8291, "InvisibleTimes" => 8290,
    "Iogon" => 302, "Iopf" => 120128, "Iota" => 921, "Iscr" => 8464, "Itilde" => 296,
    "Iukcy" => 1030, "Iuml" => 207, "Jcirc" => 308, "Jcy" => 1049, "Jfr" => 120077,
    "Jopf" => 120129, "Jscr" => 119973, "Jsercy" => 1032, "Jukcy" => 1028, "KHcy" => 1061,
    "KJcy" => 1036, "Kappa" => 922, "Kcedil" => 310, "Kcy" => 1050, "Kfr" => 120078,
    "Kopf" => 120130, "Kscr" => 119974, "LJcy" => 1033, "LT" => 60, "Lacute" => 313,
    "Lambda" => 923, "Lang" => 10218, "Laplacetrf" => 8466, "Larr" => 8606,
    "Lcaron" => 317, "Lcedil" => 315, "Lcy" => 1051, "LeftAngleBracket" => 10216,
    "LeftArrow" => 8592, "LeftArrowBar" => 8676, "LeftArrowRightArrow" => 8646,
    "LeftCeiling" => 8968, "LeftDoubleBracket" => 10214, "LeftDownTeeVector" => 10593,
    "LeftDownVector" => 8643, "LeftDownVectorBar" => 10585, "LeftFloor" => 8970,
    "LeftRightArrow" => 8596, "LeftRightVector" => 10574, "LeftTee" => 8867,
    "LeftTeeArrow" => 8612, "LeftTeeVector" => 10586, "LeftTriangle" => 8882,
    "LeftTriangleBar" => 10703, "LeftTriangleEqual" => 8884, "LeftUpDownVector" => 10577,
    "LeftUpTeeVector" => 10592, "LeftUpVector" => 8639, "LeftUpVectorBar" => 10584,
    "LeftVector" => 8636, "LeftVectorBar" => 10578, "Leftarrow" => 8656,
    "Leftrightarrow" => 8660, "LessEqualGreater" => 8922, "LessFullEqual" => 8806,
    "LessGreater" => 8822, "LessLess" => 10913, "LessSlantEqual" => 10877,
    "LessTilde" => 8818, "Lfr" => 120079, "Ll" => 8920, "Lleftarrow" => 8666,
    "Lmidot" => 319, "LongLeftArrow" => 10229, "LongLeftRightArrow" => 10231,
    "LongRightArrow" => 10230, "Longleftarrow" => 10232, "Longleftrightarrow" => 10234,
    "Longrightarrow" => 10233, "Lopf" => 120131, "LowerLeftArrow" => 8601,
    "LowerRightArrow" => 8600, "Lscr" => 8466, "Lsh" => 8624, "Lstrok" => 321,
    "Lt" => 8810, "Map" => 10501, "Mcy" => 1052, "MediumSpace" => 8287,
    "Mellintrf" => 8499, "Mfr" => 120080, "MinusPlus" => 8723, "Mopf" => 120132,
    "Mscr" => 8499, "Mu" => 924, "NJcy" => 1034, "Nacute" => 323, "Ncaron" => 327,
    "Ncedil" => 325, "Ncy" => 1053, "NegativeMediumSpace" => 8203,
    "NegativeThickSpace" => 8203, "NegativeThinSpace" => 8203,
    "NegativeVeryThinSpace" => 8203, "NestedGreaterGreater" => 8811,
    "NestedLessLess" => 8810, "NewLine" => 10, "Nfr" => 120081, "NoBreak" => 8288,
    "NonBreakingSpace" => 160, "Nopf" => 8469, "Not" => 10988, "NotCongruent" => 8802,
    "NotCupCap" => 8813, "NotDoubleVerticalBar" => 8742, "NotElement" => 8713,
    "NotEqual" => 8800, "NotEqualTilde" => [8770, 824], "NotExists" => 8708,
    "NotGreater" => 8815, "NotGreaterEqual" => 8817, "NotGreaterFullEqual" => [8807, 824],
    "NotGreaterGreater" => [8811, 824], "NotGreaterLess" => 8825,
    "NotGreaterSlantEqual" => [10878, 824], "NotGreaterTilde" => 8821,
    "NotHumpDownHump" => [8782, 824], "NotHumpEqual" => [8783, 824],
    "NotLeftTriangle" => 8938, "NotLeftTriangleBar" => [10703, 824],
    "NotLeftTriangleEqual" => 8940, "NotLess" => 8814, "NotLessEqual" => 8816,
    "NotLessGreater" => 8824, "NotLessLess" => [8810, 824],
    "NotLessSlantEqual" => [10877, 824], "NotLessTilde" => 8820,
    "NotNestedGreaterGreater" => [10914, 824], "NotNestedLessLess" => [10913, 824],
    "NotPrecedes" => 8832, "NotPrecedesEqual" => [10927, 824],
    "NotPrecedesSlantEqual" => 8928, "NotReverseElement" => 8716,
    "NotRightTriangle" => 8939, "NotRightTriangleBar" => [10704, 824],
    "NotRightTriangleEqual" => 8941, "NotSquareSubset" => [8847, 824],
    "NotSquareSubsetEqual" => 8930, "NotSquareSuperset" => [8848, 824],
    "NotSquareSupersetEqual" => 8931, "NotSubset" => [8834, 8402],
    "NotSubsetEqual" => 8840, "NotSucceeds" => 8833, "NotSucceedsEqual" => [10928, 824],
    "NotSucceedsSlantEqual" => 8929, "NotSucceedsTilde" => [8831, 824],
    "NotSuperset" => [8835, 8402], "NotSupersetEqual" => 8841, "NotTilde" => 8769,
    "NotTildeEqual" => 8772, "NotTildeFullEqual" => 8775, "NotTildeTilde" => 8777,
    "NotVerticalBar" => 8740, "Nscr" => 119977, "Ntilde" => 209, "Nu" => 925,
    "OElig" => 338, "Oacute" => 211, "Ocirc" => 212, "Ocy" => 1054, "Odblac" => 336,
    "Ofr" => 120082, "Ograve" => 210, "Omacr" => 332, "Omega" => 937, "Omicron" => 927,
    "Oopf" => 120134, "OpenCurlyDoubleQuote" => 8220, "OpenCurlyQuote" => 8216,
    "Or" => 10836, "Oscr" => 119978, "Oslash" => 216, "Otilde" => 213, "Otimes" => 10807,
    "Ouml" => 214, "OverBar" => 8254, "OverBrace" => 9182, "OverBracket" => 9140,
    "OverParenthesis" => 9180, "PartialD" => 8706, "Pcy" => 1055, "Pfr" => 120083,
    "Phi" => 934, "Pi" => 928, "PlusMinus" => 177, "Poincareplane" => 8460, "Popf" => 8473,
    "Pr" => 10939, "Precedes" => 8826, "PrecedesEqual" => 10927,
    "PrecedesSlantEqual" => 8828, "PrecedesTilde" => 8830, "Prime" => 8243,
    "Product" => 8719, "Proportion" => 8759, "Proportional" => 8733, "Pscr" => 119979,
    "Psi" => 936, "QUOT" => 34, "Qfr" => 120084, "Qopf" => 8474, "Qscr" => 119980,
    "RBarr" => 10512, "REG" => 174, "Racute" => 340, "Rang" => 10219, "Rarr" => 8608,
    "Rarrtl" => 10518, "Rcaron" => 344, "Rcedil" => 342, "Rcy" => 1056, "Re" => 8476,
    "ReverseElement" => 8715, "ReverseEquilibrium" => 8651,
    "ReverseUpEquilibrium" => 10607, "Rfr" => 8476, "Rho" => 929,
    "RightAngleBracket" => 10217, "RightArrow" => 8594, "RightArrowBar" => 8677,
    "RightArrowLeftArrow" => 8644, "RightCeiling" => 8969, "RightDoubleBracket" => 10215,
    "RightDownTeeVector" => 10589, "RightDownVector" => 8642,
    "RightDownVectorBar" => 10581, "RightFloor" => 8971, "RightTee" => 8866,
    "RightTeeArrow" => 8614, "RightTeeVector" => 10587, "RightTriangle" => 8883,
    "RightTriangleBar" => 10704, "RightTriangleEqual" => 8885,
    "RightUpDownVector" => 10575, "RightUpTeeVector" => 10588, "RightUpVector" => 8638,
    "RightUpVectorBar" => 10580, "RightVector" => 8640, "RightVectorBar" => 10579,
    "Rightarrow" => 8658, "Ropf" => 8477, "RoundImplies" => 10608, "Rrightarrow" => 8667,
    "Rscr" => 8475, "Rsh" => 8625, "RuleDelayed" => 10740, "SHCHcy" => 1065,
    "SHcy" => 1064, "SOFTcy" => 1068, "Sacute" => 346, "Sc" => 10940, "Scaron" => 352,
    "Scedil" => 350, "Scirc" => 348, "Scy" => 1057, "Sfr" => 120086,
    "ShortDownArrow" => 8595, "ShortLeftArrow" => 8592, "ShortRightArrow" => 8594,
    "ShortUpArrow" => 8593, "Sigma" => 931, "SmallCircle" => 8728, "Sopf" => 120138,
    "Sqrt" => 8730, "Square" => 9633, "SquareIntersection" => 8851, "SquareSubset" => 8847,
    "SquareSubsetEqual" => 8849, "SquareSuperset" => 8848, "SquareSupersetEqual" => 8850,
    "SquareUnion" => 8852, "Sscr" => 119982, "Star" => 8902, "Sub" => 8912,
    "Subset" => 8912, "SubsetEqual" => 8838, "Succeeds" => 8827, "SucceedsEqual" => 10928,
    "SucceedsSlantEqual" => 8829, "SucceedsTilde" => 8831, "SuchThat" => 8715,
    "Sum" => 8721, "Sup" => 8913, "Superset" => 8835, "SupersetEqual" => 8839,
    "Supset" => 8913, "THORN" => 222, "TRADE" => 8482, "TSHcy" => 1035, "TScy" => 1062,
    "Tab" => 9, "Tau" => 932, "Tcaron" => 356, "Tcedil" => 354, "Tcy" => 1058,
    "Tfr" => 120087, "Therefore" => 8756, "Theta" => 920, "ThickSpace" => [8287, 8202],
    "ThinSpace" => 8201, "Tilde" => 8764, "TildeEqual" => 8771, "TildeFullEqual" => 8773,
    "TildeTilde" => 8776, "Topf" => 120139, "TripleDot" => 8411, "Tscr" => 119983,
    "Tstrok" => 358, "Uacute" => 218, "Uarr" => 8607, "Uarrocir" => 10569, "Ubrcy" => 1038,
    "Ubreve" => 364, "Ucirc" => 219, "Ucy" => 1059, "Udblac" => 368, "Ufr" => 120088,
    "Ugrave" => 217, "Umacr" => 362, "UnderBar" => 95, "UnderBrace" => 9183,
    "UnderBracket" => 9141, "UnderParenthesis" => 9181, "Union" => 8899,
    "UnionPlus" => 8846, "Uogon" => 370, "Uopf" => 120140, "UpArrow" => 8593,
    "UpArrowBar" => 10514, "UpArrowDownArrow" => 8645, "UpDownArrow" => 8597,
    "UpEquilibrium" => 10606, "UpTee" => 8869, "UpTeeArrow" => 8613, "Uparrow" => 8657,
    "Updownarrow" => 8661, "UpperLeftArrow" => 8598, "UpperRightArrow" => 8599,
    "Upsi" => 978, "Upsilon" => 933, "Uring" => 366, "Uscr" => 119984, "Utilde" => 360,
    "Uuml" => 220, "VDash" => 8875, "Vbar" => 10987, "Vcy" => 1042, "Vdash" => 8873,
    "Vdashl" => 10982, "Vee" => 8897, "Verbar" => 8214, "Vert" => 8214,
    "VerticalBar" => 8739, "VerticalLine" => 124, "VerticalSeparator" => 10072,
    "VerticalTilde" => 8768, "VeryThinSpace" => 8202, "Vfr" => 120089, "Vopf" => 120141,
    "Vscr" => 119985, "Vvdash" => 8874, "Wcirc" => 372, "Wedge" => 8896, "Wfr" => 120090,
    "Wopf" => 120142, "Wscr" => 119986, "Xfr" => 120091, "Xi" => 926, "Xopf" => 120143,
    "Xscr" => 119987, "YAcy" => 1071, "YIcy" => 1031, "YUcy" => 1070, "Yacute" => 221,
    "Ycirc" => 374, "Ycy" => 1067, "Yfr" => 120092, "Yopf" => 120144, "Yscr" => 119988,
    "Yuml" => 376, "ZHcy" => 1046, "Zacute" => 377, "Zcaron" => 381, "Zcy" => 1047,
    "Zdot" => 379, "ZeroWidthSpace" => 8203, "Zeta" => 918, "Zfr" => 8488, "Zopf" => 8484,
    "Zscr" => 119989, "aacute" => 225, "abreve" => 259, "ac" => 8766, "acE" => [8766, 819],
    "acd" => 8767, "acirc" => 226, "acute" => 180, "acy" => 1072, "aelig" => 230,
    "af" => 8289, "afr" => 120094, "agrave" => 224, "alefsym" => 8501, "aleph" => 8501,
    "alpha" => 945, "amacr" => 257, "amalg" => 10815, "amp" => 38, "and" => 8743,
    "andand" => 10837, "andd" => 10844, "andslope" => 10840, "andv" => 10842,
    "ang" => 8736, "ange" => 10660, "angle" => 8736, "angmsd" => 8737, "angmsdaa" => 10664,
    "angmsdab" => 10665, "angmsdac" => 10666, "angmsdad" => 10667, "angmsdae" => 10668,
    "angmsdaf" => 10669, "angmsdag" => 10670, "angmsdah" => 10671, "angrt" => 8735,
    "angrtvb" => 8894, "angrtvbd" => 10653, "angsph" => 8738, "angst" => 197,
    "angzarr" => 9084, "aogon" => 261, "aopf" => 120146, "ap" => 8776, "apE" => 10864,
    "apacir" => 10863, "ape" => 8778, "apid" => 8779, "apos" => 39, "approx" => 8776,
    "approxeq" => 8778, "aring" => 229, "ascr" => 119990, "ast" => 42, "asymp" => 8776,
    "asympeq" => 8781, "atilde" => 227, "auml" => 228, "awconint" => 8755,
    "awint" => 10769, "bNot" => 10989, "backcong" => 8780, "backepsilon" => 1014,
    "backprime" => 8245, "backsim" => 8765, "backsimeq" => 8909, "barvee" => 8893,
    "barwed" => 8965, "barwedge" => 8965, "bbrk" => 9141, "bbrktbrk" => 9142,
    "bcong" => 8780, "bcy" => 1073, "bdquo" => 8222, "becaus" => 8757, "because" => 8757,
    "bemptyv" => 10672, "bepsi" => 1014, "bernou" => 8492, "beta" => 946, "beth" => 8502,
    "between" => 8812, "bfr" => 120095, "bigcap" => 8898, "bigcirc" => 9711,
    "bigcup" => 8899, "bigodot" => 10752, "bigoplus" => 10753, "bigotimes" => 10754,
    "bigsqcup" => 10758, "bigstar" => 9733, "bigtriangledown" => 9661,
    "bigtriangleup" => 9651, "biguplus" => 10756, "bigvee" => 8897, "bigwedge" => 8896,
    "bkarow" => 10509, "blacklozenge" => 10731, "blacksquare" => 9642,
    "blacktriangle" => 9652, "blacktriangledown" => 9662, "blacktriangleleft" => 9666,
    "blacktriangleright" => 9656, "blank" => 9251, "blk12" => 9618, "blk14" => 9617,
    "blk34" => 9619, "block" => 9608, "bne" => [61, 8421], "bnequiv" => [8801, 8421],
    "bnot" => 8976, "bopf" => 120147, "bot" => 8869, "bottom" => 8869, "bowtie" => 8904,
    "boxDL" => 9559, "boxDR" => 9556, "boxDl" => 9558, "boxDr" => 9555, "boxH" => 9552,
    "boxHD" => 9574, "boxHU" => 9577, "boxHd" => 9572, "boxHu" => 9575, "boxUL" => 9565,
    "boxUR" => 9562, "boxUl" => 9564, "boxUr" => 9561, "boxV" => 9553, "boxVH" => 9580,
    "boxVL" => 9571, "boxVR" => 9568, "boxVh" => 9579, "boxVl" => 9570, "boxVr" => 9567,
    "boxbox" => 10697, "boxdL" => 9557, "boxdR" => 9554, "boxdl" => 9488, "boxdr" => 9484,
    "boxh" => 9472, "boxhD" => 9573, "boxhU" => 9576, "boxhd" => 9516, "boxhu" => 9524,
    "boxminus" => 8863, "boxplus" => 8862, "boxtimes" => 8864, "boxuL" => 9563,
    "boxuR" => 9560, "boxul" => 9496, "boxur" => 9492, "boxv" => 9474, "boxvH" => 9578,
    "boxvL" => 9569, "boxvR" => 9566, "boxvh" => 9532, "boxvl" => 9508, "boxvr" => 9500,
    "bprime" => 8245, "breve" => 728, "brvbar" => 166, "bscr" => 119991, "bsemi" => 8271,
    "bsim" => 8765, "bsime" => 8909, "bsol" => 92, "bsolb" => 10693, "bsolhsub" => 10184,
    "bull" => 8226, "bullet" => 8226, "bump" => 8782, "bumpE" => 10926, "bumpe" => 8783,
    "bumpeq" => 8783, "cacute" => 263, "cap" => 8745, "capand" => 10820,
    "capbrcup" => 10825, "capcap" => 10827, "capcup" => 10823, "capdot" => 10816,
    "caps" => [8745, 65024], "caret" => 8257, "caron" => 711, "ccaps" => 10829,
    "ccaron" => 269, "ccedil" => 231, "ccirc" => 265, "ccups" => 10828, "ccupssm" => 10832,
    "cdot" => 267, "cedil" => 184, "cemptyv" => 10674, "cent" => 162, "centerdot" => 183,
    "cfr" => 120096, "chcy" => 1095, "check" => 10003, "checkmark" => 10003, "chi" => 967,
    "cir" => 9675, "cirE" => 10691, "circ" => 710, "circeq" => 8791,
    "circlearrowleft" => 8634, "circlearrowright" => 8635, "circledR" => 174,
    "circledS" => 9416, "circledast" => 8859, "circledcirc" => 8858, "circleddash" => 8861,
    "cire" => 8791, "cirfnint" => 10768, "cirmid" => 10991, "cirscir" => 10690,
    "clubs" => 9827, "clubsuit" => 9827, "colon" => 58, "colone" => 8788,
    "coloneq" => 8788, "comma" => 44, "commat" => 64, "comp" => 8705, "compfn" => 8728,
    "complement" => 8705, "complexes" => 8450, "cong" => 8773, "congdot" => 10861,
    "conint" => 8750, "copf" => 120148, "coprod" => 8720, "copy" => 169, "copysr" => 8471,
    "crarr" => 8629, "cross" => 10007, "cscr" => 119992, "csub" => 10959, "csube" => 10961,
    "csup" => 10960, "csupe" => 10962, "ctdot" => 8943, "cudarrl" => 10552,
    "cudarrr" => 10549, "cuepr" => 8926, "cuesc" => 8927, "cularr" => 8630,
    "cularrp" => 10557, "cup" => 8746, "cupbrcap" => 10824, "cupcap" => 10822,
    "cupcup" => 10826, "cupdot" => 8845, "cupor" => 10821, "cups" => [8746, 65024],
    "curarr" => 8631, "curarrm" => 10556, "curlyeqprec" => 8926, "curlyeqsucc" => 8927,
    "curlyvee" => 8910, "curlywedge" => 8911, "curren" => 164, "curvearrowleft" => 8630,
    "curvearrowright" => 8631, "cuvee" => 8910, "cuwed" => 8911, "cwconint" => 8754,
    "cwint" => 8753, "cylcty" => 9005, "dArr" => 8659, "dHar" => 10597, "dagger" => 8224,
    "daleth" => 8504, "darr" => 8595, "dash" => 8208, "dashv" => 8867, "dbkarow" => 10511,
    "dblac" => 733, "dcaron" => 271, "dcy" => 1076, "dd" => 8518, "ddagger" => 8225,
    "ddarr" => 8650, "ddotseq" => 10871, "deg" => 176, "delta" => 948, "demptyv" => 10673,
    "dfisht" => 10623, "dfr" => 120097, "dharl" => 8643, "dharr" => 8642, "diam" => 8900,
    "diamond" => 8900, "diamondsuit" => 9830, "diams" => 9830, "die" => 168,
    "digamma" => 989, "disin" => 8946, "div" => 247, "divide" => 247,
    "divideontimes" => 8903, "divonx" => 8903, "djcy" => 1106, "dlcorn" => 8990,
    "dlcrop" => 8973, "dollar" => 36, "dopf" => 120149, "dot" => 729, "doteq" => 8784,
    "doteqdot" => 8785, "dotminus" => 8760, "dotplus" => 8724, "dotsquare" => 8865,
    "doublebarwedge" => 8966, "downarrow" => 8595, "downdownarrows" => 8650,
    "downharpoonleft" => 8643, "downharpoonright" => 8642, "drbkarow" => 10512,
    "drcorn" => 8991, "drcrop" => 8972, "dscr" => 119993, "dscy" => 1109, "dsol" => 10742,
    "dstrok" => 273, "dtdot" => 8945, "dtri" => 9663, "dtrif" => 9662, "duarr" => 8693,
    "duhar" => 10607, "dwangle" => 10662, "dzcy" => 1119, "dzigrarr" => 10239,
    "eDDot" => 10871, "eDot" => 8785, "eacute" => 233, "easter" => 10862, "ecaron" => 283,
    "ecir" => 8790, "ecirc" => 234, "ecolon" => 8789, "ecy" => 1101, "edot" => 279,
    "ee" => 8519, "efDot" => 8786, "efr" => 120098, "eg" => 10906, "egrave" => 232,
    "egs" => 10902, "egsdot" => 10904, "el" => 10905, "elinters" => 9191, "ell" => 8467,
    "els" => 10901, "elsdot" => 10903, "emacr" => 275, "empty" => 8709, "emptyset" => 8709,
    "emptyv" => 8709, "emsp" => 8195, "emsp13" => 8196, "emsp14" => 8197, "eng" => 331,
    "ensp" => 8194, "eogon" => 281, "eopf" => 120150, "epar" => 8917, "eparsl" => 10723,
    "eplus" => 10865, "epsi" => 949, "epsilon" => 949, "epsiv" => 1013, "eqcirc" => 8790,
    "eqcolon" => 8789, "eqsim" => 8770, "eqslantgtr" => 10902, "eqslantless" => 10901,
    "equals" => 61, "equest" => 8799, "equiv" => 8801, "equivDD" => 10872,
    "eqvparsl" => 10725, "erDot" => 8787, "erarr" => 10609, "escr" => 8495,
    "esdot" => 8784, "esim" => 8770, "eta" => 951, "eth" => 240, "euml" => 235,
    "euro" => 8364, "excl" => 33, "exist" => 8707, "expectation" => 8496,
    "exponentiale" => 8519, "fallingdotseq" => 8786, "fcy" => 1092, "female" => 9792,
    "ffilig" => 64259, "fflig" => 64256, "ffllig" => 64260, "ffr" => 120099,
    "filig" => 64257, "fjlig" => [102, 106], "flat" => 9837, "fllig" => 64258,
    "fltns" => 9649, "fnof" => 402, "fopf" => 120151, "forall" => 8704, "fork" => 8916,
    "forkv" => 10969, "fpartint" => 10765, "frac12" => 189, "frac13" => 8531,
    "frac14" => 188, "frac15" => 8533, "frac16" => 8537, "frac18" => 8539,
    "frac23" => 8532, "frac25" => 8534, "frac34" => 190, "frac35" => 8535,
    "frac38" => 8540, "frac45" => 8536, "frac56" => 8538, "frac58" => 8541,
    "frac78" => 8542, "frasl" => 8260, "frown" => 8994, "fscr" => 119995, "gE" => 8807,
    "gEl" => 10892, "gacute" => 501, "gamma" => 947, "gammad" => 989, "gap" => 10886,
    "gbreve" => 287, "gcirc" => 285, "gcy" => 1075, "gdot" => 289, "ge" => 8805,
    "gel" => 8923, "geq" => 8805, "geqq" => 8807, "geqslant" => 10878, "ges" => 10878,
    "gescc" => 10921, "gesdot" => 10880, "gesdoto" => 10882, "gesdotol" => 10884,
    "gesl" => [8923, 65024], "gesles" => 10900, "gfr" => 120100, "gg" => 8811,
    "ggg" => 8921, "gimel" => 8503, "gjcy" => 1107, "gl" => 8823, "glE" => 10898,
    "gla" => 10917, "glj" => 10916, "gnE" => 8809, "gnap" => 10890, "gnapprox" => 10890,
    "gne" => 10888, "gneq" => 10888, "gneqq" => 8809, "gnsim" => 8935, "gopf" => 120152,
    "grave" => 96, "gscr" => 8458, "gsim" => 8819, "gsime" => 10894, "gsiml" => 10896,
    "gt" => 62, "gtcc" => 10919, "gtcir" => 10874, "gtdot" => 8919, "gtlPar" => 10645,
    "gtquest" => 10876, "gtrapprox" => 10886, "gtrarr" => 10616, "gtrdot" => 8919,
    "gtreqless" => 8923, "gtreqqless" => 10892, "gtrless" => 8823, "gtrsim" => 8819,
    "gvertneqq" => [8809, 65024], "gvnE" => [8809, 65024], "hArr" => 8660,
    "hairsp" => 8202, "half" => 189, "hamilt" => 8459, "hardcy" => 1098, "harr" => 8596,
    "harrcir" => 10568, "harrw" => 8621, "hbar" => 8463, "hcirc" => 293, "hearts" => 9829,
    "heartsuit" => 9829, "hellip" => 8230, "hercon" => 8889, "hfr" => 120101,
    "hksearow" => 10533, "hkswarow" => 10534, "hoarr" => 8703, "homtht" => 8763,
    "hookleftarrow" => 8617, "hookrightarrow" => 8618, "hopf" => 120153, "horbar" => 8213,
    "hscr" => 119997, "hslash" => 8463, "hstrok" => 295, "hybull" => 8259,
    "hyphen" => 8208, "iacute" => 237, "ic" => 8291, "icirc" => 238, "icy" => 1080,
    "iecy" => 1077, "iexcl" => 161, "iff" => 8660, "ifr" => 120102, "igrave" => 236,
    "ii" => 8520, "iiiint" => 10764, "iiint" => 8749, "iinfin" => 10716, "iiota" => 8489,
    "ijlig" => 307, "imacr" => 299, "image" => 8465, "imagline" => 8464,
    "imagpart" => 8465, "imath" => 305, "imof" => 8887, "imped" => 437, "in" => 8712,
    "incare" => 8453, "infin" => 8734, "infintie" => 10717, "inodot" => 305, "int" => 8747,
    "intcal" => 8890, "integers" => 8484, "intercal" => 8890, "intlarhk" => 10775,
    "intprod" => 10812, "iocy" => 1105, "iogon" => 303, "iopf" => 120154, "iota" => 953,
    "iprod" => 10812, "iquest" => 191, "iscr" => 119998, "isin" => 8712, "isinE" => 8953,
    "isindot" => 8949, "isins" => 8948, "isinsv" => 8947, "isinv" => 8712, "it" => 8290,
    "itilde" => 297, "iukcy" => 1110, "iuml" => 239, "jcirc" => 309, "jcy" => 1081,
    "jfr" => 120103, "jmath" => 567, "jopf" => 120155, "jscr" => 119999, "jsercy" => 1112,
    "jukcy" => 1108, "kappa" => 954, "kappav" => 1008, "kcedil" => 311, "kcy" => 1082,
    "kfr" => 120104, "kgreen" => 312, "khcy" => 1093, "kjcy" => 1116, "kopf" => 120156,
    "kscr" => 120000, "lAarr" => 8666, "lArr" => 8656, "lAtail" => 10523, "lBarr" => 10510,
    "lE" => 8806, "lEg" => 10891, "lHar" => 10594, "lacute" => 314, "laemptyv" => 10676,
    "lagran" => 8466, "lambda" => 955, "lang" => 10216, "langd" => 10641,
    "langle" => 10216, "lap" => 10885, "laquo" => 171, "larr" => 8592, "larrb" => 8676,
    "larrbfs" => 10527, "larrfs" => 10525, "larrhk" => 8617, "larrlp" => 8619,
    "larrpl" => 10553, "larrsim" => 10611, "larrtl" => 8610, "lat" => 10923,
    "latail" => 10521, "late" => 10925, "lates" => [10925, 65024], "lbarr" => 10508,
    "lbbrk" => 10098, "lbrace" => 123, "lbrack" => 91, "lbrke" => 10635,
    "lbrksld" => 10639, "lbrkslu" => 10637, "lcaron" => 318, "lcedil" => 316,
    "lceil" => 8968, "lcub" => 123, "lcy" => 1083, "ldca" => 10550, "ldquo" => 8220,
    "ldquor" => 8222, "ldrdhar" => 10599, "ldrushar" => 10571, "ldsh" => 8626,
    "le" => 8804, "leftarrow" => 8592, "leftarrowtail" => 8610, "leftharpoondown" => 8637,
    "leftharpoonup" => 8636, "leftleftarrows" => 8647, "leftrightarrow" => 8596,
    "leftrightarrows" => 8646, "leftrightharpoons" => 8651, "leftrightsquigarrow" => 8621,
    "leftthreetimes" => 8907, "leg" => 8922, "leq" => 8804, "leqq" => 8806,
    "leqslant" => 10877, "les" => 10877, "lescc" => 10920, "lesdot" => 10879,
    "lesdoto" => 10881, "lesdotor" => 10883, "lesg" => [8922, 65024], "lesges" => 10899,
    "lessapprox" => 10885, "lessdot" => 8918, "lesseqgtr" => 8922, "lesseqqgtr" => 10891,
    "lessgtr" => 8822, "lesssim" => 8818, "lfisht" => 10620, "lfloor" => 8970,
    "lfr" => 120105, "lg" => 8822, "lgE" => 10897, "lhard" => 8637, "lharu" => 8636,
    "lharul" => 10602, "lhblk" => 9604, "ljcy" => 1113, "ll" => 8810, "llarr" => 8647,
    "llcorner" => 8990, "llhard" => 10603, "lltri" => 9722, "lmidot" => 320,
    "lmoust" => 9136, "lmoustache" => 9136, "lnE" => 8808, "lnap" => 10889,
    "lnapprox" => 10889, "lne" => 10887, "lneq" => 10887, "lneqq" => 8808, "lnsim" => 8934,
    "loang" => 10220, "loarr" => 8701, "lobrk" => 10214, "longleftarrow" => 10229,
    "longleftrightarrow" => 10231, "longmapsto" => 10236, "longrightarrow" => 10230,
    "looparrowleft" => 8619, "looparrowright" => 8620, "lopar" => 10629, "lopf" => 120157,
    "loplus" => 10797, "lotimes" => 10804, "lowast" => 8727, "lowbar" => 95, "loz" => 9674,
    "lozenge" => 9674, "lozf" => 10731, "lpar" => 40, "lparlt" => 10643, "lrarr" => 8646,
    "lrcorner" => 8991, "lrhar" => 8651, "lrhard" => 10605, "lrm" => 8206, "lrtri" => 8895,
    "lsaquo" => 8249, "lscr" => 120001, "lsh" => 8624, "lsim" => 8818, "lsime" => 10893,
    "lsimg" => 10895, "lsqb" => 91, "lsquo" => 8216, "lsquor" => 8218, "lstrok" => 322,
    "lt" => 60, "ltcc" => 10918, "ltcir" => 10873, "ltdot" => 8918, "lthree" => 8907,
    "ltimes" => 8905, "ltlarr" => 10614, "ltquest" => 10875, "ltrPar" => 10646,
    "ltri" => 9667, "ltrie" => 8884, "ltrif" => 9666, "lurdshar" => 10570,
    "luruhar" => 10598, "lvertneqq" => [8808, 65024], "lvnE" => [8808, 65024],
    "mDDot" => 8762, "macr" => 175, "male" => 9794, "malt" => 10016, "maltese" => 10016,
    "map" => 8614, "mapsto" => 8614, "mapstodown" => 8615, "mapstoleft" => 8612,
    "mapstoup" => 8613, "marker" => 9646, "mcomma" => 10793, "mcy" => 1084,
    "mdash" => 8212, "measuredangle" => 8737, "mfr" => 120106, "mho" => 8487,
    "micro" => 181, "mid" => 8739, "midast" => 42, "midcir" => 10992, "middot" => 183,
    "minus" => 8722, "minusb" => 8863, "minusd" => 8760, "minusdu" => 10794,
    "mlcp" => 10971, "mldr" => 8230, "mnplus" => 8723, "models" => 8871, "mopf" => 120158,
    "mp" => 8723, "mscr" => 120002, "mstpos" => 8766, "mu" => 956, "multimap" => 8888,
    "mumap" => 8888, "nGg" => [8921, 824], "nGt" => [8811, 8402], "nGtv" => [8811, 824],
    "nLeftarrow" => 8653, "nLeftrightarrow" => 8654, "nLl" => [8920, 824],
    "nLt" => [8810, 8402], "nLtv" => [8810, 824], "nRightarrow" => 8655, "nVDash" => 8879,
    "nVdash" => 8878, "nabla" => 8711, "nacute" => 324, "nang" => [8736, 8402],
    "nap" => 8777, "napE" => [10864, 824], "napid" => [8779, 824], "napos" => 329,
    "napprox" => 8777, "natur" => 9838, "natural" => 9838, "naturals" => 8469,
    "nbsp" => 160, "nbump" => [8782, 824], "nbumpe" => [8783, 824], "ncap" => 10819,
    "ncaron" => 328, "ncedil" => 326, "ncong" => 8775, "ncongdot" => [10861, 824],
    "ncup" => 10818, "ncy" => 1085, "ndash" => 8211, "ne" => 8800, "neArr" => 8663,
    "nearhk" => 10532, "nearr" => 8599, "nearrow" => 8599, "nedot" => [8784, 824],
    "nequiv" => 8802, "nesear" => 10536, "nesim" => [8770, 824], "nexist" => 8708,
    "nexists" => 8708, "nfr" => 120107, "ngE" => [8807, 824], "nge" => 8817,
    "ngeq" => 8817, "ngeqq" => [8807, 824], "ngeqslant" => [10878, 824],
    "nges" => [10878, 824], "ngsim" => 8821, "ngt" => 8815, "ngtr" => 8815,
    "nhArr" => 8654, "nharr" => 8622, "nhpar" => 10994, "ni" => 8715, "nis" => 8956,
    "nisd" => 8954, "niv" => 8715, "njcy" => 1114, "nlArr" => 8653, "nlE" => [8806, 824],
    "nlarr" => 8602, "nldr" => 8229, "nle" => 8816, "nleftarrow" => 8602,
    "nleftrightarrow" => 8622, "nleq" => 8816, "nleqq" => [8806, 824],
    "nleqslant" => [10877, 824], "nles" => [10877, 824], "nless" => 8814, "nlsim" => 8820,
    "nlt" => 8814, "nltri" => 8938, "nltrie" => 8940, "nmid" => 8740, "nopf" => 120159,
    "not" => 172, "notin" => 8713, "notinE" => [8953, 824], "notindot" => [8949, 824],
    "notinva" => 8713, "notinvb" => 8951, "notinvc" => 8950, "notni" => 8716,
    "notniva" => 8716, "notnivb" => 8958, "notnivc" => 8957, "npar" => 8742,
    "nparallel" => 8742, "nparsl" => [11005, 8421], "npart" => [8706, 824],
    "npolint" => 10772, "npr" => 8832, "nprcue" => 8928, "npre" => [10927, 824],
    "nprec" => 8832, "npreceq" => [10927, 824], "nrArr" => 8655, "nrarr" => 8603,
    "nrarrc" => [10547, 824], "nrarrw" => [8605, 824], "nrightarrow" => 8603,
    "nrtri" => 8939, "nrtrie" => 8941, "nsc" => 8833, "nsccue" => 8929,
    "nsce" => [10928, 824], "nscr" => 120003, "nshortmid" => 8740,
    "nshortparallel" => 8742, "nsim" => 8769, "nsime" => 8772, "nsimeq" => 8772,
    "nsmid" => 8740, "nspar" => 8742, "nsqsube" => 8930, "nsqsupe" => 8931, "nsub" => 8836,
    "nsubE" => [10949, 824], "nsube" => 8840, "nsubset" => [8834, 8402],
    "nsubseteq" => 8840, "nsubseteqq" => [10949, 824], "nsucc" => 8833,
    "nsucceq" => [10928, 824], "nsup" => 8837, "nsupE" => [10950, 824], "nsupe" => 8841,
    "nsupset" => [8835, 8402], "nsupseteq" => 8841, "nsupseteqq" => [10950, 824],
    "ntgl" => 8825, "ntilde" => 241, "ntlg" => 8824, "ntriangleleft" => 8938,
    "ntrianglelefteq" => 8940, "ntriangleright" => 8939, "ntrianglerighteq" => 8941,
    "nu" => 957, "num" => 35, "numero" => 8470, "numsp" => 8199, "nvDash" => 8877,
    "nvHarr" => 10500, "nvap" => [8781, 8402], "nvdash" => 8876, "nvge" => [8805, 8402],
    "nvgt" => [62, 8402], "nvinfin" => 10718, "nvlArr" => 10498, "nvle" => [8804, 8402],
    "nvlt" => [60, 8402], "nvltrie" => [8884, 8402], "nvrArr" => 10499,
    "nvrtrie" => [8885, 8402], "nvsim" => [8764, 8402], "nwArr" => 8662, "nwarhk" => 10531,
    "nwarr" => 8598, "nwarrow" => 8598, "nwnear" => 10535, "oS" => 9416, "oacute" => 243,
    "oast" => 8859, "ocir" => 8858, "ocirc" => 244, "ocy" => 1086, "odash" => 8861,
    "odblac" => 337, "odiv" => 10808, "odot" => 8857, "odsold" => 10684, "oelig" => 339,
    "ofcir" => 10687, "ofr" => 120108, "ogon" => 731, "ograve" => 242, "ogt" => 10689,
    "ohbar" => 10677, "ohm" => 937, "oint" => 8750, "olarr" => 8634, "olcir" => 10686,
    "olcross" => 10683, "oline" => 8254, "olt" => 10688, "omacr" => 333, "omega" => 969,
    "omicron" => 959, "omid" => 10678, "ominus" => 8854, "oopf" => 120160, "opar" => 10679,
    "operp" => 10681, "oplus" => 8853, "or" => 8744, "orarr" => 8635, "ord" => 10845,
    "order" => 8500, "orderof" => 8500, "ordf" => 170, "ordm" => 186, "origof" => 8886,
    "oror" => 10838, "orslope" => 10839, "orv" => 10843, "oscr" => 8500, "oslash" => 248,
    "osol" => 8856, "otilde" => 245, "otimes" => 8855, "otimesas" => 10806, "ouml" => 246,
    "ovbar" => 9021, "par" => 8741, "para" => 182, "parallel" => 8741, "parsim" => 10995,
    "parsl" => 11005, "part" => 8706, "pcy" => 1087, "percnt" => 37, "period" => 46,
    "permil" => 8240, "perp" => 8869, "pertenk" => 8241, "pfr" => 120109, "phi" => 966,
    "phiv" => 981, "phmmat" => 8499, "phone" => 9742, "pi" => 960, "pitchfork" => 8916,
    "piv" => 982, "planck" => 8463, "planckh" => 8462, "plankv" => 8463, "plus" => 43,
    "plusacir" => 10787, "plusb" => 8862, "pluscir" => 10786, "plusdo" => 8724,
    "plusdu" => 10789, "pluse" => 10866, "plusmn" => 177, "plussim" => 10790,
    "plustwo" => 10791, "pm" => 177, "pointint" => 10773, "popf" => 120161, "pound" => 163,
    "pr" => 8826, "prE" => 10931, "prap" => 10935, "prcue" => 8828, "pre" => 10927,
    "prec" => 8826, "precapprox" => 10935, "preccurlyeq" => 8828, "preceq" => 10927,
    "precnapprox" => 10937, "precneqq" => 10933, "precnsim" => 8936, "precsim" => 8830,
    "prime" => 8242, "primes" => 8473, "prnE" => 10933, "prnap" => 10937, "prnsim" => 8936,
    "prod" => 8719, "profalar" => 9006, "profline" => 8978, "profsurf" => 8979,
    "prop" => 8733, "propto" => 8733, "prsim" => 8830, "prurel" => 8880, "pscr" => 120005,
    "psi" => 968, "puncsp" => 8200, "qfr" => 120110, "qint" => 10764, "qopf" => 120162,
    "qprime" => 8279, "qscr" => 120006, "quaternions" => 8461, "quatint" => 10774,
    "quest" => 63, "questeq" => 8799, "quot" => 34, "rAarr" => 8667, "rArr" => 8658,
    "rAtail" => 10524, "rBarr" => 10511, "rHar" => 10596, "race" => [8765, 817],
    "racute" => 341, "radic" => 8730, "raemptyv" => 10675, "rang" => 10217,
    "rangd" => 10642, "range" => 10661, "rangle" => 10217, "raquo" => 187, "rarr" => 8594,
    "rarrap" => 10613, "rarrb" => 8677, "rarrbfs" => 10528, "rarrc" => 10547,
    "rarrfs" => 10526, "rarrhk" => 8618, "rarrlp" => 8620, "rarrpl" => 10565,
    "rarrsim" => 10612, "rarrtl" => 8611, "rarrw" => 8605, "ratail" => 10522,
    "ratio" => 8758, "rationals" => 8474, "rbarr" => 10509, "rbbrk" => 10099,
    "rbrace" => 125, "rbrack" => 93, "rbrke" => 10636, "rbrksld" => 10638,
    "rbrkslu" => 10640, "rcaron" => 345, "rcedil" => 343, "rceil" => 8969, "rcub" => 125,
    "rcy" => 1088, "rdca" => 10551, "rdldhar" => 10601, "rdquo" => 8221, "rdquor" => 8221,
    "rdsh" => 8627, "real" => 8476, "realine" => 8475, "realpart" => 8476, "reals" => 8477,
    "rect" => 9645, "reg" => 174, "rfisht" => 10621, "rfloor" => 8971, "rfr" => 120111,
    "rhard" => 8641, "rharu" => 8640, "rharul" => 10604, "rho" => 961, "rhov" => 1009,
    "rightarrow" => 8594, "rightarrowtail" => 8611, "rightharpoondown" => 8641,
    "rightharpoonup" => 8640, "rightleftarrows" => 8644, "rightleftharpoons" => 8652,
    "rightrightarrows" => 8649, "rightsquigarrow" => 8605, "rightthreetimes" => 8908,
    "ring" => 730, "risingdotseq" => 8787, "rlarr" => 8644, "rlhar" => 8652, "rlm" => 8207,
    "rmoust" => 9137, "rmoustache" => 9137, "rnmid" => 10990, "roang" => 10221,
    "roarr" => 8702, "robrk" => 10215, "ropar" => 10630, "ropf" => 120163,
    "roplus" => 10798, "rotimes" => 10805, "rpar" => 41, "rpargt" => 10644,
    "rppolint" => 10770, "rrarr" => 8649, "rsaquo" => 8250, "rscr" => 120007,
    "rsh" => 8625, "rsqb" => 93, "rsquo" => 8217, "rsquor" => 8217, "rthree" => 8908,
    "rtimes" => 8906, "rtri" => 9657, "rtrie" => 8885, "rtrif" => 9656,
    "rtriltri" => 10702, "ruluhar" => 10600, "rx" => 8478, "sacute" => 347,
    "sbquo" => 8218, "sc" => 8827, "scE" => 10932, "scap" => 10936, "scaron" => 353,
    "sccue" => 8829, "sce" => 10928, "scedil" => 351, "scirc" => 349, "scnE" => 10934,
    "scnap" => 10938, "scnsim" => 8937, "scpolint" => 10771, "scsim" => 8831,
    "scy" => 1089, "sdot" => 8901, "sdotb" => 8865, "sdote" => 10854, "seArr" => 8664,
    "searhk" => 10533, "searr" => 8600, "searrow" => 8600, "sect" => 167, "semi" => 59,
    "seswar" => 10537, "setminus" => 8726, "setmn" => 8726, "sext" => 10038,
    "sfr" => 120112, "sfrown" => 8994, "sharp" => 9839, "shchcy" => 1097, "shcy" => 1096,
    "shortmid" => 8739, "shortparallel" => 8741, "shy" => 173, "sigma" => 963,
    "sigmaf" => 962, "sigmav" => 962, "sim" => 8764, "simdot" => 10858, "sime" => 8771,
    "simeq" => 8771, "simg" => 10910, "simgE" => 10912, "siml" => 10909, "simlE" => 10911,
    "simne" => 8774, "simplus" => 10788, "simrarr" => 10610, "slarr" => 8592,
    "smallsetminus" => 8726, "smashp" => 10803, "smeparsl" => 10724, "smid" => 8739,
    "smile" => 8995, "smt" => 10922, "smte" => 10924, "smtes" => [10924, 65024],
    "softcy" => 1100, "sol" => 47, "solb" => 10692, "solbar" => 9023, "sopf" => 120164,
    "spades" => 9824, "spadesuit" => 9824, "spar" => 8741, "sqcap" => 8851,
    "sqcaps" => [8851, 65024], "sqcup" => 8852, "sqcups" => [8852, 65024], "sqsub" => 8847,
    "sqsube" => 8849, "sqsubset" => 8847, "sqsubseteq" => 8849, "sqsup" => 8848,
    "sqsupe" => 8850, "sqsupset" => 8848, "sqsupseteq" => 8850, "squ" => 9633,
    "square" => 9633, "squarf" => 9642, "squf" => 9642, "srarr" => 8594, "sscr" => 120008,
    "ssetmn" => 8726, "ssmile" => 8995, "sstarf" => 8902, "star" => 9734, "starf" => 9733,
    "straightepsilon" => 1013, "straightphi" => 981, "strns" => 175, "sub" => 8834,
    "subE" => 10949, "subdot" => 10941, "sube" => 8838, "subedot" => 10947,
    "submult" => 10945, "subnE" => 10955, "subne" => 8842, "subplus" => 10943,
    "subrarr" => 10617, "subset" => 8834, "subseteq" => 8838, "subseteqq" => 10949,
    "subsetneq" => 8842, "subsetneqq" => 10955, "subsim" => 10951, "subsub" => 10965,
    "subsup" => 10963, "succ" => 8827, "succapprox" => 10936, "succcurlyeq" => 8829,
    "succeq" => 10928, "succnapprox" => 10938, "succneqq" => 10934, "succnsim" => 8937,
    "succsim" => 8831, "sum" => 8721, "sung" => 9834, "sup" => 8835, "sup1" => 185,
    "sup2" => 178, "sup3" => 179, "supE" => 10950, "supdot" => 10942, "supdsub" => 10968,
    "supe" => 8839, "supedot" => 10948, "suphsol" => 10185, "suphsub" => 10967,
    "suplarr" => 10619, "supmult" => 10946, "supnE" => 10956, "supne" => 8843,
    "supplus" => 10944, "supset" => 8835, "supseteq" => 8839, "supseteqq" => 10950,
    "supsetneq" => 8843, "supsetneqq" => 10956, "supsim" => 10952, "supsub" => 10964,
    "supsup" => 10966, "swArr" => 8665, "swarhk" => 10534, "swarr" => 8601,
    "swarrow" => 8601, "swnwar" => 10538, "szlig" => 223, "target" => 8982, "tau" => 964,
    "tbrk" => 9140, "tcaron" => 357, "tcedil" => 355, "tcy" => 1090, "tdot" => 8411,
    "telrec" => 8981, "tfr" => 120113, "there4" => 8756, "therefore" => 8756,
    "theta" => 952, "thetasym" => 977, "thetav" => 977, "thickapprox" => 8776,
    "thicksim" => 8764, "thinsp" => 8201, "thkap" => 8776, "thksim" => 8764,
    "thorn" => 254, "tilde" => 732, "times" => 215, "timesb" => 8864, "timesbar" => 10801,
    "timesd" => 10800, "tint" => 8749, "toea" => 10536, "top" => 8868, "topbot" => 9014,
    "topcir" => 10993, "topf" => 120165, "topfork" => 10970, "tosa" => 10537,
    "tprime" => 8244, "trade" => 8482, "triangle" => 9653, "triangledown" => 9663,
    "triangleleft" => 9667, "trianglelefteq" => 8884, "triangleq" => 8796,
    "triangleright" => 9657, "trianglerighteq" => 8885, "tridot" => 9708, "trie" => 8796,
    "triminus" => 10810, "triplus" => 10809, "trisb" => 10701, "tritime" => 10811,
    "trpezium" => 9186, "tscr" => 120009, "tscy" => 1094, "tshcy" => 1115, "tstrok" => 359,
    "twixt" => 8812, "twoheadleftarrow" => 8606, "twoheadrightarrow" => 8608,
    "uArr" => 8657, "uHar" => 10595, "uacute" => 250, "uarr" => 8593, "ubrcy" => 1118,
    "ubreve" => 365, "ucirc" => 251, "ucy" => 1091, "udarr" => 8645, "udblac" => 369,
    "udhar" => 10606, "ufisht" => 10622, "ufr" => 120114, "ugrave" => 249, "uharl" => 8639,
    "uharr" => 8638, "uhblk" => 9600, "ulcorn" => 8988, "ulcorner" => 8988,
    "ulcrop" => 8975, "ultri" => 9720, "umacr" => 363, "uml" => 168, "uogon" => 371,
    "uopf" => 120166, "uparrow" => 8593, "updownarrow" => 8597, "upharpoonleft" => 8639,
    "upharpoonright" => 8638, "uplus" => 8846, "upsi" => 965, "upsih" => 978,
    "upsilon" => 965, "upuparrows" => 8648, "urcorn" => 8989, "urcorner" => 8989,
    "urcrop" => 8974, "uring" => 367, "urtri" => 9721, "uscr" => 120010, "utdot" => 8944,
    "utilde" => 361, "utri" => 9653, "utrif" => 9652, "uuarr" => 8648, "uuml" => 252,
    "uwangle" => 10663, "vArr" => 8661, "vBar" => 10984, "vBarv" => 10985, "vDash" => 8872,
    "vangrt" => 10652, "varepsilon" => 1013, "varkappa" => 1008, "varnothing" => 8709,
    "varphi" => 981, "varpi" => 982, "varpropto" => 8733, "varr" => 8597, "varrho" => 1009,
    "varsigma" => 962, "varsubsetneq" => [8842, 65024], "varsubsetneqq" => [10955, 65024],
    "varsupsetneq" => [8843, 65024], "varsupsetneqq" => [10956, 65024], "vartheta" => 977,
    "vartriangleleft" => 8882, "vartriangleright" => 8883, "vcy" => 1074, "vdash" => 8866,
    "vee" => 8744, "veebar" => 8891, "veeeq" => 8794, "vellip" => 8942, "verbar" => 124,
    "vert" => 124, "vfr" => 120115, "vltri" => 8882, "vnsub" => [8834, 8402],
    "vnsup" => [8835, 8402], "vopf" => 120167, "vprop" => 8733, "vrtri" => 8883,
    "vscr" => 120011, "vsubnE" => [10955, 65024], "vsubne" => [8842, 65024],
    "vsupnE" => [10956, 65024], "vsupne" => [8843, 65024], "vzigzag" => 10650,
    "wcirc" => 373, "wedbar" => 10847, "wedge" => 8743, "wedgeq" => 8793, "weierp" => 8472,
    "wfr" => 120116, "wopf" => 120168, "wp" => 8472, "wr" => 8768, "wreath" => 8768,
    "wscr" => 120012, "xcap" => 8898, "xcirc" => 9711, "xcup" => 8899, "xdtri" => 9661,
    "xfr" => 120117, "xhArr" => 10234, "xharr" => 10231, "xi" => 958, "xlArr" => 10232,
    "xlarr" => 10229, "xmap" => 10236, "xnis" => 8955, "xodot" => 10752, "xopf" => 120169,
    "xoplus" => 10753, "xotime" => 10754, "xrArr" => 10233, "xrarr" => 10230,
    "xscr" => 120013, "xsqcup" => 10758, "xuplus" => 10756, "xutri" => 9651,
    "xvee" => 8897, "xwedge" => 8896, "yacute" => 253, "yacy" => 1103, "ycirc" => 375,
    "ycy" => 1099, "yen" => 165, "yfr" => 120118, "yicy" => 1111, "yopf" => 120170,
    "yscr" => 120014, "yucy" => 1102, "yuml" => 255, "zacute" => 378, "zcaron" => 382,
    "zcy" => 1079, "zdot" => 380, "zeetrf" => 8488, "zeta" => 950, "zfr" => 120119,
    "zhcy" => 1078, "zigrarr" => 8669, "zopf" => 120171, "zscr" => 120015, "zwj" => 8205,
    "zwnj" => 8204
    ];
}
function addslashes($string) {
    return strtr($string, ["\\" => "\\\\", "'" => "\\'", "\"" => "\\\"", "\x00" => "\\0"]);
}
// A backslash is dropped and the byte after it kept (\0 is a NUL byte).
function stripslashes($string) {
    $string = (string)$string;
    $out = "";
    $at = 0;
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        if (substr($string, $i, 1) === "\\") {
            $out .= substr($string, $at, $i - $at);
            $at = $i + 1;
            if ($i + 1 < $n && substr($string, $i + 1, 1) === "0") {
                $out .= "\x00";
                $at = $i + 2;
            }
            $i++;
        }
    }
    return $out . substr($string, $at);
}
// A word is a run of ASCII letters, ' and - and the bytes $characters lists; the
// string's first byte cannot open one with ' or -, nor its last byte end one with
// -, unless $characters lists it (php_str_word_count).
function str_word_count(string $string, int $format = 0, ?string $characters = null) {
    $words = [];
    $p = 0;
    $e = strlen($string);
    $set = $characters === null ? [] : __mec_charset($characters);
    $first = substr($string, 0, 1);
    if (($first == "'" || $first == "-") && !isset($set[$first])) { $p = 1; }
    if ($e > 0 && substr($string, $e - 1, 1) == "-" && !isset($set["-"])) { $e--; }
    while ($p < $e) {
        $s = $p;
        while ($p < $e) {
            $c = substr($string, $p, 1);
            $o = ord($c);
            if (($o >= 65 && $o <= 90) || ($o >= 97 && $o <= 122) || $o == 39 || $o == 45 || isset($set[$c])) { $p++; } else { break; }
        }
        if ($p > $s) { $words[$s] = substr($string, $s, $p - $s); }
        $p++;
    }
    if ($format == 0) { return count($words); }
    if ($format == 1) { return array_values($words); }
    return $words;
}
// php_wordwrap's two loops: a one-byte break without cutting rewrites spaces in
// place; anything else copies line by line, cutting a long word when asked.
function wordwrap(string $string, int $width = 75, string $break = "\n", bool $cut_long_words = false) {
    $text = $string;
    if ($text === "") { return ""; }
    if ($break === "") { throw new ValueError("wordwrap(): Argument #3 (\$break) cannot be empty"); }
    if ($width == 0 && $cut_long_words) { throw new ValueError("wordwrap(): Argument #4 (\$cut_long_words) cannot be true when argument #2 (\$width) is 0"); }
    $n = strlen($text);
    $bl = strlen($break);
    $laststart = 0;
    $lastspace = 0;
    if ($bl == 1 && !$cut_long_words) {
        $at = [];
        for ($cur = 0; $cur < $n; $cur++) {
            $c = substr($text, $cur, 1);
            if ($c === $break) { $laststart = $lastspace = $cur + 1; }
            elseif ($c === " ") {
                if ($cur - $laststart >= $width) { $at[$cur] = true; $laststart = $cur + 1; }
                $lastspace = $cur;
            } elseif ($cur - $laststart >= $width && $laststart != $lastspace) {
                $at[$lastspace] = true;
                $laststart = $lastspace + 1;
            }
        }
        $out = "";
        $from = 0;
        foreach (array_keys($at) as $p) {
            $out .= substr($text, $from, $p - $from) . $break;
            $from = $p + 1;
        }
        return $out . substr($text, $from);
    }
    $out = "";
    for ($cur = 0; $cur < $n; $cur++) {
        $c = substr($text, $cur, 1);
        if ($c === substr($break, 0, 1) && $cur + $bl < $n && substr($text, $cur, $bl) === $break) {
            $out .= substr($text, $laststart, $cur - $laststart + $bl);
            $cur += $bl - 1;
            $laststart = $lastspace = $cur + 1;
        } elseif ($c === " ") {
            if ($cur - $laststart >= $width) {
                $out .= substr($text, $laststart, $cur - $laststart) . $break;
                $laststart = $cur + 1;
            }
            $lastspace = $cur;
        } elseif ($cur - $laststart >= $width && $cut_long_words && $laststart >= $lastspace) {
            $out .= substr($text, $laststart, $cur - $laststart) . $break;
            $laststart = $lastspace = $cur;
        } elseif ($cur - $laststart >= $width && $laststart < $lastspace) {
            $out .= substr($text, $laststart, $lastspace - $laststart) . $break;
            $laststart = $lastspace = $lastspace + 1;
        }
    }
    if ($laststart != $n) { $out .= substr($text, $laststart); }
    return $out;
}

// ----- regular expressions: preg_* --------------------------------------------
// php-src ext/pcre/php_pcre.c, over two natives on lib/regex.js's C (PCRE2)
// dialect (lib/php-preg.js): __mec_pregc(key) compiles, answering the group
// names by number ("" for an unnamed group) or PCRE2's error text, and
// __mec_pregx(key, subject, offset) runs one pcre2_match from a byte offset,
// answering the 2 * (groups + 1) byte offsets (-1 for a group that did not take
// part) and then the name of the latest (*MARK) on the matching path when there
// is one, [] for no match, or a PREG_*_ERROR code. An offset -o - 1 is the retry
// after an empty match at o: PCRE2_NOTEMPTY_ATSTART | PCRE2_ANCHORED there, and
// when that fails the search from the next character. A key is the modifiers
// that reach PCRE2, "/", and the pattern between the delimiters. The warnings
// PHP prints beside a false or null are not printed.

const PREG_PATTERN_ORDER = 1;
const PREG_SET_ORDER = 2;
const PREG_OFFSET_CAPTURE = 256;
const PREG_UNMATCHED_AS_NULL = 512;
const PREG_SPLIT_NO_EMPTY = 1;
const PREG_SPLIT_DELIM_CAPTURE = 2;
const PREG_SPLIT_OFFSET_CAPTURE = 4;
const PREG_GREP_INVERT = 1;
const PREG_NO_ERROR = 0;
const PREG_INTERNAL_ERROR = 1;
const PREG_BACKTRACK_LIMIT_ERROR = 2;
const PREG_RECURSION_LIMIT_ERROR = 3;
const PREG_BAD_UTF8_ERROR = 4;
const PREG_BAD_UTF8_OFFSET_ERROR = 5;
const PREG_JIT_STACKLIMIT_ERROR = 6;

// PCRE_G(error_code): answers it, and sets it first unless $code is null.
function __mec_pregerr($code = null) {
    static $last = 0;
    if ($code !== null) { $last = $code; }
    return $last;
}

// pcre_get_compiled_regex_cache_ex: [key, group names] for $regex, or false
// (PREG_INTERNAL_ERROR) for an empty pattern, a bad delimiter, a missing end
// delimiter, an unknown modifier or a pattern PCRE2 refuses. A construct the
// engine does not implement (script runs, non-atomic lookbehinds) is an Error.
function __mec_pregkey($fn, $regex) {
    static $cache = [];
    $regex = (string)$regex;
    if (array_key_exists($regex, $cache)) {
        $got = $cache[$regex];
        if ($got === false) { __mec_pregerr(PREG_INTERNAL_ERROR); }
        return $got;
    }
    $n = strlen($regex);
    $i = 0;
    while ($i < $n && strpos(" \t\n\v\f\r", $regex[$i]) !== false) { $i++; }
    $key = false;
    if ($i < $n) {
        $d = $regex[$i];
        $o = ord($d);
        $i++;
        if (!(($o >= 48 && $o <= 57) || ($o >= 65 && $o <= 90) || ($o >= 97 && $o <= 122) || $o == 92 || $o == 0)) {
            $bk = strpos("([{<", $d);
            $end = $bk === false ? $d : ")]}>"[$bk];
            $depth = 1;
            $j = $i;
            while ($j < $n) {
                $c = $regex[$j];
                if ($c === "\\" && $j + 1 < $n) { $j++; }
                elseif ($c === $end && ($bk === false || --$depth <= 0)) { break; }
                elseif ($bk !== false && $c === $d) { $depth++; }
                $j++;
            }
            if ($j < $n) {
                $mods = "";
                $ok = true;
                for ($k = $j + 1; $k < $n; $k++) {
                    $m = $regex[$k];
                    if (strpos("imnsxADUuJ", $m) !== false) { $mods .= $m; }
                    elseif (strpos("SXr \n\r", $m) === false) { $ok = false; break; }
                }
                if ($ok) { $key = $mods . "/" . substr($regex, $i, $j - $i); }
            }
        }
    }
    $got = false;
    if ($key !== false) {
        $names = __mec_pregc($key);
        if (is_array($names)) { $got = [$key, $names]; }
        elseif (str_starts_with($names, "unsupported")) { throw new Error($fn . "(): " . $names); }
    }
    $cache[$regex] = $got;
    if ($got === false) { __mec_pregerr(PREG_INTERNAL_ERROR); }
    return $got;
}

// A string parameter as zpp reads it: an array is a TypeError.
function __mec_pregstr($fn, $n, $name, $v) {
    if (is_array($v)) {
        throw new TypeError($fn . "(): Argument #" . $n . " (\$" . $name . ") must be of type string, array given");
    }
    return (string)$v;
}

// The start offset, negative from the end; null (PREG_INTERNAL_ERROR) past it.
function __mec_pregoff($subject, $offset) {
    $n = strlen($subject);
    if ($offset < 0) { $offset = -$offset <= $n ? $n + $offset : 0; }
    if ($offset > $n) {
        __mec_pregerr(PREG_INTERNAL_ERROR);
        return null;
    }
    return $offset;
}

// pcre2_match's count: one more than the highest group that took part.
function __mec_pregcount($o) {
    $count = intdiv(count($o), 2);
    while ($count > 1 && $o[2 * $count - 2] < 0) { $count--; }
    return $count;
}

// Group $i of a match, as populate_match_value / add_offset_pair make it.
function __mec_pregval($s, $o, $i, $flags) {
    $a = $o[2 * $i];
    if ($a < 0) { $v = ($flags & PREG_UNMATCHED_AS_NULL) ? null : ""; }
    else { $v = substr($s, $a, $o[2 * $i + 1] - $a); }
    return ($flags & PREG_OFFSET_CAPTURE) ? [$v, $a] : $v;
}

// populate_subpat_array: the groups up to the last that took part (all of them
// under PREG_UNMATCHED_AS_NULL), a named one under its name as well - where
// several share a name, the one that took part.
function __mec_pregsub($s, $o, $names, $flags) {
    $count = __mec_pregcount($o);
    $all = ($flags & PREG_UNMATCHED_AS_NULL) ? count($names) : $count;
    $r = [];
    for ($i = 0; $i < $all; $i++) {
        $v = $i < $count ? __mec_pregval($s, $o, $i, $flags) : __mec_pregval($s, [-1, -1], 0, $flags);
        $nm = $names[$i];
        if ($nm !== "" && (($i < $count && $o[2 * $i] >= 0) || !array_key_exists($nm, $r))) { $r[$nm] = $v; }
        $r[] = $v;
    }
    if (count($o) % 2 == 1) { $r["MARK"] = $o[count($o) - 1]; }
    return $r;
}

function preg_match($pattern, $subject, &$matches = null, $flags = 0, $offset = 0) {
    $pattern = __mec_pregstr("preg_match", 1, "pattern", $pattern);
    $subject = __mec_pregstr("preg_match", 2, "subject", $subject);
    $c = __mec_pregkey("preg_match", $pattern);
    if ($c === false) { return false; }
    $matches = [];
    if ($flags & 0xff) { throw new ValueError("preg_match(): Argument #4 (\$flags) must be a PREG_* constant"); }
    $offset = __mec_pregoff($subject, $offset);
    if ($offset === null) { return false; }
    __mec_pregerr(PREG_NO_ERROR);
    $o = __mec_pregx($c[0], $subject, $offset);
    if (is_int($o)) {
        __mec_pregerr($o);
        return false;
    }
    if (count($o) == 0) { return 0; }
    $matches = __mec_pregsub($subject, $o, $c[1], $flags);
    return 1;
}

function preg_match_all($pattern, $subject, &$matches = null, $flags = 0, $offset = 0) {
    $pattern = __mec_pregstr("preg_match_all", 1, "pattern", $pattern);
    $subject = __mec_pregstr("preg_match_all", 2, "subject", $subject);
    $c = __mec_pregkey("preg_match_all", $pattern);
    if ($c === false) { return false; }
    $matches = [];
    $order = PREG_PATTERN_ORDER;
    if ($flags & 0xff) {
        $order = $flags & 0xff;
        if ($order != PREG_PATTERN_ORDER && $order != PREG_SET_ORDER) {
            throw new ValueError("preg_match_all(): Argument #4 (\$flags) must be a PREG_* constant");
        }
    }
    $offset = __mec_pregoff($subject, $offset);
    if ($offset === null) { return false; }
    __mec_pregerr(PREG_NO_ERROR);
    $names = $c[1];
    $sets = [];
    foreach ($names as $nm) { $sets[] = []; }
    $matched = 0;
    $marks = null;
    $pos = $offset;
    while (true) {
        $o = __mec_pregx($c[0], $subject, $pos);
        if (is_int($o)) {
            __mec_pregerr($o);
            break;
        }
        if (count($o) == 0) { break; }
        if ($o[1] < $o[0]) { return false; }
        $matched++;
        if ($order == PREG_SET_ORDER) { $matches[] = __mec_pregsub($subject, $o, $names, $flags); }
        else {
            $count = __mec_pregcount($o);
            foreach ($names as $i => $nm) {
                $sets[$i][] = $i < $count ? __mec_pregval($subject, $o, $i, $flags) : __mec_pregval($subject, [-1, -1], 0, $flags);
            }
            // The (*MARK) names, by the number of the match.
            if (count($o) % 2 == 1) {
                if ($marks === null) { $marks = []; }
                $marks[$matched - 1] = $o[count($o) - 1];
            }
        }
        $pos = $o[1] == $o[0] ? -$o[1] - 1 : $o[1];
    }
    if ($order == PREG_PATTERN_ORDER) {
        foreach ($names as $i => $nm) {
            if ($nm !== "") { $matches[$nm] = $sets[$i]; }
            $matches[] = $sets[$i];
        }
        if ($marks !== null) { $matches["MARK"] = $marks; }
    }
    return __mec_pregerr() == PREG_NO_ERROR ? $matched : false;
}

// The replacement text for one match: preg_get_backref's \n, $n and ${n}
// (n of one or two digits; a group past the last that took part is empty),
// and a \ or $ after a backslash replaces that backslash.
function __mec_pregexpand($repl, $s, $o) {
    $count = __mec_pregcount($o);
    $n = strlen($repl);
    $out = "";
    $from = 0;
    $prevBs = false;
    $i = 0;
    while ($i < $n) {
        $c = $repl[$i];
        if ($c !== "\\" && $c !== "$") {
            $prevBs = false;
            $i++;
            continue;
        }
        if ($prevBs) {
            $out .= substr($repl, $from, $i - 1 - $from);
            $from = $i;
            $prevBs = false;
            $i++;
            continue;
        }
        $j = $i + 1;
        $brace = $c === "$" && $j < $n && $repl[$j] === '{';
        if ($brace) { $j++; }
        $g = -1;
        if ($i + 1 < $n && $j < $n && ord($repl[$j]) >= 48 && ord($repl[$j]) <= 57) {
            $g = ord($repl[$j]) - 48;
            $j++;
            if ($j < $n && ord($repl[$j]) >= 48 && ord($repl[$j]) <= 57) {
                $g = $g * 10 + ord($repl[$j]) - 48;
                $j++;
            }
            if ($brace) {
                if ($j < $n && $repl[$j] === '}') { $j++; } else { $g = -1; }
            }
        }
        if ($g < 0) {
            $prevBs = $c === "\\";
            $i++;
            continue;
        }
        $out .= substr($repl, $from, $i - $from);
        if ($g < $count && $o[2 * $g] >= 0) { $out .= substr($s, $o[2 * $g], $o[2 * $g + 1] - $o[2 * $g]); }
        $i = $j;
        $from = $j;
    }
    return $out . substr($repl, $from);
}

// php_pcre_replace_impl / php_pcre_replace_func_impl on one pattern and one
// subject: the new string, or null after an error; $count counts the
// replacements.
function __mec_pregrepl($fn, $regex, $repl, $callback, $subject, $limit, &$count, $flags) {
    $c = __mec_pregkey($fn, $regex);
    if ($c === false) { return null; }
    __mec_pregerr(PREG_NO_ERROR);
    $s = (string)$subject;
    $out = "";
    $last = 0;
    $pos = 0;
    while ($limit != 0) {
        $o = __mec_pregx($c[0], $s, $pos);
        if (is_int($o)) {
            __mec_pregerr($o);
            return null;
        }
        if (count($o) == 0) { break; }
        if ($o[1] < $o[0]) {
            __mec_pregerr(PREG_INTERNAL_ERROR);
            return null;
        }
        $count++;
        $out .= substr($s, $last, $o[0] - $last);
        if ($callback === null) { $out .= __mec_pregexpand($repl, $s, $o); }
        else { $out .= (string)$callback(__mec_pregsub($s, $o, $c[1], $flags)); }
        $limit--;
        $last = $o[1];
        $pos = $o[1] == $o[0] ? -$o[1] - 1 : $o[1];
    }
    return $out . substr($s, $last);
}

// One subject through every pattern in turn (php_pcre_replace_array): the
// replacements are taken in order, "" once they run out.
function __mec_pregsubject($fn, $pattern, $replacement, $callback, $subject, $limit, &$count, $flags) {
    if (!is_array($pattern)) { return __mec_pregrepl($fn, $pattern, $replacement, $callback, $subject, $limit, $count, $flags); }
    $reps = is_array($replacement) ? array_values($replacement) : null;
    $k = 0;
    foreach ($pattern as $p) {
        $r = $reps === null ? $replacement : ($k < count($reps) ? $reps[$k] : "");
        $k++;
        $subject = __mec_pregrepl($fn, $p, (string)$r, $callback, $subject, $limit, $count, $flags);
        if ($subject === null) { return null; }
    }
    return $subject;
}

// _preg_replace_common: a string subject answers a string (null after an
// error), an array one an array with the same keys, losing the entries that
// failed - and under preg_filter the ones nothing replaced.
function __mec_pregreplace($fn, $pattern, $replacement, $callback, $subject, $limit, &$count, $flags, $filter) {
    $count = 0;
    if (!is_array($subject)) {
        $r = __mec_pregsubject($fn, $pattern, $replacement, $callback, $subject, $limit, $count, $flags);
        return $filter && $count == 0 ? null : $r;
    }
    $out = [];
    foreach ($subject as $key => $s) {
        $before = $count;
        $r = __mec_pregsubject($fn, $pattern, $replacement, $callback, $s, $limit, $count, $flags);
        if ($r !== null && (!$filter || $count > $before)) { $out[$key] = $r; }
    }
    return $out;
}

function preg_replace($pattern, $replacement, $subject, $limit = -1, &$count = null) {
    if (is_array($replacement) && !is_array($pattern)) {
        throw new TypeError("preg_replace(): Argument #1 (\$pattern) must be of type array when argument #2 (\$replacement) is an array, string given");
    }
    return __mec_pregreplace("preg_replace", $pattern, $replacement, null, $subject, $limit, $count, 0, false);
}

function preg_filter($pattern, $replacement, $subject, $limit = -1, &$count = null) {
    if (is_array($replacement) && !is_array($pattern)) {
        throw new TypeError("preg_filter(): Argument #1 (\$pattern) must be of type array when argument #2 (\$replacement) is an array, string given");
    }
    return __mec_pregreplace("preg_filter", $pattern, $replacement, null, $subject, $limit, $count, 0, true);
}

function preg_replace_callback($pattern, $callback, $subject, $limit = -1, &$count = null, $flags = 0) {
    __mec_callable($callback, 'preg_replace_callback(): Argument #2 ($callback) must be a valid callback');
    return __mec_pregreplace("preg_replace_callback", $pattern, null, $callback, $subject, $limit, $count, $flags, false);
}

function preg_replace_callback_array($pattern, $subject, $limit = -1, &$count = null, $flags = 0) {
    $count = 0;
    foreach ($pattern as $regex => $callback) {
        __mec_callable($callback, 'preg_replace_callback_array(): Argument #1 ($pattern) must contain only valid callbacks');
        if (!is_string($regex)) {
            throw new TypeError("preg_replace_callback_array(): Argument #1 (\$pattern) must contain only string patterns as keys");
        }
        $n = 0;
        $subject = __mec_pregreplace("preg_replace_callback_array", $regex, null, $callback, $subject, $limit, $n, $flags, false);
        $count += $n;
        if ($subject === null) { return null; }
    }
    return $subject;
}

function preg_split($pattern, $subject, $limit = -1, $flags = 0) {
    $pattern = __mec_pregstr("preg_split", 1, "pattern", $pattern);
    $s = __mec_pregstr("preg_split", 2, "subject", $subject);
    $c = __mec_pregkey("preg_split", $pattern);
    if ($c === false) { return false; }
    $n = strlen($s);
    $noEmpty = $flags & PREG_SPLIT_NO_EMPTY;
    $delim = $flags & PREG_SPLIT_DELIM_CAPTURE;
    $oflags = ($flags & PREG_SPLIT_OFFSET_CAPTURE) ? PREG_OFFSET_CAPTURE : 0;
    $r = [];
    $last = 0;
    __mec_pregerr(PREG_NO_ERROR);
    if ($limit == 0) { $limit = -1; }
    if ($limit == -1 || $limit > 1) {
        $pos = 0;
        while (true) {
            $o = __mec_pregx($c[0], $s, $pos);
            if (is_int($o)) {
                __mec_pregerr($o);
                break;
            }
            if (count($o) == 0) { break; }
            if ($o[1] < $o[0]) {
                __mec_pregerr(PREG_INTERNAL_ERROR);
                break;
            }
            if (!$noEmpty || $o[0] != $last) {
                $r[] = __mec_pregval($s, [$last, $o[0]], 0, $oflags);
                if ($limit != -1) { $limit--; }
            }
            if ($delim) {
                $count = __mec_pregcount($o);
                for ($i = 1; $i < $count; $i++) {
                    if (!$noEmpty || $o[2 * $i] != $o[2 * $i + 1]) { $r[] = __mec_pregval($s, $o, $i, $oflags); }
                }
            }
            $last = $o[1];
            if ($limit != -1 && $limit <= 1) { break; }
            $pos = $o[1] == $o[0] ? -$o[1] - 1 : $o[1];
        }
        if (__mec_pregerr() != PREG_NO_ERROR) { return false; }
    }
    if (!$noEmpty || $last < $n) { $r[] = __mec_pregval($s, [$last, $n], 0, $oflags); }
    return $r;
}

// The characters preg_quote escapes, and NUL as \000.
function preg_quote($str, $delimiter = null) {
    $str = __mec_pregstr("preg_quote", 1, "str", $str);
    $d = $delimiter === null || $delimiter === "" ? "" : $delimiter[0];
    $n = strlen($str);
    $out = "";
    $from = 0;
    for ($i = 0; $i < $n; $i++) {
        $c = $str[$i];
        if ($c === "\0") {
            $out .= substr($str, $from, $i - $from) . "\\000";
            $from = $i + 1;
        } elseif (strpos(".\\+*?[^]$(){}=!<>|:-#", $c) !== false || ($d !== "" && $c === $d)) {
            $out .= substr($str, $from, $i - $from) . "\\";
            $from = $i;
        }
    }
    return $out . substr($str, $from);
}

function preg_grep($pattern, $array, $flags = 0) {
    $pattern = __mec_pregstr("preg_grep", 1, "pattern", $pattern);
    $c = __mec_pregkey("preg_grep", $pattern);
    if ($c === false) { return false; }
    __mec_pregerr(PREG_NO_ERROR);
    $invert = ($flags & PREG_GREP_INVERT) != 0;
    $out = [];
    foreach ($array as $key => $v) {
        $o = __mec_pregx($c[0], (string)$v, 0);
        if (is_int($o)) {
            __mec_pregerr($o);
            break;
        }
        if ((count($o) > 0) != $invert) { $out[$key] = $v; }
    }
    return $out;
}

function preg_last_error() { return __mec_pregerr(); }

function preg_last_error_msg() {
    $code = __mec_pregerr();
    if ($code == PREG_NO_ERROR) { return "No error"; }
    if ($code == PREG_INTERNAL_ERROR) { return "Internal error"; }
    if ($code == PREG_BAD_UTF8_ERROR) { return "Malformed UTF-8 characters, possibly incorrectly encoded"; }
    if ($code == PREG_BAD_UTF8_OFFSET_ERROR) { return "The offset did not correspond to the beginning of a valid UTF-8 code point"; }
    if ($code == PREG_BACKTRACK_LIMIT_ERROR) { return "Backtrack limit exhausted"; }
    if ($code == PREG_RECURSION_LIMIT_ERROR) { return "Recursion limit exhausted"; }
    if ($code == PREG_JIT_STACKLIMIT_ERROR) { return "JIT stack limit exhausted"; }
    return "Unknown error";
}

// ----- formatting ------------------------------------------------------------
// sprintf's padding: $s padded to $width with $pad, on the left unless $left;
// with '0' padding the sign stays in front of the zeros.
function __mec_pad($s, $width, $pad, $left) {
    $n = $width - strlen($s);
    if ($n <= 0) { return $s; }
    if ($left) { return $s . str_repeat($pad, $n); }
    if ($pad === "0" && ($s[0] == "-" || $s[0] == "+")) { return $s[0] . str_repeat("0", $n) . substr($s, 1); }
    return str_repeat($pad, $n) . $s;
}
// The unsigned 64 bit digits of an int in base 2^$bits ('b', 'o', 'x').
function __mec_radix($v, $bits, $upper) {
    $digits = $upper ? "0123456789ABCDEF" : "0123456789abcdef";
    $mask = (1 << $bits) - 1;
    $out = "";
    $n = 0;
    do {
        $out = $digits[$v & $mask] . $out;
        $v = ($v >> $bits) & (PHP_INT_MAX >> ($bits - 1));
        $n++;
    } while ($v != 0 && $n < 64);
    return $out;
}
function __mec_format($format, $args) {
    $out = "";
    $n = strlen($format);
    $next = 0;
    $i = 0;
    while ($i < $n) {
        $p = strpos($format, "%", $i);
        if ($p === false) { $p = $n; }
        $out .= substr($format, $i, $p - $i);
        $i = $p;
        if ($i >= $n) { break; }
        $i++;
        if (substr($format, $i, 1) === "%") { $out .= "%"; $i++; continue; }
        $argnum = -1;
        $j = $i;
        while ($j < $n && ord(substr($format, $j, 1)) >= 48 && ord(substr($format, $j, 1)) <= 57) { $j++; }
        if ($j > $i && $j < $n && substr($format, $j, 1) == "\$") {
            $argnum = (int)substr($format, $i, $j - $i) - 1;
            if ($argnum < 0) { throw new ValueError("Argument number specifier must be greater than zero and less than 2147483647"); }
            $i = $j + 1;
        }
        $left = false;
        $plus = false;
        $pad = " ";
        while ($i < $n) {
            $f = substr($format, $i, 1);
            if ($f == "-") { $left = true; }
            elseif ($f == "+") { $plus = true; }
            elseif ($f == "0" || $f == " ") { $pad = $f; }
            elseif ($f == "'") { $i++; $pad = substr($format, $i, 1); }
            else { break; }
            $i++;
        }
        $width = 0;
        while ($i < $n && ord(substr($format, $i, 1)) >= 48 && ord(substr($format, $i, 1)) <= 57) { $width = $width * 10 + (int)substr($format, $i, 1); $i++; }
        $prec = -1;
        if ($i < $n && substr($format, $i, 1) == ".") {
            $i++;
            $prec = 0;
            while ($i < $n && ord(substr($format, $i, 1)) >= 48 && ord(substr($format, $i, 1)) <= 57) { $prec = $prec * 10 + (int)substr($format, $i, 1); $i++; }
        }
        if ($i >= $n) { throw new ValueError("Missing format specifier at end of string"); }
        $conv = substr($format, $i, 1);
        $i++;
        $at = $argnum >= 0 ? $argnum : $next++;
        if ($at >= count($args)) {
            throw new ArgumentCountError(($at + 2) . " arguments are required, " . (count($args) + 1) . " given");
        }
        $v = $args[$at];
        if ($conv == "s") {
            $s = is_array($v) ? "Array" : (string)$v;
            if ($prec >= 0) { $s = substr($s, 0, $prec); }
        } elseif ($conv == "d") {
            $iv = (int)$v;
            $s = (string)$iv;
            if ($plus && $iv >= 0) { $s = "+" . $s; }
        } elseif ($conv == "u") {
            $iv = (int)$v;
            $s = $iv >= 0 ? (string)$iv : __mec_udec($iv);
        } elseif ($conv == "c") {
            $out .= chr((int)$v);
            continue;
        } elseif ($conv == "x" || $conv == "X") {
            $s = __mec_radix((int)$v, 4, $conv == "X");
        } elseif ($conv == "o") {
            $s = __mec_radix((int)$v, 3, false);
        } elseif ($conv == "b") {
            $s = __mec_radix((int)$v, 1, false);
        } elseif ($conv == "e" || $conv == "E" || $conv == "f" || $conv == "F" || $conv == "g" || $conv == "G") {
            $fv = (float)$v;
            if ($prec < 0) { $prec = 6; }
            if ($prec > 53) { $prec = 53; }
            if (is_nan($fv)) { $s = "NaN"; }
            elseif (is_infinite($fv)) { $s = $fv < 0 ? "-Inf" : "Inf"; }
            else {
                $lc = strtolower($conv);
                if ($lc == "g" && $prec == 0) { $prec = 1; }
                $s = __mec_fmtfloat($fv, $lc, $prec);
                if ($conv == "E" || $conv == "G") { $s = strtoupper($s); }
                if ($fv < 0) { $s = "-" . $s; }
                elseif ($plus) { $s = "+" . $s; }
            }
        } else {
            throw new ValueError("Unknown format specifier \"" . $conv . "\"");
        }
        $out .= __mec_pad($s, $width, $pad, $left);
    }
    return $out;
}
// A negative int read as unsigned 64 bit, in decimal.
function __mec_udec($v) {
    $hi = ($v >> 32) & 0xFFFFFFFF;
    $lo = $v & 0xFFFFFFFF;
    $digits = "";
    while ($hi > 0 || $lo > 0) {
        $r = ($hi % 10) * 4294967296 + $lo;
        $hi = intdiv($hi, 10);
        $lo = intdiv($r, 10);
        $hi += intdiv($lo, 4294967296);
        $lo = $lo % 4294967296;
        $digits = ($r % 10) . $digits;
    }
    return $digits;
}
function is_nan($num) { return $num != $num; }
function is_infinite($num) { return $num == INF || $num == -INF; }

// ----- math ------------------------------------------------------------------
// Written on __mec_math, the C library function of one or two doubles. An
// int|float parameter keeps an int as an int; a float one converts it. Both take
// a bool, null or a numeric string the way a coercive-mode call does, and refuse
// anything else with PHP's TypeError.
function __mec_num($fn, $n, $name, $v, $type) {
    if (is_int($v) || is_float($v)) { return $type == "float" ? (float)$v : $v; }
    if (is_bool($v) || $v === null || (is_string($v) && is_numeric($v))) {
        return $type == "float" ? (float)$v : $v + 0;
    }
    throw new TypeError($fn . "(): Argument #" . $n . " (\$" . $name . ") must be of type " . $type . ", " . __mec_vname($v) . " given");
}
function __mec_f1($fn, $num) { return __mec_math($fn, __mec_num($fn, 1, "num", $num, "float"), 0.0); }
function floor($num) { return __mec_math("floor", __mec_num("floor", 1, "num", $num, "int|float"), 0.0); }
function ceil($num) { return __mec_math("ceil", __mec_num("ceil", 1, "num", $num, "int|float"), 0.0); }
function sqrt($num) { return __mec_f1("sqrt", $num); }
function exp($num) { return __mec_f1("exp", $num); }
function expm1($num) { return __mec_f1("expm1", $num); }
function log10($num) { return __mec_f1("log10", $num); }
function log1p($num) { return __mec_f1("log1p", $num); }
function sin($num) { return __mec_f1("sin", $num); }
function cos($num) { return __mec_f1("cos", $num); }
function tan($num) { return __mec_f1("tan", $num); }
function asin($num) { return __mec_f1("asin", $num); }
function acos($num) { return __mec_f1("acos", $num); }
function atan($num) { return __mec_f1("atan", $num); }
function sinh($num) { return __mec_f1("sinh", $num); }
function cosh($num) { return __mec_f1("cosh", $num); }
function tanh($num) { return __mec_f1("tanh", $num); }
function asinh($num) { return __mec_f1("asinh", $num); }
function acosh($num) { return __mec_f1("acosh", $num); }
function atanh($num) { return __mec_f1("atanh", $num); }
function atan2($y, $x) {
    return __mec_math("atan2", __mec_num("atan2", 1, "y", $y, "float"), __mec_num("atan2", 2, "x", $x, "float"));
}
function hypot($x, $y) {
    return __mec_math("hypot", __mec_num("hypot", 1, "x", $x, "float"), __mec_num("hypot", 2, "y", $y, "float"));
}
function fmod($num1, $num2) {
    return __mec_math("fmod", __mec_num("fmod", 1, "num1", $num1, "float"), __mec_num("fmod", 2, "num2", $num2, "float"));
}
// fdiv is IEEE division: a zero divisor answers INF, -INF or NAN, never throws.
function fdiv($num1, $num2) {
    return __mec_math("div", __mec_num("fdiv", 1, "num1", $num1, "float"), __mec_num("fdiv", 2, "num2", $num2, "float"));
}
function fpow($num, $exponent) {
    return __mec_math("pow", __mec_num("fpow", 1, "num", $num, "float"), __mec_num("fpow", 2, "exponent", $exponent, "float"));
}
function pow($num, $exponent) { return $num ** $exponent; }
// log with a base: 2 and 10 are the exact log2 / log10, 1 has no logarithm.
function log($num, $base = null) {
    $x = __mec_num("log", 1, "num", $num, "float");
    if ($base === null) { return __mec_math("log", $x, 0.0); }
    $b = __mec_num("log", 2, "base", $base, "float");
    if ($b == 2.0) { return __mec_math("log2", $x, 0.0); }
    if ($b == 10.0) { return __mec_math("log10", $x, 0.0); }
    if ($b == 1.0) { return NAN; }
    if ($b <= 0.0) { throw new ValueError("log(): Argument #2 (\$base) must be greater than 0"); }
    return __mec_math("log", $x, 0.0) / __mec_math("log", $b, 0.0);
}
function pi() { return M_PI; }
// microtime (ext/standard/microtime.c) on __mec_clock, the wall clock in
// microseconds since the epoch: "<usec / 10^6 to 8 places> <sec>", or with
// $as_float sec + usec / 10^6 as one float.
function microtime(bool $as_float = false) {
    $us = __mec_clock();
    $sec = intdiv($us, 1000000);
    $frac = ($us % 1000000) / 1000000.0;
    return $as_float ? $sec + $frac : sprintf("%.8F %d", $frac, $sec);
}
function deg2rad($num) { return (__mec_num("deg2rad", 1, "num", $num, "float") / 180.0) * M_PI; }
function rad2deg($num) { return (__mec_num("rad2deg", 1, "num", $num, "float") / M_PI) * 180.0; }
function is_finite($num) { return !is_nan($num) && !is_infinite($num); }
function intval($value, $base = 10) {
    if (!is_string($value) || $base == 10) { return (int)$value; }
    return __mec_strtol($value, $base);
}
function floatval($value) { return (float)$value; }
function doubleval($value) { return (float)$value; }
function boolval($value) { return (bool)$value; }
function strval($value) { return (string)$value; }

const PHP_ROUND_HALF_UP = 1;
const PHP_ROUND_HALF_DOWN = 2;
const PHP_ROUND_HALF_EVEN = 3;
const PHP_ROUND_HALF_ODD = 4;
// round() is PHP's _php_math_round: the value scaled by 10^places, its integral
// part corrected when the next integer scales back to the value itself, then
// __mec_roundedge decides the last step against the UNSCALED value. An int with
// no negative precision only converts.
function round($num, int $precision = 0, $mode = PHP_ROUND_HALF_UP) {
    $v = __mec_num("round", 1, "num", $num, "int|float");
    if ($mode < 1 || $mode > 4) {
        throw new ValueError("round(): Argument #3 (\$mode) must be a valid rounding mode (RoundingMode::*)");
    }
    $places = $precision;
    if (is_int($v) && $places >= 0) { return (float)$v; }
    $value = (float)$v;
    if (!is_finite($value) || $value == 0.0) { return $value; }
    if ($places == 0 && $value == __mec_math("trunc", $value, 0.0)) { return $value; }
    $ap = $places < 0 ? -$places : $places;
    $exponent = (float)("1e" . $ap);
    if ($value >= 0.0) {
        $tmp = __mec_math("floor", $places > 0 ? $value * $exponent : $value / $exponent, 0.0);
        $tmp2 = $tmp + 1.0;
    } else {
        $tmp = __mec_math("ceil", $places > 0 ? $value * $exponent : $value / $exponent, 0.0);
        $tmp2 = $tmp - 1.0;
    }
    if (($places > 0 ? $tmp2 / $exponent : $tmp2 * $exponent) == $value) { $tmp = $tmp2; }
    if (abs($tmp) >= 1e16) { return $value; }
    $tmp = __mec_roundedge($tmp, $value, $exponent, $places, $mode);
    if ($ap < 23) { return $places > 0 ? $tmp / $exponent : $tmp * $exponent; }
    $r = (float)(sprintf("%.0F", $tmp) . "e" . -$places);
    return is_finite($r) ? $r : $value;
}
// php_round_helper: whether the integral part steps away from zero, by where
// the value lies against the half-way point between it and the next one.
function __mec_roundedge($integral, $value, $exponent, $places, $mode) {
    $abs = abs($value);
    $half = $integral + __mec_math("copysign", 0.5, $integral);
    $edge = abs($places > 0 ? $half / $exponent : $half * $exponent);
    $away = $integral + __mec_math("copysign", 1.0, $integral);
    if ($abs > $edge) { return $away; }
    if ($abs < $edge || $mode == PHP_ROUND_HALF_DOWN) { return $integral; }
    if ($mode == PHP_ROUND_HALF_UP) { return $away; }
    $even = __mec_math("fmod", $integral, 2.0) == 0.0;
    return ($mode == PHP_ROUND_HALF_EVEN) == $even ? $integral : $away;
}

// C's isspace: space, \t, \n, \v, \f, \r.
function __mec_cspace($c) {
    $o = ord($c);
    return $o == 32 || ($o >= 9 && $o <= 13);
}
// A digit's value in any base up to 36; 99 for anything that is not a digit.
function __mec_digit($c) {
    $o = $c === "" ? 0 : ord($c);
    if ($o >= 48 && $o <= 57) { return $o - 48; }
    if ($o >= 65 && $o <= 90) { return $o - 55; }
    if ($o >= 97 && $o <= 122) { return $o - 87; }
    return 99;
}
// C's strtol, as intval() with a base other than 10 reads a string: leading
// space, a sign, the base's own prefix (0x, 0b, or a leading 0 for octal when
// the base is 0), then the digits up to the first that is not one; a value out
// of range saturates. It accumulates NEGATIVELY so PHP_INT_MIN is reachable.
function __mec_strtol($s, $base) {
    $n = strlen($s);
    $i = 0;
    while ($i < $n && __mec_cspace($s[$i])) { $i++; }
    $neg = false;
    if ($i < $n && ($s[$i] == "-" || $s[$i] == "+")) { $neg = $s[$i] == "-"; $i++; }
    $two = strtolower(substr($s, $i, 2));
    if (($base == 0 || $base == 16) && $two == "0x" && __mec_digit(substr($s, $i + 2, 1)) < 16) { $base = 16; $i += 2; }
    elseif (($base == 0 || $base == 2) && $two == "0b" && __mec_digit(substr($s, $i + 2, 1)) < 2) { $base = 2; $i += 2; }
    elseif ($base == 0) { $base = substr($s, $i, 1) === "0" ? 8 : 10; }
    if ($base < 2 || $base > 36) { return 0; }
    $lim = $neg ? PHP_INT_MIN : -PHP_INT_MAX;
    $acc = 0;
    $over = false;
    while ($i < $n) {
        $d = __mec_digit($s[$i]);
        if ($d >= $base) { break; }
        if ($acc < intdiv($lim + $d, $base)) { $over = true; }
        else { $acc = $acc * $base - $d; }
        $i++;
    }
    if ($over) { return $neg ? PHP_INT_MIN : PHP_INT_MAX; }
    return $neg ? $acc : -$acc;
}
// _php_math_basetozval: surrounding space and the base's own prefix are
// dropped, every other character that is not a digit of the base is skipped,
// and past PHP_INT_MAX the value continues as a float.
function __mec_basetonum($str, $base) {
    $s = (string)$str;
    $a = 0;
    $e = strlen($s);
    while ($a < $e && __mec_cspace($s[$a])) { $a++; }
    while ($e > $a && __mec_cspace($s[$e - 1])) { $e--; }
    if ($e - $a >= 2) {
        $two = strtolower(substr($s, $a, 2));
        if (($base == 16 && $two == "0x") || ($base == 8 && $two == "0o") || ($base == 2 && $two == "0b")) { $a += 2; }
    }
    $num = 0;
    $isFloat = false;
    $cutoff = intdiv(PHP_INT_MAX, $base);
    $cutlim = PHP_INT_MAX % $base;
    for ($i = $a; $i < $e; $i++) {
        $d = __mec_digit($s[$i]);
        if ($d >= $base) { continue; }
        if (!$isFloat && ($num < $cutoff || ($num == $cutoff && $d <= $cutlim))) { $num = $num * $base + $d; continue; }
        if (!$isFloat) { $num = (float)$num; $isFloat = true; }
        $num = $num * $base + $d;
    }
    return $num;
}
function bindec($binary_string) { return __mec_basetonum($binary_string, 2); }
function hexdec($hex_string) { return __mec_basetonum($hex_string, 16); }
function octdec($octal_string) { return __mec_basetonum($octal_string, 8); }
// decbin / decoct / dechex render the 64 bits UNSIGNED, so a negative int is its
// two's complement.
function decbin($num) { return __mec_radix((int)$num, 1, false); }
function decoct($num) { return __mec_radix((int)$num, 3, false); }
function dechex($num) { return __mec_radix((int)$num, 4, false); }
function base_convert($num, $from_base, $to_base) {
    if ($from_base < 2 || $from_base > 36) {
        throw new ValueError("base_convert(): Argument #2 (\$from_base) must be between 2 and 36 (inclusive)");
    }
    if ($to_base < 2 || $to_base > 36) {
        throw new ValueError("base_convert(): Argument #3 (\$to_base) must be between 2 and 36 (inclusive)");
    }
    $v = __mec_basetonum($num, $from_base);
    $digits = "0123456789abcdefghijklmnopqrstuvwxyz";
    $out = "";
    if (is_float($v)) {
        $f = __mec_math("floor", $v, 0.0);
        if (is_infinite($f)) { throw new ValueError("An infinite value cannot be converted to base " . $to_base); }
        do {
            $out = $digits[(int)__mec_math("fmod", $f, (float)$to_base)] . $out;
            $f = $f / $to_base;
        } while (abs($f) >= 1);
        return $out;
    }
    do {
        $out = $digits[$v % $to_base] . $out;
        $v = intdiv($v, $to_base);
    } while ($v != 0);
    return $out;
}
function sprintf($format, ...$values) { return __mec_format((string)$format, $values); }
function vsprintf($format, $values) { return __mec_format((string)$format, array_values($values)); }
function printf($format, ...$values) {
    $s = __mec_format((string)$format, $values);
    echo $s;
    return strlen($s);
}
function vprintf($format, $values) {
    $s = __mec_format((string)$format, array_values($values));
    echo $s;
    return strlen($s);
}
// number_format rounds half away from zero the way round() does - on the value
// as 15 significant digits when the requested places lie inside them - and then
// groups the integer digits by thousands.
function number_format(float $num, int $decimals = 0, ?string $decimal_separator = ".", ?string $thousands_separator = ",") {
    $f = $num;
    $decimal_separator ??= ".";
    $thousands_separator ??= ",";
    if ($decimals < 0) { $decimals = 0; }
    $neg = $f < 0;
    $a = $neg ? -$f : $f;
    $e = __mec_fmtfloat($a, "e", 14);
    $p = strpos($e, "e");
    $exp = (int)substr($e, $p + 1);
    if ($a == 0 || $decimals > 14 - $exp) {
        $digits = __mec_fmtfloat($a, "f", $decimals);
    } else {
        $sig = substr($e, 0, 1) . substr($e, 2, $p - 2);
        $digits = __mec_roundsig($sig, $exp, $decimals);
    }
    $dot = strpos($digits, ".");
    $int = $dot === false ? $digits : substr($digits, 0, $dot);
    $frac = $dot === false ? "" : substr($digits, $dot + 1);
    $grouped = "";
    $len = strlen($int);
    for ($i = 0; $i < $len; $i++) {
        if ($i > 0 && ($len - $i) % 3 == 0) { $grouped .= $thousands_separator; }
        $grouped .= $int[$i];
    }
    $zero = trim($int . $frac, "0") === "";
    $out = $grouped . ($decimals > 0 ? $decimal_separator . $frac : "");
    return $neg && !$zero ? "-" . $out : $out;
}
// The fixed form of the significant digits $sig (the first one worth 10^$exp)
// rounded half up to $places decimals.
function __mec_roundsig($sig, $exp, $places) {
    $keep = $exp + 1 + $places;
    if ($keep < 0) { return $places > 0 ? "0." . str_repeat("0", $places) : "0"; }
    $digs = $sig;
    if (strlen($digs) < $keep + 1) { $digs .= str_repeat("0", $keep + 1 - strlen($digs)); }
    $head = substr($digs, 0, $keep);
    if (ord(substr($digs, $keep, 1)) >= 53) {
        $carry = 1;
        $r = "";
        for ($i = strlen($head) - 1; $i >= 0; $i--) {
            $d = ord(substr($head, $i, 1)) - 48 + $carry;
            $carry = $d == 10 ? 1 : 0;
            $r = ($d == 10 ? "0" : (string)$d) . $r;
        }
        $head = $carry ? "1" . $r : $r;
        if ($carry) { $exp++; }
    }
    $intLen = $exp + 1;
    if ($intLen <= 0) {
        $int = "0";
        $frac = str_repeat("0", -$intLen) . $head;
    } else {
        $int = substr($head, 0, $intLen);
        $frac = substr($head, $intLen);
    }
    if ($int === "") { $int = "0"; }
    $frac = substr($frac . str_repeat("0", $places), 0, $places);
    return $places > 0 ? $int . "." . $frac : $int;
}

// ----- print_r ---------------------------------------------------------------
// zend_print_zval_r. A scalar is its string form; an array or object is a header
// line, then "(" indented $indent spaces, each entry 4 further as "[key] => value"
// with a nested block 8 further, then ")". An object already being printed further
// up is " *RECURSION*" after its header, and so is an array reached again through
// a reference; $seen holds those objects and the references' __mec_refid. The
// reference $value itself was read through is not passed, so an array holding a
// reference to itself is printed once more than php does before it is seen again.
function print_r($value, $return = false) {
    $s = __mec_printr($value, 0, []);
    if ($return) { return $s; }
    echo $s;
    return true;
}
function __mec_printr($v, $indent, $seen) {
    if (is_array($v)) { return "Array\n" . __mec_printrhash($v, $indent, false, $seen); }
    if (!is_object($v)) { return (string)$v; }
    $name = get_class($v);
    $z = strpos($name, "\0");
    if ($z !== false) { $name = substr($name, 0, $z); }
    if ($v instanceof UnitEnum) {
        $props = ["name" => $v->name];
        $head = $name . " Enum";
        if ($v instanceof BackedEnum) {
            $props["value"] = $v->value;
            $head .= is_int($v->value) ? ":int" : ":string";
        }
        $head .= "\n";
    } else {
        $props = (array)$v;
        $head = $name . " Object\n";
    }
    foreach ($seen as $o) {
        if ($o === $v) { return $head . " *RECURSION*"; }
    }
    $seen[] = $v;
    return $head . __mec_printrhash($props, $indent, true, $seen);
}
// A mangled property name shows its visibility: "\0*\0p" is [p:protected] and
// "\0C\0p" is [p:C:private].
function __mec_printrhash($h, $indent, $isObj, $seen) {
    $pad = str_repeat(" ", $indent);
    $out = $pad . "(\n";
    foreach ($h as $k => $e) {
        $id = is_array($e) ? __mec_refid($h, $k) : 0;
        if ($isObj && is_string($k) && $k !== "" && $k[0] === "\0") {
            $p = strpos($k, "\0", 1);
            $cls = substr($k, 1, $p - 1);
            $k = substr($k, $p + 1) . ($cls === "*" ? ":protected" : ":" . $cls . ":private");
        }
        if ($id !== 0 && in_array($id, $seen, true)) { $s = "Array\n *RECURSION*"; }
        else { $s = __mec_printr($e, $indent + 8, $id === 0 ? $seen : [...$seen, $id]); }
        $out .= $pad . "    [" . $k . "] => " . $s . "\n";
    }
    return $out . $pad . ")\n";
}

// ----- var_export and JSON ---------------------------------------------------
// A float at serialize_precision=-1, which var_export and json_encode share with
// var_dump: the shortest round-tripping digits laid out by zend_gcvt at 17
// digits, with exponent marker $e; INF and NAN by name.
function __mec_serfloat($num, $e) {
    if (is_nan($num)) { return "NAN"; }
    if (is_infinite($num)) { return $num < 0 ? "-INF" : "INF"; }
    if ($num == 0) { return fdiv(1, $num) < 0 ? "-0" : "0"; }
    $s = __mec_fmtfloat($num, "r", 17);
    if ($e === "E") { $s = strtoupper($s); }
    return $num < 0 ? "-" . $s : $s;
}
function var_export($value, $return = false) {
    $s = __mec_export($value, 1);
    if ($return) { return $s; }
    echo $s;
    return null;
}
// php_var_export_ex. $level counts from 1; an array element is indented
// $level + 1 spaces, an object property $level + 2, and a nested array or object
// starts on a new line indented $level - 1. A float always shows a fraction or an
// exponent, and an object is rebuilt through __set_state ((object) for stdClass).
function __mec_export($v, $level) {
    if ($v === null) { return "NULL"; }
    if ($v === true) { return "true"; }
    if ($v === false) { return "false"; }
    if (is_int($v)) { return $v == PHP_INT_MIN ? "-9223372036854775807-1" : (string)$v; }
    if (is_float($v)) {
        $s = __mec_serfloat($v, "E");
        return is_finite($v) && strpos($s, ".") === false && strpos($s, "E") === false ? $s . ".0" : $s;
    }
    if (is_string($v)) { return __mec_exportstr($v); }
    $close = $level > 1 ? str_repeat(" ", $level - 1) : "";
    $out = $level > 1 ? "\n" . $close : "";
    if (is_array($v)) {
        $out .= "array (\n";
        foreach ($v as $k => $e) {
            $out .= str_repeat(" ", $level + 1) . (is_int($k) ? $k : __mec_exportstr($k)) . " => " . __mec_export($e, $level + 2) . ",\n";
        }
        return $out . $close . ")";
    }
    if ($v instanceof UnitEnum) { return $out . "\\" . get_class($v) . "::" . $v->name; }
    $std = get_class($v) === "stdClass";
    $out .= $std ? "(object) array(\n" : "\\" . get_class($v) . "::__set_state(array(\n";
    foreach ((array)$v as $k => $e) {
        $out .= str_repeat(" ", $level + 2) . __mec_exportstr(__mec_unmangle((string)$k)) . " => " . __mec_export($e, $level + 2) . ",\n";
    }
    return $out . $close . ($std ? ")" : "))");
}
// A property name out of an (array) cast without its visibility mangling:
// "\0*\0p" and "\0Class\0p" are both p.
function __mec_unmangle($k) {
    return $k !== "" && $k[0] === "\0" ? substr($k, strpos($k, "\0", 1) + 1) : $k;
}
// An object's public properties, which are what json_encode writes: the mangled
// names of the others are dropped.
function __mec_pubprops($v) {
    $out = [];
    foreach ((array)$v as $k => $e) {
        if (!is_string($k) || $k === "" || $k[0] !== "\0") { $out[$k] = $e; }
    }
    return $out;
}
// A single-quoted literal: backslash and quote escaped, a NUL byte spliced in as "\0".
function __mec_exportstr($s) {
    return "'" . strtr($s, ["\\" => "\\\\", "'" => "\\'", "\0" => "' . \"\\0\" . '"]) . "'";
}

// The error of the last json_encode / json_decode call without
// JSON_THROW_ON_ERROR; $code >= 0 records one.
function __mec_jsonerr($code = -1) {
    static $last = 0;
    if ($code >= 0) { $last = $code; }
    return $last;
}
function json_last_error() { return __mec_jsonerr(); }
function json_last_error_msg() { return __mec_jsonmsg(__mec_jsonerr()); }
function __mec_jsonmsg($code) {
    $msgs = ["No error", "Maximum stack depth exceeded", "State mismatch (invalid or malformed JSON)",
        "Control character error, possibly incorrectly encoded", "Syntax error",
        "Malformed UTF-8 characters, possibly incorrectly encoded", "Recursion detected",
        "Inf and NaN cannot be JSON encoded", "Type is not supported", "The decoded property name is invalid",
        "Single unpaired UTF-16 surrogate in unicode escape", "Non-backed enums have no default serialization"];
    return $msgs[$code] ?? "Unknown error";
}
// The end of a call: record the error, or throw it under JSON_THROW_ON_ERROR.
function __mec_jsondone($code, $flags) {
    if (!($flags & JSON_THROW_ON_ERROR)) { __mec_jsonerr($code); }
    elseif ($code != 0) { throw new JsonException(__mec_jsonmsg($code), $code); }
}
// The code point of the well-formed UTF-8 sequence at byte $i of $s and its
// length, or null for a malformed one (overlong forms, surrogates and values
// past U+10FFFF included).
function __mec_utf8at($s, $i, $n) {
    $c = ord($s[$i]);
    if ($c < 0x80) { return [$c, 1]; }
    if ($c < 0xC2) { return null; }
    $len = $c < 0xE0 ? 2 : ($c < 0xF0 ? 3 : ($c < 0xF5 ? 4 : 0));
    if ($len == 0 || $i + $len > $n) { return null; }
    $cp = $c & (0xFF >> ($len + 1));
    for ($k = 1; $k < $len; $k++) {
        $b = ord($s[$i + $k]);
        if (($b & 0xC0) != 0x80) { return null; }
        $cp = ($cp << 6) | ($b & 0x3F);
    }
    if (($len == 3 && ($cp < 0x800 || ($cp >= 0xD800 && $cp <= 0xDFFF))) || ($len == 4 && ($cp < 0x10000 || $cp > 0x10FFFF))) {
        return null;
    }
    return [$cp, $len];
}
function __mec_utf8chr($cp) {
    if ($cp < 0x80) { return chr($cp); }
    if ($cp < 0x800) { return chr(0xC0 | ($cp >> 6)) . chr(0x80 | ($cp & 0x3F)); }
    if ($cp < 0x10000) { return chr(0xE0 | ($cp >> 12)) . chr(0x80 | (($cp >> 6) & 0x3F)) . chr(0x80 | ($cp & 0x3F)); }
    return chr(0xF0 | ($cp >> 18)) . chr(0x80 | (($cp >> 12) & 0x3F)) . chr(0x80 | (($cp >> 6) & 0x3F)) . chr(0x80 | ($cp & 0x3F));
}

// json_encode: php_json_encode_zval. $st carries the flags, the depth limit, the
// error (the first one, or the last under JSON_PARTIAL_OUTPUT_ON_ERROR, which
// substitutes for what failed and carries on) and the objects being encoded.
function json_encode($value, $flags = 0, $depth = 512) {
    $st = new stdClass();
    $st->flags = $flags;
    $st->depth = $depth;
    $st->err = 0;
    $st->open = [];
    $out = __mec_jenc($value, $st, 0);
    if ($flags & JSON_PARTIAL_OUTPUT_ON_ERROR) {
        __mec_jsonerr($st->err);
        return $out;
    }
    __mec_jsondone($st->err, $flags);
    return $st->err == 0 ? $out : false;
}
function __mec_jfail($st, $code, $sub) {
    $st->err = $code;
    return $sub;
}
function __mec_jstop($st) { return $st->err != 0 && !($st->flags & JSON_PARTIAL_OUTPUT_ON_ERROR); }
function __mec_jfloat($v, $flags) {
    $s = __mec_serfloat($v, "e");
    return ($flags & JSON_PRESERVE_ZERO_FRACTION) && strpos($s, ".") === false && strpos($s, "e") === false ? $s . ".0" : $s;
}
function __mec_jenc($v, $st, $level) {
    if ($v === null) { return "null"; }
    if ($v === true) { return "true"; }
    if ($v === false) { return "false"; }
    if (is_int($v)) { return (string)$v; }
    if (is_float($v)) { return is_finite($v) ? __mec_jfloat($v, $st->flags) : __mec_jfail($st, JSON_ERROR_INF_OR_NAN, "0"); }
    if (is_string($v)) {
        if (($st->flags & JSON_NUMERIC_CHECK) && is_numeric($v)) {
            $num = $v + 0;
            if (is_int($num)) { return (string)$num; }
            if (is_finite($num)) { return __mec_jfloat($num, $st->flags); }
        }
        return __mec_jstr($v, $st);
    }
    if (is_array($v)) { return __mec_jlist($v, !($st->flags & JSON_FORCE_OBJECT) && array_is_list($v), $st, $level); }
    if (!is_object($v)) { return __mec_jfail($st, JSON_ERROR_UNSUPPORTED_TYPE, "null"); }
    if ($v instanceof UnitEnum && !($v instanceof JsonSerializable)) {
        return $v instanceof BackedEnum ? __mec_jenc($v->value, $st, $level) : __mec_jfail($st, JSON_ERROR_NON_BACKED_ENUM, "0");
    }
    if (in_array($v, $st->open, true)) { return __mec_jfail($st, JSON_ERROR_RECURSION, "null"); }
    $st->open[] = $v;
    if ($v instanceof JsonSerializable) {
        $r = $v->jsonSerialize();
        $out = $r === $v ? __mec_jlist(__mec_pubprops($v), false, $st, $level) : __mec_jenc($r, $st, $level);
    } else {
        $out = __mec_jlist(__mec_pubprops($v), false, $st, $level);
    }
    array_pop($st->open);
    return $out;
}
// An array or object's elements, as a JSON array ($list) or object. Pretty
// printing puts each element on its own line, four spaces per level, and keeps
// an empty one closed on the same line.
function __mec_jlist($a, $list, $st, $level) {
    $pretty = ($st->flags & JSON_PRETTY_PRINT) != 0;
    $nl = $pretty ? "\n" . str_repeat("    ", $level + 1) : "";
    $out = $list ? "[" : "{";
    $first = true;
    foreach ($a as $k => $e) {
        $out .= ($first ? "" : ",") . $nl;
        $first = false;
        if (!$list) {
            $out .= __mec_jstr((string)$k, $st) . ($pretty ? ": " : ":");
            if (__mec_jstop($st)) { return ""; }
        }
        $out .= __mec_jenc($e, $st, $level + 1);
        if (__mec_jstop($st)) { return ""; }
    }
    if ($level + 1 > $st->depth && !($st->flags & JSON_PARTIAL_OUTPUT_ON_ERROR)) { return __mec_jfail($st, JSON_ERROR_DEPTH, ""); }
    if (!$first && $pretty) { $out .= "\n" . str_repeat("    ", $level); }
    return $out . ($list ? "]" : "}");
}
// php_json_escape_string: '"', '\' and '/' escaped, the short escapes for
// \b \f \n \r \t, any other control character and (unless
// JSON_UNESCAPED_UNICODE) every non-ASCII character as \u with lowercase hex,
// a character past the BMP as a surrogate pair. U+2028 and U+2029 stay escaped
// unless JSON_UNESCAPED_LINE_TERMINATORS too.
function __mec_jstr($s, $st) {
    $f = $st->flags;
    $n = strlen($s);
    $out = "\"";
    $from = 0;
    $i = 0;
    while ($i < $n) {
        $c = ord($s[$i]);
        $esc = null;
        $len = 1;
        if ($c >= 0x80) {
            $u = __mec_utf8at($s, $i, $n);
            if ($u === null) {
                if ($f & JSON_INVALID_UTF8_IGNORE) { $esc = ""; }
                elseif ($f & JSON_INVALID_UTF8_SUBSTITUTE) { $esc = $f & JSON_UNESCAPED_UNICODE ? "\u{FFFD}" : "\\ufffd"; }
                else { return __mec_jfail($st, JSON_ERROR_UTF8, "null"); }
            } else {
                $len = $u[1];
                $cp = $u[0];
                if (!($f & JSON_UNESCAPED_UNICODE) || (($cp == 0x2028 || $cp == 0x2029) && !($f & JSON_UNESCAPED_LINE_TERMINATORS))) {
                    if ($cp >= 0x10000) {
                        $cp -= 0x10000;
                        $esc = __mec_jhex(0xD800 | ($cp >> 10)) . __mec_jhex(0xDC00 | ($cp & 0x3FF));
                    } else {
                        $esc = __mec_jhex($cp);
                    }
                }
            }
        } elseif ($c == 34) { $esc = $f & JSON_HEX_QUOT ? "\\u0022" : "\\\""; }
        elseif ($c == 92) { $esc = "\\\\"; }
        elseif ($c == 47) { $esc = $f & JSON_UNESCAPED_SLASHES ? null : "\\/"; }
        elseif ($c == 8) { $esc = "\\b"; }
        elseif ($c == 12) { $esc = "\\f"; }
        elseif ($c == 10) { $esc = "\\n"; }
        elseif ($c == 13) { $esc = "\\r"; }
        elseif ($c == 9) { $esc = "\\t"; }
        elseif ($c < 32) { $esc = __mec_jhex($c); }
        elseif ($c == 60 && ($f & JSON_HEX_TAG)) { $esc = "\\u003C"; }
        elseif ($c == 62 && ($f & JSON_HEX_TAG)) { $esc = "\\u003E"; }
        elseif ($c == 38 && ($f & JSON_HEX_AMP)) { $esc = "\\u0026"; }
        elseif ($c == 39 && ($f & JSON_HEX_APOS)) { $esc = "\\u0027"; }
        if ($esc !== null) {
            $out .= substr($s, $from, $i - $from) . $esc;
            $from = $i + $len;
        }
        $i += $len;
    }
    return $out . substr($s, $from) . "\"";
}
function __mec_jhex($u) { return "\\u" . str_pad(dechex($u), 4, "0", STR_PAD_LEFT); }

// json_decode: a recursive descent over the bytes, to php_json_scanner.re's
// token rules. $st carries the text, the position, the depth limit, the flags
// and the first error; any error makes the answer null.
function json_decode($json, $assoc = null, $depth = 512, $flags = 0) {
    if ($depth <= 0) { throw new ValueError("json_decode(): Argument #3 (\$depth) must be greater than 0"); }
    $st = new stdClass();
    $st->s = (string)$json;
    $st->n = strlen($st->s);
    $st->i = 0;
    $st->err = 0;
    $st->depth = $depth;
    $st->flags = $flags;
    $st->assoc = $assoc === null ? ($flags & JSON_OBJECT_AS_ARRAY) != 0 : (bool)$assoc;
    $v = null;
    if ($st->n == 0) { $st->err = JSON_ERROR_SYNTAX; }
    else {
        $v = __mec_jdec($st, 0);
        if ($st->err == 0) {
            __mec_jws($st);
            if ($st->i < $st->n) { __mec_jbad($st); }
        }
    }
    __mec_jsondone($st->err, $flags);
    return $st->err == 0 ? $v : null;
}
function __mec_jws($st) {
    while ($st->i < $st->n) {
        $c = $st->s[$st->i];
        if ($c !== " " && $c !== "\t" && $c !== "\n" && $c !== "\r") { return; }
        $st->i++;
    }
}
// The scanner's verdict on an unexpected byte (or the end of the text): NUL is a
// control character, a byte that starts no UTF-8 character is malformed UTF-8,
// anything else a syntax error.
function __mec_jbad($st) {
    $c = $st->i < $st->n ? ord($st->s[$st->i]) : -1;
    if ($c == 0) { $st->err = JSON_ERROR_CTRL_CHAR; }
    elseif ($c >= 0x80 && __mec_utf8at($st->s, $st->i, $st->n) === null) { $st->err = JSON_ERROR_UTF8; }
    else { $st->err = JSON_ERROR_SYNTAX; }
    return null;
}
function __mec_jdec($st, $level) {
    __mec_jws($st);
    if ($st->i >= $st->n) { return __mec_jbad($st); }
    $c = $st->s[$st->i];
    if ($c === "{" || $c === "[") {
        if ($level >= $st->depth) { return __mec_jfail($st, JSON_ERROR_DEPTH, null); }
        $st->i++;
        return $c === "[" ? __mec_jdecarr($st, $level + 1) : __mec_jdecobj($st, $level + 1);
    }
    if ($c === "\"") { return __mec_jdecstr($st); }
    if ($c === "-" || ($c >= "0" && $c <= "9")) { return __mec_jdecnum($st); }
    foreach (["true" => true, "false" => false, "null" => null] as $word => $val) {
        if (substr($st->s, $st->i, strlen($word)) === $word) {
            $st->i += strlen($word);
            return $val;
        }
    }
    return __mec_jbad($st);
}
// The byte after an element: true for the closing $close, false for a comma,
// null (with the error set) for anything else.
function __mec_jsep($st, $close) {
    __mec_jws($st);
    if ($st->i < $st->n && $st->s[$st->i] === $close) { $st->i++; return true; }
    if ($st->i < $st->n && $st->s[$st->i] === ",") { $st->i++; return false; }
    return __mec_jbad($st);
}
function __mec_jdecarr($st, $level) {
    $out = [];
    __mec_jws($st);
    if ($st->i < $st->n && $st->s[$st->i] === "]") { $st->i++; return $out; }
    while (true) {
        $out[] = __mec_jdec($st, $level);
        if ($st->err != 0) { return null; }
        $end = __mec_jsep($st, "]");
        if ($end === null) { return null; }
        if ($end) { return $out; }
    }
}
function __mec_jdecobj($st, $level) {
    $out = $st->assoc ? [] : new stdClass();
    __mec_jws($st);
    if ($st->i < $st->n && $st->s[$st->i] === "}") { $st->i++; return $out; }
    while (true) {
        __mec_jws($st);
        if ($st->i >= $st->n || $st->s[$st->i] !== "\"") { return __mec_jbad($st); }
        $key = __mec_jdecstr($st);
        if ($st->err != 0) { return null; }
        __mec_jws($st);
        if ($st->i >= $st->n || $st->s[$st->i] !== ":") { return __mec_jbad($st); }
        $st->i++;
        $v = __mec_jdec($st, $level);
        if ($st->err != 0) { return null; }
        if ($st->assoc) { $out[$key] = $v; }
        elseif ($key !== "" && $key[0] === "\0") { return __mec_jfail($st, JSON_ERROR_INVALID_PROPERTY_NAME, null); }
        else { $out->{$key} = $v; }
        $end = __mec_jsep($st, "}");
        if ($end === null) { return null; }
        if ($end) { return $out; }
    }
}
// -?(0|[1-9][0-9]*)(.[0-9]+)?([eE][+-]?[0-9]+)?: an int when it has neither a
// fraction nor an exponent and fits in 64 bits, otherwise a float (or the digits
// themselves under JSON_BIGINT_AS_STRING).
function __mec_jdecnum($st) {
    $s = $st->s;
    $n = $st->n;
    $start = $st->i;
    $i = $s[$start] === "-" ? $start + 1 : $start;
    $digits = __mec_jdigits($s, $i, $n);
    if ($digits == 0) {
        $st->i = $i;
        return __mec_jbad($st);
    }
    $i += $s[$i] === "0" ? 1 : $digits;
    $isInt = true;
    $fd = $i < $n && $s[$i] === "." ? __mec_jdigits($s, $i + 1, $n) : 0;
    if ($fd > 0) {
        $i += 1 + $fd;
        $isInt = false;
    }
    if ($i < $n && ($s[$i] === "e" || $s[$i] === "E")) {
        $j = $i + 1;
        if ($j < $n && ($s[$j] === "+" || $s[$j] === "-")) { $j++; }
        $ed = __mec_jdigits($s, $j, $n);
        if ($ed > 0) {
            $i = $j + $ed;
            $isInt = false;
        }
    }
    $st->i = $i;
    $tok = substr($s, $start, $i - $start);
    if (!$isInt) { return (float)$tok; }
    $mag = $tok[0] === "-" ? substr($tok, 1) : $tok;
    if (strlen($mag) > 19 || (strlen($mag) == 19 && strcmp($mag, "9223372036854775808") >= 0 && $tok !== "-9223372036854775808")) {
        return $st->flags & JSON_BIGINT_AS_STRING ? $tok : (float)$tok;
    }
    return (int)$tok;
}
function __mec_jdigits($s, $i, $n) {
    $k = $i;
    while ($k < $n && $s[$k] >= "0" && $s[$k] <= "9") { $k++; }
    return $k - $i;
}
// A string literal from its opening quote: the escapes decoded, \u to UTF-8 (a
// surrogate pair to one character); a raw control character, an unterminated
// literal or an unpaired surrogate is an error, and so is malformed UTF-8 unless
// it is ignored or substituted.
function __mec_jdecstr($st) {
    $s = $st->s;
    $n = $st->n;
    $i = $st->i + 1;
    $from = $i;
    $out = "";
    $short = ["\"" => "\"", "\\" => "\\", "/" => "/", "b" => "\x08", "f" => "\f", "n" => "\n", "r" => "\r", "t" => "\t"];
    while (true) {
        if ($i >= $n) { return __mec_jfail($st, JSON_ERROR_CTRL_CHAR, null); }
        $c = ord($s[$i]);
        if ($c == 34) {
            $st->i = $i + 1;
            return $out . substr($s, $from, $i - $from);
        }
        if ($c < 32) { return __mec_jfail($st, JSON_ERROR_CTRL_CHAR, null); }
        if ($c >= 0x80) {
            $u = __mec_utf8at($s, $i, $n);
            if ($u !== null) {
                $i += $u[1];
                continue;
            }
            if (!($st->flags & (JSON_INVALID_UTF8_IGNORE | JSON_INVALID_UTF8_SUBSTITUTE))) { return __mec_jfail($st, JSON_ERROR_UTF8, null); }
            $out .= substr($s, $from, $i - $from) . ($st->flags & JSON_INVALID_UTF8_IGNORE ? "" : "\u{FFFD}");
            $i++;
            $from = $i;
            continue;
        }
        if ($c != 92) {
            $i++;
            continue;
        }
        $out .= substr($s, $from, $i - $from);
        $e = substr($s, $i + 1, 1);
        if (isset($short[$e])) {
            $out .= $short[$e];
            $i += 2;
        } elseif ($e === "u") {
            $cp = __mec_jhex4($s, $i + 2, $n);
            if ($cp < 0) { return __mec_jfail($st, JSON_ERROR_SYNTAX, null); }
            $i += 6;
            if ($cp >= 0xD800 && $cp <= 0xDBFF) {
                $lo = substr($s, $i, 2) === "\\u" ? __mec_jhex4($s, $i + 2, $n) : -1;
                if ($lo < 0xDC00 || $lo > 0xDFFF) { return __mec_jfail($st, JSON_ERROR_UTF16, null); }
                $cp = 0x10000 + (($cp - 0xD800) << 10) + ($lo - 0xDC00);
                $i += 6;
            } elseif ($cp >= 0xDC00 && $cp <= 0xDFFF) {
                return __mec_jfail($st, JSON_ERROR_UTF16, null);
            }
            $out .= __mec_utf8chr($cp);
        } else {
            return __mec_jfail($st, JSON_ERROR_SYNTAX, null);
        }
        $from = $i;
    }
}
// Four hex digits at byte $i as a number, or -1.
function __mec_jhex4($s, $i, $n) {
    if ($i + 4 > $n) { return -1; }
    $v = 0;
    for ($k = $i; $k < $i + 4; $k++) {
        $d = strpos("0123456789abcdef", strtolower($s[$k]));
        if ($d === false) { return -1; }
        $v = $v * 16 + $d;
    }
    return $v;
}

// ----- exceptions ------------------------------------------------------------
// Exception::__toString (Zend/zend_exceptions.c), which Error's aliases: $e and
// then each previous one, as "Class: message in file:line\nStack trace:\n" and the
// trace, without ": message" when the message is empty; a TypeError's or an
// ArgumentCountError's message naming its caller (", called in ") gets " and
// defined" before " in file:line". Each link goes in front
// of the text so far, joined by "\n\nNext ", so the innermost prints first; a
// link the walk already met ends it.
function __mec_excstr($e) {
    $str = "";
    $seen = [];
    while ($e instanceof Throwable && !in_array($e, $seen, true)) {
        $seen[] = $e;
        $message = (string) $e->getMessage();
        $k = get_class($e);
        if (($k === "TypeError" || $k === "ArgumentCountError") && strpos($message, ", called in ") !== false) {
            $message .= " and defined";
        }
        $str = $k . ($message === "" ? "" : ": " . $message) . " in " . $e->getFile() . ":" . $e->getLine()
             . "\nStack trace:\n" . $e->getTraceAsString() . ($str === "" ? "" : "\n\nNext " . $str);
        $e = $e->getPrevious();
    }
    return $str;
}
// A Throwable's trace (zend_fetch_debug_backtrace): $raw lists the calls in
// progress innermost first as the engines record them, each [file, line,
// function, class, kind, lib, $this, current, passed, rest]. file and line are
// where the call stands, "" and 0 for a call the library or the engine made. kind
// is 0 for a function, 1 a static method, 2 an instance method, 3 a closure; lib
// marks an internal function: the library's, an engine native or a Generator
// method. current holds the parameters' values now, passed the arguments as the call
// handed them over up to a variadic parameter, and rest the variadic ones packed
// into one array (null without one). A library function called from the library
// is no frame of its own, as an internal function calling another is none. The
// arguments are the parameters' current values for the arguments passed, an
// internal function's as passed, and then the variadic ones, a named one under
// its name (debug_backtrace_get_args).
function __mec_trace($raw) {
    $out = [];
    foreach ($raw as $r) {
        $lib = $r[5];
        if ($lib && $r[0] === "") { continue; }
        $f = [];
        if ($r[0] !== "") { $f["file"] = $r[0]; $f["line"] = $r[1]; }
        $f["function"] = $r[2];
        $kind = $r[4];
        if ($r[3] !== "" && $kind != 0) {
            $f["class"] = $r[3];
            $f["type"] = $kind == 2 || ($kind == 3 && is_object($r[6])) ? "->" : "::";
        }
        $cur = $r[7];
        $passed = $r[8];
        $args = [];
        $n = count($passed);
        for ($i = 0; $i < $n; $i++) {
            $args[] = !$lib && $i < count($cur) ? $cur[$i] : $passed[$i];
        }
        if (is_array($r[9])) {
            foreach ($r[9] as $k => $v) {
                if (is_string($k)) { $args[$k] = $v; } else { $args[] = $v; }
            }
        }
        $f["args"] = $args;
        $out[] = $f;
    }
    return $out;
}
// Exception::getTraceAsString (zend_trace_to_string with {main}): one line per
// frame, "#n file(line): " or "#n [internal function]: ", the class and call
// type, the function and its arguments (_build_trace_args), then "#n {main}".
function __mec_tracestr($trace) {
    $s = "";
    $n = 0;
    foreach ($trace as $f) {
        $s .= "#" . $n . " " . (isset($f["file"]) ? $f["file"] . "(" . $f["line"] . "): " : "[internal function]: ")
            . ($f["class"] ?? "") . ($f["type"] ?? "") . $f["function"] . "(";
        $sep = "";
        foreach ($f["args"] ?? [] as $k => $v) {
            $s .= $sep . (is_string($k) ? $k . ": " : "") . __mec_tracearg($v);
            $sep = ", ";
        }
        $s .= ")\n";
        $n++;
    }
    return $s . "#" . $n . " {main}";
}
// One argument as smart_str_append_zval shows it: a string quoted, its first
// exception_string_param_max_len (15) bytes escaped and "..." after a longer
// one, a float at precision 14 with a fraction, an enum case by name, and
// otherwise Array or Object(class).
function __mec_tracearg($v) {
    if ($v === null) { return "NULL"; }
    if (is_bool($v)) { return $v ? "true" : "false"; }
    if (is_int($v)) { return (string)$v; }
    if (is_float($v)) {
        $s = (string)$v;
        return is_finite($v) && strpos($s, ".") === false ? $s . ".0" : $s;
    }
    if (is_string($v)) {
        $len = strlen($v);
        $m = $len > 15 ? 15 : $len;
        $out = "'";
        $from = 0;
        for ($i = 0; $i < $m; $i++) {
            $c = ord($v[$i]);
            if ($c >= 32 && $c != 92 && $c <= 126) { continue; }
            $out .= substr($v, $from, $i - $from) . "\\" . __mec_traceesc($c);
            $from = $i + 1;
        }
        return $out . substr($v, $from, $m - $from) . "'" . ($len > 15 ? "..." : "");
    }
    if ($v instanceof UnitEnum) { return get_class($v) . "::" . $v->name; }
    if (is_array($v)) { return "Array"; }
    return "Object(" . get_class($v) . ")";
}
// The escape of byte $c after its backslash (smart_str_append_escaped).
function __mec_traceesc($c) {
    switch ($c) {
        case 10: return "n";
        case 13: return "r";
        case 9: return "t";
        case 12: return "f";
        case 11: return "v";
        case 92: return "\\";
        case 27: return "e";
    }
    return "x" . strtoupper(str_pad(dechex($c), 2, "0", STR_PAD_LEFT));
}
// An exception nobody caught (zend_exception_error): the fatal error
// "Uncaught " . (string) $e . "\n  thrown" at the exception's file and line, or
// at "Unknown" and line 0 when it has no file. The answer is the error's text
// with its location, which php_error_cb (main/main.c) logs on stderr as
// "PHP Fatal error:  " . $text, or, with display_errors=1 (php-cli's defaults),
// displays through __mec_fatalshow instead.
function __mec_uncaught($e) {
    $file = $e->getFile();
    $line = $e->getLine();
    if ($file === "") { $file = "Unknown"; $line = 0; }
    return "Uncaught " . $e->__toString() . "\n  thrown in " . $file . " on line " . $line;
}
// A fatal error's text displayed on stdout, if error_reporting has E_ERROR.
function __mec_fatalshow($text) {
    if ((__mec_erlevel() & 1) != 0) { echo "\nFatal error: ", $text, "\n"; }
}

// ----- objects ---------------------------------------------------------------
// An object's handle, var_dump's #id; a closure literal keeps its handle too, and
// any other Closure answers 0.
function spl_object_id($object) {
    if (!is_object($object)) {
        throw new TypeError("spl_object_id(): Argument #1 (\$object) must be of type object, " . __mec_vname($object) . " given");
    }
    return __mec_oid($object);
}

// ----- runtime configuration -------------------------------------------------
// Of the ini settings only precision is modelled: the digits echo, (string) and
// interpolation give a float. Any other option reads as one that does not exist.
// The value is read the way the setting's handler reads it, as an integer, and
// one below -1 is refused.
function ini_set($option, $value) {
    if ($option !== "precision") { return false; }
    $p = (int) $value;
    if ($p < -1) { return false; }
    return (string) __mec_precision($p);
}
function ini_get($option) {
    if ($option !== "precision") { return false; }
    return (string) __mec_precision(null);
}

// An object's handle as spl_object_hash spells it: 16 hex digits, then 16 zeros
// (php_spl_object_hash).
function spl_object_hash($object) {
    if (!is_object($object)) {
        throw new TypeError("spl_object_hash(): Argument #1 (\$object) must be of type object, " . __mec_vname($object) . " given");
    }
    return sprintf("%016x0000000000000000", __mec_oid($object));
}

// ----- diagnostics -----------------------------------------------------------
// A grammar compiled with display_errors=1 raises the program's warnings,
// notices and deprecations through __mec_diag, and checks the operations that
// raise them with the __mec_*chk functions before it performs them; $file and
// $line are the site's, or null and 0 with lines=0. With display_errors=0 none of
// this is compiled in, and error_get_last() never has an error to answer.
//
// EG(error_reporting), E_ALL at startup: answers it, and sets it first unless
// $level is null.
function __mec_erlevel($level = null) {
    static $cur = 30719;
    $old = $cur;
    if ($level !== null) { $cur = $level; }
    return $old;
}
// PG(last_error_*): answers the record, and sets it first unless $error is false.
function __mec_erlast($error = false) {
    static $last = null;
    if ($error !== false) { $last = $error; }
    return $last;
}
// php_error_cb (main/main.c) at the CLI's defaults: the error is recorded, then,
// if error_reporting has its bit, displayed on stdout as
// "%s\n%s: %s in %s on line %d\n%s" with an empty prepend and append string. A
// site without a location shows none.
function __mec_diag($type, $message, $file = null, $line = 0) {
    if (__mec_diagbuf(2, [$type, $message, $file, $line])) { return; }
    __mec_erlast(["type" => $type, "message" => $message, "file" => $file ?? "Unknown", "line" => $line]);
    if ((__mec_erlevel() & $type) == 0) { return; }
    $label = $type == 8 ? "Notice" : ($type == 8192 ? "Deprecated" : "Warning");
    echo "\n", $label, ": ", $message, $file === null ? "" : " in " . $file . " on line " . $line, "\n";
}
// The diagnostics of an argument the interpreter reads before it knows whether
// the parameter takes it by reference: $op 0 starts keeping them, 1 stops and
// answers the kept ones (each __mec_diag's arguments), and 2 keeps $entry if
// keeping, answering whether it did. Starts nest.
function __mec_diagbuf($op, $entry = null) {
    static $kept = [];
    if ($op === 0) { $kept[] = []; return null; }
    if ($op === 1) { return array_pop($kept); }
    $n = count($kept);
    if ($n === 0) { return false; }
    $kept[$n - 1][] = $entry;
    return true;
}
// Raises the kept diagnostics of an argument passed by value after all.
function __mec_diagreplay($kept) {
    foreach ($kept as $e) { __mec_diag($e[0], $e[1], $e[2], $e[3]); }
}
// After ->bindTo(), ->call() or Closure::bind(): the warning of a binding
// zend_valid_closure_binding refused, which the engine keeps (__mec_bindwhy).
function __mec_bindchk($file = null, $line = 0) {
    $why = __mec_bindwhy();
    if ($why !== "") { __mec_diag(2, $why, $file, $line); }
}
// After count() with a mode: php_count_recursive's warning for each array it
// reached again (__mec_countrecs).
function __mec_countchk($file = null, $line = 0) {
    for ($n = __mec_countrecs(); $n > 0; $n--) { __mec_diag(2, "count(): Recursion detected", $file, $line); }
}
// error_reporting() under display_errors=1: the previous level.
function __mec_erreport($error_level = null) {
    return __mec_erlevel($error_level === null ? null : (int) $error_level);
}
function error_get_last() { return __mec_erlast(); }
function error_clear_last() { __mec_erlast(null); }
// The levels the '@' operands being evaluated restore, innermost last: $op 0
// pushes $level, 1 pops one, 2 answers how many there are.
function __mec_silenced($op, $level = null) {
    static $saved = [];
    if ($op === 0) { $saved[] = $level; return null; }
    if ($op === 1) { return array_pop($saved); }
    return count($saved);
}
// ZEND_BEGIN_SILENCE: '@' leaves only the fatal errors (E_FATAL_ERRORS, 4437) on.
function __mec_silence() {
    $old = __mec_erlevel();
    __mec_silenced(0, $old);
    if (($old & ~4437) != 0) { __mec_erlevel($old & 4437); }
}
// ZEND_END_SILENCE: the level is restored unless the operand itself turned a
// non-fatal error back on.
function __mec_unsilence() {
    $old = __mec_silenced(1);
    if ((__mec_erlevel() & ~4437) == 0 && ($old & ~4437) != 0) { __mec_erlevel($old); }
}
// An exception caught, or a finally entered, ends the '@' operands it left
// (ZEND_LIVE_SILENCE in cleanup_live_vars) down to the $depth its try began at.
function __mec_unsilenceto($depth) {
    while (__mec_silenced(2) > $depth) { __mec_unsilence(); }
}
// The Deprecated of a float that is not an integer used as one
// (zend_incompatible_double_to_long_error); answers the int.
function __mec_f2ichk($v, $file = null, $line = 0) {
    $i = (int) $v;
    if ((float) $i !== $v) {
        __mec_diag(8192, "Implicit conversion from float " . __mec_serfloat($v, "E") . " to int loses precision", $file, $line);
    }
    return $i;
}
// The operand of ++ ($inc) or -- (increment_function, decrement_function): a bool,
// a null decremented, and a string that is not numeric.
function __mec_incdecchk($v, $inc, $file = null, $line = 0) {
    $what = $inc ? "Increment" : "Decrement";
    if (is_bool($v) || ($v === null && !$inc)) {
        __mec_diag(2, $what . " on type " . ($v === null ? "null" : "bool") . " has no effect, this will change in the next major version of PHP", $file, $line);
    } elseif (!is_string($v) || is_numeric($v)) {
        return;
    } elseif ($inc && ($v === "" || preg_match('/^[A-Za-z0-9]*$/', $v) !== 1)) {
        __mec_diag(8192, "Increment on non-alphanumeric string is deprecated", $file, $line);
    } elseif (!$inc) {
        __mec_diag(8192, $v === "" ? "Decrement on empty string is deprecated as non-numeric" : "Decrement on non-numeric string has no effect and is deprecated", $file, $line);
    }
}
// An operand of %, the bitwise operators and the shifts, as zendi_try_get_long
// takes it: a float that is not an integer, and a numeric string with trailing
// data or with a float value that is not one. A string that is not numeric at all
// is the operator's own TypeError.
function __mec_intchk($v, $file = null, $line = 0) {
    if (is_float($v)) { __mec_f2ichk($v, $file, $line); return; }
    if (!is_string($v) || is_numeric($v)) {
        if (is_string($v) && (float) $v != (int) $v) {
            __mec_diag(8192, "Implicit conversion from float-string \"" . $v . "\" to int loses precision", $file, $line);
        }
        return;
    }
    if (preg_match('/^[ \t\n\r\v\f]*[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)([eE][+-]?[0-9]+)?/', $v, $m) !== 1) { return; }
    __mec_diag(2, "A non-numeric value encountered", $file, $line);
    if ((float) $m[0] != (int) $m[0]) {
        __mec_diag(8192, "Implicit conversion from float-string \"" . $v . "\" to int loses precision", $file, $line);
    }
}
// An operand of +, -, *, / and **: a numeric string with trailing data
// (_zendi_try_convert_scalar_to_number).
function __mec_numchk($v, $file = null, $line = 0) {
    if (is_string($v) && !is_numeric($v)
        && preg_match('/^[ \t\n\r\v\f]*[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)/', $v) === 1) {
        __mec_diag(2, "A non-numeric value encountered", $file, $line);
    }
}
// The operand of an (int), (float) or (string) cast ($kind as written), intval,
// floatval or strval: an array as a string (__mec_strchk), and an object as a
// number, which no cast handler converts (convert_object_to_type) and reads as 1.
function __mec_castchk($kind, $v, $file = null, $line = 0) {
    if ($kind === "string" || $kind === "binary") {
        __mec_strchk($v, $file, $line);
    } elseif (is_object($v)) {
        $to = $kind === "int" || $kind === "integer" ? "int" : "float";
        __mec_diag(2, "Object of class " . get_class($v) . " could not be converted to " . $to, $file, $line);
    }
}
// A value echo, print, an interpolation or '.' converts to a string: an array
// (zval_get_string).
function __mec_strchk($v, $file = null, $line = 0) {
    if (is_array($v)) { __mec_diag(2, "Array to string conversion", $file, $line); }
}
// The operands of binary operator $op, or of ~ ($b null), left first: as ints for
// %, the shifts and a bitwise operator not given two strings, as numbers for the
// arithmetic ones, and as strings for the concatenation (__mec_strchk).
function __mec_opchk($op, $a, $b, $file = null, $line = 0) {
    if ($op === ".") {
        __mec_strchk($a, $file, $line);
        __mec_strchk($b, $file, $line);
    } elseif ($op === "+" || $op === "-" || $op === "*" || $op === "/" || $op === "**") {
        __mec_numchk($a, $file, $line);
        __mec_numchk($b, $file, $line);
    } elseif ($op === "~") {
        if (is_float($a)) { __mec_f2ichk($a, $file, $line); }
    } elseif ($op === "%" || $op === "<<" || $op === ">>" || !is_string($a) || !is_string($b)) {
        __mec_intchk($a, $file, $line);
        __mec_intchk($b, $file, $line);
    }
}
// Reading $container[$offset] (zend_fetch_dimension_address_read_R): a missing
// key of an array, a cast or out-of-range offset of a string, and a scalar
// container. An offset the read itself refuses is left to it.
function __mec_dimchk($container, $offset, $file = null, $line = 0) {
    if (is_array($container)) {
        __mec_keychk($container, $offset, $file, $line);
        return;
    }
    if (is_string($container)) {
        if (is_float($offset) || is_bool($offset) || $offset === null) {
            __mec_diag(2, "String offset cast occurred", $file, $line);
            $offset = (int) $offset;
        } elseif (is_string($offset)) {
            // An integer string is the offset, and so is one's integer prefix,
            // with a warning; anything else is the read's own Error.
            if (is_numeric($offset)) {
                if (!is_int($offset + 0)) { return; }
            } elseif (preg_match('/^[ \t\n\r\v\f]*[+-]?[0-9]+(\.[0-9]*)?([eE][+-]?[0-9]+)?/', $offset, $m) === 1 && is_int($m[0] + 0)) {
                __mec_diag(2, "Illegal string offset \"" . $offset . "\"", $file, $line);
            } else { return; }
            $offset = (int) $offset;
        } elseif (!is_int($offset)) { return; }
        $n = strlen($container);
        if ($offset >= $n || $offset < -$n) { __mec_diag(2, "Uninitialized string offset " . $offset, $file, $line); }
        return;
    }
    if (!is_object($container)) {
        __mec_diag(2, "Trying to access array offset on " . __mec_vname($container), $file, $line);
    }
}
// The key of an array element that is written (zend_fetch_dimension_address_inner
// for BP_VAR_W, ZEND_ASSIGN_DIM), unset (ZEND_UNSET_DIM), fetched for '??'
// (slow_index_convert), given in an array literal (ZEND_ADD_ARRAY_ELEMENT) or to
// array_key_exists (zend_array_key_exists_fast): a float that is not an integer.
function __mec_fkeychk($container, $offset, $file = null, $line = 0) {
    if (is_float($offset) && is_array($container)) { __mec_f2ichk($offset, $file, $line); }
}
// The key of an isset() or empty() step: a float that is not an integer, of an
// array (zend_find_array_dim_slow) and, at the $last step, of a string too
// (zend_isset_dim_slow's strict zval_get_long_ex).
function __mec_issetchk($container, $offset, $last, $file = null, $line = 0) {
    if (is_float($offset) && (is_array($container) || ($last && is_string($container)))) {
        __mec_f2ichk($offset, $file, $line);
    }
}
// The key of an array element that is read, also by a compound assignment or
// ++/-- (zend_undefined_offset, zend_undefined_index): a float that is not an
// integer, and a key the array does not have.
function __mec_keychk($container, $offset, $file = null, $line = 0) {
    if (!is_array($container)) { return; }
    if (is_float($offset)) { $offset = __mec_f2ichk($offset, $file, $line); }
    elseif (is_bool($offset)) { $offset = (int) $offset; }
    elseif ($offset === null) { $offset = ""; }
    elseif (!is_int($offset) && !is_string($offset)) { return; }
    if (array_key_exists($offset, $container)) { return; }
    if (is_string($offset) && (string) (int) $offset !== $offset) {
        __mec_diag(2, "Undefined array key \"" . $offset . "\"", $file, $line);
    } else {
        __mec_diag(2, "Undefined array key " . $offset, $file, $line);
    }
}
// Reading $object->$name: a container that is no object
// (zend_wrong_property_read), then __mec_ownpropchk.
function __mec_propchk($object, $name, $file = null, $line = 0) {
    if (!is_object($object)) {
        __mec_diag(2, "Attempt to read property \"" . $name . "\" on " . __mec_vname($object), $file, $line);
    }
    __mec_ownpropchk($object, $name, $file, $line);
}
// A property an object does not have and no __get supplies, read, also by a
// compound assignment or ++/-- (zend_std_read_property).
function __mec_ownpropchk($object, $name, $file = null, $line = 0) {
    if (is_object($object) && !property_exists($object, $name) && !method_exists($object, "__get")) {
        __mec_diag(2, "Undefined property: " . get_class($object) . "::\$" . $name, $file, $line);
    }
}

// ----- SPL: the iterator functions, class_implements ------------------------
function iterator_to_array($iterator, $preserve_keys = true) {
    if (is_array($iterator)) { return $preserve_keys ? $iterator : array_values($iterator); }
    if (!($iterator instanceof Traversable)) {
        throw new TypeError("iterator_to_array(): Argument #1 (\$iterator) must be of type Traversable|array, " . __mec_vname($iterator) . " given");
    }
    $out = [];
    foreach ($iterator as $k => $v) {
        if ($preserve_keys) { $out[$k] = $v; } else { $out[] = $v; }
    }
    return $out;
}
function iterator_count($iterator) {
    if (is_array($iterator)) { return count($iterator); }
    if (!($iterator instanceof Traversable)) {
        throw new TypeError("iterator_count(): Argument #1 (\$iterator) must be of type Traversable|array, " . __mec_vname($iterator) . " given");
    }
    $n = 0;
    foreach ($iterator as $v) { $n++; }
    return $n;
}
// The callback runs once per element until it answers a falsy value; the count
// includes that last call (spl_iterator_func_apply).
function iterator_apply($iterator, $callback, $args = null) {
    $n = 0;
    foreach ($iterator as $v) {
        $n++;
        if (!($args === null ? $callback() : $callback(...$args))) { break; }
    }
    return $n;
}
// class_implements(): the class's interface list (__mec_ifacelist), each name
// keyed by itself; false for a string that names no class.
function class_implements($object_or_class, $autoload = true) {
    if (!is_object($object_or_class) && !is_string($object_or_class)) {
        throw new TypeError("class_implements(): Argument #1 (\$object_or_class) must be of type object|string, " . __mec_vname($object_or_class) . " given");
    }
    $list = __mec_ifacelist($object_or_class);
    if ($list === false) { return false; }
    $out = [];
    foreach ($list as $i) { $out[$i] = $i; }
    return $out;
}
// class_parents(): the parent chain, nearest first, each name keyed by itself.
function class_parents($object_or_class, $autoload = true) {
    if (!is_object($object_or_class) && !is_string($object_or_class)) {
        throw new TypeError("class_parents(): Argument #1 (\$object_or_class) must be of type object|string, " . __mec_vname($object_or_class) . " given");
    }
    if (__mec_ifaces($object_or_class) === false) { return false; }
    $out = [];
    for ($p = get_parent_class($object_or_class); $p !== false; $p = get_parent_class($p)) { $out[$p] = $p; }
    return $out;
}
// The interface list of a class or an interface, ce->interfaces: the parent's,
// then the ones the class names (a parent's skipped), then for each of those the
// ones it extends, taken from the END of its own list; a class that names none
// takes its parent's whole list, also from the end (zend_do_implement_interfaces,
// zend_do_inherit_interfaces). False when $c denotes neither.
function __mec_ifacelist($c) {
    $own = __mec_ifaces($c);
    if ($own === false) {
        if (is_string($c) && interface_exists($c)) { return array_slice(__mec_coreiface($c), 1); }
        return false;
    }
    $p = get_parent_class($c);
    $list = $p === false ? [] : __mec_ifacelist($p);
    if (count($own) == 0) { return array_reverse($list); }
    $named = [];
    foreach ($own as $i) {
        $nm = __mec_ifaces($i) === false ? __mec_coreiface($i)[0] : $i;
        if (!in_array($nm, $list, true)) {
            $list[] = $nm;
            $named[] = $nm;
        }
    }
    foreach ($named as $nm) {
        $sub = __mec_ifacelist($nm);
        for ($k = count($sub) - 1; $k >= 0; $k--) {
            if (!in_array($sub[$k], $list, true)) { $list[] = $sub[$k]; }
        }
    }
    return $list;
}
// The engine's own interfaces have no declaration: each one's spelling, then the
// list of the ones it extends.
function __mec_coreiface($name) {
    switch (strtolower($name)) {
        case "traversable": return ["Traversable"];
        case "iterator": return ["Iterator", "Traversable"];
        case "iteratoraggregate": return ["IteratorAggregate", "Traversable"];
        case "arrayaccess": return ["ArrayAccess"];
        case "countable": return ["Countable"];
        case "serializable": return ["Serializable"];
        case "stringable": return ["Stringable"];
        case "throwable": return ["Throwable", "Stringable"];
        case "jsonserializable": return ["JsonSerializable"];
        case "unitenum": return ["UnitEnum"];
        case "backedenum": return ["BackedEnum", "UnitEnum"];
    }
    return [$name];
}

// ----- SPL: the data structures and their exceptions -------------------------
// Written on arrays. var_dump shows each one's private storage property, named
// as its get_debug_info handler names it; an iteration position is the
// array's own internal pointer. An implements list spells out the interfaces
// the ones it names extend, in the order an engine class registers them
// (zend_class_implements: each one, then the ones it extends from the end of
// its own list), which is what class_implements answers.
interface SeekableIterator extends Iterator {
    public function seek(int $offset): void;
}
class OutOfBoundsException extends RuntimeException {}
class OutOfRangeException extends LogicException {}
class UnexpectedValueException extends RuntimeException {}
class UnderflowException extends RuntimeException {}
class OverflowException extends RuntimeException {}
class RangeException extends RuntimeException {}
class LengthException extends LogicException {}
class DomainException extends LogicException {}
class BadFunctionCallException extends LogicException {}
class BadMethodCallException extends BadFunctionCallException {}
class ArrayIterator implements SeekableIterator, Traversable, Iterator, ArrayAccess, Serializable, Countable {
    const STD_PROP_LIST = 1;
    const ARRAY_AS_PROPS = 2;
    private $storage;
    public function __construct($array = [], int $flags = 0) { $this->storage = ArrayObject::__mec_array($array); }
    // True for a key holding null too, unlike isset($it[$key]), which the engine
    // answers through __mec_isset while offsetExists is not overridden
    // (spl_array_has_dimension_ex).
    public function offsetExists($key): bool { return array_key_exists($key, $this->storage); }
    public function __mec_isset($key): bool { return isset($this->storage[$key]); }
    public function offsetGet($key): mixed { return $this->storage[$key]; }
    public function offsetSet($key, $value): void {
        if ($key === null) { $this->storage[] = $value; } else { $this->storage[$key] = $value; }
    }
    public function offsetUnset($key): void { unset($this->storage[$key]); }
    public function append($value): void { $this->storage[] = $value; }
    public function getArrayCopy(): array { return $this->storage; }
    public function count(): int { return count($this->storage); }
    public function getFlags(): int { return 0; }
    public function setFlags(int $flags): void {}
    public function asort(int $flags = SORT_REGULAR): bool { return asort($this->storage, $flags); }
    public function ksort(int $flags = SORT_REGULAR): bool { return ksort($this->storage, $flags); }
    public function uasort($callback): bool { return uasort($this->storage, $callback); }
    public function uksort($callback): bool { return uksort($this->storage, $callback); }
    public function natsort(): bool { return natsort($this->storage); }
    public function natcasesort(): bool { return natcasesort($this->storage); }
    public function rewind(): void { reset($this->storage); }
    public function valid(): bool { return key($this->storage) !== null; }
    public function current(): mixed { return key($this->storage) === null ? null : current($this->storage); }
    public function key(): string|int|null { return key($this->storage); }
    public function next(): void { next($this->storage); }
    public function seek(int $offset): void {
        if ($offset < 0 || $offset >= count($this->storage)) {
            throw new OutOfBoundsException("Seek position " . $offset . " is out of range");
        }
        reset($this->storage);
        for ($i = 0; $i < $offset; $i++) { next($this->storage); }
    }
}
class ArrayObject implements IteratorAggregate, Traversable, ArrayAccess, Serializable, Countable {
    const STD_PROP_LIST = 1;
    const ARRAY_AS_PROPS = 2;
    private $storage;
    public function __construct($array = [], int $flags = 0, string $iteratorClass = "ArrayIterator") {
        $this->storage = self::__mec_array($array);
    }
    // The array one is made over: an array, another one's array, or an object's
    // properties.
    public static function __mec_array($array) {
        if ($array instanceof ArrayObject || $array instanceof ArrayIterator) { return $array->getArrayCopy(); }
        if (is_object($array)) { return (array) $array; }
        if (!is_array($array)) {
            throw new TypeError("ArrayObject::__construct(): Argument #1 (\$array) must be of type array, " . __mec_vname($array) . " given");
        }
        return $array;
    }
    // As ArrayIterator's.
    public function offsetExists($key): bool { return array_key_exists($key, $this->storage); }
    public function __mec_isset($key): bool { return isset($this->storage[$key]); }
    public function offsetGet($key): mixed { return $this->storage[$key]; }
    public function offsetSet($key, $value): void {
        if ($key === null) { $this->storage[] = $value; } else { $this->storage[$key] = $value; }
    }
    public function offsetUnset($key): void { unset($this->storage[$key]); }
    public function append($value): void { $this->storage[] = $value; }
    public function getArrayCopy(): array { return $this->storage; }
    // Answers the array it replaces.
    public function exchangeArray($array): array {
        $old = $this->storage;
        $this->storage = self::__mec_array($array);
        return $old;
    }
    public function count(): int { return count($this->storage); }
    public function getFlags(): int { return 0; }
    public function setFlags(int $flags): void {}
    public function getIterator(): Iterator { return new ArrayIterator($this->storage); }
    public function getIteratorClass(): string { return "ArrayIterator"; }
    public function asort(int $flags = SORT_REGULAR): bool { return asort($this->storage, $flags); }
    public function ksort(int $flags = SORT_REGULAR): bool { return ksort($this->storage, $flags); }
    public function uasort($callback): bool { return uasort($this->storage, $callback); }
    public function uksort($callback): bool { return uksort($this->storage, $callback); }
    public function natsort(): bool { return natsort($this->storage); }
    public function natcasesort(): bool { return natcasesort($this->storage); }
}
// A list, iterated front to back, or back to front under IT_MODE_LIFO, and
// consumed under IT_MODE_DELETE (spl_dllist.c). flags also carries the bit 4
// that freezes the direction, which SplStack and SplQueue set; MEC_MODE is the
// mode a class iterates in even before its constructor has run.
class SplDoublyLinkedList implements Iterator, Traversable, Countable, ArrayAccess, Serializable {
    const IT_MODE_LIFO = 2;
    const IT_MODE_FIFO = 0;
    const IT_MODE_DELETE = 1;
    const IT_MODE_KEEP = 0;
    const MEC_MODE = 0;
    private $flags = 0;
    private $dllist = [];
    public function push($value): void { $this->dllist[] = $value; }
    public function unshift($value): void { array_unshift($this->dllist, $value); }
    public function pop(): mixed {
        if (count($this->dllist) == 0) { throw new RuntimeException("Can't pop from an empty datastructure"); }
        return array_pop($this->dllist);
    }
    public function shift(): mixed {
        if (count($this->dllist) == 0) { throw new RuntimeException("Can't shift from an empty datastructure"); }
        return array_shift($this->dllist);
    }
    public function top(): mixed {
        if (count($this->dllist) == 0) { throw new RuntimeException("Can't peek at an empty datastructure"); }
        return $this->dllist[count($this->dllist) - 1];
    }
    public function bottom(): mixed {
        if (count($this->dllist) == 0) { throw new RuntimeException("Can't peek at an empty datastructure"); }
        return $this->dllist[0];
    }
    public function isEmpty(): bool { return count($this->dllist) == 0; }
    public function count(): int { return count($this->dllist); }
    public function toArray(): array { return $this->dllist; }
    public function offsetExists($index): bool { return is_numeric($index) && $index >= 0 && $index < count($this->dllist); }
    public function offsetGet($index): mixed {
        $this->checkIndex("offsetGet", $index);
        return $this->dllist[$index];
    }
    public function offsetSet($index, $value): void {
        if ($index === null) {
            $this->dllist[] = $value;
            return;
        }
        $this->checkIndex("offsetSet", $index);
        $this->dllist[$index] = $value;
    }
    public function offsetUnset($index): void {
        $this->checkIndex("offsetUnset", $index);
        array_splice($this->dllist, $index, 1);
    }
    public function add($index, $value): void {
        if (!is_numeric($index) || $index < 0 || $index > count($this->dllist)) {
            throw new OutOfRangeException("SplDoublyLinkedList::add(): Argument #1 (\$index) is out of range");
        }
        array_splice($this->dllist, $index, 0, [$value]);
    }
    private function checkIndex($method, $index) {
        if (!is_numeric($index)) {
            throw new OutOfRangeException("SplDoublyLinkedList::" . $method . "(): Argument #1 (\$index) is an invalid offset");
        }
        if ($index < 0 || $index >= count($this->dllist)) {
            throw new OutOfRangeException("SplDoublyLinkedList::" . $method . "(): Argument #1 (\$index) is out of range");
        }
    }
    public function setIteratorMode(int $mode): int {
        $f = $this->flags | static::MEC_MODE;
        if (($f & 4) && ($f & 2) != ($mode & 2)) {
            throw new RuntimeException("Iterators' LIFO/FIFO modes for SplStack/SplQueue objects are frozen");
        }
        $this->flags = ($mode & 3) | ($f & 4);
        return $this->flags;
    }
    public function getIteratorMode(): int { return $this->flags | static::MEC_MODE; }
    public function rewind(): void {
        if (($this->flags | static::MEC_MODE) & 2) { end($this->dllist); } else { reset($this->dllist); }
    }
    public function valid(): bool { return key($this->dllist) !== null; }
    public function current(): mixed { return key($this->dllist) === null ? null : current($this->dllist); }
    public function key(): int { return (int) key($this->dllist); }
    public function next(): void { $this->step($this->flags | static::MEC_MODE); }
    public function prev(): void { $this->step(($this->flags | static::MEC_MODE) ^ 2); }
    private function step($flags) {
        if ($flags & 1) {
            if ($flags & 2) {
                array_pop($this->dllist);
                end($this->dllist);
            } else {
                array_shift($this->dllist);
                reset($this->dllist);
            }
        } elseif ($flags & 2) {
            prev($this->dllist);
        } else {
            next($this->dllist);
        }
    }
}
class SplQueue extends SplDoublyLinkedList {
    const MEC_MODE = 4;
    public function __construct() { $this->setIteratorMode(0); }
    public function enqueue($value): void { $this->push($value); }
    public function dequeue(): mixed { return $this->shift(); }
}
class SplStack extends SplDoublyLinkedList {
    const MEC_MODE = 6;
    public function __construct() { $this->setIteratorMode(2); }
}
// A set of objects, each with its data: storage is the list var_dump shows,
// [obj, inf] pairs in insertion order (spl_object_storage_debug_info), searched
// by identity.
class SplObjectStorage implements Countable, SeekableIterator, Traversable, Iterator, Serializable, ArrayAccess {
    private $storage = [];
    public function attach(object $object, mixed $info = null): void {
        $at = $this->place($object);
        if ($at === null) {
            $this->storage[] = ["obj" => $object, "inf" => $info];
        } else {
            $this->storage[$at]["inf"] = $info;
        }
    }
    public function detach(object $object): void {
        $at = $this->place($object);
        if ($at !== null) {
            array_splice($this->storage, $at, 1);
            reset($this->storage);
        }
    }
    public function contains(object $object): bool { return $this->place($object) !== null; }
    public function addAll(SplObjectStorage $storage): int {
        foreach ($storage->storage as $e) { $this->attach($e["obj"], $e["inf"]); }
        return count($this->storage);
    }
    public function removeAll(SplObjectStorage $storage): int {
        foreach ($storage->storage as $e) { $this->detach($e["obj"]); }
        return count($this->storage);
    }
    public function removeAllExcept(SplObjectStorage $storage): int {
        foreach ($this->storage as $e) {
            if (!$storage->contains($e["obj"])) { $this->detach($e["obj"]); }
        }
        return count($this->storage);
    }
    public function getInfo(): mixed {
        $k = key($this->storage);
        return $k === null ? null : $this->storage[$k]["inf"];
    }
    public function setInfo(mixed $info): void {
        $k = key($this->storage);
        if ($k !== null) { $this->storage[$k]["inf"] = $info; }
    }
    public function getHash(object $object): string { return spl_object_hash($object); }
    public function count(int $mode = COUNT_NORMAL): int { return count($this->storage); }
    public function offsetExists($object): bool { return $this->contains($object); }
    public function offsetGet($object): mixed {
        $at = $this->place($object);
        if ($at === null) { throw new UnexpectedValueException("Object not found"); }
        return $this->storage[$at]["inf"];
    }
    public function offsetSet($object, mixed $info = null): void { $this->attach($object, $info); }
    public function offsetUnset($object): void { $this->detach($object); }
    public function rewind(): void { reset($this->storage); }
    public function valid(): bool { return key($this->storage) !== null; }
    public function key(): int { return (int) key($this->storage); }
    public function current(): object {
        $k = key($this->storage);
        if ($k === null) { throw new RuntimeException("Called current() on invalid iterator"); }
        return $this->storage[$k]["obj"];
    }
    public function next(): void { next($this->storage); }
    public function seek(int $offset): void {
        if ($offset < 0 || $offset >= count($this->storage)) {
            throw new OutOfBoundsException("Seek position " . $offset . " is out of range");
        }
        reset($this->storage);
        for ($i = 0; $i < $offset; $i++) { next($this->storage); }
    }
    // The object's place in storage, or null.
    private function place($object) {
        foreach ($this->storage as $i => $e) {
            if ($e["obj"] === $object) { return $i; }
        }
        return null;
    }
}
