// Self-checking test for the JavaScript interpreter (js-interpreter.abnf) and the
// LLVM-IR compiler (js-to-llvm-ir.abnf). THEME: string + number processing and parsing.
//
// Centerpiece is a recursive-descent arithmetic evaluator (tokenizer + parser with
// operator precedence, parentheses and unary minus) that runs entirely on strings and
// characters. Around it: string reversal / palindromes, a character-frequency map read
// with for-in, run-length encode/decode, a Caesar cipher via charCodeAt /
// String.fromCharCode, Roman-numeral conversion, hand-written atoi / itoa cross-checked
// against parseInt, CSV summation, word counting, anagram detection and title-casing.
// Every transform is round-tripped or compared to an expected value. main() returns the
// number of failed checks; exit 0 means all passed. Only genuinely implemented
// constructs are used, so both grammars pass and the compiler IR is byte-identical.

var failures = 0;
function check(cond) { if (!cond) { failures = failures + 1; } }

// ===== character helpers =====
function isDigitChar(ch) { return ch >= "0" && ch <= "9"; }
function isSpaceChar(ch) { return ch === " "; }

// ===== a recursive-descent arithmetic expression evaluator =====
// Grammar (integer arithmetic, '/' is floor division):
//   expr   = term   (('+' | '-') term)*
//   term   = factor (('*' | '/') factor)*
//   factor = number | '(' expr ')' | '-' factor

function tokenize(s) {
    var toks = [];
    var i = 0;
    while (i < s.length) {
        var ch = s.charAt(i);
        if (isSpaceChar(ch)) { i++; continue; }
        if (isDigitChar(ch)) {
            var num = 0;
            while (i < s.length && isDigitChar(s.charAt(i))) {
                num = num * 10 + (s.charCodeAt(i) - 48);
                i++;
            }
            toks.push({ t: "num", v: num });
        } else {
            toks.push({ t: "op", v: ch });
            i++;
        }
    }
    return toks;
}

// The parser threads a mutable cursor {i} through the recursion (no 'this' needed).
function peekOp(toks, cur) {
    if (cur.i < toks.length && toks[cur.i].t === "op") { return toks[cur.i].v; }
    return "";
}
function parseExpr(toks, cur) {
    var value = parseTerm(toks, cur);
    var op = peekOp(toks, cur);
    while (op === "+" || op === "-") {
        cur.i++;
        var rhs = parseTerm(toks, cur);
        value = (op === "+") ? value + rhs : value - rhs;
        op = peekOp(toks, cur);
    }
    return value;
}
function parseTerm(toks, cur) {
    var value = parseFactor(toks, cur);
    var op = peekOp(toks, cur);
    while (op === "*" || op === "/") {
        cur.i++;
        var rhs = parseFactor(toks, cur);
        value = (op === "*") ? value * rhs : Math.floor(value / rhs);
        op = peekOp(toks, cur);
    }
    return value;
}
function parseFactor(toks, cur) {
    var tk = toks[cur.i];
    if (tk.t === "num") { cur.i++; return tk.v; }
    if (tk.v === "-") { cur.i++; return -parseFactor(toks, cur); }
    // must be '('
    cur.i++;                        // consume '('
    var value = parseExpr(toks, cur);
    cur.i++;                        // consume ')'
    return value;
}
function evalExpr(s) {
    var toks = tokenize(s);
    var cur = { i: 0 };
    return parseExpr(toks, cur);
}
function testEvaluator() {
    check(evalExpr("7") === 7);
    check(evalExpr("1+2") === 3);
    check(evalExpr("1+2*3") === 7);
    check(evalExpr("(1+2)*3") === 9);
    check(evalExpr("2*3+4*5") === 26);
    check(evalExpr("10-2-3") === 5);            // left associative
    check(evalExpr("100/5/2") === 10);
    check(evalExpr("20/6") === 3);              // floor division
    check(evalExpr("-(3+4)") === -7);
    check(evalExpr("2*(3+(4-1))") === 12);
    check(evalExpr("((42))") === 42);
    check(evalExpr("1+2+3+4+5+6+7+8+9+10") === 55);
    check(evalExpr("2*2*2*2*2") === 32);
    check(evalExpr(" 3  +  4 * 2 ") === 11);    // spaces ignored
    check(evalExpr("(2+3)*(4+5)") === 45);
    check(evalExpr("-5*-4") === 20);            // unary minus on both sides
}

// ===== string reversal + palindrome =====
function reverseStr(s) {
    var out = "";
    for (var i = s.length - 1; i >= 0; i--) { out = out + s.charAt(i); }
    return out;
}
function isPalindrome(s) { return s === reverseStr(s); }
function testReverse() {
    check(reverseStr("") === "");
    check(reverseStr("a") === "a");
    check(reverseStr("abc") === "cba");
    check(reverseStr("hello") === "olleh");
    check(isPalindrome("racecar") === true);
    check(isPalindrome("level") === true);
    check(isPalindrome("hello") === false);
    check(isPalindrome("") === true);
    // Reversing twice is the identity.
    var samples = ["metacompiler", "abcdefg", "12321"];
    for (var i = 0; i < samples.length; i++) {
        check(reverseStr(reverseStr(samples[i])) === samples[i]);
    }
}

// ===== character frequency, read back with for-in =====
function charFreq(s) {
    var f = {};
    for (var i = 0; i < s.length; i++) {
        var c = s.charAt(i);
        if (f[c] === undefined) { f[c] = 0; }
        f[c] = f[c] + 1;
    }
    return f;
}
function testCharFreq() {
    var f = charFreq("mississippi");
    check(f["m"] === 1);
    check(f["i"] === 4);
    check(f["s"] === 4);
    check(f["p"] === 2);
    // for-in visits every distinct character exactly once; counts sum to the length.
    var distinct = 0;
    var total = 0;
    var best = "";
    var bestN = 0;
    for (var c in f) {
        distinct++;
        total = total + f[c];
        if (f[c] > bestN) { bestN = f[c]; best = c; }
    }
    check(distinct === 4);            // m, i, s, p
    check(total === 11);
    check(bestN === 4);
    check(best === "i" || best === "s");   // tie between i and s
}

// ===== run-length encoding, round-tripped =====
function rleEncode(s) {
    if (s.length === 0) { return ""; }
    var out = "";
    var count = 1;
    for (var i = 1; i <= s.length; i++) {
        if (i < s.length && s.charAt(i) === s.charAt(i - 1)) {
            count++;
        } else {
            out = out + count + s.charAt(i - 1);
            count = 1;
        }
    }
    return out;
}
function rleDecode(s) {
    var out = "";
    var i = 0;
    while (i < s.length) {
        var num = 0;
        while (i < s.length && isDigitChar(s.charAt(i))) {
            num = num * 10 + (s.charCodeAt(i) - 48);
            i++;
        }
        var ch = s.charAt(i);
        i++;
        for (var j = 0; j < num; j++) { out = out + ch; }
    }
    return out;
}
function testRle() {
    check(rleEncode("aaabbbcccd") === "3a3b3c1d");
    check(rleEncode("") === "");
    check(rleEncode("x") === "1x");
    check(rleEncode("aaaaaaaaaaaa") === "12a");    // 12 a's (multi-digit count)
    var samples = ["aaabbbcccd", "abcdef", "wwwwww", "aabbaabb", "x"];
    for (var i = 0; i < samples.length; i++) {
        check(rleDecode(rleEncode(samples[i])) === samples[i]);
    }
}

// ===== Caesar cipher via char codes, round-tripped =====
function caesar(s, shift) {
    var k = ((shift % 26) + 26) % 26;
    var out = "";
    for (var i = 0; i < s.length; i++) {
        var code = s.charCodeAt(i);
        if (code >= 65 && code <= 90) {
            out = out + String.fromCharCode((code - 65 + k) % 26 + 65);
        } else if (code >= 97 && code <= 122) {
            out = out + String.fromCharCode((code - 97 + k) % 26 + 97);
        } else {
            out = out + s.charAt(i);
        }
    }
    return out;
}
function testCaesar() {
    check(caesar("abc", 3) === "def");
    check(caesar("xyz", 3) === "abc");            // wrap-around
    check(caesar("Hello, World!", 0) === "Hello, World!");
    check(caesar("ABC", 1) === "BCD");
    // Encoding then decoding with the inverse shift is the identity.
    var samples = ["Attack at Dawn!", "The Quick Brown Fox", "abcXYZ 123"];
    for (var i = 0; i < samples.length; i++) {
        for (var k = 1; k <= 25; k++) {
            check(caesar(caesar(samples[i], k), 26 - k) === samples[i]);
        }
    }
    // A shift of 13 applied twice (ROT13) is the identity for letters.
    check(caesar(caesar("HelloWorld", 13), 13) === "HelloWorld");
}

// ===== Roman numerals, round-tripped =====
function toRoman(n) {
    var vals = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1];
    var syms = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"];
    var out = "";
    for (var i = 0; i < vals.length; i++) {
        while (n >= vals[i]) { out = out + syms[i]; n = n - vals[i]; }
    }
    return out;
}
function fromRoman(s) {
    var value = { I: 1, V: 5, X: 10, L: 50, C: 100, D: 500, M: 1000 };
    var total = 0;
    for (var i = 0; i < s.length; i++) {
        var cur = value[s.charAt(i)];
        var nxt = (i + 1 < s.length) ? value[s.charAt(i + 1)] : 0;
        if (cur < nxt) { total = total - cur; } else { total = total + cur; }
    }
    return total;
}
function testRoman() {
    check(toRoman(1) === "I");
    check(toRoman(4) === "IV");
    check(toRoman(9) === "IX");
    check(toRoman(14) === "XIV");
    check(toRoman(40) === "XL");
    check(toRoman(90) === "XC");
    check(toRoman(1994) === "MCMXCIV");
    check(toRoman(2023) === "MMXXIII");
    check(fromRoman("IV") === 4);
    check(fromRoman("MCMXCIV") === 1994);
    // Round-trip every number in a range.
    for (var n = 1; n <= 100; n++) { check(fromRoman(toRoman(n)) === n); }
    var wide = [444, 999, 1000, 1666, 2024, 3888];
    for (var i = 0; i < wide.length; i++) { check(fromRoman(toRoman(wide[i])) === wide[i]); }
}

// ===== hand-written atoi / itoa, cross-checked against parseInt and "" + n =====
function myAtoi(s) {
    var i = 0;
    var sign = 1;
    if (s.charAt(0) === "-") { sign = -1; i = 1; }
    else if (s.charAt(0) === "+") { i = 1; }
    var n = 0;
    while (i < s.length && isDigitChar(s.charAt(i))) {
        n = n * 10 + (s.charCodeAt(i) - 48);
        i++;
    }
    return sign * n;
}
function myItoa(n) {
    if (n === 0) { return "0"; }
    var neg = n < 0;
    if (neg) { n = -n; }
    var out = "";
    while (n > 0) {
        out = String.fromCharCode(48 + (n % 10)) + out;
        n = Math.floor(n / 10);
    }
    return neg ? "-" + out : out;
}
function testAtoiItoa() {
    var nums = [0, 1, 9, 10, 42, 100, 999, 12345, 1000000];
    for (var i = 0; i < nums.length; i++) {
        check(myItoa(nums[i]) === "" + nums[i]);
        check(myAtoi(myItoa(nums[i])) === nums[i]);
        check(myAtoi("" + nums[i]) === parseInt("" + nums[i]));
    }
    check(myAtoi("-1234") === -1234);
    check(myItoa(-1234) === "-1234");
    check(myAtoi("+56") === 56);
    check(myAtoi("007") === 7);
    check(parseInt("-1234") === myAtoi("-1234"));
    // itoa/atoi are inverses over a signed range.
    var signed = [-9999, -100, -1, 0, 1, 100, 9999];
    for (var j = 0; j < signed.length; j++) {
        check(myAtoi(myItoa(signed[j])) === signed[j]);
    }
}

// ===== CSV summation, word counting, anagrams, title-casing =====
function csvSum(s) {
    var parts = s.split(",");
    var sum = 0;
    for (var i = 0; i < parts.length; i++) { sum = sum + myAtoi(parts[i]); }
    return sum;
}
function wordCount(s) {
    var words = s.split(" ");
    var n = 0;
    for (var i = 0; i < words.length; i++) { if (words[i].length > 0) { n++; } }
    return n;
}
function isAnagram(a, b) {
    if (a.length !== b.length) { return false; }
    var fa = charFreq(a);
    var fb = charFreq(b);
    for (var k in fa) { if (fb[k] !== fa[k]) { return false; } }
    for (var k2 in fb) { if (fa[k2] !== fb[k2]) { return false; } }
    return true;
}
function capitalize(w) {
    if (w.length === 0) { return w; }
    return w.charAt(0).toUpperCase() + w.slice(1).toLowerCase();
}
function titleCase(s) {
    var words = s.split(" ");
    var out = [];
    for (var i = 0; i < words.length; i++) { out.push(capitalize(words[i])); }
    return out.join(" ");
}
function testTextUtils() {
    check(csvSum("1,2,3,4,5") === 15);
    check(csvSum("10,-5,3") === 8);
    check(csvSum("100") === 100);
    check(csvSum("0,0,0") === 0);

    check(wordCount("the quick brown fox") === 4);
    check(wordCount("one") === 1);
    check(wordCount("") === 0);

    check(isAnagram("listen", "silent") === true);
    check(isAnagram("triangle", "integral") === true);
    check(isAnagram("hello", "world") === false);
    check(isAnagram("abc", "abcd") === false);
    check(isAnagram("aabbcc", "abcabc") === true);

    check(capitalize("hELLO") === "Hello");
    check(capitalize("x") === "X");
    check(titleCase("hello WORLD foo") === "Hello World Foo");
    check(titleCase("the metacompiler project") === "The Metacompiler Project");
}

function main() {
    testEvaluator();
    testReverse();
    testCharFreq();
    testRle();
    testCaesar();
    testRoman();
    testAtoiItoa();
    testTextUtils();
    return failures;
}
