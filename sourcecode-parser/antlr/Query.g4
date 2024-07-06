grammar Query;

query           : 'FIND' select_list ('WHERE' expression)? ;
select_list     : select_item (',' select_item)* ;
select_item     : entity 'AS' alias ;
entity          : IDENTIFIER ;
alias           : IDENTIFIER ;
expression      : orExpression ;
orExpression    : andExpression ( 'OR' andExpression )* ;
andExpression   : notExpression ( 'AND' notExpression )* ;
notExpression   : 'NOT' notExpression
                | primary ;
primary         : condition
                | '(' expression ')' ;
condition       : alias '.' method_chain comparator value ;
method_chain    : method_or_variable ('.' method_or_variable)* ;
method_or_variable : method | variable ;
method          : IDENTIFIER '(' ')' ;
variable        : IDENTIFIER ;
comparator      : '=' | '!=' | '<' | '>' | '<=' | '>=' ;
value           : STRING | NUMBER ;
value_list      : value (',' value)* ;
STRING          : '"' ( ~('"' | '\\') | '\\' . )* '"' ;
NUMBER          : [0-9]+ ('.' [0-9]+)? ;
IDENTIFIER      : [a-zA-Z_][a-zA-Z0-9_]* ;
WS              : [ \t\r\n]+ -> skip ;
