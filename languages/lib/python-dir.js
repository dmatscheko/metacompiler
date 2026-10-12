// python-dir.js -- the fixed part of CPython 3.14's dir() of a user class or
// of its instance, for python-interpreter.abnf (include) and
// languages/lib/python-rt.metajs (import). Go twin: pyDirObject and
// pyDirUserClass in abnf/jsrt.go.
//
// type.__dir__ merges the __dict__ of every class on the resolution order:
// object's answers pydirObject, and a class statement gives each class
// __module__, __firstlineno__ and __static_attributes__ and the root user class
// __dict__ and __weakref__. The engine adds each user class's own members, the
// instance's attributes and the two annotation names.

var pydirObject = ["__class__", "__delattr__", "__dir__", "__doc__", "__eq__", "__format__", "__ge__",
                   "__getattribute__", "__getstate__", "__gt__", "__hash__", "__init__",
                   "__init_subclass__", "__le__", "__lt__", "__ne__", "__new__", "__reduce__",
                   "__reduce_ex__", "__repr__", "__setattr__", "__sizeof__", "__str__",
                   "__subclasshook__"]
var pydirUserClass = ["__dict__", "__firstlineno__", "__module__", "__static_attributes__", "__weakref__"]

// The names sorted by code unit and without repeats, as sorted(set(...)).
function pydirSorted(names) {
    var out = []
    for (var i = 0; i < names.length; i++) {
        var n = names[i]
        var j = out.length
        var dup = false
        while (j > 0 && !(out[j - 1] < n)) {
            if (out[j - 1] == n) { dup = true }
            j = j - 1
        }
        if (dup) { continue }
        out.push(n)
        for (var k = out.length - 1; k > j; k--) { out[k] = out[k - 1] }
        out[j] = n
    }
    return out
}
