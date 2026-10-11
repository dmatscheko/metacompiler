<?php
// php-stdlib.php -- PHP's array and string library, written in PHP.
//
// Both php grammars compile this file into a program that names one of its
// functions (phpLibRef), so the interpreter half, llvm.Run and the native binary
// all run this one text and agree by construction. It is written on the engine
// natives - count, strlen, array_push, array_values, array_key_exists,
// array_key_last, array_pop, substr, strpos, strrpos, ord, chr, strtolower,
// strtoupper (lib/php-strings.js, abnf/jsrtphpstr.go) and __mec_fmtfloat, the
// one intrinsic, for sprintf's float conversions. A native also gets a wrapper
// here of the same name, which is what a string callable or f(...) reaches: a
// direct call never does, since the call is lowered to the native first.
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
const SORT_FLAG_CASE = 8;
const ARRAY_FILTER_USE_BOTH = 1;
const ARRAY_FILTER_USE_KEY = 2;
const CASE_LOWER = 0;
const CASE_UPPER = 1;
const STR_PAD_LEFT = 0;
const STR_PAD_RIGHT = 1;
const STR_PAD_BOTH = 2;

class ArgumentCountError extends TypeError {}

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
function array_walk(&$array, $callback, ...$arg) {
    foreach ($array as $k => &$v) {
        if (count($arg) > 0) { $callback($v, $k, $arg[0]); } else { $callback($v, $k); }
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
// Byte order, answering -1 / 0 / 1 (PHP 8.2).
function strcmp($string1, $string2) {
    $a = (string)$string1;
    $b = (string)$string2;
    if ($a === $b) { return 0; }
    $na = strlen($a);
    $nb = strlen($b);
    $n = $na < $nb ? $na : $nb;
    for ($i = 0; $i < $n; $i++) {
        $x = ord(substr($a, $i, 1));
        $y = ord(substr($b, $i, 1));
        if ($x != $y) { return $x < $y ? -1 : 1; }
    }
    return $na <=> $nb;
}
function strcasecmp($string1, $string2) { return strcmp(strtolower($string1), strtolower($string2)); }
function strncmp($string1, $string2, $length) {
    if ($length < 0) { throw new ValueError("strncmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(substr($string1, 0, $length), substr($string2, 0, $length));
}
function strncasecmp($string1, $string2, $length) {
    if ($length < 0) { throw new ValueError("strncasecmp(): Argument #3 (\$length) must be greater than or equal to 0"); }
    return strcmp(strtolower(substr($string1, 0, $length)), strtolower(substr($string2, 0, $length)));
}

// ----- strings: case and whitespace ------------------------------------------
function ucfirst($string) { return strtoupper(substr($string, 0, 1)) . substr($string, 1); }
function lcfirst($string) { return strtolower(substr($string, 0, 1)) . substr($string, 1); }
function ucwords($string, $separators = " \t\r\n\f\v") {
    $out = "";
    $up = true;
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($string, $i, 1);
        $out .= $up ? strtoupper($c) : $c;
        $up = strpos($separators, $c) !== false;
    }
    return $out;
}
// The byte set a trim character list names: "a..z" is a range.
function __mec_charset($characters) {
    $set = [];
    $n = strlen($characters);
    for ($i = 0; $i < $n; $i++) {
        $c = ord(substr($characters, $i, 1));
        if ($i + 3 < $n && substr($characters, $i + 1, 2) == "..") {
            $e = ord(substr($characters, $i + 3, 1));
            for ($x = $c; $x <= $e; $x++) { $set[$x] = true; }
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
    if ($left) { while ($a < $b && isset($set[ord(substr($string, $a, 1))])) { $a++; } }
    if ($right) { while ($b > $a && isset($set[ord(substr($string, $b - 1, 1))])) { $b--; } }
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
    if (count($args) == 2) {
        $from = (string)$args[0];
        $to = (string)$args[1];
        $m = strlen($from) < strlen($to) ? strlen($from) : strlen($to);
        $map = [];
        for ($i = 0; $i < $m; $i++) { $map[ord(substr($from, $i, 1))] = substr($to, $i, 1); }
        $out = "";
        $n = strlen($string);
        for ($i = 0; $i < $n; $i++) {
            $c = substr($string, $i, 1);
            $out .= $map[ord($c)] ?? $c;
        }
        return $out;
    }
    $pairs = [];
    foreach ($args[0] as $k => $v) { if ((string)$k !== "") { $pairs[(string)$k] = (string)$v; } }
    $out = "";
    $n = strlen($string);
    $i = 0;
    while ($i < $n) {
        $best = null;
        foreach ($pairs as $k => $v) {
            $k = (string)$k;
            if (($best === null || strlen($k) > strlen($best)) && substr($string, $i, strlen($k)) === $k) { $best = $k; }
        }
        if ($best === null) { $out .= substr($string, $i, 1); $i++; }
        else { $out .= $pairs[$best]; $i += strlen($best); }
    }
    return $out;
}
function nl2br($string, $use_xhtml = true) {
    $br = $use_xhtml ? "<br />" : "<br>";
    $out = "";
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($string, $i, 1);
        if ($c == "\r" || $c == "\n") {
            $out .= $br . $c;
            $d = substr($string, $i + 1, 1);
            if (($d == "\r" || $d == "\n") && $d != $c) { $out .= $d; $i++; }
        } else {
            $out .= $c;
        }
    }
    return $out;
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
function stripslashes($string) {
    $out = "";
    $n = strlen($string);
    for ($i = 0; $i < $n; $i++) {
        $c = substr($string, $i, 1);
        if ($c == "\\" && $i + 1 < $n) {
            $i++;
            $d = substr($string, $i, 1);
            $out .= $d == "0" ? "\x00" : $d;
        } elseif ($c != "\\") {
            $out .= $c;
        }
    }
    return $out;
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
        $out = str_split($text);
        for ($cur = 0; $cur < $n; $cur++) {
            $c = $out[$cur];
            if ($c === $break) { $laststart = $lastspace = $cur + 1; }
            elseif ($c === " ") {
                if ($cur - $laststart >= $width) { $out[$cur] = $break; $laststart = $cur + 1; }
                $lastspace = $cur;
            } elseif ($cur - $laststart >= $width && $laststart != $lastspace) {
                $out[$lastspace] = $break;
                $laststart = $lastspace + 1;
            }
        }
        return implode("", $out);
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
        $c = $format[$i];
        if ($c != "%") { $out .= $c; $i++; continue; }
        $i++;
        if ($i < $n && $format[$i] == "%") { $out .= "%"; $i++; continue; }
        $argnum = -1;
        $j = $i;
        while ($j < $n && ord($format[$j]) >= 48 && ord($format[$j]) <= 57) { $j++; }
        if ($j > $i && $j < $n && $format[$j] == "\$") {
            $argnum = (int)substr($format, $i, $j - $i) - 1;
            if ($argnum < 0) { throw new ValueError("Argument number specifier must be greater than zero and less than 2147483647"); }
            $i = $j + 1;
        }
        $left = false;
        $plus = false;
        $pad = " ";
        while ($i < $n) {
            $f = $format[$i];
            if ($f == "-") { $left = true; }
            elseif ($f == "+") { $plus = true; }
            elseif ($f == "0" || $f == " ") { $pad = $f; }
            elseif ($f == "'") { $i++; $pad = $format[$i]; }
            else { break; }
            $i++;
        }
        $width = 0;
        while ($i < $n && ord($format[$i]) >= 48 && ord($format[$i]) <= 57) { $width = $width * 10 + (int)$format[$i]; $i++; }
        $prec = -1;
        if ($i < $n && $format[$i] == ".") {
            $i++;
            $prec = 0;
            while ($i < $n && ord($format[$i]) >= 48 && ord($format[$i]) <= 57) { $prec = $prec * 10 + (int)$format[$i]; $i++; }
        }
        if ($i >= $n) { throw new ValueError("Missing format specifier at end of string"); }
        $conv = $format[$i];
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
