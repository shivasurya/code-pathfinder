package dsl

import "log"

// SemanticExpander expands compact semantic source/sink categories
// into concrete TypeConstrainedCallIR/CallMatcherIR matcher maps.
type SemanticExpander struct{}

// NewSemanticExpander creates a new expander.
func NewSemanticExpander() *SemanticExpander {
	return &SemanticExpander{}
}

// ExpandSource expands a semantic source category into matcher maps.
// If framework is empty, returns all framework patterns.
// Unknown categories return nil and log a warning.
func (e *SemanticExpander) ExpandSource(category, framework string) []map[string]any {
	switch category {
	case "http_params":
		return e.expandHTTPParams(framework)
	case "http_body":
		return e.expandHTTPBody(framework)
	case "http_headers":
		return e.expandHTTPHeaders(framework)
	case "http_cookies":
		return e.expandHTTPCookies(framework)
	case "http_input":
		return e.expandHTTPInput(framework)
	case "file_read":
		return e.expandFileRead(framework)
	case "file_path":
		return e.expandFilePath(framework)
	case "env_vars":
		return e.expandEnvVars(framework)
	case "cli_args":
		return e.expandCliArgs(framework)
	case "database_result":
		return e.expandDatabaseResult(framework)
	case "user_input":
		return e.expandUserInput(framework)
	default:
		log.Printf("WARNING: SemanticExpander: unknown source category %q", category)
		return nil
	}
}

// ExpandSink expands a semantic sink category into matcher maps.
func (e *SemanticExpander) ExpandSink(category, framework string) []map[string]any {
	switch category {
	case "sql_execution":
		return e.expandSQLExecution(framework)
	case "command_execution":
		return e.expandCommandExecution(framework)
	case "code_execution":
		return e.expandCodeExecution(framework)
	case "template_render":
		return e.expandTemplateRender(framework)
	case "xpath_query":
		return e.expandXPathQuery(framework)
	case "ldap_query":
		return e.expandLDAPQuery(framework)
	case "file_write":
		return e.expandFileWrite(framework)
	case "file_open":
		return e.expandFileOpen(framework)
	case "path_operation":
		return e.expandPathOperation(framework)
	case "http_request":
		return e.expandHTTPRequest(framework)
	case "socket_connect":
		return e.expandSocketConnect(framework)
	case "deserialize":
		return e.expandDeserialize(framework)
	default:
		log.Printf("WARNING: SemanticExpander: unknown sink category %q", category)
		return nil
	}
}

// --- Helper constructors ---

func typeConstrainedMap(receiverType, methodName, fallbackMode string) map[string]any {
	return map[string]any{
		"type":          "type_constrained_call",
		"receiverType":  receiverType,
		"methodName":    methodName,
		"minConfidence": 0.5,
		"fallbackMode":  fallbackMode,
	}
}

func callMatcherMap(patterns ...string) map[string]any {
	return map[string]any{
		"type":      "call_matcher",
		"patterns":  toAnySlice(patterns),
		"wildcard":  containsWildcard(patterns),
		"matchMode": "any",
	}
}

func toAnySlice(ss []string) []any {
	result := make([]any, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}

func containsWildcard(patterns []string) bool {
	for _, p := range patterns {
		for _, c := range p {
			if c == '*' {
				return true
			}
		}
	}
	return false
}

// --- Source expansions (mirror Python sources.py exactly) ---

func (e *SemanticExpander) expandHTTPParams(fw string) []map[string]any {
	var m []map[string]any
	if fw == "" || fw == "django" {
		m = append(m,
			typeConstrainedMap("HttpRequest", "GET", "name"),
			typeConstrainedMap("HttpRequest", "POST", "name"),
			typeConstrainedMap("QueryDict", "__getitem__", "name"),
		)
	}
	if fw == "" || fw == "flask" {
		m = append(m,
			typeConstrainedMap("Request", "args", "name"),
			typeConstrainedMap("Request", "form", "name"),
			typeConstrainedMap("Request", "values", "name"),
		)
	}
	// Generic (always included)
	m = append(m,
		callMatcherMap("request.get"),
		callMatcherMap("request.args.*"),
	)
	return m
}

func (e *SemanticExpander) expandHTTPBody(fw string) []map[string]any {
	var m []map[string]any
	if fw == "" || fw == "flask" {
		m = append(m,
			typeConstrainedMap("Request", "json", "name"),
			typeConstrainedMap("Request", "data", "name"),
			typeConstrainedMap("Request", "get_json", "name"),
		)
	}
	if fw == "" || fw == "django" {
		m = append(m, typeConstrainedMap("HttpRequest", "body", "name"))
	}
	return m
}

func (e *SemanticExpander) expandHTTPHeaders(fw string) []map[string]any {
	var m []map[string]any
	if fw == "" || fw == "django" {
		m = append(m,
			typeConstrainedMap("HttpRequest", "META", "name"),
			typeConstrainedMap("HttpRequest", "headers", "name"),
		)
	}
	if fw == "" || fw == "flask" {
		m = append(m, typeConstrainedMap("Request", "headers", "name"))
	}
	return m
}

func (e *SemanticExpander) expandHTTPCookies(fw string) []map[string]any {
	var m []map[string]any
	if fw == "" || fw == "django" {
		m = append(m, typeConstrainedMap("HttpRequest", "COOKIES", "name"))
	}
	if fw == "" || fw == "flask" {
		m = append(m, typeConstrainedMap("Request", "cookies", "name"))
	}
	return m
}

func (e *SemanticExpander) expandHTTPInput(fw string) []map[string]any {
	var m []map[string]any
	m = append(m, e.expandHTTPParams(fw)...)
	m = append(m, e.expandHTTPBody(fw)...)
	m = append(m, e.expandHTTPHeaders(fw)...)
	m = append(m, e.expandHTTPCookies(fw)...)
	return m
}

func (e *SemanticExpander) expandFileRead(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("open"),
		callMatcherMap("read"),
		callMatcherMap("readlines"),
		callMatcherMap("readline"),
		typeConstrainedMap("Path", "read_text", "name"),
		typeConstrainedMap("Path", "read_bytes", "name"),
	}
}

func (e *SemanticExpander) expandFilePath(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("os.path.join"),
		callMatcherMap("os.path.abspath"),
		callMatcherMap("os.path.expanduser"),
		typeConstrainedMap("Path", "__init__", "name"),
	}
}

func (e *SemanticExpander) expandEnvVars(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("os.getenv"),
		callMatcherMap("os.environ.get"),
		callMatcherMap("os.environ.*"),
	}
}

func (e *SemanticExpander) expandCliArgs(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("sys.argv"),
		typeConstrainedMap("ArgumentParser", "parse_args", "name"),
		typeConstrainedMap("ArgumentParser", "parse_known_args", "name"),
	}
}

func (e *SemanticExpander) expandDatabaseResult(_ string) []map[string]any {
	return []map[string]any{
		typeConstrainedMap("Cursor", "fetchone", "name"),
		typeConstrainedMap("Cursor", "fetchall", "name"),
		typeConstrainedMap("Cursor", "fetchmany", "name"),
		typeConstrainedMap("Query", "all", "name"),
		typeConstrainedMap("Query", "first", "name"),
		typeConstrainedMap("Query", "one", "name"),
	}
}

func (e *SemanticExpander) expandUserInput(fw string) []map[string]any {
	var m []map[string]any
	m = append(m, e.expandHTTPInput(fw)...)
	m = append(m, e.expandFileRead(fw)...)
	m = append(m, e.expandEnvVars(fw)...)
	m = append(m, e.expandCliArgs(fw)...)
	return m
}

// --- Sink expansions (mirror Python sinks.py exactly) ---

func (e *SemanticExpander) expandSQLExecution(_ string) []map[string]any {
	return []map[string]any{
		typeConstrainedMap("Cursor", "execute", "none"),
		typeConstrainedMap("Cursor", "executemany", "none"),
		typeConstrainedMap("Connection", "execute", "none"),
		typeConstrainedMap("Engine", "execute", "none"),
		typeConstrainedMap("Session", "execute", "none"),
		typeConstrainedMap("QuerySet", "raw", "none"),
		typeConstrainedMap("QuerySet", "extra", "none"),
		callMatcherMap("RawSQL"),
	}
}

func (e *SemanticExpander) expandCommandExecution(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("os.system"),
		callMatcherMap("os.popen"),
		callMatcherMap("subprocess.run"),
		callMatcherMap("subprocess.call"),
		callMatcherMap("subprocess.Popen"),
		callMatcherMap("subprocess.check_output"),
		callMatcherMap("subprocess.check_call"),
	}
}

func (e *SemanticExpander) expandCodeExecution(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("eval"),
		callMatcherMap("exec"),
		callMatcherMap("compile"),
		callMatcherMap("__import__"),
	}
}

func (e *SemanticExpander) expandTemplateRender(_ string) []map[string]any {
	return []map[string]any{
		typeConstrainedMap("Template", "render", "none"),
		typeConstrainedMap("Environment", "from_string", "none"),
		callMatcherMap("mark_safe"),
		callMatcherMap("render_template_string"),
	}
}

func (e *SemanticExpander) expandXPathQuery(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("lxml.etree.parse"),
		callMatcherMap("lxml.etree.fromstring"),
		typeConstrainedMap("XPath", "evaluate", "none"),
		callMatcherMap("xml.etree.ElementTree.parse"),
		callMatcherMap("xml.etree.ElementTree.fromstring"),
	}
}

func (e *SemanticExpander) expandLDAPQuery(_ string) []map[string]any {
	return []map[string]any{
		typeConstrainedMap("LDAPObject", "search_s", "none"),
		typeConstrainedMap("LDAPObject", "search_st", "none"),
		typeConstrainedMap("LDAPObject", "search_ext_s", "none"),
	}
}

func (e *SemanticExpander) expandFileWrite(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("write"),
		callMatcherMap("writelines"),
		typeConstrainedMap("Path", "write_text", "none"),
		typeConstrainedMap("Path", "write_bytes", "none"),
	}
}

func (e *SemanticExpander) expandFileOpen(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("open"),
		callMatcherMap("io.open"),
	}
}

func (e *SemanticExpander) expandPathOperation(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("os.remove"),
		callMatcherMap("os.unlink"),
		callMatcherMap("os.rename"),
		callMatcherMap("os.chmod"),
		callMatcherMap("os.mkdir"),
		callMatcherMap("shutil.copy"),
		callMatcherMap("shutil.move"),
		callMatcherMap("shutil.rmtree"),
		callMatcherMap("open"),
	}
}

func (e *SemanticExpander) expandHTTPRequest(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("requests.get"),
		callMatcherMap("requests.post"),
		callMatcherMap("requests.put"),
		callMatcherMap("requests.delete"),
		callMatcherMap("requests.request"),
		callMatcherMap("urllib.request.urlopen"),
		callMatcherMap("urllib.request.Request"),
		callMatcherMap("httpx.get"),
		callMatcherMap("httpx.post"),
	}
}

func (e *SemanticExpander) expandSocketConnect(_ string) []map[string]any {
	return []map[string]any{
		typeConstrainedMap("socket", "connect", "none"),
		typeConstrainedMap("socket", "bind", "none"),
	}
}

func (e *SemanticExpander) expandDeserialize(_ string) []map[string]any {
	return []map[string]any{
		callMatcherMap("pickle.loads"),
		callMatcherMap("pickle.load"),
		callMatcherMap("yaml.load"),
		callMatcherMap("yaml.unsafe_load"),
		callMatcherMap("marshal.loads"),
		callMatcherMap("marshal.load"),
		callMatcherMap("jsonpickle.decode"),
	}
}
