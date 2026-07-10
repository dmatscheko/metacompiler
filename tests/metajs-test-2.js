/* MetaJS type pinning self test.
 * The first non-undefined, non-null value a variable holds decides its type
 * class; undefined and null stay assignable to everything. main() returns the
 * number of failed checks, so the run exits with 0 exactly when all is well. **/

var failures = 0;

function check(name, got, want) {
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

function fib(n) {
    if (n < 2) { return n; }
    return fib(n - 1) + fib(n - 2);
}

function makeCounter(start) {
    var count = start;
    return function() { count += 1; return count; };
}

function main() {
    // Numbers stay numbers (integer or float does not matter: one class).
    var n = 1;
    n = 2.5;
    n = -7;
    check("number stays number", n, -7);
    var h = 0xff;
    h = 0x100;                      // a hex literal is a plain number: OK
    check("hex stays number", h, 256);

    // Strings stay strings.
    var s = "a";
    s += "b";
    s = "renewed";
    check("string stays string", s, "renewed");

    // Booleans stay booleans.
    var b = true;
    b = 1 > 2;
    check("boolean stays boolean", b, false);

    // A declaration without a value stays untyped until the first real value.
    var late;
    check("undefined start", late, undefined);
    late = "now a string";
    late = "still a string";
    check("late typing", late, "still a string");

    // Assigning undefined is allowed and keeps the pinned type.
    var t = 5;
    t = undefined;
    t = 7;
    check("undefined does not unpin", t, 7);

    // Arrays and objects share the object class (like typeof); null assigns freely.
    var o = {a: 1};
    o = [1, 2, 3];
    o = null;
    o = {back: true};
    check("object class", o.back, true);

    // Functions stay functions.
    var f = function(x) { return x * 2; };
    f = fib;
    check("function stays function", f(10), 55);

    // Parameters are fresh variables per call: different types per call are fine.
    function id(x) { return x; }
    check("param int", id(4), 4);
    check("param str", id("four"), "four");

    // The pinning follows the one variable, also through closures.
    var c1 = makeCounter(10);
    c1();
    check("closure counter", c1(), 12);

    // Members are not variables: object fields may change their type freely.
    var box = {v: 1};
    box.v = "free";
    check("members are free", box.v, "free");

    // anytype declares a variable that is exempt from pinning (starts undefined).
    var any = anytype;
    check("anytype starts undefined", any, undefined);
    any = 1;
    any = "one";
    any = [1, 2];
    check("anytype changes type", any.length, 2);
    var any = 7; // A redeclaration with a value pins normally again.
    any = 8;
    check("anytype redeclared repins", any, 8);

    // The normal language still works.
    var sum = 0;
    for (var i = 1; i <= 10; i++) { sum += i; }
    check("for", sum, 55);
    check("ternary", sum == 55 ? "y" : "n", "y");
    check("fib", fib(12), 144);
    var arr = [3, 2, 1];
    arr.push(0);
    check("arrays", arr.length + arr[0], 7);

    // switch keeps the pinned types happy (the subject stays a number).
    var sw = 0;
    switch (arr.length) {
    case 3:
        sw = 3;
        break;
    case 4:                             // falls through into default
        sw = 4;
    default:
        sw += 100;
    }
    check("switch", sw, 104);

    // Prefix ++/-- pin and keep the number type; the value is the new one.
    var pn = 7;
    check("pre inc", ++pn, 8);
    check("pre dec", --pn, 7);
    check("post after pre", pn++, 7);
    check("effects", pn, 8);

    if (failures == 0) { println("MetaJS pinning self test passed"); }
    return failures;
}
