package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKnownFramework_Django(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
		category string
	}{
		{
			name:     "Django core",
			fqn:      "django.db.models.Model",
			expected: true,
			category: "web",
		},
		{
			name:     "Django REST framework",
			fqn:      "rest_framework.serializers.ModelSerializer",
			expected: true,
			category: "web",
		},
		{
			name:     "Django forms",
			fqn:      "django.forms.Form",
			expected: true,
			category: "web",
		},
		{
			name:     "Django ORM",
			fqn:      "django.db.models.ForeignKey",
			expected: true,
			category: "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.Equal(t, tt.expected, isKnown)
			if isKnown {
				assert.NotNil(t, framework)
				assert.Equal(t, tt.category, framework.Category)
			}
		})
	}
}

func TestIsKnownFramework_Testing(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
		category string
	}{
		{
			name:     "pytest",
			fqn:      "pytest.fixture",
			expected: true,
			category: "testing",
		},
		{
			name:     "unittest",
			fqn:      "unittest.TestCase",
			expected: true,
			category: "testing",
		},
		{
			name:     "mock",
			fqn:      "unittest.mock.patch",
			expected: true,
			category: "testing",
		},
		{
			name:     "_pytest internal",
			fqn:      "_pytest.fixtures",
			expected: true,
			category: "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.Equal(t, tt.expected, isKnown)
			if isKnown {
				assert.NotNil(t, framework)
				assert.Equal(t, tt.category, framework.Category)
			}
		})
	}
}

func TestIsKnownFramework_HTTP(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
		category string
	}{
		{
			name:     "requests library",
			fqn:      "requests.get",
			expected: true,
			category: "http",
		},
		{
			name:     "httpx",
			fqn:      "httpx.AsyncClient",
			expected: true,
			category: "http",
		},
		{
			name:     "urllib3",
			fqn:      "urllib3.PoolManager",
			expected: true,
			category: "http",
		},
		{
			name:     "aiohttp",
			fqn:      "aiohttp.ClientSession",
			expected: true,
			category: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.Equal(t, tt.expected, isKnown)
			if isKnown {
				assert.NotNil(t, framework)
				assert.Equal(t, tt.category, framework.Category)
			}
		})
	}
}

func TestIsKnownFramework_DataScience(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
		category string
	}{
		{
			name:     "numpy",
			fqn:      "numpy.array",
			expected: true,
			category: "data_science",
		},
		{
			name:     "pandas",
			fqn:      "pandas.DataFrame",
			expected: true,
			category: "data_science",
		},
		{
			name:     "sklearn",
			fqn:      "sklearn.ensemble.RandomForestClassifier",
			expected: true,
			category: "data_science",
		},
		{
			name:     "tensorflow",
			fqn:      "tensorflow.keras.Model",
			expected: true,
			category: "data_science",
		},
		{
			name:     "pytorch",
			fqn:      "torch.nn.Module",
			expected: true,
			category: "data_science",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.Equal(t, tt.expected, isKnown)
			if isKnown {
				assert.NotNil(t, framework)
				assert.Equal(t, tt.category, framework.Category)
			}
		})
	}
}

func TestIsKnownFramework_Stdlib(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
		category string
	}{
		{
			name:     "json",
			fqn:      "json.loads",
			expected: true,
			category: "serialization",
		},
		{
			name:     "pickle",
			fqn:      "pickle.dumps",
			expected: true,
			category: "serialization",
		},
		{
			name:     "logging",
			fqn:      "logging.getLogger",
			expected: true,
			category: "logging",
		},
		{
			name:     "datetime",
			fqn:      "datetime.datetime",
			expected: true,
			category: "stdlib",
		},
		{
			name:     "collections",
			fqn:      "collections.defaultdict",
			expected: true,
			category: "stdlib",
		},
		{
			name:     "os",
			fqn:      "os.path.join",
			expected: true,
			category: "stdlib",
		},
		{
			name:     "subprocess",
			fqn:      "subprocess.run",
			expected: true,
			category: "stdlib",
		},
		{
			name:     "hashlib",
			fqn:      "hashlib.sha256",
			expected: true,
			category: "stdlib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.Equal(t, tt.expected, isKnown)
			if isKnown {
				assert.NotNil(t, framework)
				assert.Equal(t, tt.category, framework.Category)
			}
		})
	}
}

func TestIsKnownFramework_NotFound(t *testing.T) {
	tests := []struct {
		name string
		fqn  string
	}{
		{
			name: "Custom application module",
			fqn:  "myapp.utils.helpers",
		},
		{
			name: "Custom package",
			fqn:  "internal.services.auth",
		},
		{
			name: "Unknown framework",
			fqn:  "unknownframework.module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKnown, framework := IsKnownFramework(tt.fqn)
			assert.False(t, isKnown)
			assert.Nil(t, framework)
		})
	}
}

func TestGetFrameworkCategory(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected string
	}{
		{
			name:     "Django web framework",
			fqn:      "django.http.HttpResponse",
			expected: "web",
		},
		{
			name:     "pytest testing",
			fqn:      "pytest.mark.parametrize",
			expected: "testing",
		},
		{
			name:     "requests http",
			fqn:      "requests.post",
			expected: "http",
		},
		{
			name:     "Unknown framework",
			fqn:      "myapp.custom",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := GetFrameworkCategory(tt.fqn)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestGetFrameworkName(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected string
	}{
		{
			name:     "Django",
			fqn:      "django.contrib.auth",
			expected: "Django",
		},
		{
			name:     "Flask",
			fqn:      "flask.Flask",
			expected: "Flask",
		},
		{
			name:     "FastAPI",
			fqn:      "fastapi.FastAPI",
			expected: "FastAPI",
		},
		{
			name:     "Unknown",
			fqn:      "myapp.unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := GetFrameworkName(tt.fqn)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestLoadFrameworks(t *testing.T) {
	frameworks := LoadFrameworks()

	// Should have at least 50 frameworks defined
	assert.GreaterOrEqual(t, len(frameworks), 50)

	// Check that all frameworks have required fields
	for _, fw := range frameworks {
		assert.NotEmpty(t, fw.Name, "Framework should have a name")
		assert.NotEmpty(t, fw.Prefixes, "Framework should have at least one prefix")
		assert.NotEmpty(t, fw.Category, "Framework should have a category")
		assert.NotEmpty(t, fw.Description, "Framework should have a description")
	}
}
