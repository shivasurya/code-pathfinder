package model

import (
	"database/sql"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type ExprParent struct{}

func (e *ExprParent) GetAChildExpr() *Expr {
	return nil
}

func (e *ExprParent) GetChildExpr(_ int) *Expr {
	return nil
}

func (e *ExprParent) GetNumChildExpr() int64 {
	return 0
}

func (e *ExprParent) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{}
}

type Expr struct {
	ExprParent
	Kind       int
	Node       sitter.Node
	NodeString string
	Type       string
}

func (e *Expr) String() string {
	return fmt.Sprintf("Expr(%s)", e.NodeString)
}

func (e *Expr) GetAChildExpr() *Expr {
	return e
}

func (e *Expr) GetChildExpr(_ int) *Expr {
	return e
}

func (e *Expr) GetNumChildExpr() int64 {
	return 1
}

func (e *Expr) GetBoolValue() {
}

func (e *Expr) GetKind() int {
	return e.Kind
}

func (e *Expr) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetID":   e.Node.ID(),
		"GetType": e.Type,
	}
}

type BinaryExpr struct {
	Expr
	Op                string
	LeftOperand       *Expr
	RightOperand      *Expr
	SourceDeclaration string
}

func (e *BinaryExpr) Insert(db *sql.DB) error {
	query := "INSERT INTO binary_expr (left_operand, right_operand, operator, source_declaration) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(query, e.GetLeftOperandString(), e.GetRightOperandString(), e.Op, e.SourceDeclaration)
	if err != nil {
		fmt.Println("Error inserting binary expression:", err)
		return err
	}
	return nil
}

func (e *BinaryExpr) GetLeftOperand() *Expr {
	return e.LeftOperand
}

func (e *BinaryExpr) GetLeftOperandString() string {
	return e.LeftOperand.NodeString
}

func (e *BinaryExpr) GetRightOperand() *Expr {
	return e.RightOperand
}

func (e *BinaryExpr) GetRightOperandString() string {
	return e.RightOperand.NodeString
}

func (e *BinaryExpr) GetOp() string {
	return e.Op
}

func (e *BinaryExpr) GetKind() int {
	return 1
}

func (e *BinaryExpr) ToString() string {
	return fmt.Sprintf("BinaryExpr(%s, %v, %v)", e.Op, e.LeftOperand, e.RightOperand)
}

func (e *BinaryExpr) GetAnOperand() *Expr {
	if e.LeftOperand != nil {
		return e.LeftOperand
	}
	return e.RightOperand
}

func (e *BinaryExpr) HasOperands(expr1, expr2 *Expr) bool {
	return e.LeftOperand == expr1 && e.RightOperand == expr2
}

func (e *BinaryExpr) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetLeftOperand":  e.LeftOperand,
		"GetRightOperand": e.RightOperand,
		"GetOp":           e.GetOp(),
	}
}

type AddExpr struct {
	BinaryExpr
	op string
}

func (e *AddExpr) GetOp() string {
	return e.op
}

type AndBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *AndBitwiseExpr) GetOp() string {
	return e.op
}

type ComparisonExpr struct {
	BinaryExpr
}

func (e *ComparisonExpr) GetGreaterThanOperand() *Expr {
	return nil
}

func (e *ComparisonExpr) GetLessThanOperand() *Expr {
	return nil
}

func (e *ComparisonExpr) IsStrict() bool {
	return true
}

type AndLogicalExpr struct {
	BinaryExpr
	op string
}

func (e *AndLogicalExpr) GetOp() string {
	return e.op
}

type DivExpr struct {
	BinaryExpr
	op string
}

func (e *DivExpr) GetOp() string {
	return e.op
}

type EqExpr struct {
	BinaryExpr
	op string
}

func (e *EqExpr) GetOp() string {
	return e.op
}

type GEExpr struct {
	BinaryExpr
	op string
}

func (e *GEExpr) GetOp() string {
	return e.op
}

type GTExpr struct {
	BinaryExpr
	op string
}

func (e *GTExpr) GetOp() string {
	return e.op
}

type LEExpr struct {
	BinaryExpr
	op string
}

func (e *LEExpr) GetOp() string {
	return e.op
}

type LTExpr struct {
	BinaryExpr
	op string
}

func (e *LTExpr) GetOp() string {
	return e.op
}

type NEExpr struct {
	BinaryExpr
	op string
}

func (e *NEExpr) GetOp() string {
	return e.op
}

type LeftShiftExpr struct {
	BinaryExpr
	op string
}

func (e *LeftShiftExpr) GetOp() string {
	return e.op
}

type MulExpr struct {
	BinaryExpr
	op string
}

func (e *MulExpr) GetOp() string {
	return e.op
}

type OrBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *OrBitwiseExpr) GetOp() string {
	return e.op
}

type OrLogicalExpr struct {
	BinaryExpr
	op string
}

func (e *OrLogicalExpr) GetOp() string {
	return e.op
}

type RemExpr struct {
	BinaryExpr
	op string
}

func (e *RemExpr) GetOp() string {
	return e.op
}

type RightShiftExpr struct {
	BinaryExpr
	op string
}

func (e *RightShiftExpr) GetOp() string {
	return e.op
}

type SubExpr struct {
	BinaryExpr
	op string
}

func (e *SubExpr) GetOp() string {
	return e.op
}

type UnsignedRightShiftExpr struct {
	BinaryExpr
	op string
}

func (e *UnsignedRightShiftExpr) GetOp() string {
	return e.op
}

type XorBitwiseExpr struct {
	BinaryExpr
	op string
}

func (e *XorBitwiseExpr) GetOp() string {
	return e.op
}

type ClassInstanceExpr struct {
	Expr
	ClassName string
	Args      []*Expr
}

func (e *ClassInstanceExpr) GetClassName() string {
	return e.ClassName
}

func (e *ClassInstanceExpr) GetArgs() []*Expr {
	return e.Args
}

func (e *ClassInstanceExpr) GetArg(i int) *Expr {
	return e.Args[i]
}

func (e *ClassInstanceExpr) GetNumArgs() int {
	return len(e.Args)
}

func (e *ClassInstanceExpr) String() string {
	return fmt.Sprintf("ClassInstanceExpr(%s, %v)", e.ClassName, e.Args)
}

func (e *ClassInstanceExpr) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetClassName": e.ClassName,
		"GetArgs":      e.Args,
		"GetNumArgs":   len(e.Args),
		"GetArg":       e.GetArg,
	}
}

// Annotation represents a Java annotation applied to language elements.
type Annotation struct {
	Expr
	QualifiedName    string         // Fully qualified name of the annotation (e.g., "javax.persistence.Entity")
	AnnotatedElement string         // The element this annotation applies to
	AnnotationType   string         // The type of this annotation
	Values           map[string]any // Stores annotation elements and their values
	IsDeclAnnotation bool           // Whether this annotation applies to a declaration
	IsTypeAnnotation bool           // Whether this annotation applies to a type
	HalsteadID       string         // Placeholder for Halstead metric computation
}

// NewAnnotation initializes a new Annotation instance.
func NewAnnotation(qualifiedName, annotatedElement, annotationType string, values map[string]any, isDeclAnnotation, isTypeAnnotation bool, halsteadID string) *Annotation {
	return &Annotation{
		QualifiedName:    qualifiedName,
		AnnotatedElement: annotatedElement,
		AnnotationType:   annotationType,
		Values:           values,
		IsDeclAnnotation: isDeclAnnotation,
		IsTypeAnnotation: isTypeAnnotation,
		HalsteadID:       halsteadID,
	}
}

// ✅ Implementing Only the Provided Predicates for Annotation

// GetAPrimaryQlClass returns the primary CodeQL class name for this annotation.
func (a *Annotation) GetAPrimaryQlClass() string {
	return "Annotation"
}

// GetAStringArrayValue retrieves a string array value from the annotation.
func (a *Annotation) GetAStringArrayValue(name string) []string {
	if val, ok := a.Values[name].([]string); ok {
		return val
	}
	return nil
}

// GetATypeArrayValue retrieves a Class array value from the annotation.
func (a *Annotation) GetATypeArrayValue(name string) []string {
	if val, ok := a.Values[name].([]string); ok {
		return val
	}
	return nil
}

// GetAnArrayValue retrieves an array value from the annotation.
func (a *Annotation) GetAnArrayValue(name string) any {
	if val, ok := a.Values[name]; ok {
		return val
	}
	return nil
}

// GetAnEnumConstantArrayValue retrieves an enum array value from the annotation.
func (a *Annotation) GetAnEnumConstantArrayValue(name string) []string {
	if val, ok := a.Values[name].([]string); ok {
		return val
	}
	return nil
}

// GetAnIntArrayValue retrieves an int array value from the annotation.
func (a *Annotation) GetAnIntArrayValue(name string) []int {
	if val, ok := a.Values[name].([]int); ok {
		return val
	}
	return nil
}

// GetAnnotatedElement returns the element being annotated.
func (a *Annotation) GetAnnotatedElement() string {
	return a.AnnotatedElement
}

// GetAnnotationElement retrieves the annotation element with the specified name.
func (a *Annotation) GetAnnotationElement(name string) any {
	if val, ok := a.Values[name]; ok {
		return val
	}
	return nil
}

// GetArrayValue retrieves a specific index value from an annotation array.
func (a *Annotation) GetArrayValue(name string, index int) any {
	if val, ok := a.Values[name].([]any); ok && index < len(val) {
		return val[index]
	}
	return nil
}

// GetBooleanValue retrieves a boolean value from the annotation.
func (a *Annotation) GetBooleanValue(name string) bool {
	if val, ok := a.Values[name].(bool); ok {
		return val
	}
	return false
}

// GetEnumConstantValue retrieves an enum constant value from the annotation.
func (a *Annotation) GetEnumConstantValue(name string) string {
	if val, ok := a.Values[name].(string); ok {
		return val
	}
	return ""
}

// GetHalsteadID returns the Halstead metric ID for this annotation.
func (a *Annotation) GetHalsteadID() string {
	return a.HalsteadID
}

// GetIntValue retrieves an integer value from the annotation.
func (a *Annotation) GetIntValue(name string) int {
	if val, ok := a.Values[name].(int); ok {
		return val
	}
	return 0
}

// GetStringValue retrieves a string value from the annotation.
func (a *Annotation) GetStringValue(name string) string {
	if val, ok := a.Values[name].(string); ok {
		return val
	}
	return ""
}

// GetTarget returns the element being annotated.
func (a *Annotation) GetTarget() string {
	return a.AnnotatedElement
}

// GetType returns the annotation type declaration.
func (a *Annotation) GetType() string {
	return a.AnnotationType
}

// GetTypeValue retrieves a `java.lang.Class` reference value from the annotation.
func (a *Annotation) GetTypeValue(name string) string {
	if val, ok := a.Values[name].(string); ok {
		return val
	}
	return ""
}

// GetValue retrieves any value of an annotation element.
func (a *Annotation) GetValue(name string) any {
	if val, ok := a.Values[name]; ok {
		return val
	}
	return nil
}

// IsDeclAnnotation checks whether this annotation applies to a declaration.
func (a *Annotation) GetIsDeclAnnotation() bool {
	return a.IsDeclAnnotation
}

// IsTypeAnnotation checks whether this annotation applies to a type.
func (a *Annotation) GetIsTypeAnnotation() bool {
	return a.IsTypeAnnotation
}

// ToString returns a textual representation of the annotation.
func (a *Annotation) ToString() string {
	return "@" + a.QualifiedName
}

func (a *Annotation) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetQualifiedName":    a.QualifiedName,
		"GetAnnotatedElement": a.AnnotatedElement,
		"GetAnnotationType":   a.AnnotationType,
		"GetValues":           a.Values,
		"GetIsDeclAnnotation": a.IsDeclAnnotation,
		"GetIsTypeAnnotation": a.IsTypeAnnotation,
	}
}

// MethodCall represents an invocation of a method with arguments.
type MethodCall struct {
	PrimaryQlClass    string   // Primary CodeQL class name
	MethodName        string   // The method being called
	QualifiedMethod   string   // Fully qualified method name
	Arguments         []string // List of arguments passed to the method
	TypeArguments     []string // Type arguments for generic method calls
	Qualifier         string   // The qualifying expression of the method call (e.g., obj in obj.method())
	ReceiverType      string   // The type of the qualifier or the enclosing type if none
	EnclosingCallable string   // The method or function containing this method call
	EnclosingStmt     string   // The statement enclosing this method call
	HasQualifier      bool     // Whether this call has a qualifier
	IsEnclosingCall   bool     // Whether this is a call to an instance method of the enclosing class
	IsOwnMethodCall   bool     // Whether this is a call to an instance method of 'this'
}

func (m *MethodCall) Insert(db *sql.DB) error {
	query := `INSERT INTO method_call (method_name, qualified_name, parameters, parameters_names) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, m.MethodName, m.QualifiedMethod, strings.Join(m.Arguments, ","), strings.Join(m.TypeArguments, ","))
	if err != nil {
		fmt.Println("Error inserting method call:", err)
		return err
	}
	return nil
}

// NewMethodCall initializes a new MethodCall instance.
func NewMethodCall(primaryQlClass, methodName, qualifiedMethod string, arguments, typeArguments []string, qualifier, receiverType, enclosingCallable, enclosingStmt string, hasQualifier, isEnclosingCall, isOwnMethodCall bool) *MethodCall {
	return &MethodCall{
		PrimaryQlClass:    primaryQlClass,
		MethodName:        methodName,
		QualifiedMethod:   qualifiedMethod,
		Arguments:         arguments,
		TypeArguments:     typeArguments,
		Qualifier:         qualifier,
		ReceiverType:      receiverType,
		EnclosingCallable: enclosingCallable,
		EnclosingStmt:     enclosingStmt,
		HasQualifier:      hasQualifier,
		IsEnclosingCall:   isEnclosingCall,
		IsOwnMethodCall:   isOwnMethodCall,
	}
}

// ✅ Implementing the Predicates for `MethodCall`

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (m *MethodCall) GetAPrimaryQlClass() string {
	return m.PrimaryQlClass
}

// GetATypeArgument retrieves a type argument in this method call, if any.
func (m *MethodCall) GetATypeArgument() []string {
	return m.TypeArguments
}

// GetAnArgument retrieves all arguments supplied to this method call.
func (m *MethodCall) GetAnArgument() []string {
	return m.Arguments
}

// GetArgument retrieves an argument at the specified index.
func (m *MethodCall) GetArgument(index int) string {
	if index >= 0 && index < len(m.Arguments) {
		return m.Arguments[index]
	}
	return ""
}

// GetEnclosingCallable retrieves the callable that contains this method call.
func (m *MethodCall) GetEnclosingCallable() string {
	return m.EnclosingCallable
}

// GetEnclosingStmt retrieves the statement that contains this method call.
func (m *MethodCall) GetEnclosingStmt() string {
	return m.EnclosingStmt
}

// GetMethod retrieves the fully qualified name of the method being called.
func (m *MethodCall) GetMethod() string {
	return m.QualifiedMethod
}

// GetQualifier retrieves the qualifier of the method call, if any.
func (m *MethodCall) GetQualifier() string {
	return m.Qualifier
}

// GetReceiverType retrieves the receiver type of the method call.
func (m *MethodCall) GetReceiverType() string {
	return m.ReceiverType
}

// GetTypeArgument retrieves a specific type argument at the specified index.
func (m *MethodCall) GetTypeArgument(index int) string {
	if index >= 0 && index < len(m.TypeArguments) {
		return m.TypeArguments[index]
	}
	return ""
}

// HasQualifier checks if the method call has a qualifier.
func (m *MethodCall) GetHasQualifier() bool {
	return m.HasQualifier
}

// IsEnclosingMethodCall checks if this is a call to an instance method of the enclosing class.
func (m *MethodCall) GetIsEnclosingMethodCall() bool {
	return m.IsEnclosingCall
}

// IsOwnMethodCall checks if this is a call to an instance method of `this`.
func (m *MethodCall) GetIsOwnMethodCall() bool {
	return m.IsOwnMethodCall
}

// PrintAccess returns a printable representation of the method call.
func (m *MethodCall) PrintAccess() string {
	if m.HasQualifier {
		return fmt.Sprintf("%s.%s(%v)", m.Qualifier, m.MethodName, m.Arguments)
	}
	return fmt.Sprintf("%s(%v)", m.MethodName, m.Arguments)
}

// ToString returns a textual representation of the method call.
func (m *MethodCall) ToString() string {
	return m.PrintAccess()
}

func (m *MethodCall) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetMethodName":        m.MethodName,
		"GetQualifiedMethod":   m.QualifiedMethod,
		"GetArguments":         m.Arguments,
		"GetTypeArguments":     m.TypeArguments,
		"GetQualifier":         m.Qualifier,
		"GetReceiverType":      m.ReceiverType,
		"GetEnclosingCallable": m.EnclosingCallable,
		"GetEnclosingStmt":     m.EnclosingStmt,
	}
}

// FieldDeclaration represents a declaration of one or more fields in a class.
type FieldDeclaration struct {
	ExprParent
	Type              string   // Type of the field (e.g., int, String)
	FieldNames        []string // Names of the fields declared in this statement
	Visibility        string   // Visibility (public, private, protected, package-private)
	IsStatic          bool     // Whether the field is static
	IsFinal           bool     // Whether the field is final
	IsVolatile        bool     // Whether the field is volatile
	IsTransient       bool     // Whether the field is transient
	SourceDeclaration string   // Location of the field declaration
}

func (f *FieldDeclaration) Insert(db *sql.DB) error {
	query := `
		INSERT INTO field_decl (field_name, type, visibility, is_static, is_final, is_transient, is_volatile, source_declaration)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	for _, fieldName := range f.FieldNames {
		_, err := db.Exec(query, fieldName, f.Type, f.Visibility, f.IsStatic, f.IsFinal, f.IsTransient, f.IsVolatile, f.SourceDeclaration)
		if err != nil {
			fmt.Println("Error inserting field:", err)
			return err
		}
	}
	return nil
}

// NewFieldDeclaration initializes a new FieldDeclaration instance.
func NewFieldDeclaration(fieldType string, fieldNames []string, visibility string, isStatic, isFinal, isVolatile, isTransient bool, sourceDeclaration string) *FieldDeclaration {
	return &FieldDeclaration{
		Type:              fieldType,
		FieldNames:        fieldNames,
		Visibility:        visibility,
		IsStatic:          isStatic,
		IsFinal:           isFinal,
		IsVolatile:        isVolatile,
		IsTransient:       isTransient,
		SourceDeclaration: sourceDeclaration,
	}
}

// ✅ Implementing AST-Based Predicates

// GetAField retrieves all fields declared in this field declaration.
func (f *FieldDeclaration) GetAField() []string {
	return f.FieldNames
}

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (f *FieldDeclaration) GetAPrimaryQlClass() string {
	return "FieldDeclaration"
}

// GetField retrieves the field declared at the specified index.
func (f *FieldDeclaration) GetField(index int) string {
	if index >= 0 && index < len(f.FieldNames) {
		return f.FieldNames[index]
	}
	return ""
}

// GetNumField returns the number of fields declared in this declaration.
func (f *FieldDeclaration) GetNumField() int {
	return len(f.FieldNames)
}

// GetTypeAccess retrieves the type of the field(s) in this declaration.
func (f *FieldDeclaration) GetTypeAccess() string {
	return f.Type
}

// ToString returns a textual representation of the field declaration.
func (f *FieldDeclaration) ToString() string {
	modifiers := []string{}
	if f.Visibility != "" {
		modifiers = append(modifiers, f.Visibility)
	}
	if f.IsStatic {
		modifiers = append(modifiers, "static")
	}
	if f.IsFinal {
		modifiers = append(modifiers, "final")
	}
	if f.IsVolatile {
		modifiers = append(modifiers, "volatile")
	}
	if f.IsTransient {
		modifiers = append(modifiers, "transient")
	}

	return fmt.Sprintf("%s %s %s;", strings.Join(modifiers, " "), f.Type, strings.Join(f.FieldNames, ", "))
}

func (f *FieldDeclaration) GetProxyEnv() map[string]interface{} {
	return map[string]interface{}{
		"GetFieldNames":        f.FieldNames,
		"GetVisibility":        f.Visibility,
		"GetType":              f.Type,
		"GetIsStatic":          f.IsStatic,
		"GetIsFinal":           f.IsFinal,
		"GetIsVolatile":        f.IsVolatile,
		"GetIsTransient":       f.IsTransient,
		"GetSourceDeclaration": f.SourceDeclaration,
	}
}
