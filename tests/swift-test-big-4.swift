// Swift subset self test: string and number processing.
//
// Theme: text and numbers pushed through many small formatters and parsers built
// from the implemented subset - string reversal and palindrome tests by indexing,
// vowel and word counting by character comparison, integer parsing from a digit
// string, base conversion (binary/octal/hex) via a digit lookup, Roman numerals,
// prime factorization rendered as "2*2*3", zero padding, digit sums, and a
// FizzBuzz line builder. Top level code runs in source order, counts failed
// checks and ends with exit(failures), so the run exits 0 exactly when every
// check passes. The interpreter and the LLVM-IR compiler must agree on every
// result.
//
// Subset note: there is no String() constructor or .uppercased() in the shared
// subset, so numbers become text through interpolation "\(n)" and characters are
// compared against single-character string literals. Indexing s[i] yields a
// one-character string.

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkS(_ name: String, _ got: String, _ want: String) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkB(_ name: String, _ got: Bool, _ want: Bool) {
    check(name, got ? 1 : 0, want ? 1 : 0)
}

// ---- reverse and palindrome by indexing ----

func reverseStr(_ s: String) -> String {
    var out = ""
    let n = s.count
    var i = n - 1
    while i >= 0 {
        out = out + s[i]
        i -= 1
    }
    return out
}

func isPalindromeStr(_ s: String) -> Bool {
    let n = s.count
    var i = 0
    var j = n - 1
    while i < j {
        if s[i] != s[j] {
            return false
        }
        i += 1
        j -= 1
    }
    return true
}

// ---- character classification ----

func countVowels(_ s: String) -> Int {
    let vowels = ["a", "e", "i", "o", "u"]
    var count = 0
    let n = s.count
    var i = 0
    while i < n {
        let ch = s[i]
        var j = 0
        while j < 5 {
            if ch == vowels[j] {
                count += 1
            }
            j += 1
        }
        i += 1
    }
    return count
}

func countWords(_ s: String) -> Int {
    let n = s.count
    var count = 0
    var inWord = false
    var i = 0
    while i < n {
        let ch = s[i]
        if ch == " " {
            inWord = false
        } else {
            if !inWord {
                count += 1
            }
            inWord = true
        }
        i += 1
    }
    return count
}

// ---- parse an integer from a string of digits (with optional leading '-') ----

func digitValue(_ ch: String) -> Int {
    let digits = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]
    var d = 0
    while d < 10 {
        if ch == digits[d] {
            return d
        }
        d += 1
    }
    return -1
}

func parseInt(_ s: String) -> Int {
    let n = s.count
    if n == 0 {
        return 0
    }
    var i = 0
    var sign = 1
    if s[0] == "-" {
        sign = -1
        i = 1
    }
    var value = 0
    while i < n {
        let d = digitValue(s[i])
        value = value * 10 + d
        i += 1
    }
    return sign * value
}

// ---- base conversion using a digit-symbol lookup ----

func toBase(_ n: Int, _ base: Int) -> String {
    let syms = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
                "a", "b", "c", "d", "e", "f"]
    if n == 0 {
        return "0"
    }
    var x = n
    var out = ""
    while x > 0 {
        let r = x % base
        out = syms[r] + out
        x = x / base
    }
    return out
}

// ---- Roman numerals ----

func toRoman(_ n: Int) -> String {
    let values = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1]
    let symbols = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"]
    var x = n
    var out = ""
    var i = 0
    let cnt = values.count
    while i < cnt {
        while x >= values[i] {
            out = out + symbols[i]
            x = x - values[i]
        }
        i += 1
    }
    return out
}

// ---- prime factorization rendered as "2*2*3" ----

func factorize(_ n: Int) -> String {
    var x = n
    var out = ""
    var first = true
    var d = 2
    while d * d <= x {
        while x % d == 0 {
            if !first {
                out = out + "*"
            }
            out = out + "\(d)"
            first = false
            x = x / d
        }
        d += 1
    }
    if x > 1 {
        if !first {
            out = out + "*"
        }
        out = out + "\(x)"
    }
    return out
}

// ---- zero pad a number to a minimum width ----

func pad(_ n: Int, _ width: Int) -> String {
    var s = "\(n)"
    let len = s.count
    var need = width - len
    while need > 0 {
        s = "0" + s
        need -= 1
    }
    return s
}

// ---- digit sum of a numeric string ----

func digitSumStr(_ s: String) -> Int {
    var sum = 0
    let n = s.count
    var i = 0
    while i < n {
        let v = digitValue(s[i])
        if v >= 0 {
            sum += v
        }
        i += 1
    }
    return sum
}

// ---- FizzBuzz ----

func fizzbuzz(_ n: Int) -> String {
    if n % 15 == 0 {
        return "FizzBuzz"
    }
    if n % 3 == 0 {
        return "Fizz"
    }
    if n % 5 == 0 {
        return "Buzz"
    }
    return "\(n)"
}

func fizzbuzzLine(_ n: Int) -> String {
    var out = ""
    var i = 1
    while i <= n {
        if i > 1 {
            out = out + ","
        }
        out = out + fizzbuzz(i)
        i += 1
    }
    return out
}

func main() {
    // reversal and palindromes
    checkS("reverse hello", reverseStr("hello"), "olleh")
    checkS("reverse empty", reverseStr(""), "")
    checkS("reverse ab", reverseStr("ab"), "ba")
    checkB("pal racecar", isPalindromeStr("racecar"), true)
    checkB("pal abba", isPalindromeStr("abba"), true)
    checkB("pal hello", isPalindromeStr("hello"), false)
    checkB("pal single", isPalindromeStr("x"), true)
    checkB("pal empty", isPalindromeStr(""), true)
    // reverse-equals-self is another way to see a palindrome
    checkB("pal via reverse", reverseStr("level") == "level", true)

    // character classification
    check("vowels hello world", countVowels("hello world"), 3)
    check("vowels aeiou", countVowels("aeiou"), 5)
    check("vowels xyz", countVowels("xyz"), 0)
    check("words three", countWords("hello world foo"), 3)
    check("words padded", countWords("  a  b  "), 2)
    check("words empty", countWords(""), 0)
    check("words one", countWords("single"), 1)

    // integer parsing
    check("parse 0", parseInt("0"), 0)
    check("parse 12345", parseInt("12345"), 12345)
    check("parse neg", parseInt("-4096"), -4096)
    check("digitValue 7", digitValue("7"), 7)
    check("digitValue x", digitValue("x"), -1)
    // round trip: number -> interpolated text -> parsed back
    let probe = 987654
    check("roundtrip", parseInt("\(probe)"), 987654)

    // base conversion
    checkS("bin 13", toBase(13, 2), "1101")
    checkS("bin 255", toBase(255, 2), "11111111")
    checkS("bin 0", toBase(0, 2), "0")
    checkS("oct 8", toBase(8, 8), "10")
    checkS("hex 255", toBase(255, 16), "ff")
    checkS("hex 100", toBase(100, 16), "64")
    checkS("dec 12345", toBase(12345, 10), "12345")
    // parse the base-10 rendering back to the original
    check("base10 roundtrip", parseInt(toBase(54321, 10)), 54321)

    // Roman numerals
    checkS("roman 4", toRoman(4), "IV")
    checkS("roman 9", toRoman(9), "IX")
    checkS("roman 58", toRoman(58), "LVIII")
    checkS("roman 1994", toRoman(1994), "MCMXCIV")
    checkS("roman 2023", toRoman(2023), "MMXXIII")
    checkS("roman 40", toRoman(40), "XL")

    // prime factorization
    checkS("factor 2", factorize(2), "2")
    checkS("factor 12", factorize(12), "2*2*3")
    checkS("factor 97", factorize(97), "97")
    checkS("factor 360", factorize(360), "2*2*2*3*3*5")

    // zero padding
    checkS("pad 42/5", pad(42, 5), "00042")
    checkS("pad 7/1", pad(7, 1), "7")
    checkS("pad wide", pad(12345, 3), "12345")

    // digit sums
    check("digitsum 12345", digitSumStr("12345"), 15)
    check("digitsum 9999", digitSumStr("9999"), 36)

    // FizzBuzz
    checkS("fb 3", fizzbuzz(3), "Fizz")
    checkS("fb 5", fizzbuzz(5), "Buzz")
    checkS("fb 15", fizzbuzz(15), "FizzBuzz")
    checkS("fb 7", fizzbuzz(7), "7")
    checkS("fizzbuzz line",
           fizzbuzzLine(15),
           "1,2,Fizz,4,Buzz,Fizz,7,8,Fizz,Buzz,11,Fizz,13,14,FizzBuzz")

    // a small combined pipeline: for each number 1..12, factorize it, and count
    // how many are prime (their factorization is the number itself)
    var primeCount = 0
    var k = 2
    while k <= 12 {
        if factorize(k) == "\(k)" {
            primeCount += 1
        }
        k += 1
    }
    // primes in [2,12]: 2,3,5,7,11 -> 5
    check("prime count 2..12", primeCount, 5)

    if fails == 0 {
        print("Swift string and number self test passed")
    }
}

main()
exit(fails)
