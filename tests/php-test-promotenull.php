<?php
/* A promoted constructor parameter's null default does not make its type
   nullable, so "public int $x = null" is a COMPILE-time error in PHP ("Cannot use
   null as default value for parameter $x of type int", zend_compile_params): the
   program must be refused before it runs, and "before" must never appear. The
   nullable spellings beside it are legal. This file is a SHOULD-FAIL ratchet for
   both halves. */

echo "before\n";
class PromoteNullOk {
    function __construct(public ?int $a = null, public mixed $b = null, public int|null $c = null) {}
}
class PromoteNull {
    function __construct(public int $x = null) {}
}
