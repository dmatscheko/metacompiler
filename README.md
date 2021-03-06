# ParserParserCompiler

Note: The developed version is at [./parserparsercompiler_test](./parserparsercompiler_test).

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
"aEBNF of aEBNF as text" <~~ 
		var names = [];
		function getNameIdx(name) {
			var pos = names.indexOf(name);
			if (pos != -1) { return pos };
			return names.push(name);
		}
		~~> {
			program          <~~ print('{'+upstream.str+'}') ~~>                             = [ title ] [ tag ] "{" { production } "}" [ tag ] start [ comment ] ;
			production       <~~ upstream.str = '{"'+upstream.name+'", '+getNameIdx(upstream.name)+', '+upstream.str+'}' ~~>       = name [ tag ] "=" <~~ upstream.str = '' ~~> [ expression ] ( "." | ";" ) <~~ upstream.str = '' ~~> ;
			expression       <~~ if (upstream.or) { upstream.str = '{"OR", '+upstream.str+'}' } ~~> = alternative <~~ upstream.or = false ~~> { "|" <~~ upstream.str = '' ~~> alternative <~~ upstream.or = true; upstream.str = ', '+upstream.str ~~> } ;
			alternative      = term { term <~~ upstream.str = ', '+upstream.str ~~> } ;
			term             = ( name | ( text [ "..." text ] ) | group | option | repetition | skipspaces ) [ tag ] ;
			group            <~~ upstream.str = '{'+upstream.str+'}' ~~>                  = "(" <~~ upstream.str = '' ~~> expression ")" <~~ upstream.str = '' ~~> ;
			option           <~~ upstream.str = '{"OPTIONAL", '+upstream.str+'}' ~~>      = "[" <~~ upstream.str = '' ~~> expression "]" <~~ upstream.str = '' ~~> ;
			repetition       <~~ upstream.str = '{"REPEAT", '+upstream.str+'}' ~~>        = "{" <~~ upstream.str = '' ~~> expression "}" <~~ upstream.str = '' ~~> ;
			skipspaces       = "+" <~~ upstream.str = '{"SKIPSPACES", true}' ~~> | "-" <~~ upstream.str = '{"SKIPSPACES", false}' ~~> ;
	
			title            = text ;
			start            = name ;
			comment          = text ;
	
			tag <~~ upstream.str = '{"TAG", '+upstream.str+'}' ~~>                        = "<" <~~ upstream.str = '' ~~> code { "," <~~ upstream.str = '' ~~> code <~~ upstream.str = ', '+upstream.str ~~> } ">" <~~ upstream.str = '' ~~> ;

			code             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>      = '~~' - { [ "~" ] codeinner } '~~' + ;
			codeinner        = small | caps | digit | special | "'" | '"' | "\\~" ;

			name             <~~ upstream.name = upstream.str; upstream.str = '{"IDENT", "'+upstream.name+'", '+getNameIdx(upstream.name)+'}' ~~>  = ( small | caps ) - { small | caps | digit | "_" } + ;

			text             <~~ upstream.str = '{"TERMINAL", '+upstream.str+'}' ~~>                     = dquotetext | squotetext ;
			dquotetext       = '"' - { small | caps | digit | special | "~" | "'" | '\\"' } '"' + ;
			squotetext       = "'" - { small | caps | digit | special | "~" | '"' | "\\'" } "'" + ;
	
			digit            = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
			small            = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z" ;
			caps             = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" ;
			special          = "_" | " " | "." | "," | ":" | ";" | "!" | "?" | "+" | "-" | "*" | "/" | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "<" | ">" | "\\\\" | "\\n" | "\n" | "\\t" | "\t" | "|" | "%" | "$" | "&" | "#" | "@" ;
	
			} program
```
