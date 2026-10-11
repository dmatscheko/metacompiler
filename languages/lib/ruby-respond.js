// ruby-respond.js -- Object#respond_to? for a value that is not a user object.
//
// The answer is the ENGINES' implemented surface: a name is listed for a class
// only when a call of it DISPATCHES in all three ruby engines (the interpreter
// half, the Go twin and layer 2) and MRI's class has it too, so respond_to? never
// promises a call that would abort and never claims a method MRI lacks. The
// dispatch is if-chains in each engine, so the list is measured, not derived: a
// name missing here that an engine does answer only makes respond_to? say false.
//
// Row 0 ("*") holds the names every builtin class answers; each other row is a
// class name followed by its own names. Row "private" holds the private Kernel
// methods every object has, which respond_to? reports only with include_all.
//
// Included by ruby-interpreter.abnf (include("lib/ruby-respond.js")) and
// imported by lib/ruby-rt.metajs. The Go twin's copy is rubyRespondNames in
// abnf/jsrtruby.go; abnf/ruby_respond_test.go keeps the two equal. Written in the
// restricted subset regex.js describes (no for..in, no split).
// Built on the first respond_to?, so a program that never asks pays nothing at
// start-up.
function rbRespondRows() {
    return [
        ["*", "!", "!=", "==", "===", "class", "hash", "inspect", "itself", "object_id",
         "respond_to?", "tap", "to_s"],
        ["private", "format", "lambda", "loop", "p", "print", "printf", "proc", "puts",
         "raise", "rand", "require", "sprintf"],
        ["Integer", "%", "&", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">",
         ">=", "[]", "^", "abs", "abs2", "allbits?", "angle", "anybits?", "arg",
         "bit_length", "ceil", "chr", "clone", "coerce", "conj", "conjugate", "denominator",
         "digits", "div", "divmod", "dup", "eql?", "equal?", "even?", "fdiv", "finite?",
         "floor", "frozen?", "gcd", "gcdlcm", "i", "imag", "imaginary", "infinite?",
         "instance_of?", "instance_variables", "integer?", "is_a?", "kind_of?", "lcm",
         "magnitude", "modulo", "negative?", "next", "nil?", "nobits?", "nonzero?",
         "numerator", "odd?", "ord", "phase", "polar", "positive?", "pow", "pred",
         "public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular",
         "remainder", "round", "send", "size", "succ", "to_c", "to_f", "to_i", "to_int",
         "to_r", "truncate", "zero?", "|"],
        ["Float", "%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=",
         "abs", "abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate",
         "denominator", "div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?",
         "floor", "frozen?", "i", "imag", "imaginary", "infinite?", "instance_of?",
         "instance_variables", "integer?", "is_a?", "kind_of?", "magnitude", "modulo",
         "nan?", "negative?", "next_float", "nil?", "nonzero?", "numerator", "phase",
         "polar", "positive?", "prev_float", "public_send", "quo", "rationalize", "real",
         "real?", "rect", "rectangular", "remainder", "round", "send", "to_c", "to_f",
         "to_i", "to_int", "to_r", "truncate", "zero?"],
        ["Rational", "%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=",
         "abs", "abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate",
         "denominator", "div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?",
         "floor", "frozen?", "i", "imag", "imaginary", "infinite?", "instance_of?",
         "instance_variables", "integer?", "is_a?", "kind_of?", "magnitude", "negative?",
         "nil?", "nonzero?", "numerator", "phase", "polar", "positive?", "public_send",
         "quo", "rationalize", "real", "real?", "rect", "rectangular", "remainder", "round",
         "send", "to_c", "to_f", "to_i", "to_int", "to_r", "truncate", "zero?"],
        ["Complex", "*", "+", "+@", "-", "-@", "/", "abs", "abs2", "angle", "arg", "clone",
         "coerce", "conj", "conjugate", "denominator", "dup", "eql?", "equal?", "fdiv",
         "finite?", "frozen?", "imag", "imaginary", "infinite?", "instance_of?",
         "instance_variables", "integer?", "is_a?", "kind_of?", "magnitude", "nil?",
         "nonzero?", "numerator", "phase", "polar", "public_send", "quo", "rationalize",
         "real", "real?", "rect", "rectangular", "send", "to_c", "to_f", "to_i", "to_int",
         "to_r", "zero?"],
        ["String", "%", "*", "+", "+@", "<", "<<", "<=", "<=>", ">", ">=", "[]",
         "ascii_only?", "b", "bytes", "bytesize", "capitalize", "casecmp", "casecmp?",
         "center", "chars", "chomp", "chomp!", "chop", "chop!", "chr", "clear", "clone",
         "codepoints", "concat", "count", "delete", "delete!", "delete_prefix",
         "delete_prefix!", "delete_suffix", "delete_suffix!", "downcase", "dump", "dup",
         "each_byte", "each_char", "each_codepoint", "each_line", "empty?", "encode",
         "end_with?", "equal?", "force_encoding", "freeze", "frozen?", "getbyte", "hex",
         "include?", "index", "insert", "instance_of?", "instance_variables", "intern",
         "is_a?", "kind_of?", "length", "lines", "ljust", "lstrip", "lstrip!", "next",
         "nil?", "oct", "ord", "partition", "prepend", "public_send", "replace", "reverse",
         "reverse!", "rindex", "rjust", "rpartition", "rstrip", "rstrip!", "scrub", "send",
         "size", "slice", "slice!", "split", "squeeze", "squeeze!", "start_with?", "strip",
         "strip!", "succ", "sum", "swapcase", "swapcase!", "to_f", "to_i", "to_str",
         "to_sym", "tr", "tr!", "tr_s", "tr_s!", "unicode_normalize", "unicode_normalized?",
         "upcase", "upto", "valid_encoding?"],
        ["Symbol", "<", "<=", "<=>", ">", ">=", "[]", "capitalize", "downcase", "empty?",
         "equal?", "frozen?", "id2name", "instance_of?", "instance_variables", "is_a?",
         "kind_of?", "length", "name", "next", "nil?", "public_send", "send", "size",
         "succ", "swapcase", "to_proc", "to_sym", "upcase"],
        ["Array", "&", "*", "+", "-", "<<", "<=>", "[]", "all?", "any?", "append", "assoc",
         "clear", "clone", "collect", "collect!", "collect_concat", "combination",
         "compact", "compact!", "concat", "count", "cycle", "delete", "delete_at",
         "delete_if", "detect", "drop", "drop_while", "dup", "each", "each_cons",
         "each_index", "each_slice", "each_with_index", "each_with_object", "empty?",
         "entries", "equal?", "fetch", "fill", "filter", "filter!", "find", "find_index",
         "first", "flat_map", "flatten", "flatten!", "freeze", "frozen?", "group_by",
         "include?", "index", "inject", "insert", "instance_of?", "instance_variables",
         "is_a?", "itself", "join", "keep_if", "kind_of?", "last", "length", "map", "map!",
         "max", "max_by", "min", "min_by", "minmax", "nil?", "none?", "one?", "partition",
         "permutation", "pop", "prepend", "product", "public_send", "push", "reduce",
         "reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex", "rotate",
         "sample", "select", "select!", "send", "shift", "shuffle", "shuffle!", "size",
         "slice", "sort", "sort!", "sort_by", "sort_by!", "sum", "take", "take_while",
         "tally", "to_a", "uniq", "uniq!", "unshift", "values_at", "zip", "|"],
        ["Hash", "<", "<=", "<=>", ">", ">=", "[]", "all?", "any?", "assoc", "clear",
         "clone", "collect", "collect_concat", "count", "default", "delete", "delete_if",
         "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_key",
         "each_pair", "each_slice", "each_value", "each_with_index", "each_with_object",
         "empty?", "entries", "equal?", "fetch", "filter", "filter!", "find", "find_index",
         "first", "flat_map", "freeze", "frozen?", "group_by", "has_key?", "include?",
         "inject", "instance_of?", "instance_variables", "invert", "is_a?", "itself",
         "keep_if", "key", "key?", "keys", "kind_of?", "length", "map", "max", "max_by",
         "merge", "merge!", "min", "min_by", "minmax", "nil?", "none?", "one?", "partition",
         "public_send", "reduce", "reject", "reject!", "select", "select!", "send", "size",
         "sort", "sort_by", "sum", "take", "take_while", "tally", "to_a", "to_h",
         "transform_keys", "transform_values", "uniq", "update", "values", "values_at",
         "zip"],
        ["Range", "<=>", "all?", "any?", "begin", "clone", "collect", "collect_concat",
         "count", "cover?", "cycle", "detect", "drop", "drop_while", "dup", "each",
         "each_cons", "each_slice", "each_with_index", "each_with_object", "end", "entries",
         "eql?", "equal?", "exclude_end?", "filter", "find", "find_index", "first",
         "flat_map", "freeze", "frozen?", "group_by", "hash", "include?", "inject",
         "instance_of?", "instance_variables", "is_a?", "itself", "kind_of?", "last", "map",
         "max", "max_by", "member?", "min", "min_by", "minmax", "nil?", "none?",
         "object_id", "one?", "partition", "public_send", "reduce", "reject",
         "reverse_each", "select", "send", "size", "sort", "sort_by", "step", "sum", "take",
         "take_while", "tally", "to_a", "uniq", "zip"],
        ["NilClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "to_a",
         "to_f", "to_i", "|"],
        ["TrueClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "|"],
        ["FalseClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "|"],
        ["Proc", "<=>", "[]", "arity", "call", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "lambda?", "nil?", "public_send",
         "send", "to_proc", "yield"],
        ["Method", "<=>", "[]", "arity", "call", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "name", "nil?", "public_send", "send",
         "to_proc"],
        ["Regexp", "<=>", "=~", "equal?", "frozen?", "instance_of?", "instance_variables",
         "is_a?", "kind_of?", "match", "match?", "names", "nil?", "options", "public_send",
         "send", "source"],
        ["MatchData", "<=>", "[]", "begin", "captures", "end", "equal?", "frozen?",
         "instance_of?", "instance_variables", "is_a?", "kind_of?", "length",
         "named_captures", "names", "nil?", "post_match", "pre_match", "public_send",
         "send", "size", "to_a"],
        ["Enumerator", "<=>", "all?", "any?", "clone", "collect", "collect_concat", "count",
         "cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_slice",
         "each_with_index", "each_with_object", "entries", "equal?", "filter", "find",
         "find_index", "first", "flat_map", "freeze", "frozen?", "group_by", "include?",
         "inject", "instance_of?", "instance_variables", "is_a?", "itself", "kind_of?",
         "map", "max", "max_by", "min", "min_by", "minmax", "next", "nil?", "none?", "one?",
         "partition", "peek", "public_send", "reduce", "reject", "reverse_each", "rewind",
         "select", "send", "size", "sort", "sort_by", "sum", "take", "take_while", "tally",
         "to_a", "uniq", "with_index", "with_object", "zip"]
    ]
}
var rbRespondSet = {}
var rbRespondReady = false

// Does a value of builtin class `cls` answer `name`? The "$" prefix keeps a name
// like "toString" out of the host's own object members.
function rbResponds(cls, name) {
    if (!rbRespondReady) {
        var rows = rbRespondRows()
        var i = 0
        while (i < rows.length) {
            var row = rows[i]
            var j = 1
            while (j < row.length) {
                rbRespondSet["$" + row[0] + "#" + row[j]] = true
                j = j + 1
            }
            i = i + 1
        }
        rbRespondReady = true
    }
    return rbRespondSet["$*#" + name] === true || rbRespondSet["$" + cls + "#" + name] === true
}

// The private Kernel methods of the "private" row (rbResponds has built the set).
function rbRespondsPrivate(name) {
    rbResponds("", name)
    return rbRespondSet["$private#" + name] === true
}
