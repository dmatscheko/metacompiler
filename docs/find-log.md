# Find log

When each activity of [master-prompt-find-tickets.md](master-prompt-find-tickets.md)
last ran, per language or component, and what it found. Only the find master edits
this file. Each run picks the cells that ran longest ago (`never` first).

Activities: **A** wrong-answer hunt · **B** consolidation audit · **C** reference-corpus
sweep · **D** guard review · **E** small open-source programs · **F** performance scan.

| Cell | Last run | Commit | Covered | Tickets / numbers |
|---|---|---|---|---|
| A python | never | | | |
| A java | never | | | |
| A js | never | | | |
| A typescript | never | | | |
| A go | never | | | |
| A swift | never | | | |
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
