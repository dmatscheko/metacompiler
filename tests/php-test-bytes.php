<?php
// echo, print, printf and var_dump write the BYTES a PHP string holds, valid
// UTF-8 or not. Nothing inside a program can observe its own output, so this
// file is checked from outside: tests/native-full.sh compares the native
// binary's stdout with tests/php-bytes.expected, and the matrix and --cross hold
// the two halves to the same bytes. There is no PHP toolchain here; the expected
// bytes are the ones the PHP manual specifies (chr($n) is the one byte $n, a
// string is written unchanged).
$cs = [200, 65, 255, 128, 195];
foreach ($cs as $c) { echo chr($c); }
echo "\n";
print chr(200) . "é" . chr(169) . "\n";
echo "é", chr(195) . chr(169), "\n";
printf("%s|%d\n", chr(250), 3);
var_dump(chr(200));
echo strlen(chr(200)), "\n";
echo "a" . chr(0xE2) . chr(0x82), "\n";
echo "\xC8\xFF", substr("é", 0, 1), "\n";
// 'echo a, b, c' is one ZEND_ECHO per operand: a is written before b is
// evaluated, so what b's evaluation writes lands between a and b.
function bytesSide($s) { echo "[", $s, "]"; return $s; }
echo "a", bytesSide("b"), "c\n";
