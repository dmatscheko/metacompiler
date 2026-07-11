<?php
// PHP subset BIG test 4 -- String & number processing.
//
// Theme: text and numeric formatting without any built-in string helpers beyond
// strlen and character indexing -- string reversal, run-length encoding, base
// conversion, integer parsing, Roman numerals -- culminating in a recursive
// descent arithmetic evaluator built as a small parser object. The program
// counts failures in $fails and ends with exit($fails); the interpreter and the
// LLVM-IR compiler run the same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- character classification (no ord/chr, so use ordered comparison) -----
function isDigitChar($c) {
    return $c >= "0" && $c <= "9";
}

function isVowel($c) {
    return $c === "a" || $c === "e" || $c === "i" || $c === "o" || $c === "u";
}

// Map a digit character to its integer value. Only call on a verified digit.
function digitValue($c) {
    $map = ["0" => 0, "1" => 1, "2" => 2, "3" => 3, "4" => 4,
            "5" => 5, "6" => 6, "7" => 7, "8" => 8, "9" => 9];
    return $map[$c];
}

// ----- basic string routines -----
function strRev($s) {
    $out = "";
    $n = strlen($s);
    for ($i = $n - 1; $i >= 0; $i--) {
        $out .= $s[$i];
    }
    return $out;
}

function isPalindrome($s) {
    return $s === strRev($s);
}

function countVowels($s) {
    $count = 0;
    $n = strlen($s);
    for ($i = 0; $i < $n; $i++) {
        if (isVowel($s[$i])) {
            $count++;
        }
    }
    return $count;
}

function repeatStr($s, $times) {
    $out = "";
    for ($i = 0; $i < $times; $i++) {
        $out .= $s;
    }
    return $out;
}

function padLeft($s, $width, $ch) {
    $out = $s;
    while (strlen($out) < $width) {
        $out = $ch . $out;
    }
    return $out;
}

function stripSpaces($s) {
    $out = "";
    $n = strlen($s);
    for ($i = 0; $i < $n; $i++) {
        $c = $s[$i];
        if ($c !== " ") {
            $out .= $c;
        }
    }
    return $out;
}

// Run-length encode: "aaabbc" -> "a3b2c1".
function runLengthEncode($s) {
    $n = strlen($s);
    if ($n === 0) {
        return "";
    }
    $out = "";
    $prev = $s[0];
    $run = 1;
    for ($i = 1; $i < $n; $i++) {
        $c = $s[$i];
        if ($c === $prev) {
            $run++;
        } else {
            $out .= $prev . $run;
            $prev = $c;
            $run = 1;
        }
    }
    $out .= $prev . $run;
    return $out;
}

// Count how many words are separated by single or multiple spaces.
function wordCount($s) {
    $count = 0;
    $inWord = false;
    $n = strlen($s);
    for ($i = 0; $i < $n; $i++) {
        if ($s[$i] === " ") {
            $inWord = false;
        } else {
            if (!$inWord) {
                $count++;
                $inWord = true;
            }
        }
    }
    return $count;
}

check("reverse", strRev("hello"), "olleh");
check("reverse empty", strRev(""), "");
check("reverse one", strRev("x"), "x");
check("palindrome yes", isPalindrome("racecar"), true);
check("palindrome no", isPalindrome("hello"), false);
check("palindrome empty", isPalindrome(""), true);
check("palindrome even", isPalindrome("abba"), true);
check("vowels", countVowels("hello world"), 3);
check("vowels none", countVowels("rhythm"), 0);
check("vowels all", countVowels("aeiou"), 5);
check("repeat", repeatStr("ab", 3), "ababab");
check("repeat zero", repeatStr("z", 0), "");
check("pad", padLeft("7", 3, "0"), "007");
check("pad nochange", padLeft("1234", 3, "0"), "1234");
check("strip", stripSpaces("a b  c   d"), "abcd");
check("rle", runLengthEncode("aaabbc"), "a3b2c1");
check("rle single", runLengthEncode("x"), "x1");
check("rle empty", runLengthEncode(""), "");
check("rle long run", runLengthEncode("wwwwwwwwww"), "w10");
check("words", wordCount("the quick brown fox"), 4);
check("words spaces", wordCount("  spread   out  words "), 3);
check("words empty", wordCount(""), 0);

// ----- base conversion and integer parsing -----
function intToBase($n, $base) {
    $digits = ["0", "1", "2", "3", "4", "5", "6", "7",
               "8", "9", "a", "b", "c", "d", "e", "f"];
    if ($n === 0) {
        return "0";
    }
    $neg = false;
    if ($n < 0) {
        $neg = true;
        $n = -$n;
    }
    $out = "";
    while ($n > 0) {
        $d = $n % $base;
        $out = $digits[$d] . $out;
        $n = intdiv($n, $base);
    }
    if ($neg) {
        $out = "-" . $out;
    }
    return $out;
}

function toBinary($n) {
    return intToBase($n, 2);
}

function toHex($n) {
    return intToBase($n, 16);
}

// Parse a (possibly signed) decimal integer from a string.
function parseIntStr($s) {
    $len = strlen($s);
    $i = 0;
    $neg = false;
    if ($len > 0 && $s[0] === "-") {
        $neg = true;
        $i = 1;
    }
    $n = 0;
    while ($i < $len) {
        $n = $n * 10 + digitValue($s[$i]);
        $i++;
    }
    if ($neg) {
        $n = -$n;
    }
    return $n;
}

check("bin 0", toBinary(0), "0");
check("bin 10", toBinary(10), "1010");
check("bin 255", toBinary(255), "11111111");
check("bin neg", toBinary(-6), "-110");
check("hex 255", toHex(255), "ff");
check("hex 4096", toHex(4096), "1000");
check("hex 0", toHex(0), "0");
check("octal 64", intToBase(64, 8), "100");
check("base3 26", intToBase(26, 3), "222");
check("parse", parseIntStr("12345"), 12345);
check("parse neg", parseIntStr("-42"), -42);
check("parse leading zero", parseIntStr("007"), 7);

// Decimal round-trip: parse(format(n)) == n.
for ($n = -50; $n <= 50; $n += 7) {
    check("decimal roundtrip " . $n, parseIntStr(intToBase($n, 10)), $n);
}

// Binary round-trip via manual reconstruction.
function fromBinary($s) {
    $n = 0;
    $len = strlen($s);
    for ($i = 0; $i < $len; $i++) {
        $n = $n * 2 + digitValue($s[$i]);
    }
    return $n;
}
check("bin roundtrip 42", fromBinary(toBinary(42)), 42);
check("bin roundtrip 1000", fromBinary(toBinary(1000)), 1000);

// ----- Roman numerals -----
function toRoman($n) {
    $values = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1];
    $symbols = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"];
    $out = "";
    $k = count($values);
    for ($i = 0; $i < $k; $i++) {
        while ($n >= $values[$i]) {
            $out .= $symbols[$i];
            $n -= $values[$i];
        }
    }
    return $out;
}

check("roman 1", toRoman(1), "I");
check("roman 4", toRoman(4), "IV");
check("roman 9", toRoman(9), "IX");
check("roman 58", toRoman(58), "LVIII");
check("roman 2023", toRoman(2023), "MMXXIII");
check("roman 1994", toRoman(1994), "MCMXCIV");
check("roman 3888", toRoman(3888), "MMMDCCCLXXXVIII");

// ----- recursive descent arithmetic evaluator -----
// Grammar:
//   expr   = term   (('+' | '-') term)*
//   term   = factor (('*' | '/') factor)*
//   factor = number | '(' expr ')'
// '/' is integer division. The three methods recurse through $this, sharing the
// $pos cursor across the whole parse.
class Calculator {
    public $src;
    public $pos;
    public function __construct($src) {
        $this->src = $src;
        $this->pos = 0;
    }
    public function peek() {
        if ($this->pos < strlen($this->src)) {
            return $this->src[$this->pos];
        }
        return "";
    }
    public function advance() {
        $c = $this->peek();
        $this->pos++;
        return $c;
    }
    public function parseExpr() {
        $val = $this->parseTerm();
        while (true) {
            $c = $this->peek();
            if ($c === "+") {
                $this->advance();
                $val += $this->parseTerm();
            } elseif ($c === "-") {
                $this->advance();
                $val -= $this->parseTerm();
            } else {
                break;
            }
        }
        return $val;
    }
    public function parseTerm() {
        $val = $this->parseFactor();
        while (true) {
            $c = $this->peek();
            if ($c === "*") {
                $this->advance();
                $val *= $this->parseFactor();
            } elseif ($c === "/") {
                $this->advance();
                $val = intdiv($val, $this->parseFactor());
            } else {
                break;
            }
        }
        return $val;
    }
    public function parseFactor() {
        $c = $this->peek();
        if ($c === "(") {
            $this->advance();
            $val = $this->parseExpr();
            $this->advance();
            return $val;
        }
        return $this->parseNumber();
    }
    public function parseNumber() {
        $num = 0;
        while (isDigitChar($this->peek())) {
            $num = $num * 10 + digitValue($this->advance());
        }
        return $num;
    }
}

function calc($expr) {
    $parser = new Calculator(stripSpaces($expr));
    return $parser->parseExpr();
}

check("calc number", calc("42"), 42);
check("calc add", calc("1 + 2"), 3);
check("calc precedence", calc("1 + 2 * 3"), 7);
check("calc parens", calc("(1 + 2) * 3"), 9);
check("calc left assoc sub", calc("7 - 3 - 2"), 2);
check("calc left assoc div", calc("100 / 5 / 2"), 10);
check("calc nested", calc("2 * (3 + 4) * (5 - 1)"), 56);
check("calc deep parens", calc("((1 + 1))"), 2);
check("calc multi digit", calc("10 + 20 * 3 - 4"), 66);
check("calc int div", calc("20 / 3"), 6);
check("calc big chain", calc("1+2+3+4+5+6+7+8+9+10"), 55);
check("calc mixed", calc("100 - 2 * (3 + 4 * 5) + 6"), 60);
check("calc all ops", calc("(8 + 4) / 3 * 2 - 1"), 7);

// Cross-check the evaluator against direct arithmetic over a range of inputs.
for ($a = 1; $a <= 6; $a++) {
    for ($b = 1; $b <= 6; $b++) {
        $expr = "(" . $a . " + " . $b . ") * " . $a;
        check("calc gen " . $a . "_" . $b, calc($expr), ($a + $b) * $a);
    }
}

// ----- done -----
if ($fails === 0) {
    echo "php-test-big-4 (string & number processing) passed\n";
}
exit($fails);
