# Multi-file Ruby test: class Vec and def gcd live in tests/imports/geomlib.rb, found via
# the -i include root (mec -i tests/imports ...). Ruby's require is an ordinary method
# call, so the grammar intercepts require_relative / require with a string-literal path and
# loads the file at parse time; its top-level class and def register globally (flat), so
# this file can use Vec and gcd directly. The interpreter (ruby-interpreter.abnf) and the
# LLVM-IR compiler (ruby-to-llvm-ir.abnf) run the same file and must agree; the program
# counts failures and ends with exit(fails), so the run exits 0 exactly when all pass.
require_relative 'geomlib'

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

a = Vec.new(3, 4)
b = Vec.new(2, -1)
check("imported dot", a.dot(b), 2)
check("imported scale then dot", a.scale(2).dot(b), 4)
check("imported getter x", a.x, 3)
check("imported getter y", a.y, 4)
check("imported def gcd", gcd(48, 36), 12)

if fails == 0
  puts "ruby multifile test passed"
end
exit(fails)
