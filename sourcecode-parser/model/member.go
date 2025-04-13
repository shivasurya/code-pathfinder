package model

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Callable represents an invocable Java element (Method or Constructor).
type Callable struct {
	StmtParent
	Name              string   // Name of the callable (e.g., method or constructor)
	QualifiedName     string   // Fully qualified name (e.g., "com.example.User.getName")
	ReturnType        string   // Return type (void for constructors)
	Parameters        []string // List of parameter types
	ParameterNames    []string // List of parameter names
	IsVarargs         bool     // Whether the last parameter is a varargs parameter
	SourceDeclaration string   // Source code location of this callable
}

// NewCallable initializes a new Callable instance.
func NewCallable(name, qualifiedName, returnType string, parameters []string, parameterNames []string, isVarargs bool, sourceDeclaration string) *Callable {
	return &Callable{
		Name:              name,
		QualifiedName:     qualifiedName,
		ReturnType:        returnType,
		Parameters:        parameters,
		ParameterNames:    parameterNames,
		IsVarargs:         isVarargs,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAParamType retrieves all parameter types of this callable.
func (c *Callable) GetAParamType() []string {
	return c.Parameters
}

// GetAParameter retrieves all formal parameters (type + name).
func (c *Callable) GetAParameter() []string {
	params := []string{}
	for i, paramType := range c.Parameters {
		params = append(params, fmt.Sprintf("%s %s", paramType, c.ParameterNames[i]))
	}
	return params
}

// GetNumberOfParameters returns the number of parameters.
func (c *Callable) GetNumberOfParameters() int {
	return len(c.Parameters)
}

// GetParameter retrieves a specific parameter type by index.
func (c *Callable) GetParameter(index int) string {
	if index >= 0 && index < len(c.Parameters) {
		return fmt.Sprintf("%s %s", c.Parameters[index], c.ParameterNames[index])
	}
	return ""
}

// GetParameterType retrieves a specific parameter type by index.
func (c *Callable) GetParameterType(index int) string {
	if index >= 0 && index < len(c.Parameters) {
		return c.Parameters[index]
	}
	return ""
}

// GetReturnType returns the declared return type of this callable.
func (c *Callable) GetReturnType() string {
	return c.ReturnType
}

// GetSignature returns the fully qualified method signature.
func (c *Callable) GetSignature() string {
	return fmt.Sprintf("%s %s(%v)", c.ReturnType, c.Name, strings.Join(c.Parameters, ", "))
}

// GetSourceDeclaration returns the source declaration of this callable.
func (c *Callable) GetSourceDeclaration() string {
	return c.SourceDeclaration
}

// GetStringSignature returns a string signature of this callable.
func (c *Callable) GetStringSignature() string {
	return fmt.Sprintf("%s(%v)", c.Name, strings.Join(c.Parameters, ", "))
}

// GetVarargsParameterIndex returns the index of the varargs parameter, if one exists.
func (c *Callable) GetVarargsParameterIndex() int {
	if c.IsVarargs {
		return len(c.Parameters) - 1
	}
	return -1 // Indicates no varargs parameter
}

// HasNoParameters checks if this callable has no parameters.
func (c *Callable) HasNoParameters() bool {
	return len(c.Parameters) == 0
}

// IsVarargs checks if the last parameter of this callable is a varargs parameter.
func (c *Callable) GetIsVarargs() bool {
	return c.IsVarargs
}

// ParamsString returns a formatted string of parameter types.
func (c *Callable) ParamsString() string {
	if len(c.Parameters) == 0 {
		return "()"
	}
	return fmt.Sprintf("(%v)", strings.Join(c.Parameters, ", "))
}

// Method represents a Java method declaration.
type Method struct {
	Callable
	Name              string   // Name of the method
	QualifiedName     string   // Fully qualified method name
	ReturnType        string   // Return type of the method
	Parameters        []string // List of parameter types
	ParameterNames    []string // List of parameter names
	Visibility        string   // Visibility (public, private, protected, package-private)
	IsAbstract        bool     // Whether this method is abstract
	IsStrictfp        bool     // Whether this method is strictfp
	IsStatic          bool     // Whether this method is static
	IsFinal           bool     // Whether this method is final
	IsConstructor     bool     // Whether this method is a constructor
	SourceDeclaration string   // Location of the source declaration
	ID                string   // ID of the method
	ClassId           string   // ID of the class
}

// NewMethod initializes a new Method instance.
func NewMethod(name, qualifiedName, returnType string, parameters []string, parameterNames []string, visibility string, isAbstract, isStrictfp, isStatic, isFinal, isConstructor bool, sourceDeclaration string) *Method {
	return &Method{
		Name:              name,
		QualifiedName:     qualifiedName,
		ReturnType:        returnType,
		Parameters:        parameters,
		ParameterNames:    parameterNames,
		Visibility:        visibility,
		IsAbstract:        isAbstract,
		IsStrictfp:        isStrictfp,
		IsStatic:          isStatic,
		IsFinal:           isFinal,
		IsConstructor:     isConstructor,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (m *Method) GetAPrimaryQlClass() string {
	return "Method"
}

func (m *Method) GetName() string {
	return m.Name
}

func (m *Method) GetFullyQualifiedName() string {
	return m.QualifiedName
}

func (m *Method) GetReturnType() string {
	return m.ReturnType
}

func (m *Method) GetParameters() []string {
	return m.Parameters
}

func (m *Method) GetParameterNames() []string {
	return m.ParameterNames
}

func (m *Method) GetVisibility() string {
	return m.Visibility
}

func (m *Method) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"getVisibility":     m.GetVisibility,
		"getReturnType":     m.GetReturnType,
		"getName":           m.GetName,
		"getParameters":     m.GetParameters,
		"getParameterNames": m.GetParameterNames,
	}
}

// GetSignature returns the fully qualified method signature.
func (m *Method) GetSignature() string {
	return fmt.Sprintf("%s %s(%v)", m.ReturnType, m.Name, strings.Join(m.Parameters, ", "))
}

// GetSourceDeclaration returns the source declaration of this method.
func (m *Method) GetSourceDeclaration() string {
	return m.SourceDeclaration
}

// IsAbstract checks if this method is abstract.
func (m *Method) GetIsAbstract() bool {
	return m.IsAbstract
}

// IsInheritable checks if this method is inheritable (not private, static, or final).
func (m *Method) IsInheritable() bool {
	return m.Visibility != "private" && !m.IsStatic && !m.IsFinal
}

// IsPublic checks if this method is public.
func (m *Method) IsPublic() bool {
	return m.Visibility == "public"
}

// IsStrictfp checks if this method is strictfp.
func (m *Method) GetIsStrictfp() bool {
	return m.IsStrictfp
}

// SameParamTypes checks if two methods have the same parameter types.
func (m *Method) SameParamTypes(other *Method) bool {
	if len(m.Parameters) != len(other.Parameters) {
		return false
	}
	for i := range m.Parameters {
		if m.Parameters[i] != other.Parameters[i] {
			return false
		}
	}
	return true
}

// Add these methods to the existing Method struct.
func (m *Method) Insert(db *sql.DB) error {
	query := `INSERT INTO method_decl (
        name, qualified_name, return_type, parameters, parameter_names,
        visibility, is_abstract, is_strictfp, is_static, is_final,
        is_constructor, source_declaration
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		m.Name, m.QualifiedName, m.ReturnType,
		strings.Join(m.Parameters, ","),
		strings.Join(m.ParameterNames, ","),
		m.Visibility, m.IsAbstract, m.IsStrictfp,
		m.IsStatic, m.IsFinal, m.IsConstructor,
		m.SourceDeclaration)
	if err != nil {
		log.Printf("Failed to insert method: %v", err)
		return err
	}

	return nil
}

func (m *Method) Update(db *sql.DB) error {
	query := `UPDATE methods SET 
        qualified_name = ?, return_type = ?, parameters = ?,
        parameter_names = ?, visibility = ?, is_abstract = ?,
        is_strictfp = ?, is_static = ?, is_final = ?,
        is_constructor = ?, source_declaration = ?
        WHERE name = ?`

	_, err := db.Exec(query,
		m.QualifiedName, m.ReturnType,
		strings.Join(m.Parameters, ","),
		strings.Join(m.ParameterNames, ","),
		m.Visibility, m.IsAbstract, m.IsStrictfp,
		m.IsStatic, m.IsFinal, m.IsConstructor,
		m.SourceDeclaration, m.Name)
	return err
}

func (m *Method) Delete(db *sql.DB) error {
	query := `DELETE FROM methods WHERE name = ? AND qualified_name = ?`
	_, err := db.Exec(query, m.Name, m.QualifiedName)
	return err
}

// Query helper methods.
func FindMethodByName(db *sql.DB, name string) (*Method, error) {
	query := `SELECT * FROM methods WHERE name = ?`
	row := db.QueryRow(query, name)

	method := &Method{}
	var params, paramNames string

	err := row.Scan(&method.Name, &method.QualifiedName,
		&method.ReturnType, &params, &paramNames,
		&method.Visibility, &method.IsAbstract,
		&method.IsStrictfp, &method.IsStatic,
		&method.IsFinal, &method.IsConstructor,
		&method.SourceDeclaration)
	if err != nil {
		return nil, err
	}

	method.Parameters = strings.Split(params, ",")
	method.ParameterNames = strings.Split(paramNames, ",")
	return method, nil
}
