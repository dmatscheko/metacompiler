# ParserParserCompiler

This is
* a parser that parses an annotated EBNF
* and (at runtime) creates a second parser with the EBNF
* plus a compiler with the annotations in the EBNF.

That new runtime generated parser plus compiler should allow to compile an arbitrary new language, specified in the EBNF and its annotations.

## What is an annotated EBNF (a-EBNF)?

* The EBNF stores the grammar of e.g. a new language.
* The annotations in the EBNF store the semantic of the new language.
* The combined format is annotated EBNF or a-EBNF.

## Examples

### Example of an annotated EBNF
<details>
  <summary>Click to expand!</summary>

```
"a-EBNF of a-EBNF as text"
<~~
var names = [];
function getNameIdx(name) {
    var pos = names.indexOf(name);
    if (pos != -1) { return pos };
    return names.push(name)-1;
}
c.compile(c.asg, up.in)
~~>

{

program                                 <~~push(up.in)~~>
                =
                [
                    title               <~~ up.in += ', \\n' ~~>
                ]
                programtag
                "{"                     <~~ up.in = '\\n{\\n\\n' ~~>
                { production }
                "}"                     <~~ up.in = '\\n}, \\n\\n' ~~>
                programtag
                start
                [
                    comment             <~~ up.in = ', \\n'+up.in+'\\n' ~~>
                ] ;

programtag      =
                [
                    tag                 <~~ up.in = '{"TAG", '+up.in+'}, \\n' ~~>
                ] ;

production                              <~~ var productionTag = pop();
                                        if (productionTag != undefined) {
                                            pop()/*=undefined*/;
                                            up.in = '{"TAG", '+productionTag+', '+up.in+'}'
                                        };
                                        up.in += ',\\n' ~~>       
                =
                name                    <~~ up.in = '{"'+up.in+'", '+getNameIdx(up.in)+', ';
                                        push(undefined) ~~>
                [
                    tag                 <~~ push(up.in); up.in = '' ~~>
                ]
                "="                     <~~up.in=''~~>
                [ expression ]
                ( "." | ";" )           <~~ up.in = '}' ~~> ;

expression                              <~~ if (/*or*/ pop()) {
                                            up.in = '{"OR", '+up.in+'}'
                                        } ~~>
                =
                alternative             <~~ /*or=false*/ push(false); ~~>
                {
                    "|"                 <~~up.in=''~~>
                    alternative         <~~ /*or=true*/ pop();push(true); up.in = ', '+up.in ~~>
                } ;

alternative     =
                taggedterm
                {
                    taggedterm          <~~ up.in = ', '+up.in ~~>
                } ;

taggedterm                              <~~up.in=pop()~~>
                =
                term                    <~~push(up.in)~~>
                [
                    tag                 <~~ push('{"TAG", '+up.in+', '+pop()+'}') ~~>
                ] ;

term            =
                (
                    name                <~~ up.in = '{"IDENT", "'+up.in+'", '+getNameIdx(up.in)+'}' ~~>
                    |
                    ( text [ "..." text ] )
                    |
                    group
                    |
                    option
                    |
                    repetition
                    |
                    skipspaces
                ) ;

group           <~~ up.in = '{'+pop()+'}' ~~>                    = "(" expression <~~push(up.in)~~> ")" ;
option          <~~ up.in = '{"OPTIONAL", '+pop()+'}' ~~>        = "[" expression <~~push(up.in)~~> "]" ;
repetition      <~~ up.in = '{"REPEAT", '+pop()+'}' ~~>          = "{" expression <~~push(up.in)~~> "}" ;
skipspaces      =
                "+"                     <~~ up.in = '{"SKIPSPACES", true}' ~~>
                |
                "-"                     <~~ up.in = '{"SKIPSPACES", false}' ~~> ;

title           = text ;
start           = name                  <~~ up.in = '{"IDENT", "'+up.in+'", '+getNameIdx(up.in)+'}' ~~> ;
comment         = text ;

tag                                     <~~up.in=pop()~~>
                =
                "<"
                code                    <~~push(up.in)~~>
                {
                    ","
                    code                <~~ push(pop()+', '+up.in) ~~>
                }
                ">" ;

code                                    <~~ up.in = '{"TERMINAL", '+sprintf("%q",pop())+'}' ~~>
                =
                '~~'
                -
                { [ "~" ] codeinner }   <~~push(up.in)~~>
                '~~'
                + ;

codeinner       = small | caps | digit | special | "'" | '"' | "\\~" ;

name            = ( small | caps ) - { small | caps | digit | "_" } + ;

text                                    <~~ up.in = '{"TERMINAL", '+sprintf("%q",pop())+'}' ~~>
                = dquotetext | squotetext ;
dquotetext      = '"' - { small | caps | digit | special | "~" | "'" | '\\"' } <~~push(up.in)~~> '"' + ;
squotetext      = "'" - { small | caps | digit | special | "~" | '"' | "\\'" } <~~push(up.in)~~> "'" + ;

digit           = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
small           = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" |
                  "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
caps            = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" |
                  "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
special         = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" |
                  "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "|" | "%" | "$" | "&" | "#" |
                  "@" | "\\\\" | "\\t" | "\t" | "\\n" | "\n" | "\\r" | "\r" ;

}

<~~ print(pop()) ~~>
program
"This aEBNF contains the grammatic and semantic information for annotated EBNF.
It allows to automatically create a compiler for everything described in aEBNF (yes, that format)."
```
</details>

### Its output, when it gets applied on itself:
<details>
  <summary>Click to expand!</summary>

```
{"TERMINAL", "a-EBNF of a-EBNF as text"}, 
{"TAG", {"TERMINAL", "\nvar names = [];\nfunction getNameIdx(name) {\n    var pos = names.indexOf(name);\n    if (pos != -1) { return pos };\n    return names.push(name)-1;\n}\nc.compile(c.asg, up.in)\n"}}, 

{

{"TAG", {"TERMINAL", "push(up.in)"}, {"program", 0, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in += ', \\\\n' "}, {"IDENT", "title", 1}}}, {"IDENT", "programtag", 2}, {"TAG", {"TERMINAL", " up.in = '\\\\n{\\\\n\\\\n' "}, {"TERMINAL", "{"}}, {"REPEAT", {"IDENT", "production", 3}}, {"TAG", {"TERMINAL", " up.in = '\\\\n}, \\\\n\\\\n' "}, {"TERMINAL", "}"}}, {"IDENT", "programtag", 2}, {"IDENT", "start", 4}, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in = ', \\\\n'+up.in+'\\\\n' "}, {"IDENT", "comment", 5}}}}},
{"programtag", 2, {"OPTIONAL", {"TAG", {"TERMINAL", " up.in = '{\"TAG\", '+up.in+'}, \\\\n' "}, {"IDENT", "tag", 6}}}},
{"TAG", {"TERMINAL", " var productionTag = pop();\n                                        if (productionTag != undefined) {\n                                            pop()/*=undefined*/;\n                                            up.in = '{\"TAG\", '+productionTag+', '+up.in+'}'\n                                        };\n                                        up.in += ',\\\\n' "}, {"production", 3, {"TAG", {"TERMINAL", " up.in = '{\"'+up.in+'\", '+getNameIdx(up.in)+', ';\n                                        push(undefined) "}, {"IDENT", "name", 7}}, {"OPTIONAL", {"TAG", {"TERMINAL", " push(up.in); up.in = '' "}, {"IDENT", "tag", 6}}}, {"TAG", {"TERMINAL", "up.in=''"}, {"TERMINAL", "="}}, {"OPTIONAL", {"IDENT", "expression", 8}}, {"TAG", {"TERMINAL", " up.in = '}' "}, {{"OR", {"TERMINAL", "."}, {"TERMINAL", ";"}}}}}},
{"TAG", {"TERMINAL", " if (/*or*/ pop()) {\n                                            up.in = '{\"OR\", '+up.in+'}'\n                                        } "}, {"expression", 8, {"TAG", {"TERMINAL", " /*or=false*/ push(false); "}, {"IDENT", "alternative", 9}}, {"REPEAT", {"TAG", {"TERMINAL", "up.in=''"}, {"TERMINAL", "|"}}, {"TAG", {"TERMINAL", " /*or=true*/ pop();push(true); up.in = ', '+up.in "}, {"IDENT", "alternative", 9}}}}},
{"alternative", 9, {"IDENT", "taggedterm", 10}, {"REPEAT", {"TAG", {"TERMINAL", " up.in = ', '+up.in "}, {"IDENT", "taggedterm", 10}}}},
{"TAG", {"TERMINAL", "up.in=pop()"}, {"taggedterm", 10, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "term", 11}}, {"OPTIONAL", {"TAG", {"TERMINAL", " push('{\"TAG\", '+up.in+', '+pop()+'}') "}, {"IDENT", "tag", 6}}}}},
{"term", 11, {{"OR", {"TAG", {"TERMINAL", " up.in = '{\"IDENT\", \"'+up.in+'\", '+getNameIdx(up.in)+'}' "}, {"IDENT", "name", 7}}, {{"IDENT", "text", 12}, {"OPTIONAL", {"TERMINAL", "..."}, {"IDENT", "text", 12}}}, {"IDENT", "group", 13}, {"IDENT", "option", 14}, {"IDENT", "repetition", 15}, {"IDENT", "skipspaces", 16}}}},
{"TAG", {"TERMINAL", " up.in = '{'+pop()+'}' "}, {"group", 13, {"TERMINAL", "("}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "expression", 8}}, {"TERMINAL", ")"}}},
{"TAG", {"TERMINAL", " up.in = '{\"OPTIONAL\", '+pop()+'}' "}, {"option", 14, {"TERMINAL", "["}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "expression", 8}}, {"TERMINAL", "]"}}},
{"TAG", {"TERMINAL", " up.in = '{\"REPEAT\", '+pop()+'}' "}, {"repetition", 15, {"TERMINAL", "{"}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "expression", 8}}, {"TERMINAL", "}"}}},
{"skipspaces", 16, {"OR", {"TAG", {"TERMINAL", " up.in = '{\"SKIPSPACES\", true}' "}, {"TERMINAL", "+"}}, {"TAG", {"TERMINAL", " up.in = '{\"SKIPSPACES\", false}' "}, {"TERMINAL", "-"}}}},
{"title", 1, {"IDENT", "text", 12}},
{"start", 4, {"TAG", {"TERMINAL", " up.in = '{\"IDENT\", \"'+up.in+'\", '+getNameIdx(up.in)+'}' "}, {"IDENT", "name", 7}}},
{"comment", 5, {"IDENT", "text", 12}},
{"TAG", {"TERMINAL", "up.in=pop()"}, {"tag", 6, {"TERMINAL", "<"}, {"TAG", {"TERMINAL", "push(up.in)"}, {"IDENT", "code", 17}}, {"REPEAT", {"TERMINAL", ","}, {"TAG", {"TERMINAL", " push(pop()+', '+up.in) "}, {"IDENT", "code", 17}}}, {"TERMINAL", ">"}}},
{"TAG", {"TERMINAL", " up.in = '{\"TERMINAL\", '+sprintf(\"%q\",pop())+'}' "}, {"code", 17, {"TERMINAL", "~~"}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OPTIONAL", {"TERMINAL", "~"}}, {"IDENT", "codeinner", 18}}}, {"TERMINAL", "~~"}, {"SKIPSPACES", true}}},
{"codeinner", 18, {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "'"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\~"}}},
{"name", 7, {{"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}}}, {"SKIPSPACES", false}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"TERMINAL", "_"}}}, {"SKIPSPACES", true}},
{"TAG", {"TERMINAL", " up.in = '{\"TERMINAL\", '+sprintf(\"%q\",pop())+'}' "}, {"text", 12, {"OR", {"IDENT", "dquotetext", 23}, {"IDENT", "squotetext", 24}}}},
{"dquotetext", 23, {"TERMINAL", "\""}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "'"}, {"TERMINAL", "\\\\\""}}}}, {"TERMINAL", "\""}, {"SKIPSPACES", true}},
{"squotetext", 24, {"TERMINAL", "'"}, {"SKIPSPACES", false}, {"TAG", {"TERMINAL", "push(up.in)"}, {"REPEAT", {"OR", {"IDENT", "small", 19}, {"IDENT", "caps", 20}, {"IDENT", "digit", 21}, {"IDENT", "special", 22}, {"TERMINAL", "~"}, {"TERMINAL", "\""}, {"TERMINAL", "\\\\'"}}}}, {"TERMINAL", "'"}, {"SKIPSPACES", true}},
{"digit", 21, {"OR", {"TERMINAL", "0"}, {"TERMINAL", "1"}, {"TERMINAL", "2"}, {"TERMINAL", "3"}, {"TERMINAL", "4"}, {"TERMINAL", "5"}, {"TERMINAL", "6"}, {"TERMINAL", "7"}, {"TERMINAL", "8"}, {"TERMINAL", "9"}}},
{"small", 19, {"OR", {"TERMINAL", "a"}, {"TERMINAL", "b"}, {"TERMINAL", "c"}, {"TERMINAL", "d"}, {"TERMINAL", "e"}, {"TERMINAL", "f"}, {"TERMINAL", "g"}, {"TERMINAL", "h"}, {"TERMINAL", "i"}, {"TERMINAL", "j"}, {"TERMINAL", "k"}, {"TERMINAL", "l"}, {"TERMINAL", "m"}, {"TERMINAL", "n"}, {"TERMINAL", "o"}, {"TERMINAL", "p"}, {"TERMINAL", "q"}, {"TERMINAL", "r"}, {"TERMINAL", "s"}, {"TERMINAL", "t"}, {"TERMINAL", "u"}, {"TERMINAL", "v"}, {"TERMINAL", "w"}, {"TERMINAL", "x"}, {"TERMINAL", "y"}, {"TERMINAL", "z"}}},
{"caps", 20, {"OR", {"TERMINAL", "A"}, {"TERMINAL", "B"}, {"TERMINAL", "C"}, {"TERMINAL", "D"}, {"TERMINAL", "E"}, {"TERMINAL", "F"}, {"TERMINAL", "G"}, {"TERMINAL", "H"}, {"TERMINAL", "I"}, {"TERMINAL", "J"}, {"TERMINAL", "K"}, {"TERMINAL", "L"}, {"TERMINAL", "M"}, {"TERMINAL", "N"}, {"TERMINAL", "O"}, {"TERMINAL", "P"}, {"TERMINAL", "Q"}, {"TERMINAL", "R"}, {"TERMINAL", "S"}, {"TERMINAL", "T"}, {"TERMINAL", "U"}, {"TERMINAL", "V"}, {"TERMINAL", "W"}, {"TERMINAL", "X"}, {"TERMINAL", "Y"}, {"TERMINAL", "Z"}}},
{"special", 22, {"OR", {"TERMINAL", "_"}, {"TERMINAL", " "}, {"TERMINAL", "."}, {"TERMINAL", ","}, {"TERMINAL", ":"}, {"TERMINAL", ";"}, {"TERMINAL", "!"}, {"TERMINAL", "?"}, {"TERMINAL", "+"}, {"TERMINAL", "-"}, {"TERMINAL", "*"}, {"TERMINAL", "/"}, {"TERMINAL", "="}, {"TERMINAL", "("}, {"TERMINAL", ")"}, {"TERMINAL", "{"}, {"TERMINAL", "}"}, {"TERMINAL", "["}, {"TERMINAL", "]"}, {"TERMINAL", "<"}, {"TERMINAL", ">"}, {"TERMINAL", "|"}, {"TERMINAL", "%"}, {"TERMINAL", "$"}, {"TERMINAL", "&"}, {"TERMINAL", "#"}, {"TERMINAL", "@"}, {"TERMINAL", "\\\\\\\\"}, {"TERMINAL", "\\\\t"}, {"TERMINAL", "\\t"}, {"TERMINAL", "\\\\n"}, {"TERMINAL", "\\n"}, {"TERMINAL", "\\\\r"}, {"TERMINAL", "\\r"}}},

}, 

{"TAG", {"TERMINAL", " print(pop()) "}}, 
{"IDENT", "program", 0}, 
{"TERMINAL", "This aEBNF contains the grammatic and semantic information for annotated EBNF.\nIt allows to automatically create a compiler for everything described in aEBNF (yes, that format)."}
```
</details>


## Documentation

### Installation / Build

```
go get "github.com/dop251/goja" "github.com/llir/llvm/ir"
go run . -f tests/llvm-ir-tests.aebnf -s tests/tiny.aebnf
```

### Exposed JS API

#### Output

* print [fmt.Print](https://golang.org/pkg/fmt/#Print)
* println [fmt.Println](https://golang.org/pkg/fmt/#Println)
* printf [fmt.Printf](https://golang.org/pkg/fmt/#Printf)
* sprintf [fmt.Sprintf](https://golang.org/pkg/fmt/#Sprintf)

#### Data Handling

* __up__ (for upstream)  
  All local variables. All _'up.*'_ variables can be changed by the user. This includes _'up.in'_.
  * __up.in__  
  (string) The collective matched strings of all child nodes.
  * __up.\*__  
  User generated local variables. They can be arbitrary objects. Those objects are concatenated to arrays of objects when being propagated upwards.
  * __up.str\*__  
  User generated local variables. All variables that start with _'str'_ must be strings. Those objects are concatenated as strings when being propagated upwards. up.in is an example of such string concatenation.
* __ltr__ (for left to right)  
  All global variables (Global JS variables can be used too). All _'ltr.*'_ variables can be changed by the user. This includes _'ltr.in'_.
  * __ltr.in__  
  (string) The collective matched strings of all nodes from left to right. Only matched strings of nodes to the right (that are not processed yet), are not included.  
  * __ltr.\*__  
  User generated global variables. They can be arbitrary objects. Except for _'ltr.in'_, those objects are not changed by the compiler.

##### The stack

This JS API provides an added stack. This is useful to bring data to the other side of EBNF matchers.

* __pop() object__  
Pops an arbitrary object from the stack.
* __push(object)__  
Pushes an arbitrary object onto the stack.

#### Compiler API

* __c.asg__
The whole abstract semantic graph.
* __c.localAsg__
The local part of the abstract semantic graph.
* __c.compile(asg, string) string__  
  Compiles the given ASG and upstream string and returns the combined matched upstream string. Those combined string can have been modified by the annotations.  
Normally, _'c.compile()'_ is called as `c.compile(asg, up.in);` or even `c.compile(asg, "");`.
  The compiler works like this:  
```
    OUT
     ^
     |
     C--.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of TAG Rules.)
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

#### LLVM IR API

This tool uses [https://github.com/llir/llvm](https://github.com/llir/llvm) to create LLVM IR. 
The API documentation can be found here: [https://pkg.go.dev/github.com/llir/llvm/](https://pkg.go.dev/github.com/llir/llvm/).

The functions and constants are exposed to JS as:

* [llvm.ir](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir)
* [llvm.constant](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/constant)
* [llvm.metadata](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/metadata)
* [llvm.types](https://pkg.go.dev/github.com/llir/llvm@v0.3.2/ir/types)

* Custom functions:
  * The function `llvm.Callgraph(m ir.Module) string` creates the callgraph of the given LLVM IR module in Graphviz DOT format (can be viewed e.g. at [http://magjac.com/graphviz-visual-editor/](http://magjac.com/graphviz-visual-editor/)).
  * The function `llvm.Callgraph(m ir.Module, f string)` tries to execute the function `f` inside the IR module `m` and returns the resulting uint32.