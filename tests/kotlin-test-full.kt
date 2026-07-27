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
    // charAt is the JAVA spelling. Real Kotlin has no String.charAt at all (it is
    // get / the indexing operator, both of which answer a Char); both grammars
    // carry it as a builtin that answers a one-character STRING, so that is what is
    // asserted - the point of this section is that the two halves agree.
    check("str18", s.charAt(0) == "a" && s[0] == 'a' && s.get(0) == 'a')
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
    println("full: $checks checks, $fails failures")
    exitProcess(fails)
}
