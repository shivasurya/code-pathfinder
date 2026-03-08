package registry

// stdlibKnownReturnTypes contains hardcoded return types for high-value
// stdlib functions where the CDN registry data is incomplete.
// This is a stopgap until the registry generator is improved.
//
// Format: module -> function -> return type FQN.
var stdlibKnownReturnTypes = map[string]map[string]string{
	"sqlite3": {
		"connect": "sqlite3.Connection",
	},
	"re": {
		"compile": "re.Pattern",
		"search":  "re.Match",
		"match":   "re.Match",
		"findall": "builtins.list",
	},
	"io": {
		"open":            "io.IOBase",
		"BytesIO":         "io.BytesIO",
		"StringIO":        "io.StringIO",
		"BufferedReader":  "io.BufferedReader",
	},
	"hashlib": {
		"md5":    "hashlib.HASH",
		"sha1":   "hashlib.HASH",
		"sha256": "hashlib.HASH",
		"sha512": "hashlib.HASH",
		"new":    "hashlib.HASH",
	},
	"json": {
		"loads": "builtins.dict",
		"load":  "builtins.dict",
	},
	"os": {
		"listdir": "builtins.list",
		"getcwd":  "builtins.str",
		"getenv":  "builtins.str",
	},
	"subprocess": {
		"run":   "subprocess.CompletedProcess",
		"Popen": "subprocess.Popen",
	},
	"tempfile": {
		"NamedTemporaryFile": "tempfile.NamedTemporaryFile",
		"mkdtemp":            "builtins.str",
		"mkstemp":            "builtins.tuple",
	},
	"pathlib": {
		"Path": "pathlib.Path",
	},
	"logging": {
		"getLogger": "logging.Logger",
	},
	"socket": {
		"socket": "socket.socket",
	},
	"http.client": {
		"HTTPConnection":  "http.client.HTTPConnection",
		"HTTPSConnection": "http.client.HTTPSConnection",
	},
}

// stdlibKnownMethodReturnTypes contains hardcoded return types for class methods.
// Format: module -> class -> method -> return type FQN.
var stdlibKnownMethodReturnTypes = map[string]map[string]map[string]string{
	"sqlite3": {
		"Connection": {
			"cursor":  "sqlite3.Cursor",
			"execute": "sqlite3.Cursor",
		},
		"Cursor": {
			"execute":     "sqlite3.Cursor",
			"executemany": "sqlite3.Cursor",
			"fetchone":    "builtins.tuple",
			"fetchall":    "builtins.list",
			"fetchmany":   "builtins.list",
		},
	},
	"re": {
		"Pattern": {
			"search":  "re.Match",
			"match":   "re.Match",
			"findall": "builtins.list",
			"sub":     "builtins.str",
		},
		"Match": {
			"group":  "builtins.str",
			"groups": "builtins.tuple",
		},
	},
	"hashlib": {
		"HASH": {
			"hexdigest": "builtins.str",
			"digest":    "builtins.bytes",
			"update":    "builtins.NoneType",
		},
	},
	"subprocess": {
		"Popen": {
			"communicate": "builtins.tuple",
			"wait":        "builtins.int",
		},
		"CompletedProcess": {
			"check_returncode": "builtins.NoneType",
		},
	},
	"logging": {
		"Logger": {
			"getChild":          "logging.Logger",
			"getEffectiveLevel": "builtins.int",
		},
	},
	"pathlib": {
		"Path": {
			"resolve":    "pathlib.Path",
			"parent":     "pathlib.Path",
			"joinpath":   "pathlib.Path",
			"open":       "io.IOBase",
			"read_text":  "builtins.str",
			"read_bytes": "builtins.bytes",
			"exists":     "builtins.bool",
			"is_file":    "builtins.bool",
			"is_dir":     "builtins.bool",
		},
	},
	"socket": {
		"socket": {
			"accept":   "builtins.tuple",
			"recv":     "builtins.bytes",
			"send":     "builtins.int",
			"connect":  "builtins.NoneType",
			"bind":     "builtins.NoneType",
			"listen":   "builtins.NoneType",
			"makefile": "socket.SocketIO",
		},
	},
}

// GetKnownStdlibReturnType returns the hardcoded return type for a stdlib function.
func GetKnownStdlibReturnType(moduleName, funcName string) string {
	if mod, ok := stdlibKnownReturnTypes[moduleName]; ok {
		return mod[funcName]
	}
	return ""
}

// GetKnownStdlibMethodReturnType returns the hardcoded return type for a stdlib class method.
func GetKnownStdlibMethodReturnType(moduleName, className, methodName string) string {
	if mod, ok := stdlibKnownMethodReturnTypes[moduleName]; ok {
		if cls, ok := mod[className]; ok {
			return cls[methodName]
		}
	}
	return ""
}

// HasKnownStdlibTypes checks if we have hardcoded type data for a module.
func HasKnownStdlibTypes(moduleName string) bool {
	_, hasFuncs := stdlibKnownReturnTypes[moduleName]
	_, hasMethods := stdlibKnownMethodReturnTypes[moduleName]
	return hasFuncs || hasMethods
}
