# Find log

When each activity of [master-prompt-find-tickets.md](master-prompt-find-tickets.md)
last ran, per language or component, and what it found. Only the find master edits
this file. Each run picks the cells that ran longest ago (`never` first).

Activities: **A** wrong-answer hunt · **B** consolidation audit · **C** reference-corpus
sweep · **D** guard review · **E** small open-source programs · **F** performance scan.

| Cell | Last run | Commit | Covered | Tickets / numbers |
|---|---|---|---|---|
| A python | 2026-10-31 | f54cee07 | data model (dunders, reflected ops, NotImplemented, truthiness, protocols, MRO/super, descriptors, properties, builtin subclasses, dataclasses), exceptions (chaining, context, else/finally, ExceptionGroup, ~60 messages), generators (send/throw/close, yield from), scoping (nonlocal, class scope, walrus), big ints, str/bytes repr and codecs, unpacking and call binding, builtins; earlier 2026-10-13 fda7a2b6 format/arith/slicing/str/sort/dict/generators | 14: T-941..T-954 (single-iterable map/filter eagerness is T-902) |
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
| A metajs | 2026-10-31 | 2eba3f2e | first run, oracle node 24 (vm.Script): number->string (~5700 doubles), bitwise/shift tables, Math specials, charAt/slice/substring/indexOf args (~2700 rows), array sort/slice/splice/length, typeof, exceptions/finally, switch, closures, eval order, object keys, String/Array methods, parseInt/Float, ==/ToNumber tables, goja vs -frozen | 14: T-888..T-901 |
| A regression (run 2 tickets: c, python, swift, java, go, bash, ruby T-772, harness) | 2026-10-31 | 2eba3f2e | every DONE / done part of T-753..T-801 (non-js) and the c/bash/go/java/python/gates follow-ups T-833..T-887 re-probed with neighbours: no regressions; T-768, T-769, T-869 incomplete; matrix 544/544 | 9: T-902..T-910 |
| A regression (run 2 tickets: ts/js, kotlin, ruby) | 2026-10-31 | 2eba3f2e | every DONE / done part of the ts/js tickets T-777..T-794, T-802..T-832 and the kotlin/ruby/ts follow-ups T-833..T-887 re-probed with neighbours (MEC_GC stress/poison on every agreeing native leg): no regressions; T-777, T-783, T-786, T-805, T-810, T-814, T-822, T-823, T-830 incomplete | 14: T-911..T-924 |
| A regression (run 1 tickets) | 2026-10-29 | 9bd2229e | every DONE / done-part of T-465..T-555 re-probed on HEAD with neighbours: no regressions | 4: T-776..T-779 (new neighbouring defects) |
| B kotlin | 2026-07-14 | 5a5f5ab5 | both Kotlin grammars | concepts consolidated in that commit |
| B other languages | never | | | |
| B shared runtime | never | | | |
| C all corpora | 2026-10-31 | 2eba3f2e | all 15 languages, both halves (pins match; fetched = swept except kotlin's 12 fragment files). must-parse interp / compiler: python 28/28 28/28, lua 33/33, batch 2/2, c 217/220 217/220, ruby 159/162, swift 94/100, kotlin 390/392, bash 2426/2489 2429/2489, php 2736/2753 2735/2753, js 2623/2653 2619/2653, go 2634/2682 2641/2682, csharp 2709/2752, dart 3371/3449 3369/3449, java 4332/4346 4329/4346, typescript 5287/5527 5197/5527 (honest interp ceiling ~5303: 219 files tsc flags as grammar errors). Lost since July: bash 19, php 3, go 3 (interp), js 3 (vs Aug) - all ticketed | 16: T-925..T-940 |
| D launch.json guards | 2026-10-29 | 9bd2229e | every SHOULD FAIL / SHOULD ABORT row (both halves, goja and -frozen), the test.sh matrix summary, gates.sh floors, --cross, native-full, probe.sh; sabotage-proved vacuities | 7: T-795..T-801; gates all green, every floor equals its measured count |
| E all languages | 2026-10-29 | 9bd2229e | first run: tests/programs/fetch.sh pins 40 Benchmarks Game + 36 Rosetta Code programs (python, go, java, js, lua, c, swift, ruby); ~70 four-way probes, ~45 fully agreed; tests/gates.sh on a clean checkout of d12bd958+: ALL GATES GREEN (matrix 496/496, --full 14659) | 11: T-765..T-775 |
| F all bench rows | 2026-10-31 | a88b4b42 | tests/bench.sh --draws 5 on an idle machine; no profiling this run (budget spent) | 0 tickets: no row beyond 2.5% of baseline.txt; largest moves go +1.52%, batch +0.97% (a control), ruby -4.62% (faster); csharp 4.23e9 instructions remains the largest row |
