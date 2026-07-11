//go:build ignore

// Go subset big test 2: RECURSION AND CONTROL FLOW.
//
// Fibonacci three ways (recursive, iterative, memoized with a map), factorial,
// Euclid's gcd, fast exponentiation and modular power, Ackermann, the Towers of
// Hanoi (recursive move generation replayed and validated on three stacks),
// Collatz, binomial coefficients and Pascal rows, Catalan numbers, mutual
// recursion, digit recursions, and a turnstile finite-state machine driven by a
// token stream. main() ends with os.Exit(fails).

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

// ----- Fibonacci -----

func fibRec(n int) int {
	if n < 2 {
		return n
	}
	return fibRec(n-1) + fibRec(n-2)
}

func fibIter(n int) int {
	a := 0
	b := 1
	for i := 0; i < n; i++ {
		a, b = b, a+b
	}
	return a
}

func fibMemo(n int, memo map[int]int) int {
	if n < 2 {
		return n
	}
	v, ok := memo[n]
	if ok {
		return v
	}
	r := fibMemo(n-1, memo) + fibMemo(n-2, memo)
	memo[n] = r
	return r
}

func tribonacci(n int) int {
	if n == 0 {
		return 0
	}
	if n == 1 || n == 2 {
		return 1
	}
	a := 0
	b := 1
	c := 1
	for i := 3; i <= n; i++ {
		a, b, c = b, c, a+b+c
	}
	return c
}

// ----- factorial and combinatorics -----

func factRec(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factRec(n-1)
}

func factIter(n int) int {
	r := 1
	for i := 2; i <= n; i++ {
		r *= i
	}
	return r
}

func binom(n int, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	return binom(n-1, k-1) + binom(n-1, k)
}

func pascalRow(n int) []int {
	row := []int{}
	for k := 0; k <= n; k++ {
		row = append(row, binom(n, k))
	}
	return row
}

func catalan(n int) int {
	// C(2n, n) / (n + 1)
	return binom(2*n, n) / (n + 1)
}

// ----- number theory recursions -----

func gcdRec(a int, b int) int {
	if b == 0 {
		return a
	}
	return gcdRec(b, a%b)
}

func gcdIter(a int, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func lcm(a int, b int) int {
	return a / gcdRec(a, b) * b
}

func powFast(base int, exp int) int {
	if exp == 0 {
		return 1
	}
	half := powFast(base, exp/2)
	if exp%2 == 0 {
		return half * half
	}
	return base * half * half
}

func powMod(base int, exp int, mod int) int {
	result := 1
	b := base % mod
	e := exp
	for e > 0 {
		if e%2 == 1 {
			result = result * b % mod
		}
		b = b * b % mod
		e = e / 2
	}
	return result
}

func ackermann(m int, n int) int {
	if m == 0 {
		return n + 1
	}
	if n == 0 {
		return ackermann(m-1, 1)
	}
	return ackermann(m-1, ackermann(m, n-1))
}

// ----- digit recursions -----

func sumDigits(n int) int {
	if n < 0 {
		return sumDigits(-n)
	}
	if n < 10 {
		return n
	}
	return n%10 + sumDigits(n/10)
}

func countDigits(n int) int {
	if n < 0 {
		return countDigits(-n)
	}
	if n < 10 {
		return 1
	}
	return 1 + countDigits(n/10)
}

func reverseNum(n int) int {
	r := 0
	for n > 0 {
		r = r*10 + n%10
		n = n / 10
	}
	return r
}

func digitalRoot(n int) int {
	for n >= 10 {
		n = sumDigits(n)
	}
	return n
}

func collatzSteps(n int) int {
	steps := 0
	for n != 1 {
		if n%2 == 0 {
			n = n / 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return steps
}

// ----- mutual recursion -----

func isEven(n int) bool {
	if n == 0 {
		return true
	}
	return isOdd(n - 1)
}

func isOdd(n int) bool {
	if n == 0 {
		return false
	}
	return isEven(n - 1)
}

// ----- Towers of Hanoi: generate the moves, then replay and validate them -----

// A move is encoded as from*3 + to (pegs numbered 0..2).
func hanoi(n int, from int, to int, via int, moves []int) []int {
	if n == 0 {
		return moves
	}
	moves = hanoi(n-1, from, via, to, moves)
	moves = append(moves, from*3+to)
	moves = hanoi(n-1, via, to, from, moves)
	return moves
}

type Peg struct {
	data []int
	sp   int
}

func (p *Peg) push(v int) {
	if p.sp < len(p.data) {
		p.data[p.sp] = v
	} else {
		p.data = append(p.data, v)
	}
	p.sp++
}

func (p *Peg) pop() int {
	p.sp--
	return p.data[p.sp]
}

func (p *Peg) top() int {
	if p.sp == 0 {
		return 1000000 // an "infinitely large" disk floor
	}
	return p.data[p.sp-1]
}

func (p *Peg) empty() bool {
	return p.sp == 0
}

// Replays the move list; returns the number of illegal moves (a bigger disk onto
// a smaller one, or a pop from an empty peg).
func replayHanoi(n int, moves []int) int {
	pegs := []Peg{Peg{[]int{}, 0}, Peg{[]int{}, 0}, Peg{[]int{}, 0}}
	// Stack disks n..1 on peg 0 (largest at the bottom).
	for d := n; d >= 1; d-- {
		pegs[0].push(d)
	}
	bad := 0
	for _, mv := range moves {
		from := mv / 3
		to := mv % 3
		if pegs[from].empty() {
			bad++
			continue
		}
		disk := pegs[from].pop()
		if disk > pegs[to].top() {
			bad++
		}
		pegs[to].push(disk)
	}
	return bad
}

// ----- turnstile finite-state machine -----

// states: 0 = locked, 1 = unlocked. Returns [finalState, opened, rejected].
func turnstile(inputs []string) []int {
	state := 0
	opened := 0
	rejected := 0
	for _, in := range inputs {
		switch state {
		case 0: // locked
			switch in {
			case "coin":
				state = 1
			case "push":
				rejected++
			}
		default: // unlocked
			switch in {
			case "push":
				state = 0
				opened++
			case "coin":
				// extra coin, stays unlocked
			}
		}
	}
	return []int{state, opened, rejected}
}

// ----- nested parentheses over a token stream -----

// Returns [maxDepth, balanced] where balanced is 1 when every ')' has a match.
func parenDepth(tokens []string) []int {
	depth := 0
	maxDepth := 0
	balanced := 1
	for _, t := range tokens {
		switch t {
		case "(":
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case ")":
			depth--
			if depth < 0 {
				balanced = 0
				depth = 0
			}
		}
	}
	if depth != 0 {
		balanced = 0
	}
	return []int{maxDepth, balanced}
}

func main() {
	// Fibonacci: the three definitions agree.
	memo := make(map[int]int)
	fibOk := true
	for n := 0; n <= 20; n++ {
		if fibIter(n) != fibMemo(n, memo) {
			fibOk = false
		}
	}
	checkBool("fib iter == memo", fibOk, true)
	check("fibRec 10", fibRec(10), 55)
	check("fibRec 20", fibRec(20), 6765)
	check("fibIter 25", fibIter(25), 75025)
	check("fibMemo 30", fibMemo(30, memo), 832040)
	check("tribonacci 10", tribonacci(10), 149)

	// factorial
	check("factRec 5", factRec(5), 120)
	check("factIter 10", factIter(10), 3628800)
	checkBool("fact agree", factRec(8) == factIter(8), true)

	// combinatorics
	check("binom 5 2", binom(5, 2), 10)
	check("binom 10 3", binom(10, 3), 120)
	check("binom edge", binom(6, 0)+binom(6, 6), 2)
	row := pascalRow(6)
	check("pascal len", len(row), 7)
	check("pascal middle", row[3], 20)
	rowSum := 0
	sym := true
	for i := range row {
		rowSum += row[i]
		if row[i] != row[len(row)-1-i] {
			sym = false
		}
	}
	check("pascal row sum", rowSum, 64) // 2^6
	checkBool("pascal symmetric", sym, true)
	// Catalan sequence 1,1,2,5,14,42,132
	cat := []int{}
	for n := 0; n <= 6; n++ {
		cat = append(cat, catalan(n))
	}
	check("catalan 4", cat[4], 14)
	check("catalan 6", cat[6], 132)

	// gcd / lcm / powers
	check("gcdRec", gcdRec(48, 36), 12)
	check("gcdIter", gcdIter(1071, 462), 21)
	checkBool("gcd agree", gcdRec(270, 192) == gcdIter(270, 192), true)
	check("lcm", lcm(4, 6), 12)
	check("lcm coprime", lcm(7, 5), 35)
	check("powFast 2^10", powFast(2, 10), 1024)
	check("powFast 3^7", powFast(3, 7), 2187)
	check("powFast n^0", powFast(9, 0), 1)
	check("powMod", powMod(3, 13, 7), 3)   // 1594323 mod 7 = 3
	check("powMod big", powMod(7, 128, 13), 3)

	// Ackermann (kept small)
	check("ack 0 0", ackermann(0, 0), 1)
	check("ack 2 2", ackermann(2, 2), 7)
	check("ack 3 3", ackermann(3, 3), 61)
	check("ack 3 4", ackermann(3, 4), 125)

	// digit recursions
	check("sumDigits", sumDigits(12345), 15)
	check("sumDigits neg", sumDigits(-9999), 36)
	check("countDigits", countDigits(1000000), 7)
	check("reverseNum", reverseNum(1234500), 54321)
	check("digitalRoot", digitalRoot(9875), 2) // 9875->29->11->2
	check("collatz 27", collatzSteps(27), 111)
	check("collatz 6", collatzSteps(6), 8)

	// mutual recursion
	evenOk := true
	for n := 0; n <= 15; n++ {
		want := (n%2 == 0)
		if isEven(n) != want || isOdd(n) == want {
			evenOk = false
		}
	}
	checkBool("mutual even/odd", evenOk, true)

	// Towers of Hanoi
	for n := 1; n <= 8; n++ {
		moves := hanoi(n, 0, 2, 1, []int{})
		check("hanoi move count", len(moves), powFast(2, n)-1)
		check("hanoi legal replay", replayHanoi(n, moves), 0)
	}

	// turnstile FSM
	res := turnstile([]string{"push", "coin", "push", "push", "coin", "coin", "push"})
	check("turnstile final locked", res[0], 0)
	check("turnstile opened", res[1], 2)
	check("turnstile rejected", res[2], 2)

	// nested parentheses
	pd := parenDepth([]string{"(", "(", "a", ")", "(", "b", ")", ")"})
	check("paren max depth", pd[0], 2)
	check("paren balanced", pd[1], 1)
	bad := parenDepth([]string{"(", ")", ")", "("})
	check("paren unbalanced", bad[1], 0)

	if fails == 0 {
		fmt.Println("Go big test 2 (recursion and control flow) passed")
	}
	os.Exit(fails)
}
