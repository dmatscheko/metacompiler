// Full-syntax test: Kotlin (Kotlin 2.x core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the kotlin grammars
// are. It walks the whole practical Kotlin syntax, one self-contained SECTION
// per language area. The --full runner runs the file, and whenever a grammar
// aborts it removes the section around the error and retries - so the report
// lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() ends with exitProcess(fails) (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// packages and imports (println/listOf/exitProcess are builtins here, exactly
// as in kotlin-test-features.kt; under kotlinc this file additionally needs
// 'import kotlin.system.exitProcess'), reflection, the collections API beyond
// the minimal calls the features file already uses, KDoc, and multiplatform
// (expect/actual). The coroutine runtime is a SYNCHRONOUS subset (section 55):
// the results are Kotlin's and the ordering is not - section 55 asserts the
// eager ordering deliberately, so a change to it cannot pass unnoticed.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Kotlin language specification with the ANTLR
// grammars-v4 kotlin grammar as a coverage checklist. Validated against the
// spec by hand; no local kotlinc was available.

var fails = 0
var checks = 0

fun check(id: String, cond: Boolean) {
    checks++
    if (!cond) { println("FAIL $id"); fails++ }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
class Box01(val start: Int) {
    var value = start
    fun bump(): Int { value += 1; return value }
}
fun twice01(x: Int): Int = x * 2
fun s01() {
    var n = 0
    for (i in 1..3) { n += i }
    var w = 0
    do { w += 1 } while (w < 2)
    while (w < 4) { w += 2 }
    check("bas1", n == 6 && w == 4)
    val b = Box01(4)
    check("bas2", b.bump() == 5 && b.value == 5)
    check("bas3", twice01(21) == 42)
    check("bas4", "n=$n" == "n=6")
    val kind = when (n) { in 0..5 -> "low"; else -> "high" }
    check("bas5", kind == "high")
}

// ===== SECTION 02: literals and constants =====
const val MAX02 = 100
const val NAME02 = "lit"
fun s02() {
    check("lit1", 0xFF == 255 && 0b1010 == 10)
    check("lit2", 1_000_000 == 1000000)
    check("lit3", 3_000_000_000L > 2_999_999_999L)
    val f = 1.5f; val d = 2.5e-2
    check("lit4", f > 1.4f && f < 1.6f && d > 0.02 && d < 0.03)
    val c: Char = 'A'
    check("lit5", c + 1 == 'B' && 'z' > 'a' && '\n' < 'a')
    check("lit6", MAX02 == 100 && NAME02.length == 3)
}

// ===== SECTION 03: string templates and raw strings =====
fun s03() {
    val x = 6
    check("tpl1", "${x} + ${x} = ${x + x}" == "6 + 6 = 12")
    val raw = """line1
line2"""
    check("tpl2", raw.length == 11)
    check("tpl3", """no \n escape""".length == 12)
    val price = 5
    check("tpl4", """total: ${'$'}$price""" == "total: \$5")
    check("tpl5", "he said \"hi\"".length == 12)
    check("tpl6", "outer ${"inner $x"}" == "outer inner 6")
}

// ===== SECTION 04: ranges and progressions =====
fun s04() {
    var a = 0; for (i in 1..4) { a = a * 10 + i }
    check("rng1", a == 1234)
    var b = 0; for (i in 0..<3) { b += i }
    check("rng2", b == 3)
    var c = ""; for (i in 3 downTo 1) { c += i }
    check("rng3", c == "321")
    var d = 0; for (i in 0..6 step 2) { d += i }
    check("rng4", d == 12)
    check("rng5", 3 in 1..5 && 9 !in 1..5)
    var g = ""; for (ch in 'a'..'c') { g += ch }
    check("rng6", g == "abc")
}

// ===== SECTION 05: when forms =====
fun describe05(x: Any): String = when (x) {
    is Int -> if (x > 5) "big int" else "small int"
    is String -> "str" + x.length
    else -> "other"
}
fun s05() {
    val a = when (10) { 1, 2 -> "lo"; 10 -> "ten"; else -> "hi" }
    check("whn1", a == "ten")
    val b = when (7) { in 0..5 -> "low"; in 6..9 -> "mid"; else -> "high" }
    check("whn2", b == "mid")
    check("whn3", describe05(9) == "big int" && describe05("ab") == "str2" && describe05(true) == "other")
    val c = when (val n = 4 * 4) { in 0..9 -> "1digit:$n"; else -> "big:$n" }
    check("whn4", c == "big:16")
    val t = 3
    val e = when { t < 0 -> "neg"; t % 2 == 1 -> "odd"; else -> "even" }
    check("whn5", e == "odd")
}

// ===== SECTION 06: labels and labeled returns =====
fun firstBig06(xs: List<Int>): Int {
    xs.forEach { if (it > 10) return it }
    return -1
}
fun exec06(f: () -> Unit) { f() }
fun s06() {
    var hits = 0
    outer@ for (i in 0..2) {
        for (j in 0..2) {
            if (j == 1) continue@outer
            if (i == 2) break@outer
            hits++
        }
    }
    check("lop1", hits == 2)
    val sum = listOf(1, 2, 3, 4).sumOf {
        if (it == 3) return@sumOf 0
        it
    }
    check("lop2", sum == 7)
    check("lop3", firstBig06(listOf(4, 40, 9)) == 40)
    var log = ""
    exec06 lab@{
        log += "a"
        if (log.length == 1) return@lab
        log += "b"
    }
    check("lop4", log == "a")
}

// ===== SECTION 07: functions: defaults, named args, vararg, tailrec =====
fun volume07(w: Int, h: Int = 2, d: Int = 3): Int = w * 100 + h * 10 + d
fun sumAll07(vararg xs: Int): Int { var s = 0; for (x in xs) { s += x }; return s }
tailrec fun gcd07(a: Int, b: Int): Int = if (b == 0) a else gcd07(b, a % b)
fun s07() {
    check("fun1", volume07(1) == 123)
    check("fun2", volume07(1, 9) == 193)
    check("fun3", volume07(1, d = 7) == 127)
    check("fun4", volume07(d = 5, w = 4, h = 6) == 465)
    check("fun5", sumAll07() == 0 && sumAll07(1, 2, 3) == 6)
    val packed = intArrayOf(4, 5)
    check("fun6", sumAll07(*packed, 6) == 15)
    check("fun7", gcd07(12, 18) == 6)
    fun local(n: Int): Int = n + packed[0]
    check("fun8", local(6) == 10)
}

// ===== SECTION 08: lambdas and function types =====
data class Duo08(val a: Int, val b: Int)
fun combine08(x: Int, y: Int, f: (Int, Int) -> Int): Int = f(x, y)
fun s08() {
    val inc = { n: Int -> n + 1 }
    check("lam1", inc(41) == 42)
    val doubled = listOf(1, 2, 3).map { it * 2 }
    check("lam2", doubled[2] == 6)
    check("lam3", combine08(3, 4) { a, b -> a * 10 + b } == 34)
    check("lam4", combine08(5, 9) { _, b -> b } == 9)
    val anon = fun(n: Int): Int { return n * n }
    check("lam5", anon(6) == 36)
    val dot: Int.(Int) -> Int = { other -> this + other }
    check("lam6", 40.dot(2) == 42 && dot(1, 2) == 3)
    val folded = listOf(Duo08(1, 2), Duo08(3, 4)).map { (a, b) -> a * 10 + b }
    check("lam7", folded[0] == 12 && folded[1] == 34)
    val slot: ((Int) -> Int)? = if (doubled.size == 3) inc else null
    check("lam8", slot?.invoke(1) == 2)
}

// ===== SECTION 09: operator conventions and infix functions =====
class Vec09(val x: Int, val y: Int) {
    operator fun plus(o: Vec09) = Vec09(x + o.x, y + o.y)
    operator fun times(k: Int) = Vec09(x * k, y * k)
    operator fun get(i: Int): Int = if (i == 0) x else y
    operator fun invoke(): Int = x * 10 + y
    operator fun unaryMinus() = Vec09(-x, -y)
    operator fun contains(n: Int): Boolean = n == x || n == y
    infix fun dot(o: Vec09): Int = x * o.x + y * o.y
}
fun s09() {
    val v = Vec09(1, 2) + Vec09(3, 4)
    check("opr1", v.x == 4 && v.y == 6)
    val w = Vec09(2, 3) * 3
    check("opr2", w[0] == 6 && w[1] == 9)
    check("opr3", w() == 69)
    val m = -Vec09(5, 7)
    check("opr4", m.x == -5 && 7 in Vec09(5, 7) && 9 !in Vec09(5, 7))
    check("opr5", (Vec09(1, 0) dot Vec09(3, 9)) == 3)
    var acc = Vec09(1, 1); acc += Vec09(2, 2)
    check("opr6", acc.x == 3 && acc.y == 3)
    check("bit1", (5 and 3) == 1 && (5 or 3) == 7 && (5 xor 3) == 6)
    check("bit2", (1 shl 4) == 16 && (32 shr 1) == 16)
    check("bit3", (-8 ushr 28) == 15)
    check("bit4", 5.inv() == -6)
}

// ===== SECTION 10: extension functions and properties =====
fun Int.squared10(): Int = this * this
fun String.shout10(): String = this + "!"
val String.mid10: Char get() = this[length / 2]
fun String?.orDash10(): String = this ?: "-"
fun s10() {
    check("ext1", 7.squared10() == 49)
    check("ext2", "hey".shout10() == "hey!")
    check("ext3", "abc".mid10 == 'b')
    val gone: String? = null
    check("ext4", gone.orDash10() == "-" && "ok".orDash10() == "ok")
    fun Int.tripled10(): Int = this * 3
    check("ext5", 4.tripled10() == 12)
}

// ===== SECTION 11: classes: constructors, accessors, nesting =====
class Meter11(val start: Int) {
    var log = ""
    init { log += "i$start" }
    var level: Int = start
        get() = field + 100
        set(v) { field = if (v < 0) 0 else v }
    constructor(a: Int, b: Int) : this(a + b) { log += ":sec" }
    private fun hidden(): Int = 5
    internal fun reveal(): Int = hidden()
}
class Outer11(val tag: String) {
    val plain = "o"
    class Nested { fun label(): String = "nested" }
    inner class Inner { fun label(): String = tag + "-in" + this@Outer11.plain }
}
class Late11 { lateinit var name: String; fun ready(): Boolean = this::name.isInitialized }
fun s11() {
    val m = Meter11(4)
    check("cls1", m.log == "i4" && m.level == 104)
    m.level = -9
    check("cls2", m.level == 100)
    val m2 = Meter11(2, 3)
    check("cls3", m2.log == "i5:sec" && m2.start == 5)
    check("cls4", m.reveal() == 5)
    check("cls5", Outer11.Nested().label() == "nested")
    check("cls6", Outer11("z").Inner().label() == "z-ino")
    val l = Late11()
    check("cls7", !l.ready())
    l.name = "set"
    check("cls8", l.ready() && l.name.length == 3)
}

// ===== SECTION 12: inheritance and interfaces =====
interface Named12 { val kind: String; fun name(): String = "some " + kind }
abstract class Shape12(val id: Int) : Named12 {
    abstract fun area(): Int
    open fun describe(): String = "shape" + id
    protected fun secret(): Int = id + 10
}
open class Rect12(id: Int, val w: Int, val h: Int) : Shape12(id) {
    override val kind: String = "rect"
    override fun area(): Int = w * h
    final override fun describe(): String = "rect" + id + "/" + super.describe()
    fun leak(): Int = secret()
}
class Square12(id: Int, side: Int) : Rect12(id, side, side)
fun s12() {
    val r = Rect12(1, 3, 4)
    check("inh1", r.area() == 12)
    check("inh2", r.describe() == "rect1/shape1")
    check("inh3", r.name() == "some rect" && r.kind == "rect")
    check("inh4", r.leak() == 11)
    val s: Shape12 = Square12(2, 5)
    check("inh5", s.area() == 25 && s.describe() == "rect2/shape2")
    check("inh6", s is Named12 && s is Rect12)
}

// ===== SECTION 13: data and enum classes =====
data class Pt13(val x: Int, val y: Int)
enum class Dir13(val dx: Int, val dy: Int) {
    N(0, -1), E(1, 0), S(0, 1);
    fun flipped(): Dir13 = when (this) { N -> S; S -> N; E -> E }
}
fun s13() {
    val p = Pt13(1, 2)
    val q = p.copy(y = 9)
    check("dat1", q.x == 1 && q.y == 9 && p.y == 2)
    check("dat2", p == Pt13(1, 2) && p != q)
    check("dat3", p.toString() == "Pt13(x=1, y=2)")
    val (a, b) = q
    check("dat4", a == 1 && b == 9)
    check("dat5", p.component2() == 2)
    check("enm1", Dir13.E.dx == 1 && Dir13.N.dy == -1)
    check("enm2", Dir13.S.name == "S" && Dir13.E.ordinal == 1)
    check("enm3", Dir13.N.flipped() == Dir13.S && Dir13.E.flipped() == Dir13.E)
    val turn = when (Dir13.N) { Dir13.N, Dir13.S -> "vertical"; Dir13.E -> "horizontal" }
    check("enm4", turn == "vertical")
}

// ===== SECTION 14: sealed hierarchies =====
sealed interface Expr14
sealed class Node14 : Expr14
class Num14(val v: Int) : Node14()
class Add14(val l: Expr14, val r: Expr14) : Node14()
object Zero14 : Expr14
fun eval14(e: Expr14): Int = when (e) {
    is Num14 -> e.v
    is Add14 -> eval14(e.l) + eval14(e.r)
    Zero14 -> 0
}
fun s14() {
    check("sld1", eval14(Num14(7)) == 7)
    check("sld2", eval14(Add14(Num14(2), Add14(Num14(3), Zero14))) == 5)
    val n: Expr14 = Num14(1); val z: Expr14 = Zero14
    check("sld3", n is Node14 && z !is Node14)
    val tags = listOf<Expr14>(Num14(3), Zero14).map {
        when (it) { is Num14 -> "n"; is Add14 -> "a"; Zero14 -> "z" }
    }
    check("sld4", tags[0] == "n" && tags[1] == "z")
}

// ===== SECTION 15: objects, companions and delegation =====
interface Greeter15 {
    fun greet(): String
    fun bye(): String = "bye"
}
class EnGreeter15 : Greeter15 { override fun greet(): String = "hello" }
class Loud15(base: Greeter15) : Greeter15 by base { override fun bye(): String = "BYE" }
object Registry15 { var count = 0; fun add(n: Int): Int { count += n; return count } }
class Tagged15(val v: Int) {
    companion object Maker { val SEED = 7; fun of(n: Int): Tagged15 = Tagged15(n + SEED) }
}
fun s15() {
    check("obj1", Registry15.add(2) == 2 && Registry15.add(3) == 5)
    check("obj2", Tagged15.of(1).v == 8 && Tagged15.SEED == 7 && Tagged15.Maker.SEED == 7)
    val loud = Loud15(EnGreeter15())
    check("obj3", loud.greet() == "hello" && loud.bye() == "BYE")
    val anon = object { val x = 3; fun twice(): Int = x * 2 }
    check("obj4", anon.twice() == 6)
    val hi = object : Greeter15 { override fun greet(): String = "hi" + Registry15.count }
    check("obj5", hi.greet() == "hi5" && hi.bye() == "bye")
}

// ===== SECTION 16: generics =====
class Box16<T>(val item: T) { fun swap(other: T): T = other }
class Producer16<out T>(val value: T) { fun produce(): T = value }
class Consumer16<in T> { var seen = 0; fun consume(x: T) { seen += 1 } }
fun <T> pick16(a: T, b: T, first: Boolean): T = if (first) a else b
fun <T> larger16(a: T, b: T): T where T : Comparable<T> = if (a >= b) a else b
fun sizeOf16(b: Box16<*>): Int = if (b.item == null) 0 else 1
open class Animal16 { open fun cry(): String = "..." }
class Dog16 : Animal16() { override fun cry(): String = "woof" }
fun s16() {
    val bi = Box16(42); val bs = Box16<String>("hi")
    check("gen1", bi.item == 42 && bs.item.length == 2)
    check("gen2", bi.swap(7) == 7 && pick16("a", "b", false) == "b")
    check("gen3", larger16(3, 9) == 9 && larger16("b", "a") == "b")
    val prod: Producer16<Animal16> = Producer16<Dog16>(Dog16())
    check("gen4", prod.produce().cry() == "woof")
    val sink: Consumer16<Dog16> = Consumer16<Animal16>()
    sink.consume(Dog16())
    check("gen5", sink.seen == 1)
    check("gen6", sizeOf16(bi) == 1 && sizeOf16(bs) == 1)
}

// ===== SECTION 17: inline functions and reified =====
inline fun runTwice17(f: (Int) -> Int): Int = f(1) + f(2)
inline fun keep17(noinline f: () -> Int): () -> Int = f
inline fun defer17(crossinline f: () -> Int): () -> Int = { f() + 1 }
inline fun <reified T> tag17(x: Any): String = if (x is T) "yes" else "no"
fun s17() {
    check("inl1", runTwice17 { it * 10 } == 30)
    val kept = keep17 { 5 }
    check("inl2", kept() == 5)
    val d = defer17 { 40 }
    check("inl3", d() == 41)
    check("inl4", tag17<String>("abc") == "yes" && tag17<String>(9) == "no")
    check("inl5", tag17<Int>(7) == "yes")
}

// ===== SECTION 18: null safety and casts =====
fun stretch18(x: Any): Int {
    if (x !is String) return -1
    return x.length
}
fun s18() {
    var s: String? = null
    check("nul1", (s?.length ?: -1) == -1)
    s = if (checks > 0) "abcd" else null
    check("nul2", s?.length == 4 && s!!.length == 4)
    val any: Any = "hello"
    if (any is String) check("nul3", any.length == 5)
    check("nul4", stretch18("abc") == 3 && stretch18(42) == -1)
    val num: Any = 5
    val asStr: String? = num as? String
    check("nul5", asStr == null && (num as? Int ?: 0) == 5)
    val forced: String = any as String
    check("nul6", forced.length == 5)
}

// ===== SECTION 19: destructuring and typealias =====
typealias Mapper19 = (Int) -> Int
typealias Pts19 = List<Pt19>
data class Pt19(val x: Int, val y: Int)
class Split19(val s: String) {
    operator fun component1(): String = s + "1"
    operator fun component2(): Int = s.length
}
fun s19() {
    val (a, b) = Pt19(3, 4)
    check("des1", a == 3 && b == 4)
    val (_, second) = Pt19(7, 8)
    check("des2", second == 8)
    val (tag, len) = Split19("hi")
    check("des3", tag == "hi1" && len == 2)
    var sum = 0
    val pts: Pts19 = listOf(Pt19(1, 2), Pt19(3, 4))
    for ((px, py) in pts) { sum += px * py }
    check("des4", sum == 14)
    val twice: Mapper19 = { it * 2 }
    check("des5", twice(21) == 42)
}

// ===== SECTION 20: annotations =====
annotation class Note20(val label: String)
@Note20("fn") fun tagged20(n: Int): Int = n + 1
@Note20("cls") class Carrier20(@param:Note20("p") val v: Int) {
    @get:Note20("g") val doubled: Int get() = v * 2
    @field:Note20("f") var slot: Int = 1
}
fun s20() {
    check("ann1", tagged20(41) == 42)
    val c = Carrier20(5)
    check("ann2", c.v == 5 && c.doubled == 10)
    c.slot = 3
    check("ann3", c.slot == 3)
    @Note20("local") val here = 9
    check("ann4", here == 9)
}

// ===== SECTION 21: exceptions as expressions =====
class AppError21(msg: String) : Exception(msg)
fun bail21(msg: String): Nothing = throw AppError21(msg)
fun half21(n: Int): Int = if (n % 2 == 0) n / 2 else bail21("odd$n")
fun s21() {
    val r1 = try { half21(8) } catch (e: AppError21) { -1 }
    check("exc1", r1 == 4)
    var fin = ""
    val r2 = try { half21(3) } catch (e: AppError21) { e.message ?: "?" } finally { fin = "ran" }
    check("exc2", r2 == "odd3" && fin == "ran")
    var order = ""
    try {
        try { throw AppError21("in") } finally { order += "f1" }
    } catch (e: Exception) { order += "c" + (e.message ?: "") } finally { order += "f2" }
    check("exc3", order == "f1cinf2")
    fun readOrBail(v: Int?): Int = v ?: bail21("nil")
    val got = try { readOrBail(null) } catch (e: AppError21) { -9 }
    check("exc4", got == -9 && readOrBail(6) == 6)
    val which = try { throw AppError21("a") } catch (e: AppError21) { "app" } catch (e2: Exception) { "exc" }
    check("exc5", which == "app")
}

// ===== SECTION 22: suspend and value classes =====
@JvmInline value class Meters22(val v: Int) {
    operator fun plus(o: Meters22): Meters22 = Meters22(v + o.v)
    fun doubled(): Meters22 = Meters22(v * 2)
}
suspend fun tick22(n: Int): Int = n + 1
fun s22() {
    val m = Meters22(20) + Meters22(1)
    check("val1", m.v == 21 && m.doubled().v == 42)
    check("val2", Meters22(3) == Meters22(3) && Meters22(3) != Meters22(4))
    var op: (suspend (Int) -> Int)? = null
    if (checks > 0) { op = ::tick22 }
    check("sus1", op != null)
}

// ===== SECTION 23: regular expressions =====
// Regex and MatchResult over the shared engine (languages/lib/regex.js for the
// interpreter, abnf/jsrtregex.go + abnf/jsrtregexkt.go for the compiler). Kotlin
// has no regexp literal, so every pattern arrives as Regex("...") or
// "...".toRegex(). Regex.escape is Pattern.quote and answers the same \Q...\E TEXT
// the JVM does (verified against java.util.regex on this machine);
// RegexOption.CANON_EQ and UNIX_LINES are reported as unsupported
// instead of being silently ignored, and lookbehind, POSIX brackets and \p{...}
// are not implemented by the engine.
fun s23() {
    val word23 = Regex("[a-c]+")
    check("re01", word23.matches("abc") && !word23.matches("abcd"))
    check("re02", word23.containsMatchIn("xxabyy") && !word23.containsMatchIn("zzz"))
    check("re03", word23.pattern == "[a-c]+" && word23.toString() == "[a-c]+")
    val hit23 = word23.find("xxabyy")
    check("re04", hit23 != null && hit23.value == "ab")
    check("re05", hit23 != null && hit23.range.first == 2 && hit23.range.last == 3)
    check("re06", word23.find("zzz") == null)
    check("re07", Regex("b").find("abcb", 2)!!.range.first == 3)

    val pair23 = Regex("(\\w+)@(\\w+)")
    val m23 = pair23.find("mail bob@host end")!!
    check("re08", m23.value == "bob@host" && m23.groupValues[0] == "bob@host")
    check("re09", m23.groupValues[1] == "bob" && m23.groupValues[2] == "host")
    check("re10", m23.groups[1]!!.value == "bob" && m23.groups[2]!!.range.first == 9)
    check("re11", m23.groups.size == 3 && Regex("(a)|(b)").find("b")!!.groups[1] == null)
    val (user23, host23) = m23.destructured
    check("re12", user23 == "bob" && host23 == "host")
    val named23 = Regex("(?<who>[a-z]+)!").find("say hey! now")!!
    check("re13", named23.groups["who"]!!.value == "hey")

    // The replacement template is Kotlin's ($1 / ${name}, backslash escapes), not
    // the engine's \1 - the grammar rewrites it on both sides of the pair.
    check("re14", pair23.replace("a@b c@d", "\$2-\$1") == "b-a d-c")
    check("re15", pair23.replaceFirst("a@b c@d", "\$2-\$1") == "b-a c@d")
    check("re16", Regex("(?<d>[0-9])").replace("a1b2", "[\${d}]") == "a[1]b[2]")
    check("re17", Regex("x").replace("x", Regex.escapeReplacement("\$1")) == "\$1")
    check("re18", Regex("y").replace("y", "a\\\\b") == "a\\b")
    check("re19", Regex("[a-z]").replace("abc") { "<" + it.value + ">" } == "<a><b><c>")

    val parts23 = Regex("\\s+").split("a b  c")
    check("re20", parts23.size == 3 && parts23[0] == "a" && parts23[2] == "c")
    val lim23 = Regex(",").split("a,b,c", 2)
    check("re21", lim23.size == 2 && lim23[0] == "a" && lim23[1] == "b,c")

    check("re22", "a1b2".replace(Regex("[0-9]"), "#") == "a#b#")
    check("re23", "a1b2".contains(Regex("[0-9]")) && !"abc".contains(Regex("[0-9]")))
    check("re24", "a1b2".matches(Regex("[a-z0-9]+")) && !"a1b2".matches(Regex("[a-z]+")))
    val sp23 = "x1y22z".split(Regex("[0-9]+"))
    check("re25", sp23.size == 3 && sp23[1] == "y" && sp23[2] == "z")
    check("re26", "[0-9]".toRegex().containsMatchIn("a7"))
    check("re27", "he.lo".toRegex(RegexOption.LITERAL).matches("he.lo"))

    // RegexOption maps onto the engine's neutral i / s / m / x. Kotlin (like Java,
    // unlike Ruby) anchors ^ and $ to the whole input unless MULTILINE is given.
    check("re28", Regex("A", RegexOption.IGNORE_CASE).containsMatchIn("xax"))
    check("re29", Regex("a.b", setOf(RegexOption.IGNORE_CASE, RegexOption.DOT_MATCHES_ALL)).matches("A\nB"))
    check("re30", Regex("^b\$", RegexOption.MULTILINE).containsMatchIn("a\nb\nc"))
    check("re31", !Regex("^b\$").containsMatchIn("a\nb\nc"))
    check("re32", Regex("a b # note", RegexOption.COMMENTS).matches("ab"))
    val lit23 = Regex("a.b", RegexOption.LITERAL)
    check("re33", lit23.matches("a.b") && !lit23.matches("axb"))
    check("re34", Regex(Regex.escape("1+1")).matches("1+1"))

    check("re35", Regex("b").findAll("abcb").count() == 2)
    var next23 = Regex("b").find("abcb")
    var seen23 = 0
    while (next23 != null) { seen23 += 1; next23 = next23.next() }
    check("re36", seen23 == 2)
    val whole23 = Regex("[0-9]+").matchEntire("123")
    check("re37", whole23 != null && whole23.value == "123" && whole23.range.last == 2)
    check("re38", Regex("[0-9]+").matchEntire("12a") == null)

    check("re39", Regex("(ab)\\1").containsMatchIn("xabab"))
    check("re40", Regex("a+?b").find("aaab")!!.value == "aaab")
    check("re41", Regex("colou?r").matches("color") && Regex("x{2,3}").find("xxxx")!!.value == "xxx")
    check("re42", Regex("(?=a)[a-z]").find("za")!!.range.first == 1)
    // Regex.escape IS Pattern.quote: the \Q...\E text, not merely an equivalent
    // pattern. Checked against java.util.regex.Pattern.quote.
    check("re43", Regex.escape("a.b") == "\\Qa.b\\E" && Regex.escape("a+b*c") == "\\Qa+b*c\\E")
    check("re44", Regex.escape("a").length == 5)
    // \Q...\E quote regions, written out by hand: every character between them is a
    // literal, and a quantifier after \E binds to the LAST quoted character.
    check("re45", Regex("\\Qa.b\\E").matches("a.b") && !Regex("\\Qa.b\\E").matches("axb"))
    check("re46", Regex("x\\Qa+b\\Ey").containsMatchIn("xa+by") && Regex("\\Qab\\E+").matches("abb"))
    check("re47", Regex("a.b", RegexOption.LITERAL).matches("a.b")
                  && !Regex("a.b", RegexOption.LITERAL).matches("axb"))
    // Named backreferences inside a pattern.
    check("re48", Regex("(?<n>a)\\k<n>").matches("aa") && !Regex("(?<n>a)\\k<n>").matches("ab"))
}

// ===== SECTION 24: floating point arithmetic =====
// Double/Float are a DIFFERENT type from Int, and every value below was checked
// against the equivalent Java program run on `java` (JDK 24) - Kotlin/JVM's
// Double is java.lang.Double, its arithmetic is the JVM's and its toString is
// java.lang.Double.toString, so Java is the oracle here (no kotlinc available).
// The section exists because both grammars used to evaluate every arithmetic
// operator in 32 bit integers: 1.0 / 3.0 was 0, 2.5 * 1.5 was 3, println(1.0)
// printed "1" and Infinity / NaN / -0.0 did not exist at all.
class Money24(val amount: Double) {
    fun half(): Double = amount / 2
}
fun s24(d: Double): String = "" + d
fun sec24() {
    // Division is real division as soon as one side is a Double, and integer
    // division (truncating towards zero) when both sides are Int.
    check("flt1", 1.0 / 3.0 == 0.3333333333333333)
    check("flt2", 7 / 2 == 3 && -7 / 2 == -3)
    check("flt3", 7 / 2.0 == 3.5 && 7.0 / 2 == 3.5)
    check("flt4", 2.5 * 1.5 == 3.75 && 7.0 - 0.5 == 6.5 && 3.0 * 2.0 == 6.0)
    check("flt5", 2.0 % 0.75 == 0.5 && 1.5 + 1 == 2.5)
    // A Double operand promotes the whole operation, so an integral result still
    // prints with its decimal point.
    check("flt6", s24(1.0) == "1.0" && s24(6.0) == "6.0")
    check("flt7", s24(0.1 + 0.2) == "0.30000000000000004")
    check("flt8", s24(1.0 / 3.0) == "0.3333333333333333")
    // Infinity, NaN and the two zeroes.
    val inf = 1.0 / 0.0
    val nan = 0.0 / 0.0
    check("flt9", s24(inf) == "Infinity" && s24(-1.0 / 0.0) == "-Infinity")
    check("flt10", s24(nan) == "NaN" && nan != nan)
    check("flt11", s24(-0.0) == "-0.0" && 0.0 == -0.0)
    check("flt12", 1e300 * 1e300 == inf)
    // Double.toString switches to scientific notation outside [1e-3, 1e7).
    check("flt13", s24(1e20) == "1.0E20" && s24(1.5e-8) == "1.5E-8")
    check("flt14", s24(1e-3) == "0.001" && s24(2.5) == "2.5")
    // ++ and the compound operators keep a Double a Double.
    var d = 1.0
    check("flt15", s24(d) == "1.0")
    d += 0.25
    check("flt16", d == 1.25)
    d++
    check("flt17", d == 2.25 && s24(d) == "2.25")
    d *= 2
    check("flt18", d == 4.5)
    d /= 3
    check("flt19", d == 1.5)
    d--
    check("flt20", s24(d) == "0.5")
    // A string template renders a Double like toString does.
    check("flt21", "$d/${1.0}" == "0.5/1.0")
    check("flt22", s24(-2.5) == "-2.5")
    // Comparisons mix Double and Int freely.
    check("flt23", 2.5 > 2 && 2 < 2.5 && 1.0 == 1.0 && 3.0 <= 3 && 3.0 >= 3)
    // A Double survives a property, a list element and a function boundary.
    val m = Money24(5.0)
    check("flt24", m.half() == 2.5 && s24(m.amount) == "5.0")
    val xs = listOf(0.5, 1.5)
    check("flt25", xs[0] + xs[1] == 2.0 && s24(xs[0] * 2) == "1.0")
}

// ===== SECTION 25: value rendering =====
// What println / toString() / a string template make of a value. Every answer
// below was checked against the equivalent Java program run on `java` (JDK 24):
// Kotlin/JVM's String, Int and Double ARE java.lang.*, println IS
// System.out.println, a List renders through java.util.AbstractCollection.toString
// and an enum entry through java.lang.Enum.toString, so Java is the oracle here
// (no kotlinc available). The section exists because both grammars leaked their
// HOST's rendering instead: the interpreter printed a list through JavaScript's
// Array.prototype.toString ("1,2") and the compiler handed the raw runtime value
// to Go's %v ("<nil>" for null, "[1 2]" for a list, a map of pointer addresses for
// an instance - and an infinite recursion for an enum entry).
// Not asserted here, because this value model cannot reproduce it: an Array
// renders like a List, where real Kotlin prints "[Ljava.lang.Integer;@1b6d3586"
// (arrayOf and listOf build the same value here), and a class without a toString
// gets a FIXED synthetic hash, so only the "Name@" prefix is checked below.
// The two data classes below deliberately name their properties x and y, like
// every other data class in this file: the compiler grammar registers the
// named-argument slots of the generated copy() under the METHOD name alone, so a
// data class whose property names differ silently breaks .copy(y = 9) in an
// earlier section. That limitation is orthogonal to rendering and pre-existing.
data class Pair25(val x: Int, val y: String)
data class Holder25(val x: List<Int>, val y: Int)
class Plain25(val x: Int)
class Custom25 { override fun toString(): String = "custom!" }
enum class Suit25 { HEART, SPADE }
fun sec25() {
    // A null reference is "null" everywhere it is rendered.
    val s: String? = null
    check("ren1", "$s" == "null" && "" + s == "null")
    // A list: square brackets, ", " between the elements, no quotes on a String.
    val xs = listOf(1, 2)
    check("ren2", xs.toString() == "[1, 2]" && "$xs" == "[1, 2]")
    check("ren3", "" + xs == "[1, 2]")
    check("ren4", listOf("x", "y").toString() == "[x, y]")
    check("ren5", listOf<Int>().toString() == "[]")
    check("ren6", listOf(1, null).toString() == "[1, null]")
    check("ren7", listOf(listOf(1), listOf(2, 3)).toString() == "[[1], [2, 3]]")
    check("ren8", listOf('a', 'b').toString() == "[a, b]")
    check("ren9", listOf(1.5, 2.0).toString() == "[1.5, 2.0]")
    // A data class renders through its SYNTHESIZED toString, and its properties
    // render the same way a top-level value does.
    val p = Pair25(1, "q")
    check("ren10", p.toString() == "Pair25(x=1, y=q)" && "$p" == "Pair25(x=1, y=q)")
    check("ren11", Holder25(listOf(1, 2), 3).toString() == "Holder25(x=[1, 2], y=3)")
    check("ren12", listOf(Pair25(1, "a")).toString() == "[Pair25(x=1, y=a)]")
    // An explicit toString() wins over every built-in rendering.
    check("ren13", Custom25().toString() == "custom!" && "${Custom25()}" == "custom!")
    // A class without one renders as Name@<hash>; the hash itself is the JVM's
    // identity hash, which differs between runs, so only its shape is asserted.
    val plain = Plain25(5)
    check("ren14", plain.toString().indexOf("Plain25@") == 0 && plain.toString().indexOf("@") == 7)
    check("ren15", "$plain" == plain.toString())
    // An enum entry renders as its name.
    check("ren16", Suit25.HEART.toString() == "HEART" && "${Suit25.SPADE}" == "SPADE")
    check("ren17", "" + Suit25.HEART == "HEART" && Suit25.HEART.name == "HEART")
    check("ren18", listOf(Suit25.HEART, Suit25.SPADE).toString() == "[HEART, SPADE]")
    // The primitives keep the rendering they already had.
    check("ren19", true.toString() == "true" && 42.toString() == "42" && 1.5.toString() == "1.5")
    check("ren20", 'c'.toString() == "c" && "s".toString() == "s")
}

// ===== SECTION 26: the String member surface =====
// The members java.lang.String / kotlin.text define on a String. The section
// exists because the two halves carried DIFFERENT tables: kotlin-interpreter.abnf
// answered nine names its own mcall implements, while the compiler dispatched to
// the shared js_mcall, whose String switch is the JAVA one
// (length/charAt/equals/substring/indexOf/isEmpty) - so "aBc".uppercase() printed
// ABC in the interpreter and aborted the compiler with
// `unknown String method: uppercase`. Every check below therefore runs in BOTH
// halves; abnf/jsrtkotlin.go's js_ktsmcall is the Go twin of that mcall branch.
// No kotlinc on this machine: the expected values are Kotlin's documented
// semantics (kotlin.text.uppercase/lowercase are locale-invariant, hashCode is
// java.lang.String.hashCode = the 31*h + char fold, compareTo is
// lexicographic by code unit).
fun sec26() {
    val s = "aBc"
    // Kotlin 1.5 renamed toUpperCase/toLowerCase to uppercase/lowercase. Both
    // spellings resolve (the old pair is deprecated, not removed), and BOTH halves
    // must accept all four - the divergence this section pins.
    check("str1", s.uppercase() == "ABC")
    check("str2", s.lowercase() == "abc")
    check("str3", s.toUpperCase() == "ABC")
    check("str4", s.toLowerCase() == "abc")
    check("str5", "".uppercase() == "" && "9!".lowercase() == "9!")
    check("str6", s.uppercase().lowercase() == "abc")
    // The rest of the surface the compiler half was missing.
    check("str7", s.repeat(2) == "aBcaBc" && s.repeat(0) == "")
    check("str8", s.contains("B") && !s.contains("z") && s.contains(""))
    check("str9", s.plus("!") == "aBc!")
    check("str10", s.compareTo("aBd") < 0 && s.compareTo("aBc") == 0 && s.compareTo("aBa") > 0)
    check("str11", s.isNotEmpty() && !"".isNotEmpty())
    check("str12", s.get(1) == 'B' && s.get(0) == 'a')
    check("str13", s.hashCode() == 95362 && "".hashCode() == 0)
    check("str14", "abc".hashCode() == "abc".hashCode() && "abc".hashCode() != "abd".hashCode())
    // The names the shared table already carried keep working.
    check("str15", s.length == 3 && s.isEmpty() == false && s.equals("aBc"))
    check("str16", s.substring(1) == "Bc" && s.indexOf("B") == 1)
    // charAt is the JAVA spelling. Real Kotlin has no String.charAt at all - it is
    // get / the indexing operator - but both grammars carry it as a builtin, and it
    // used to answer a one-character STRING while its two Kotlin spellings answered a
    // Char, so `s.charAt(0) == 'a'` was FALSE. All three now answer a Char.
    check("str18", s.charAt(0) == 'a' && s[0] == 'a' && s.get(0) == 'a')
    check("str17", s.toString() == "aBc")
}

// ===== SECTION 27: nested type declarations =====
// A nested enum / interface / object / annotation class, in each of the four
// things that can hold one. Genuine in both halves: the declaration lowers like a
// top-level one and the enclosing descriptor gets it under its simple name, so
// Outer.Kind.LEFT resolves. The section exists because the compiler half used to
// report `nested type declaration not implemented (ignored)` for everything except
// a plain nested class, which the interpreter has always built.
// Every nested type below has a DISTINCT simple name, which keeps this section
// about nesting itself; SECTION 29 is the one that makes the simple names COLLIDE
// across owners. (Both work now. Until the compiler gave each nested declaration
// its own scope, its flat class model bound every nested type into one shared
// program scope under its simple name, so `class A { class Inner }` and
// `class B { class Inner }` shared a slot and A.Inner().who() answered "B".)
class Holder27(val tag: Int) {
    enum class Kind27 { LEFT, RIGHT }
    interface Shape27 { fun area(): Int }
    object Reg27 { val seed: Int = 5 }
    annotation class Mark27
    class Slot27(val n: Int)
    fun doubled(): Int = tag * 2
}
object Outer27 {
    enum class Mode27 { UP }
    class Cell27(val v: Int)
    fun go(): Int = 1
}
enum class Top27 { A, B;
    class Helper27(val n: Int)
}
interface Iface27 {
    object Registry27 { val seed: Int = 7 }
}
fun sec27() {
    // The enclosing class's own members are unaffected by the nested declarations.
    check("nst1", Holder27(21).doubled() == 42)
    // A nested enum: the entries, their names and their ordinals.
    check("nst2", Holder27.Kind27.LEFT.name == "LEFT" && Holder27.Kind27.RIGHT.name == "RIGHT")
    check("nst3", Holder27.Kind27.LEFT.ordinal == 0 && Holder27.Kind27.RIGHT.ordinal == 1)
    check("nst4", Holder27.Kind27.LEFT.toString() == "LEFT")
    // A nested object is a singleton reached through the enclosing name.
    check("nst5", Holder27.Reg27.seed == 5)
    // A nested class is CONSTRUCTED through the enclosing name.
    check("nst6", Holder27.Slot27(3).n == 3)
    // The same four holders: an object, an enum class and an interface.
    check("nst7", Outer27.go() == 1 && Outer27.Mode27.UP.name == "UP")
    check("nst8", Outer27.Cell27(4).v == 4)
    check("nst9", Top27.A.name == "A" && Top27.Helper27(5).n == 5)
    check("nst10", Iface27.Registry27.seed == 7)
}

// ===== SECTION 28: assignment to a computed target =====
// An assignment whose left side is a postfix chain containing a CALL, so a plain
// variable/field/index target cannot cover it: boxOf(9).tag = 100, boxes()[0] = 9.
// Genuine in both halves - the chain up to the last step is evaluated ONCE and the
// last `.name` / `[i]` step is written like any other slot. The section exists
// because the compiler half used to report `assignment to a computed target not
// implemented (ignored)` and silently write nothing, while the interpreter's
// makeCallAssign performed the write.
class Box28(var tag: Int)
val shared28 = Box28(1)
val store28 = mutableListOf(1, 2, 3)
var order28 = ""
fun boxOf28(k: Int): Box28 = Box28(k)
fun sharedOf28(): Box28 = shared28
fun boxes28(): MutableList<Int> = store28
fun lhs28(): Box28 { order28 = order28 + "L"; return shared28 }
fun rhs28(): Int { order28 = order28 + "R"; return 7 }
fun idx28(): Int { order28 = order28 + "I"; return 1 }
fun sec28() {
    // A fresh object each call: the write happens, and is then unobservable.
    boxOf28(9).tag = 100
    check("cta1", boxOf28(9).tag == 9)
    // The same object: the write IS observable.
    sharedOf28().tag = 42
    check("cta2", shared28.tag == 42)
    // A compound operator reads and writes the same slot.
    sharedOf28().tag += 8
    check("cta3", shared28.tag == 50)
    // An indexed slot behind a call.
    boxes28()[0] = 9
    check("cta4", store28[0] == 9)
    boxes28()[2] -= 1
    check("cta5", store28[2] == 2)
    // Kotlin evaluates the receiver before the right-hand side, and the index of an
    // indexed slot after both. Pinned so the two halves cannot drift on the order.
    order28 = ""
    lhs28().tag = rhs28()
    check("cta6", order28 == "LR" && shared28.tag == 7)
    order28 = ""
    boxes28()[idx28()] = rhs28()
    check("cta7", order28 == "RI" && store28[1] == 7)
}

// ===== SECTION 29: nested types with COLLIDING simple names =====
// Two owners each declaring a nested type of the SAME simple name. The compiler's
// class model is flat - one program scope - so binding a nested declaration under
// its simple name made the second owner's declaration win for both: A29.Inner29(1)
// constructed B29's Inner29 and answered "B". Each nested declaration now runs in
// its own child scope and the enclosing descriptor closes over THAT scope, so the
// factory reaches the descriptor declared beside it whatever anyone else declares
// under the same name. The simple name is still mirrored into the enclosing scope,
// which is what keeps a bare `Inner29(...)` inside the owner's own body working.
// KNOWN RESIDUE, identical in BOTH halves (so not a divergence, and deliberately
// not asserted here): an UNQUALIFIED reference to a nested class from inside its
// owner's own body still goes through that one mirrored program-scope binding, so
// with two owners it finds whichever was declared last. Kotlin resolves the owner's
// own nested type first. Qualified use - `A29.Inner29(1)`, which is the form that
// diverged - is what this section pins.
class A29 {
    class Inner29(val v: Int) { fun who(): String = "A" }
    enum class Kind29 { LEFT }
    object Reg29 { val seed: Int = 1 }
}
class B29 {
    class Inner29(val v: Int) { fun who(): String = "B" }
    enum class Kind29 { RIGHT }
    object Reg29 { val seed: Int = 2 }
}
object C29 {
    class Inner29(val v: Int) { fun who(): String = "C" }
}
class Host29(val tag: Int) {
    inner class Inner29(val v: Int) { fun who(): String = "H" + tag + v }
}
fun sec29() {
    // Constructed through the enclosing name: each owner gets its OWN nested class.
    check("col1", A29.Inner29(1).who() == "A" && B29.Inner29(2).who() == "B")
    check("col2", A29.Inner29(1).v == 1 && B29.Inner29(2).v == 2)
    check("col3", C29.Inner29(3).who() == "C")
    // Repeated construction through either owner keeps answering that owner.
    check("col4", A29.Inner29(6).who() == "A" && A29.Inner29(7).who() == "A")
    // The same for the value-shaped nested types: an enum and an object.
    check("col5", A29.Kind29.LEFT.name == "LEFT" && B29.Kind29.RIGHT.name == "RIGHT")
    check("col6", A29.Reg29.seed == 1 && B29.Reg29.seed == 2)
    // An INNER class of the same simple name keeps its stored outer receiver.
    check("col7", Host29(7).Inner29(8).who() == "H78")
    check("col8", Host29(9).Inner29(0).who() == "H90" && A29.Inner29(1).who() == "A")
}

// ===== SECTION 30: a raw string ends at the LAST triple quote =====
// A run of more than three quotes closing a raw string belongs to the CONTENT up to the
// final three - Kotlin's own psi/stringTemplates/RawStringsWithManyQuotes.kt writes
// runs of six, seven and eight quotes with a clean parse tree. Both halves used to stop
// at the FIRST triple quote and then fail on the leftover quote, so a raw string could
// not end with a quote character at all.
fun sec30() {
    val one = """""""
    val two = """"""""
    check("raw1", one.length == 1 && one == "\"")
    check("raw2", two.length == 2 && two == "\"\"")
    // The empty raw string and a quote in the MIDDLE are unaffected.
    check("raw3", """""".length == 0)
    check("raw4", """a""b""" == "a\"\"b")
    // A raw string whose content ends with one or two quotes.
    check("raw5", """ok"""" == "ok\"")
    check("raw6", """ok""""" == "ok\"\"")
}

// ===== SECTION 31: Int and Long have real WIDTHS =====
// Kotlin's integral types are fixed-width and their arithmetic WRAPS silently on
// overflow (unlike Swift, which traps). Both grammars used to model every integer
// as one JS double, so Int.MAX_VALUE + 1 grew instead of wrapping, 1L shl 40 was
// 256 (a 32 bit shift), Long.MAX_VALUE could not be written down at all and
// `Int` was an unknown name.
//
// Kotlin/JVM's Int and Long ARE java.lang's, so every value below was checked
// against the EQUIVALENT JAVA PROGRAM run on `java` (JDK 24) - there is no
// kotlinc on this machine. The Java probe and both halves of this grammar pair
// print the same 60 lines under goja and under -frozen.
fun s31(v: Any?): String = "" + v
fun zero31(): Int = 0
fun sec31() {
    // Overflow wraps at the type's width, in both directions.
    check("int1", Int.MAX_VALUE + 1 == Int.MIN_VALUE)
    check("int2", Int.MIN_VALUE - 1 == Int.MAX_VALUE)
    check("int3", Int.MAX_VALUE * 2 == -2)
    check("int4", Long.MAX_VALUE + 1L == Long.MIN_VALUE)
    // A Long is exact to 64 bits, well past what a double holds. 9007199254740993
    // is 2^53 + 1: a double rounds it DOWN to 2^53 and cannot represent it.
    check("int5", s31(Long.MAX_VALUE) == "9223372036854775807")
    check("int6", s31(Long.MIN_VALUE) == "-9223372036854775808")
    check("int7", s31(9223372036854775807L - 1L) == "9223372036854775806")
    check("int8", s31(9007199254740993L + 0L) == "9007199254740993")
    check("int9", 1000000L * 1000000L == 1000000000000L)
    // The shift COUNT is masked - & 31 for an Int, & 63 for a Long - so a count at
    // or above the width shifts by count mod width rather than clearing the value.
    check("int10", 1 shl 32 == 1 && 1L shl 64 == 1L)
    check("int11", 1 shl 31 == Int.MIN_VALUE && 1L shl 63 == Long.MIN_VALUE)
    check("int12", s31(1L shl 40) == "1099511627776")
    // shr keeps the sign, ushr does not; both stay at the left operand's width.
    check("int13", -16 shr 2 == -4 && -16L shr 2 == -4L)
    check("int14", -1 ushr 1 == Int.MAX_VALUE)
    check("int15", -1L ushr 1 == Long.MAX_VALUE)
    // and / or / xor / inv are infix FUNCTIONS in Kotlin, not operators.
    check("int16", 0xFF and 0x0F == 15 && 0xF0 or 0x0F == 255)
    check("int17", 0xFF xor 0x0F == 240)
    check("int18", 5.inv() == -6 && 0L.inv() == -1L)
    check("int19", s31((1L shl 40).inv()) == "-1099511627777")
    // Division truncates towards zero and MIN / -1 wraps back to MIN.
    check("int20", 7 / 2 == 3 && -7 / 2 == -3 && 7 / -2 == -3)
    check("int21", -7 % 2 == -1 && 7 % -2 == 1)
    check("int22", Int.MIN_VALUE / -1 == Int.MIN_VALUE && Int.MIN_VALUE % -1 == 0)
    check("int23", Long.MIN_VALUE / -1L == Long.MIN_VALUE)
    // Division and remainder by zero THROW a catchable ArithmeticException instead
    // of quietly yielding 0, and the message is java.lang's.
    var caught = ""
    try { s31(1 / zero31()) } catch (e: ArithmeticException) { caught = "" + e.message }
    check("int24", caught == "/ by zero")
    caught = ""
    try { s31(1 % zero31()) } catch (e: ArithmeticException) { caught = "" + e.message }
    check("int25", caught == "/ by zero")
    // A DECLARED type puts its width on the value, so the multiplication below is
    // 64 bit throughout instead of wrapping at 32.
    val big: Long = 1
    check("int26", s31(big * 1000000 * 1000000) == "1000000000000")
    // ++ steps at the value's own width, which is what makes a Byte wrap at 127.
    var b: Byte = 127
    b++
    check("int27", s31(b) == "-128")
    var l: Long = 9223372036854775806L
    l++
    check("int28", l == Long.MAX_VALUE)
    l++
    check("int29", l == Long.MIN_VALUE)
    // Byte and Short arithmetic PROMOTES to Int, so a sum of two bytes does not
    // wrap at 8 bits, and a Byte compares against an Int at Int width.
    val b1: Byte = 100
    val b2: Byte = 100
    check("int30", b1 + b2 == 200)
    check("int31", b1 < 1000 && b1 > -1000)
    check("int32", s31(Byte.MAX_VALUE) == "127" && s31(Byte.MIN_VALUE) == "-128")
    check("int33", s31(Short.MAX_VALUE) == "32767" && s31(Short.MIN_VALUE) == "-32768")
    // A Long compares EXACTLY, past the range where a double would round.
    check("int34", Long.MAX_VALUE > 0L && Long.MIN_VALUE < 0L)
    check("int35", 9007199254740993L > 9007199254740992L)
    check("int36", 1L == 1L && !(1L == 2L))
    // A sized value survives a list, a template and a function boundary.
    val xs = listOf(1L, 2L)
    check("int37", s31(xs) == "[1, 2]" && xs[0] + xs[1] == 3L)
    check("int38", "v=${Long.MAX_VALUE}" == "v=9223372036854775807")
}

// ===== SECTION 32: the CONVERSION methods =====
// Kotlin has NO implicit numeric conversions and no cast syntax for them either,
// so every width change is a METHOD: toByte(), toShort(), toInt(), toLong(),
// toDouble(). An integral source truncates its bits; a Double source SATURATES at
// the target's range and answers 0 for NaN, because Kotlin/JVM compiles
// Double.toInt() to the JVM's d2i. Java's `(byte)` / `(int)` casts are the same
// two instructions, so every value below was checked against the equivalent Java
// program run on `java` (JDK 24).
fun s32(v: Any?): String = "" + v
fun sec32() {
    // An integral source is a pure bit truncation.
    check("cnv1", s32(255.toByte()) == "-1" && s32(200.toByte()) == "-56")
    check("cnv2", s32(70000.toShort()) == "4464")
    check("cnv3", s32(3000000000L.toInt()) == "-1294967296")
    check("cnv4", s32(Long.MAX_VALUE.toInt()) == "-1")
    check("cnv5", s32(5.toLong()) == "5" && s32((-1).toLong()) == "-1")
    // A Double source truncates towards zero and saturates at the range ends.
    check("cnv6", 3.9.toInt() == 3 && (-3.9).toInt() == -3)
    check("cnv7", 1e20.toInt() == Int.MAX_VALUE && (-1e20).toInt() == Int.MIN_VALUE)
    check("cnv8", 1e20.toLong() == Long.MAX_VALUE)
    check("cnv9", Double.NaN.toInt() == 0)
    // toDouble() is a genuine conversion, which is what makes 7.toDouble() / 2
    // divide as floating point where 7 / 2 does not.
    check("cnv10", 7.toDouble() / 2 == 3.5 && s32(7.toDouble()) == "7.0")
    check("cnv11", s32(Long.MAX_VALUE.toDouble()) == "9.223372036854776E18")
    // toChar() takes the low 16 bits.
    check("cnv12", 65.toChar() == 'A' && 65601.toChar() == 'A')
    // The companion constants exist at all - `Int` was an unknown name in both
    // halves before, and Long.MAX_VALUE cannot be written as a plain number.
    check("cnv13", Int.SIZE_BITS == 32 && Int.SIZE_BYTES == 4)
    check("cnv14", Long.SIZE_BITS == 64 && Long.SIZE_BYTES == 8)
    check("cnv15", s32(Double.POSITIVE_INFINITY) == "Infinity")
    check("cnv16", s32(Double.NEGATIVE_INFINITY) == "-Infinity")
    check("cnv17", s32(Double.NaN) == "NaN")
    // Mixed Int/Double arithmetic still promotes to Double: the sized-integer box
    // and the floating-point box of the previous section compose.
    check("cnv18", 2.0 + 1 == 3.0 && 1 + 2.0 == 3.0 && 7 / 2.0 == 3.5)
    check("cnv19", s32(1 + 2.0) == "3.0" && s32(1L + 2.0) == "3.0")
    check("cnv20", 1L < 2.0 && 2.0 > 1L)
    // A declared type retypes its INITIALIZER, which is the one place Kotlin does
    // convert an integer literal without a method: `val d: Double = 1` holds 1.0.
    val dd: Double = 1
    val ll: Long = 1
    check("cnv21", s32(dd) == "1.0" && dd / 2 == 0.5)
    check("cnv22", s32(ll) == "1" && ll is Long)
}

// ===== SECTION 33: the UNSIGNED types, and how a LITERAL is typed =====
// UByte / UShort / UInt / ULong are the one place Kotlin's integers are not
// Java's, so java cannot be the oracle here and every rule below is cited to the
// Kotlin documentation and is UNVERIFIED by any toolchain (there is no kotlinc on
// this machine).
//
//   - kotlin.UInt / ULong / UByte / UShort are unsigned 32 / 64 / 8 / 16 bit
//     types whose arithmetic wraps, whose division and comparison are unsigned,
//     and whose toString prints the UNSIGNED reading.
//   - the literal suffixes are u / U (unsigned) and uL / UL (unsigned long);
//     a `u` literal is a UInt when it fits in one and a ULong otherwise.
//   - an UNSUFFIXED literal is an Int when it fits in one and a Long otherwise -
//     which is why 0xFFFFFFFF is the Long 4294967295 in Kotlin where the same
//     text is the int -1 in Java.
//   - there are no implicit conversions between signed and unsigned, so a mixed
//     expression is a compile error rather than a promotion; the value model
//     cannot enforce that and simply takes the unsigned reading.
fun s33(v: Any?): String = "" + v
fun sec33() {
    // An unsigned type prints and wraps as unsigned.
    check("uns1", s33(1u) == "1" && s33(UInt.MAX_VALUE) == "4294967295")
    check("uns2", UInt.MAX_VALUE + 1u == 0u && s33(0u - 1u) == "4294967295")
    check("uns3", s33(ULong.MAX_VALUE) == "18446744073709551615")
    check("uns4", s33(ULong.MAX_VALUE / 2uL) == "9223372036854775807")
    check("uns5", s33(5uL - 7uL) == "18446744073709551614")
    check("uns6", s33(UByte.MAX_VALUE) == "255" && s33(UShort.MAX_VALUE) == "65535")
    check("uns7", s33(UInt.MIN_VALUE) == "0" && s33(ULong.MIN_VALUE) == "0")
    // Division and comparison are UNSIGNED: a value above 2^31 is large, not
    // negative, which is exactly what a signed reading would get wrong.
    check("uns8", s33(4294967295u / 2u) == "2147483647")
    check("uns9", 3000000000u > 1u && 3000000000u > 2147483647u)
    check("uns10", ULong.MAX_VALUE > 1uL)
    // A shift keeps the left operand's type and masks its count.
    check("uns11", s33(1uL shl 63) == "9223372036854775808")
    check("uns12", s33(1u shl 31) == "2147483648")
    // inv() answers the operand's OWN type, not Int.
    check("uns13", s33(1u.inv()) == "4294967294")
    // The conversions reinterpret the bits in both directions.
    check("uns14", s33((-1).toUInt()) == "4294967295")
    check("uns15", s33((-1L).toULong()) == "18446744073709551615")
    check("uns16", s33(256u.toUByte()) == "0" && s33(300.toUByte()) == "44")
    check("uns17", s33(4294967295u.toInt()) == "-1")
    // A Double conversion to an unsigned type saturates at [0, MAX].
    check("uns18", s33(1e20.toUInt()) == "4294967295" && s33((-5.0).toUInt()) == "0")
    // ++ wraps at the unsigned width too.
    var u: UInt = 1u
    u++
    check("uns19", s33(u) == "2")
    var m: UByte = 255u
    m++
    check("uns20", s33(m) == "0")
    // An unsuffixed literal is an Int when it fits and a Long otherwise, so this
    // hexadecimal is 4294967295 and not -1 (the Java answer for the same text).
    check("uns21", 0xFFFFFFFF == 4294967295L)
    check("uns22", s33(2147483648) == "2147483648")
    check("uns23", 0b1010 == 10 && 1_000_000 == 1000000)
    check("uns24", 0xFFFFFFFFL == 4294967295L)
    // A value carries its real type into `is`, which the untyped value model could
    // not answer before: every integer was an Int AND a Long at once.
    val a: Any = 10L
    val i: Any = 10
    val q: Any = 1u
    check("uns25", a is Long && !(a is Int))
    check("uns26", i is Int && !(i is Long))
    check("uns27", q is UInt && !(q is Int))
    // An unsigned value survives a list and a string template.
    check("uns28", s33(listOf(1uL, 2uL)) == "[1, 2]")
    check("uns29", "u=${UInt.MAX_VALUE}" == "u=4294967295")
}


// ===== SECTION 34: return / break / continue as EXPRESSIONS, and PREFIX ++ / -- =====
// `x ?: return` is the idiom this section exists for. Kotlin types return, break and
// continue as Nothing, so each is a legal EXPRESSION that never yields a value and
// instead transfers control out of the whole surrounding expression.
// Ground truth for the SYNTAX is Kotlin's own parse tree: the corpus file
// tests/reference/kotlin/.../psi/greatSyntacticShift/nullableTypes.txt records
// `x as? X ?: return` as BINARY_EXPRESSION(BINARY_WITH_TYPE, ELVIS, RETURN) with no
// PsiErrorElement, and psi/SimpleExpressions.txt puts `= break@la` / `= continue@la` in a
// parameter default, also clean. The VALUES asserted below are the language rule (the
// prefix form yields the NEW value, the postfix form the old one) and are UNVERIFIED by
// any toolchain - there is no kotlinc on this machine.
fun firstOr34(v: Int?): Int {
    val x = v ?: return -1
    return x + 100
}
fun sumUntilNull34(limit: Int): Int {
    var s = 0
    for (i in 1..10) {
        val x = if (i > limit) null else i
        s = s + (x ?: break)
    }
    return s
}
fun sumOdd34(): Int {
    var s = 0
    for (i in 1..6) {
        val x = if (i % 2 == 0) null else i
        s = s + (x ?: continue)
    }
    return s
}
fun labelledBreak34(): Int {
    var s = 0
    outer@ for (i in 1..5) {
        for (j in 1..5) {
            val x = if (i > 2) null else j
            s = s + (x ?: break@outer)
        }
    }
    return s
}
fun labelledContinue34(): Int {
    var s = 0
    outer@ for (i in 1..5) {
        for (j in 1..5) {
            val x = if (j > 2) null else j
            s = s + (x ?: continue@outer)
        }
        s = s + 100
    }
    return s
}
class Cell34(var v: Int)

fun sec34() {
    // return in expression position, in both directions.
    check("ctl01", firstOr34(3) == 103)
    check("ctl02", firstOr34(null) == -1)
    // break leaves the loop AND the expression that asked for it: 1+2+3+4.
    check("ctl03", sumUntilNull34(4) == 10)
    // continue skips only the rest of the iteration: 1+3+5.
    check("ctl04", sumOdd34() == 9)
    // A LABELLED signal raised from inside an expression still names the outer loop.
    check("ctl05", labelledBreak34() == 30)
    check("ctl06", labelledContinue34() == 15)
    // A `return` raised out of a lambda body is Kotlin's NON-LOCAL return: it leaves
    // sumWhilePositive34, not just the lambda, so the 9 after the -1 is never added.
    check("ctl07", sumWhilePositive34(listOf(3, 5, -1, 9)) == 8)
    check("ctl08", sumWhilePositive34(listOf(-2, 4)) == 0)

    // PREFIX ++ / -- yields the NEW value; the postfix form yields the old one.
    var i = 1
    check("ctl09", ++i == 2)
    check("ctl10", i == 2)
    check("ctl11", i++ == 2)
    check("ctl12", i == 3)
    check("ctl13", --i == 2 && i == 2)
    // ... on an indexed slot ...
    val arr = intArrayOf(5, 6)
    check("ctl14", ++arr[0] == 6 && arr[0] == 6)
    // ... and on a field.
    val c = Cell34(9)
    check("ctl15", ++c.v == 10 && c.v == 10)
    check("ctl16", c.v-- == 10 && c.v == 9)
}
fun sumWhilePositive34(xs: List<Int>): Int {
    var acc = 0
    xs.forEach { acc = acc + (if (it > 0) it else return acc) }
    return acc
}

// ===== SECTION 35: the CALL SURFACE - a parenthesized callee, and a body's `;` =====
// Every form here is ordinary Kotlin; each was refused by these grammars before.
// SYNTAX ground truth is again Kotlin's own tree: psi/CallWithManyClosures.txt records
// `val a = (f) {} {} {}` and `(f)() {} {} {}` with clean trees, psi/DoubleColon.txt has
// `(a::b)()` as a CALL_EXPRESSION whose callee is a PARENTHESIZED expression, and
// psi/FunctionExpressions.txt has `fun c() = fun name();` - the trailing semicolon
// included - also clean.
// psi/DoubleColon.txt is also the ground truth for the guard the last two checks pin:
// it puts `a::b.c` on one line and `(a::b)()` two lines below, and records them as TWO
// statements rather than one call of `c`. So an argument list must OPEN on its callee's
// own line, while a call whose arguments merely SPAN lines is unaffected.
fun add35(a: Int, b: Int): Int = a + b
fun twice35(f: (Int) -> Int, v: Int): Int = f(f(v))
fun konst35(): Int = 41;
fun anon35() = fun(): Int { return 5 };

fun sec35() {
    val tripler = { x: Int -> x * 3 }
    // A PARENTHESIZED callee is invoked like any other.
    check("cal01", (tripler)(7) == 21)
    check("cal02", (::add35)(2, 5) == 7)
    check("cal03", ((tripler))(2) == 6)
    // A trailing `;` after an expression body and after a block body.
    check("cal04", konst35() == 41)
    check("cal05", anon35()() == 5)
    // A lambda argument still reaches an ordinary call.
    check("cal06", twice35({ it + 1 }, 9) == 11)
    // A call whose ARGUMENTS span several lines keeps working - only a `(` that opens a
    // NEW line is refused (see the DoubleColon citation above).
    check("cal07", add35(
        1,
        2) == 3)
    check("cal08", add35(1,
                         2) == 3)
}

// ===== SECTION 36: scope functions =====
// let / also / takeIf / takeUnless - the four kotlin.standard scope functions whose
// receiver arrives as `it`. They are extensions on Any, so no receiver type owns them
// and both halves answer them from one universal branch (kScopeMethod /
// ktScopeMethod). A class MEMBER of the same name still wins, which is Kotlin's rule
// that a member always beats an extension - checked below with a class that declares
// its own let().
// run / apply / with bind the receiver as `this` instead, so they live in SECTION 40.
class Scope36(val n: Int) {
    fun let(k: Int): Int = n + k          // A MEMBER named let: it must win.
}
fun sec36() {
    check("scp1", "x".let { it + "y" } == "xy")
    check("scp2", 5.let { it * 2 } == 10)
    check("scp3", listOf(1, 2, 3).let { it.size } == 3)
    // also returns the RECEIVER, and its lambda is run for its effect.
    var seen = 0
    val same = listOf(4, 5).also { seen = it.size }
    check("scp4", seen == 2 && same.size == 2)
    check("scp5", "q".also { seen = it.length } == "q" && seen == 1)
    // takeIf answers the receiver or null; takeUnless is its negation.
    check("scp6", 7.takeIf { it > 3 } == 7)
    check("scp7", 2.takeIf { it > 3 } == null)
    check("scp8", 2.takeUnless { it > 3 } == 2)
    check("scp9", 7.takeUnless { it > 3 } == null)
    // A chain: each step hands its own value on.
    check("scp10", "ab".let { it + "c" }.let { it.length } == 3)
    // The member wins over the extension.
    check("scp11", Scope36(10).let(5) == 15)
    // let on a nullable receiver, the idiom the function exists for.
    val maybe: String? = "z"
    check("scp12", maybe?.let { it + "!" } == "z!")
    val none: String? = null
    check("scp13", none?.let { it + "!" } == null)
}

// ===== SECTION 37: the collections API =====
// The kotlin.collections operations a real program leans on. Every one of them is a
// METHOD, so both halves answer from the same table (kSeqMethod in
// kotlin-interpreter.abnf, ktSeqMethod in abnf/jsrtkotlin.go) and --cross diffs them.
// Values checked against the equivalent Java program under `java` (JDK 24) wherever
// Kotlin's answer is java.util's: List.hashCode is the 31*h + element fold, and a
// `sumOf` whose selector answers a Long sums at 64 bits.
fun sec37() {
    val xs = listOf(1, 2, 3, 4, 5)
    check("col1", xs.fold(0) { a, b -> a + b } == 15)
    check("col2", xs.fold("") { a, b -> a + b } == "12345")
    check("col3", xs.foldRight("") { a, b -> a + b } == "12345")
    check("col4", xs.reduce { a, b -> a * b } == 120)
    check("col5", xs.sumOf { it * 2 } == 30)
    // sumOf takes the WIDTH of its selector: a Long selector sums at 64 bits, where a
    // 32 bit accumulator would have wrapped 6000000000 to 1705032704.
    check("col6", "" + xs.sumOf { it.toLong() * 1000000000L } == "15000000000")
    check("col7", xs.filter { it % 2 == 1 }.toString() == "[1, 3, 5]")
    check("col8", xs.filterNot { it % 2 == 1 }.toString() == "[2, 4]")
    check("col9", xs.map { it * it }.toString() == "[1, 4, 9, 16, 25]")
    check("col10", xs.mapIndexed { i, v -> i * v }.toString() == "[0, 2, 6, 12, 20]")
    check("col11", xs.flatMap { listOf(it, it) }.size == 10)
    check("col12", listOf(listOf(1, 2), listOf(3)).flatten().toString() == "[1, 2, 3]")
    check("col13", xs.partition { it > 2 }.toString() == "([3, 4, 5], [1, 2])")
    check("col14", xs.chunked(2).toString() == "[[1, 2], [3, 4], [5]]")
    check("col15", xs.windowed(3).toString() == "[[1, 2, 3], [2, 3, 4], [3, 4, 5]]")
    check("col16", xs.zip(listOf("a", "b")).toString() == "[(1, a), (2, b)]")
    check("col17", xs.zip(listOf(10, 20)) { a, b -> a + b }.toString() == "[11, 22]")
    check("col18", xs.take(2).toString() == "[1, 2]" && xs.drop(3).toString() == "[4, 5]")
    check("col19", xs.takeLast(2).toString() == "[4, 5]" && xs.dropLast(3).toString() == "[1, 2]")
    check("col20", xs.reversed().toString() == "[5, 4, 3, 2, 1]")
    check("col21", listOf(3, 1, 2).sorted().toString() == "[1, 2, 3]")
    check("col22", listOf(3, 1, 2).sortedDescending().toString() == "[3, 2, 1]")
    val words = listOf("bbb", "a", "cc")
    check("col23", words.sortedBy { it.length }.toString() == "[a, cc, bbb]")
    check("col24", words.sortedByDescending { it.length }.toString() == "[bbb, cc, a]")
    check("col25", words.maxByOrNull { it.length } == "bbb" && words.minByOrNull { it.length } == "a")
    check("col26", xs.maxOrNull() == 5 && xs.minOrNull() == 1)
    check("col27", listOf<Int>().maxOrNull() == null)
    check("col28", xs.all { it > 0 } && !xs.all { it > 1 })
    check("col29", xs.none { it > 9 } && !xs.none { it > 4 })
    check("col30", xs.find { it > 3 } == 4 && xs.find { it > 9 } == null)
    check("col31", xs.first { it % 2 == 0 } == 2 && xs.firstOrNull { it > 9 } == null)
    check("col32", xs.indexOfFirst { it == 3 } == 2 && xs.indexOfFirst { it == 9 } == -1)
    check("col33", listOf(1, 1, 2, 2, 3).distinct().toString() == "[1, 2, 3]")
    check("col34", xs.joinToString("-") == "1-2-3-4-5")
    check("col35", xs.joinToString(", ", "[", "]") == "[1, 2, 3, 4, 5]")
    check("col36", xs.withIndex().toString() == "[(0, 1), (1, 2), (2, 3), (3, 4), (4, 5)]")
    // java.util.List.hashCode: 31*h + element, starting at 1.
    check("col37", listOf(1, 2, 3).hashCode() == 30817)
    check("col38", listOf<Int>().hashCode() == 1 && listOf(1, 2, 3).hashCode() == listOf(1, 2, 3).hashCode())
    check("col39", xs.count { it > 2 } == 3 && xs.count() == 5)
    check("col40", xs.mapNotNull { if (it > 3) it else null }.toString() == "[4, 5]")
    // A Sequence pipeline. Sequences are EAGER in this subset - the answer of a
    // finite pipeline is identical either way, which is what is asserted.
    check("col41", xs.asSequence().map { it * 2 }.filter { it > 4 }.toList().toString() == "[6, 8, 10]")
    check("col42", xs.single { it == 3 } == 3)
    check("col43", listOf(2).single() == 2)
    var acc37 = 0
    xs.forEachIndexed { i, v -> acc37 += i * v }
    check("col44", acc37 == 40)
    check("col45", xs.average() == 3.0)
    check("col46", xs.sum() == 15)
}

// ===== SECTION 38: Char is a real type, and the String is a sequence of it =====
// Kotlin's String has no charAt: it is get / the indexing operator, both answering a
// CHAR. Both halves used to answer a one-character STRING from charAt, so
// `s.charAt(0) == 'a'` was false while `s[0] == 'a'` was true - the same expression
// written two ways disagreeing with itself. Every Char-valued member now answers a
// Char, and the Char member surface (.code, .digitToInt(), the isX predicates, the
// case conversions) reads the CODE.
// Kotlin's Char is java.lang.Character's, so the classification and case answers were
// checked against the equivalent Java program under `java` (JDK 24).
fun sec38() {
    val s = "aBc"
    check("chr1", s[0] == 'a' && s.get(1) == 'B' && s.charAt(2) == 'c')
    check("chr2", s.first() == 'a' && s.last() == 'c')
    check("chr3", "z".single() == 'z')
    check("chr4", s.toCharArray().toString() == "[a, B, c]")
    var n38 = 0
    for (c in s) { if (c == 'a' || c == 'c') n38++ }
    check("chr5", n38 == 2)
    check("chr6", 'a'.code == 97 && 'A'.code == 65)
    check("chr7", '7'.digitToInt() == 7)
    check("chr8", 'a'.isLetter() && !'1'.isLetter())
    check("chr9", '1'.isDigit() && !'a'.isDigit())
    check("chr10", 'a'.isLetterOrDigit() && !' '.isLetterOrDigit())
    check("chr11", ' '.isWhitespace() && !'a'.isWhitespace())
    check("chr12", 'A'.isUpperCase() && 'a'.isLowerCase())
    check("chr13", 'a'.uppercaseChar() == 'A' && 'B'.lowercaseChar() == 'b')
    check("chr14", 'a'.toString() == "a" && "" + 'a' == "a")
    // Char arithmetic: 'a' + 1 is a Char in Kotlin, and comparison is by code.
    check("chr15", 'a' < 'b' && 'b' > 'a' && 'a' == 'a')
    check("chr16", 'a'.compareTo('b') == -1 && 'b'.compareTo('b') == 0)
    check("chr17", 'a'.hashCode() == 97)
    // A String is a sequence of Char, so the collection operations apply to it.
    check("chr18", s.map { it.code }.toString() == "[97, 66, 99]")
    check("chr19", s.filter { it.isUpperCase() }.toString() == "[B]")
    check("chr20", s.count { it.isLetter() } == 3)
    check("chr21", s.any { it == 'B' } && !s.any { it == 'z' })
    // The rest of the kotlin.text surface both halves were missing.
    check("str19", "  pad  ".trim() == "pad")
    check("str20", "  pad  ".trimStart() == "pad  " && "  pad  ".trimEnd() == "  pad")
    check("str21", "abc".startsWith("ab") && "abc".endsWith("bc") && !"abc".startsWith("b"))
    check("str22", "7".padStart(3, '0') == "007" && "7".padEnd(3) == "7  ")
    check("str23", "abc".reversed() == "cba")
    check("str24", "a,b,c".split(",").toString() == "[a, b, c]")
    check("str25", "a\nb".lines().toString() == "[a, b]")
    check("str26", "abcde".drop(2) == "cde" && "abcde".take(2) == "ab")
    check("str27", "abcde".dropLast(2) == "abc" && "abcde".takeLast(2) == "de")
    check("str28", "42".toInt() == 42 && "-7".toInt() == -7)
    check("str29", "" + "9000000000".toLong() == "9000000000")
    check("str30", "1.5".toDouble() == 1.5)
    check("str31", "abc".indexOf('b') == 1 && "abc".contains('b') && !"abc".contains('z'))
    check("str32", "abab".lastIndexOf("ab") == 2)
    // kotlin.text.format is java.util.Formatter.
    check("str33", "%.2f".format(3.14159) == "3.14")
    check("str34", "%d/%s".format(42, "z") == "42/z")
    // %f rounds HALF UP, which is java.util.Formatter's rule: `java` (JDK 24, with
    // the locale forced to en_US) prints "  1.3|", "2.68" and "0.3" for these three,
    // and Go's Sprintf - which rounds half to EVEN - printed "  1.2|", "2.67", "0.2"
    // until the compiler half got its own fixed-point renderer.
    check("str35", "%5.1f|".format(1.25) == "  1.3|")
    check("str36", "%.2f".format(2.675) == "2.68" && "%.1f".format(0.25) == "0.3")
    check("str37", "%s|%b".format("q", true) == "q|true")
}

// ===== SECTION 39: maps =====
// A Map is the runtime's shared {__dict, keys, vals} handle - the same shape Go maps
// and Python dicts use - so both halves carry one map value model and one member
// table. The maps below are built with associate / groupBy / toMap; the mapOf family
// of global builders is asserted in SECTION 40.
// A Map renders as java.util.AbstractMap.toString does: {a=1, b=2}.
fun sec39() {
    val squares = listOf(1, 2, 3).associate { it to it * it }
    check("map1", squares.toString() == "{1=1, 2=4, 3=9}")
    check("map2", squares[2] == 4)
    check("map3", squares.size == 3)
    check("map4", squares.containsKey(3) && !squares.containsKey(9))
    check("map5", squares.keys.toString() == "[1, 2, 3]")
    check("map6", squares.values.toString() == "[1, 4, 9]")
    check("map7", squares.isNotEmpty() && !squares.isEmpty())
    val byLen = listOf("a", "bb", "cc", "d").groupBy { it.length }
    check("map8", byLen.toString() == "{1=[a, d], 2=[bb, cc]}")
    check("map9", byLen[2].toString() == "[bb, cc]")
    val withV = listOf("x", "yy").associateWith { it.length }
    check("map10", withV.toString() == "{x=1, yy=2}")
    val byK = listOf("x", "yy").associateBy { it.length }
    check("map11", byK.toString() == "{1=x, 2=yy}")
    val fromPairs = listOf(1 to "a", 2 to "b").toMap()
    check("map12", fromPairs.toString() == "{1=a, 2=b}")
    check("map13", fromPairs.getValue(1) == "a")
    check("map14", fromPairs.getOrDefault(9, "?") == "?")
    // An entry is destructurable and carries key/value. Iterating the map DIRECTLY
    // (rather than its .entries) is asserted in SECTION 40.
    var joined = ""
    for ((k, v) in fromPairs.entries) { joined += "" + k + v }
    check("map15", joined == "1a2b")
    check("map16", fromPairs.entries.size == 2)
    check("map17", fromPairs.containsValue("b") && !fromPairs.containsValue("z"))
    // A MISS answers null, not Unit: kotlin.Map.get is declared to return V?, and the
    // indexed form is that same get. The compiler half used to answer the runtime's
    // undefined here (printing "kotlin.Unit", so `m[9] == null` was false) while the
    // interpreter answered null - a live cross-half divergence with no assertion on it.
    check("map21", fromPairs[9] == null && fromPairs.get(9) == null)
    check("map22", "" + fromPairs[9] == "null")
    check("map23", byLen[7] == null && squares[9] == null)
    // A Pair is (first, second) and destructures; `to` builds one.
    val p = 3 to "c"
    check("pair1", p.toString() == "(3, c)")
    check("pair2", p.first == 3 && p.second == "c")
    val (pa, pb) = p
    check("pair3", pa == 3 && pb == "c")
    check("pair4", listOf(1, 2, 3, 4).partition { it > 2 }.first.toString() == "[3, 4]")
    // A map is a sequence of entries, so the collection operations apply.
    check("map18", squares.map { it.value }.toString() == "[1, 4, 9]")
    check("map19", squares.filter { it.key > 1 }.size == 2)
    check("map20", squares.count() == 3)
}

// ===== SECTION 40: the global BUILDERS and the this-bound scope functions =====
// A global NAME is declared by each grammar's own builtin block - hostGlobals in
// kotlin-interpreter.abnf, js_scope_decl in kotlin-to-llvm-ir.abnf - not by the shared
// runtime, so mapOf/Pair/Triple/with/StringBuilder/buildString/sequenceOf are
// registered in BOTH by hand. What they BUILD is a shared shape whose whole member
// surface lives in abnf/jsrtkotlin.go, so the compiler-side port was the declaration.
//
// run / apply / with bind the receiver as `this`, so an unqualified `size` or
// `append("a")` inside the lambda is a member call on it. The interpreter binds it
// through kSetRecv (the channel a lambda-with-receiver already uses); the compiler's
// lambda is a bare IR closure with no receiver slot, so the builder parks the receiver
// on ktRecvStack and js_ktget consults it after a local and after the enclosing
// `this` - which can only ever turn "unknown name" into a member read.
fun sec40() {
    val m = mapOf(1 to "a", 2 to "b")
    check("gbl1", m.toString() == "{1=a, 2=b}" && m[1] == "a")
    val mm = mutableMapOf(1 to "a")
    mm[2] = "b"
    check("gbl2", mm.toString() == "{1=a, 2=b}" && mm.size == 2)
    check("gbl3", emptyMap<Int, Int>().size == 0)
    check("gbl4", Pair(1, 2).toString() == "(1, 2)")
    check("gbl5", Triple(1, 2, 3).toString() == "(1, 2, 3)")
    val (t1, t2, t3) = Triple(4, 5, 6)
    check("gbl6", t1 + t2 + t3 == 15)
    // with / run / apply bind the receiver as `this`.
    check("gbl7", with(listOf(1, 2, 3)) { size } == 3)
    check("gbl8", listOf(1, 2, 3).run { size } == 3)
    val built = mutableListOf(1, 2).apply { add(3) }
    check("gbl9", built.toString() == "[1, 2, 3]")
    // StringBuilder and buildString.
    val sb = StringBuilder()
    sb.append("a")
    sb.append(1)
    check("gbl10", sb.toString() == "a1" && sb.length == 2)
    check("gbl11", buildString { append("x"); append("y") } == "xy")
    check("gbl12", with(StringBuilder("s")) { append("t"); toString() } == "st")
    // Sequences and buildList / buildMap.
    check("gbl13", sequenceOf(1, 2, 3).map { it + 1 }.toList().toString() == "[2, 3, 4]")
    check("gbl14", buildList { add(1); add(2) }.toString() == "[1, 2]")
    check("gbl15", buildMap { put(1, "a") }.toString() == "{1=a}")
    var r40 = 0
    repeat(3) { r40 += it }
    check("gbl16", r40 == 3)
    // Iterating the MAP itself, rather than its .entries.
    var j40 = ""
    for ((k, v) in m) { j40 += "" + k + v }
    check("gbl17", j40 == "1a2b")
    var c40 = 0
    for (e in m) { c40++ }
    check("gbl18", c40 == 2)
}

// ===== SECTION 41: a declared type carries its WIDTH past the declaration =====
// SECTION 31 pinned the width a LOCAL `val x: Long = 1` carries. The same rule applies
// at every other binding site, and none of them had it: a property, a parameter, a
// parameter DEFAULT, a function's return type and a field WRITE all dropped the
// declared width, so a Long that never passed through a local was silently 32 bit.
// Kotlin/JVM's Int and Long are java.lang's, so each value below is what the
// equivalent Java program prints under `java` (JDK 24) - verified, not assumed.
class Widths41(val x: Long, val b: Byte = 127) {
    var y: Long = 1
}
fun ret41(): Long = 1
fun dflt41(v: Long = 1): Long = v
fun sec41() {
    // A declared RETURN type: `fun ret41(): Long = 1` answers a Long.
    check("wid1", "" + ret41() * 1000000 * 1000000 == "1000000000000")
    // A PROPERTY initializer, both the constructor-parameter form and the body form.
    val w = Widths41(1L)
    check("wid2", "" + w.x * 1000000 * 1000000 == "1000000000000")
    check("wid3", "" + w.y * 1000000 * 1000000 == "1000000000000")
    // A parameter DEFAULT, and the parameter itself.
    check("wid4", "" + dflt41() * 1000000 * 1000000 == "1000000000000")
    check("wid5", "" + dflt41(1L) * 1000000 * 1000000 == "1000000000000")
    // A field WRITE adopts the declared type: `w.y = 1` stores a Long, not an Int.
    w.y = 1
    check("wid6", "" + w.y * 1000000 * 1000000 == "1000000000000")
    // A Byte property wraps at its own width, which is what proves the width is on
    // the stored value rather than on the literal that produced it.
    var bb: Byte = w.b
    bb++
    check("wid7", "" + bb == "-128")
}

// ===== SECTION 42: LOCAL type declarations =====
// A class, an interface or an object declared INSIDE a function body. Every value
// below is what the equivalent Java program prints under `java` (JDK 24) - a local
// class, a local class implementing an interface, and a local class declared inside a
// loop body were all written out and run; only the local `object` singleton has no
// Java spelling and is asserted from the Kotlin specification.
// Known limitation, deliberately NOT asserted: a local class body does not CAPTURE the
// enclosing function's locals here (the constructor runs with a fresh scope chain, the
// same one a top-level class gets), so a member reading an enclosing `val` is out of
// scope. Kotlin captures; this subset does not.
fun sec42() {
    class L42(val v: Int) {
        fun twice(): Int = v * 2
    }
    val l = L42(21)
    check("loc1", l.v == 21)
    check("loc2", l.twice() == 42)
    // A local INTERFACE and a local class that implements it.
    interface Shape42 {
        fun area(): Int
    }
    class Sq42(val s: Int) : Shape42 {
        override fun area(): Int = s * s
    }
    val sh: Shape42 = Sq42(4)
    check("loc3", sh.area() == 16)
    check("loc4", sh is Shape42)
    // A local class declared inside a loop body is one type, not one per iteration.
    var total = 0
    for (i in 0 until 3) {
        class P42 {
            fun p(): Int = 7
        }
        total += P42().p()
    }
    check("loc5", total == 21)
    // A local `object` is a singleton, so its state survives between calls.
    object Counter42 {
        var n = 0
        fun bump(): Int { n += 1; return n }
    }
    check("loc6", Counter42.bump() == 1)
    check("loc7", Counter42.bump() == 2)
}

// ===== SECTION 43: `fun interface` and SAM conversion =====
// `fun interface F { fun go(x: Int): Int }` declares a single-abstract-method
// interface, and `F { ... }` converts a lambda into an instance of it. Java's
// @FunctionalInterface is the same construct, so the equivalent Java program was
// written and run under `java` (JDK 24): the values below are its output. The only
// piece with no Java spelling is Kotlin's implicit single parameter `it`, which the
// lambda sections already pin.
fun interface Op43 {
    fun run(a: Int): Int
}
fun interface Mk43 {
    fun make(a: String, b: String): String
}
fun sec43() {
    val o = Op43 { it + 100 }
    check("sam1", o.run(5) == 105)
    // A second conversion of the SAME interface is an independent instance.
    val o2 = Op43 { x -> x * x }
    check("sam2", o2.run(9) == 81)
    check("sam3", "" + o.run(1) + "/" + o2.run(1) == "101/1")
    // More than one parameter is fine; `fun interface` constrains the count of
    // abstract MEMBERS, not of parameters.
    val m = Mk43 { x, y -> x + "-" + y }
    check("sam4", m.make("a", "b") == "a-b")
    // The converted value is an instance of the interface.
    check("sam5", o is Op43)
}

// ===== SECTION 44: a unary minus is folded into the LITERAL =====
// `2147483648` on its own is out of Int range and is a Long; `-2147483648` is
// Int.MIN_VALUE and stays 32 bit, because Kotlin types the negated CONSTANT rather than
// negating a Long. Kotlin/JVM's Int is java.lang's, so the values below are what the
// equivalent Java program prints under `java` (JDK 24) - verified, not assumed:
// -2147483648, 2147483647, 0 for x, x - 1 and x * 2, and -2147483649 for the Long.
fun sec44() {
    val x = -2147483648
    check("neg1", "" + x == "-2147483648")
    // The wrap is the proof the value is 32 bit: a Long would answer -2147483649.
    check("neg2", "" + (x - 1) == "2147483647")
    check("neg3", "" + (x * 2) == "0")
    // An explicit Long suffix is untouched - the fold applies to the literal's TYPE,
    // not to the minus.
    val y = -2147483648L
    check("neg4", "" + (y - 1) == "-2147483649")
    // Everything smaller is unchanged, and the minus still works as an operator.
    val z = -1
    check("neg5", "" + (z * 3) == "-3")
    val w = 2147483648
    check("neg6", "" + (w - 1) == "2147483647")
}

// ===== SECTION 45: structural equality on the collection shapes =====
// Kotlin's `==` is equals(), and a List, a Set, a Map, a Pair and a Triple all have
// STRUCTURAL equality - they are java.util's on the JVM, so the answers below are the
// ones the equivalent Java program prints under `java` (JDK 24): List.of(1,2).equals(
// List.of(1,2)) is true, Map.of(1,2).equals(Map.of(1,2)) is true, and the hashes are
// 994 for [1, 2], 4066 for [a, b], 3 for {1=2} and 1025 for [[1, 2]]. Before this both
// halves answered FALSE for every one of them while hashCode was already element-wise,
// so equality and hash disagreed.
data class Cfg45(val n: Int, val tags: List<String>)
fun sec45() {
    check("seq1", listOf(1, 2) == listOf(1, 2))
    check("seq2", listOf(1, 2) != listOf(1, 3))
    check("seq3", listOf(listOf(1), listOf(2)) == listOf(listOf(1), listOf(2)))
    check("seq4", mapOf(1 to 2) == mapOf(1 to 2))
    check("seq5", mapOf("a" to 1, "b" to 2) == mapOf("b" to 2, "a" to 1))
    check("seq6", setOf(1, 2) == setOf(1, 2))
    check("seq7", Pair(1, 2) == Pair(1, 2) && Pair(1, 2) != Pair(1, 3))
    check("seq8", Triple(1, 2, 3) == Triple(1, 2, 3))
    // A data class whose component IS a collection compares component-wise too.
    check("seq9", Cfg45(1, listOf("a")) == Cfg45(1, listOf("a")))
    check("sq10", Cfg45(1, listOf("a")) != Cfg45(1, listOf("b")))
    // Equality and hashCode must agree; the numbers are java's.
    check("sq11", listOf(1, 2).hashCode() == 994)
    check("sq12", listOf("a", "b").hashCode() == 4066)
    check("sq13", listOf(listOf(1, 2)).hashCode() == 1025)
    check("sq14", mapOf(1 to 2).hashCode() == 3)
    check("sq15", Pair(1, 2).hashCode() == 33 && Triple(1, 2, 3).hashCode() == 1026)
    // `in`, indexOf and distinct all go through the same equality.
    check("sq16", listOf(1, 2) in listOf(listOf(1, 2)))
    check("sq17", listOf(listOf(1, 2), listOf(3)).indexOf(listOf(3)) == 1)
    check("sq18", listOf(listOf(1), listOf(1)).distinct().size == 1)
    // A named argument to `copy` must land in THIS class's slot. The compiler half
    // keyed its parameter-name table by method name alone, so the second data class in
    // a file overwrote the first one's slots and `Pt13(1, 2).copy(y = 9)` answered
    // Pt13(x=9, y=2) - which is exactly what section 13 asserts, and which stayed green
    // only because this file had one data class with a named copy in it.
    check("sq19", "" + Cfg45(1, listOf("a")).copy(tags = listOf("b")) == "Cfg45(n=1, tags=[b])")
}

// ===== SECTION 46: sorting through a user Comparable, and Comparator =====
// `sorted()` on a class that declares `operator fun compareTo` sorts by THAT, the way
// `<` already did; before this the key comparison fell through to a numeric one, no
// pair was ever "less", and the list came back in its original order in both halves.
class V46(val n: Int) : Comparable<V46> {
    override fun compareTo(other: V46): Int = n - other.n
    override fun toString(): String = "V($n)"
}
fun sec46() {
    val l = listOf(V46(3), V46(1), V46(2))
    check("cmp1", "" + l.sorted() == "[V(1), V(2), V(3)]")
    check("cmp2", "" + l.sortedDescending() == "[V(3), V(2), V(1)]")
    check("cmp3", l.maxOrNull()!!.n == 3 && l.minOrNull()!!.n == 1)
    check("cmp4", l.max().n == 3 && l.min().n == 1)
    check("cmp5", V46(1) < V46(2) && V46(3) > V46(2))
    // A Comparator is a plain function of two elements; compareBy builds one.
    check("cmp6", "" + listOf(3, 1, 2).sortedWith(compareBy { it }) == "[1, 2, 3]")
    check("cmp7", "" + listOf(3, 1, 2).sortedWith(compareByDescending { it }) == "[3, 2, 1]")
    check("cmp8", "" + l.sortedWith(compareBy { it.n }) == "[V(1), V(2), V(3)]")
}

// ===== SECTION 47: the throwable hierarchy and typed catch =====
// java.lang's chain is Kotlin/JVM's; every relation below was checked with `instanceof`
// under `java` (JDK 24). The interpreter half used to parent every builtin directly to
// Throwable (so `IllegalStateException("z") is Exception` was false) and the compiler
// half used to run the FIRST catch clause whatever its declared type.
fun sec47() {
    check("exc1", IllegalStateException("z") is Exception)
    check("exc2", IllegalStateException("z") is RuntimeException)
    check("exc3", NumberFormatException("z") is IllegalArgumentException)
    check("exc4", IndexOutOfBoundsException("z") is RuntimeException)
    check("exc5", StackOverflowError() is Error && OutOfMemoryError() is Error)
    check("exc6", IllegalAccessException() is Exception)
    var hit = ""
    try { throw IllegalStateException("a") } catch (e: RuntimeException) { hit = "rt:" + e.message }
    check("exc7", hit == "rt:a")
    // The clause SELECTION is by declared type, and a clause that does not match is
    // skipped rather than taken.
    hit = ""
    try { throw NumberFormatException("b") } catch (e: IllegalStateException) { hit = "ise" } catch (e: IllegalArgumentException) { hit = "iae" }
    check("exc8", hit == "iae")
    // An Error is not an Exception, so the inner clause must NOT take it.
    hit = ""
    try {
        try { throw StackOverflowError() } catch (e: Exception) { hit = "inner" }
    } catch (e: Throwable) { hit = "outer" }
    check("exc9", hit == "outer")
    // A user exception class still matches its own declared supertype.
    hit = ""
    try { throw MyExc47("m") } catch (e: IllegalStateException) { hit = "ise" } catch (e: Exception) { hit = "exc:" + e.message }
    check("ex10", hit == "exc:m")
}
class MyExc47(m: String) : Exception(m)

// ===== SECTION 48: implicit receivers form a stack =====
// A name the innermost receiver does not answer falls through to the enclosing one:
// `with(P) { tag }` inside a member of Q is this@Q.tag. Both halves used to stop at the
// innermost `this` and fail with "unknown name: tag". And a bare `Inner()` built inside
// its outer's own member captures that instance, so this@Outer answers the OUTER one -
// before this it answered the inner instance in both halves.
class P48(val n: Int)
class Q48(val tag: String) {
    fun viaWith(): String = with(P48(1)) { "" + n + tag }
    inner class In48(val q: Int) {
        fun who(): String = this@Q48.tag + "/" + q
        fun bare(): String = tag
    }
    fun make(): In48 = In48(7)
}
fun sec48() {
    check("rcv1", Q48("hi").viaWith() == "1hi")
    check("rcv2", Q48("O").make().who() == "O/7")
    check("rcv3", Q48("O").make().bare() == "O")
    check("rcv4", Q48("Z").In48(1).who() == "Z/1")
}

// ===== SECTION 49: the stdlib members the subset was missing =====
// Every call below aborted the run with "unknown list method" / "unknown String method"
// / "unknown name" in BOTH halves. They are grouped rather than spread out because they
// were added as one sweep over the kotlin.collections, kotlin.text and kotlin.ranges
// member lists.
fun sec49() {
    check("std1", "" + listOf(1, null, 2).filterNotNull() == "[1, 2]")
    check("std2", "" + listOf(1, 2, 3).filterIndexed { i, _ -> i > 0 } == "[2, 3]")
    check("std3", listOf(1, 2, 3).findLast { it < 3 } == 2)
    check("std4", listOf(1, 2, 1).lastIndexOf(1) == 2)
    check("std5", "" + listOf(1, 2, 3).takeWhile { it < 3 } == "[1, 2]")
    check("std6", "" + listOf(1, 2, 3).dropWhile { it < 3 } == "[3]")
    check("std7", "" + listOf(1, 2, 3).distinctBy { it % 2 } == "[1, 2]")
    check("std8", listOf(1, 2, 3).containsAll(listOf(1, 2)))
    check("std9", listOf(1, 2, 3).elementAt(1) == 2 && listOf(1, 2, 3).getOrNull(9) == null)
    check("st10", listOf(1, 2, 3).getOrElse(9) { -1 } == -1)
    check("st11", listOf(1, 2, 3).maxOf { it * 2 } == 6 && listOf(1, 2, 3).minOf { it * 2 } == 2)
    check("st12", "" + listOf(1, 2, 3).plus(4) == "[1, 2, 3, 4]")
    check("st13", "" + listOf(1, 2, 3).minus(2) == "[1, 3]")
    check("st14", "" + listOf(1, 2, 3).intersect(listOf(2, 3)) == "[2, 3]")
    check("st15", "" + listOf(1, 2).union(listOf(2, 4)) == "[1, 2, 4]")
    check("st16", "" + listOf(1, 2, 3).subtract(listOf(1)) == "[2, 3]")
    check("st17", "" + listOf(1, 2, 3).slice(0..1) == "[1, 2]")
    check("st18", "" + listOf(1, 2, 3).subList(0, 2) == "[1, 2]")
    check("st19", "" + listOf(1, 2, 3).zipWithNext() == "[(1, 2), (2, 3)]")
    check("st20", "" + listOf(1, 2, 3).runningFold(0) { a, b -> a + b } == "[0, 1, 3, 6]")
    check("st21", "" + listOf(1, 2, 3).ifEmpty { listOf(0) } == "[1, 2, 3]")
    check("st22", "" + listOf(1, 2, 3).onEach { } == "[1, 2, 3]")
    val ml = mutableListOf(1, 2, 3)
    check("st23", ml.removeAt(1) == 2 && "" + ml == "[1, 3]")
    // kotlin.text.
    check("st24", "hello".replace("l", "L") == "heLLo")
    check("st25", "hello".replaceFirst("l", "L") == "heLlo")
    check("st26", "  ".isBlank() && "x".isNotBlank())
    check("st27", "hello".removePrefix("he") == "llo" && "hello".removeSuffix("lo") == "hel")
    check("st28", "hello".removeSurrounding("h", "o") == "ell")
    check("st29", "a.b.c".substringBefore(".") == "a" && "a.b.c".substringAfter(".") == "b.c")
    check("st30", "a.b.c".substringBeforeLast(".") == "a.b" && "a.b.c".substringAfterLast(".") == "c")
    check("st31", "".ifEmpty { "x" } == "x" && "  ".ifBlank { "y" } == "y")
    check("st32", "hello".capitalize() == "Hello")
    check("st33", "hello".subSequence(1, 3) == "el")
    check("st34", "hello".single { it == 'h' } == 'h')
    // kotlin.collections.Map.
    check("st35", "" + mapOf("a" to 1, "b" to 2).filterValues { it > 1 } == "{b=2}")
    check("st36", "" + mapOf("a" to 1, "b" to 2).filterKeys { it == "a" } == "{a=1}")
    check("st37", "" + mapOf("a" to 1).mapValues { it.value + 1 } == "{a=2}")
    check("st38", "" + mapOf("a" to 1).mapKeys { it.key + "!" } == "{a!=1}")
    check("st39", "" + mapOf("a" to 1).plus("b" to 2) == "{a=1, b=2}")
    check("st40", "" + mapOf("a" to 1, "b" to 2).minus("a") == "{b=2}")
    // Map.filter answers a MAP, not a list of entries.
    check("st41", "" + mapOf("a" to 1, "b" to 2).filter { it.value > 1 } == "{b=2}")
    // The top-level functions.
    check("st42", maxOf(1, 2) == 2 && minOf(1, 2, 0) == 0)
    check("st43", "" + listOfNotNull(1, null, 2) == "[1, 2]")
    check("st44", "" + List(3) { it } == "[0, 1, 2]" && "" + arrayOfNulls<Int>(2) == "[null, null]")
    check("st45", lazy { 5 }.value == 5)
    check("st46", 7.coerceIn(0, 5) == 5 && 1.coerceIn(0, 5) == 1)
    // kotlin.math keeps the operand's TYPE: abs(-1.5) is 1.5 and max(1.5, 2.0) is 2.0,
    // which is what `java` prints for Math.abs / Math.max on the same values.
    check("st47", "" + abs(-1.5) == "1.5" && "" + max(1.5, 2.0) == "2.0")
    check("st48", "" + abs(-3L) == "3" && abs(-3) == 3)
    // require / check / error, and runCatching's Result.
    check("st49", runCatching { 1 }.isSuccess && runCatching { error("x") }.isFailure)
    check("st50", runCatching { error("boom") }.exceptionOrNull()!!.message == "boom")
    check("st51", runCatching { error("x") }.getOrDefault(9) == 9 && runCatching { 5 }.getOrThrow() == 5)
    var msg = ""
    try { require(false) { "need it" } } catch (e: IllegalArgumentException) { msg = e.message!! }
    check("st52", msg == "need it")
    // NOT `check(false)`: this file declares its own two-argument `check`, which wins
    // over the stdlib one exactly as Kotlin's shadowing rule says. checkNotNull is the
    // same contract function under a name the file does not shadow.
    try { checkNotNull(null) } catch (e: IllegalStateException) { msg = e.message!! }
    check("st53", msg == "Required value was null.")
    // joinToString's TRAILING LAMBDA is its transform, not its separator. Both halves
    // read it as the separator and printed the lambda between the elements - a page of
    // script text in the interpreter and "[function]" in the compiler.
    check("st54", listOf(1, 2, 3).joinToString { "" + it * 2 } == "2, 4, 6")
    check("st55", listOf(1, 2, 3).joinToString("-") { "" + it } == "1-2-3")
    // `+` on a collection is kotlin.collections.plus. It used to fall through to the
    // numeric/JavaScript tail: 0 in the interpreter, the string "1,23" in the compiler.
    check("st56", "" + (listOf(1, 2) + listOf(3)) == "[1, 2, 3]")
    check("st57", "" + (listOf(1, 2) + 3) == "[1, 2, 3]")
    check("st58", "" + (mapOf("a" to 1) + ("b" to 2)) == "{a=1, b=2}")
    // A String on the left still wins, and renders the list Kotlin's way.
    check("st59", "x" + listOf(1) == "x[1]")
}

// ===== SECTION 50: a Set is its own shape, not a List =====
// Every set constructor used to be an alias of listOf, so a set did not
// deduplicate, two sets with the same elements in different orders were unequal,
// and a List compared EQUAL to a Set. A Set is its own value now.
// The answers below are java's (JDK 24), because Kotlin/JVM's Set IS
// java.util.Set: Set.of(1, 2).equals(Set.of(2, 1)) is true, List.of(1).equals(
// Set.of(1)) is false and Set.of(1, 2).hashCode() is 3 - AbstractSet's hash is the
// plain SUM of the element hashes, where AbstractList's is the 31-fold.
fun sec50() {
    val s = setOf(3, 1, 2, 1)
    check("set1", s.size == 3 && "$s" == "[3, 1, 2]")
    check("set2", setOf(2, 1) == setOf(1, 2))
    check("set3", !(listOf(1) == setOf(1)) && !(setOf(1) == listOf(1)))
    check("set4", setOf(1, 2).hashCode() == 3 && listOf(1, 2).hashCode() == 994)
    check("set5", setOf("a", "b").hashCode() == 195)
    // Iteration and `in`.
    var seen = 0
    for (x in s) { seen += x }
    check("set6", seen == 6 && 2 in s && 9 !in s && s.contains(1))
    // Every Iterable member answers a LIST even for a Set receiver; only the
    // set-valued operators keep the Set shape.
    check("set7", "" + s.map { it * 2 } == "[6, 2, 4]")
    check("set8", "" + s.sorted() == "[1, 2, 3]" && "" + s.filter { it > 1 } == "[3, 2]")
    check("set9", s.joinToString("-") == "3-1-2" && s.sum() == 6 && s.count() == 3)
    check("set10", "" + s.toList() == "[3, 1, 2]")
    // The mutable surface: add reports whether the set GREW.
    val ms = mutableSetOf(1)
    check("set11", ms.add(2) && !ms.add(1) && "$ms" == "[1, 2]")
    check("set12", ms.remove(1) && "$ms" == "[2]" && ms.size == 1)
    // union / intersect / subtract answer a Set, distinct answers a List.
    check("set13", setOf(1, 2).union(setOf(2, 3)) == setOf(1, 2, 3))
    check("set14", (setOf(1, 2) intersect setOf(2, 3)) == setOf(2))
    check("set15", (setOf(1, 2) subtract setOf(2, 3)) == setOf(1))
    check("set16", listOf(1, 2).union(listOf(2, 3)) == setOf(3, 2, 1))
    check("set17", listOf(1, 1, 2).toSet() == setOf(2, 1))
    check("set18", "" + listOf(1, 1, 2).distinct() == "[1, 2]")
    // `+` and `-` keep the receiver's shape.
    check("set19", setOf(1, 2) + 3 == setOf(1, 2, 3) && setOf(1, 2) - 1 == setOf(2))
    // The other constructors. sortedSetOf is a TreeSet, so it is ordered.
    check("set20", "" + sortedSetOf(3, 1, 2) == "[1, 2, 3]")
    check("set21", emptySet<Int>().isEmpty() && setOfNotNull(1, null, 1) == setOf(1))
    check("set22", hashSetOf(1, 1) == linkedSetOf(1) && s.toHashSet() == s)
    check("set23", setOf(1, 2).containsAll(listOf(1)) && !setOf(1).containsAll(listOf(1, 2)))
    // Map.keys is a Set (java.util.Map.keySet); Map.values is a Collection.
    val m = mapOf("a" to 1, "b" to 2)
    check("set24", m.keys == setOf("b", "a") && "" + m.values == "[1, 2]")
    check("set25", m.keys.hashCode() == 195)
    // A set OF sets still compares structurally.
    check("set26", setOf(setOf(1), setOf(2)) == setOf(setOf(2), setOf(1)))
    // A `-` on a LIST or a MAP is kotlin.collections.minus too; it used to fall
    // through to the numeric tail and answer 0 in both halves.
    check("set27", "" + (listOf(1, 2, 3) - 1) == "[2, 3]")
    check("set28", "" + (listOf(1, 2, 3) - listOf(2)) == "[1, 3]")
    check("set29", "" + (mapOf("a" to 1, "b" to 2) - "a") == "{b=2}")
}

// ===== SECTION 51: hashCode is java's, and it agrees with equals =====
// Every answer below was read off the equivalent Java program under `java`
// (JDK 24), because Kotlin/JVM's Double/Long/Boolean/String/Char hashCode ARE
// java.lang's. The Double one was the defect: the value model used the TRUNCATED
// value, so 1.5.hashCode() was 1 where java says 1073217536 (doubleToLongBits xor
// its own high half). This matters beyond hashing, because hashCode must agree
// with the structural equals the collections use.
fun sec51() {
    check("hsh1", 1.5.hashCode() == 1073217536)
    check("hsh2", listOf(1.5).hashCode() == 1073217567)
    check("hsh3", (0.1).hashCode() == -1507852285 && (-1.5).hashCode() == -1074266112)
    check("hsh4", (3.0).hashCode() == 1074266112 && (1e300).hashCode() == -164130400)
    // +-0.0 differ in their BITS, and NaN collapses to one canonical pattern.
    check("hsh5", (0.0).hashCode() == 0 && (-0.0).hashCode() == -2147483648)
    check("hsh6", Double.NaN.hashCode() == 2146959360)
    check("hsh7", Double.POSITIVE_INFINITY.hashCode() == 2146435072)
    check("hsh8", Double.NEGATIVE_INFINITY.hashCode() == -1048576)
    // The subnormal floor: 4.9E-324 is the smallest positive Double, bits == 1.
    check("hsh9", (4.9E-324).hashCode() == 1)
    // The rest of the family, unchanged but never pinned before.
    check("hsh10", 1234567890123L.hashCode() == 1912276436 && 7.hashCode() == 7)
    check("hsh11", true.hashCode() == 1231 && false.hashCode() == 1237)
    check("hsh12", "ab".hashCode() == 3105 && 'a'.hashCode() == 97)
    check("hsh13", mapOf(1.5 to 2.5).hashCode() == 2147221504)
    // hashCode() and toString() are extensions on Any?, so a null receiver answers
    // them instead of throwing; both halves used to abort the run.
    val n: String? = null
    check("hsh14", n.hashCode() == 0 && n.toString() == "null")
    // Boolean owns hashCode/toString/not/compareTo; the compiled half used to abort
    // with "method call 'hashCode' on a boolean".
    check("hsh15", true.toString() == "true" && true.not() == false)
    check("hsh16", true.compareTo(false) == 1 && false.compareTo(false) == 0)
}

// ===== SECTION 52: Map.Entry is not a Pair =====
// An entry destructures like a Pair - `for ((k, v) in m)` - but it renders as a=1
// where a Pair renders as (a, 1), and it hashes as key XOR value where a Pair uses
// the data-class 31-fold. Both halves used to print `[(a, 1)]` for m.entries.
// java (JDK 24) on a LinkedHashMap {a=1, b=2}: entrySet() prints [a=1, b=2], one
// entry prints a=1, that entry hashes to 96 and the entry SET hashes to 192, which
// is exactly the map's own hash.
fun sec52() {
    val m = mapOf("a" to 1, "b" to 2)
    check("ent1", "" + m.entries == "[a=1, b=2]")
    check("ent2", "" + m.entries.first() == "a=1")
    check("ent3", m.entries.first().hashCode() == 96)
    check("ent4", m.entries.hashCode() == 192 && m.hashCode() == 192)
    // toList is a list of PAIRS and still renders as one.
    check("ent5", "" + m.toList() == "[(a, 1), (b, 2)]")
    // The entry's own members, and the destructuring that made it Pair-shaped.
    val e = m.entries.first()
    check("ent6", e.key == "a" && e.value == 1)
    check("ent7", e.component1() == "a" && e.component2() == 1)
    var acc = ""
    for ((k, v) in m) { acc += "$k$v" }
    check("ent8", acc == "a1b2")
    check("ent9", "" + m.entries.map { it.key } == "[a, b]")
    // The rendering surface around it: nested collections, a data class in a list,
    // a Pair as a map VALUE.
    check("ent10", "" + mapOf("x" to listOf(1, 2)) == "{x=[1, 2]}")
    check("ent11", "" + mapOf("p" to (1 to 2)) == "{p=(1, 2)}")
    check("ent12", "" + listOf(listOf(1), setOf(2)) == "[[1], [2]]")
    check("ent13", "" + listOf(mapOf("a" to 1)) == "[{a=1}]")
    check("ent14", "" + setOf(1 to 2) == "[(1, 2)]")
}

// ===== SECTION 53: lateinit is enforced =====
// Reading an unassigned `lateinit var` used to answer null in both halves; Kotlin
// throws UninitializedPropertyAccessException. A lateinit property cannot hold null,
// so "the field is still null" IS "not initialized yet" and no per-instance flag is
// needed. `::name.isInitialized` is the language's own test for it, and it used to
// abort both halves with "property .isInitialized of null" because the bare `::name`
// READ the property first.
class Li53 {
    lateinit var a: String
    var plain: String? = null
    fun bare(): Boolean = ::a.isInitialized
    fun qual(): Boolean = this::a.isInitialized
    fun read(): String = a
}
open class Base53 { lateinit var s: String }
class Sub53 : Base53()
fun sec53() {
    val o = Li53()
    check("lat1", !o.bare() && !o.qual())
    // A NULLABLE var is not lateinit: it still reads as null.
    check("lat2", o.plain == null)
    var msg = ""
    try { o.read() } catch (e: UninitializedPropertyAccessException) { msg = e.message ?: "" }
    check("lat3", msg == "lateinit property a has not been initialized")
    // It is a RuntimeException, so the ordinary catch clauses see it.
    var caught = false
    try { println(o.a) } catch (e: RuntimeException) { caught = true }
    check("lat4", caught)
    o.a = "z"
    check("lat5", o.bare() && o.qual() && o.read() == "z" && o.a == "z")
    // An INHERITED lateinit property is enforced through the super chain.
    val sub = Sub53()
    var inh = false
    try { println(sub.s) } catch (e: Exception) { inh = true }
    check("lat6", inh)
    sub.s = "q"
    check("lat7", sub.s == "q")
}

// ===== SECTION 54: the qualified stdlib paths =====
// `kotlin.math.abs(-3)` resolved in NEITHER half ("unknown name: kotlin") while the
// unqualified `abs(-3)` worked, because the builtins are global names here. A package
// path is a value now and its last segment is looked up wherever the builtin lives.
fun sec54() {
    check("pkg1", kotlin.math.abs(-3) == 3 && abs(-3) == 3)
    check("pkg2", kotlin.math.max(1, 2) == 2 && kotlin.math.min(1, 2) == 1)
    check("pkg3", "" + kotlin.collections.listOf(1, 2) == "[1, 2]")
    check("pkg4", kotlin.collections.setOf(1, 1) == setOf(1))
    check("pkg5", kotlin.text.buildString { append("x") } == "x")
    // The root package resolves too.
    check("pkg6", "" + kotlin.Pair(1, 2) == "(1, 2)")
}

// ===== SECTION 55: coroutines, as a synchronous subset =====
// launch / async / runBlocking / delay were UNDEFINED in both halves, so any program
// with a coroutine in it stopped at the import. They run their block immediately and
// to completion now. THE LIMITATION IS THE ORDERING, not the results: nothing
// suspends, so a `launch` block runs before the statement after it rather than
// concurrently. Both :description blocks state it; the assertions below only pin what
// this subset actually promises - the VALUES.
suspend fun work55(): Int { delay(1); return 42 }
fun sec55() = runBlocking {
    val d = async { work55() }
    check("cor1", d.await() == 42)
    val order = StringBuilder()
    launch { order.append("a") }
    order.append("b")
    // Eager, so "a" lands FIRST. Real kotlinx.coroutines prints "ba" here; this is
    // the documented ordering limitation, asserted so it cannot change silently.
    check("cor2", order.toString() == "ab")
    val j = launch { }
    j.join()
    check("cor3", j.isCompleted && !j.isActive)
    check("cor4", withContext(Dispatchers.Default) { 7 } == 7)
    check("cor5", coroutineScope { 8 } == 8 && supervisorScope { 9 } == 9)
    val two = listOf(async { 1 }, async { 2 })
    check("cor6", "" + awaitAll(two[0], two[1]) == "[1, 2]")
}

// ===== SECTION 57: the callable-reference surface =====
// `b::v`, `b::w`, `b::method` and `b::class` are ordinary Kotlin (bound references
// since 1.1) and the COMPILER aborted on every one of them with "callable reference
// not implemented" while the interpreter answered them - a live cross-half divergence
// that --cross never exercised because no test used one. Alongside it, an UNBOUND
// reference's accessors gave a wrong answer in the interpreter: `(Box57::v).get(b)`
// answered kotlin.Unit where Kotlin answers 3, because a type-qualified reference kept
// the class DESCRIPTOR as its receiver instead of taking one from the call.
//
// Everything below is the KCallable / KClass surface both halves now implement, pinned
// name by name. Semantics from the Kotlin documentation ("Callable references", "Class
// references", "Reflection"): an unbound reference takes the receiver as the FIRST
// argument of get/set/invoke, a bound one has it already; .name is the member's simple
// name and "<init>" for a constructor reference; KClass.simpleName and .qualifiedName
// agree for a top-level class in the default package; and two bound references to the
// same member of the same receiver are EQUAL.
class Box57(val v: Int) {
    var w: Int = 5
    fun twice(): Int = v * 2
    fun ownName(): String = this::v.name
    fun ownBare(): Int = ::v.get()
}
fun Box57.tripled(): Int = v * 3
fun top57(x: Int): Int = x + 100
fun apply57(f: (Box57) -> Int, b: Box57): Int = f(b)
fun sec57() {
    val b = Box57(3)
    val lst = listOf(Box57(2), Box57(1))
    // BOUND property references: get / invoke / set / name.
    check("ref1", b::v.get() == 3 && b::v.invoke() == 3 && b::v.name == "v")
    check("ref2", b::w.get() == 5)
    b::w.set(9)
    check("ref3", b.w == 9 && b::w.get() == 9)
    // UNBOUND (type-qualified) references take the receiver from the call.
    check("ref4", Box57::v.get(b) == 3 && Box57::v.invoke(b) == 3 && Box57::v.name == "v")
    Box57::w.set(b, 4)
    check("ref5", b.w == 4)
    // FUNCTION references, bound and unbound, called directly and through invoke.
    check("ref6", b::twice.invoke() == 6 && Box57::twice.invoke(b) == 6)
    check("ref7", b::twice() == 6 && Box57::twice.name == "twice")
    val f = b::twice
    check("ref8", f() == 6)
    // A reference to a member of a BUILTIN type: the base names a type, not a value.
    check("ref9", String::length.get("abcd") == 4)
    // A top-level function reference is the function value, and carries its name.
    check("ref10", ::top57.invoke(1) == 101 && ::top57.name == "top57")
    // A CONSTRUCTOR reference.
    check("ref11", ::Box57.invoke(7).v == 7)
    check("ref12", "" + listOf(7, 8).map(::Box57).map { it.v } == "[7, 8]")
    // CLASS LITERALS, from a value and from a type name.
    check("ref13", b::class.simpleName == "Box57" && Box57::class.simpleName == "Box57")
    check("ref14", Box57::class.qualifiedName == "Box57")
    check("ref15", Box57::class.isInstance(b) && !Box57::class.isInstance(3))
    check("ref16", "" + Box57::class == "class Box57")
    // .java is the same handle again - java.lang.Class renders "class Box57" too.
    check("ref17", b::class.java.simpleName == "Box57")
    check("ref18", String::class.simpleName == "String" && 3::class.simpleName == "Int")
    // A reference passed where a LAMBDA is expected - a builtin and a user function.
    check("ref19", "" + lst.map(Box57::v) == "[2, 1]")
    check("ref20", "" + lst.map(Box57::twice) == "[4, 2]")
    check("ref21", "" + lst.sortedBy(Box57::v).map(Box57::v) == "[1, 2]")
    check("ref22", apply57(Box57::v, b) == 3)
    // An EXTENSION function reference: a member the receiver really has still wins.
    check("ref23", b::tripled.invoke() == 9 && "" + lst.map(Box57::tripled) == "[6, 3]")
    // EQUALITY: two bound references to the same member of the same receiver.
    check("ref24", b::v == b::v && !(b::v == b::w))
    check("ref25", b::class == Box57::class)
    // A reference whose RECEIVER is an arbitrary expression.
    check("ref26", Box57(1)::v.get() == 1 && Box57(1)::class.simpleName == "Box57")
    // A NULLABLE receiver in front of the `::` - the `?` belongs to the type.
    check("ref27", b?::v.get() == 3)
    // A reference read from inside the class, qualified and bare.
    check("ref28", b.ownName() == "v" && b.ownBare() == 3)
    // A reference stored in a val keeps its accessors.
    val p = b::w
    p.set(11)
    check("ref29", p.get() == 11 && p.name == "w")
    check("ref30", "" + listOf(1, 2).map(::top57) == "[101, 102]")
    // A constructor reference is named "<init>", and KCallable.toString() renders the
    // REFERENCE rather than the value it reads. Real kotlin-reflect prints
    // "val Box57.v: kotlin.Int" and, without the reflection library on the class path,
    // "property v (Kotlin reflection is not available)"; neither is reproducible here,
    // so both halves agree on a short form and the divergence from the JVM is
    // deliberate. The interpreter used to answer "3" for the line below - the value -
    // where the compiled half rendered the reference.
    check("ref31", ::Box57.name == "<init>")
    check("ref32", b::v.toString() == "" + b::v && b::v.toString() != "3")
}

// ===== SECTION 56: the enum statics, and the MutableList mutating surface =====
// Two clusters found by probing next to the sections above, both wrong in BOTH
// halves and therefore invisible to the cross-check:
//   - `Col.values()` and `Col.valueOf(name)` aborted the COMPILED half outright
//     ("unknown method 'values'") while the interpreter answered them, and
//     `Col.entries` - Kotlin 1.9's EnumEntries property - answered kotlin.Unit in
//     the interpreter and aborted the compiler.
//   - every IN-PLACE MutableList mutator was missing in both halves while its
//     copying twin was present: sort/sortDescending/sortBy/sortWith/reverse next to
//     sorted/sortedBy/sortedWith/reversed, plus set, add(index, e), removeAll,
//     retainAll and clear.
enum class Col56 { RED, GREEN, BLUE }
fun sec56() {
    check("enm1", "" + Col56.values().toList() == "[RED, GREEN, BLUE]")
    check("enm2", "" + Col56.entries == "[RED, GREEN, BLUE]")
    check("enm3", Col56.valueOf("GREEN") == Col56.GREEN)
    check("enm4", Col56.BLUE.ordinal == 2 && Col56.BLUE.name == "BLUE")
    check("enm5", Col56.values().size == 3)
    // `when` over an enum, with the entries reachable unqualified in the branches.
    val w = when (Col56.GREEN) { Col56.RED -> 1; Col56.GREEN -> 2; Col56.BLUE -> 3 }
    check("enm6", w == 2)

    val l = mutableListOf(3, 1, 2)
    l.sort()
    check("mut1", "" + l == "[1, 2, 3]")
    l.sortDescending()
    check("mut2", "" + l == "[3, 2, 1]")
    l.sortBy { it }
    check("mut3", "" + l == "[1, 2, 3]")
    l.sortByDescending { it }
    check("mut4", "" + l == "[3, 2, 1]")
    l.sortWith(compareBy { it })
    check("mut5", "" + l == "[1, 2, 3]")
    l.reverse()
    check("mut6", "" + l == "[3, 2, 1]")
    check("mut7", l.set(0, 9) == 3 && "" + l == "[9, 2, 1]")
    l.add(1, 7)
    check("mut8", "" + l == "[9, 7, 2, 1]")
    check("mut9", l.removeAll(listOf(9, 1)) && "" + l == "[7, 2]")
    check("mut10", l.retainAll(listOf(2)) && "" + l == "[2]")
    l.clear()
    check("mut11", l.isEmpty() && "" + l == "[]")
    // Pair/Triple.toList, which aborted both halves with "unknown method 'toList'".
    check("mut12", "" + (1 to 2).toList() == "[1, 2]")
    check("mut13", "" + Triple(1, 2, 3).toList() == "[1, 2, 3]")
}

// ===== SECTION 58: a declaration in EVERY kind of block =====
// A user report - ordinary application Kotlin that would not compile here - reduced to
// a destructuring declaration inside a lambda body and inside a `when` branch. The
// COMPILER half had two different block productions: Statement/PlainStmt (a function
// body, an if/while/for body) and LamStmt/LamCtrl (a lambda body, and the `{ ... }` of
// a when/if ARM, which is a ValueBlock made of LamStmt). LamCtrl admitted a strict
// SUBSET of PlainStmt, so `val (a, b) = p`, a bare `val x: Int`, a modified local
// (`private val x = 1`) and a delegated `val x by lazy { }` all parsed at statement
// level and failed inside a lambda or a when branch. PEG's ordered choice never
// retries a committed alternative, so a rule missing from a block production cannot be
// recovered later - the two lists have to agree. The interpreter half had the mirror
// gap: its LamCtrl was missing the keyword-less destructuring ASSIGNMENT and the local
// `typealias`, which its own PlainStmt accepts. Every form below is therefore checked
// at all four sites: statement level, an if block, a lambda body and a when branch.
fun mk58(): Pair<Int, Int> = Pair(1, 2)
data class Named58(val first: Int, val second: Int)
fun sec58() {
    // 1. statement level.
    val (a1, b1) = mk58()
    // The keyword-less form. Parenthesized, it is NAME based - each entry reads the
    // property of its own name, which for a Pair is first/second - while the bracketed
    // spelling below is POSITION based whatever its entries are called.
    (val first, val second) = mk58()
    [val p1, val p2] = mk58()
    val bare1: Int
    bare1 = 7
    private58()
    check("blk1", a1 + b1 == 3 && first + second == 3 && p1 + p2 == 3 && bare1 == 7)
    check("blk1b", first == 1 && second == 2 && p1 == 1 && p2 == 2)

    // 2. an if block.
    if (a1 == 1) {
        val (a2, b2) = mk58()
        [val p3, val p4] = mk58()
        val lz2 by lazy { 40 + 2 }
        check("blk2", a2 + b2 == 3 && p3 + p4 == 3 && lz2 == 42)
    }

    // 3. a LAMBDA body - the reported failure.
    val fromLambda = run {
        val (a3, b3) = mk58()
        [val p5, val p6] = mk58()
        typealias58()
        fun local3(n: Int) = n * 10
        class Local3(val n: Int)
        val bare3: Int = 5
        val lz3 by lazy { 3 }
        a3 + b3 + p5 + p6 + local3(1) + Local3(2).n + bare3 + lz3
    }
    check("blk3", fromLambda == 3 + 3 + 10 + 2 + 5 + 3)

    // 4. a `when` BRANCH block - the second reported failure. Same production.
    val fromWhen = when (a1) {
        1 -> {
            val (a4, b4) = mk58()
            [val p7, val p8] = mk58()
            fun local4(n: Int) = n * 100
            val bare4: Int = 6
            val lz4 by lazy { 4 }
            a4 + b4 + p7 + p8 + local4(1) + bare4 + lz4
        }
        else -> 0
    }
    check("blk4", fromWhen == 3 + 3 + 100 + 6 + 4)

    // 5. the reported program's own shape: a local `fun`, then a destructuring
    //    declaration, inside the `{ }` of a when branch used as an expression body.
    check("blk5", extract58(1) == "[1, 2]/[3]" && extract58(2) == "none")
    // The NAME based reading in full: reversing the entries must NOT reverse the
    // values, and the same entries in BRACKETS must read positionally instead. The
    // interpreter used to read the parenthesized form positionally too.
    (val second: Int, val first: Int) = Named58(10, 20)
    check("blk6", first == 10 && second == 20)
    [val q1, val q2] = Named58(10, 20)
    check("blk7", q1 == 10 && q2 == 20)
    (val aa = first) = Named58(30, 40)
    check("blk8", aa == 30)
}
private fun private58() { }
fun typealias58() { }
fun decode58(): Pair<ByteArray, ByteArray> = Pair(byteArrayOf(1, 2), byteArrayOf(3))
fun extract58(cmd: Int): String = when (cmd) {
    1 -> {
        fun shortList(bytes: ByteArray) = bytes.asList().map { it.toShort() }
        val (opened, notOpened) = decode58()
        "" + shortList(opened) + "/" + shortList(notOpened)
    }
    else -> { "none" }
}

// ===== SECTION 59: the primitive-array builders and the array conversions =====
// From the same user report. `byteArrayOf` was "unknown name" in BOTH halves - only
// int/long/double/float/boolean/charArrayOf and arrayOf were declared - and so were
// shortArrayOf, the four unsigned builders and emptyArray. The ARRAY-to-collection
// surface was missing too: `asList` (the report's own call), sortedArray,
// sortedArrayDescending, reversedArray, contentToString and contentEquals all aborted
// with "unknown list method". An Array and a List are one shape in this value model, so
// each of them is the list operator under Kotlin's array name; contentToString and
// contentEquals matter because Kotlin's Array does NOT get a structural toString or
// equals. The value model is also width-blind - `byteArrayOf(200)`, which kotlinc
// rejects (200 is not a Byte; the legal spelling is `byteArrayOf((-56).toByte())`), is
// accepted here, which is a TYPE question and is documented at both declaration sites.
fun sec59() {
    check("arr1", byteArrayOf(1, 2, 3).size == 3)
    check("arr2", shortArrayOf(1, 2).size == 2)
    check("arr3", charArrayOf('a', 'b').size == 2 && booleanArrayOf(true).size == 1)
    check("arr4", intArrayOf(1).size == 1 && longArrayOf(1L).size == 1)
    check("arr5", doubleArrayOf(1.0).size == 1 && floatArrayOf(1.0f).size == 1)
    check("arr6", ubyteArrayOf(1u).size == 1 && ushortArrayOf(1u).size == 1)
    check("arr7", uintArrayOf(1u).size == 1 && ulongArrayOf(1uL).size == 1)
    check("arr8", emptyArray<Int>().size == 0 && arrayOf(1, 2).size == 2)
    // The size-and-initializer constructors of the same family.
    check("arr9", IntArray(3).size == 3 && ByteArray(2).size == 2 && CharArray(2).size == 2)
    check("arr10", UIntArray(2).size == 2 && ULongArray(2).size == 2)

    // A Byte element is signed 8-bit in Kotlin; the legal way to write one is a
    // conversion, and widening it with .toShort() keeps the value.
    val bs = byteArrayOf((-56).toByte(), 1)
    check("arr11", "" + bs.asList().map { it.toShort() } == "[-56, 1]")
    check("arr12", bs[0].toInt() == -56 && bs.size == 2)

    // The conversions - asList was the reported one.
    val xs = intArrayOf(3, 1, 2)
    check("arr13", "" + xs.asList() == "[3, 1, 2]")
    check("arr14", "" + xs.toList() == "[3, 1, 2]" && "" + xs.toMutableList() == "[3, 1, 2]")
    check("arr15", xs.toTypedArray().size == 3 && "" + xs.toSet() == "[3, 1, 2]")
    check("arr16", "" + xs.asIterable().toList() == "[3, 1, 2]")
    check("arr17", "" + xs.asSequence().toList() == "[3, 1, 2]")
    check("arr18", "" + listOf(1, 2).toIntArray().asList() == "[1, 2]")
    // The array spellings of sorted / sortedDescending / reversed.
    check("arr19", "" + xs.sortedArray().asList() == "[1, 2, 3]")
    check("arr20", "" + xs.sortedArrayDescending().asList() == "[3, 2, 1]")
    check("arr21", "" + xs.reversedArray().asList() == "[2, 1, 3]")
    // contentToString / contentEquals: Kotlin's Array has no structural toString/equals.
    check("arr22", xs.contentToString() == "[3, 1, 2]")
    check("arr23", intArrayOf(1, 2).contentEquals(intArrayOf(1, 2)))
    check("arr24", !intArrayOf(1, 2).contentEquals(intArrayOf(1, 3)))
    check("arr25", !intArrayOf(1, 2).contentEquals(intArrayOf(1)))
    // Indexing, iteration and the numeric folds on a primitive array.
    var sum59 = 0
    for (v in xs) { sum59 += v }
    check("arr26", sum59 == 6 && xs[0] == 3 && xs.sum() == 6)
    check("arr27", xs.average() == 2.0 && xs.max() == 3 && xs.min() == 1)
    check("arr28", xs.joinToString("-") == "3-1-2")
}

// ===== SECTION 60: the supertype GRAPH, qualified `is`, and the enum companion =====
// Four defects found by probing ordinary application Kotlin (a sealed hierarchy, an
// interface with a default method, a companion factory), each one a cross-half
// divergence or a wrong answer in both halves:
//   - the interpreter's member lookup walked ONE chain (`__super`), so a second
//     interface's default method, anything reached through an interface's own
//     supertypes, and a superclass's interface seen from an OBJECT declaration were
//     all invisible - while the compiled half answered every one of them. A nested
//     object made it worse: it is built while its owner's body is walked, so it
//     memoized an EMPTY parent list for the whole run.
//   - `is Shape.Circle` - the QUALIFIED spelling of a nested class, which is how a
//     sealed hierarchy is normally written - matched nothing in EITHER half, so the
//     `is` branch of a `when (this)` silently fell through to the else.
//   - an enum class's COMPANION was never attached by the compiled half ("unknown
//     method 'of'"), and in the interpreter its members could not see the enum's own
//     entries ("unknown name: A").
//   - `Level.valueOf(bad)` aborted the run in both halves instead of throwing the
//     IllegalArgumentException Kotlin throws, so the ordinary
//     `runCatching { valueOf(s) }.getOrNull()` parse idiom could not be written.
interface Aud60 { val label: String
                  fun audit(): String = "audit:" + label }
interface Extra60 { fun extra(): String = "extra" }
interface Deep60 : Extra60
sealed class Shape60 : Aud60 {
    data class Circle(val r: Int) : Shape60() { override val label = "circle" }
    data class Rect(val w: Int, val h: Int) : Shape60() { override val label = "rect" }
    object Empty : Shape60() { override val label = "empty" }
}
class Both60 : Aud60, Deep60 { override val label = "both" }
fun Shape60.area60(): Int = when (this) {
    is Shape60.Circle -> r * r
    is Shape60.Rect -> w * h
    Shape60.Empty -> 0
}
enum class Level60(val severity: Int) {
    DEBUG(0), INFO(1), ERROR(2);
    fun louderThan(o: Level60) = severity > o.severity
    fun countAll() = entries.size
    companion object {
        val fallback = DEBUG
        fun of(s: Int): Level60 = entries.first { it.severity == s }
    }
}
fun sec60() {
    // The supertype GRAPH: a second interface, an interface's own supertype, and a
    // superclass's interface reached from an object declaration.
    check("grf1", Both60().audit() == "audit:both")
    check("grf2", Both60().extra() == "extra")
    check("grf3", Shape60.Empty.audit() == "audit:empty")
    check("grf4", Shape60.Circle(2).audit() == "audit:circle")
    // The QUALIFIED nested-type name in an `is` check.
    check("grf5", Shape60.Circle(3).area60() == 9)
    check("grf6", Shape60.Rect(2, 3).area60() == 6 && Shape60.Empty.area60() == 0)
    check("grf7", Shape60.Circle(1) is Shape60.Circle && !(Shape60.Circle(1) is Shape60.Rect))
    check("grf8", listOf(Shape60.Circle(2), Shape60.Rect(2, 3), Shape60.Empty).map { it.area60() }
                  .sum() == 10)
    // The enum COMPANION: a factory, a property initialized from an entry, and the
    // bare `entries` of Kotlin 1.9 inside a member and inside the companion.
    check("enc1", Level60.of(2) == Level60.ERROR)
    check("enc2", Level60.fallback == Level60.DEBUG)
    check("enc3", Level60.DEBUG.countAll() == 3)
    check("enc4", Level60.ERROR.louderThan(Level60.INFO))
    // valueOf THROWS IllegalArgumentException for an unknown name, so runCatching
    // boxes it - the ordinary way to parse an enum out of text.
    check("enc5", runCatching { Level60.valueOf("WARN") }.getOrNull() == null)
    check("enc6", runCatching { Level60.valueOf("INFO") }.getOrNull() == Level60.INFO)
    var caught60 = ""
    try { Level60.valueOf("NOPE") } catch (e: IllegalArgumentException) { caught60 = "iae" }
    check("enc7", caught60 == "iae")
}

// ===== SECTION 61: labelled returns, including to an OUTER lambda =====
// `return@label` targeting a lambda was implemented for the INNERMOST lambda only:
// the return builder walked the walk-time retLabels stack, accepted a match whose
// kind was "fun", and reported everything else as "non-local labelled return not
// implemented". A plain non-local `return` out of a lambda already worked, so the
// unwinding machinery existed - what was missing was naming a target.
// Kotlin's rule is the one implemented here: `return@l` targets the NEAREST enclosing
// function or lambda labelled `l`, with that frame's value. A lambda target raises a
// {__lamret: l} marker rather than returning a control value, because the target may
// be an outer lambda with forEach/map/a user function in between - only the lambda
// whose own label matches absorbs it. A FUNCTION target reached through one or more
// lambdas reuses the __nlret marker every function frame already catches.
// The implicit label of an unlabelled lambda is its callee's name (`return@forEach`).
fun s61outer(): Int {
    var seen = 0
    listOf(1, 2).forEach outer@{ a ->
        listOf(3, 4).forEach { b -> if (b == 3) return@outer }
        seen += a
    }
    return seen
}
fun s61inner(): String {
    var out = ""
    listOf(1, 2, 3).forEach { if (it == 2) return@forEach; out += it }
    return out
}
fun s61nonlocal(): Int { listOf(1, 2, 3).forEach { if (it == 2) return 99 }; return 7 }
fun s61labelfun(): Int {
    listOf(1, 2).forEach { a -> listOf(3, 4).forEach { b -> if (b == 3) return@s61labelfun 42 } }
    return 0
}
fun s61expr(): Int {
    var n = 0
    listOf(1, 2, 3).forEach lam@{ x -> val y = if (x == 2) return@lam else x; n += y }
    return n
}
fun s61value(): Int = listOf(1, 2, 3).map outer@{ a -> listOf(9).map { return@outer a * 2 }; 0 }.sum()
fun s61run(): Int = run outer@{ run { return@outer 5 }; 9 }
fun sec61() {
    // The inner lambda's return@outer leaves the OUTER lambda, so `seen` is never added to.
    check("lret1", s61outer() == 0)
    check("lret2", s61inner() == "13")
    check("lret3", s61nonlocal() == 99)
    // return@functionName from two lambdas deep: a genuine non-local return.
    check("lret4", s61labelfun() == 42)
    // return@label in EXPRESSION position (the `?:` / if-else idiom).
    check("lret5", s61expr() == 4)
    // The labelled lambda's VALUE is what return@label carries.
    check("lret6", s61value() == 12)
    check("lret7", s61run() == 5)
}

// ===== SECTION 62: MEMBER extension functions, including the bodiless ones =====
// A member extension function - one declared inside a class or interface, whose `this`
// is the extension receiver while `this@Class` is the enclosing instance - already
// worked WITH a body. The BODILESS forms did not: `fun Int.qty(): String` in an
// interface and `abstract fun Int.scaled(): Int` in an abstract class both fell
// through to UnsupFun and aborted the whole run, even though the override beside them
// was already understood. MemberExtAbs is AbstractFun's shape with a receiver in
// front; like AbstractFun it contributes no implementation, and the override installs
// the real entry, which the lookup reaches from the runtime receiver.
interface Unit62 { fun Int.qty(): String }
abstract class Base62 {
    abstract fun Int.scaled(): Int
    fun apply62(x: Int): Int = x.scaled()
}
class Impl62 : Base62(), Unit62 {
    val factor = 3
    override fun Int.scaled(): Int = this * this@Impl62.factor
    override fun Int.qty(): String = "$this pcs"
    fun describe(x: Int): String = x.qty()
}
class Dsl62 { var out = ""
    fun String.emit() { out += this + ";" }
    fun build(): String { "a".emit(); "b".emit(); return out } }
fun sec62() {
    val i = Impl62()
    // Declared abstract in Base62, overridden in Impl62, called through a base member.
    check("mxf1", i.apply62(5) == 15)
    // Declared bodiless in an INTERFACE.
    check("mxf2", i.describe(4) == "4 pcs")
    // The dispatch receiver stays reachable from the body (`out` is Dsl62's).
    check("mxf3", Dsl62().build() == "a;b;")
    // In scope wherever an instance of the declaring class is the enclosing receiver.
    check("mxf4", with(i) { 7.scaled() } == 21)
}

// ===== SECTION 63: the whole `super` surface =====
// Three defects, all found by probing and all wrong ANSWERS rather than missing
// features:
//   - `super.property` had no branch at all in the field reader, so the {__superref}
//     marker fell through to the extension-property fallback and answered undefined:
//     `override val v get() = super.v + 1` quietly evaluated to 0.
//   - `super` inside an OBJECT EXPRESSION resolved its declaring class through the
//     walk-time class name, which an anonymous object does not have - so it failed
//     with "no super method". The object expression now pushes "<object>" (its
//     descriptor's own name) and the lookup walks the RECEIVER's supertype graph.
//   - a `super<Base>` qualifier was parsed and DROPPED, so both halves of
//     `super<I1>.tag() + super<I2>.tag()` resolved to the first parent.
// And one more the probe exposed underneath them: an object expression did not
// CONSTRUCT its superclass, so `(object : B() {}).v` answered Unit in both halves.
open class SB63 { open val v = 1
                  open val w: Int get() = 7
                  open fun g() = 10 }
interface SI63a { fun tag(): String = "a" }
interface SI63b { fun tag(): String = "b" }
class SD63 : SB63() {
    override val v get() = super.v + 100
    override val w: Int get() = super.w + 100
    override fun g() = super.g() + 1
    fun viaLambda(): Int = listOf(1).map { super.g() + 1000 }[0]
}
class STwo63 : SI63a, SI63b { override fun tag(): String = super<SI63a>.tag() + super<SI63b>.tag() }
class SOuter63 { open class N63 { open fun n() = 7 }
                 class NI63 : N63() { override fun n() = super.n() + 1 } }
fun s63local(): Int {
    open class LB63 { open fun q() = 3 }
    class LD63 : LB63() { override fun q() = super.q() + 1 }
    return LD63().q()
}
fun sec63() {
    check("sup1", SD63().v == 101)          // super.<backing field>
    check("sup2", SD63().w == 107)          // super.<computed property>
    check("sup3", SD63().g() == 11)         // super.method()
    check("sup4", SD63().viaLambda() == 1010)   // super from inside a lambda
    check("sup5", STwo63().tag() == "ab")   // super<Interface>.method(), two supertypes
    check("sup6", SOuter63.NI63().n() == 8) // super in a NESTED class
    check("sup7", s63local() == 4)          // super in a LOCAL class
    check("sup8", (object : SB63() { override fun g() = super.g() + 5 }).g() == 15)
    check("sup9", (object : SB63() { override val v get() = super.v + 5 }).v == 6)
    check("sup10", (object : SB63() {}).v == 1)   // the supertype's initializer RAN
    check("sup11", (object : SB63() {}).g() == 10)
}

// ===== SECTION 64: a lambda-with-receiver as a PARAMETER (the DSL builder shape) =====
// `val f: T.() -> Unit = { ... }` bound the receiver to `this`; the same type used as a
// FUNCTION PARAMETER did not, so every DSL builder - the commonest receiver-lambda
// shape in real Kotlin - died with "unknown name: <member>" the moment the body called
// a member of the receiver. The receiver arrived as the lambda's `it` instead of as
// `this`. A parameter whose declared type is a receiver function type now gets the
// same wrapper the local declaration already got, so both `t.body()` and `body(t)`
// bind `this`.
class Node64(val name: String) {
    val kids = mutableListOf<Node64>()
    val attrs = mutableListOf<String>()
    fun child(n: String, body: Node64.() -> Unit): Node64 { val c = Node64(n); c.body(); kids.add(c); return c }
    fun attr(k: String, v: String) { attrs.add("$k=$v") }
    fun render(): String {
        var s = "<" + name
        for (a in attrs) s += " " + a
        s += ">"
        for (k in kids) s += k.render()
        return s + "</" + name + ">"
    }
}
fun doc64(body: Node64.() -> Unit): Node64 { val r = Node64("root"); r.body(); return r }
fun applyTwice64(x: Int, f: Int.() -> Int): Int = x.f().f()
fun sec64() {
    val d = doc64 { attr("lang", "en"); child("body") { attr("id", "b") } }
    check("rcv1", d.render() == "<root lang=en><body id=b></body></root>")
    // A receiver function type on a BUILTIN receiver, invoked in extension position.
    check("rcv2", applyTwice64(3) { this + 1 } == 5)
    check("rcv3", doc64 { attr("k", "v") }.attrs.size == 1)
}

// ===== SECTION 65: super.property, and a setter that stores somewhere else =====
// `super.p = v` had no reading in the interpreter half at all: `Target` is rooted at an
// identifier and `super` matches one, so the write went to a variable named `super` and
// aborted with "unknown name: super", while the compiler half answered correctly. The
// same probe then exposed a second, deeper defect on this half only - a property setter
// wrote its ENTRY snapshot of `field` back over the slot when it returned, so a setter
// that stores anywhere else (`set(v) { super.p = v }`, the delegating override) had its
// store silently undone: `d.p = 7` left the base's property at 1. Kotlin materializes a
// backing field only when an accessor actually mentions `field`, which is what the
// write-back now honours. UNVERIFIED against kotlinc (none on this machine); the Kotlin
// documentation's "Calling the superclass implementation" and "Backing fields" sections
// are the source.
open class Base65 {
    open var p: Int = 1
    open val tag: String get() = "base"
    fun rawP(): Int = p
}
class Sub65 : Base65() {
    var log = ""
    override var p: Int
        get() = super.p
        set(v) { log += "s"; super.p = v + 10 }
    override val tag: String get() = super.tag + "+sub"
    fun poke(v: Int) { super.p = v }
    fun peek(): Int = super.p
}
open class Counted65 {
    var raw: Int = 0
    var doubled: Int
        get() = raw * 2
        set(v) { raw = v / 2 }
}
fun sec65() {
    val d = Sub65()
    check("sup1", d.p == 1 && d.peek() == 1)
    d.poke(3)
    check("sup2", d.p == 3)
    d.p = 7
    // The setter's `super.p = v + 10` must SURVIVE the setter returning.
    check("sup3", d.p == 17 && d.peek() == 17 && d.log == "s")
    d.p += 3
    check("sup4", d.p == 30)
    check("sup5", d.tag == "base+sub")
    // A plain base instance is unaffected, and a setter that DOES use `field` still
    // writes through it.
    val b = Base65()
    b.p = 9
    check("sup6", b.p == 9 && b.rawP() == 9)
    val c = Counted65()
    c.doubled = 10
    check("sup7", c.raw == 5 && c.doubled == 10)
}

// ===== SECTION 66: the delegated-property protocol beyond a user getValue =====
// Kotlin has four `by` delegate kinds. The interpreter half recognized only ONE of them
// (a user object with getValue), because its check was a bare method lookup that does
// not see the builtin Map surface - so `val a: Int by map`, the kotlin.properties
// Delegates boxes and `provideDelegate` all warned "delegated property not implemented"
// or aborted with "unknown name: Delegates", while the compiler half ran every one of
// them. Under kotlinc this section additionally needs `import kotlin.properties.Delegates`
// and `import kotlin.reflect.KProperty`.
class Conf66(val map: Map<String, Any?>) {
    val name: String by map
    val size: Int by map
}
class MConf66(val m: MutableMap<String, Any?>) {
    var nick: String by m
}
class Named66(val n: String) {
    operator fun getValue(t: Any?, p: KProperty<*>): String = "d:" + n
}
class Prov66 {
    operator fun provideDelegate(t: Any?, p: KProperty<*>): Named66 = Named66(p.name)
}
class Holder66 {
    val alpha: String by Prov66()
    val beta: String by Named66("fixed")
}
var obs66 = ""
var watched66: Int by Delegates.observable(0) { _, old, new -> obs66 += "$old>$new;" }
var gated66: Int by Delegates.vetoable(0) { _, _, new -> new >= 0 }
var late66: String by Delegates.notNull()
fun sec66() {
    // A Map delegate is keyed by the PROPERTY NAME, not by a key the code passes.
    val c = Conf66(mapOf("name" to "ann", "size" to 7))
    check("dlg1", c.name == "ann" && c.size == 7)
    val mc = MConf66(mutableMapOf("nick" to "zz"))
    check("dlg2", mc.nick == "zz")
    mc.nick = "qq"
    check("dlg3", mc.nick == "qq" && mc.m["nick"] == "qq")
    // provideDelegate substitutes the delegate at the DECLARATION, so it can read the
    // property's name before any access happens.
    val h = Holder66()
    check("dlg4", h.alpha == "d:alpha" && h.beta == "d:fixed")
    // kotlin.properties.Delegates: observable notifies AFTER, vetoable decides BEFORE.
    watched66 = 3
    watched66 = 5
    check("dlg5", watched66 == 5 && obs66 == "0>3;3>5;")
    gated66 = 4
    check("dlg6", gated66 == 4)
    gated66 = -1
    check("dlg7", gated66 == 4)
    late66 = "hi"
    check("dlg8", late66 == "hi")
    // A LOCAL delegated property binds the same box.
    val local: String by Named66("loc")
    check("dlg9", local == "d:loc")
    var lobs = ""
    var lwatched: Int by Delegates.observable(1) { _, old, new -> lobs += "$old/$new" }
    lwatched = 2
    check("dlg10", lwatched == 2 && lobs == "1/2")
}

// ===== SECTION 67: a supertype declared LATER in the file =====
// Kotlin has no forward-declaration rule: a class may extend one declared further down,
// and forward-referenced FUNCTIONS already worked in both halves. The compiler half
// emitted the top level in SOURCE order and resolved a supertype with a scope read at
// the moment the class statement ran, so `class Inh : Base()` above `open class Base`
// aborted with "variable not defined: Base" while the interpreter half answered. The
// same read serves an INTERFACE supertype, so one fix covers both; the class items of
// the top level are now emitted in topological order. A one-line interface body is here
// too: `fun f(): Int;` - a bodiless member followed by the ordinary statement separator
// - fell out of the compiler's member set and made the whole interface "not implemented",
// while the identical interface written across lines worked.
class Fwd67 : FwdBase67() {
    override fun label(): String = "fwd:" + super.label()
    fun total(): Int = seed + bonus67
}
class FwdIface67 : Marker67 { override fun f(): Int = 1 }
open class FwdBase67 {
    val seed = 5
    open fun label(): String = "base"
}
interface Marker67 { fun f(): Int; fun g(): Int = f() + 10 }
val bonus67 = 2
enum class Mode67 : Marker67 { ON, OFF; override fun f(): Int = if (this == ON) 1 else 0 }
object Solo67 : Marker67 { override fun f(): Int = 4 }
fun sec67() {
    val a = Fwd67()
    check("fwd1", a.total() == 7)
    check("fwd2", a.label() == "fwd:base")
    check("fwd3", FwdIface67().g() == 11)
    // An enum and an object may name a later-declared supertype as well.
    check("fwd4", Mode67.ON.g() == 11 && Mode67.OFF.g() == 10)
    check("fwd5", Solo67.g() == 14)
    check("fwd6", a is FwdBase67 && FwdIface67() is Marker67)
}

// ===== SECTION 68: member extension PROPERTIES (the DSL scope shape) =====
// `val Int.scaled get() = this * factor` declared INSIDE a class is the property twin of
// a member extension function, and the shape every scoped DSL (kotlinx.html, Compose
// modifiers, a units scope) is built from. The interpreter half had it; the compiler half
// dropped it into "property accessor not implemented (ignored)", so a read answered Unit
// and a WRITE aborted with "cannot set member 'slot' on float64". The accessor pair is
// now installed as extget$name / extset$name taking (dispatch, extensionReceiver[, value]),
// exactly like the ext$name member extension FUNCTIONS beside it. Probing this also found
// that a member extension declared in a COMPANION object was invisible to the compiled
// half's runtime lookup - its members hang on the class descriptor, one level above the
// method holder the lookup walked - so a companion member extension FUNCTION was broken
// too, and is asserted here.
class Units68(val factor: Int) {
    var store = 0
    val Int.scaled: Int get() = this * factor
    var Int.slot: Int
        get() = store
        set(v) { store = v + this }
    fun Int.viaFun(): Int = this.scaled + 1
    fun run(): String {
        val n = 4
        var s = "" + n.scaled
        n.slot = 5
        s += "/" + n.slot
        n.slot += 3
        s += "/" + n.slot
        s += "/" + n.viaFun()
        return s
    }
    fun direct(): Int = 7.scaled
}
class Wrap68(val sep: String) {
    val String.wrapped: String get() = sep + this + sep
    fun go(s: String): String = s.wrapped
}
object Reg68 {
    val factor = 3
    val String.width: Int get() = length * factor
    fun w(s: String): Int = s.width
}
class Holder68 {
    companion object {
        val f = 3
        val Int.tripled: Int get() = this * f
        fun Int.tripledFun(): Int = this * f
        fun go(n: Int): Int = n.tripled
        fun goFun(n: Int): Int = n.tripledFun()
    }
}
fun sec68() {
    check("mxp1", Units68(10).run() == "40/9/16/41")
    check("mxp2", Units68(2).direct() == 14)
    check("mxp3", Wrap68("|").go("x") == "|x|")
    // An `object` declaration and a companion object host them too.
    check("mxp4", Reg68.w("abcd") == 12)
    check("mxp5", Holder68.go(5) == 15 && Holder68.goFun(5) == 15)
}

// ===== SECTION 69: three collection builtins, and a call's EXPLICIT TYPE ARGUMENTS =====
// Map.getOrPut, the PREDICATE overload of MutableList.removeAll/retainAll, and
// List.filterIsInstance were missing from BOTH halves - the first two aborting with
// "unknown Map method" / "not a sequence: <lambda>", which is what a builtin table that
// lands on one half only looks like from the other side, so all three are added together.
// filterIsInstance is the interesting one: both grammars spelled a method-call suffix as
// `"." KId [ SkipAngle ] MArgs`, so a call's explicit type arguments were SKIPPED and
// never captured. They are captured now (MTypeArgs) and delivered for the two things
// that need them: filterIsInstance's element type, and a REIFIED type parameter of an
// extension function called in method position - `"x".isA<String>()` answered false for
// every T in both halves, because an extension's type-parameter list was skipped too.
inline fun <reified T> Any.isA69(): Boolean = this is T
class Bag69 { val n = 1 }
fun sec69() {
    // getOrPut: the standard way to build a map of collections.
    val g = mutableMapOf<String, MutableList<Int>>()
    g.getOrPut("a") { mutableListOf() }.add(1)
    g.getOrPut("a") { mutableListOf() }.add(2)
    g.getOrPut("b") { mutableListOf() }.add(3)
    check("blt1", g["a"]!!.size == 2 && g["b"]!!.size == 1 && g.size == 2)
    var calls = 0
    val c = mutableMapOf<String, Int>()
    check("blt2", c.getOrPut("x") { calls++; 5 } == 5)
    check("blt3", c.getOrPut("x") { calls++; 9 } == 5 && calls == 1)
    // removeAll / retainAll take a COLLECTION or a PREDICATE.
    val l = mutableListOf(1, 2, 3, 4, 5)
    check("blt4", l.removeAll { it % 2 == 0 } && l == listOf(1, 3, 5))
    check("blt5", !l.removeAll { it > 100 } && l.size == 3)
    val r = mutableListOf(1, 2, 3, 4, 5)
    check("blt6", r.retainAll { it > 3 } && r == listOf(4, 5))
    val d = mutableListOf(1, 2, 3, 4)
    check("blt7", d.removeAll(listOf(2, 4)) && d == listOf(1, 3))
    // filterIsInstance<T>() reads the call's type argument, and drops nulls.
    val xs: List<Any?> = listOf(1, "a", 2.5, "b", 3, null, Bag69())
    check("blt8", xs.filterIsInstance<String>() == listOf("a", "b"))
    check("blt9", xs.filterIsInstance<Int>() == listOf(1, 3))
    check("blt10", xs.filterIsInstance<Bag69>().size == 1)
    check("blt11", xs.filterIsInstance<Double>() == listOf(2.5))
    // A reified type parameter of an extension, called in METHOD position.
    check("blt12", "x".isA69<String>() && !"x".isA69<Int>())
    check("blt13", 7.isA69<Int>() && !7.isA69<String>())
    // A type argument on a method that does not use one is still ignored.
    check("blt14", listOf(1, 2, 3).map<Int, Int> { it + 1 } == listOf(2, 3, 4))
}

// ===== SECTION 70: six defects one page of application Kotlin found =====
// Written as ordinary application code (a value type with an infix extension, an enum
// whose constants carry their own behaviour, a nullable-receiver extension property,
// price strings, a range-dispatch `when`, and input parsing). Every assertion below
// failed somewhere before:
//   - `enum class L(w) { A(1) { override fun f() = ... }; abstract fun f() }` - an entry
//     with its OWN CLASS BODY had no reading in the compiler half, so the whole
//     declaration fell to "enum class not implemented" and the enum's NAME was then
//     undefined for the rest of the program; the interpreter half ran it.
//   - `infix fun Int.pctOf(m: Money)` used INFIX went through the method dispatcher in
//     the compiler half and aborted with "method call 'pctOf' on a number", while the
//     identical `25.pctOf(m)` in dotted form worked - and the interpreter half ran both.
//   - `val String?.safeLen get() = this?.length ?: -1` on a NULL receiver aborted the
//     interpreter half with "property .safeLen of null"; a nullable-receiver extension
//     property is exactly where a null receiver is legal.
//   - `"$"`, `"100$"`: a `$` not followed by an identifier or `{` is an ordinary
//     character (psi/StringTemplates.txt records it as REGULAR_STRING_PART with no error
//     element). BOTH halves refused every string containing a literal dollar.
//   - two `when` arms in a row whose conditions are `in <range>`: the first arm's VALUE
//     swallowed the next line's `in` as a membership operator and the parse died on the
//     second arm, in BOTH halves. An `in` that opens a line is a condition.
//   - `"nope".toInt()` aborted the program in both halves instead of THROWING
//     NumberFormatException, so neither try/catch nor runCatching could see it.
class Money70(val cents: Int) : Comparable<Money70> {
    operator fun plus(o: Money70) = Money70(cents + o.cents)
    operator fun times(n: Int) = Money70(cents * n)
    override fun compareTo(other: Money70) = cents.compareTo(other.cents)
    override fun toString(): String = "$" + (cents / 100) + "." + (cents % 100).toString().padStart(2, '0')
}
infix fun Int.pctOf70(m: Money70) = Money70(m.cents * this / 100)
enum class Level70(val weight: Int) {
    LOW(1) { override fun label() = "low" },
    HIGH(5) { override fun label() = "high" };
    abstract fun label(): String
    fun heavy() = weight > 3
}
val String?.safeLen70: Int get() = this?.length ?: -1
fun classify70(n: Int): String = when (n) {
    in 0..9 -> "digit"
    in 10..99 -> "two"
    in 100..999 -> "three"
    else -> if (n < 0) "neg" else "big"
}
fun main70(raw: String): Int = runCatching { raw.toInt() }.getOrElse { -1 }
fun sec70() {
    val a = Money70(1250)
    val b = Money70(375)
    check("app1", (a + b).toString() == "$16.25" && (a * 2).toString() == "$25.00")
    check("app2", maxOf(a, b) === a && listOf(a, b).sorted().first() === b)
    // An infix EXTENSION function, both ways round.
    check("app3", (25 pctOf70 a).cents == 312 && 25.pctOf70(a).cents == 312)
    // An enum entry with its own class body, plus the enum's own members.
    check("app4", Level70.LOW.label() == "low" && Level70.HIGH.label() == "high")
    check("app5", !Level70.LOW.heavy() && Level70.HIGH.heavy())
    check("app6", Level70.entries.size == 2 && Level70.valueOf("HIGH").ordinal == 1)
    check("app7", Level70.LOW.name == "LOW" && Level70.LOW.weight == 1)
    // An extension property on a NULLABLE receiver.
    val missing: String? = null
    check("app8", "abc".safeLen70 == 3 && missing.safeLen70 == -1)
    // A literal dollar, in a normal string and in a raw one.
    check("app9", "$".length == 1 && "cost: 100$" == "cost: " + 100 + "$")
    check("app10", """raw $ end""".length == 9)
    // Consecutive `in <range>` when arms.
    check("app11", classify70(5) == "digit" && classify70(50) == "two")
    check("app12", classify70(500) == "three" && classify70(-1) == "neg" && classify70(5000) == "big")
    // A bad number THROWS, and is catchable both ways.
    check("app13", main70("42") == 42 && main70("nope") == -1)
    var caught = ""
    try { "nope".toInt() } catch (e: NumberFormatException) { caught = e.message ?: "" }
    check("app14", caught == "For input string: \"nope\"")
    check("app15", runCatching { "x".toDouble() }.isFailure && "nope".toIntOrNull() == null)
}

// ===== SECTION 71: four more, from a registry/DTO page of application Kotlin =====
//   - `buildList { addAll(listOf(2, 3)) }`: inside a builder lambda the implicit
//     receiver claimed EVERY unqualified name, so the global `listOf` was bound as a
//     method of the list under construction and the interpreter half aborted with
//     "unknown list method: listOf" (its receiver lookup runs before the hostGlobals
//     lookup). The compiler half consults the scope first and ran it.
//   - an unqualified read or write of the class's OWN computed or delegated property
//     inside a member: the accessor pair lives on the DESCRIPTOR (js_defprop), not on
//     the instance, so the compiler half's `this` fallback - which looked at own
//     properties only - aborted with "unknown name: c" / "assignment to unknown name:
//     m" where the interpreter half answered.
//   - a NAMED argument filling a `vararg` parameter. Kotlin's rule is that the single
//     named argument IS the array; the interpreter half bound the EMPTY list (it only
//     collected the positional tail) and the compiler half wrapped the array in another
//     array. Neither was Kotlin's answer.
//   - named arguments to an OBJECT's method: an object registered its method NAMES but
//     not their parameter lists, so `R.f(c = 9)` put 9 in the FIRST slot in the
//     compiler half while the identical method on a class was correct.
object Reg71 {
    var writes = 0
    val doubled: Int get() = writes * 2
    var tracked: Int by Delegates.observable(5) { _, _, _ -> writes++ }
    fun bump(): Int { tracked += 1; return tracked }
    fun pick(a: Int = 1, b: Int = 2, c: Int = 3): String = "$a/$b/$c"
    fun label(name: String, sep: String = "-", vararg tags: String): String =
        name + sep + tags.toList().joinToString(",")
}
class Own71 {
    var raw = 3
    val trebled: Int get() = raw * 3
    var mirrored: Int
        get() = raw
        set(v) { raw = v }
    fun readAll(): String = "" + trebled + "/" + mirrored
    fun writeThrough(v: Int) { mirrored = v }
}
fun sec71() {
    // A builder lambda may still call the ordinary global builders.
    check("reg1", buildList { add(1); addAll(listOf(2, 3)) } == listOf(1, 2, 3))
    check("reg2", buildString { append(listOf(1, 2).joinToString("-")) } == "1-2")
    check("reg3", buildMap<String, Int> { put("a", maxOf(1, 2)) }["a"] == 2)
    // An unqualified read and write of the class's own accessor properties.
    val o = Own71()
    check("reg4", o.readAll() == "9/3")
    o.writeThrough(5)
    check("reg5", o.raw == 5 && o.readAll() == "15/5")
    // The same through an object, including a delegated property.
    check("reg6", Reg71.bump() == 6 && Reg71.tracked == 6 && Reg71.writes == 1)
    check("reg7", Reg71.doubled == 2)
    // Named arguments to an object's method land in the right slots.
    check("reg8", Reg71.pick(c = 9) == "1/2/9" && Reg71.pick(2, c = 9) == "2/2/9")
    // A named argument filling a vararg parameter IS the array.
    check("reg9", Reg71.label("n", tags = arrayOf("a", "b")) == "n-a,b")
    check("reg10", Reg71.label("n", ":", "a") == "n:a")
    check("reg11", Reg71.label("n") == "n-")
}

// ===== END SECTIONS =====

fun main() {
    s01() // SECTION-CALL 01
    s02() // SECTION-CALL 02
    s03() // SECTION-CALL 03
    s04() // SECTION-CALL 04
    s05() // SECTION-CALL 05
    s06() // SECTION-CALL 06
    s07() // SECTION-CALL 07
    s08() // SECTION-CALL 08
    s09() // SECTION-CALL 09
    s10() // SECTION-CALL 10
    s11() // SECTION-CALL 11
    s12() // SECTION-CALL 12
    s13() // SECTION-CALL 13
    s14() // SECTION-CALL 14
    s15() // SECTION-CALL 15
    s16() // SECTION-CALL 16
    s17() // SECTION-CALL 17
    s18() // SECTION-CALL 18
    s19() // SECTION-CALL 19
    s20() // SECTION-CALL 20
    s21() // SECTION-CALL 21
    s22() // SECTION-CALL 22
    s23() // SECTION-CALL 23
    sec24() // SECTION-CALL 24
    sec25() // SECTION-CALL 25
    sec26() // SECTION-CALL 26
    sec27() // SECTION-CALL 27
    sec28() // SECTION-CALL 28
    sec29() // SECTION-CALL 29
    sec30() // SECTION-CALL 30
    sec31() // SECTION-CALL 31
    sec32() // SECTION-CALL 32
    sec33() // SECTION-CALL 33
    sec34() // SECTION-CALL 34
    sec35() // SECTION-CALL 35
    sec36() // SECTION-CALL 36
    sec37() // SECTION-CALL 37
    sec38() // SECTION-CALL 38
    sec39() // SECTION-CALL 39
    sec40() // SECTION-CALL 40
    sec41() // SECTION-CALL 41
    sec42() // SECTION-CALL 42
    sec43() // SECTION-CALL 43
    sec44() // SECTION-CALL 44
    sec45() // SECTION-CALL 45
    sec46() // SECTION-CALL 46
    sec47() // SECTION-CALL 47
    sec48() // SECTION-CALL 48
    sec49() // SECTION-CALL 49
    sec50() // SECTION-CALL 50
    sec51() // SECTION-CALL 51
    sec52() // SECTION-CALL 52
    sec53() // SECTION-CALL 53
    sec54() // SECTION-CALL 54
    sec55() // SECTION-CALL 55
    sec56() // SECTION-CALL 56
    sec57() // SECTION-CALL 57
    sec58() // SECTION-CALL 58
    sec59() // SECTION-CALL 59
    sec60() // SECTION-CALL 60
    sec61() // SECTION-CALL 61
    sec62() // SECTION-CALL 62
    sec63() // SECTION-CALL 63
    sec64() // SECTION-CALL 64
    sec65() // SECTION-CALL 65
    sec66() // SECTION-CALL 66
    sec67() // SECTION-CALL 67
    sec68() // SECTION-CALL 68
    sec69() // SECTION-CALL 69
    sec70() // SECTION-CALL 70
    sec71() // SECTION-CALL 71
    println("full: $checks checks, $fails failures")
    exitProcess(fails)
}
