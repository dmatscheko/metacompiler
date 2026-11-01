<?php
// A program that names no function or class of lib/php-stdlib.php and declares
// no function, so the library's global constants are the only part of it a
// reference compiles: a bareword, or a string defined() and constant() are
// handed. Exits with the failure count.

$fails = 0;
$got = [COUNT_RECURSIVE, SORT_STRING, count([1, [2, 3]], COUNT_RECURSIVE), defined('JSON_PRETTY_PRINT'),
        constant('PATHINFO_FILENAME')];
if ($got !== [1, 2, 4, true, 8]) {
    echo "FAIL libconst\n";
    $fails = $fails + 1;
}
echo "libconst: " . $fails . " failures\n";
exit($fails);
