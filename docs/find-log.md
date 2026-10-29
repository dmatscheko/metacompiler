# Find log

When each activity of [master-prompt-find-tickets.md](master-prompt-find-tickets.md)
last ran, per language or component, and what it found. Only the find master edits
this file. Each run picks the cells that ran longest ago (`never` first).

Activities: **A** wrong-answer hunt · **B** consolidation audit · **C** reference-corpus
sweep · **D** guard review · **E** small open-source programs · **F** performance scan.

| Cell | Last run | Commit | Covered | Tickets / numbers |
|---|---|---|---|---|
| A python | 2026-10-13 | fda7a2b6 | format mini-language (560 rows), %-formatting, f-strings, int/float arithmetic, divmod/pow, int()/float() parsing, slicing (648 combos), str methods (694 rows), sorted/min/max, dict/set mutation, generators, iterators, eval order | 15: T-541..T-555 |
| A java | 2026-10-13 | fda7a2b6 | int/long arithmetic and casts, Double/Float text, String.format, String/StringBuilder, HashMap order, collections, switch, exceptions, null/NPE, ternary typing, eval order | 12: T-494..T-504, T-506 (+T-505 CORE) |
| A js | 2026-10-13 | fda7a2b6 | number formatting, parseInt/Float, Math, bitwise, Array/String methods, own-key order, regex replace, JSON, Map/Set, labels/try-finally, generators, coercion tables, eval order | 16: T-478..T-493 |
| A typescript | 2026-10-29 | 9bd2229e | classes (fields, accessors, #private, statics, super), enums, namespaces and merging, parameters/destructuring, ?. and ??, decorators, regex flags, promises/async, ToPrimitive, computed keys, ambient declarations; re-check of the js fixes T-478..T-493 on TS (all green) | 15: T-780..T-794 (+TS evidence on T-779) |
| A go | 2026-10-13 | fda7a2b6 | sized ints, fmt verbs, strings pkg, panics/recover, slices/append/copy, maps, closures, channels, conversions, min/max/clear | 13: T-465..T-477 |
| A swift | 2026-10-13 | fda7a2b6 | integers/shifts/conversions, Double rounding and text, graphemes, value semantics, for-in, switch patterns, Character properties, literal type adoption | 10: T-507..T-516 |
| A ruby | 2026-10-29 | 9bd2229e | Float/Integer arithmetic and text, String#% (36 flag x 17 values), String methods (split, [], index, succ, sub/gsub, tr, center), Array/Hash/Range methods, Comparable, blocks/procs/lambdas, exceptions, eval order (oracle ruby 2.6) | 17: T-816..T-832 |
| A lua | 2026-10-13 | fda7a2b6 | int/float arithmetic and printing, string.format, patterns, string/table library, pcall/error, goto, numeric/generic for, closures, multiple assignment, int-float comparison, recursion | 9: T-532..T-540 (+lua evidence on T-505) |
| A bash | 2026-10-13 | fda7a2b6 | arithmetic, parameter expansion, word splitting/IFS, indexed and sparse arrays, [[ ]] tests and =~, case, return codes, printf, command-substitution depth | 15: T-517..T-529, T-531 (+T-530 CORE, probe.sh) |
| A c | 2026-10-29 | 9bd2229e | integer promotions and conversions, 64-bit unsigned, / % shifts, floats and -0.0, string.h/stdlib, structs/unions/bit-fields, function pointers, control flow, sizeof/_Generic, ?: typing, string and char literals, preprocessor | 12: T-753..T-764 |
| A kotlin | 2026-10-29 | ae9a4e63 | Double/Float text and comparison, Int/Long arithmetic and shifts, Char, String and collection stdlib (vs the kotlin-stdlib jar), ranges, operator conventions, null-safety, casts, exceptions and messages, eval order (oracle: java + stdlib jar, no kotlinc) | 14: T-802..T-815 |
| A php | never | | | |
| A csharp | never | | | |
| A dart | never | | | |
| A metajs | never | | | |
| A regression (run 1 tickets) | 2026-10-29 | 9bd2229e | every DONE / done-part of T-465..T-555 re-probed on HEAD with neighbours: no regressions | 4: T-776..T-779 (new neighbouring defects) |
| B kotlin | 2026-07-14 | 5a5f5ab5 | both Kotlin grammars | concepts consolidated in that commit |
| B other languages | never | | | |
| B shared runtime | never | | | |
| C all corpora | about 2026-07 | | per-language corpus sweep | date approximate (from commit history); only PHP was re-measured since, in October |
| D launch.json guards | 2026-10-29 | 9bd2229e | every SHOULD FAIL / SHOULD ABORT row (both halves, goja and -frozen), the test.sh matrix summary, gates.sh floors, --cross, native-full, probe.sh; sabotage-proved vacuities | 7: T-795..T-801; gates all green, every floor equals its measured count |
| E all languages | 2026-10-29 | 9bd2229e | first run: tests/programs/fetch.sh pins 40 Benchmarks Game + 36 Rosetta Code programs (python, go, java, js, lua, c, swift, ruby); ~70 four-way probes, ~45 fully agreed; tests/gates.sh on a clean checkout of d12bd958+: ALL GATES GREEN (matrix 496/496, --full 14659) | 11: T-765..T-775 |
| F all bench rows | 2026-10-12 | 8bbba503 | final bench of the work run | all rows within +-0.11% of the baseline |
