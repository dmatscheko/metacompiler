<?php
// See mod.kt. Same loop, so the languages are comparable to each other.
//
// php had NO bench row until docs/todo.md 7.7, so nothing in this project ever
// loaded languages/lib/php-rt.ll under a timer. php is one of the three
// languages that need the __raw truthiness rule (manual 7.1), so its condition
// path goes through layer 2 on every iteration of this loop.
$s = 0;
$i = 0;
while ($i < 40000) {
    $s = $s + $i % 7;
    $i = $i + 1;
}
echo $s, "\n";
