// `::prop.isInitialized` is available ONLY for a property that is lexically accessible
// at the site - declared in the same type, in one of the outer types, or at top level in
// the same file (the Kotlin documentation, "Checking whether a lateinit var is
// initialized"). `l::a.isInitialized` reaches INTO another object, and Kotlin refuses it
// with "Backing field of 'var a: String' is not accessible at this point".
//
// This is a SCOPE question, not a type one, so a syntax-driven front end can decide it:
// at the `::name.isInitialized` site the grammar knows what the receiver was spelled as,
// and a receiver that is neither absent nor `this` / `this@Label` cannot name a
// lexically accessible property. (`Type::prop` is refused by the same rule for a second
// reason: it is a KProperty1 - an UNBOUND reference - and isInitialized is declared on
// KProperty0 alone.) Both halves must refuse it, and say so.
class Li {
    lateinit var a: String
}

fun main() {
    val l = Li()
    println(l::a.isInitialized)
}
