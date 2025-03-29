package model

import (
	"fmt"
)

// Variable represents a field, local variable, or method parameter.
type Variable struct {
	Name              string   // Name of the variable
	Type              string   // Data type of the variable
	Scope             string   // Scope of the variable (e.g., "field", "local", "parameter")
	Initializer       string   // Initial value if available (e.g., `int x = 10;` → "10")
	AssignedValues    []string // List of expressions assigned to this variable
	SourceDeclaration string   // Location of the variable declaration
}

// NewVariable initializes a new Variable instance.
func NewVariable(name, varType, scope, initializer string, assignedValues []string, sourceDeclaration string) *Variable {
	return &Variable{
		Name:              name,
		Type:              varType,
		Scope:             scope,
		Initializer:       initializer,
		AssignedValues:    assignedValues,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAnAssignedValue retrieves values assigned to this variable.
func (v *Variable) GetAnAssignedValue() []string {
	return v.AssignedValues
}

// GetInitializer retrieves the initializer of this variable.
func (v *Variable) GetInitializer() string {
	return v.Initializer
}

// GetType retrieves the type of this variable.
func (v *Variable) GetType() string {
	return v.Type
}

// PP returns a formatted representation of the variable.
func (v *Variable) PP() string {
	initStr := ""
	if v.Initializer != "" {
		initStr = fmt.Sprintf(" = %s", v.Initializer)
	}
	return fmt.Sprintf("%s %s%s;", v.Type, v.Name, initStr)
}

// LocalScopeVariable represents a method parameter or a local variable.
type LocalScopeVariable struct {
	Variable
	Name              string // Name of the variable
	Type              string // Data type of the variable
	Scope             string // Either "local" or "parameter"
	DeclaredIn        string // Callable (method or constructor) in which the variable is declared
	SourceDeclaration string // Location of the variable declaration
}

// NewLocalScopeVariable initializes a new LocalScopeVariable instance.
func NewLocalScopeVariable(name, varType, scope, declaredIn, sourceDeclaration string) *LocalScopeVariable {
	return &LocalScopeVariable{
		Name:              name,
		Type:              varType,
		Scope:             scope,
		DeclaredIn:        declaredIn,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicate

// GetCallable retrieves the method or constructor in which this variable is declared.
func (v *LocalScopeVariable) GetCallable() string {
	return v.DeclaredIn
}

// LocalVariableDecl represents a local variable declaration inside a method or block.
type LocalVariableDecl struct {
	LocalScopeVariable
	Name              string // Name of the local variable
	Type              string // Data type of the variable
	Callable          string // The callable (method/constructor) in which this variable is declared
	DeclExpr          string // The declaration expression (e.g., `int x = 5;`)
	Initializer       string // The right-hand side of the declaration (if any)
	ParentScope       string // The enclosing block or statement
	SourceDeclaration string // Location of the variable declaration
}

// NewLocalVariableDecl initializes a new LocalVariableDecl instance.
func NewLocalVariableDecl(name, varType, callable, declExpr, initializer, parentScope, sourceDeclaration string) *LocalVariableDecl {
	return &LocalVariableDecl{
		Name:              name,
		Type:              varType,
		Callable:          callable,
		DeclExpr:          declExpr,
		Initializer:       initializer,
		ParentScope:       parentScope,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (lv *LocalVariableDecl) GetAPrimaryQlClass() string {
	return "LocalVariableDecl"
}

// GetCallable retrieves the method or constructor in which this variable is declared.
func (lv *LocalVariableDecl) GetCallable() string {
	return lv.Callable
}

// GetDeclExpr retrieves the full declaration expression of this variable.
func (lv *LocalVariableDecl) GetDeclExpr() string {
	return lv.DeclExpr
}

// GetEnclosingCallable retrieves the enclosing callable (same as `GetCallable()`).
func (lv *LocalVariableDecl) GetEnclosingCallable() string {
	return lv.Callable
}

// GetInitializer retrieves the initializer expression if available.
func (lv *LocalVariableDecl) GetInitializer() string {
	return lv.Initializer
}

// GetParent retrieves the parent block or statement that encloses this variable.
func (lv *LocalVariableDecl) GetParent() string {
	return lv.ParentScope
}

// GetType retrieves the type of this local variable.
func (lv *LocalVariableDecl) GetType() string {
	return lv.Type
}

// ToString returns a textual representation of the local variable declaration.
func (lv *LocalVariableDecl) ToString() string {
	if lv.Initializer != "" {
		return fmt.Sprintf("%s %s = %s;", lv.Type, lv.Name, lv.Initializer)
	}
	return fmt.Sprintf("%s %s;", lv.Type, lv.Name)
}
