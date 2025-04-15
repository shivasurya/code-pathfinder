package model

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewCallableAndMethods(t *testing.T) {
	params := []string{"int", "String"}
	paramNames := []string{"a", "b"}
	callable := NewCallable("foo", "com.example.Foo.foo", "void", params, paramNames, true, "Foo.java:10")

	assert.Equal(t, "foo", callable.Name)
	assert.Equal(t, "com.example.Foo.foo", callable.QualifiedName)
	assert.Equal(t, "void", callable.ReturnType)
	assert.Equal(t, params, callable.GetAParamType())
	assert.Equal(t, []string{"int a", "String b"}, callable.GetAParameter())
	assert.Equal(t, 2, callable.GetNumberOfParameters())
	assert.Equal(t, "int a", callable.GetParameter(0))
	assert.Equal(t, "String b", callable.GetParameter(1))
	assert.Equal(t, "", callable.GetParameter(2))
	assert.Equal(t, "int", callable.GetParameterType(0))
	assert.Equal(t, "String", callable.GetParameterType(1))
	assert.Equal(t, "", callable.GetParameterType(2))
	assert.Equal(t, "void", callable.GetReturnType())
	assert.Contains(t, callable.GetSignature(), "void foo(")
	assert.Equal(t, "Foo.java:10", callable.GetSourceDeclaration())
	assert.Contains(t, callable.GetStringSignature(), "foo(")
	assert.Equal(t, 1, callable.GetVarargsParameterIndex())
	assert.False(t, NewCallable("bar", "Bar.bar", "int", nil, nil, false, "Bar.java:1").GetIsVarargs())
	assert.False(t, NewCallable("bar", "Bar.bar", "int", nil, nil, false, "Bar.java:1").IsVarargs)
	assert.True(t, callable.GetIsVarargs())
	assert.Equal(t, false, NewCallable("bar", "Bar.bar", "int", nil, nil, false, "Bar.java:1").HasNoParameters())
	assert.True(t, NewCallable("bar", "Bar.bar", "int", []string{}, []string{}, false, "Bar.java:1").HasNoParameters())
	assert.Equal(t, "(int, String)", callable.ParamsString())
	assert.Equal(t, "()", NewCallable("bar", "Bar.bar", "int", nil, nil, false, "Bar.java:1").ParamsString())
}

func TestNewMethodAndMethods(t *testing.T) {
	params := []string{"int", "String"}
	paramNames := []string{"a", "b"}
	m := NewMethod("foo", "com.example.Foo.foo", "void", params, paramNames, "public", true, false, false, false, false, "Foo.java:10")
	m2 := NewMethod("foo", "com.example.Foo.foo", "void", params, paramNames, "public", false, false, false, false, false, "Foo.java:10")

	assert.Equal(t, "Method", m.GetAPrimaryQlClass())
	assert.Equal(t, "foo", m.GetName())
	assert.Equal(t, "com.example.Foo.foo", m.GetFullyQualifiedName())
	assert.Equal(t, "void", m.GetReturnType())
	assert.Equal(t, params, m.GetParameters())
	assert.Equal(t, paramNames, m.GetParameterNames())
	assert.Equal(t, "public", m.GetVisibility())
	assert.Contains(t, m.GetSignature(), "void foo(")
	assert.Equal(t, "Foo.java:10", m.GetSourceDeclaration())
	assert.True(t, m.GetIsAbstract())
	assert.False(t, m2.GetIsAbstract())
	assert.True(t, m.IsPublic())
	assert.True(t, m.IsInheritable())
	assert.False(t, NewMethod("bar", "Bar.bar", "int", nil, nil, "private", false, false, true, true, false, "Bar.java:1").IsInheritable())
	assert.True(t, m.SameParamTypes(m2))
	m3 := NewMethod("foo", "com.example.Foo.foo", "void", []string{"int"}, []string{"a"}, "public", false, false, false, false, false, "Foo.java:10")
	assert.False(t, m.SameParamTypes(m3))
	proxy := m.GetProxyEnv()
	assert.NotNil(t, proxy["getVisibility"])
	assert.NotNil(t, proxy["getReturnType"])
	assert.NotNil(t, proxy["getName"])
	assert.NotNil(t, proxy["getParameters"])
	assert.NotNil(t, proxy["getParameterNames"])
}
