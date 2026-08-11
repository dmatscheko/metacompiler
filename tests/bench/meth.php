<?php
// NOT part of the shared bench gate: tests/bench.sh globs tests/bench/mod.*, so
// this file is only measured deliberately. It exists because mod.php is a WHILE
// loop over two locals with no class in it at all - it emits js_phthis zero times
// - so the php row could not see the one question docs/todo.md 5.3 asks: what a
// scope probe costs when it leaves the emitter for layer 2.
//
// The recipe (the kotlin analogue is tests/bench/forcap.kt at a6df3a8):
//
//     rm -rf /tmp/a /tmp/b && mkdir /tmp/a /tmp/b
//     git archive <base> | tar x -C /tmp/a ; git archive <base> | tar x -C /tmp/b
//     ... copy the change into /tmp/b ...
//     cp tests/bench/meth.php /tmp/a/tests/bench/mod.php
//     cp tests/bench/meth.php /tmp/b/tests/bench/mod.php
//     (cd /tmp/a && go build -o mec .) ; (cd /tmp/b && go build -o mec .)
//     (cd /tmp/b && tests/bench.sh --draws 9 --paired /tmp/a php)
//
// Three $this reads and one method call per iteration is the maximum pressure a
// PHP program can put on phThis, so a cost that does not show here does not exist.
class Acc {
    public $s = 0;
    public $n = 0;
    function step($i) {
        $this->n = $this->n + 1;
        $this->s = $this->s + $i % 7;
        return $this->s;
    }
}
$a = new Acc();
$i = 0;
while ($i < 40000) {
    $a->step($i);
    $i = $i + 1;
}
echo $a->s, " ", $a->n, "\n";
