# ABNF (annotated) — VSCode syntax highlighting

Highlighting for the metacompiler's annotated ABNF dialect (EBNF with parser
commands, char-set operators, and embedded JavaScript). Applies to `.abnf` files.

## What it colors

| Construct | Example | Scope |
|---|---|---|
| Line / block comments | `// ...`, `/* ... */` | `comment.*` |
| Rule being defined | `Alternative =`, `Term <~~…~~> =` | `entity.name.function.rule` |
| Rule references | `Sequence`, `Token` | `variable.other.rule-ref` |
| Commands | `:whitespace(...)`, `:startRule(S)`, `:title(...)` | `keyword.control.command` |
| Tokens with escapes | `"a"`, `'~~'`, `"\xc3"`, `"ä"`, `"say \"hi\""` | `string.quoted.*` |
| Rune / byte ranges | `"a"..."z"`, `"\x20"..b"\x7e"` | `keyword.operator.range.*` |
| Char-set family | `@`, `@+`, `@b`, `@b+`, `!@`, `!@+`, `!@b`, `!@b+` | `keyword.operator.charset` |
| Negative lookahead | `!"x"` | `keyword.operator.lookahead.negative` |
| EBNF operators | `\|` `=` `;` `( )` `[ ]` `{ }` | `keyword.operator.*` / `punctuation.*` |
| Numbers (counted reps) | `3...5 ( X )` | `constant.numeric.integer` |
| **Embedded JS in tags** | `<~~ push(pop()) ~~>` | full `source.js` grammar |
| **Embedded JS code blocks** | `:startScript(~~ … ~~)` | full `source.js` grammar |
| Identifier tags | `< Name, Token >` | `variable.parameter.tag` |

JavaScript inside `<~~ … ~~>` tags and `~~ … ~~` code blocks is handed to
VSCode's built-in JavaScript grammar (via `meta.embedded.block.js`), so it gets
real JS coloring — and, because the embedded language is mapped to `javascript`,
JS bracket matching and comment toggling inside those regions too.

## Install (local dev)

Symlink or copy the folder into your VSCode extensions dir and reload:

```bash
ln -s "$PWD/editor/vscode-abnf" ~/.vscode/extensions/abnf-annotated-0.1.0
```

Then run **Developer: Reload Window** in VSCode. Open any `.abnf` file.

To iterate on the grammar, use **Developer: Inspect Editor Tokens and Scopes**
to see the scope under the cursor.

## Troubleshooting

**Changed a file association (or anything in `contributes`) and it has no
effect — even after restarting VSCode?** Bump the `"version"` in
`package.json`. VSCode caches each extension's `contributes` block keyed by
*id + version*, and on startup it trusts that cache instead of re-reading the
manifest, so a same-version edit stays invisible across full restarts. Changing
the version invalidates the cache and forces a re-scan. (Clearing the extension
cache works too, but a version bump is easier.)

Edits to `syntaxes/abnf.tmLanguage.json` or `language-configuration.json` are
*not* cached this way — those just need **Developer: Reload Window**. The rule
of thumb: touch `contributes` → bump the version; touch the grammar → reload.

## Package a `.vsix` (optional)

```bash
npx @vscode/vsce package
```
