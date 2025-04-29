package model

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
)

func TestBinaryExpr(t *testing.T) {
	leftExpr := &Expr{Kind: 0, NodeString: "left"}
	rightExpr := &Expr{Kind: 0, NodeString: "right"}
	binaryExpr := &BinaryExpr{
		Op:           "+",
		LeftOperand:  leftExpr,
		RightOperand: rightExpr,
	}

	t.Run("GetLeftOperand", func(t *testing.T) {
		assert.Equal(t, leftExpr, binaryExpr.GetLeftOperand())
	})

	t.Run("GetRightOperand", func(t *testing.T) {
		assert.Equal(t, rightExpr, binaryExpr.GetRightOperand())
	})

	t.Run("GetOp", func(t *testing.T) {
		assert.Equal(t, "+", binaryExpr.GetOp())
	})

	t.Run("GetKind", func(t *testing.T) {
		assert.Equal(t, 1, binaryExpr.GetKind())
	})

	t.Run("GetAnOperand", func(t *testing.T) {
		assert.Equal(t, leftExpr, binaryExpr.GetAnOperand())
	})

	t.Run("HasOperands", func(t *testing.T) {
		assert.True(t, binaryExpr.HasOperands(leftExpr, rightExpr))
		assert.False(t, binaryExpr.HasOperands(rightExpr, leftExpr))
	})

	t.Run("GetLeftOperandString", func(t *testing.T) {
		assert.Equal(t, "left", binaryExpr.GetLeftOperandString())
	})

	t.Run("GetRightOperandString", func(t *testing.T) {
		assert.Equal(t, "right", binaryExpr.GetRightOperandString())
	})

	t.Run("ToString", func(t *testing.T) {
		str := binaryExpr.ToString()
		assert.Contains(t, str, "BinaryExpr(")
		assert.Contains(t, str, "+")
		assert.Contains(t, str, "left")
		assert.Contains(t, str, "right")
	})
}

func TestAddExpr(t *testing.T) {
	addExpr := &AddExpr{
		BinaryExpr: BinaryExpr{Op: "+"},
		op:         "+",
	}

	assert.Equal(t, "+", addExpr.GetOp())
}

func TestOtherBinaryExprTypes_GetOp(t *testing.T) {
	types := []struct {
		name     string
		expr     interface{ GetOp() string }
		expected string
	}{
		{"SubExpr", &SubExpr{op: "-"}, "-"},
		{"DivExpr", &DivExpr{op: "/"}, "/"},
		{"MulExpr", &MulExpr{op: "*"}, "*"},
		{"RemExpr", &RemExpr{op: "%"}, "%"},
		{"EqExpr", &EqExpr{op: "=="}, "=="},
		{"NEExpr", &NEExpr{op: "!="}, "!="},
		{"GTExpr", &GTExpr{op: ">"}, ">"},
		{"GEExpr", &GEExpr{op: ">="}, ">="},
		{"LTExpr", &LTExpr{op: "<"}, "<"},
		{"LEExpr", &LEExpr{op: "<="}, "<="},
		{"AndBitwiseExpr", &AndBitwiseExpr{op: "&"}, "&"},
		{"OrBitwiseExpr", &OrBitwiseExpr{op: "|"}, "|"},
		{"LeftShiftExpr", &LeftShiftExpr{op: "<<"}, "<<"},
		{"RightShiftExpr", &RightShiftExpr{op: ">>"}, ">>"},
		{"UnsignedRightShiftExpr", &UnsignedRightShiftExpr{op: ">>>"}, ">>>"},
		{"AndLogicalExpr", &AndLogicalExpr{op: "&&"}, "&&"},
		{"OrLogicalExpr", &OrLogicalExpr{op: "||"}, "||"},
	}
	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.GetOp())
		})
	}
}

func TestComparisonExpr(t *testing.T) {
	compExpr := &ComparisonExpr{}

	assert.Nil(t, compExpr.GetGreaterThanOperand())
	assert.Nil(t, compExpr.GetLessThanOperand())
	assert.True(t, compExpr.IsStrict())
}

func TestExpr(t *testing.T) {
	expr := &Expr{Kind: 42, NodeString: "foo"}

	t.Run("GetAChildExpr", func(t *testing.T) {
		assert.Equal(t, expr, expr.GetAChildExpr())
	})

	t.Run("GetChildExpr", func(t *testing.T) {
		assert.Equal(t, expr, expr.GetChildExpr(0))
	})

	t.Run("GetNumChildExpr", func(t *testing.T) {
		assert.Equal(t, int64(1), expr.GetNumChildExpr())
	})

	t.Run("GetKind", func(t *testing.T) {
		assert.Equal(t, 42, expr.GetKind())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "Expr(foo)", expr.String())
	})
}

func TestExprParent(t *testing.T) {
	parent := &ExprParent{}

	assert.Nil(t, parent.GetAChildExpr())
	assert.Nil(t, parent.GetChildExpr(0))
	assert.Equal(t, int64(0), parent.GetNumChildExpr())
}

func TestClassInstanceExpr(t *testing.T) {
	t.Run("GetClassName", func(t *testing.T) {
		testCases := []struct {
			name     string
			expr     *ClassInstanceExpr
			expected string
		}{
			{"Normal class name", &ClassInstanceExpr{ClassName: "MyClass"}, "MyClass"},
			{"Empty class name", &ClassInstanceExpr{ClassName: ""}, ""},
			{"Class name with special characters", &ClassInstanceExpr{ClassName: "My_Class$123"}, "My_Class$123"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.expr.GetClassName()
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestAnnotation(t *testing.T) {
	values := map[string]any{
		"strArray":    []string{"test1", "test2"},
		"typeArray":   []string{"String", "Integer"},
		"mixedArray":  []any{"test", 1, true},
		"boolValue":   true,
		"enumValue":   "ENUM_VAL",
		"intValue":    42,
		"stringValue": "hello",
		"classValue":  "java.lang.String",
	}

	annotation := NewAnnotation(
		"com.example.TestAnnotation",
		"TestClass",
		"TestType",
		values,
		true,
		false,
		"halstead123",
	)

	t.Run("Constructor and basic getters", func(t *testing.T) {
		assert.Equal(t, "com.example.TestAnnotation", annotation.QualifiedName)
		assert.Equal(t, "TestClass", annotation.AnnotatedElement)
		assert.Equal(t, "TestType", annotation.AnnotationType)
		assert.True(t, annotation.IsDeclAnnotation)
		assert.False(t, annotation.IsTypeAnnotation)
		assert.Equal(t, "halstead123", annotation.HalsteadID)
	})

	t.Run("Array value getters", func(t *testing.T) {
		assert.Equal(t, []string{"test1", "test2"}, annotation.GetAStringArrayValue("strArray"))
		assert.Equal(t, []string{"String", "Integer"}, annotation.GetATypeArrayValue("typeArray"))
		assert.Equal(t, []any{"test", 1, true}, annotation.GetAnArrayValue("mixedArray"))
		assert.Equal(t, "test1", annotation.GetArrayValue("strArray", 0))
		assert.Nil(t, annotation.GetArrayValue("nonexistent", 0))
		assert.Nil(t, annotation.GetArrayValue("strArray", 99))
	})

	t.Run("Primitive value getters", func(t *testing.T) {
		assert.True(t, annotation.GetBooleanValue("boolValue"))
		assert.False(t, annotation.GetBooleanValue("nonexistent"))
		assert.Equal(t, "ENUM_VAL", annotation.GetEnumConstantValue("enumValue"))
		assert.Equal(t, "", annotation.GetEnumConstantValue("nonexistent"))
		assert.Equal(t, 42, annotation.GetIntValue("intValue"))
		assert.Equal(t, 0, annotation.GetIntValue("nonexistent"))
		assert.Equal(t, "hello", annotation.GetStringValue("stringValue"))
		assert.Equal(t, "", annotation.GetStringValue("nonexistent"))
		assert.Equal(t, "java.lang.String", annotation.GetTypeValue("classValue"))
		assert.Equal(t, "", annotation.GetTypeValue("nonexistent"))
	})

	t.Run("General methods", func(t *testing.T) {
		assert.Equal(t, "Annotation", annotation.GetAPrimaryQlClass())
		assert.Equal(t, "TestClass", annotation.GetAnnotatedElement())
		assert.Equal(t, values["boolValue"], annotation.GetAnnotationElement("boolValue"))
		assert.Equal(t, "halstead123", annotation.GetHalsteadID())
		assert.Equal(t, "TestClass", annotation.GetTarget())
		assert.Equal(t, "TestType", annotation.GetType())
		assert.Equal(t, values["stringValue"], annotation.GetValue("stringValue"))
		assert.True(t, annotation.GetIsDeclAnnotation())
		assert.False(t, annotation.GetIsTypeAnnotation())
		assert.Equal(t, "@com.example.TestAnnotation", annotation.ToString())
	})

	t.Run("GetProxyEnv", func(t *testing.T) {
		proxyEnv := annotation.GetProxyEnv()
		assert.Equal(t, "com.example.TestAnnotation", proxyEnv["GetQualifiedName"])
		assert.Equal(t, "TestClass", proxyEnv["GetAnnotatedElement"])
		assert.Equal(t, "TestType", proxyEnv["GetAnnotationType"])
		assert.Equal(t, values, proxyEnv["GetValues"])
		assert.Equal(t, true, proxyEnv["GetIsDeclAnnotation"])
		assert.Equal(t, false, proxyEnv["GetIsTypeAnnotation"])
	})
}

func TestMethodCall(t *testing.T) {
	methodCall := NewMethodCall(
		"MethodCall",
		"testMethod",
		"com.example.TestClass.testMethod",
		[]string{"arg1", "arg2"},
		[]string{"String", "Integer"},
		"com.example.TestClass",
		"receiverType",
		"enclosingMethod",
		"enclosingStmt",
		true, // hasQualifier
		true, // isEnclosingCall
		true, // isOwnMethodCall
	)

	t.Run("Constructor and basic getters", func(t *testing.T) {
		assert.Equal(t, "testMethod", methodCall.MethodName)
		assert.Equal(t, "com.example.TestClass.testMethod", methodCall.QualifiedMethod)
		assert.Equal(t, []string{"arg1", "arg2"}, methodCall.Arguments)
		assert.Equal(t, []string{"String", "Integer"}, methodCall.TypeArguments)
		assert.Equal(t, "com.example.TestClass", methodCall.Qualifier)
		assert.Equal(t, "enclosingMethod", methodCall.EnclosingCallable)
		assert.Equal(t, "enclosingStmt", methodCall.EnclosingStmt)
		assert.Equal(t, "receiverType", methodCall.ReceiverType)
		assert.True(t, methodCall.HasQualifier)
	})

	t.Run("Method related getters", func(t *testing.T) {
		assert.Equal(t, "com.example.TestClass.testMethod", methodCall.GetMethod())
		assert.Equal(t, "com.example.TestClass", methodCall.GetQualifier())
		assert.Equal(t, "receiverType", methodCall.GetReceiverType())
		assert.True(t, methodCall.GetHasQualifier())
		assert.Equal(t, []string{"arg1", "arg2"}, methodCall.GetAnArgument())
		assert.Equal(t, "arg1", methodCall.GetArgument(0))
		assert.Equal(t, "", methodCall.GetArgument(99))
		assert.Equal(t, []string{"String", "Integer"}, methodCall.GetATypeArgument())
		assert.Equal(t, "String", methodCall.GetTypeArgument(0))
		assert.Equal(t, "", methodCall.GetTypeArgument(99))
	})

	t.Run("Enclosing related methods", func(t *testing.T) {
		assert.Equal(t, "enclosingMethod", methodCall.GetEnclosingCallable())
		assert.Equal(t, "enclosingStmt", methodCall.GetEnclosingStmt())
		assert.True(t, methodCall.GetHasQualifier())
		assert.True(t, methodCall.GetIsOwnMethodCall())
	})

	t.Run("String representations", func(t *testing.T) {
		assert.Contains(t, methodCall.PrintAccess(), "com.example.TestClass.testMethod")
		assert.Contains(t, methodCall.ToString(), "com.example.TestClass.testMethod")
	})

	t.Run("GetProxyEnv", func(t *testing.T) {
		proxyEnv := methodCall.GetProxyEnv()
		assert.Equal(t, methodCall.QualifiedMethod, proxyEnv["GetMethod"])
		assert.Equal(t, methodCall.Qualifier, proxyEnv["GetQualifier"])
		assert.Equal(t, methodCall.Arguments, proxyEnv["GetArguments"])
		assert.Equal(t, methodCall.TypeArguments, proxyEnv["GetTypeArguments"])
		assert.Equal(t, methodCall.EnclosingCallable, proxyEnv["GetEnclosingCallable"])
		assert.Equal(t, methodCall.EnclosingStmt, proxyEnv["GetEnclosingStmt"])
		assert.Equal(t, methodCall.HasQualifier, proxyEnv["GetHasQualifier"])
		assert.Equal(t, methodCall.IsOwnMethodCall, proxyEnv["GetIsOwnMethodCall"])
	})
}

func TestFieldDeclaration(t *testing.T) {
	fieldDecl := NewFieldDeclaration(
		"String",
		[]string{"field1", "field2"},
		"private",
		true,
		true,
		true,
		false,
		"Test.java",
	)

	t.Run("Constructor and basic getters", func(t *testing.T) {
		assert.Equal(t, "String", fieldDecl.Type)
		assert.Equal(t, []string{"field1", "field2"}, fieldDecl.FieldNames)
		assert.Equal(t, "private", fieldDecl.Visibility)
		assert.True(t, fieldDecl.IsStatic)
		assert.True(t, fieldDecl.IsFinal)
		assert.True(t, fieldDecl.IsVolatile)
		assert.False(t, fieldDecl.IsTransient)
		assert.Equal(t, "Test.java", fieldDecl.SourceDeclaration)
	})

	t.Run("Field related getters", func(t *testing.T) {
		assert.Equal(t, []string{"field1", "field2"}, fieldDecl.GetAField())
		assert.Equal(t, "field1", fieldDecl.GetField(0))
		assert.Equal(t, "", fieldDecl.GetField(99))
		assert.Equal(t, 2, fieldDecl.GetNumField())
		assert.Equal(t, "String", fieldDecl.GetTypeAccess())
	})

	t.Run("Class and string methods", func(t *testing.T) {
		assert.Equal(t, "FieldDeclaration", fieldDecl.GetAPrimaryQlClass())
		assert.Contains(t, fieldDecl.ToString(), "String")
		assert.Contains(t, fieldDecl.ToString(), "field1")
	})

	t.Run("GetProxyEnv", func(t *testing.T) {
		proxyEnv := fieldDecl.GetProxyEnv()
		assert.Equal(t, fieldDecl.Type, proxyEnv["GetTypeAccess"])
		assert.Equal(t, fieldDecl.FieldNames, proxyEnv["GetAField"])
		assert.Equal(t, fieldDecl.Visibility, proxyEnv["GetVisibility"])
		assert.Equal(t, fieldDecl.IsStatic, proxyEnv["GetIsStatic"])
		assert.Equal(t, fieldDecl.IsFinal, proxyEnv["GetIsFinal"])
		assert.Equal(t, fieldDecl.IsVolatile, proxyEnv["GetIsVolatile"])
		assert.Equal(t, fieldDecl.IsTransient, proxyEnv["GetIsTransient"])
	})
}

func TestDatabaseOperations(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	t.Run("BinaryExpr Insert", func(t *testing.T) {
		// Create binary_expr table
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS binary_expr (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				left_operand TEXT NOT NULL,
				right_operand TEXT NOT NULL,
				operator TEXT NOT NULL,
				source_declaration TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		assert.NoError(t, err)

		expr := &BinaryExpr{
			Op:                "+",
			LeftOperand:       &Expr{NodeString: "a"},
			RightOperand:      &Expr{NodeString: "b"},
			SourceDeclaration: "Test.java",
		}
		err = expr.Insert(db)
		assert.NoError(t, err)
	})

	t.Run("MethodCall Insert", func(t *testing.T) {
		// Create method_call table
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS method_call (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				method_name TEXT NOT NULL,
				qualified_name TEXT NOT NULL,
				parameters TEXT,
				parameters_names TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		assert.NoError(t, err)

		methodCall := &MethodCall{
			MethodName:      "test",
			QualifiedMethod: "com.example.Test.test",
			Arguments:       []string{"arg1", "arg2"},
			TypeArguments:   []string{"String", "Integer"},
		}
		err = methodCall.Insert(db)
		assert.NoError(t, err)
	})

	t.Run("FieldDeclaration Insert", func(t *testing.T) {
		// Create field_decl table
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS field_decl (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				field_name TEXT NOT NULL,
				type TEXT NOT NULL,
				visibility TEXT NOT NULL,
				is_static BOOLEAN NOT NULL,
				is_final BOOLEAN NOT NULL,
				is_transient BOOLEAN NOT NULL,
				is_volatile BOOLEAN NOT NULL,
				source_declaration TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		assert.NoError(t, err)

		fieldDecl := &FieldDeclaration{
			Type:              "String",
			FieldNames:        []string{"test1", "test2"},
			Visibility:        "private",
			IsStatic:          true,
			IsFinal:           true,
			IsVolatile:        false,
			IsTransient:       false,
			SourceDeclaration: "Test.java",
		}
		err = fieldDecl.Insert(db)
		assert.NoError(t, err)
	})
}

func TestExprGetBoolValue(t *testing.T) {
	expr := &Expr{}
	expr.GetBoolValue() // Should not panic
}
