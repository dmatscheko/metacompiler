/* MetaJS big test 4 - String and number processing.
 *
 * Character-level string work (reverse, palindrome, vowel and char counts,
 * anagrams via a hand-written sort, manual upper-casing through char codes,
 * Caesar cipher and rot13, run-length encode/decode, word capitalisation,
 * replace-all), positional number systems (arbitrary-base to/from strings,
 * binary/hex), number theory (primality, prime factorisation, digit counts),
 * Roman numerals both directions, an edit-distance DP table, and a small
 * rolling string hash.
 *
 * Self checking: main() returns the number of failed checks, so the run exits 0
 * exactly when all pass, identically under both engines and both back ends. **/

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks = checks + 1;
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

var DIGITS = "0123456789abcdef";

function isDigit(c) { return c >= "0" && c <= "9"; }

// ----- basic string algorithms -----

function reverseString(s) {
    var out = "";
    for (var i = s.length - 1; i >= 0; i--) { out += s.charAt(i); }
    return out;
}

function isPalindrome(s) {
    var i = 0;
    var j = s.length - 1;
    while (i < j) {
        if (s.charAt(i) != s.charAt(j)) { return false; }
        i++;
        j--;
    }
    return true;
}

function isVowel(c) {
    return c == "a" || c == "e" || c == "i" || c == "o" || c == "u";
}

function countVowels(s) {
    var lower = s.toLowerCase();
    var n = 0;
    for (var i = 0; i < lower.length; i++) {
        if (isVowel(lower.charAt(i))) { n++; }
    }
    return n;
}

function countChar(s, c) {
    var n = 0;
    for (var i = 0; i < s.length; i++) {
        if (s.charAt(i) == c) { n++; }
    }
    return n;
}

// Upper-case A..Z by subtracting 32 from the code of a..z, via char codes.
function toUpperManual(s) {
    var out = "";
    for (var i = 0; i < s.length; i++) {
        var code = s.charCodeAt(i);
        if (code >= 97 && code <= 122) { out += String.fromCharCode(code - 32); }
        else { out += s.charAt(i); }
    }
    return out;
}

function replaceAll(s, from, to) {
    return s.split(from).join(to);
}

function wordCount(s) {
    var trimmed = s.trim();
    if (trimmed.length == 0) { return 0; }
    return trimmed.split(" ").length;
}

function capitalizeWords(s) {
    var words = s.split(" ");
    var out = [];
    for (var i = 0; i < words.length; i++) {
        var w = words[i];
        if (w.length > 0) { out.push(w.charAt(0).toUpperCase() + w.substring(1)); }
        else { out.push(w); }
    }
    return out.join(" ");
}

// ----- anagram check via a hand-written character sort -----

function sortString(s) {
    var arr = [];
    for (var i = 0; i < s.length; i++) { arr.push(s.charCodeAt(i)); }
    for (var i = 1; i < arr.length; i++) {
        var key = arr[i];
        var j = i - 1;
        while (j >= 0 && arr[j] > key) {
            arr[j + 1] = arr[j];
            j--;
        }
        arr[j + 1] = key;
    }
    var out = "";
    for (var i = 0; i < arr.length; i++) { out += String.fromCharCode(arr[i]); }
    return out;
}

function isAnagram(a, b) { return sortString(a) == sortString(b); }

// ----- Caesar cipher and rot13 -----

function caesarShiftChar(c, shift) {
    var code = c.charCodeAt(0);
    if (code >= 65 && code <= 90) { return String.fromCharCode((code - 65 + shift) % 26 + 65); }
    if (code >= 97 && code <= 122) { return String.fromCharCode((code - 97 + shift) % 26 + 97); }
    return c;
}

function caesar(s, shift) {
    var out = "";
    for (var i = 0; i < s.length; i++) { out += caesarShiftChar(s.charAt(i), shift); }
    return out;
}

function caesarDecode(s, shift) { return caesar(s, (26 - shift) % 26); }

function rot13(s) { return caesar(s, 13); }

// ----- run-length encoding -----

function rleEncode(s) {
    var out = "";
    var i = 0;
    while (i < s.length) {
        var c = s.charAt(i);
        var run = 1;
        while (i + run < s.length && s.charAt(i + run) == c) { run++; }
        out += c + run;
        i += run;
    }
    return out;
}

function rleDecode(s) {
    var out = "";
    var i = 0;
    while (i < s.length) {
        var c = s.charAt(i);
        i++;
        var num = 0;
        while (i < s.length && isDigit(s.charAt(i))) {
            num = num * 10 + (s.charCodeAt(i) - 48);
            i++;
        }
        for (var k = 0; k < num; k++) { out += c; }
    }
    return out;
}

// ----- positional number systems -----

function toBase(n, base) {
    if (n == 0) { return "0"; }
    var v = n;
    var out = "";
    while (v > 0) {
        out = DIGITS.charAt(v % base) + out;
        v = Math.floor(v / base);
    }
    return out;
}

function fromBase(s, base) {
    var n = 0;
    for (var i = 0; i < s.length; i++) {
        n = n * base + DIGITS.indexOf(s.charAt(i));
    }
    return n;
}

// ----- number theory -----

function isPrime(n) {
    if (n < 2) { return false; }
    if (n < 4) { return true; }
    if (n % 2 == 0) { return false; }
    var d = 3;
    while (d * d <= n) {
        if (n % d == 0) { return false; }
        d += 2;
    }
    return true;
}

function primeFactors(n) {
    var out = [];
    var v = n;
    var d = 2;
    while (d * d <= v) {
        while (v % d == 0) {
            out.push(d);
            v = v / d;
        }
        d++;
    }
    if (v > 1) { out.push(v); }
    return out;
}

function countDigits(n) {
    if (n == 0) { return 1; }
    var v = n;
    var c = 0;
    while (v > 0) {
        c++;
        v = Math.floor(v / 10);
    }
    return c;
}

// ----- Roman numerals -----

function intToRoman(n) {
    var values = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1];
    var symbols = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"];
    var out = "";
    var v = n;
    for (var i = 0; i < values.length; i++) {
        while (v >= values[i]) {
            out += symbols[i];
            v -= values[i];
        }
    }
    return out;
}

function romanToInt(s) {
    var vmap = {I: 1, V: 5, X: 10, L: 50, C: 100, D: 500, M: 1000};
    var total = 0;
    for (var i = 0; i < s.length; i++) {
        var cur = vmap[s.charAt(i)];
        var nxt = i + 1 < s.length ? vmap[s.charAt(i + 1)] : 0;
        if (cur < nxt) { total -= cur; }
        else { total += cur; }
    }
    return total;
}

// ----- Levenshtein edit distance (rolling DP rows) -----

function levenshtein(a, b) {
    var m = a.length;
    var n = b.length;
    var prev = [];
    for (var j = 0; j <= n; j++) { prev.push(j); }
    for (var i = 1; i <= m; i++) {
        var cur = [i];
        for (var j = 1; j <= n; j++) {
            var cost = a.charAt(i - 1) == b.charAt(j - 1) ? 0 : 1;
            var best = prev[j] + 1;          // deletion
            var ins = cur[j - 1] + 1;         // insertion
            var sub = prev[j - 1] + cost;     // substitution
            if (ins < best) { best = ins; }
            if (sub < best) { best = sub; }
            cur.push(best);
        }
        prev = cur;
    }
    return prev[n];
}

// ----- rolling string hash -----

function hashString(s) {
    var h = 0;
    for (var i = 0; i < s.length; i++) {
        h = (h * 31 + s.charCodeAt(i)) % 1000000007;
    }
    return h;
}

function main() {

    // ----- reversal and palindromes -----
    check("reverse", reverseString("metacompiler"), "relipmocatem");
    check("reverse empty", reverseString(""), "");
    check("palindrome yes", isPalindrome("racecar"), true);
    check("palindrome no", isPalindrome("hello"), false);
    check("palindrome even", isPalindrome("abba"), true);
    check("palindrome single", isPalindrome("z"), true);

    // ----- counting -----
    check("count vowels", countVowels("Hello World"), 3);
    check("count vowels caps", countVowels("AEIOU"), 5);
    check("count vowels none", countVowels("rhythm"), 0);
    check("count char l", countChar("parallel", "l"), 3);
    check("count char missing", countChar("abc", "z"), 0);

    // ----- case, replace, words -----
    check("upper manual", toUpperManual("Hello, World 42!"), "HELLO, WORLD 42!");
    check("upper matches builtin", toUpperManual("mixedCASE"), "mixedCASE".toUpperCase());
    check("replace all", replaceAll("a.b.c.d", ".", "-"), "a-b-c-d");
    check("replace all word", replaceAll("na na na", "na", "yo"), "yo yo yo");
    check("replace all none", replaceAll("abc", "x", "y"), "abc");
    check("word count", wordCount("the quick brown fox"), 4);
    check("word count one", wordCount("word"), 1);
    check("word count spaces", wordCount("   padded text   "), 2);
    check("word count empty", wordCount("   "), 0);
    check("capitalize", capitalizeWords("hello world from metajs"), "Hello World From Metajs");
    check("capitalize single", capitalizeWords("solo"), "Solo");

    // ----- anagrams -----
    check("anagram yes", isAnagram("listen", "silent"), true);
    check("anagram yes2", isAnagram("triangle", "integral"), true);
    check("anagram no", isAnagram("hello", "world"), false);
    check("anagram length differs", isAnagram("abc", "abcd"), false);
    check("sort string", sortString("dbca"), "abcd");

    // ----- Caesar / rot13 -----
    check("caesar encode", caesar("Hello, World!", 3), "Khoor, Zruog!");
    check("caesar roundtrip", caesarDecode(caesar("Attack at Dawn", 7), 7), "Attack at Dawn");
    check("caesar wrap", caesar("xyz", 3), "abc");
    check("caesar zero", caesar("same", 0), "same");
    check("rot13", rot13("abc"), "nop");
    check("rot13 involutive", rot13(rot13("The Quick Brown Fox")), "The Quick Brown Fox");

    // ----- run-length encoding -----
    check("rle encode", rleEncode("aaabbc"), "a3b2c1");
    check("rle encode single", rleEncode("abc"), "a1b1c1");
    check("rle roundtrip", rleDecode(rleEncode("aaaaabbbcccccccccc")), "aaaaabbbcccccccccc");
    check("rle decode", rleDecode("x5y2"), "xxxxxyy");
    check("rle multi digit", rleEncode("aaaaaaaaaabbb"), "a10b3");

    // ----- number bases -----
    check("to binary", toBase(10, 2), "1010");
    check("to binary 255", toBase(255, 2), "11111111");
    check("to hex", toBase(255, 16), "ff");
    check("to hex 4096", toBase(4096, 16), "1000");
    check("to base zero", toBase(0, 2), "0");
    check("to octal", toBase(64, 8), "100");
    check("from binary", fromBase("1010", 2), 10);
    check("from hex", fromBase("ff", 16), 255);
    check("base roundtrip", fromBase(toBase(123456, 7), 7), 123456);
    check("base roundtrip 13", fromBase(toBase(98765, 13), 13), 98765);

    // ----- number theory -----
    check("prime 2", isPrime(2), true);
    check("prime 97", isPrime(97), true);
    check("prime 1", isPrime(1), false);
    check("prime 100", isPrime(100), false);
    check("prime 561", isPrime(561), false);
    check("factors 360", primeFactors(360).join(","), "2,2,2,3,3,5");
    check("factors prime", primeFactors(97).join(","), "97");
    check("factors power", primeFactors(64).join(","), "2,2,2,2,2,2");
    check("factors 1", primeFactors(1).join(","), "");
    check("count digits", countDigits(12345), 5);
    check("count digits zero", countDigits(0), 1);
    check("count digits ten", countDigits(1000000), 7);

    // ----- Roman numerals -----
    check("roman 4", intToRoman(4), "IV");
    check("roman 9", intToRoman(9), "IX");
    check("roman 58", intToRoman(58), "LVIII");
    check("roman 1994", intToRoman(1994), "MCMXCIV");
    check("roman 2024", intToRoman(2024), "MMXXIV");
    check("roman parse 1994", romanToInt("MCMXCIV"), 1994);
    check("roman parse 58", romanToInt("LVIII"), 58);
    check("roman roundtrip", romanToInt(intToRoman(3888)), 3888);

    // ----- edit distance -----
    check("edit kitten sitting", levenshtein("kitten", "sitting"), 3);
    check("edit flaw lawn", levenshtein("flaw", "lawn"), 2);
    check("edit equal", levenshtein("same", "same"), 0);
    check("edit empty a", levenshtein("", "abc"), 3);
    check("edit empty b", levenshtein("abc", ""), 3);
    check("edit one sub", levenshtein("cat", "cut"), 1);

    // ----- rolling hash: deterministic, in range, distinguishing -----
    check("hash A", hashString("A"), 65);
    check("hash AB", hashString("AB"), 2081);
    check("hash empty", hashString(""), 0);
    check("hash deterministic", hashString("metacompiler") == hashString("metacompiler"), true);
    check("hash distinguishes", hashString("abc") != hashString("acb"), true);
    check("hash in range", hashString("a very long string to hash") < 1000000007, true);

    printf("%c%c %d checks\n", 79, 75, checks);
    if (failures == 0) { println("MetaJS big test 4 (string and number processing) passed"); }
    return failures;
}
