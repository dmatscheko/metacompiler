-- Lua subset self test (big 1): data structures + algorithms. Exercises numeric arrays
-- (1-based), the # length operator on tables, table.insert/remove, nested tables, numeric
-- and generic-for loops, while loops, recursion and multiple local declaration through a
-- battery of classic algorithms: four sorts, binary search, a stack and a queue, a singly
-- linked list, gcd/lcm, the sieve of Eratosthenes, a Fibonacci table, matrix
-- multiplication and a sorted merge. Counts failures and exits with that count (0 == all
-- pass). Run by both the tree-walking interpreter and the LLVM-IR compiler; both engines
-- must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- ----- array helpers -----
local function copyArray(a)
    local r = {}
    for i = 1, #a do
        r[i] = a[i]
    end
    return r
end

local function sumArray(a)
    local s = 0
    for i = 1, #a do
        s = s + a[i]
    end
    return s
end

local function maxArray(a)
    local m = a[1]
    for i = 2, #a do
        if a[i] > m then
            m = a[i]
        end
    end
    return m
end

local function minArray(a)
    local m = a[1]
    for i = 2, #a do
        if a[i] < m then
            m = a[i]
        end
    end
    return m
end

local function isSorted(a)
    for i = 2, #a do
        if a[i - 1] > a[i] then
            return false
        end
    end
    return true
end

local function arrayEqual(a, b)
    if #a ~= #b then
        return false
    end
    for i = 1, #a do
        if a[i] ~= b[i] then
            return false
        end
    end
    return true
end

local function swap(a, i, j)
    local t = a[i]
    a[i] = a[j]
    a[j] = t
end

-- ----- four sorting algorithms -----
local function insertionSort(a)
    for i = 2, #a do
        local key = a[i]
        local j = i - 1
        while j >= 1 and a[j] > key do
            a[j + 1] = a[j]
            j = j - 1
        end
        a[j + 1] = key
    end
end

local function bubbleSort(a)
    local n = #a
    for i = 1, n - 1 do
        for j = 1, n - i do
            if a[j] > a[j + 1] then
                swap(a, j, j + 1)
            end
        end
    end
end

local function selectionSort(a)
    local n = #a
    for i = 1, n - 1 do
        local m = i
        for j = i + 1, n do
            if a[j] < a[m] then
                m = j
            end
        end
        swap(a, i, m)
    end
end

local function quicksortRange(a, lo, hi)
    if lo < hi then
        local pivot = a[hi]
        local i = lo - 1
        for j = lo, hi - 1 do
            if a[j] <= pivot then
                i = i + 1
                swap(a, i, j)
            end
        end
        swap(a, i + 1, hi)
        local p = i + 1
        quicksortRange(a, lo, p - 1)
        quicksortRange(a, p + 1, hi)
    end
end

local function quicksort(a)
    quicksortRange(a, 1, #a)
end

local data = {5, 2, 8, 1, 9, 3, 7, 4, 6, 0}
local expected = {0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
check("data sum", sumArray(data), 45)
check("data max", maxArray(data), 9)
check("data min", minArray(data), 0)
check("data unsorted", isSorted(data), false)

local ins = copyArray(data)
insertionSort(ins)
check("insertion sorted", isSorted(ins), true)
check("insertion equal", arrayEqual(ins, expected), true)
check("insertion sum kept", sumArray(ins), 45)

local bub = copyArray(data)
bubbleSort(bub)
check("bubble sorted", isSorted(bub), true)
check("bubble equal", arrayEqual(bub, expected), true)

local sel = copyArray(data)
selectionSort(sel)
check("selection sorted", isSorted(sel), true)
check("selection equal", arrayEqual(sel, expected), true)

local qk = copyArray(data)
quicksort(qk)
check("quicksort sorted", isSorted(qk), true)
check("quicksort equal", arrayEqual(qk, expected), true)

-- ----- binary search over the sorted array -----
local function binarySearch(a, target)
    local lo = 1
    local hi = #a
    while lo <= hi do
        local mid = (lo + hi) // 2
        if a[mid] == target then
            return mid
        elseif a[mid] < target then
            lo = mid + 1
        else
            hi = mid - 1
        end
    end
    return -1
end

check("bsearch first", binarySearch(expected, 0), 1)
check("bsearch middle", binarySearch(expected, 5), 6)
check("bsearch last", binarySearch(expected, 9), 10)
check("bsearch absent", binarySearch(expected, 42), -1)

-- ----- stack (LIFO) via table.insert/remove -----
local function newStack()
    return {}
end
local function push(s, v)
    table.insert(s, v)
end
local function pop(s)
    return table.remove(s)
end

local st = newStack()
push(st, 1)
push(st, 2)
push(st, 3)
check("stack depth", #st, 3)
check("stack pop 3", pop(st), 3)
check("stack pop 2", pop(st), 2)
check("stack after pops", #st, 1)
check("stack pop 1", pop(st), 1)

-- ----- queue (FIFO) via a head/tail pair (table.remove only pops the tail) -----
local function newQueue()
    return {items = {}, head = 1, tail = 0}
end
local function enqueue(q, v)
    q.tail = q.tail + 1
    q.items[q.tail] = v
end
local function dequeue(q)
    if q.head > q.tail then
        return nil
    end
    local v = q.items[q.head]
    q.items[q.head] = nil
    q.head = q.head + 1
    return v
end
local function qsize(q)
    return q.tail - q.head + 1
end

local q = newQueue()
enqueue(q, "a")
enqueue(q, "b")
enqueue(q, "c")
check("queue size", qsize(q), 3)
check("queue first out", dequeue(q), "a")
check("queue second out", dequeue(q), "b")
enqueue(q, "d")
check("queue third out", dequeue(q), "c")
check("queue fourth out", dequeue(q), "d")
check("queue empty out", dequeue(q), nil)
check("queue size empty", qsize(q), 0)

-- ----- singly linked list built from {value, next} nodes -----
local function cons(v, nxt)
    return {value = v, next = nxt}
end
local function listFromArray(a)
    local head = nil
    for i = #a, 1, -1 do
        head = cons(a[i], head)
    end
    return head
end
local function listLength(node)
    local n = 0
    while node ~= nil do
        n = n + 1
        node = node.next
    end
    return n
end
local function listSum(node)
    local s = 0
    while node ~= nil do
        s = s + node.value
        node = node.next
    end
    return s
end
local function listReverse(node)
    local prev = nil
    while node ~= nil do
        local nxt = node.next
        node.next = prev
        prev = node
        node = nxt
    end
    return prev
end

local list = listFromArray({10, 20, 30, 40})
check("list length", listLength(list), 4)
check("list head", list.value, 10)
check("list sum", listSum(list), 100)
local rev = listReverse(list)
check("list reversed head", rev.value, 40)
check("list reversed length", listLength(rev), 4)
check("list reversed sum", listSum(rev), 100)

-- ----- number theory: gcd, lcm, sieve of Eratosthenes -----
local function gcd(a, b)
    while b ~= 0 do
        local t = b
        b = a % b
        a = t
    end
    return a
end
local function lcm(a, b)
    return a // gcd(a, b) * b
end

check("gcd 48 36", gcd(48, 36), 12)
check("gcd coprime", gcd(17, 5), 1)
check("gcd zero", gcd(9, 0), 9)
check("lcm 4 6", lcm(4, 6), 12)
check("lcm 21 6", lcm(21, 6), 42)

local function sieve(limit)
    local composite = {}
    local primes = {}
    for i = 2, limit do
        if not composite[i] then
            table.insert(primes, i)
            local j = i * i
            while j <= limit do
                composite[j] = true
                j = j + i
            end
        end
    end
    return primes
end

local primes = sieve(30)
check("primes count under 30", #primes, 10)
check("first prime", primes[1], 2)
check("fourth prime", primes[4], 7)
check("last prime under 30", primes[#primes], 29)
check("primes sum", sumArray(primes), 129)

-- ----- Fibonacci table filled iteratively -----
local function fibTable(n)
    local f = {1, 1}
    for i = 3, n do
        f[i] = f[i - 1] + f[i - 2]
    end
    return f
end

local fibs = fibTable(10)
check("fib length", #fibs, 10)
check("fib 7", fibs[7], 13)
check("fib 10", fibs[10], 55)

-- ----- matrix multiplication over 2D tables -----
local function matMul(A, B, n, m, p)
    local C = {}
    for i = 1, n do
        C[i] = {}
        for j = 1, p do
            local s = 0
            for k = 1, m do
                s = s + A[i][k] * B[k][j]
            end
            C[i][j] = s
        end
    end
    return C
end

local A = {{1, 2}, {3, 4}}
local B = {{5, 6}, {7, 8}}
local C = matMul(A, B, 2, 2, 2)
check("mat 1,1", C[1][1], 19)
check("mat 1,2", C[1][2], 22)
check("mat 2,1", C[2][1], 43)
check("mat 2,2", C[2][2], 50)

-- identity times a matrix returns the matrix
local I = {{1, 0}, {0, 1}}
local AI = matMul(A, I, 2, 2, 2)
check("identity 1,1", AI[1][1], 1)
check("identity 2,2", AI[2][2], 4)

-- ----- merge two sorted arrays into one sorted array -----
local function merge(a, b)
    local r = {}
    local i, j = 1, 1
    while i <= #a and j <= #b do
        if a[i] <= b[j] then
            table.insert(r, a[i])
            i = i + 1
        else
            table.insert(r, b[j])
            j = j + 1
        end
    end
    while i <= #a do
        table.insert(r, a[i])
        i = i + 1
    end
    while j <= #b do
        table.insert(r, b[j])
        j = j + 1
    end
    return r
end

local merged = merge({1, 3, 5, 7}, {2, 4, 6, 8})
check("merge length", #merged, 8)
check("merge sorted", isSorted(merged), true)
check("merge equal", arrayEqual(merged, {1, 2, 3, 4, 5, 6, 7, 8}), true)
check("merge sum", sumArray(merged), 36)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset big self test 1 passed")
end
exit(fails)
