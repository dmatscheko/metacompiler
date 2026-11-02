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
// class name followed by its own names. Rows "Class" and "Module" are a class or
// module OBJECT's. Row "private" holds the private Kernel
// methods every object has, which respond_to? reports only with include_all.
// Row "Enumerable" holds what an instance of a class that includes Enumerable and
// defines #each answers: MRI 2.6's Enumerable methods, run on the Array of what
// #each yields. Row "Enumerator::Lazy" is not Enumerator's: a Lazy answers its
// own one-value-at-a-time surface and only the eager Enumerable methods its
// lazyMethod forwards.
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
         "respond_to?", "tap", "then", "to_s", "yield_self"],
        ["private", "format", "lambda", "loop", "p", "print", "printf", "proc", "puts",
         "raise", "rand", "require", "sprintf"],
        ["Integer", "%", "&", "*", "**", "+", "+@", "-", "-@", "/", "<", "<<", "<=", "<=>", ">",
         ">=", ">>", "[]", "^", "abs", "abs2", "allbits?", "angle", "anybits?", "arg",
         "between?", "bit_length", "ceil", "chr", "clamp", "clone", "coerce", "conj",
         "conjugate", "denominator", "digits", "div", "divmod", "downto", "dup", "eql?",
         "equal?", "even?", "fdiv", "finite?", "floor", "frozen?", "gcd", "gcdlcm", "i", "imag",
         "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?", "is_a?",
         "kind_of?", "lcm", "magnitude", "modulo", "negative?", "next", "nil?", "nobits?",
         "nonzero?", "numerator", "odd?", "ord", "phase", "polar", "positive?", "pow", "pred",
         "public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular",
         "remainder", "round", "send", "size", "step", "succ", "times", "to_c", "to_f", "to_i",
         "to_int", "to_r", "truncate", "upto", "zero?", "|", "~"],
        ["Float", "%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=", "abs",
         "abs2", "angle", "arg", "between?", "ceil", "clamp", "clone", "coerce", "conj",
         "conjugate", "denominator", "div", "divmod", "dup", "eql?", "equal?", "fdiv",
         "finite?", "floor", "frozen?", "i", "imag", "imaginary", "infinite?", "instance_of?",
         "instance_variables", "integer?", "is_a?", "kind_of?", "magnitude", "modulo", "nan?",
         "negative?", "next_float", "nil?", "nonzero?", "numerator", "phase", "polar",
         "positive?", "prev_float", "public_send", "quo", "rationalize", "real", "real?",
         "rect", "rectangular", "remainder", "round", "send", "step", "to_c", "to_f", "to_i",
         "to_int", "to_r", "truncate", "zero?"],
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
        ["String", "%", "*", "+", "+@", "-@", "<", "<<", "<=", "<=>", ">", ">=", "[]", "[]=",
         "ascii_only?", "b", "between?", "bytes", "bytesize", "byteslice", "capitalize",
         "capitalize!", "casecmp", "casecmp?", "center", "chars", "chomp", "chomp!", "chop",
         "chop!", "chr", "clamp", "clear", "clone", "codepoints", "concat", "count", "delete",
         "delete!", "delete_prefix", "delete_prefix!", "delete_suffix", "delete_suffix!",
         "downcase", "downcase!", "dump", "dup", "each_byte", "each_char", "each_codepoint",
         "each_line", "empty?", "encode", "encoding", "end_with?", "equal?", "force_encoding",
         "freeze", "frozen?", "getbyte", "gsub", "gsub!", "hex", "include?", "index", "insert",
         "instance_of?", "instance_variables", "intern", "is_a?", "kind_of?", "length", "lines",
         "ljust", "lstrip", "lstrip!", "match", "match?", "next", "next!", "nil?", "oct", "ord",
         "partition", "prepend", "public_send", "replace", "reverse", "reverse!", "rindex",
         "rjust", "rpartition", "rstrip", "rstrip!", "scan", "scrub", "send", "setbyte", "size",
         "slice", "slice!", "split", "squeeze", "squeeze!", "start_with?", "strip", "strip!",
         "sub", "sub!", "succ", "succ!", "sum", "swapcase", "swapcase!", "to_c", "to_f", "to_i",
         "to_r", "to_str", "to_sym", "tr", "tr!", "tr_s", "tr_s!", "undump", "unicode_normalize",
         "unicode_normalized?",
         "unpack", "unpack1", "upcase", "upcase!", "upto", "valid_encoding?"],
        ["Symbol", "<", "<=", "<=>", ">", ">=", "[]", "between?", "capitalize", "clamp",
         "downcase", "empty?", "equal?", "frozen?", "id2name", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "length", "next", "nil?", "public_send",
         "send", "size", "succ", "swapcase", "to_proc", "to_sym", "upcase"],
        ["Array", "&", "*", "+", "-", "<<", "<=>", "[]", "[]=", "all?", "any?", "append",
         "assoc", "at", "bsearch", "chain", "chunk", "chunk_while", "clear", "clone", "collect",
         "collect!", "collect_concat", "combination", "compact", "compact!", "concat", "count",
         "cycle", "delete", "delete_at", "delete_if", "detect", "dig", "drop", "drop_while",
         "dup", "each", "each_cons", "each_entry", "each_index", "each_slice",
         "each_with_index", "each_with_object", "empty?", "entries", "equal?", "fetch", "fill",
         "filter", "filter!", "find", "find_all", "find_index", "first", "flat_map", "flatten",
         "flatten!", "freeze", "frozen?", "grep", "grep_v", "group_by", "include?", "index",
         "inject", "insert", "instance_of?", "instance_variables", "is_a?", "itself", "join",
         "keep_if", "kind_of?", "last", "lazy", "length", "map", "map!", "max", "max_by",
         "member?", "min", "min_by", "minmax", "minmax_by", "nil?", "none?", "one?", "pack",
         "partition", "permutation", "pop", "prepend", "product", "public_send", "push",
         "reduce", "reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex",
         "rotate", "sample", "select", "select!", "send", "shift", "shuffle", "shuffle!",
         "size", "slice", "slice!", "slice_after", "slice_before", "slice_when", "sort",
         "sort!", "sort_by", "sort_by!", "sum", "take", "take_while", "to_a", "to_h",
         "transpose", "uniq", "uniq!", "unshift", "values_at", "zip", "|"],
        ["Hash", "<", "<=", "<=>", ">", ">=", "[]", "[]=", "all?", "any?", "assoc", "chain",
         "chunk", "chunk_while", "clear", "clone", "collect", "collect_concat", "compact",
         "compare_by_identity", "compare_by_identity?", "count", "cycle", "default", "default=",
         "default_proc", "default_proc=", "delete", "delete_if", "detect", "dig", "drop",
         "drop_while", "dup", "each", "each_cons", "each_entry", "each_key", "each_pair",
         "each_slice", "each_value", "each_with_index", "each_with_object", "empty?", "entries",
         "equal?", "fetch", "fetch_values", "filter", "filter!", "find", "find_all",
         "find_index", "first", "flat_map", "freeze", "frozen?", "grep", "grep_v", "group_by",
         "has_key?", "include?", "index", "inject", "instance_of?", "instance_variables",
         "invert", "is_a?", "itself", "keep_if", "key", "key?", "keys", "kind_of?", "lazy",
         "length", "map", "max", "max_by", "member?", "merge", "merge!", "min", "min_by",
         "minmax", "minmax_by", "nil?", "none?", "one?", "partition", "public_send", "reduce",
         "rehash", "reject", "reject!", "replace", "select", "select!", "send", "shift", "size",
         "slice_after", "slice_before", "sort", "sort_by", "store", "sum", "take", "take_while",
         "to_a", "to_h", "to_proc", "transform_keys", "transform_keys!", "transform_values",
         "transform_values!", "uniq", "update", "values", "values_at", "zip"],
        ["Range", "%", "<=>", "all?", "any?", "begin", "bsearch", "chain", "chunk",
         "chunk_while", "clone", "collect", "collect_concat", "count", "cover?", "cycle",
         "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_entry", "each_slice",
         "each_with_index", "each_with_object", "end", "entries", "eql?", "equal?",
         "exclude_end?", "filter", "find", "find_all", "find_index", "first", "flat_map",
         "freeze", "frozen?", "grep", "grep_v", "group_by", "hash", "include?", "inject",
         "instance_of?", "instance_variables", "is_a?", "itself", "kind_of?", "last", "lazy",
         "map", "max", "max_by", "member?", "min", "min_by", "minmax", "minmax_by", "nil?",
         "none?", "object_id", "one?", "partition", "public_send", "reduce", "reject",
         "reverse_each", "select", "send", "size", "slice_after", "slice_before", "slice_when",
         "sort", "sort_by", "step", "sum", "take", "take_while", "to_a", "to_h", "uniq", "zip"],
        ["NilClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "to_a",
         "to_f", "to_i", "|"],
        ["TrueClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "|"],
        ["FalseClass", "&", "<=>", "^", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "nil?", "public_send", "send", "|"],
        ["Proc", "<<", "<=>", ">>", "[]", "arity", "call", "curry", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "lambda?", "nil?", "public_send",
         "send", "to_proc", "yield"],
        ["Method", "<<", "<=>", ">>", "[]", "arity", "call", "curry", "equal?", "frozen?", "instance_of?",
         "instance_variables", "is_a?", "kind_of?", "name", "nil?", "public_send", "send",
         "to_proc"],
        ["Regexp", "<=>", "=~", "equal?", "frozen?", "instance_of?", "instance_variables",
         "is_a?", "kind_of?", "match", "match?", "names", "nil?", "options", "public_send",
         "send", "source"],
        ["MatchData", "<=>", "[]", "begin", "captures", "end", "equal?", "frozen?",
         "instance_of?", "instance_variables", "is_a?", "kind_of?", "length",
         "named_captures", "names", "nil?", "post_match", "pre_match", "public_send",
         "send", "size", "to_a"],
        ["Enumerator", "<=>", "all?", "any?", "chain", "chunk", "clone", "collect", "collect_concat", "count",
         "cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_slice",
         "each_with_index", "each_with_object", "entries", "equal?", "filter", "find",
         "find_index", "first", "flat_map", "freeze", "frozen?", "grep", "grep_v", "group_by", "include?",
         "inject", "instance_of?", "instance_variables", "is_a?", "itself", "kind_of?", "lazy",
         "map", "max", "max_by", "member?", "min", "min_by", "minmax", "minmax_by", "next", "nil?", "none?", "one?",
         "partition", "peek", "public_send", "reduce", "reject", "reverse_each", "rewind",
         "select", "send", "size", "slice_after", "slice_before", "sort", "sort_by", "sum", "take", "take_while",
         "to_a", "uniq", "with_index", "with_object", "zip"],
        ["Enumerator::ArithmeticSequence", "<=>", "all?", "any?", "begin", "chain", "chunk",
         "clone", "collect", "collect_concat", "count", "cycle", "detect", "drop",
         "drop_while", "dup", "each", "each_cons", "each_slice", "each_with_index",
         "each_with_object", "end", "entries", "eql?", "equal?", "exclude_end?", "filter",
         "find", "find_index", "first", "flat_map", "freeze", "frozen?", "grep", "grep_v",
         "group_by", "include?", "inject", "instance_of?", "instance_variables", "is_a?",
         "itself", "kind_of?", "last", "lazy", "map", "max", "max_by", "member?", "min",
         "min_by", "minmax", "minmax_by", "next", "nil?", "none?", "one?", "partition",
         "peek", "public_send", "reduce", "reject", "reverse_each", "rewind", "select",
         "send", "size", "slice_after", "slice_before", "sort", "sort_by", "step", "sum",
         "take", "take_while", "to_a", "uniq", "with_index", "with_object", "zip"],
        ["Enumerator::Lazy", "<=>", "all?", "any?", "chunk_while", "clone", "collect",
         "collect_concat", "count", "detect", "drop", "drop_while", "dup", "each", "each_cons",
         "each_slice", "each_with_index", "each_with_object", "entries", "eql?", "equal?",
         "filter", "find", "find_all", "find_index", "first", "flat_map", "force", "freeze",
         "frozen?", "grep", "grep_v", "group_by", "include?", "inject", "instance_of?",
         "instance_variables", "is_a?", "itself", "kind_of?", "lazy", "map", "max", "max_by",
         "member?", "min", "min_by", "minmax", "minmax_by", "next", "nil?", "none?", "one?",
         "partition", "peek", "public_send", "reduce", "reject", "rewind", "select", "send",
         "size", "slice_when", "sort", "sort_by", "sum", "take", "take_while", "to_a", "to_h",
         "uniq", "with_index", "with_object", "zip"],
        ["Enumerable", "all?", "any?", "chain", "chunk", "chunk_while", "collect", "collect_concat", "count",
         "cycle", "detect", "drop", "drop_while", "each_cons", "each_entry", "each_slice",
         "each_with_index", "each_with_object", "entries", "filter", "find", "find_all",
         "find_index", "first", "flat_map", "grep", "grep_v", "group_by", "include?", "inject", "lazy", "map",
         "max", "max_by", "member?", "min", "min_by", "minmax", "minmax_by", "none?", "one?", "partition", "reduce",
         "reject", "reverse_each", "select", "slice_after", "slice_before", "slice_when", "sort", "sort_by", "sum", "take",
         "take_while", "to_a", "to_h", "uniq", "zip"],
        ["Class", "<", "<=", ">", ">=", "ancestors", "class_eval", "constants", "equal?",
         "frozen?", "include?", "instance_methods", "instance_of?",
         "instance_variable_get", "instance_variable_set", "instance_variables", "is_a?",
         "kind_of?", "method_defined?", "module_eval", "name", "new", "nil?",
         "private_instance_methods", "private_method_defined?",
         "public_instance_methods", "public_method_defined?", "public_send", "send",
         "singleton_methods", "superclass"],
        ["Module", "<", "<=", ">", ">=", "ancestors", "class_eval", "constants",
         "equal?", "frozen?", "include?", "instance_methods", "instance_of?",
         "instance_variable_get", "instance_variable_set", "instance_variables", "is_a?",
         "kind_of?", "method_defined?", "module_eval", "name", "nil?",
         "private_instance_methods", "private_method_defined?",
         "public_instance_methods", "public_method_defined?", "public_send", "send",
         "singleton_methods"]
    ]
}
// Module#instance_methods(false) of a builtin class: per class, the names of its
// row (or of row "*") that MRI 2.6 defines on the class itself - its
// instance_methods(false) restricted to what the engines dispatch. The Go twin's
// copy is rubyOwnNames in abnf/jsrtruby.go; abnf/ruby_respond_test.go keeps the
// two equal and each row inside its respond_to? names.
function rbOwnRows() {
    return [
        ["Integer", "%", "&", "*", "**", "+", "-", "-@", "/", "<", "<<", "<=", "<=>", "==",
         "===", ">", ">=", ">>", "[]", "^", "abs", "allbits?", "anybits?", "bit_length", "ceil",
         "chr", "coerce", "denominator", "digits", "div", "divmod", "even?", "fdiv",
         "floor", "gcd", "gcdlcm", "inspect", "integer?", "lcm", "magnitude", "modulo",
         "next", "nobits?", "numerator", "odd?", "ord", "pow", "pred", "rationalize",
         "remainder", "round", "size", "succ", "to_f", "to_i", "to_int", "to_r", "to_s",
         "truncate", "|", "~"],
        ["Float", "%", "*", "**", "+", "-", "-@", "/", "<", "<=", "<=>", "==", "===", ">",
         ">=", "abs", "angle", "arg", "ceil", "coerce", "denominator", "divmod", "eql?",
         "fdiv", "finite?", "floor", "hash", "infinite?", "inspect", "magnitude", "modulo",
         "nan?", "negative?", "next_float", "numerator", "phase", "positive?",
         "prev_float", "quo", "rationalize", "round", "to_f", "to_i", "to_int", "to_r",
         "to_s", "truncate", "zero?"],
        ["Rational", "*", "**", "+", "-", "-@", "/", "<=>", "==", "abs", "ceil", "coerce",
         "denominator", "fdiv", "floor", "hash", "inspect", "magnitude", "negative?",
         "numerator", "positive?", "quo", "rationalize", "round", "to_f", "to_i", "to_r",
         "to_s", "truncate"],
        ["Complex", "*", "+", "-", "-@", "/", "==", "abs", "abs2", "angle", "arg",
         "coerce", "conj", "conjugate", "denominator", "eql?", "fdiv", "finite?", "hash",
         "imag", "imaginary", "infinite?", "inspect", "magnitude", "numerator", "phase",
         "polar", "quo", "rationalize", "real", "real?", "rect", "rectangular", "to_c",
         "to_f", "to_i", "to_r", "to_s"],
        ["String", "%", "*", "+", "+@", "<<", "<=>", "==", "===", "[]", "ascii_only?", "b",
         "bytes", "bytesize", "byteslice", "capitalize", "casecmp", "casecmp?", "center", "chars",
         "chomp", "chomp!", "chop", "chop!", "chr", "clear", "codepoints", "concat",
         "count", "delete", "delete!", "delete_prefix", "delete_prefix!", "delete_suffix",
         "delete_suffix!", "downcase", "dump", "each_byte", "each_char", "each_codepoint",
         "each_line", "empty?", "encode", "end_with?", "force_encoding", "freeze",
         "getbyte", "hash", "hex", "include?", "index", "insert", "inspect", "intern",
         "length", "lines", "ljust", "lstrip", "lstrip!", "next", "oct", "ord",
         "partition", "prepend", "replace", "reverse", "reverse!", "rindex", "rjust",
         "rpartition", "rstrip", "rstrip!", "scrub", "setbyte", "size", "slice", "slice!",
         "split", "squeeze", "squeeze!", "start_with?", "strip", "strip!", "succ", "sum",
         "swapcase", "swapcase!", "to_c", "to_f", "to_i", "to_r", "to_s", "to_str", "to_sym",
         "tr", "tr!", "tr_s", "tr_s!", "undump", "unicode_normalize", "unicode_normalized?",
         "upcase", "upto",
         "valid_encoding?"],
        ["Symbol", "<=>", "==", "===", "[]", "capitalize", "downcase", "empty?", "id2name",
         "inspect", "length", "next", "size", "succ", "swapcase", "to_proc", "to_s",
         "to_sym", "upcase"],
        ["Array", "&", "*", "+", "-", "<<", "<=>", "==", "[]", "all?", "any?", "append",
         "assoc", "clear", "collect", "collect!", "combination", "compact", "compact!",
         "concat", "count", "cycle", "delete", "delete_at", "delete_if", "drop",
         "drop_while", "each", "each_index", "empty?", "fetch", "fill", "filter",
         "filter!", "find_index", "first", "flatten", "flatten!", "hash", "include?",
         "index", "insert", "inspect", "join", "keep_if", "last", "length", "map", "map!",
         "max", "min", "none?", "one?", "permutation", "pop", "prepend", "product", "push",
         "reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex", "rotate",
         "sample", "select", "select!", "shift", "shuffle", "shuffle!", "size", "slice",
         "sort", "sort!", "sort_by!", "sum", "take", "take_while", "to_a", "to_h", "to_s",
         "uniq", "uniq!", "unshift", "values_at", "zip", "|"],
        ["Hash", "<", "<=", "==", ">", ">=", "[]", "any?", "assoc", "clear", "default",
         "delete", "delete_if", "each", "each_key", "each_pair", "each_value", "empty?",
         "fetch", "filter", "filter!", "has_key?", "hash", "include?", "index", "inspect", "invert",
         "keep_if", "key", "key?", "keys", "length", "member?", "merge", "merge!", "reject",
         "reject!", "select", "select!", "size", "to_a", "to_h", "to_s", "transform_keys",
         "transform_values", "update", "values", "values_at"],
        ["Range", "==", "===", "begin", "cover?", "each", "end", "entries", "eql?",
         "exclude_end?", "first", "hash", "include?", "inspect", "last", "max", "member?",
         "min", "size", "step", "to_a", "to_s"],
        ["NilClass", "&", "===", "^", "inspect", "nil?", "to_a", "to_f", "to_i", "to_s",
         "|"],
        ["TrueClass", "&", "===", "^", "inspect", "to_s", "|"],
        ["FalseClass", "&", "===", "^", "inspect", "to_s", "|"],
        ["Proc", "<<", "===", ">>", "[]", "arity", "call", "curry", "hash", "inspect",
         "lambda?", "to_proc", "to_s", "yield"],
        ["Method", "<<", "==", "===", ">>", "[]", "arity", "call", "curry", "hash",
         "inspect", "name", "to_proc", "to_s"],
        ["Regexp", "==", "===", "=~", "hash", "inspect", "match", "match?", "names",
         "options", "source", "to_s"],
        ["MatchData", "==", "[]", "begin", "captures", "end", "hash", "inspect", "length",
         "named_captures", "names", "post_match", "pre_match", "size", "to_a", "to_s"],
        ["Enumerator", "each", "each_with_index", "each_with_object", "inspect", "next",
         "peek", "rewind", "size", "with_index", "with_object"],
        ["Enumerator::ArithmeticSequence", "==", "===", "begin", "each", "end", "eql?",
         "exclude_end?", "first", "hash", "inspect", "last", "size", "step"]
    ]
}
// The own names of builtin class cls (row 0 is its name), or [] for a class
// without a row.
function rbOwnRow(cls) {
    var rows = rbOwnRows()
    var i = 0
    while (i < rows.length) {
        if (rows[i][0] == cls) { return rows[i] }
        i = i + 1
    }
    return [cls]
}
// Method#owner of builtin class cls's method name: the nearest of cls and its
// builtin superclasses whose own row lists it, else Enumerable for a class that
// includes it and has no row between, else "" where the rows cannot tell (a
// class or module without a row, such as Numeric or Kernel). Go twin:
// rubyBuiltinOwner in abnf/jsrtruby.go.
function rbBuiltinOwner(cls, name) {
    var c = cls
    while (c != "") {
        var row = rbOwnRow(c)
        var i = 1
        while (i < row.length) {
            if (row[i] == name) { return c }
            i = i + 1
        }
        c = c == "Enumerator::ArithmeticSequence" ? "Enumerator" : ""
    }
    var enumerable = cls == "Array" || cls == "Hash" || cls == "Range" || cls == "Enumerator" || cls == "Enumerator::ArithmeticSequence"
    if (enumerable && rbEnumName(name)) { return "Enumerable" }
    return ""
}
// Method#arity of a builtin method, MRI 2.6's as measured for every name the
// respond_to? rows give a value class: row 0 is "Class arity", or "* arity" for
// the arity a name has in every class whose own rows do not name it, and a name
// in no row has arity 0. The Go twin's copy is rubyArityNames in
// abnf/jsrtruby.go; abnf/ruby_respond_test.go keeps the two equal.
function rbArityRows() {
    return [
        ["* -1", "[]", "ceil", "clone", "floor", "public_send", "rationalize", "round", "send",
         "truncate", "respond_to?", "count", "sum", "all?", "any?", "chain", "cycle", "detect",
         "entries", "find", "find_index", "first", "inject", "last", "max", "max_by", "min",
         "min_by", "none?", "one?", "reduce", "reverse_each", "slice_after", "slice_before", "zip",
         "with_index"],
        ["* 1", "%", "&", "*", "**", "+", "-", "/", "<", "<=", "<=>", ">", ">=", "^", "coerce",
         "div", "divmod", "eql?", "equal?", "fdiv", "instance_of?", "is_a?", "kind_of?", "quo",
         "remainder", "|", "!=", "==", "===", "<<", "delete", "include?", "drop", "each_cons",
         "each_slice", "each_with_object", "grep", "grep_v", "member?", "take", "with_object"],
        ["Integer -1", "chr", "digits", "pow", "inspect", "to_s"],
        ["Integer 1", ">>", "[]", "allbits?", "anybits?", "gcd", "gcdlcm", "lcm", "modulo", "nobits?"],
        ["Float 1", "modulo"],
        ["String -1", "byteslice", "capitalize", "center", "chomp", "chomp!", "concat", "delete", "delete!",
         "downcase", "each_line", "encode", "end_with?", "index", "lines", "ljust", "prepend",
         "rindex", "rjust", "scrub", "slice", "slice!", "split", "squeeze", "squeeze!",
         "start_with?", "swapcase", "swapcase!", "to_i", "unicode_normalize",
         "unicode_normalized?", "upcase", "upto"],
        ["String 1", "casecmp", "casecmp?", "delete_prefix", "delete_prefix!", "delete_suffix",
         "delete_suffix!", "force_encoding", "getbyte", "partition", "replace", "rpartition"],
        ["String 2", "insert", "setbyte", "tr", "tr!", "tr_s", "tr_s!"],
        ["Symbol -1", "capitalize", "downcase", "swapcase", "upcase"],
        ["Array -1", "append", "concat", "each_with_index", "fetch", "fill", "flatten", "flatten!",
         "index", "insert", "join", "permutation", "pop", "prepend", "product", "push", "rindex",
         "rotate", "sample", "shift", "shuffle", "shuffle!", "slice", "unshift", "values_at"],
        ["Array 0", "reverse_each"],
        ["Array 1", "assoc", "combination", "delete_at"],
        ["Hash -1", "default", "each_with_index", "fetch", "merge", "merge!", "update", "values_at"],
        ["Hash 1", "[]", "assoc", "has_key?", "index", "key", "key?"],
        ["Range -1", "each_with_index", "step"],
        ["Range 0", "entries"],
        ["Range 1", "cover?"],
        ["Proc -1", "call", "curry", "yield", "==="],
        ["Proc 1", ">>"],
        ["Method -1", "call", "curry", "==="],
        ["Method 1", ">>"],
        ["Regexp -1", "match", "match?"],
        ["Regexp 1", "=~"],
        ["MatchData 1", "begin", "end"],
        ["Enumerator -1", "each", "to_a"],
        ["Enumerator::ArithmeticSequence -1", "to_a"],
        ["Enumerator::Lazy -1", "chunk_while", "each", "force", "slice_when", "to_a", "to_h"]
    ]
}
// Method#arity of builtin class cls's method name. Go twin: rubyBuiltinArity in
// abnf/jsrtruby.go.
function rbBuiltinArity(cls, name) {
    var rows = rbArityRows()
    var star = 0
    var i = 0
    while (i < rows.length) {
        var key = rows[i][0]
        var sp = key.indexOf(" ")
        var c = key.substring(0, sp)
        var j = 1
        while ((c == cls || c == "*") && j < rows[i].length) {
            if (rows[i][j] == name) {
                var a = key.charCodeAt(key.length - 1) - 48
                if (key.charAt(sp + 1) == "-") { a = 0 - a }
                if (c == cls) { return a }
                star = a
            }
            j = j + 1
        }
        i = i + 1
    }
    return star
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

// The Enumerable methods of the "Enumerable" row.
function rbEnumName(name) {
    rbResponds("", name)
    return rbRespondSet["$Enumerable#" + name] === true
}
