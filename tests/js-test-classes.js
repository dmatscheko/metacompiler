// Self-checking test for JavaScript classes and for-in, for both the interpreter
// (js-interpreter.abnf) and the LLVM-IR compiler (js-to-llvm-ir.abnf). It counts
// failed checks and returns that count from main(); exit 0 means every check passed.
//
// Exercises: a base class with a field initializer, a constructor and instance/static
// methods; a subclass with 'extends', 'super(...)', 'super.method()' and an override;
// object creation with 'new'; 'this'; a static method; and 'for (k in obj)' over an
// object's own keys.

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// A base class: a field with an initializer, a constructor that sets instance fields,
// instance methods and a static method.
class Animal {
    alive = true;
    constructor(name, legs) {
        this.name = name;
        this.legs = legs;
    }
    describe() {
        return this.name + " has " + this.legs + " legs";
    }
    speak() {
        return "...";
    }
    static kingdom() {
        return "Animalia";
    }
}

// A subclass: extends, super(...) in the constructor, an overridden method, and a
// method that reaches the parent implementation with super.method().
class Dog extends Animal {
    constructor(name) {
        super(name, 4);
        this.sound = "Woof";
    }
    speak() {
        return this.sound;
    }
    describeLoud() {
        return super.describe() + "!";
    }
}

function testBase() {
    var a = new Animal("bird", 2);
    check(a.alive === true);                       // field initializer ran
    check(a.name === "bird");                       // constructor field
    check(a.legs === 2);
    check(a.describe() === "bird has 2 legs");      // instance method + this
    check(a.speak() === "...");
}

function testInheritance() {
    var d = new Dog("Rex");
    check(d.alive === true);                         // inherited field initializer
    check(d.name === "Rex");                         // super(...) set the base field
    check(d.legs === 4);
    check(d.sound === "Woof");                       // subclass field
    check(d.speak() === "Woof");                     // override
    check(d.describe() === "Rex has 4 legs");        // inherited method
    check(d.describeLoud() === "Rex has 4 legs!");   // super.method()
}

function testStatic() {
    check(Animal.kingdom() === "Animalia");
}

// for-in over a plain object's own keys, in insertion order.
function testForInObject() {
    var obj = {};
    obj.a = 1;
    obj.b = 2;
    obj.c = 3;
    var keys = [];
    var sum = 0;
    for (var k in obj) {
        keys.push(k);
        sum = sum + obj[k];
    }
    check(keys.length === 3);
    check(keys.join(",") === "a,b,c");
    check(sum === 6);

    // An object literal already carries its keys.
    var lit = {x: 10, y: 20};
    var litCount = 0;
    for (var kk in lit) { litCount = litCount + 1; }
    check(litCount === 2);
}

// for-in over a class instance sees only the instance's own data fields, never the
// internal __class / __keys slots and never the methods (which live on the class).
function testForInInstance() {
    var d = new Dog("Fido");
    var count = 0;
    var sawInternal = false;
    for (var k in d) {
        count = count + 1;
        if (k === "__class") { sawInternal = true; }
        if (k === "__keys") { sawInternal = true; }
    }
    check(count === 4); // alive, name, legs, sound
    check(sawInternal === false);
}

function main() {
    testBase();
    testInheritance();
    testStatic();
    testForInObject();
    testForInInstance();
    return failures;
}
