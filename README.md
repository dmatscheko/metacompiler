# ParserParserCompiler

This is the attempt to create
* a parser that parses an annotated EBNF
* and (at runtime!) creates a second parser with the EBNF
* plus a compiler with the annotations in the EBNF.

That new runtime generated parser plus compiler should allow to compile an arbitrary new language, specified in the EBNF and its annotations.

## What is an annotated EBNF?

* The EBNF stores the grammar of the new language.
* The annotations in the EBNF store the semantic of the new language.

## Example of an annotated EBNF

```
{
program     <"" '{{"{"}}{{.childCode}}}'>                                                                = [ title ] [ tag ] "{" [ production ] { production <"" ", {{.childCode}}"> } "}" [ tag ] [ comment ] .
production  <"" '{"{{.vars.name}}", {{ident .vars.name}}, {{.childCode}}}'>                              = name <"name"> [ tag ] "=" [ expression ] ( "." | ";" ) .
expression  <"" '{{if (eq .setVars.or true)}}{"OR", {{.childCode}}}{{else}}{{.childCode}}{{end}}'>       = alternative <"" '{{set "or" false}}{{.childCode}}'> { "|" alternative <"" '{{set "or" true}}, {{.childCode}}'> } .
alternative <"" '{{"{"}}{{.childCode}}{{"}"}}'>                                                          = term <"" '{{.childCode}}'> { term <"" ', {{.childCode}}'> } .
term        = ( name | text [ "..." text ] | group | option | repetition | skipspaces ) [ tag ] .
group       <"" '{{"{"}}{{.childCode}}{{"}"}}'>             = "(" expression ")" .
option      <"" '{"OPTIONAL", {{.childCode}}}'>             = "[" expression "]" .
repetition  <"" '{"REPEAT", {{.childCode}}}'>               = "{" expression "}" .
skipspaces  = "+" <"" '{"SKIPSPACES", true}'> | "-" <"" '{"SKIPSPACES", false}'> .
title = text .
comment = text .
name        <"" '{"IDENT", "{{.childStr}}", {{ident .childStr}}}'>                      = ( small | caps ) - { small | caps | digit | "_" } + .
text        <"" '{"TERMINAL", {{.childStr}}}'>                                          = dquotetext | squotetext .
dquotetext = '"' - { small | caps | digit | special } '"' + .
squotetext = "'" - { small | caps | digit | special } "'" + .
tag  = "<" text text ">" .
digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" .
small = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" .
caps = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" .
special = "_" | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\\"" | "\\n" | "\\t" | " " | "|" | "%" | "$" | "&" | "'" | "#" | "~" | "@" .
}
```