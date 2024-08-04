grammar Query;

query           : 'FIND' select_list ('WHERE' expression)? ;
select_list     : select_item (',' select_item)* ;
select_item     : entity 'AS' alias ;
entity          : IDENTIFIER ;
alias           : IDENTIFIER ;
expression      : orExpression ;
orExpression    : andExpression ( '||' andExpression )* ;
andExpression   : primary ( '&&' primary )* ;
primary         : condition
                | '(' expression ')' ;
condition       : (value | alias '.' method_chain | '[' value_list ']') comparator (value | alias '.' method_chain | '[' value_list ']') ;
method_chain    : method_or_variable ('.' method_or_variable)* ;
method_or_variable : method | variable ;
method          : IDENTIFIER '(' ')' ;
variable        : IDENTIFIER ;
comparator      : '==' | '!=' | '<' | '>' | '<=' | '>=' | 'LIKE' | 'in' ;
value           : STRING | NUMBER | STRING_WITH_WILDCARD ;
value_list      : value (',' value)* ;
STRING          : '"' ( ~('"' | '\\') | '\\' . )* '"' ;
STRING_WITH_WILDCARD : '"' ( ~('"' | '\\') | '\\' . | '%' )* '"' ;
NUMBER          : [0-9]+ ('.' [0-9]+)? ;
IDENTIFIER      : [a-zA-Z_][a-zA-Z0-9_]* ;
WS              : [ \t\r\n]+ -> skip ;
