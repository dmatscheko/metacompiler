<?php
// php-stdlib.php -- PHP's array and string library and SPL, written in PHP.
//
// Both php grammars compile this file into a program that names one of its
// functions or classes (phpLibRef), so the interpreter half, llvm.Run and the native binary
// all run this one text and agree by construction. It is written on the engine
// natives - count, strlen, array_push, array_values, array_key_exists,
// array_key_last, array_pop, substr, strpos, strrpos, strcmp, ord, chr,
// strtolower, strtoupper (lib/php-strings.js, abnf/jsrtphpstr.go) and nine
// intrinsics: __mec_fmtfloat for sprintf's float conversions ("r" is the
// shortest digits serialize_precision=-1 lays out), __mec_math for
// the math library's double operations, __mec_aptr for the internal array
// pointer, __mec_precision for the precision setting, __mec_oid for an
// object's handle, __mec_callable for the TypeError of an invalid callback
// argument, __mec_ifaces for the interfaces a class names and __mec_pregc /
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

class ArgumentCountError extends TypeError {}
class JsonException extends Exception {}
interface JsonSerializable { public function jsonSerialize(): mixed; }

// ----- the natives, as callable values ---------------------------------------
function count($value, $mode = 0) { return count($value); }
function sizeof($value, $mode = 0) { return count($value); }
function strlen($string) { return strlen($string); }
function array_push(&$array, ...$values) {
    foreach ($values as $v) { $array[] = $v; }
    return count($array);
}
function array_values($array) { return array_values($array); }
function abs($num) { return abs($num); }
function intdiv($num1, $num2) { return intdiv($num1, $num2); }
function array_key_exists($key, $array) { return array_key_exists($key, $array); }
function key_exists($key, $array) { return array_key_exists($key, $array); }
function array_key_last($array) { return array_key_last($array); }
function array_pop(&$array) { return array_pop($array); }
function substr($string, $offset, $length = null) { return substr($string, $offset, $length); }
function strpos($haystack, $needle, $offset = 0) { return strpos($haystack, $needle, $offset); }
function strrpos($haystack, $needle, $offset = 0) { return strrpos($haystack, $needle, $offset); }
function strcmp($string1, $string2) { return strcmp($string1, $string2); }
function ord($character) { return ord($character); }
function chr($codepoint) { return chr($codepoint); }
function strtolower($string) { return strtolower($string); }
function strtoupper($string) { return strtoupper($string); }
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
function explode($separator, $string, $limit = PHP_INT_MAX) {
    $separator = (string)$separator;
    $string = (string)$string;
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
function str_split($string, $length = 1) {
    if ($length < 1) { throw new ValueError("str_split(): Argument #2 (\$length) must be greater than 0"); }
    $out = [];
    $n = strlen($string);
    for ($i = 0; $i < $n; $i += $length) { $out[] = substr($string, $i, $length); }
    return $out;
}
function str_repeat($string, $times) {
    if ($times < 0) { throw new ValueError("str_repeat(): Argument #2 (\$times) must be greater than or equal to 0"); }
    $out = "";
    $piece = (string)$string;
    while ($times > 0) {
        if ($times % 2 == 1) { $out .= $piece; }
        $piece .= $piece;
        $times = intdiv($times, 2);
    }
    return $out;
}
function str_pad($string, $length, $pad_string = " ", $pad_type = STR_PAD_RIGHT) {
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
function chunk_split($string, $length = 76, $separator = "\r\n") {
    if ($length < 1) { throw new ValueError("chunk_split(): Argument #2 (\$length) must be greater than 0"); }
    $out = "";
    foreach (str_split($string, $length) as $p) { $out .= $p . $separator; }
    return $out;
}
function strrev($string) {
    $out = "";
    for ($i = strlen($string) - 1; $i >= 0; $i--) { $out .= substr($string, $i, 1); }
    return $out;
}
function bin2hex($string) {
    $hex = "0123456789abcdef";
    $out = "";
    $string = (string)$string;
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $b = ord(substr($string, $i, 1));
        $out .= $hex[$b >> 4] . $hex[$b & 15];
    }
    return $out;
}

// ----- strings: searching ----------------------------------------------------
function str_contains($haystack, $needle) { return strpos($haystack, $needle) !== false; }
function str_starts_with($haystack, $needle) { return substr($haystack, 0, strlen($needle)) === (string)$needle; }
function str_ends_with($haystack, $needle) {
    $n = strlen($needle);
    return $n == 0 || substr($haystack, -$n) === (string)$needle;
}
function stripos($haystack, $needle, $offset = 0) {
    try {
        return strpos(strtolower($haystack), strtolower($needle), $offset);
    } catch (ValueError $e) {
        throw new ValueError("stripos(): Argument #3 (\$offset) must be contained in argument #1 (\$haystack)");
    }
}
function strripos($haystack, $needle, $offset = 0) {
    try {
        return strrpos(strtolower($haystack), strtolower($needle), $offset);
    } catch (ValueError $e) {
        throw new ValueError("strripos(): Argument #3 (\$offset) must be contained in argument #1 (\$haystack)");
    }
}
function strstr($haystack, $needle, $before_needle = false) {
    $p = strpos($haystack, $needle);
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function strchr($haystack, $needle, $before_needle = false) { return strstr($haystack, $needle, $before_needle); }
function stristr($haystack, $needle, $before_needle = false) {
    $p = stripos($haystack, $needle);
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function strrchr($haystack, $needle, $before_needle = false) {
    $p = strrpos($haystack, substr((string)$needle, 0, 1));
    if ($p === false) { return false; }
    return $before_needle ? substr($haystack, 0, $p) : substr($haystack, $p);
}
function substr_count($haystack, $needle, $offset = 0, $length = null) {
    if ((string)$needle === "") { throw new ValueError("substr_count(): Argument #2 (\$needle) cannot be empty"); }
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
function strcasecmp($string1, $string2) { return strcmp(strtolower($string1), strtolower($string2)); }
function strncmp($string1, $string2, $length) {
    if ($length < 0) { throw new ValueError("strncmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(substr($string1, 0, $length), substr($string2, 0, $length));
}
function strncasecmp($string1, $string2, $length) {
    if ($length < 0) { throw new ValueError("strncasecmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(strtolower(substr($string1, 0, $length)), strtolower(substr($string2, 0, $length)));
}
// strnatcmp_ex: leading zeros skipped once, whitespace skipped, a run of digits
// compared by value (or left-aligned when either starts with 0), anything else
// byte by byte - folding ASCII case for strnatcasecmp.
function strnatcmp($string1, $string2) { return __mec_natcmp($string1, $string2, false); }
function strnatcasecmp($string1, $string2) { return __mec_natcmp($string1, $string2, true); }
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
function ucfirst($string) { return strtoupper(substr($string, 0, 1)) . substr($string, 1); }
function lcfirst($string) { return strtolower(substr($string, 0, 1)) . substr($string, 1); }
function ucwords($string, $separators = " \t\r\n\f\v") {
    $string = (string)$string;
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
function trim($string, $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, true, true); }
function ltrim($string, $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, true, false); }
function rtrim($string, $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, false, true); }
function chop($string, $characters = " \n\r\t\v\x00") { return __mec_trim($string, $characters, false, true); }

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
function nl2br($string, $use_xhtml = true) {
    $string = (string)$string;
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
function htmlspecialchars($string, $flags = 11, $encoding = null, $double_encode = true) {
    $map = ["&" => "&amp;", "\"" => "&quot;", "<" => "&lt;", ">" => "&gt;"];
    if (($flags & 3) == 3) { $map["'"] = "&#039;"; }
    if (($flags & 2) == 0) { unset($map["\""]); }
    return strtr($string, $map);
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
function str_word_count($string, $format = 0, $characters = null) {
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
function wordwrap($string, $width = 75, $break = "\n", $cut_long_words = false) {
    $text = (string)$string;
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
function round($num, $precision = 0, $mode = PHP_ROUND_HALF_UP) {
    $v = __mec_num("round", 1, "num", $num, "int|float");
    if ($mode < 1 || $mode > 4) {
        throw new ValueError("round(): Argument #3 (\$mode) must be a valid rounding mode (RoundingMode::*)");
    }
    $places = (int)$precision;
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
function number_format($num, $decimals = 0, $decimal_separator = ".", $thousands_separator = ",") {
    $f = (float)$num;
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
// up is " *RECURSION*" after its header.
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
        if ($isObj && is_string($k) && $k !== "" && $k[0] === "\0") {
            $p = strpos($k, "\0", 1);
            $cls = substr($k, 1, $p - 1);
            $k = substr($k, $p + 1) . ($cls === "*" ? ":protected" : ":" . $cls . ":private");
        }
        $out .= $pad . "    [" . $k . "] => " . __mec_printr($e, $indent + 8, $seen) . "\n";
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
