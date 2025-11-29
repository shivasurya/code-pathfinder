package core

import (
	"strings"
)

// FrameworkDefinition represents a known external framework or library.
// This is used to mark calls to external code as resolved, even though
// we don't have the source code for these frameworks.
type FrameworkDefinition struct {
	Name        string   // Display name (e.g., "Django")
	Prefixes    []string // Module prefixes to match (e.g., ["django.", "django"])
	Description string   // Human-readable description
	Category    string   // Category: "web", "orm", "testing", "stdlib", etc.
}

// builtinFrameworks contains the list of known Python frameworks and libraries.
// This list focuses on the most common frameworks found in Python projects.
var builtinFrameworks = []FrameworkDefinition{
	// Web Frameworks
	{
		Name:        "Django",
		Prefixes:    []string{"django."},
		Description: "Django web framework",
		Category:    "web",
	},
	{
		Name:        "Django REST Framework",
		Prefixes:    []string{"rest_framework."},
		Description: "Django REST framework for building Web APIs",
		Category:    "web",
	},
	{
		Name:        "Flask",
		Prefixes:    []string{"flask."},
		Description: "Flask web framework",
		Category:    "web",
	},
	{
		Name:        "FastAPI",
		Prefixes:    []string{"fastapi."},
		Description: "FastAPI web framework",
		Category:    "web",
	},
	{
		Name:        "Starlette",
		Prefixes:    []string{"starlette."},
		Description: "Starlette ASGI framework",
		Category:    "web",
	},
	{
		Name:        "Tornado",
		Prefixes:    []string{"tornado."},
		Description: "Tornado web framework",
		Category:    "web",
	},
	{
		Name:        "Pyramid",
		Prefixes:    []string{"pyramid."},
		Description: "Pyramid web framework",
		Category:    "web",
	},
	{
		Name:        "Bottle",
		Prefixes:    []string{"bottle."},
		Description: "Bottle web framework",
		Category:    "web",
	},

	// ORM and Database
	{
		Name:        "SQLAlchemy",
		Prefixes:    []string{"sqlalchemy."},
		Description: "SQLAlchemy ORM",
		Category:    "orm",
	},
	{
		Name:        "Peewee",
		Prefixes:    []string{"peewee."},
		Description: "Peewee ORM",
		Category:    "orm",
	},
	{
		Name:        "Tortoise ORM",
		Prefixes:    []string{"tortoise."},
		Description: "Tortoise ORM",
		Category:    "orm",
	},
	{
		Name:        "Pony ORM",
		Prefixes:    []string{"pony."},
		Description: "Pony ORM",
		Category:    "orm",
	},

	// Testing Frameworks
	{
		Name:        "pytest",
		Prefixes:    []string{"pytest.", "_pytest."},
		Description: "pytest testing framework",
		Category:    "testing",
	},
	{
		Name:        "unittest",
		Prefixes:    []string{"unittest."},
		Description: "Python unittest framework",
		Category:    "testing",
	},
	{
		Name:        "nose",
		Prefixes:    []string{"nose."},
		Description: "nose testing framework",
		Category:    "testing",
	},
	{
		Name:        "mock",
		Prefixes:    []string{"mock.", "unittest.mock."},
		Description: "Python mock library",
		Category:    "testing",
	},

	// HTTP and Requests
	{
		Name:        "requests",
		Prefixes:    []string{"requests."},
		Description: "HTTP library for Python",
		Category:    "http",
	},
	{
		Name:        "httpx",
		Prefixes:    []string{"httpx."},
		Description: "Async HTTP client",
		Category:    "http",
	},
	{
		Name:        "urllib3",
		Prefixes:    []string{"urllib3."},
		Description: "HTTP client library",
		Category:    "http",
	},
	{
		Name:        "aiohttp",
		Prefixes:    []string{"aiohttp."},
		Description: "Async HTTP client/server",
		Category:    "http",
	},

	// Data Science and ML
	{
		Name:        "numpy",
		Prefixes:    []string{"numpy.", "np."},
		Description: "Numerical computing library",
		Category:    "data_science",
	},
	{
		Name:        "pandas",
		Prefixes:    []string{"pandas.", "pd."},
		Description: "Data analysis library",
		Category:    "data_science",
	},
	{
		Name:        "scikit-learn",
		Prefixes:    []string{"sklearn.", "scikit_learn."},
		Description: "Machine learning library",
		Category:    "data_science",
	},
	{
		Name:        "tensorflow",
		Prefixes:    []string{"tensorflow.", "tf."},
		Description: "TensorFlow ML framework",
		Category:    "data_science",
	},
	{
		Name:        "pytorch",
		Prefixes:    []string{"torch."},
		Description: "PyTorch ML framework",
		Category:    "data_science",
	},

	// Async and Concurrency
	{
		Name:        "asyncio",
		Prefixes:    []string{"asyncio."},
		Description: "Async I/O library",
		Category:    "async",
	},
	{
		Name:        "celery",
		Prefixes:    []string{"celery."},
		Description: "Distributed task queue",
		Category:    "async",
	},

	// Serialization and Data Formats
	{
		Name:        "json",
		Prefixes:    []string{"json."},
		Description: "JSON encoder/decoder",
		Category:    "serialization",
	},
	{
		Name:        "pickle",
		Prefixes:    []string{"pickle.", "_pickle."},
		Description: "Python object serialization",
		Category:    "serialization",
	},
	{
		Name:        "yaml",
		Prefixes:    []string{"yaml.", "pyyaml."},
		Description: "YAML parser",
		Category:    "serialization",
	},
	{
		Name:        "xml",
		Prefixes:    []string{"xml."},
		Description: "XML processing",
		Category:    "serialization",
	},

	// Logging and Monitoring
	{
		Name:        "logging",
		Prefixes:    []string{"logging."},
		Description: "Python logging library",
		Category:    "logging",
	},
	{
		Name:        "sentry",
		Prefixes:    []string{"sentry_sdk."},
		Description: "Sentry error tracking",
		Category:    "logging",
	},

	// Utilities
	{
		Name:        "datetime",
		Prefixes:    []string{"datetime."},
		Description: "Date and time types",
		Category:    "stdlib",
	},
	{
		Name:        "collections",
		Prefixes:    []string{"collections."},
		Description: "Container datatypes",
		Category:    "stdlib",
	},
	{
		Name:        "itertools",
		Prefixes:    []string{"itertools."},
		Description: "Iterator functions",
		Category:    "stdlib",
	},
	{
		Name:        "functools",
		Prefixes:    []string{"functools."},
		Description: "Higher-order functions",
		Category:    "stdlib",
	},
	{
		Name:        "os",
		Prefixes:    []string{"os."},
		Description: "Operating system interfaces",
		Category:    "stdlib",
	},
	{
		Name:        "sys",
		Prefixes:    []string{"sys."},
		Description: "System-specific parameters",
		Category:    "stdlib",
	},
	{
		Name:        "pathlib",
		Prefixes:    []string{"pathlib."},
		Description: "Object-oriented filesystem paths",
		Category:    "stdlib",
	},
	{
		Name:        "re",
		Prefixes:    []string{"re."},
		Description: "Regular expressions",
		Category:    "stdlib",
	},
	{
		Name:        "subprocess",
		Prefixes:    []string{"subprocess."},
		Description: "Subprocess management",
		Category:    "stdlib",
	},
	{
		Name:        "threading",
		Prefixes:    []string{"threading."},
		Description: "Thread-based parallelism",
		Category:    "stdlib",
	},
	{
		Name:        "multiprocessing",
		Prefixes:    []string{"multiprocessing."},
		Description: "Process-based parallelism",
		Category:    "stdlib",
	},
	{
		Name:        "socket",
		Prefixes:    []string{"socket."},
		Description: "Low-level networking",
		Category:    "stdlib",
	},
	{
		Name:        "http",
		Prefixes:    []string{"http."},
		Description: "HTTP modules",
		Category:    "stdlib",
	},
	{
		Name:        "urllib",
		Prefixes:    []string{"urllib."},
		Description: "URL handling modules",
		Category:    "stdlib",
	},
	{
		Name:        "email",
		Prefixes:    []string{"email."},
		Description: "Email and MIME handling",
		Category:    "stdlib",
	},
	{
		Name:        "hashlib",
		Prefixes:    []string{"hashlib."},
		Description: "Secure hash and message digest",
		Category:    "stdlib",
	},
	{
		Name:        "hmac",
		Prefixes:    []string{"hmac."},
		Description: "Keyed-hashing for message authentication",
		Category:    "stdlib",
	},
	{
		Name:        "secrets",
		Prefixes:    []string{"secrets."},
		Description: "Generate secure random numbers",
		Category:    "stdlib",
	},
	{
		Name:        "typing",
		Prefixes:    []string{"typing."},
		Description: "Type hints support",
		Category:    "stdlib",
	},
	{
		Name:        "dataclasses",
		Prefixes:    []string{"dataclasses."},
		Description: "Data classes",
		Category:    "stdlib",
	},
	{
		Name:        "abc",
		Prefixes:    []string{"abc."},
		Description: "Abstract base classes",
		Category:    "stdlib",
	},
}

// LoadFrameworks returns the list of known frameworks.
// This function provides an extensibility hook for future enhancements
// where frameworks might be loaded from a configuration file.
func LoadFrameworks() []FrameworkDefinition {
	return builtinFrameworks
}

// IsKnownFramework checks if the given fully qualified name (FQN)
// belongs to a known external framework or standard library.
//
// Parameters:
//   - fqn: fully qualified name (e.g., "django.db.models.ForeignKey")
//
// Returns:
//   - true if the FQN matches any known framework
//   - the matching framework definition
func IsKnownFramework(fqn string) (bool, *FrameworkDefinition) {
	frameworks := LoadFrameworks()

	for i := range frameworks {
		framework := &frameworks[i]
		for _, prefix := range framework.Prefixes {
			// Check for exact match or prefix match
			if fqn == prefix || strings.HasPrefix(fqn, prefix) {
				return true, framework
			}
		}
	}

	return false, nil
}

// GetFrameworkCategory returns the category of a framework given its FQN.
// Returns empty string if not a known framework.
func GetFrameworkCategory(fqn string) string {
	isKnown, framework := IsKnownFramework(fqn)
	if isKnown {
		return framework.Category
	}
	return ""
}

// GetFrameworkName returns the name of a framework given its FQN.
// Returns empty string if not a known framework.
func GetFrameworkName(fqn string) string {
	isKnown, framework := IsKnownFramework(fqn)
	if isKnown {
		return framework.Name
	}
	return ""
}
