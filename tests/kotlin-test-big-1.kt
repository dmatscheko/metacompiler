/* Kotlin subset self test - THEME 1: data structures & algorithms.
 *
 * Exercises: in-place sorting (bubble / insertion / selection), binary search,
 * an array-backed stack and a queue built as classes over mutableListOf, matrix
 * multiplication / transpose / trace over lists-of-lists (with element writes
 * m[i][j] = v), and list utilities cross-checked between manual loops and the
 * higher-order methods (map/filter/sumOf/count/any). main() counts failed checks
 * and ends with exitProcess(fails), so the run exits 0 exactly when all pass. **/

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) { println("FAIL $name: got $got want $want"); fails++ }
}
fun checkB(name: String, got: Boolean, want: Boolean) {
    check(name, if (got) 1 else 0, if (want) 1 else 0)
}

// ----- list helpers -----

fun copyList(a: List<Int>): List<Int> {
    val r = mutableListOf()
    for (x in a) { r.add(x) }
    return r
}
fun swap(a: List<Int>, i: Int, j: Int) {
    val t = a[i]
    a[i] = a[j]
    a[j] = t
}
fun isSorted(a: List<Int>): Boolean {
    for (i in 1 until a.size) { if (a[i - 1] > a[i]) { return false } }
    return true
}
fun listEq(a: List<Int>, b: List<Int>): Boolean {
    if (a.size != b.size) { return false }
    for (i in 0 until a.size) { if (a[i] != b[i]) { return false } }
    return true
}

// ----- three sorts, all in place -----

fun bubbleSort(a: List<Int>) {
    val n = a.size
    for (i in 0 until n - 1) {
        for (j in 0 until n - 1 - i) {
            if (a[j] > a[j + 1]) { swap(a, j, j + 1) }
        }
    }
}
fun insertionSort(a: List<Int>) {
    for (i in 1 until a.size) {
        val key = a[i]
        var j = i - 1
        while (j >= 0 && a[j] > key) {
            a[j + 1] = a[j]
            j -= 1
        }
        a[j + 1] = key
    }
}
fun selectionSort(a: List<Int>) {
    val n = a.size
    for (i in 0 until n - 1) {
        var mi = i
        for (j in i + 1 until n) {
            if (a[j] < a[mi]) { mi = j }
        }
        if (mi != i) { swap(a, i, mi) }
    }
}

// ----- binary search on a sorted list -----

fun binarySearch(a: List<Int>, target: Int): Int {
    var lo = 0
    var hi = a.size - 1
    while (lo <= hi) {
        val mid = (lo + hi) / 2
        val v = a[mid]
        if (v == target) { return mid }
        if (v < target) { lo = mid + 1 } else { hi = mid - 1 }
    }
    return -1
}

// ----- array-backed stack (grows via add, reuses slots after pop) -----

class IntStack {
    val data = mutableListOf()
    var count = 0

    fun push(x: Int) {
        if (count < data.size) { data[count] = x } else { data.add(x) }
        count += 1
    }
    fun pop(): Int {
        count -= 1
        return data[count]
    }
    fun peek(): Int = data[count - 1]
    fun isEmpty(): Boolean = count == 0
    fun size(): Int = count
}

// ----- queue with a moving head index -----

class IntQueue {
    val data = mutableListOf()
    var head = 0

    fun enqueue(x: Int) { data.add(x) }
    fun dequeue(): Int {
        val v = data[head]
        head += 1
        return v
    }
    fun isEmpty(): Boolean = head >= data.size
    fun size(): Int = data.size - head
}

// ----- matrices as lists of lists -----

fun matMul(a: List<List<Int>>, b: List<List<Int>>, n: Int): List<List<Int>> {
    val res = mutableListOf()
    for (i in 0 until n) {
        val row = mutableListOf()
        for (j in 0 until n) {
            var s = 0
            for (k in 0 until n) { s += a[i][k] * b[k][j] }
            row.add(s)
        }
        res.add(row)
    }
    return res
}
fun transpose(a: List<List<Int>>, n: Int): List<List<Int>> {
    val res = mutableListOf()
    for (i in 0 until n) {
        val row = mutableListOf()
        for (j in 0 until n) { row.add(a[j][i]) }
        res.add(row)
    }
    return res
}
fun trace(a: List<List<Int>>, n: Int): Int {
    var s = 0
    for (i in 0 until n) { s += a[i][i] }
    return s
}
fun identity(n: Int): List<List<Int>> {
    val res = mutableListOf()
    for (i in 0 until n) {
        val row = mutableListOf()
        for (j in 0 until n) { row.add(if (i == j) 1 else 0) }
        res.add(row)
    }
    return res
}
fun matEq(a: List<List<Int>>, b: List<List<Int>>, n: Int): Boolean {
    for (i in 0 until n) {
        for (j in 0 until n) { if (a[i][j] != b[i][j]) { return false } }
    }
    return true
}

fun main() {
    // ---- sorting: three algorithms on independent copies of one permutation ----
    val src = listOf(9, 3, 7, 1, 8, 2, 6, 5, 4, 0)

    val b1 = copyList(src)
    bubbleSort(b1)
    checkB("bubble sorted", isSorted(b1), true)

    val b2 = copyList(src)
    insertionSort(b2)
    checkB("insertion sorted", isSorted(b2), true)

    val b3 = copyList(src)
    selectionSort(b3)
    checkB("selection sorted", isSorted(b3), true)

    checkB("sorts agree 1", listEq(b1, b2), true)
    checkB("sorts agree 2", listEq(b2, b3), true)
    // src is a permutation of 0..9, so element i lands at index i.
    check("sorted first", b1[0], 0)
    check("sorted last", b1[9], 9)
    check("sorted middle", b1[5], 5)

    // ---- binary search on the sorted result ----
    for (t in 0..9) { check("bsearch $t", binarySearch(b1, t), t) }
    check("bsearch miss high", binarySearch(b1, 10), -1)
    check("bsearch miss low", binarySearch(b1, -1), -1)

    // ---- stack: LIFO behaviour + slot reuse after pops ----
    val st = IntStack()
    checkB("stack empty init", st.isEmpty(), true)
    st.push(10)
    st.push(20)
    st.push(30)
    check("stack size", st.size(), 3)
    check("stack peek", st.peek(), 30)
    check("stack pop1", st.pop(), 30)
    check("stack pop2", st.pop(), 20)
    st.push(99)
    check("stack pop after reuse", st.pop(), 99)
    check("stack pop3", st.pop(), 10)
    checkB("stack empty end", st.isEmpty(), true)

    // reverse 1..6 through the stack
    val rst = IntStack()
    for (i in 1..6) { rst.push(i) }
    var rev = 0
    while (!rst.isEmpty()) { rev = rev * 10 + rst.pop() }
    check("stack reverse", rev, 654321)

    // ---- queue: FIFO behaviour ----
    val q = IntQueue()
    checkB("queue empty init", q.isEmpty(), true)
    for (i in 1..5) { q.enqueue(i * i) }
    check("queue size", q.size(), 5)
    check("queue deq1", q.dequeue(), 1)
    check("queue deq2", q.dequeue(), 4)
    check("queue size after 2", q.size(), 3)
    var qsum = 0
    while (!q.isEmpty()) { qsum += q.dequeue() }
    check("queue drain sum", qsum, 9 + 16 + 25)

    // ---- matrices ----
    val a = mutableListOf(mutableListOf(1, 2), mutableListOf(3, 4))
    val bb = mutableListOf(mutableListOf(5, 6), mutableListOf(7, 8))
    val prod = matMul(a, bb, 2)
    check("matmul 00", prod[0][0], 19)
    check("matmul 01", prod[0][1], 22)
    check("matmul 10", prod[1][0], 43)
    check("matmul 11", prod[1][1], 50)
    check("matmul trace", trace(prod, 2), 69)

    // element write into a nested list
    prod[0][0] = 100
    check("nested write", prod[0][0], 100)

    // A * I == A for a 3x3
    val m3 = mutableListOf(
        mutableListOf(1, 2, 3),
        mutableListOf(4, 5, 6),
        mutableListOf(7, 8, 9)
    )
    val id3 = identity(3)
    checkB("A*I == A", matEq(matMul(m3, id3, 3), m3, 3), true)
    check("trace 3x3", trace(m3, 3), 15)
    // transpose is an involution
    checkB("transpose twice == A", matEq(transpose(transpose(m3, 3), 3), m3, 3), true)
    check("transpose element", transpose(m3, 3)[0][2], 7)

    // ---- list utilities: manual loops vs higher-order ----
    val nums = listOf(4, 8, 15, 16, 23, 42)
    var manualSum = 0
    for (x in nums) { manualSum += x }
    check("sum manual vs sumOf", manualSum, nums.sumOf { it })
    var manualEven = 0
    for (x in nums) { if (x % 2 == 0) { manualEven += 1 } }
    check("even count", manualEven, nums.count { it % 2 == 0 })
    checkB("any big", nums.any { it > 40 }, true)
    checkB("any huge", nums.any { it > 100 }, false)
    check("filter+map+sumOf", nums.filter { it % 2 == 0 }.map { it / 2 }.sumOf { it }, 2 + 4 + 8 + 21)

    if (fails == 0) { println("Kotlin big-1 (data structures) passed") }
    exitProcess(fails)
}
