<?php
// php-stdlib.php -- PHP's array and string library, written in PHP.
//
// Both php grammars compile this file into a program that names one of its
// functions (phpLibRef), so the interpreter half, llvm.Run and the native binary
// all run this one text and agree by construction. It is written on the engine
// natives - count, strlen, array_push, array_values, array_key_exists,
// array_key_last, array_pop, substr, strpos, strrpos, strcmp, ord, chr,
// strtolower, strtoupper (lib/php-strings.js, abnf/jsrtphpstr.go) and two
// intrinsics: __mec_fmtfloat for sprintf's float conversions and __mec_math for
// the math library's double operations. A native also gets a wrapper
// here of the same name, which is what a string callable or f(...) reaches: a
// direct call never does, since the call is lowered to the native first.
//
// A string is walked by BYTE offsets, but rebuilt from substr() RUNS between the
// ASCII bytes a function acts on: a one-byte piece of a multi-byte character is
// a stand-in (php-strings.js), which compares right but must not be copied out.
//
// Parameter names are the manual's, so named arguments bind. Errors are PHP 8's
// own ValueError / TypeError / ArgumentCountError texts. Deliberately absent:
// the internal array pointer (current/next/prev/key; reset and end answer their
// values without one), anything random (shuffle, array_rand), and the functions
// that reach the caller's variable scope (compact, extract).

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
function usort(&$array, $callback) { return __mec_sort($array, 1, $callback, 0, 1, false); }
function asort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, 1, true); }
function arsort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 1, null, $flags, -1, true); }
function uasort(&$array, $callback) { return __mec_sort($array, 1, $callback, 0, 1, true); }
function natsort(&$array) { return __mec_sort($array, 1, null, SORT_NATURAL, 1, true); }
function natcasesort(&$array) { return __mec_sort($array, 1, null, SORT_NATURAL | SORT_FLAG_CASE, 1, true); }
function ksort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 0, null, $flags, 1, true); }
function krsort(&$array, $flags = SORT_REGULAR) { return __mec_sort($array, 0, null, $flags, -1, true); }
function uksort(&$array, $callback) { return __mec_sort($array, 0, $callback, 0, 1, true); }

// ----- keys, values, searching -----------------------------------------------
function array_keys($array, ...$filter) {
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
function reset(&$array) {
    foreach ($array as $v) { return $v; }
    return false;
}
function end(&$array) {
    $k = array_key_last($array);
    return $k === null ? false : $array[$k];
}

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
// The trailing callback(s) of the u* variants, split off the array list.
function __mec_takecb(&$arrays) {
    return array_pop($arrays);
}
function array_diff($array, ...$arrays) { return __mec_setop($array, $arrays, 0, false, null, null); }
function array_diff_key($array, ...$arrays) { return __mec_setop($array, $arrays, 1, false, null, null); }
function array_diff_assoc($array, ...$arrays) { return __mec_setop($array, $arrays, 2, false, null, null); }
function array_intersect($array, ...$arrays) { return __mec_setop($array, $arrays, 0, true, null, null); }
function array_intersect_key($array, ...$arrays) { return __mec_setop($array, $arrays, 1, true, null, null); }
function array_intersect_assoc($array, ...$arrays) { return __mec_setop($array, $arrays, 2, true, null, null); }
function array_udiff($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 0, false, $cb, null); }
function array_uintersect($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 0, true, $cb, null); }
function array_diff_ukey($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 1, false, null, $cb); }
function array_intersect_ukey($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 1, true, null, $cb); }
function array_udiff_assoc($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 2, false, $cb, null); }
function array_uintersect_assoc($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 2, true, $cb, null); }
function array_diff_uassoc($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 2, false, null, $cb); }
function array_intersect_uassoc($array, ...$rest) { $cb = __mec_takecb($rest); return __mec_setop($array, $rest, 2, true, null, $cb); }

// ----- callbacks -------------------------------------------------------------
function array_map($callback, $array, ...$arrays) {
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
    $acc = $initial;
    foreach ($array as $v) { $acc = $callback($acc, $v); }
    return $acc;
}
// The callback's by-reference parameter writes back only where a call of a
// callable VALUE passes a reference: the interpreter half does, the compiler
// half's dynamic call passes the value.
function array_walk(&$array, $callback, ...$arg) {
    foreach ($array as $k => &$v) {
        if (count($arg) > 0) { $callback($v, $k, $arg[0]); } else { $callback($v, $k); }
    }
    unset($v);
    return true;
}
function array_walk_recursive(&$array, $callback, ...$arg) {
    foreach ($array as $k => &$v) {
        if (is_array($v)) { array_walk_recursive($v, $callback, ...$arg); }
        elseif (count($arg) > 0) { $callback($v, $k, $arg[0]); }
        else { $callback($v, $k); }
    }
    unset($v);
    return true;
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
            throw new TypeError("max(): Argument #1 (\$value) must be of type array, " . get_debug_type($value) . " given");
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
            throw new TypeError("min(): Argument #1 (\$value) must be of type array, " . get_debug_type($value) . " given");
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
            throw new TypeError("implode(): Argument #2 (\$array) must be of type ?array, " . get_debug_type($array) . " given");
        }
        $array = $separator;
        $separator = "";
    } elseif (!is_array($array)) {
        throw new TypeError("implode(): Argument #2 (\$array) must be of type ?array, " . get_debug_type($array) . " given");
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
function strrchr($haystack, $needle) {
    $p = strrpos($haystack, substr((string)$needle, 0, 1));
    if ($p === false) { return false; }
    return substr($haystack, $p);
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
function strtr($string, ...$args) {
    $string = (string)$string;
    $n = strlen($string);
    $out = "";
    $at = 0;
    if (count($args) == 2) {
        $from = (string)$args[0];
        $to = (string)$args[1];
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
    foreach ($args[0] as $k => $v) { if ((string)$k !== "") { $pairs[(string)$k] = (string)$v; } }
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
// A word is a run of ASCII letters, ' and -; the string's first byte cannot open
// one with ' or -, nor its last byte end one with - (php_str_word_count).
function str_word_count($string, $format = 0) {
    $words = [];
    $p = 0;
    $e = strlen($string);
    $first = substr($string, 0, 1);
    if ($first == "'" || $first == "-") { $p = 1; }
    if ($e > 0 && substr($string, $e - 1, 1) == "-") { $e--; }
    while ($p < $e) {
        $s = $p;
        while ($p < $e) {
            $o = ord(substr($string, $p, 1));
            if (($o >= 65 && $o <= 90) || ($o >= 97 && $o <= 122) || $o == 39 || $o == 45) { $p++; } else { break; }
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
    throw new TypeError($fn . "(): Argument #" . $n . " (\$" . $name . ") must be of type " . $type . ", " . get_debug_type($v) . " given");
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

// ----- var_export and JSON ---------------------------------------------------
// A float at serialize_precision=-1, which var_export and json_encode share with
// var_dump: the shortest round-tripping digits laid out by zend_gcvt at 17
// digits, with exponent marker $e; INF and NAN by name.
function __mec_serfloat($num, $e) {
    if (is_nan($num)) { return "NAN"; }
    if (is_infinite($num)) { return $num < 0 ? "-INF" : "INF"; }
    if ($num == 0) { return fdiv(1, $num) < 0 ? "-0" : "0"; }
    $s = __mec_fmtfloat($num, "g", 17);
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
    $std = get_class($v) === "stdClass";
    $out .= $std ? "(object) array(\n" : "\\" . get_class($v) . "::__set_state(array(\n";
    foreach ((array)$v as $k => $e) {
        $out .= str_repeat(" ", $level + 2) . __mec_exportstr((string)$k) . " => " . __mec_export($e, $level + 2) . ",\n";
    }
    return $out . $close . ($std ? ")" : "))");
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
    if (in_array($v, $st->open, true)) { return __mec_jfail($st, JSON_ERROR_RECURSION, "null"); }
    $st->open[] = $v;
    if ($v instanceof JsonSerializable) {
        $r = $v->jsonSerialize();
        $out = $r === $v ? __mec_jlist((array)$v, false, $st, $level) : __mec_jenc($r, $st, $level);
    } else {
        $out = __mec_jlist((array)$v, false, $st, $level);
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
