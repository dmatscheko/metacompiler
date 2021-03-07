# ParserParserCompiler

This is
* a parser that parses an annotated EBNF
* and (at runtime) creates a second parser with the EBNF
* plus a compiler with the annotations in the EBNF.

That new runtime generated parser plus compiler should allow to compile an arbitrary new language, specified in the EBNF and its annotations.

## What is an annotated EBNF?

* The EBNF stores the grammar of e.g. a new language.
* The annotations in the EBNF store the semantic of the new language.

## Examples

### Example of an annotated EBNF
<details>
  <summary>Click to expand!</summary>

```
`"aEBNF of aEBNF as text"
<~~
var names = [];
function getNameIdx(name) {
	var pos = names.indexOf(name);
	if (pos != -1) { return pos };
	return names.push(name)-1;
}
~~>

{

program          = [ title <~~ upstream.str += ', \\n' ~~> ] programtag "{" <~~ upstream.str = '\\n{\\n\\n' ~~> { production } "}" <~~ upstream.str = '\\n}, \\n\\n' ~~> programtag start [ comment <~~ upstream.str = ', \\n'+upstream.str+'\\n' ~~> ] ;
programtag       = [ tag <~~ upstream.str = '{"TAG", '+upstream.str+'}, \\n' ~~> ] ;
production       <~~ if (upstream.productionTag != undefined) { upstream.str = '{"TAG", '+upstream.productionTag+', '+upstream.str+'}' }; upstream.str += ', \\n' ~~>       
					= name <~~ upstream.str = '{"'+upstream.str+'", '+getNameIdx(upstream.str)+', ' ~~> [ tag ] <~~ upstream.productionTag = upstream.str; upstream.str = '' ~~> "=" <~~ upstream.str = '' ~~> [ expression ] ( "." | ";" ) <~~ upstream.str = '}' ~~> ;
expression       <~~ if (upstream.or) { upstream.str = '{"OR", '+upstream.str+'}' } ~~>   
					= alternative <~~ upstream.or = false ~~> { "|" <~~ upstream.str = '' ~~> alternative <~~ upstream.or = true; upstream.str = ', '+upstream.str ~~> } ;
alternative      = taggedterm { taggedterm <~~ upstream.str = ', '+upstream.str ~~> } ;

taggedterm       <~~ if (upstream.termTag != undefined) { upstream.str = '{"TAG", '+upstream.termTag+', '+upstream.str+'}' } ~~>
					= term <~~ upstream.termTag = undefined ~~> [ tag <~~ upstream.termTag = upstream.str; upstream.str = '' ~~> ] ;

term             = ( name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) ;
group            <~~ upstream.str = '{'+upstream.str+'}' ~~>                              = "(" <~~ upstream.str = '' ~~> expression ")" <~~ upstream.str = '' ~~> ;
option           <~~ upstream.str = '{"OPTIONAL", '+upstream.str+'}' ~~>                  = "[" <~~ upstream.str = '' ~~> expression "]" <~~ upstream.str = '' ~~> ;
repetition       <~~ upstream.str = '{"REPEAT", '+upstream.str+'}' ~~>                    = "{" <~~ upstream.str = '' ~~> expression "}" <~~ upstream.str = '' ~~> ;
skipspaces       = "+" <~~ upstream.str = '{"SKIPSPACES", true}' ~~> | "-" <~~ upstream.str = '{"SKIPSPACES", false}' ~~> ;

title            = text ;
start            = name <~~ upstream.str = '{"IDENT", "'+upstream.str+'", '+getNameIdx(upstream.str)+'}' ~~> ;
comment          = text ;

tag              = "<" <~~ upstream.str = '' ~~> code { "," <~~ upstream.str = '' ~~> code <~~ upstream.str = ', '+upstream.str ~~> } ">" <~~ upstream.str = '' ~~> ;

code             <~~ upstream.str = '{"TERMINAL", '+sprintf("%q",upstream.str)+'}' ~~>                  = '~~' <~~ upstream.str = '' ~~> - { [ "~" ] codeinner } '~~' <~~ upstream.str = '' ~~> + ;
codeinner        = small | caps | digit | special | "'" | '"' | "\\~" ;

name             = ( small | caps ) - { small | caps | digit | "_" } + ;

text             <~~ upstream.str = '{"TERMINAL", '+sprintf("%q",upstream.str)+'}' ~~>                  = dquotetext | squotetext ;
dquotetext       = '"' <~~ upstream.str = '' ~~> - { small | caps | digit | special | "~" | "'" | '\\"' } '"' <~~ upstream.str = '' ~~> + ;
squotetext       = "'" <~~ upstream.str = '' ~~> - { small | caps | digit | special | "~" | '"' | "\\'" } "'" <~~ upstream.str = '' ~~> + ;

digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;

}

<~~ print(upstream.str) ~~>
program
"This aEBNF contains the grammatic and semantic information for annotated EBNF.
It allows to automatically create a compiler for everything described in aEBNF (yes, that format)."`,
```
</details>

### Its output, when it gets applied on itself:
<details>
  <summary>Click to expand!</summary>

```
{"TERMINAL", "aEBNF of aEBNF as text"}, 
{"TAG", {"TERMINAL", "\n\t\tvar names = [];\n\t\tfunction getNameIdx(name) {\n\t\t  var pos = names.indexOf(name);\n\t\t  if (pos != -1) { return pos };\n\t\t  return names.push(name)-1;\n\t\t}\n\t\t"}}, 

{

{"program", 0, {"OPTIONAL", {"TAG", {"TERMINAL", " upstream.str += ', \\\\n' "}, {"IDENT", "title", 1}}}, {"IDENT", "programtag", 2}, {"TAG", {"TERMINAL", " upstream.str = '\\\\n{\\\\n\\\\n' "}, {"TERMINAL", "{"}}, {"REPEAT", {"IDENT", "production", 3}}, {"TAG", {"TERMINAL", " upstream.str = '\\\\n}, \\\\n\\\\n' "}, {"TERMINAL", "}"}}, {"IDENT", "programtag", 2}, {"IDENT", "start", 4}, {"OPTIONAL", {"TAG", {"TERMINAL", " upstream.str = ', \\\\n'+upstream.str+'\\\\n' "}, {"IDENT", "comment", 5}}}}, 
{"programtag", 2, {"OPTIONAL", {"TAG", {"TERMINAL", " upstream.str = '{\"TAG\", '+upstream.str+'}, \\\\n' "}, {"IDENT", "tag", 6}}}}, 
{"TAG", {"TERMINAL", " if (upstream.productionTag != undefined) { upstream.str = '{\"TAG\", '+upstream.productionTag+', '+upstream.str+'}' }; upstream.str += ', \\\\n' "}, {"production", 3, {"TAG", {"TERMINAL", " upstream.str = '{\"'+upstream.str+'\", '+getNameIdx(upstream.str)+', ' "}, {"IDENT", "name", 7}}, {"TAG", {"TERMINAL", " upstream.productionTag = upstream.str; upstream.str = '' "}, {"OPTIONAL", {"IDENT", "tag", 6}}}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "="}}, {"OPTIONAL", {"IDENT", "expression", 8}}, {"TAG", {"TERMINAL", " upstream.str = '}' "}, {{"OR", {"TERMINAL", "."}, {"TERMINAL", ";"}}}}}}, 
{"TAG", {"TERMINAL", " if (upstream.or) { upstream.str = '{\"OR\", '+upstream.str+'}' } "}, {"expression", 8, {"TAG", {"TERMINAL", " upstream.or = false "}, {"IDENT", "alternative", 9}}, {"REPEAT", {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "|"}}, {"TAG", {"TERMINAL", " upstream.or = true; upstream.str = ', '+upstream.str "}, {"IDENT", "alternative", 9}}}}}, 
{"alternative", 9, {"IDENT", "taggedterm", 10}, {"REPEAT", {"TAG", {"TERMINAL", " upstream.str = ', '+upstream.str "}, {"IDENT", "taggedterm", 10}}}}, 
{"TAG", {"TERMINAL", " if (upstream.termTag != undefined) { upstream.str = '{\"TAG\", '+upstream.termTag+', '+upstream.str+'}' } "}, {"taggedterm", 10, {"TAG", {"TERMINAL", " upstream.termTag = undefined "}, {"IDENT", "term", 11}}, {"OPTIONAL", {"TAG", {"TERMINAL", " upstream.termTag = upstream.str; upstream.str = '' "}, {"IDENT", "tag", 6}}}}}, 
{"term", 11, {{"OR", {"TAG", {"TERMINAL", " upstream.str = '{\"IDENT\", \"'+upstream.str+'\", '+getNameIdx(upstream.str)+'}' "}, {"IDENT", "name", 7}}, {{"IDENT", "text", 12}, {"OPTIONAL", {"TERMINAL", "..."}, {"IDENT", "text", 12}}}, {"IDENT", "group", 13}, {"IDENT", "option", 14}, {"IDENT", "repetition", 15}, {"IDENT", "skipspaces", 16}}}}, 
{"TAG", {"TERMINAL", " upstream.str = '{'+upstream.str+'}' "}, {"group", 13, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "("}}, {"IDENT", "expression", 8}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", ")"}}}}, 
{"TAG", {"TERMINAL", " upstream.str = '{\"OPTIONAL\", '+upstream.str+'}' "}, {"option", 14, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "["}}, {"IDENT", "expression", 8}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "]"}}}}, 
{"TAG", {"TERMINAL", " upstream.str = '{\"REPEAT\", '+upstream.str+'}' "}, {"repetition", 15, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "{"}}, {"IDENT", "expression", 8}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "}"}}}}, 
{"skipspaces", 16, {"OR", {"TAG", {"TERMINAL", " upstream.str = '{\"SKIPSPACES\", true}' "}, {"TERMINAL", "+"}}, {"TAG", {"TERMINAL", " upstream.str = '{\"SKIPSPACES\", false}' "}, {"TERMINAL", "-"}}}}, 
{"title", 1, {"IDENT", "text", 12}}, 
{"start", 4, {"TAG", {"TERMINAL", " upstream.str = '{\"IDENT\", \"'+upstream.str+'\", '+getNameIdx(upstream.str)+'}' "}, {"IDENT", "name", 7}}}, 
{"comment", 5, {"IDENT", "text", 12}}, 
{"tag", 6, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "<"}}, {"IDENT", "code", 17}, {"REPEAT", {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", ","}}, {"TAG", {"TERMINAL", " upstream.str = ', '+upstream.str "}, {"IDENT", "code", 17}}}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", ">"}}}, 
{"TAG", {"TERMINAL", " upstream.str = '{\"TERMINAL\", '+sprintf(\"%q\",upstream.str)+'}' "}, {"code", 17, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "~~"}}, {"SKIPSPACES", false}, {"REPEAT", {"OPTIONAL", {"TERMINAL", "~"}}, {"IDENT", "codeinner", 18}}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "~~"}}, {"SKIPSPACES", true}}}, 
{"codeinner", 18, {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "'"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\~"}}}, 
{"name", 7, {{"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}}}, {"SKIPSPACES", false}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"TERMINAL", "_"}}}, {"SKIPSPACES", true}}, 
{"TAG", {"TERMINAL", " upstream.str = '{\"TERMINAL\", '+sprintf(\"%q\",upstream.str)+'}' "}, {"text", 12, {"OR", {"IDENT", "dquotetext", 23}, {"IDENT", "squotetext", 24}}}}, 
{"dquotetext", 23, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "\""}}, {"SKIPSPACES", false}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "'"}, {"TERMINAL", "\\\\\""}}}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "\""}}, {"SKIPSPACES", true}}, 
{"squotetext", 24, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "'"}}, {"SKIPSPACES", false}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\'"}}}, {"TAG", {"TERMINAL", " upstream.str = '' "}, {"TERMINAL", "'"}}, {"SKIPSPACES", true}}, 
{"digit", 21, {"OR", {"TERMINAL", "0"}, {"TERMINAL", "1"}, {"TERMINAL", "2"}, {"TERMINAL", "3"}, {"TERMINAL", "4"}, {"TERMINAL", "5"}, {"TERMINAL", "6"}, {"TERMINAL", "7"}, {"TERMINAL", "8"}, {"TERMINAL", "9"}}}, 
{"small", 19, {"OR", {"TERMINAL", "a"}, {"TERMINAL", "b"}, {"TERMINAL", "c"}, {"TERMINAL", "d"}, {"TERMINAL", "e"}, {"TERMINAL", "f"}, {"TERMINAL", "g"}, {"TERMINAL", "h"}, {"TERMINAL", "i"}, {"TERMINAL", "j"}, {"TERMINAL", "k"}, {"TERMINAL", "l"}, {"TERMINAL", "m"}, {"TERMINAL", "n"}, {"TERMINAL", "o"}, {"TERMINAL", "p"}, {"TERMINAL", "q"}, {"TERMINAL", "r"}, {"TERMINAL", "s"}, {"TERMINAL", "t"}, {"TERMINAL", "u"}, {"TERMINAL", "v"}, {"TERMINAL", "w"}, {"TERMINAL", "x"}, {"TERMINAL", "y"}, {"TERMINAL", "z"}}}, 
{"caps", 20, {"OR", {"TERMINAL", "A"}, {"TERMINAL", "B"}, {"TERMINAL", "C"}, {"TERMINAL", "D"}, {"TERMINAL", "E"}, {"TERMINAL", "F"}, {"TERMINAL", "G"}, {"TERMINAL", "H"}, {"TERMINAL", "I"}, {"TERMINAL", "J"}, {"TERMINAL", "K"}, {"TERMINAL", "L"}, {"TERMINAL", "M"}, {"TERMINAL", "N"}, {"TERMINAL", "O"}, {"TERMINAL", "P"}, {"TERMINAL", "Q"}, {"TERMINAL", "R"}, {"TERMINAL", "S"}, {"TERMINAL", "T"}, {"TERMINAL", "U"}, {"TERMINAL", "V"}, {"TERMINAL", "W"}, {"TERMINAL", "X"}, {"TERMINAL", "Y"}, {"TERMINAL", "Z"}}}, 
{"special", 22, {"OR", {"TERMINAL", "_"}, {"TERMINAL", " "}, {"TERMINAL", "."}, {"TERMINAL", ","}, {"TERMINAL", ":"}, {"TERMINAL", ";"}, {"TERMINAL", "!"}, {"TERMINAL", "?"}, {"TERMINAL", "+"}, {"TERMINAL", "-"}, {"TERMINAL", "*"}, {"TERMINAL", "/"}, {"TERMINAL", "="}, {"TERMINAL", "("}, {"TERMINAL", ")"}, {"TERMINAL", "{"}, {"TERMINAL", "}"}, {"TERMINAL", "["}, {"TERMINAL", "]"}, {"TERMINAL", "<"}, {"TERMINAL", ">"}, {"TERMINAL", "\\\\\\\\"}, {"TERMINAL", "\\\\n"}, {"TERMINAL", "\\n"}, {"TERMINAL", "\\\\t"}, {"TERMINAL", "\\t"}, {"TERMINAL", "|"}, {"TERMINAL", "%"}, {"TERMINAL", "$"}, {"TERMINAL", "&"}, {"TERMINAL", "#"}, {"TERMINAL", "@"}}}, 

}, 

{"TAG", {"TERMINAL", " print(upstream.str) "}}, 
{"IDENT", "program", 0}, 
{"TERMINAL", "This aEBNF contains the grammatic and semantic information for annotated EBNF.\n\t\tIt allows to automatically create a compiler for everything described in aEBNF (yes, that format)."}
```
</details>


## Documentation

* upstream
  * upstream.str + upstream.obj
  * upstream.str*
  * upstream.obj*
* c.asg
* c.localAsg
* c.compile
  * &nbsp;
```
    OUT
     ^
     |
     C--.      (C) If the Rule has childs, the childs get sent to 'compile()'. (Also the childs of TAG Rules.)
     |   |
   * ^   v     (*) All upstream values are combined.
    /|   |
   | | _ |
   T | | |     (T) The text of a terminal gets sent to 'upstream.str'.
   | X | |     (X) Here, the script of a single TAG Rule script gets executed. This is after their childs came back from being splitted at (C).
   | | O |     (O) Other Rules are ignored.
   | | | |
   \ | / |
  * \|/  |     (*) Childs from one Rule get splitted.
     |__/
     |
     ^
     IN
```

* Exposed go funcs:
  * print [fmt.Print](https://golang.org/pkg/fmt/#Print)
  * println [fmt.Println](https://golang.org/pkg/fmt/#Println)
  * printf [fmt.Printf](https://golang.org/pkg/fmt/#Printf)
  * sprintf [fmt.Sprintf](https://golang.org/pkg/fmt/#Sprintf)

### LLVM IR

This tool uses [https://github.com/llir/llvm](https://github.com/llir/llvm) to create LLVM IR. 
The API documentation can be found here: [https://pkg.go.dev/github.com/llir/llvm/](https://pkg.go.dev/github.com/llir/llvm/).

The functions and constants are exposed to JS as:

* [llvm.ir](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir)
* [llvm.constant](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant)
* [llvm.metadata](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata)
* [llvm.types](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types)

* Custom functions:
  * The function `llvm.Callgraph(m ir.Module)` creates the callgraph of the given LLVM IR module in Graphviz DOT format (can be viewed e.g. at [http://magjac.com/graphviz-visual-editor/](http://magjac.com/graphviz-visual-editor/)).
  * The function `llvm.Callgraph(m ir.Module, f string)` tries to execute the function `f` inside the IR module `m` and returns the resulting uint32.