/* Fast feature-matrix test for the Kotlin interpreter (kotlin-interpreter.abnf) and
 * the LLVM-IR compiler (kotlin-to-llvm-ir.abnf). It replaces the four algorithm-themed
 * kotlin-test-big-* stress tests: instead of large loops (sorts, Ackermann, sieves)
 * every implemented construct is exercised with the SMALLEST program that can prove it
 * works - loops run 0, 1, 3 or 4 times, recursion stays below depth 6. A failed check
 * prints its id (so a diff pinpoints it) and main() ends with exitProcess(fails);
 * exit 0 and byte-identical output on all four legs (interpreter/compiler x
 * goja/-frozen) mean everything passed. **/

var fails = 0
var checks = 0
var sideEffects = 0
var evals = 0
var finRuns = 0

fun check(id: String, cond: Boolean) {
    checks++
    if (!cond) { println("FAIL $id"); fails++ }
}

fun bumpB(): Boolean {
    sideEffects += 1
    return true
}
fun bumpN(): Int {
    evals += 1
    return 1
}

// ----- functions: expression bodies, early return, recursion -----
fun add(a: Int, b: Int): Int = a + b
fun grade(n: Int): String = if (n > 10) "big" else if (n > 5) "mid" else "small"
fun classify(n: Int): String = when {
    n < 0 -> "negative"
    n == 0 -> "zero"
    else -> "positive"
}
fun sign(n: Int): Int {
    if (n < 0) { return -1 }               // early return
    return 1
}
fun fib(n: Int): Int = if (n < 2) n else fib(n - 1) + fib(n - 2)
fun isEven(n: Int): Boolean = if (n == 0) true else isOdd(n - 1)
fun isOdd(n: Int): Boolean = if (n == 0) false else isEven(n - 1)

// A function whose parameter list carries a trailing comma (Kotlin allows it).
fun sum3(
    a: Int,
    b: Int,
    c: Int,
): Int = a + b + c

// ----- closures and higher-order functions -----
fun makeCounter(): (Int) -> Int {
    var c = 0
    return { d: Int ->
        c += d
        c
    }
}
fun applyTwice(f: (Int) -> Int, x: Int): Int = f(f(x))
fun adder(base: Int): (Int) -> Int = { x -> base + x }
fun compose(f: (Int) -> Int, g: (Int) -> Int): (Int) -> Int = { x -> f(g(x)) }

// ----- extension functions (top-level, with a body; dispatch by name) -----
fun Int.doubled(): Int = this * 2
fun String.shout(): String = this + "!"
fun String.rep(n: Int): String {
    var out = ""
    for (i in 1..n) { out = out + this }
    return out
}
fun Int?.orZero(): Int = this ?: 0         // nullable receiver: callable on null
fun Int.fold0(): Int = if (this <= 0) 0 else (this - 1).fold0() + 1   // recursion

// ----- classes -----
class Counter(val start: Int, var step: Int) {
    var value = start                      // property initializer sees the ctor params
    fun next(): Int {
        value += step                      // implicit property access
        return value
    }
    fun reset() { value = start }
}

// ----- lazy delegated properties (member level) -----
class Lazies(val n: Int) {
    val quad: Int by lazy { n * n }        // initializer reads a constructor param
    val sum: Int by lazy {                 // multi-statement initializer body
        var t = 0
        for (i in 1..n) { t = t + i }
        t
    }
}

data class Point(val x: Int, val y: Int) {
    fun manhattan(): Int = abs(x) + abs(y)
    fun shifted(dx: Int): Point = Point(x + dx, y)   // returns a fresh instance
}

class Node(val value: Int) {
    var next: Node? = null
}

object Registry {                          // object singleton with mutable state
    val name = "reg"
    var count = 0
    fun add(d: Int): Int {
        count = count + d                  // implicit property write against this
        return count
    }
    fun label(): String = name + ":" + count
    fun addTwice(d: Int): Int {            // calls sibling methods via this.m()
        this.add(d)
        this.add(d)
        return count
    }
}

class Account(val owner: String) {
    var funds = 0
    fun deposit(n: Int): Int {
        funds = funds + n
        return funds
    }
    companion object {
        val CODE = 42
        var opened = 0
        fun open(): Int {
            opened = opened + 1            // mutable companion state
            return CODE * 10 + opened
        }
        fun code(): Int = CODE
    }
}

// ----- exceptions -----
class BoomException(val code: Int)

fun risky(n: Int): Int {
    if (n > 3) { throw BoomException(n) }  // unwinds out of the call
    return n * 2
}
fun rethrow(): String {
    var result = ""
    try {
        try { throw "deep" } catch (e: Exception) { throw e + "er" }
    } catch (e2: Exception) {
        result = e2
    }
    return result
}
fun retAcrossTry(): String {
    try { return "from-try" } finally { finRuns += 1 }
}
fun retOutOfCatch(n: Int): Int {
    try {
        if (n > 0) { return n * 10 }       // return out of the try
        throw "neg"
    } catch (e: Exception) {
        return -1                          // return out of the catch
    } finally {
        finRuns += 1                       // runs on both paths
    }
}
fun nestedReturn(): Int {
    try {
        try { return 9 } finally { }
    } finally { }
    return 0
}
fun retInFinally(): Int {
    try { return 1 } finally { return 2 }  // the finally's return overrides
}
fun finCancelsThrow(): String {
    try { throw "boom" } finally { return "fin" }   // cancels the pending throw
}
fun breakInFinally(): Int {
    var i = 0
    while (true) {
        i = i + 1
        try { i = i + 10 } finally { break }
    }
    return i
}
fun continueInFinally(): Int {
    var sum = 0
    for (i in 0..2) {
        try { if (i == 1) { throw "skip" } } finally { continue }
        sum = sum + 100                    // never reached: continue jumps on
    }
    return sum
}
fun loopBreakOutOfTry(): Int {
    var sum = 0
    for (i in 0..5) {
        try {
            if (i == 3) { break }
            sum = sum + i
        } finally { }
    }
    return sum                             // 0+1+2 = 3
}
fun loopContinueOutOfTry(): Int {
    var sum = 0
    for (i in 0..3) {
        try {
            if (i == 2) { continue }
            sum = sum + i
        } catch (e: Exception) { }
    }
    return sum                             // 0+1+3 = 4
}

// ----- everything combined in one small pipeline (3-element data flow) -----
fun transform(list: List<Int>): String {
    var out = ""
    for (n in list) {
        try {
            if (n < 0) { throw "neg" }
            val piece = when {
                n % 2 == 0 -> "e$n"
                else -> "o$n"
            }
            out = out + piece
        } catch (e: Exception) {
            out = out + "x"
        }
    }
    return out
}

fun main() {
    // ----- numbers, arithmetic, precedence -----
    check("arith-precedence", 2 + 3 * 4 == 14)
    check("lit-hex-bin", 0xFF == 255 && 0b1010 == 10)
    check("lit-underscore", 1_000_000 == 1000000)
    check("lit-long", 3_000_000_000L > 2_999_999_999L)
    check("lit-double", 1.5 > 1.4 && 2.5e-2 < 0.03 && 1.5f < 1.6f)
    check("lit-char", 'A' + 1 == 'B' && 'z' > 'a' && '\n' < 'a')
    check("is-check", (5 is Int) && (5 !is String) && ("x" is String))
    check("as-safe", ("t" as? Int) == null && (7 as? Int) == 7)
    check("raw-string", """a"b
c""".length == 5 && """v=${2 + 3}""" == "v=5")
    check("arith-paren", (2 + 3) * 4 == 20)
    check("arith-unary-minus", -3 + 5 == 2)
    check("arith-div-trunc", 7 / 2 == 3)
    check("arith-div-neg", -7 / 2 == -3)
    check("arith-mod", 7 % 3 == 1)
    check("arith-mod-neg", -7 % 3 == -1)
    check("arith-chain", 20 - 5 - 3 == 12)
    var cx = 5
    cx += 3
    cx -= 2
    cx *= 4
    cx /= 6
    cx %= 3
    check("arith-compound", cx == 1)
    var inc = 5
    inc++
    inc++
    inc--
    check("arith-incdec", inc == 6)

    // ----- comparison, equality, logic -----
    check("cmp-ops", 5 > 3 && 3 >= 3 && 2 < 3 && 2 <= 2 && 1 != 2)
    check("eq-structural-string", "ab" + "c" == "abc")
    val refStr = "kotlin"
    check("eq-referential", refStr === refStr && refStr !== "other")
    val boxed: Int = 41
    check("notnull-assert", boxed!! + 1 == 42)
    val noRun = false && bumpB()
    val oneRun = true && bumpB()
    val skipRun = true || bumpB()
    check("logic-short-circuit", sideEffects == 1 && !noRun && oneRun && skipRun)
    check("logic-not", !(2 == 3) && !false)
    val ifA = if (5 > 3) "a" else "b"
    val ifB = if (5 < 3) "a" else "b"
    check("if-expr", ifA == "a" && ifB == "b")

    // ----- stdlib infix operators: to / and / or / xor / shl / shr / ushr -----
    val pair = 3 to "three"
    check("infix-to", pair.first == 3 && pair.second == "three")
    val nested = 1 to 2 to 3                 // left-assoc: (1 to 2) to 3
    check("infix-to-chain", nested.first.second == 2 && nested.second == 3)
    check("infix-bitwise", (12 and 10) == 8 && (12 or 3) == 15 && (12 xor 10) == 6)
    check("infix-shift", (1 shl 5) == 32 && (-16 shr 2) == -4 && ((0 - 1) ushr 28) == 15)
    check("infix-bool", (true and false) == false && (false or true) == true && (true xor true) == false)
    check("infix-precedence", (1 shl 2 + 1) == 8)   // + binds tighter: 1 shl 3
    val pairs = listOf(1 to 10, 2 to 20)
    check("infix-to-in-list", pairs[1].first == 2 && pairs[1].second == 20)

    // ----- strings and templates -----
    check("str-concat", "foo" + "bar" == "foobar")
    check("str-int-concat", "n=" + 42 == "n=42" && 1 + 2 + "x" == "3x")
    check("str-length", "hello".length == 5 && "".length == 0)
    check("str-unicode-len", "héllo".length == 5)
    val tpl = 6
    check("str-template", "v=${tpl + 1}!" == "v=7!" && "plain $tpl" == "plain 6")
    check("str-template-nested", "x=${if (tpl > 5) "big" else "small"}" == "x=big")
    check("str-escapes", "a\tb".length == 3 && "a\nb".length == 3 && "\\".length == 1 && "\"".length == 1)

    // ----- control flow: if / while / do-while -----
    check("if-elseif-else", grade(11) == "big" && grade(7) == "mid" && grade(1) == "small")
    var w0 = 0
    while (w0 > 0) { w0 = w0 - 1 }         // runs zero times
    check("while-zero", w0 == 0)
    var w3 = 0
    while (w3 < 3) { w3 = w3 + 1 }         // runs three times
    check("while-three", w3 == 3)
    var dw = 0
    do { dw = dw + 1 } while (false)       // body runs exactly once
    check("do-while-once", dw == 1)

    // ----- for over ranges and lists -----
    var forSum = 0
    for (i in 1..3) { forSum += i }
    check("for-range", forSum == 6)
    var fe = 0
    for (i in 1..0) { fe += 1 }            // empty range: zero iterations
    check("for-range-empty", fe == 0)
    var fu = 0
    for (i in 0 until 4) { fu += i }
    check("for-until", fu == 6)
    var fd = 0
    for (i in 3 downTo 1) { fd = fd * 10 + i }
    check("for-downto", fd == 321)
    var fs = 0
    for (i in 0..10 step 3) { fs += i }
    check("for-step", fs == 18)
    var fus = 0
    for (i in 10 until 20 step 4) { fus += i }
    check("for-until-step", fus == 42)
    var fds = 0
    for (i in 9 downTo 1 step 3) { fds = fds * 10 + i }
    check("for-downto-step", fds == 963)
    var brk = ""
    for (i in 0..5) {
        if (i == 2) { break }
        brk = brk + i
    }
    check("for-break", brk == "01")
    var cont = ""
    for (i in 0..3) {
        if (i % 2 == 1) { continue }
        cont = cont + i
    }
    check("for-continue", cont == "02")
    var nested = ""
    for (oi in 0..1) {
        for (ii in 0..2) {
            if (ii == 1) { break }         // inner break must not end the outer loop
            nested = nested + oi + ii
        }
    }
    check("nested-break", nested == "0010")
    var listSum = 0
    for (x in listOf(4, 5, 6)) { listSum += x }
    check("for-list", listSum == 15)

    // ----- when: subject, comma values, ranges, statement form -----
    val ws = when (10) {
        1, 2 -> 100
        10 -> 200
        else -> 300
    }
    check("when-subject", ws == 200)
    val wm = when (7) {
        1, 2 -> 100
        else -> 300
    }
    check("when-else", wm == 300)
    check("when-no-subject", classify(-5) == "negative" && classify(0) == "zero" && classify(3) == "positive")
    val wr = when (85) { in 90..100 -> "A"; in 80..89 -> "B"; else -> "C" }
    check("when-range", wr == "B")
    val wrm = when (42) { in 90..100 -> "A"; in 80..89 -> "B"; else -> "C" }
    check("when-range-miss", wrm == "C")
    val wn = when (5) { !in 0..3 -> "out"; else -> "in" }
    val wnm = when (2) { !in 0..3 -> "out"; else -> "in" }
    check("when-not-in", wn == "out" && wnm == "in")
    val wstr = when ("sun") { "sat", "sun" -> "weekend"; else -> "workday" }
    check("when-string-subject", wstr == "weekend")
    var bucket = 0
    when (3) {
        1, 2 -> bucket = 1
        else -> bucket = 2
    }
    check("when-stmt", bucket == 2)

    // ----- functions, recursion, closures, lambdas -----
    check("fn-expr-body", add(20, 22) == 42)
    check("fn-early-return", sign(-9) == -1 && sign(9) == 1)
    check("fn-recursion", fib(6) == 8)
    check("fn-mutual-recursion", isEven(4) && isOdd(5))
    val c1 = makeCounter()
    val c2 = makeCounter()
    c1(1)
    c1(1)
    check("closure-independent", c1(1) == 3 && c2(1) == 1)
    var acc = 10
    val plusAcc = { d: Int -> acc + d }
    check("closure-read", plusAcc(5) == 15)
    acc = 100
    check("closure-sees-update", plusAcc(5) == 105)
    val twice = { x: Int -> x * 2 }
    val addF = { a: Int, b: Int -> a + b }
    check("lambda-forms", twice(21) == 42 && addF(19, 23) == 42)
    val multi = { x: Int ->                // multi-statement lambda body
        val a = x + 1
        val b = a * a
        b - 1
    }
    check("lambda-multi-stmt", multi(4) == 24)
    check("fn-higher-order", applyTwice({ n: Int -> n * 2 }, 3) == 12)
    val add10 = adder(10)
    check("fn-factory", add10(5) == 15)
    val incF = { x: Int -> x + 1 }
    val dblThenInc = compose(incF, twice)  // incF(twice(x))
    check("fn-compose", dblThenInc(4) == 9)

    // ----- list higher-order methods (trailing lambdas, `it`) -----
    val nums = listOf(3, 1, 4)
    val doubled = nums.map { it * 2 }
    check("hof-map", doubled[0] == 6 && doubled[2] == 8)
    check("hof-map-chain", nums.map { it + 1 }.sumOf { it } == 11)
    check("hof-filter", nums.filter { it % 2 == 1 }.size == 2)
    check("hof-filter-parens", nums.filter({ it > 2 }).size == 2)
    check("hof-sumof", nums.sumOf { it * 10 } == 80)
    check("hof-count", nums.count() == 3 && nums.count { it == 1 } == 1)
    check("hof-any", nums.any { it > 3 } && !nums.any { it > 9 })
    var feSum = 0
    nums.forEach { feSum += it }
    check("hof-foreach", feSum == 8)

    // ----- callable references (bare ::fn; qualified refs stay unsupported) -----
    val addRef = ::add
    check("callable-ref-call", addRef(2, 3) == 5)
    check("callable-ref-as-arg", nums.map(::sign).sumOf { it } == 3)
    check("callable-ref-mixed", applyTwice(::sign, -9) == -1)   // sign(sign(-9)) = sign(-1)

    // ----- labelled returns (innermost lambda / enclosing function) -----
    var lrSum = 0
    nums.forEach {
        if (it == 3) return@forEach          // continue-like skip
        lrSum += it
    }
    check("lret-foreach-skip", lrSum == 5)   // 3 + 1 + 4 minus the skipped 3
    val lrMapped = nums.map {
        if (it % 2 == 0) return@map it * 10  // early value
        it
    }
    check("lret-map-value", lrMapped[0] == 3 && lrMapped[1] == 1 && lrMapped[2] == 40)
    val lrPicked = nums.filter {
        if (it > 2) return@filter true
        return@filter false
    }
    check("lret-filter-paths", lrPicked.size == 2)

    // ----- extension functions -----
    check("ext-int", 21.doubled() == 42)
    check("ext-string", "hi".shout() == "hi!")
    check("ext-args", "ab".rep(2) == "abab")
    check("ext-chain", 3.doubled().doubled() == 12)
    val extNull: Int? = null
    check("ext-nullable-recv", extNull.orZero() == 0 && 5.orZero() == 5)
    check("ext-safe-skip", extNull?.doubled() == null)
    check("ext-recursion", 6.fold0() == 6)

    // ----- object expressions (anonymous instances; supertypes ignored) -----
    val oeBase = 30
    val oe = object : Comparable<Int> {
        var hits = 0
        fun tick(): Int {
            hits = hits + 1
            return hits
        }
        fun plus(x: Int): Int = oeBase + x   // captures the enclosing local
    }
    oe.tick()
    check("objexpr-state", oe.tick() == 2 && oe.hits == 2)
    check("objexpr-capture", oe.plus(12) == 42)
    val oe2 = object { val seed = oeBase + 1; fun read(): Int = seed }
    check("objexpr-prop-init", oe2.read() == 31)

    // ----- lazy delegated properties (eager: initializer runs once, value cached) -----
    val lazyK = 6
    val lazyLocal: Int by lazy { lazyK * 7 }   // closes over the enclosing local
    check("lazy-local", lazyLocal == 42)
    val lazyModed: Int by lazy(LazyThreadSafetyMode.NONE) { 5 + 5 }
    check("lazy-moded", lazyModed == 10)
    val lz = Lazies(4)
    check("lazy-member", lz.quad == 16 && lz.sum == 10)

    // ----- lists -----
    val ro = listOf(10, 20, 30)
    check("list-literal", ro.size == 3 && ro[0] == 10 && ro[2] == 30)
    val ml = mutableListOf(1, 2)
    ml.add(3)
    check("list-add", ml.size == 3 && ml.get(2) == 3)
    check("list-contains", ml.contains(2) && !ml.contains(9))
    ml[0] = 9
    check("list-write", ml[0] == 9)
    val grid = mutableListOf(mutableListOf(1, 2), mutableListOf(3))
    check("list-nested", grid[0][1] == 2 && grid[1][0] == 3)
    grid[0][0] = 7
    check("list-nested-write", grid[0][0] == 7)

    // ----- trailing commas (arg lists, params, lambda params, when, listOf) -----
    check("trailing-fn-params", sum3(1, 2, 3,) == 6)
    val tcList = listOf(10, 20, 30,)
    check("trailing-arg-list", tcList.size == 3 && tcList[2] == 30)
    val tcAdd = add(4, 5,)
    check("trailing-call", tcAdd == 9)
    val tcLam = tcList.map { n, -> n + 1 }   // trailing comma after a lambda param
    check("trailing-lambda-param", tcLam[0] == 11)
    val tcWhen = when (2) {
        1, 2, -> "low"
        else -> "high"
    }
    check("trailing-when-cond", tcWhen == "low")

    // ----- nullability: elvis and safe calls -----
    var maybe: Int? = null
    check("elvis-null", (maybe ?: 42) == 42)
    maybe = 7
    check("elvis-value", (maybe ?: 42) == 7)
    check("elvis-short-circuit", (maybe ?: bumpN()) == 7 && evals == 0)
    var str: String? = null
    check("safe-call-null", (str?.length ?: -1) == -1)
    str = "abc"
    check("safe-call-value", (str?.length ?: -1) == 3)
    val noPoint: Point? = null
    check("safe-call-skips-args", (noPoint?.shifted(bumpN())?.x ?: -1) == -1 && evals == 0)
    val yesPoint: Point? = Point(3, -4)
    check("safe-call-chain", (yesPoint?.shifted(bumpN())?.x ?: -1) == 4 && evals == 1)
    val head = Node(1)
    head.next = Node(2)
    check("safe-chain-nodes", (head.next?.value ?: -1) == 2 && (head.next?.next?.value ?: -1) == -1)

    // ----- classes, object singleton, companion -----
    val ctr = Counter(10, 1)
    check("class-ctor-props", ctr.start == 10 && ctr.value == 10)
    check("class-method", ctr.next() == 11)
    ctr.step = 5
    check("class-prop-write", ctr.next() == 16)
    ctr.reset()
    check("class-method-reset", ctr.value == 10)
    val pt = Point(3, -4)
    check("data-class", pt.manhattan() == 7 && pt.x - pt.y == 7)
    check("class-returns-instance", pt.shifted(2).x == 5)
    check("object-fields", Registry.name == "reg" && Registry.count == 0)
    check("object-method", Registry.add(5) == 5 && Registry.count == 5)
    check("object-expr-body", Registry.label() == "reg:5")
    check("object-this-dispatch", Registry.addTwice(2) == 9)
    check("companion-const", Account.CODE == 42 && Account.code() == 42)
    check("companion-mutable", Account.open() == 421 && Account.open() == 422 && Account.opened == 2)
    val acct = Account("ada")
    check("companion-instance-side", acct.owner == "ada" && acct.deposit(7) == 7 && Account.CODE == 42)

    // ----- builtins -----
    check("builtin-abs-max-min", abs(-6) == 6 && max(3, 9) == 9 && min(3, 9) == 3)

    // ----- exceptions: throw / catch / finally / control flow -----
    var exOrder = ""
    try {
        exOrder = exOrder + "t"
        throw "boom"
    } catch (e: Exception) {
        exOrder = exOrder + "c" + e
    } finally {
        exOrder = exOrder + "f"
    }
    check("try-throw-catch-finally", exOrder == "tcboomf")
    var noThrow = ""
    try { noThrow = noThrow + "t" } catch (e: Exception) { noThrow = noThrow + "c" } finally { noThrow = noThrow + "f" }
    check("try-no-throw", noThrow == "tf")
    var caught = -1
    try {
        risky(5)
        caught = -2                        // not reached
    } catch (e: BoomException) {
        caught = e.code
    }
    check("throw-unwinds-call", caught == 5)
    check("throw-no-throw-path", risky(2) == 4)
    check("rethrow", rethrow() == "deeper")
    check("return-across-try", retAcrossTry() == "from-try" && finRuns == 1)
    check("return-out-of-catch", retOutOfCatch(4) == 40 && retOutOfCatch(-1) == -1 && finRuns == 3)
    check("nested-return", nestedReturn() == 9)
    check("return-in-finally", retInFinally() == 2)
    check("finally-cancels-throw", finCancelsThrow() == "fin")
    check("break-in-finally", breakInFinally() == 11)
    check("continue-in-finally", continueInFinally() == 0)
    check("loop-break-out-of-try", loopBreakOutOfTry() == 3)
    check("loop-continue-out-of-try", loopContinueOutOfTry() == 4)

    // ----- labeled loops -----
    var labHits = 0
    outer@ for (i in 0..2) {
        for (j in 0..2) {
            if (j == 1) continue@outer
            if (i == 2) break@outer
            labHits = labHits + 1
        }
    }
    check("labeled-loop", labHits == 2)
    var labN = 0
    var labSeen = 0
    cw@ while (labN < 4) {
        labN = labN + 1
        if (labN == 2) continue@cw
        labSeen = labSeen + 1
    }
    check("labeled-while", labN == 4 && labSeen == 3)

    // ----- everything combined -----
    check("combined-pipeline", transform(listOf(1, 2, -3)) == "o1e2x")

    println("features: $checks checks, $fails failures")
    exitProcess(fails)
}
