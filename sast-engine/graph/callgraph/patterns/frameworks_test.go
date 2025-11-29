package patterns

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestDetectFramework_Django(t *testing.T) {
	importMap := core.NewImportMap("test.py")
	importMap.AddImport("HttpResponse", "django.http.HttpResponse")

	fw := DetectFramework(importMap)
	assert.NotNil(t, fw)
	assert.Equal(t, "Django", fw.Name)
	assert.Equal(t, "web", fw.Category)
}

func TestDetectFramework_Flask(t *testing.T) {
	importMap := core.NewImportMap("test.py")
	importMap.AddImport("Flask", "flask.Flask")

	fw := DetectFramework(importMap)
	assert.NotNil(t, fw)
	assert.Equal(t, "Flask", fw.Name)
	assert.Equal(t, "web", fw.Category)
}

func TestDetectFramework_FastAPI(t *testing.T) {
	importMap := core.NewImportMap("test.py")
	importMap.AddImport("FastAPI", "fastapi.FastAPI")

	fw := DetectFramework(importMap)
	assert.NotNil(t, fw)
	assert.Equal(t, "FastAPI", fw.Name)
	assert.Equal(t, "web", fw.Category)
}

func TestDetectFramework_NoFramework(t *testing.T) {
	importMap := core.NewImportMap("test.py")
	importMap.AddImport("helper", "myapp.utils.helper")

	fw := DetectFramework(importMap)
	assert.Nil(t, fw)
}

func TestDetectFramework_NilImportMap(t *testing.T) {
	fw := DetectFramework(nil)
	assert.Nil(t, fw)
}

func TestDetectFramework_EmptyImportMap(t *testing.T) {
	importMap := core.NewImportMap("test.py")

	fw := DetectFramework(nil)
	assert.Nil(t, fw)

	fw = DetectFramework(importMap)
	assert.Nil(t, fw)
}

func TestIsKnownFramework(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		expected   bool
	}{
		{"Django framework", "django.http", true},
		{"Flask framework", "flask.app", true},
		{"FastAPI framework", "fastapi.FastAPI", true},
		{"Tornado framework", "tornado.web", true},
		{"Pyramid framework", "pyramid.view", true},
		{"Bottle framework", "bottle.Bottle", true},
		{"SQLAlchemy ORM", "sqlalchemy.orm", true},
		{"Requests library", "requests.get", true},
		{"NumPy library", "numpy.array", true},
		{"Pandas library", "pandas.DataFrame", true},
		{"Python stdlib", "os.path", true},
		{"User module", "myapp.utils", false},
		{"Unknown package", "unknown.module", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKnownFramework(tt.importPath)
			assert.Equal(t, tt.expected, result, "IsKnownFramework(%s)", tt.importPath)
		})
	}
}

func TestGetFrameworkCategory(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		expected   string
	}{
		{"Django web framework", "django.http", "web"},
		{"Flask web framework", "flask.app", "web"},
		{"SQLAlchemy ORM", "sqlalchemy.orm", "orm"},
		{"pytest testing", "pytest.fixture", "testing"},
		{"requests HTTP", "requests.get", "http"},
		{"NumPy data science", "numpy.array", "data_science"},
		{"os stdlib", "os.path", "stdlib"},
		{"Unknown module", "myapp.utils", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFrameworkCategory(tt.importPath)
			assert.Equal(t, tt.expected, result, "GetFrameworkCategory(%s)", tt.importPath)
		})
	}
}

func TestGetFrameworkName(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		expected   string
	}{
		{"Django", "django.http", "Django"},
		{"Flask", "flask.app", "Flask"},
		{"FastAPI", "fastapi.FastAPI", "FastAPI"},
		{"SQLAlchemy", "sqlalchemy.orm", "SQLAlchemy"},
		{"pytest", "pytest.fixture", "pytest"},
		{"requests", "requests.get", "requests"},
		{"NumPy", "numpy.array", "numpy"},
		{"Unknown", "myapp.utils", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFrameworkName(tt.importPath)
			assert.Equal(t, tt.expected, result, "GetFrameworkName(%s)", tt.importPath)
		})
	}
}

func TestDetectFramework_MultipleFrameworks(t *testing.T) {
	// When multiple frameworks are present, should return the first detected
	importMap := core.NewImportMap("test.py")
	importMap.AddImport("HttpResponse", "django.http.HttpResponse")
	importMap.AddImport("Flask", "flask.Flask")

	fw := DetectFramework(importMap)
	assert.NotNil(t, fw)
	// Should return one of them (order depends on map iteration)
	assert.Contains(t, []string{"Django", "Flask"}, fw.Name)
	assert.Equal(t, "web", fw.Category)
}

func TestFrameworkStruct(t *testing.T) {
	fw := &Framework{
		Name:     "TestFramework",
		Version:  "1.0.0",
		Category: "test",
	}

	assert.Equal(t, "TestFramework", fw.Name)
	assert.Equal(t, "1.0.0", fw.Version)
	assert.Equal(t, "test", fw.Category)
}
