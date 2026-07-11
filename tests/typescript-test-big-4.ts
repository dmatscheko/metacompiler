// Self-checking TypeScript test #big-4: STRING + NUMBER PROCESSING.
//
// Themes: character-level algorithms via charCodeAt / String.fromCharCode (a Caesar
// cipher), Roman-numeral encode/decode, base conversion (binary & hex) with round-trips,
// run-length encoding/decoding, palindrome detection with normalization, a hand-written
// integer parser, number formatting (zero-padding and thousands grouping), a polynomial
// string hash, word tokenizing, title-casing, and character-frequency counting via
// for-in. main() returns the failure count; 0 means every engine agrees.

let failures: number = 0;

function check(cond: boolean, _label: string): void {
    if (!cond) { failures = failures + 1; }
}

// ---- character helpers ----

function isUpper(code: number): boolean { return code >= 65 && code <= 90; }
function isLower(code: number): boolean { return code >= 97 && code <= 122; }
function isDigitCode(code: number): boolean { return code >= 48 && code <= 57; }
function isLetterCode(code: number): boolean { return isUpper(code) || isLower(code); }

function reverseString(s: string): string {
    let out: string = "";
    for (let i: number = s.length - 1; i >= 0; i--) {
        out = out + s.charAt(i);
    }
    return out;
}

function testReverse(): void {
    check(reverseString("") === "", "reverse-empty");
    check(reverseString("a") === "a", "reverse-single");
    check(reverseString("abc") === "cba", "reverse-abc");
    check(reverseString("Hello") === "olleH", "reverse-hello");
    check(reverseString(reverseString("roundtrip")) === "roundtrip", "reverse-twice");
}

// ---- Caesar cipher (letters shifted, everything else passed through) ----

function shiftChar(ch: string, shift: number): string {
    const code: number = ch.charCodeAt(0);
    if (isUpper(code)) {
        return String.fromCharCode(((code - 65 + shift) % 26) + 65);
    }
    if (isLower(code)) {
        return String.fromCharCode(((code - 97 + shift) % 26) + 97);
    }
    return ch;
}

function caesarEncode(text: string, shift: number): string {
    const normShift: number = ((shift % 26) + 26) % 26;
    let out: string = "";
    for (let i: number = 0; i < text.length; i++) {
        out = out + shiftChar(text.charAt(i), normShift);
    }
    return out;
}

function caesarDecode(text: string, shift: number): string {
    return caesarEncode(text, 26 - (((shift % 26) + 26) % 26));
}

function testCaesar(): void {
    check(caesarEncode("ABC", 3) === "DEF", "caesar-abc");
    check(caesarEncode("XYZ", 3) === "ABC", "caesar-wrap");
    check(caesarEncode("Hello, World!", 3) === "Khoor, Zruog!", "caesar-sentence");
    check(caesarEncode("abc", 0) === "abc", "caesar-zero-shift");
    // Round-trip over several shifts.
    const msg: string = "The Quick Brown Fox 123";
    for (let shift: number = 1; shift <= 25; shift++) {
        check(caesarDecode(caesarEncode(msg, shift), shift) === msg, "caesar-roundtrip-" + shift);
    }
    // ROT13 applied twice is the identity.
    check(caesarEncode(caesarEncode("Metacompiler", 13), 13) === "Metacompiler", "rot13-twice");
}

// ---- Roman numerals ----

const ROMAN_VALUES: number[] = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1];
const ROMAN_SYMBOLS: string[] = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"];

function toRoman(input: number): string {
    let num: number = input;
    let out: string = "";
    for (let i: number = 0; i < ROMAN_VALUES.length; i++) {
        while (num >= ROMAN_VALUES[i]) {
            out = out + ROMAN_SYMBOLS[i];
            num = num - ROMAN_VALUES[i];
        }
    }
    return out;
}

function romanCharValue(ch: string): number {
    if (ch === "M") { return 1000; }
    if (ch === "D") { return 500; }
    if (ch === "C") { return 100; }
    if (ch === "L") { return 50; }
    if (ch === "X") { return 10; }
    if (ch === "V") { return 5; }
    if (ch === "I") { return 1; }
    return 0;
}

function fromRoman(s: string): number {
    let total: number = 0;
    for (let i: number = 0; i < s.length; i++) {
        const cur: number = romanCharValue(s.charAt(i));
        let nextVal: number = 0;
        if (i + 1 < s.length) { nextVal = romanCharValue(s.charAt(i + 1)); }
        if (cur < nextVal) { total = total - cur; } else { total = total + cur; }
    }
    return total;
}

function testRoman(): void {
    check(toRoman(4) === "IV", "roman-4");
    check(toRoman(9) === "IX", "roman-9");
    check(toRoman(58) === "LVIII", "roman-58");
    check(toRoman(1994) === "MCMXCIV", "roman-1994");
    check(toRoman(2023) === "MMXXIII", "roman-2023");
    check(toRoman(3888) === "MMMDCCCLXXXVIII", "roman-3888");

    check(fromRoman("IV") === 4, "unroman-4");
    check(fromRoman("MCMXCIV") === 1994, "unroman-1994");
    check(fromRoman("MMMDCCCLXXXVIII") === 3888, "unroman-3888");

    // Round-trip over a range.
    for (let n: number = 1; n <= 100; n++) {
        check(fromRoman(toRoman(n)) === n, "roman-roundtrip-" + n);
    }
}

// ---- base conversion ----

const HEX_DIGITS: string = "0123456789abcdef";

function toBinary(input: number): string {
    if (input === 0) { return "0"; }
    let n: number = input;
    let out: string = "";
    while (n > 0) {
        out = ("" + (n % 2)) + out;
        n = Math.floor(n / 2);
    }
    return out;
}

function parseBinary(s: string): number {
    let value: number = 0;
    for (let i: number = 0; i < s.length; i++) {
        value = value * 2 + (s.charCodeAt(i) - 48);
    }
    return value;
}

function toHex(input: number): string {
    if (input === 0) { return "0"; }
    let n: number = input;
    let out: string = "";
    while (n > 0) {
        out = HEX_DIGITS.charAt(n % 16) + out;
        n = Math.floor(n / 16);
    }
    return out;
}

function testBaseConversion(): void {
    check(toBinary(0) === "0", "bin-0");
    check(toBinary(1) === "1", "bin-1");
    check(toBinary(5) === "101", "bin-5");
    check(toBinary(255) === "11111111", "bin-255");
    check(toBinary(1024) === "10000000000", "bin-1024");

    check(parseBinary("0") === 0, "parsebin-0");
    check(parseBinary("101") === 5, "parsebin-5");
    check(parseBinary("11111111") === 255, "parsebin-255");

    check(toHex(0) === "0", "hex-0");
    check(toHex(255) === "ff", "hex-255");
    check(toHex(4096) === "1000", "hex-4096");
    check(toHex(43981) === "abcd", "hex-abcd");

    // Round-trip decimal -> binary -> decimal.
    for (let n: number = 0; n <= 260; n++) {
        check(parseBinary(toBinary(n)) === n, "bin-roundtrip-" + n);
    }
    // 0x literal must match the hex string interpretation.
    check(0xABCD === 43981, "hex-literal");
}

// ---- run-length encoding ----

function rleEncode(s: string): string {
    if (s.length === 0) { return ""; }
    let out: string = "";
    let runChar: string = s.charAt(0);
    let runLen: number = 1;
    for (let i: number = 1; i < s.length; i++) {
        const ch: string = s.charAt(i);
        if (ch === runChar) {
            runLen = runLen + 1;
        } else {
            out = out + runChar + ("" + runLen);
            runChar = ch;
            runLen = 1;
        }
    }
    out = out + runChar + ("" + runLen);
    return out;
}

function rleDecode(s: string): string {
    let out: string = "";
    let i: number = 0;
    while (i < s.length) {
        const ch: string = s.charAt(i);
        i = i + 1;
        let count: number = 0;
        while (i < s.length && isDigitCode(s.charCodeAt(i))) {
            count = count * 10 + (s.charCodeAt(i) - 48);
            i = i + 1;
        }
        for (let k: number = 0; k < count; k++) { out = out + ch; }
    }
    return out;
}

function testRLE(): void {
    check(rleEncode("") === "", "rle-empty");
    check(rleEncode("a") === "a1", "rle-single");
    check(rleEncode("aaa") === "a3", "rle-run");
    check(rleEncode("aaabbc") === "a3b2c1", "rle-mixed");
    check(rleEncode("abcd") === "a1b1c1d1", "rle-distinct");
    check(rleEncode("aaaaaaaaaaaa") === "a12", "rle-multidigit");

    check(rleDecode("a3b2c1") === "aaabbc", "rle-decode");
    check(rleDecode("a12") === "aaaaaaaaaaaa", "rle-decode-multidigit");

    // Round-trip.
    const samples: string[] = ["", "x", "wwwwaaadexxxxxx", "mississippi", "aaabbbcccddd"];
    for (let i: number = 0; i < samples.length; i++) {
        check(rleDecode(rleEncode(samples[i])) === samples[i], "rle-roundtrip-" + i);
    }
}

// ---- palindrome with normalization ----

function normalizeLetters(s: string): string {
    let out: string = "";
    for (let i: number = 0; i < s.length; i++) {
        const code: number = s.charCodeAt(i);
        if (isLetterCode(code)) {
            out = out + s.charAt(i).toLowerCase();
        }
    }
    return out;
}

function isPalindrome(s: string): boolean {
    const norm: string = normalizeLetters(s);
    let lo: number = 0;
    let hi: number = norm.length - 1;
    while (lo < hi) {
        if (norm.charAt(lo) !== norm.charAt(hi)) { return false; }
        lo = lo + 1;
        hi = hi - 1;
    }
    return true;
}

function testPalindrome(): void {
    check(isPalindrome("racecar"), "pal-racecar");
    check(isPalindrome("RaceCar"), "pal-mixedcase");
    check(isPalindrome("A man, a plan, a canal: Panama"), "pal-panama");
    check(isPalindrome("Was it a car or a cat I saw?"), "pal-cat");
    check(isPalindrome(""), "pal-empty");
    check(isPalindrome("x"), "pal-single");
    check(!isPalindrome("hello"), "pal-hello");
    check(!isPalindrome("palindrome"), "pal-not");
}

// ---- manual integer parser (validated against the host parseInt) ----

function parseIntManual(s: string): number {
    let i: number = 0;
    let sign: number = 1;
    if (s.length > 0 && s.charAt(0) === "-") { sign = -1; i = 1; }
    let value: number = 0;
    while (i < s.length && isDigitCode(s.charCodeAt(i))) {
        value = value * 10 + (s.charCodeAt(i) - 48);
        i = i + 1;
    }
    return sign * value;
}

function testParseInt(): void {
    check(parseIntManual("0") === 0, "pi-0");
    check(parseIntManual("42") === 42, "pi-42");
    check(parseIntManual("100000") === 100000, "pi-100000");
    check(parseIntManual("-7") === -7, "pi-neg");
    check(parseIntManual("007") === 7, "pi-leading-zeros");
    // Agreement with the host parseInt over a range.
    for (let n: number = 0; n <= 50; n++) {
        const str: string = "" + n;
        check(parseIntManual(str) === parseInt(str, 10), "pi-agree-" + n);
    }
}

// ---- number formatting ----

function padLeft(s: string, width: number, fill: string): string {
    let out: string = s;
    while (out.length < width) { out = fill + out; }
    return out;
}

function groupThousands(input: number): string {
    const digits: string = "" + input;
    let out: string = "";
    let count: number = 0;
    for (let i: number = digits.length - 1; i >= 0; i--) {
        out = digits.charAt(i) + out;
        count = count + 1;
        if (count % 3 === 0 && i > 0) { out = "," + out; }
    }
    return out;
}

function testFormatting(): void {
    check(padLeft("42", 5, "0") === "00042", "pad-zeros");
    check(padLeft("7", 3, " ") === "  7", "pad-spaces");
    check(padLeft("already", 3, "0") === "already", "pad-noop");
    check(padLeft("", 4, "*") === "****", "pad-empty");

    check(groupThousands(0) === "0", "group-0");
    check(groupThousands(12) === "12", "group-12");
    check(groupThousands(999) === "999", "group-999");
    check(groupThousands(1000) === "1,000", "group-1000");
    check(groupThousands(1234567) === "1,234,567", "group-million");
    check(groupThousands(1000000) === "1,000,000", "group-1M");
}

// ---- polynomial string hash ----

function hashStr(s: string): number {
    let h: number = 5381;
    for (let i: number = 0; i < s.length; i++) {
        h = (h * 33 + s.charCodeAt(i)) % 1000000007;
    }
    return h;
}

function testHash(): void {
    check(hashStr("") === 5381, "hash-empty");
    check(hashStr("a") === 177670, "hash-a");
    check(hashStr("ab") === 5863208, "hash-ab");
    // Determinism.
    check(hashStr("metacompiler") === hashStr("metacompiler"), "hash-stable");
    // Distinctness for a few different strings.
    check(hashStr("abc") !== hashStr("acb"), "hash-order-sensitive");
    check(hashStr("hello") !== hashStr("world"), "hash-distinct");
}

// ---- word tokenizing, title-case, frequency ----

function splitWords(sentence: string): string[] {
    const raw: string[] = sentence.split(" ");
    const out: string[] = [];
    for (let i: number = 0; i < raw.length; i++) {
        if (raw[i].length > 0) { out.push(raw[i]); }
    }
    return out;
}

function titleCaseWord(w: string): string {
    if (w.length === 0) { return ""; }
    return w.charAt(0).toUpperCase() + w.slice(1).toLowerCase();
}

function titleCase(sentence: string): string {
    const words: string[] = splitWords(sentence);
    let out: string = "";
    for (let i: number = 0; i < words.length; i++) {
        if (i > 0) { out = out + " "; }
        out = out + titleCaseWord(words[i]);
    }
    return out;
}

function countVowels(s: string): number {
    let count: number = 0;
    for (let i: number = 0; i < s.length; i++) {
        const c: string = s.charAt(i).toLowerCase();
        if (c === "a" || c === "e" || c === "i" || c === "o" || c === "u") {
            count = count + 1;
        }
    }
    return count;
}

// Character frequency as an object, summed back via for-in.
function charFrequency(s: string): { [k: string]: number } {
    const freq: { [k: string]: number } = {};
    for (let i: number = 0; i < s.length; i++) {
        const ch: string = s.charAt(i);
        if (freq[ch] === undefined) { freq[ch] = 0; }
        freq[ch] = freq[ch] + 1;
    }
    return freq;
}

function testTextProcessing(): void {
    const words: string[] = splitWords("  the  quick brown   fox  ");
    check(words.length === 4, "split-count");
    check(words[0] === "the", "split-first");
    check(words[3] === "fox", "split-last");

    check(titleCaseWord("hELLO") === "Hello", "titlecase-word");
    check(titleCase("hello world foo") === "Hello World Foo", "titlecase-sentence");
    check(titleCase("the QUICK brown") === "The Quick Brown", "titlecase-mixed");

    check(countVowels("Hello World") === 3, "vowels-hello-world");
    check(countVowels("bcdfg") === 0, "vowels-none");
    check(countVowels("AEIOU") === 5, "vowels-allcaps");

    const freq: { [k: string]: number } = charFrequency("mississippi");
    check(freq["m"] === 1, "freq-m");
    check(freq["i"] === 4, "freq-i");
    check(freq["s"] === 4, "freq-s");
    check(freq["p"] === 2, "freq-p");
    // Total over all keys (for-in) equals the string length.
    let total: number = 0;
    for (const key in freq) { total = total + freq[key]; }
    check(total === 11, "freq-total");
    // Distinct-character count is the number of keys.
    let distinct: number = 0;
    for (const key in freq) { distinct = distinct + 1; }
    check(distinct === 4, "freq-distinct");
}

function main(): number {
    testReverse();
    testCaesar();
    testRoman();
    testBaseConversion();
    testRLE();
    testPalindrome();
    testParseInt();
    testFormatting();
    testHash();
    testTextProcessing();
    return failures;
}
