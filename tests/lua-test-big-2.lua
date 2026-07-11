-- Lua subset self test (big 2): heavy recursion + control flow. Exercises deep recursion
-- (factorial, power, Fibonacci, Ackermann, Towers of Hanoi, digit sums, gcd), mutual
-- recursion, iterative counterparts, deep if/elseif chains, nested loops, while/break,
-- repeat/until, and two small finite-state machines (a divisible-by-three DFA over binary
-- strings and a balanced-parenthesis matcher). Counts failures and exits with that count
-- (0 == all pass). Run by both the tree-walking interpreter and the LLVM-IR compiler;
-- both engines must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- ----- factorial: recursive and iterative agree -----
local function factRec(n)
    if n <= 1 then
        return 1
    end
    return n * factRec(n - 1)
end
local function factIter(n)
    local acc = 1
    for i = 2, n do
        acc = acc * i
    end
    return acc
end
check("fact rec 5", factRec(5), 120)
check("fact iter 5", factIter(5), 120)
check("fact rec 10", factRec(10), 3628800)
check("fact agree", factRec(8), factIter(8))

-- ----- integer power by recursion; a loop-based pow2 to cross-check -----
local function power(base, exp)
    if exp == 0 then
        return 1
    end
    return base * power(base, exp - 1)
end
local function pow2(n)
    local r = 1
    for i = 1, n do
        r = r * 2
    end
    return r
end
check("pow 2^10", power(2, 10), 1024)
check("pow 3^4", power(3, 4), 81)
check("pow 5^0", power(5, 0), 1)
check("pow2 loop", pow2(10), 1024)
check("pow agree", power(2, 12), pow2(12))

-- ----- Fibonacci: naive recursion vs iterative -----
local function fibRec(n)
    if n < 2 then
        return n
    end
    return fibRec(n - 1) + fibRec(n - 2)
end
local function fibIter(n)
    local a = 0
    local b = 1
    for i = 1, n do
        local nxt = a + b
        a = b
        b = nxt
    end
    return a
end
check("fib rec 15", fibRec(15), 610)
check("fib iter 15", fibIter(15), 610)
check("fib agree 20", fibRec(20), fibIter(20))

-- ----- Ackermann: doubly recursive -----
local function ack(m, n)
    if m == 0 then
        return n + 1
    end
    if n == 0 then
        return ack(m - 1, 1)
    end
    return ack(m - 1, ack(m, n - 1))
end
check("ack 2,2", ack(2, 2), 7)
check("ack 2,3", ack(2, 3), 9)
check("ack 3,2", ack(3, 2), 29)
check("ack 3,3", ack(3, 3), 61)

-- ----- Towers of Hanoi move count, cross-checked with 2^n - 1 -----
local function hanoi(n)
    if n == 0 then
        return 0
    end
    return 2 * hanoi(n - 1) + 1
end
check("hanoi 3", hanoi(3), 7)
check("hanoi 10", hanoi(10), 1023)
check("hanoi formula", hanoi(10), pow2(10) - 1)

-- ----- mutual recursion: isEven / isOdd -----
local function isEven(n)
    if n == 0 then
        return true
    end
    return isOdd(n - 1)
end
function isOdd(n)
    if n == 0 then
        return false
    end
    return isEven(n - 1)
end
check("even 0", isEven(0), true)
check("even 10", isEven(10), true)
check("even 7", isEven(7), false)
check("odd 7", isOdd(7), true)
check("odd 8", isOdd(8), false)

-- ----- recursive digit sum and gcd -----
local function digitSum(n)
    if n < 10 then
        return n
    end
    return n % 10 + digitSum(n // 10)
end
check("digitsum 12345", digitSum(12345), 15)
check("digitsum 99", digitSum(99), 18)
check("digitsum 1000000", digitSum(1000000), 1)

local function gcdRec(a, b)
    if b == 0 then
        return a
    end
    return gcdRec(b, a % b)
end
check("gcd rec 1071 462", gcdRec(1071, 462), 21)
check("gcd rec 270 192", gcdRec(270, 192), 6)

-- ----- Collatz stopping time (while loop with branching) -----
local function collatz(n)
    local steps = 0
    while n ~= 1 do
        if n % 2 == 0 then
            n = n // 2
        else
            n = 3 * n + 1
        end
        steps = steps + 1
    end
    return steps
end
check("collatz 1", collatz(1), 0)
check("collatz 6", collatz(6), 8)
check("collatz 27", collatz(27), 111)

-- ----- deep if/elseif chains: letter grade and weekday name -----
local function grade(score)
    if score >= 90 then
        return "A"
    elseif score >= 80 then
        return "B"
    elseif score >= 70 then
        return "C"
    elseif score >= 60 then
        return "D"
    else
        return "F"
    end
end
check("grade 95", grade(95), "A")
check("grade 83", grade(83), "B")
check("grade 71", grade(71), "C")
check("grade 60", grade(60), "D")
check("grade 40", grade(40), "F")

local function weekday(n)
    if n == 1 then
        return "Mon"
    elseif n == 2 then
        return "Tue"
    elseif n == 3 then
        return "Wed"
    elseif n == 4 then
        return "Thu"
    elseif n == 5 then
        return "Fri"
    elseif n == 6 then
        return "Sat"
    elseif n == 7 then
        return "Sun"
    else
        return "?"
    end
end
check("weekday 1", weekday(1), "Mon")
check("weekday 5", weekday(5), "Fri")
check("weekday 7", weekday(7), "Sun")
check("weekday 9", weekday(9), "?")

-- ----- nested loops: count Pythagorean triples with sides up to 20 -----
local function countTriples(limit)
    local n = 0
    for a = 1, limit do
        for b = a, limit do
            for c = b, limit do
                if a * a + b * b == c * c then
                    n = n + 1
                end
            end
        end
    end
    return n
end
check("pythagorean up to 20", countTriples(20), 6)

-- ----- nested loops summing a multiplication table -----
local function tableSum(n)
    local s = 0
    for i = 1, n do
        for j = 1, n do
            s = s + i * j
        end
    end
    return s
end
check("mult table sum 5", tableSum(5), 225)
check("mult table sum 10", tableSum(10), 3025)

-- ----- while with break: first n whose square exceeds a bound -----
local function firstSquareOver(bound)
    local n = 0
    while true do
        n = n + 1
        if n * n > bound then
            break
        end
    end
    return n
end
check("first square over 100", firstSquareOver(100), 11)
check("first square over 50", firstSquareOver(50), 8)

-- ----- repeat/until accumulation -----
local function triangular(n)
    local sum = 0
    local i = 0
    repeat
        i = i + 1
        sum = sum + i
    until i >= n
    return sum
end
check("triangular 10", triangular(10), 55)
check("triangular 100", triangular(100), 5050)

-- ----- DFA: is a binary string's value divisible by 3? -----
-- states 0/1/2 track value mod 3; reading a bit does state = (state*2 + bit) % 3.
local function divBy3(bits)
    local state = 0
    for i = 1, string.len(bits) do
        local bit = 0
        if string.sub(bits, i, i) == "1" then
            bit = 1
        end
        state = (state * 2 + bit) % 3
    end
    return state == 0
end
check("divby3 0", divBy3("0"), true)
check("divby3 11 (3)", divBy3("11"), true)
check("divby3 110 (6)", divBy3("110"), true)
check("divby3 111111 (63)", divBy3("111111"), true)
check("divby3 1 (1)", divBy3("1"), false)
check("divby3 1010 (10)", divBy3("1010"), false)
check("divby3 100 (4)", divBy3("100"), false)

-- ----- stack-machine balanced-parenthesis matcher -----
local function balanced(s)
    local depth = 0
    for i = 1, string.len(s) do
        local ch = string.sub(s, i, i)
        if ch == "(" then
            depth = depth + 1
        elseif ch == ")" then
            depth = depth - 1
            if depth < 0 then
                return false
            end
        end
    end
    return depth == 0
end
check("balanced ()", balanced("()"), true)
check("balanced (())", balanced("(())"), true)
check("balanced (()(()))", balanced("(()(()))"), true)
check("balanced (()", balanced("(()"), false)
check("balanced )(", balanced(")("), false)
check("balanced empty", balanced(""), true)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset big self test 2 passed")
end
exit(fails)
