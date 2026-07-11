// Swift subset self test: data structures and algorithms.
//
// Theme: classic algorithms over arrays plus a few container types built from
// the implemented subset - gcd/lcm, the sieve of Eratosthenes, binary search,
// three in-place sorts (insertion, selection, bubble), a merge of two sorted
// runs, an array-backed Stack and a ring-buffer Queue class, and run length
// encoding. Top level code runs in source order, counts failed checks and ends
// with exit(failures), so the run exits 0 exactly when every check passes. The
// interpreter and the LLVM-IR compiler must agree on every result.
//
// Notes on the subset: arrays and class instances are reference types (no value
// copy on assignment); a `.count` immediately before a `{` would be swallowed as
// a trailing closure, so loop bounds are hoisted into locals or parenthesized.

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkB(_ name: String, _ got: Bool, _ want: Bool) {
    check(name, got ? 1 : 0, want ? 1 : 0)
}

// ---- number theory ----

func gcd(_ a: Int, _ b: Int) -> Int {
    var x = a
    var y = b
    while y != 0 {
        let t = x % y
        x = y
        y = t
    }
    return x < 0 ? -x : x
}
func lcm(_ a: Int, _ b: Int) -> Int {
    let g = gcd(a, b)
    if g == 0 { return 0 }
    return a / g * b
}

// ---- sieve of Eratosthenes: how many primes are <= n ----

func primesUpTo(_ n: Int) -> Int {
    if n < 2 { return 0 }
    var isPrime = []
    var i = 0
    let size = n + 1
    while i < size {
        isPrime.append(true)
        i += 1
    }
    isPrime[0] = false
    isPrime[1] = false
    var p = 2
    while p * p <= n {
        if isPrime[p] {
            var k = p * p
            while k <= n {
                isPrime[k] = false
                k += p
            }
        }
        p += 1
    }
    var count = 0
    var j = 2
    while j <= n {
        if isPrime[j] {
            count += 1
        }
        j += 1
    }
    return count
}

// ---- binary search over a sorted array, returns index or -1 ----

func binarySearch(_ a: [Int], _ target: Int) -> Int {
    var lo = 0
    let cnt = a.count
    var hi = cnt - 1
    while lo <= hi {
        let mid = lo + (hi - lo) / 2
        let v = a[mid]
        if v == target {
            return mid
        } else if v < target {
            lo = mid + 1
        } else {
            hi = mid - 1
        }
    }
    return -1
}

// ---- three in-place sorts (arrays are references, so these mutate in place) ----

func insertionSort(_ a: [Int]) {
    let n = a.count
    var i = 1
    while i < n {
        let key = a[i]
        var j = i - 1
        while j >= 0 && a[j] > key {
            a[j + 1] = a[j]
            j -= 1
        }
        a[j + 1] = key
        i += 1
    }
}

func selectionSort(_ a: [Int]) {
    let n = a.count
    var i = 0
    while i < n {
        var lo = i
        var j = i + 1
        while j < n {
            if a[j] < a[lo] {
                lo = j
            }
            j += 1
        }
        let t = a[i]
        a[i] = a[lo]
        a[lo] = t
        i += 1
    }
}

func bubbleSort(_ a: [Int]) {
    let n = a.count
    var pass = 0
    while pass < n {
        var j = 0
        let inner = n - 1 - pass
        while j < inner {
            if a[j] > a[j + 1] {
                let t = a[j]
                a[j] = a[j + 1]
                a[j + 1] = t
            }
            j += 1
        }
        pass += 1
    }
}

// is the array non-decreasing?
func isSorted(_ a: [Int]) -> Bool {
    let n = a.count
    var i = 1
    while i < n {
        if a[i - 1] > a[i] {
            return false
        }
        i += 1
    }
    return true
}

// merge two sorted arrays into a new sorted array
func mergeSorted(_ a: [Int], _ b: [Int]) -> [Int] {
    var out = []
    var i = 0
    var j = 0
    let na = a.count
    let nb = b.count
    while i < na && j < nb {
        if a[i] <= b[j] {
            out.append(a[i])
            i += 1
        } else {
            out.append(b[j])
            j += 1
        }
    }
    while i < na {
        out.append(a[i])
        i += 1
    }
    while j < nb {
        out.append(b[j])
        j += 1
    }
    return out
}

// ---- an array-backed stack with a logical top pointer ----

class IntStack {
    var data: [Int]
    var top: Int
    init() {
        self.data = []
        self.top = 0
    }
    func push(_ x: Int) {
        if (top < data.count) {
            data[top] = x
        } else {
            data.append(x)
        }
        top += 1
    }
    func pop() -> Int {
        top -= 1
        return data[top]
    }
    func peek() -> Int {
        return data[top - 1]
    }
    func size() -> Int {
        return top
    }
    func isEmpty() -> Bool {
        return top == 0
    }
}

// ---- a fixed-capacity ring-buffer queue ----

class RingQueue {
    var buf: [Int]
    var head: Int
    var count: Int
    var cap: Int
    init(capacity: Int) {
        self.buf = []
        var i = 0
        while i < capacity {
            self.buf.append(0)
            i += 1
        }
        self.head = 0
        self.count = 0
        self.cap = capacity
    }
    func enqueue(_ x: Int) -> Bool {
        if (count >= cap) {
            return false
        }
        let idx = (head + count) % cap
        buf[idx] = x
        count += 1
        return true
    }
    func dequeue() -> Int {
        let v = buf[head]
        head = (head + 1) % cap
        count -= 1
        return v
    }
    func size() -> Int {
        return count
    }
}

// ---- run length encoding of an int array into "value:run" pairs, summed ----

func rleChecksum(_ a: [Int]) -> Int {
    let n = a.count
    if n == 0 { return 0 }
    var acc = 0
    var i = 0
    while i < n {
        let v = a[i]
        var run = 1
        var j = i + 1
        while j < n && a[j] == v {
            run += 1
            j += 1
        }
        // fold each (value, run) pair into the accumulator
        acc = acc * 31 + v * 100 + run
        i = j
    }
    return acc
}

func main() {
    // gcd / lcm
    check("gcd 48 36", gcd(48, 36), 12)
    check("gcd 17 5", gcd(17, 5), 1)
    check("gcd 0 9", gcd(0, 9), 9)
    check("gcd neg", gcd(-24, 18), 6)
    check("lcm 4 6", lcm(4, 6), 12)
    check("lcm 21 6", lcm(21, 6), 42)

    // sieve
    check("primes<=10", primesUpTo(10), 4)
    check("primes<=30", primesUpTo(30), 10)
    check("primes<=100", primesUpTo(100), 25)
    check("primes<=1", primesUpTo(1), 0)

    // binary search on a fixed sorted array
    let sorted = [2, 3, 5, 7, 11, 13, 17, 19, 23]
    check("bs find 2", binarySearch(sorted, 2), 0)
    check("bs find 13", binarySearch(sorted, 13), 5)
    check("bs find 23", binarySearch(sorted, 23), 8)
    check("bs miss 4", binarySearch(sorted, 4), -1)
    check("bs miss 100", binarySearch(sorted, 100), -1)

    // insertion sort
    var a1 = [5, 2, 9, 1, 5, 6, 3]
    insertionSort(a1)
    checkB("insertion sorted", isSorted(a1), true)
    check("insertion a0", a1[0], 1)
    check("insertion last", a1[6], 9)

    // selection sort
    var a2 = [9, 8, 7, 6, 5, 4, 3, 2, 1, 0]
    selectionSort(a2)
    checkB("selection sorted", isSorted(a2), true)
    check("selection a0", a2[0], 0)
    check("selection a9", a2[9], 9)

    // bubble sort
    var a3 = [3, 1, 4, 1, 5, 9, 2, 6]
    bubbleSort(a3)
    checkB("bubble sorted", isSorted(a3), true)
    check("bubble a0", a3[0], 1)
    check("bubble a7", a3[7], 9)

    // stable-ish: sum survives sorting
    var s1 = 0
    for x in a3 {
        s1 += x
    }
    check("bubble sum preserved", s1, 31)

    // merge of two sorted runs
    let left = [1, 4, 6, 8, 10]
    let right = [2, 3, 5, 9]
    let merged = mergeSorted(left, right)
    check("merge count", merged.count, 9)
    checkB("merge sorted", isSorted(merged), true)
    check("merge first", merged[0], 1)
    check("merge last", merged[8], 10)

    // stack
    let st = IntStack()
    checkB("stack empty", st.isEmpty(), true)
    st.push(10)
    st.push(20)
    st.push(30)
    check("stack size", st.size(), 3)
    check("stack peek", st.peek(), 30)
    check("stack pop 30", st.pop(), 30)
    check("stack pop 20", st.pop(), 20)
    st.push(40)
    check("stack pop 40", st.pop(), 40)
    check("stack pop 10", st.pop(), 10)
    checkB("stack empty again", st.isEmpty(), true)

    // reverse [1..5] through the stack
    let rq = IntStack()
    for x in 1...5 {
        rq.push(x)
    }
    var reversed = []
    let m = rq.size()
    var t = 0
    while t < m {
        reversed.append(rq.pop())
        t += 1
    }
    check("stack reverse 0", reversed[0], 5)
    check("stack reverse 4", reversed[4], 1)

    // ring-buffer queue (FIFO), including wrap-around
    let q = RingQueue(capacity: 3)
    checkB("q enq 1", q.enqueue(1), true)
    checkB("q enq 2", q.enqueue(2), true)
    checkB("q enq 3", q.enqueue(3), true)
    checkB("q enq full", q.enqueue(4), false)
    check("q size full", q.size(), 3)
    check("q deq 1", q.dequeue(), 1)
    check("q deq 2", q.dequeue(), 2)
    checkB("q enq after deq", q.enqueue(4), true)
    checkB("q enq wrap", q.enqueue(5), true)
    check("q deq 3", q.dequeue(), 3)
    check("q deq 4", q.dequeue(), 4)
    check("q deq 5", q.dequeue(), 5)
    check("q size empty", q.size(), 0)

    // run length encoding checksum: deterministic fold over (value,run) pairs
    check("rle empty", rleChecksum([]), 0)
    check("rle single", rleChecksum([7]), 701)
    // [4,4,4,2,2] -> (4,3),(2,2): acc = (0*31 + 403) then *31 + 202 = 12695
    check("rle runs", rleChecksum([4, 4, 4, 2, 2]), 12695)

    if fails == 0 {
        print("Swift data structures self test passed")
    }
}

main()
exit(fails)
