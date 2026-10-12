# Find log

When each activity of [master-prompt-find-tickets.md](master-prompt-find-tickets.md)
last ran, per language or component, and what it found. Only the find master edits
this file. Each run picks the cells that ran longest ago (`never` first).

Activities: **A** wrong-answer hunt · **B** consolidation audit · **C** reference-corpus
sweep · **D** guard review · **E** small open-source programs · **F** performance scan.

| Cell | Last run | Commit | Covered | Tickets / numbers |
|---|---|---|---|---|
| A python | never | | | |
| A java | 2026-10-13 | fda7a2b6 | int/long arithmetic and casts, Double/Float text, String.format, String/StringBuilder, HashMap order, collections, switch, exceptions, null/NPE, ternary typing, eval order | 12: T-494..T-504, T-506 (+T-505 CORE) |
| A js | 2026-10-13 | fda7a2b6 | number formatting, parseInt/Float, Math, bitwise, Array/String methods, own-key order, regex replace, JSON, Map/Set, labels/try-finally, generators, coercion tables, eval order | 16: T-478..T-493 |
| A typescript | never | | | |
| A go | 2026-10-13 | fda7a2b6 | sized ints, fmt verbs, strings pkg, panics/recover, slices/append/copy, maps, closures, channels, conversions, min/max/clear | 13: T-465..T-477 |
| A swift | 2026-10-13 | fda7a2b6 | integers/shifts/conversions, Double rounding and text, graphemes, value semantics, for-in, switch patterns, Character properties, literal type adoption | 10: T-507..T-516 |
| A ruby | never | | | |
| A lua | never | | | |
| A bash | never | | | |
| A c | never | | | |
| A kotlin | never | | | |
| A php | never | | | |
| A csharp | never | | | |
| A dart | never | | | |
| A metajs | never | | | |
| B kotlin | 2026-07-14 | 5a5f5ab5 | both Kotlin grammars | concepts consolidated in that commit |
| B other languages | never | | | |
| B shared runtime | never | | | |
| C all corpora | about 2026-07 | | per-language corpus sweep | date approximate (from commit history); only PHP was re-measured since, in October |
| D launch.json guards | never | | | |
| E all languages | never | | | |
| F all bench rows | 2026-10-12 | 8bbba503 | final bench of the work run | all rows within +-0.11% of the baseline |
