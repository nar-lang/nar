The Complete Syntax of Oak
===

```
module ::= "module" qualified_identifier [imports] [definitions]

qualified_identifier ::= identifier["."qualified_identifier]

identifier ::= identifier_first[identifiers_rest]
 
identifier_first ::= "a"-"z" | "A"-"Z"

identifiers_rest := identifier_rest[identifiers_rest]

identifier_rest ::= identifier_first | "0"-"9" | "_" | "`"

imports ::= import [imports]

import ::= "import" qualified_identifier ["as" identifier] ["exposing" {"*" | "(" exposings ")"}]

exposings ::= {identifier | parenthesed_infix_identifier} ["," exposings]

parenthesed_infix_identifier ::= "("infix_identifier")"

infix_identifier ::= infix_char[infix_identifier] 

infix_char ::= "!" | "#" | "$" | "%" | "&" | "*" | "+" | "-" | "/" |
                ":" | ";" | "<" | "=" | ">" | "?" | "^" | "|" | "~" | "`" | "

definitions ::= definition [definitions]

definition ::= "def" ["hidden"] ["extern"] identifier { const_definition | func_definition }

const_definition ::= [type_annotation] "=" expr

type_annotation ::= ":" type

func_definition ::= ( params ) [type_annotation] "=" expr

params ::= pattern ["," params]

type ::= "()" |
            "(" type_list ")" [type_annotation] |
            "{" type_field "}" |
            qualified_identifier ["(" type_list ")"]

type_list ::= type ["," type_list]

type_field ::= identifier ":" type ["," type_field]

pattern ::=  {
    "()" |
    "(" pattern_list ")" |
    "{" identifier_list "}" |
    "[" "]" |
    "[" pattern_list "]" |
    identifier |
    qualified_identifier ["(" type_list ")"] |
    "_" |
    const
  } [type_annotation] ["as" identifier] ["|" pattern]

pattern_list ::= pattern ["," pattern_list]

identifier_list ::= identifier ["," identifier_list]

const ::= char | int | float | string

char ::= "'" ? "'"

int ::= ["-"] [ "0x" | "0X" | "0b" | "0B" | "0o" | "0O" ] {"0"-"9"|"_"}

float ::= ["-"] {"0"-"9"|"_"} ["." {"0"-"9"|"_"}] [{"e"|"E"} ["-"|"+"] {"0"-"9"}]

string :: = `"` * `"` 

expr ::= expr_part infix expr | expr "." identifier | expr "(" expr_list ")" | expr_part

expr_part ::= const |
                "[" "]" |
                "[" expr_list "]" |
                parenthesed_infix_identifier |
                lambda |
                "if" expr "then" expr "else" expr |
                "let" let_defs "in" expr |
                "select" expr cases |
                "."identifier |
                "{" [identifier "|"] expr_kvs "}" |
                "()" |
                "(" expr_list ")" |
                qualified_identifier
    
["(" expr_list ")"] [infix expr]
    
expr_list ::= expr ["," expr_list]

lambda ::= "\(" params ")" "->" expr

let_defs ::= {
        pattern "=" expr |
        identifier "(" params ")" "=" expr
    } ["let" let_defs]

cases ::= pattern "->" expr [cases]

expr_kvs ::= identifier "=" expr ["," expr_kvs]
```
