# Contributing

## Commit message style

Write every commit message as a **single line**: a concise summary with **no body**
and **no trailers** (no `Co-Authored-By`, no `Signed-off-by`).

- Start with a capitalized verb in the imperative — *Add, Fix, Update, Refactor,
  Rename, Clarify, Remove, Bring back, Document…*
- Say **what** changed, plus a touch of **why** when it isn't obvious. One line only,
  even if it runs a little long.
- No trailing period.
- One logical change per commit. Tightly-related edits can share a line, separated
  by `;` or `,`.
- Refer to things by their real names — flags (`-render`), identifiers (`abnf.*`),
  files (`README.md`).

Examples:

```
Refactor CLI flag comments
Clarify the -render help: reads the -trace F file, writes DOT to stdout
Rename the "ToString functions" README heading to "Serializer functions"
Bring back per-stage tag slots as -slotN <v> in the positional CLI
Document every entry of the a-grammar API (abnf.* map + constants)
```

> Note for AI assistants: this **overrides** the default of appending a
> `Co-Authored-By: Claude …` trailer — omit it.
