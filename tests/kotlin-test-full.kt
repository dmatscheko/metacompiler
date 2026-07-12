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
// 'import kotlin.system.exitProcess'), the coroutine RUNTIME (suspend syntax
// is only defined, never resumed), reflection, the collections API beyond the
// minimal calls the features file already uses, KDoc, and multiplatform
// (expect/actual).
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
    println("full: $checks checks, $fails failures")
    exitProcess(fails)
}
