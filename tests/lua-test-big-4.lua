-- Lua subset self test (big 4): string + number processing. Exercises character-by-
-- character scanning with string.sub, string.len/upper/lower/rep, digit<->value lookup
-- tables, and math.floor: string reversal/palindromes, split/join, integer parsing and
-- formatting (parseInt / itoa / zero-padding), run-length encoding round trips, a ROT13
-- substitution cipher, and a full recursive-descent arithmetic expression evaluator with
-- +-*/ , parentheses and precedence. Counts failures and exits with that count (0 == all
-- pass). Run by both the tree-walking interpreter and the LLVM-IR compiler; both engines
-- must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- digit <-> value lookup tables (no string.byte in the subset, so map explicitly)
local digitVal = {["0"] = 0, ["1"] = 1, ["2"] = 2, ["3"] = 3, ["4"] = 4,
                  ["5"] = 5, ["6"] = 6, ["7"] = 7, ["8"] = 8, ["9"] = 9}
local digitChar = {[0] = "0", [1] = "1", [2] = "2", [3] = "3", [4] = "4",
                   [5] = "5", [6] = "6", [7] = "7", [8] = "8", [9] = "9"}

-- ----- string reversal and palindrome test -----
local function reverseStr(s)
    local r = ""
    for i = string.len(s), 1, -1 do
        r = r .. string.sub(s, i, i)
    end
    return r
end
local function isPalindrome(s)
    return s == reverseStr(s)
end
check("reverse hello", reverseStr("hello"), "olleh")
check("reverse empty", reverseStr(""), "")
check("reverse single", reverseStr("x"), "x")
check("palindrome racecar", isPalindrome("racecar"), true)
check("palindrome level", isPalindrome("level"), true)
check("palindrome hello", isPalindrome("hello"), false)

-- ----- count occurrences of a character -----
local function countChar(s, ch)
    local n = 0
    for i = 1, string.len(s) do
        if string.sub(s, i, i) == ch then
            n = n + 1
        end
    end
    return n
end
check("count l in hello", countChar("hello", "l"), 2)
check("count s in mississippi", countChar("mississippi", "s"), 4)
check("count z absent", countChar("hello", "z"), 0)

-- ----- split on a separator and join back -----
local function split(s, sep)
    local parts = {}
    local cur = ""
    for i = 1, string.len(s) do
        local ch = string.sub(s, i, i)
        if ch == sep then
            table.insert(parts, cur)
            cur = ""
        else
            cur = cur .. ch
        end
    end
    table.insert(parts, cur)
    return parts
end
local function join(parts, sep)
    local r = ""
    for i = 1, #parts do
        if i > 1 then
            r = r .. sep
        end
        r = r .. parts[i]
    end
    return r
end
local fields = split("alpha,beta,gamma,delta", ",")
check("split count", #fields, 4)
check("split first", fields[1], "alpha")
check("split last", fields[4], "delta")
check("split join roundtrip", join(fields, ","), "alpha,beta,gamma,delta")
check("join with dash", join(fields, "-"), "alpha-beta-gamma-delta")
-- word count via split on spaces
local words = split("the quick brown fox jumps", " ")
check("word count", #words, 5)

-- ----- integer parsing (with optional sign) and formatting -----
local function parseInt(s)
    local neg = false
    local start = 1
    if string.sub(s, 1, 1) == "-" then
        neg = true
        start = 2
    end
    local v = 0
    for i = start, string.len(s) do
        v = v * 10 + digitVal[string.sub(s, i, i)]
    end
    if neg then
        return -v
    end
    return v
end
check("parse 42", parseInt("42"), 42)
check("parse zero", parseInt("0"), 0)
check("parse leading zeros", parseInt("00700"), 700)
check("parse negative", parseInt("-13"), -13)
check("parse big", parseInt("123456"), 123456)

local function itoa(n)
    if n == 0 then
        return "0"
    end
    local neg = false
    if n < 0 then
        neg = true
        n = -n
    end
    local r = ""
    while n > 0 do
        local d = n % 10
        r = digitChar[d] .. r
        n = n // 10
    end
    if neg then
        return "-" .. r
    end
    return r
end
check("itoa 12345", itoa(12345), "12345")
check("itoa 0", itoa(0), "0")
check("itoa negative", itoa(-42), "-42")
check("itoa 1000", itoa(1000), "1000")
-- itoa agrees with the built-in number->string coercion for whole numbers
check("itoa matches concat", itoa(98765), 98765 .. "")
-- parse and format are inverse
check("parse . itoa", parseInt(itoa(54321)), 54321)
check("itoa . parse", itoa(parseInt("246")), "246")

-- zero-padded fixed-width formatting
local function padLeft(s, width, pad)
    while string.len(s) < width do
        s = pad .. s
    end
    return s
end
check("pad 42 to 5", padLeft(itoa(42), 5, "0"), "00042")
check("pad already wide", padLeft(itoa(123456), 4, "0"), "123456")
check("pad spaces", padLeft("hi", 5, " "), "   hi")

-- ----- run-length encoding round trip (runs kept under 10 so counts are one digit) -----
local function rleEncode(s)
    local r = ""
    local i = 1
    local n = string.len(s)
    while i <= n do
        local ch = string.sub(s, i, i)
        local count = 1
        while i + count <= n and string.sub(s, i + count, i + count) == ch do
            count = count + 1
        end
        r = r .. ch .. count
        i = i + count
    end
    return r
end
local function rleDecode(s)
    local r = ""
    local i = 1
    local n = string.len(s)
    while i <= n do
        local ch = string.sub(s, i, i)
        local cnt = digitVal[string.sub(s, i + 1, i + 1)]
        r = r .. string.rep(ch, cnt)
        i = i + 2
    end
    return r
end
check("rle encode", rleEncode("aaabbbbccd"), "a3b4c2d1")
check("rle single", rleEncode("abc"), "a1b1c1")
check("rle decode", rleDecode("a3b4c2d1"), "aaabbbbccd")
check("rle roundtrip", rleDecode(rleEncode("wwwwwxyyz")), "wwwwwxyyz")

-- ----- ROT13 substitution cipher built from two parallel alphabets -----
local plain = "abcdefghijklmnopqrstuvwxyz"
local cipher = "nopqrstuvwxyzabcdefghijklm"
local rotMap = {}
for i = 1, string.len(plain) do
    rotMap[string.sub(plain, i, i)] = string.sub(cipher, i, i)
end
local function rot13(s)
    local r = ""
    for i = 1, string.len(s) do
        local ch = string.sub(s, i, i)
        local m = rotMap[ch]
        if m ~= nil then
            r = r .. m
        else
            r = r .. ch
        end
    end
    return r
end
check("rot13 hello", rot13("hello"), "uryyb")
check("rot13 keeps punctuation", rot13("a-b"), "n-o")
check("rot13 involution", rot13(rot13("metacompiler")), "metacompiler")

-- ----- case conversion via the string library -----
check("upper", string.upper("Lua Rocks"), "LUA ROCKS")
check("lower", string.lower("Lua ROCKS"), "lua rocks")
check("rep banner", string.rep("=", 5), "=====")
check("rep zero", string.rep("ab", 0), "")
-- title-case the first letter of a word by hand
local function capitalize(w)
    if string.len(w) == 0 then
        return w
    end
    return string.upper(string.sub(w, 1, 1)) .. string.sub(w, 2)
end
check("capitalize lua", capitalize("lua"), "Lua")
check("capitalize single", capitalize("x"), "X")
check("capitalize empty", capitalize(""), "")

-- ----- recursive-descent arithmetic expression evaluator -----
-- grammar: expr := term { (+|-) term } ; term := factor { (*|/) factor } ;
--          factor := number | '(' expr ')' . Parser state is a table {src,pos,len}.
local Expr = {}
function Expr.peek(st)
    if st.pos > st.len then
        return ""
    end
    return string.sub(st.src, st.pos, st.pos)
end
function Expr.advance(st)
    st.pos = st.pos + 1
end
function Expr.skipSpaces(st)
    while Expr.peek(st) == " " do
        Expr.advance(st)
    end
end
function Expr.parseNumber(st)
    local v = 0
    while digitVal[Expr.peek(st)] ~= nil do
        v = v * 10 + digitVal[Expr.peek(st)]
        Expr.advance(st)
    end
    return v
end
function Expr.factor(st)
    Expr.skipSpaces(st)
    if Expr.peek(st) == "(" then
        Expr.advance(st)
        local v = Expr.expr(st)
        Expr.skipSpaces(st)
        Expr.advance(st) -- consume the ')'
        return v
    end
    return Expr.parseNumber(st)
end
function Expr.term(st)
    local v = Expr.factor(st)
    Expr.skipSpaces(st)
    local ch = Expr.peek(st)
    while ch == "*" or ch == "/" do
        Expr.advance(st)
        local rhs = Expr.factor(st)
        if ch == "*" then
            v = v * rhs
        else
            v = v / rhs
        end
        Expr.skipSpaces(st)
        ch = Expr.peek(st)
    end
    return v
end
function Expr.expr(st)
    local v = Expr.term(st)
    Expr.skipSpaces(st)
    local ch = Expr.peek(st)
    while ch == "+" or ch == "-" do
        Expr.advance(st)
        local rhs = Expr.term(st)
        if ch == "+" then
            v = v + rhs
        else
            v = v - rhs
        end
        Expr.skipSpaces(st)
        ch = Expr.peek(st)
    end
    return v
end
function Expr.eval(s)
    local st = {src = s, pos = 1, len = string.len(s)}
    return Expr.expr(st)
end

check("eval number", Expr.eval("42"), 42)
check("eval add", Expr.eval("1+2"), 3)
check("eval precedence", Expr.eval("1+2*3"), 7)
check("eval parens", Expr.eval("(1+2)*3"), 9)
check("eval left assoc sub", Expr.eval("10-2-3"), 5)
check("eval mixed", Expr.eval("2*3+4*5"), 26)
check("eval nested parens", Expr.eval("((1+1))*((2+2))"), 8)
check("eval exact division", Expr.eval("100/(2+3)"), 20)
check("eval chained division", Expr.eval("100/5/2"), 10)
check("eval spaces ignored", Expr.eval("  12 + 30  "), 42)
check("eval long", Expr.eval("7*6-5+3*2"), 43)
check("eval deep", Expr.eval("2*(3+(4*(5-2)))"), 30)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset big self test 4 passed")
end
exit(fails)
