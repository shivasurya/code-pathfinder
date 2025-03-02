grammar Query;

query: class_declarations? predicate_declarations? FROM select_list (WHERE expression)? SELECT select_clause;

class_declarations: class_declaration+;
class_declaration: 'class' class_name '{' method_declarations '}';
class_name: IDENTIFIER;
method_declarations: method_declaration+;
method_declaration: return_type method_name '(' parameter_list? ')' '{' method_body '}';
method_name: IDENTIFIER;
method_body: return_statement;
return_statement: 'result' '=' value;
return_type: type;

predicate_declarations: predicate_declaration+;
predicate_declaration: PREDICATE predicate_name '(' parameter_list? ')' '{' expression '}';
predicate_name: IDENTIFIER;
parameter_list: parameter (',' parameter)*;
parameter: (type | class_name) IDENTIFIER;
type: IDENTIFIER;

select_list: select_item (',' select_item)*;
select_item: (entity | class_name) AS alias;
entity: IDENTIFIER;
alias: IDENTIFIER;

expression: orExpression;
orExpression: andExpression ( '||' andExpression )*;
andExpression: equalityExpression ( '&&' equalityExpression )*;
equalityExpression: relationalExpression ( ( '==' | '!=' ) relationalExpression )*;
relationalExpression: additiveExpression ( ( '<' | '>' | '<=' | '>=' | ' in ' ) additiveExpression )*;
additiveExpression: multiplicativeExpression ( ( '+' | '-' ) multiplicativeExpression )*;
multiplicativeExpression: unaryExpression ( ( '*' | '/' ) unaryExpression )*;
unaryExpression: ( '!' | '-' ) unaryExpression | primary;
primary: operand | predicate_invocation | '(' expression ')';
operand: value | variable | alias '.' method_chain | class_name '.' method_chain | '[' value_list ']';
method_chain: (class_name '.')? method_name '(' argument_list? ')';
method_or_variable: method_invocation | variable | predicate_invocation;
method_invocation: IDENTIFIER '(' argument_list? ')';
variable: IDENTIFIER;
predicate_invocation: predicate_name '(' argument_list? ')';
argument_list: argument (',' argument)*;
argument: expression | STRING;
comparator: '==' | '!=' | '<' | '>' | '<=' | '>=' | 'LIKE' | 'in';
value: STRING | NUMBER | STRING_WITH_WILDCARD;
value_list: value (',' value)*;
select_clause: select_expression (',' select_expression)*;
select_expression: variable | method_chain | STRING;

STRING: '"' ( ~('"' | '\\') | '\\' . )* '"';
STRING_WITH_WILDCARD: '"' ( ~('"' | '\\') | '\\' . | '%' )* '"';
NUMBER: [0-9]+ ('.' [0-9]+)?;
PREDICATE: 'predicate';
FROM: 'FROM';
WHERE: 'WHERE';
AS: 'AS';
SELECT: 'SELECT';
IDENTIFIER: [a-zA-Z_][a-zA-Z0-9_]*;
WS: [ \t\r\n]+ -> skip;