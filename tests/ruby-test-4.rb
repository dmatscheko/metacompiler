# Ruby subset: the string / array / hash methods that the interpreter always
# had but the LLVM-IR compiler only gained with js_rmcall (rt.rubyMethod). Both
# engines run this file and must agree; it counts failures and exits with the
# count, so exit 0 means full parity.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- string methods -----
check("str length", "hello".length, 5)
check("str size", "hello".size, 5)
check("str to_s", "hi".to_s, "hi")
check("str upcase", "abc".upcase, "ABC")
check("str downcase", "ABC".downcase, "abc")
check("str include? yes", "hello".include?("ell"), true)
check("str include? no", "hello".include?("z"), false)

# ----- array: length / first / last / to_a -----
check("arr length", [1, 2, 3].length, 3)
check("arr first", [10, 20, 30].first, 10)
check("arr last", [10, 20, 30].last, 30)
check("arr to_a size", [1, 2, 3].to_a.size, 3)

# ----- array: nil-on-empty (not an error) -----
check("empty first", [].first, nil)
check("empty last", [].last, nil)
check("empty pop", [].pop, nil)

# ----- array: sum / reject / each_with_index -----
check("arr sum", [1, 2, 3, 4].sum, 10)
check("arr reject sum", [1, 2, 3, 4].reject { |x| x % 2 == 0 }.sum, 4)

acc = 0
[10, 20, 30].each_with_index { |v, i| acc += v * i }
check("each_with_index", acc, 80)   # 10*0 + 20*1 + 30*2

# ----- Ruby truthiness in select/reject: 0 is TRUTHY, only nil/false are falsy -----
check("select keeps zero", [0, 1, 2].select { |x| x }.size, 3)
check("select drops nil/false", [0, nil, 5, false, 7].select { |x| x }.sum, 12)
check("reject removes truthy zero", [0, nil, false].reject { |x| x }.size, 2)

# ----- method chaining -----
check("map then select then sum", [1, 2, 3, 4].map { |x| x * 2 }.select { |x| x > 4 }.sum, 14)

# ----- hash: size / include? / has_key? / key? -----
h = {"a" => 1, "b" => 2, "c" => 3}
check("hash size", h.size, 3)
check("hash length", h.length, 3)
check("hash include?", h.include?("a"), true)
check("hash has_key?", h.has_key?("b"), true)
check("hash key? missing", h.key?("z"), false)
check("hash values sum", h.values.sum, 6)
check("hash keys size", h.keys.size, 3)

# ----- hash.each over key/value pairs (two block params) -----
vtotal = 0
h.each { |k, v| vtotal += v }
check("hash each values", vtotal, 6)

kstr = ""
h.each { |k, v| kstr += k }
check("hash each keys order", kstr, "abc")

# ----- done -----
if fails == 0
  puts "Ruby method-parity self test passed"
end
exit(fails)
