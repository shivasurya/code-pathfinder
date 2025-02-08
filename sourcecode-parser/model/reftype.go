package model

type RefType struct {
	// Precomputed from AST
	QualifiedName   string   // Fully qualified name (e.g., "java.lang.String")
	Package         string   // Package name (e.g., "java.lang")
	SourceFile      string   // Compilation unit (filename)
	TopLevel        bool     // Whether this is a top-level type
	SuperTypes      []string // Direct supertypes (extends, implements)
	DeclaredFields  []string // Fields declared in this type
	DeclaredMethods []Method // Methods declared in this type
	Constructors    []Method // Constructor declarations
	NestedTypes     []string // Types declared inside this type
	EnclosingType   string   // If this type is nested inside another type
	ArrayType       bool     // Whether this is an array type
	TypeDescriptor  string   // JVM Type Descriptor (e.g., "[I", "[Ljava/lang/String;")

	RuntimeResolver *TypeResolver
}

// TypeResolver handles runtime computation of type relationships.
type TypeResolver struct {
	TypeHierarchy map[string][]string // Supertype -> Subtype mappings
}

func NewRefType(qualifiedName, pkg, sourceFile string, topLevel bool, superTypes []string, fields []string, methods []Method, constructors []Method, nestedTypes []string, enclosingType string, arrayType bool, typeDescriptor string, resolver *TypeResolver) *RefType {
	return &RefType{
		QualifiedName:   qualifiedName,
		Package:         pkg,
		SourceFile:      sourceFile,
		TopLevel:        topLevel,
		SuperTypes:      superTypes,
		DeclaredFields:  fields,
		DeclaredMethods: methods,
		Constructors:    constructors,
		NestedTypes:     nestedTypes,
		EnclosingType:   enclosingType,
		ArrayType:       arrayType,
		TypeDescriptor:  typeDescriptor,
		RuntimeResolver: resolver,
	}
}

func (r *RefType) GetQualifiedName() string {
	return r.QualifiedName
}

// GetPackage returns the package where the type is declared.
func (r *RefType) GetPackage() string {
	return r.Package
}

// HasSupertype checks if the type has the given supertype.
func (r *RefType) HasSupertype(t string) bool {
	for _, super := range r.SuperTypes {
		if super == t {
			return true
		}
	}
	return false
}

// DeclaresField checks if the type declares a field with the given name.
func (r *RefType) DeclaresField(name string) bool {
	for _, field := range r.DeclaredFields {
		if field == name {
			return true
		}
	}
	return false
}

// DeclaresMethod checks if the type declares a method with the given name.
func (r *RefType) DeclaresMethod(name string) bool {
	for _, method := range r.DeclaredMethods {
		if method.Name == name {
			return true
		}
	}
	return false
}

// DeclaresMethodWithParams checks if the type declares a method with the given name and parameter count.
func (r *RefType) DeclaresMethodWithParams(name string, paramCount int) bool {
	for _, method := range r.DeclaredMethods {
		if method.Name == name && len(method.Parameters) == paramCount {
			return true
		}
	}
	return false
}

// Runtime Computed Methods

// GetASupertype retrieves the direct supertype (requires global analysis).
func (r *RefType) GetASupertype() []string {
	if r.RuntimeResolver == nil {
		return nil
	}
	return r.RuntimeResolver.ResolveSupertype(r.QualifiedName)
}

// GetASubtype retrieves direct subtypes (requires global analysis).
func (r *RefType) GetASubtype() []string {
	if r.RuntimeResolver == nil {
		return nil
	}
	return r.RuntimeResolver.ResolveSubtype(r.QualifiedName)
}

// HasMethod checks if the type has a method (including inherited methods).
func (r *RefType) HasMethod(name string) bool {
	// First check declared methods
	if r.DeclaresMethod(name) {
		return true
	}

	// Then check inherited methods
	for _, super := range r.GetASupertype() {
		if r.RuntimeResolver != nil && r.RuntimeResolver.HasMethod(super, name) {
			return true
		}
	}
	return false
}

// TypeResolver Implementation

// ResolveSupertype fetches direct supertypes.
func (tr *TypeResolver) ResolveSupertype(typename string) []string {
	if supertypes, ok := tr.TypeHierarchy[typename]; ok {
		return supertypes
	}
	return nil
}

// ResolveSubtype fetches direct subtypes.
func (tr *TypeResolver) ResolveSubtype(typename string) []string {
	var subtypes []string
	for parent, children := range tr.TypeHierarchy {
		for _, child := range children {
			if child == typename {
				subtypes = append(subtypes, parent)
			}
		}
	}
	return subtypes
}

// HasMethod checks if a method is inherited from a supertype.
func (tr *TypeResolver) HasMethod(typename, methodName string) bool {
	// For simplicity, assume a predefined method lookup (to be replaced by a full method table lookup)
	methods := map[string][]string{
		"java.lang.Object": {"toString", "hashCode", "equals"},
	}

	if methodsList, ok := methods[typename]; ok {
		for _, method := range methodsList {
			if method == methodName {
				return true
			}
		}
	}
	return false
}

// ClassOrInterface represents a Java class or interface extending RefType.
type ClassOrInterface struct {
	RefType
	// Java 17 Sealed Class Feature
	IsSealed          bool     // Whether this is a sealed class.
	PermittedSubtypes []string // Permitted subtypes (if sealed class).

	// Companion Object (for future Kotlin-style support)
	CompanionObject string // If this type has a companion object.

	// Accessibility and Visibility
	IsLocal            bool // Whether this class/interface is local.
	IsPackageProtected bool // Whether this class/interface has package-private visibility.
}

// NewClassOrInterface initializes a new ClassOrInterface instance.
func NewClassOrInterface(isSealed bool, permittedSubtypes []string, companionObject string, isLocal bool, isPackageProtected bool) *ClassOrInterface {
	return &ClassOrInterface{
		IsSealed:           isSealed,
		PermittedSubtypes:  permittedSubtypes,
		CompanionObject:    companionObject,
		IsLocal:            isLocal,
		IsPackageProtected: isPackageProtected,
	}
}

// ✅ Implementing Only the Provided Predicates for ClassOrInterface

// GetAPermittedSubtype returns the permitted subtypes if this is a sealed class.
func (c *ClassOrInterface) GetAPermittedSubtype() []string {
	if c.IsSealed {
		return c.PermittedSubtypes
	}
	return nil
}

// GetCompanionObject returns the companion object of this class/interface, if any.
func (c *ClassOrInterface) GetCompanionObject() string {
	return c.CompanionObject
}

// IsSealed checks whether this is a sealed class (Java 17 feature).
func (c *ClassOrInterface) GetIsSealed() bool {
	return c.IsSealed
}

// IsLocal checks whether this class/interface is a local class.
func (c *ClassOrInterface) GetIsLocal() bool {
	return c.IsLocal
}

// IsPackageProtected checks whether this class/interface has package-private visibility.
func (c *ClassOrInterface) GetIsPackageProtected() bool {
	return c.IsPackageProtected
}

// Class represents a Java class extending ClassOrInterface.
type Class struct {
	ClassOrInterface

	// CodeQL metadata
	PrimaryQlClass string   // Name of the primary CodeQL class
	Annotations    []string // Annotations applied to this class

	// Class type properties
	IsAnonymous bool // Whether this is an anonymous class
	IsFileClass bool // Whether this is a Kotlin file class (e.g., FooKt for Foo.kt)
}

// NewClass initializes a new Class instance.
func NewClass(primaryQlClass string, annotations []string, isAnonymous bool, isFileClass bool, classOrInterface ClassOrInterface) *Class {
	return &Class{
		ClassOrInterface: classOrInterface,
		PrimaryQlClass:   primaryQlClass,
		Annotations:      annotations,
		IsAnonymous:      isAnonymous,
		IsFileClass:      isFileClass,
	}
}

// ✅ Implementing Only the Provided Predicates for Class

// GetAPrimaryQlClass returns the primary CodeQL class name.
func (c *Class) GetAPrimaryQlClass() string {
	return c.PrimaryQlClass
}

// GetAnAnnotation returns the annotations applied to this class.
func (c *Class) GetAnAnnotation() []string {
	return c.Annotations
}

// IsAnonymous checks whether this is an anonymous class.
func (c *Class) GetIsAnonymous() bool {
	return c.IsAnonymous
}

// IsFileClass checks whether this is a Kotlin file class.
func (c *Class) GetIsFileClass() bool {
	return c.IsFileClass
}
