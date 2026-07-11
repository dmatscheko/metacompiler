//go:build ignore

// Go subset big test 4: STRING AND NUMBER PROCESSING.
//
// Integer <-> string conversion (itoa / atoi over ranged characters), string
// reversal and palindromes, case mapping and a Caesar cipher through alphabet
// tables, run-length encoding, word statistics, anagram detection via character
// count maps, and a block of number theory (primality, the Sieve of
// Eratosthenes, prime factorization, perfect numbers, base conversion, digit
// work). It finishes with two evaluators over string token slices: a postfix
// (RPN) calculator on an array stack, and a recursive-descent parser for infix
// arithmetic with precedence and parentheses. main() ends with os.Exit(fails).

package main

import (
	"fmt"
	"os"
)

var fails = 0

func check(name string, got int, want int) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func checkS(name string, got string, want string) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func checkBool(name string, got bool, want bool) {
	g := 0
	w := 0
	if got {
		g = 1
	}
	if want {
		w = 1
	}
	check(name, g, w)
}

// ----- integer <-> string -----

var digitStr = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
var digitVal = map[string]int{"0": 0, "1": 1, "2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	s := ""
	for n > 0 {
		s = digitStr[n%10] + s
		n = n / 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func atoi(s string) int {
	n := 0
	neg := false
	first := true
	for _, ch := range s {
		if first && ch == "-" {
			neg = true
			first = false
			continue
		}
		first = false
		n = n*10 + digitVal[ch]
	}
	if neg {
		return -n
	}
	return n
}

func toBase(n int, base int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = digitStr[n%base] + s
		n = n / base
	}
	return s
}

// ----- string manipulation -----

func toChars(s string) []string {
	out := []string{}
	for _, ch := range s {
		out = append(out, ch)
	}
	return out
}

func reverseStr(s string) string {
	out := ""
	for _, ch := range s {
		out = ch + out
	}
	return out
}

func isPalindrome(s string) bool {
	chars := toChars(s)
	i := 0
	j := len(chars) - 1
	for i < j {
		if chars[i] != chars[j] {
			return false
		}
		i++
		j--
	}
	return true
}

func buildIndex(alpha []string) map[string]int {
	idx := make(map[string]int)
	for i, ch := range alpha {
		idx[ch] = i
	}
	return idx
}

func caesar(s string, k int, alpha []string, idx map[string]int) string {
	out := ""
	for _, ch := range s {
		j, ok := idx[ch]
		if ok {
			out = out + alpha[(j+k)%26]
		} else {
			out = out + ch
		}
	}
	return out
}

func mapCase(s string, from []string, to []string, idx map[string]int) string {
	out := ""
	for _, ch := range s {
		j, ok := idx[ch]
		if ok {
			out = out + to[j]
		} else {
			out = out + ch
		}
	}
	return out
}

func runLengthEncode(s string) string {
	chars := toChars(s)
	n := len(chars)
	out := ""
	i := 0
	for i < n {
		j := i
		for j < n && chars[j] == chars[i] {
			j++
		}
		out = out + chars[i] + itoa(j-i)
		i = j
	}
	return out
}

// returns [wordCount, longestWordLen]
func wordStats(s string) []int {
	words := 0
	longest := 0
	cur := 0
	inWord := false
	for _, ch := range s {
		if ch == " " {
			if inWord {
				words++
				if cur > longest {
					longest = cur
				}
			}
			inWord = false
			cur = 0
		} else {
			inWord = true
			cur++
		}
	}
	if inWord {
		words++
		if cur > longest {
			longest = cur
		}
	}
	return []int{words, longest}
}

func charCount(s string) map[string]int {
	m := make(map[string]int)
	for _, ch := range s {
		if ch != " " {
			m[ch]++
		}
	}
	return m
}

func mapsEqual(a map[string]int, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok || bv != v {
			return false
		}
	}
	return true
}

func isAnagram(a string, b string) bool {
	return mapsEqual(charCount(a), charCount(b))
}

// ----- number theory -----

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func sievePrimes(limit int) []int {
	sieve := []bool{}
	for i := 0; i <= limit; i++ {
		sieve = append(sieve, true)
	}
	sieve[0] = false
	if limit >= 1 {
		sieve[1] = false
	}
	for p := 2; p*p <= limit; p++ {
		if sieve[p] {
			for k := p * p; k <= limit; k += p {
				sieve[k] = false
			}
		}
	}
	out := []int{}
	for i := 2; i <= limit; i++ {
		if sieve[i] {
			out = append(out, i)
		}
	}
	return out
}

func primeFactors(n int) []int {
	out := []int{}
	d := 2
	for d*d <= n {
		for n%d == 0 {
			out = append(out, d)
			n = n / d
		}
		d++
	}
	if n > 1 {
		out = append(out, n)
	}
	return out
}

func sumProperDivisors(n int) int {
	total := 0
	for d := 1; d < n; d++ {
		if n%d == 0 {
			total += d
		}
	}
	return total
}

func isPerfect(n int) bool {
	return n > 0 && sumProperDivisors(n) == n
}

func gcd(a int, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func lcm(a int, b int) int {
	return a / gcd(a, b) * b
}

func digitSum(n int) int {
	if n < 0 {
		n = -n
	}
	s := 0
	for n > 0 {
		s += n % 10
		n = n / 10
	}
	return s
}

func reverseNum(n int) int {
	r := 0
	for n > 0 {
		r = r*10 + n%10
		n = n / 10
	}
	return r
}

func isNumberPalindrome(n int) bool {
	return n >= 0 && reverseNum(n) == n
}

func sumSlice(s []int) int {
	t := 0
	for _, v := range s {
		t += v
	}
	return t
}

func productSlice(s []int) int {
	p := 1
	for _, v := range s {
		p *= v
	}
	return p
}

// ----- postfix (RPN) evaluator on an array stack -----

func isOperator(t string) bool {
	return t == "+" || t == "-" || t == "*" || t == "/"
}

func evalRPN(tokens []string) int {
	stack := []int{}
	top := 0
	for _, t := range tokens {
		if isOperator(t) {
			b := stack[top-1]
			a := stack[top-2]
			top -= 2
			r := 0
			switch t {
			case "+":
				r = a + b
			case "-":
				r = a - b
			case "*":
				r = a * b
			case "/":
				r = a / b
			}
			if top < len(stack) {
				stack[top] = r
			} else {
				stack = append(stack, r)
			}
			top++
		} else {
			v := atoi(t)
			if top < len(stack) {
				stack[top] = v
			} else {
				stack = append(stack, v)
			}
			top++
		}
	}
	return stack[top-1]
}

// ----- recursive-descent infix evaluator over a token slice -----
//
// expr   = term   { ("+" | "-") term }
// term   = factor { ("*" | "/") factor }
// factor = number | "(" expr ")"

type Parser struct {
	toks []string
	pos  int
}

func (p *Parser) peek() string {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return ""
}

func (p *Parser) advance() string {
	t := p.toks[p.pos]
	p.pos++
	return t
}

func (p *Parser) parseFactor() int {
	t := p.peek()
	if t == "(" {
		p.advance()
		v := p.parseExpr()
		p.advance() // consume ")"
		return v
	}
	p.advance()
	return atoi(t)
}

func (p *Parser) parseTerm() int {
	v := p.parseFactor()
	for p.peek() == "*" || p.peek() == "/" {
		op := p.advance()
		rhs := p.parseFactor()
		if op == "*" {
			v = v * rhs
		} else {
			v = v / rhs
		}
	}
	return v
}

func (p *Parser) parseExpr() int {
	v := p.parseTerm()
	for p.peek() == "+" || p.peek() == "-" {
		op := p.advance()
		rhs := p.parseTerm()
		if op == "+" {
			v = v + rhs
		} else {
			v = v - rhs
		}
	}
	return v
}

func evalInfix(tokens []string) int {
	p := Parser{tokens, 0}
	return p.parseExpr()
}

func main() {
	// itoa / atoi
	checkS("itoa 0", itoa(0), "0")
	checkS("itoa 90210", itoa(90210), "90210")
	checkS("itoa neg", itoa(-4096), "-4096")
	check("atoi 0", atoi("0"), 0)
	check("atoi 90210", atoi("90210"), 90210)
	check("atoi neg", atoi("-4096"), -4096)
	// round trip over a range
	rtOk := true
	for n := -50; n <= 50; n++ {
		if atoi(itoa(n)) != n {
			rtOk = false
		}
	}
	checkBool("itoa/atoi round trip", rtOk, true)

	// base conversion
	checkS("toBase bin 10", toBase(10, 2), "1010")
	checkS("toBase bin 255", toBase(255, 2), "11111111")
	checkS("toBase hex 255", toBase(255, 16), "ff")
	checkS("toBase hex 4096", toBase(4096, 16), "1000")
	checkS("toBase oct 64", toBase(64, 8), "100")

	// reversal / palindrome
	checkS("reverse", reverseStr("stressed"), "desserts")
	checkS("reverse empty", reverseStr(""), "")
	checkBool("palindrome yes", isPalindrome("racecar"), true)
	checkBool("palindrome no", isPalindrome("hello"), false)
	checkBool("palindrome single", isPalindrome("x"), true)

	// case mapping and caesar cipher
	lower := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	upper := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	lidx := buildIndex(lower)
	checkS("toUpper", mapCase("hello world", lower, upper, lidx), "HELLO WORLD")
	enc := caesar("abcxyz", 3, lower, lidx)
	checkS("caesar +3", enc, "defabc")
	dec := caesar(enc, 23, lower, lidx) // 23 = 26 - 3
	checkS("caesar round trip", dec, "abcxyz")
	checkS("caesar keeps spaces", caesar("a b", 1, lower, lidx), "b c")

	// run-length encoding
	checkS("rle", runLengthEncode("aaabbbbcc"), "a3b4c2")
	checkS("rle single", runLengthEncode("abc"), "a1b1c1")
	checkS("rle long run", runLengthEncode("wwwwwwwwww"), "w10")

	// word stats
	ws := wordStats("the quick brown fox jumps")
	check("word count", ws[0], 5)
	check("longest word", ws[1], 5)
	ws2 := wordStats("  padded   spaces here  ")
	check("word count padded", ws2[0], 3)

	// anagrams
	checkBool("anagram listen/silent", isAnagram("listen", "silent"), true)
	checkBool("anagram dormitory", isAnagram("dormitory", "dirty room"), true)
	checkBool("anagram no", isAnagram("hello", "world"), false)

	// primality and the sieve
	checkBool("isPrime 2", isPrime(2), true)
	checkBool("isPrime 97", isPrime(97), true)
	checkBool("isPrime 1", isPrime(1), false)
	checkBool("isPrime 91", isPrime(91), false) // 7 * 13
	primes := sievePrimes(30)
	check("sieve count", len(primes), 10)
	check("sieve first", primes[0], 2)
	check("sieve last", primes[9], 29)
	check("sieve sum", sumSlice(primes), 129)
	// sieve and trial division agree up to the limit
	agree := true
	for n := 0; n <= 30; n++ {
		inSieve := false
		for _, p := range primes {
			if p == n {
				inSieve = true
			}
		}
		if inSieve != isPrime(n) {
			agree = false
		}
	}
	checkBool("sieve matches isPrime", agree, true)

	// prime factorization
	pf := primeFactors(360)
	checkBool("factor 360", func360(pf), true)
	check("factor product", productSlice(pf), 360)
	check("factor 97 len", len(primeFactors(97)), 1)
	check("factor 1024 len", len(primeFactors(1024)), 10) // 2^10

	// perfect numbers
	checkBool("perfect 6", isPerfect(6), true)
	checkBool("perfect 28", isPerfect(28), true)
	checkBool("perfect 496", isPerfect(496), true)
	checkBool("perfect 12", isPerfect(12), false)

	// gcd / lcm / digits
	check("gcd", gcd(1071, 462), 21)
	check("lcm", lcm(21, 6), 42)
	check("digitSum", digitSum(99999), 45)
	check("reverseNum", reverseNum(13020), 2031)
	checkBool("num palindrome yes", isNumberPalindrome(12321), true)
	checkBool("num palindrome no", isNumberPalindrome(12345), false)

	// postfix (RPN) evaluation
	check("rpn simple", evalRPN([]string{"3", "4", "+"}), 7)
	check("rpn chain", evalRPN([]string{"5", "1", "2", "+", "4", "*", "+", "3", "-"}), 14)
	check("rpn mul add", evalRPN([]string{"2", "3", "4", "*", "+"}), 14)
	check("rpn div", evalRPN([]string{"100", "5", "/", "2", "/"}), 10)

	// infix evaluation with precedence and parentheses
	check("infix precedence", evalInfix([]string{"3", "+", "4", "*", "2"}), 11)
	check("infix parens", evalInfix([]string{"(", "1", "+", "2", ")", "*", "3"}), 9)
	check("infix left assoc", evalInfix([]string{"10", "-", "2", "-", "3"}), 5)
	check("infix nested", evalInfix([]string{"2", "*", "(", "3", "+", "4", ")", "-", "5"}), 9)
	check("infix deep", evalInfix([]string{"(", "(", "1", "+", "1", ")", "*", "(", "2", "+", "3", ")", ")"}), 10)
	check("infix multi digit", evalInfix([]string{"100", "/", "5", "/", "2"}), 10)

	if fails == 0 {
		fmt.Println("Go big test 4 (string and number processing) passed")
	}
	os.Exit(fails)
}

// verifies primeFactors(360) == [2,2,2,3,3,5]
func func360(pf []int) bool {
	want := []int{2, 2, 2, 3, 3, 5}
	if len(pf) != len(want) {
		return false
	}
	for i := range pf {
		if pf[i] != want[i] {
			return false
		}
	}
	return true
}
