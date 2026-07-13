// A project module imported by tests/swift-test-multifile.swift (via -i tests/imports).
// 'import Geom' maps to this file (Geom.swift); its struct and global function register
// in the shared scope like the main file's declarations (flat registration).

struct Vec {
    var x: Int
    var y: Int

    init(x: Int, y: Int) {
        self.x = x
        self.y = y
    }

    // An instance method using implicit and explicit self, returning an Int.
    func dot(_ o: Vec) -> Int {
        return self.x * o.x + y * o.y
    }

    // An instance method that constructs and returns another Vec of the same type.
    func scaled(by k: Int) -> Vec {
        return Vec(x: x * k, y: y * k)
    }
}

// A global function that lives in the imported module and takes the imported type.
func lengthSq(_ v: Vec) -> Int {
    return v.x * v.x + v.y * v.y
}
