package diagnostic

import (
	"fmt"
)

// BuildAnalysisPrompt constructs the prompt for LLM pattern discovery and test generation.
// Based on diagnostic-tech-proposal.md Section 3.3 (LLM Prompt Design).
func BuildAnalysisPrompt(sourceCode string) string {
	return fmt.Sprintf(`You are an intra-procedural dataflow analysis expert. Your goal is to validate if a static analysis tool correctly tracks data flow within THIS SINGLE FUNCTION ONLY.

**FUNCTION TO ANALYZE**:
`+"```python\n%s\n```"+`

**YOUR TASK**: Test if the tool can track how data flows between variables in this function.

1. **DISCOVER DATA SOURCES** - Where does data originate in this function:
   - Function parameters (any param used in the function body)
   - Local variable assignments (x = "value")
   - Function calls that return data (result = some_function())
   - Literals/constants that get assigned to variables
   - Dictionary/list access (val = dict['key'])

2. **DISCOVER DATA SINKS** - Where does data get consumed/used:
   - Return statements (return x)
   - Function calls that use data (print(x), logger.info(x))
   - Assignments to data structures (dict[key] = x, list.append(x))
   - Operations that use variables (y = x + 1)

3. **TRACE INTRA-PROCEDURAL FLOWS** - For EACH source variable:
   - Track ALL assignments: if 'a = source', then 'b = a' means b is tainted from source
   - Track through operations: if 'b = a + "suffix"', b is still tainted from a
   - Track through containers: if 'list = [a]', then 'b = list[0]' means b is tainted
   - Track through branches: if inside 'if True: b = a', then b is tainted
   - Track through method calls: if 'b = a.upper()', b is still tainted from a
   - ONLY track within this function (do NOT analyze called functions)

4. **GENERATE TEST CASES** - Create test cases for flows YOU FOUND:
   - Only include flows where source → sink connection exists within this function
   - Set expected_detection=true if there IS a flow path
   - Set expected_detection=false if source and sink are INDEPENDENT variables

**OUTPUT FORMAT** (JSON):
`+"```json"+`
{
  "discovered_patterns": {
    "sources": [
      {
        "pattern": "request.GET",
        "lines": [10, 15],
        "variables": ["user_input", "cmd"],
        "category": "user_input",
        "description": "HTTP GET parameter access"
      }
    ],
    "sinks": [
      {
        "pattern": "os.system",
        "lines": [45],
        "variables": ["command"],
        "category": "command_exec",
        "description": "OS command execution",
        "severity": "CRITICAL"
      }
    ],
    "sanitizers": [
      {
        "pattern": "shlex.quote",
        "lines": [30],
        "variables": ["cleaned_cmd"],
        "description": "Shell escaping function"
      }
    ],
    "propagators": [
      {
        "type": "assignment",
        "line": 12,
        "from_var": "user_input",
        "to_var": "raw_cmd"
      }
    ]
  },

  "dataflow_test_cases": [
    {
      "test_id": 1,
      "description": "Unsanitized user input flows to command execution",
      "source": {
        "pattern": "request.GET['cmd']",
        "line": 10,
        "variable": "user_input"
      },
      "sink": {
        "pattern": "os.system",
        "line": 45,
        "variable": "command"
      },
      "flow_path": [
        {"line": 10, "variable": "user_input", "operation": "source"},
        {"line": 12, "variable": "raw_cmd", "operation": "assignment"},
        {"line": 45, "variable": "command", "operation": "sink"}
      ],
      "sanitizers_in_path": [],
      "expected_detection": true,
      "vulnerability_type": "COMMAND_INJECTION",
      "confidence": 0.95,
      "reasoning": "Direct flow from user input to OS command without sanitization",
      "failure_category": "none"
    }
  ],

  "variable_tracking": [
    {
      "variable": "user_input",
      "first_defined": 10,
      "last_used": 45,
      "aliases": ["raw_cmd", "command"],
      "flows_to_lines": [12, 20, 45],
      "flows_to_vars": ["raw_cmd", "processed", "command"]
    }
  ],

  "analysis_metadata": {
    "total_sources": 1,
    "total_sinks": 1,
    "total_sanitizers": 0,
    "total_flows": 1,
    "dangerous_flows": 1,
    "safe_flows": 0,
    "confidence": 0.95,
    "limitations": [
      "Intra-procedural only (did not analyze called functions)",
      "Control flow branches not fully explored"
    ]
  }
}
`+"```"+`

**IMPORTANT GUIDELINES**:

1. **RESPOND WITH PURE JSON ONLY**: No markdown, no code blocks, no explanations - just the raw JSON object
2. **INTRA-PROCEDURAL ONLY**: Only analyze flows within this function, ignore inter-procedural flows
3. **BE SPECIFIC**: Include exact line numbers, variable names from the function
4. **TRACK SIMPLE DATAFLOWS**: Focus on variable assignments, not complex analysis
5. **CONFIDENCE SCORES**: Rate how confident you are (0.0-1.0)
6. **EXPLAIN REASONING**: Describe the assignment chain (e.g., "a→b→c→return")
7. **FAILURE CATEGORY**: For each test case, specify why a static analysis tool might miss this flow:
   - "none" - Simple flow, tool should detect easily
   - "control_flow_branch" - Flow through if/else/while branches
   - "field_sensitivity" - Flow through object fields (self.x, obj.attr)
   - "sanitizer_missed" - Has sanitizer that tool might not recognize
   - "container_operation" - Flow through list/dict/array operations
   - "string_formatting" - Flow through f-strings, .format(), concatenation
   - "method_call_propagation" - Flow through method calls like .upper(), .strip()
   - "assignment_chain" - Long chain of assignments (a=b; c=a; d=c)
   - "return_flow" - Flow from variable to return statement
   - "parameter_flow" - Flow from function parameter to usage
   - "complex_expression" - Nested calls, multiple operations
   - "context_required" - Requires analyzing called functions (out of scope)

**EXAMPLE DATAFLOW PATTERNS WITH CATEGORIES**:

Simple assignments (failure_category: "none"):
- param → local_var → return
- x = param; y = x; return y
- result = function(param); return result

Assignment chains (failure_category: "assignment_chain"):
- x = param; y = x; z = y; w = z; return w

Through method calls (failure_category: "method_call_propagation"):
- x = param.upper(); return x
- y = param.strip().lower(); return y

Through string operations (failure_category: "string_formatting"):
- x = param + "suffix"; return x
- z = f"{param}"; return z
- w = "%%s" %% param; return w

Through containers (failure_category: "container_operation"):
- list = [param]; x = list[0]; return x
- dict = {"key": param}; y = dict["key"]; return y

Through branches (failure_category: "control_flow_branch"):
- if condition: x = param; return x
- if True: y = param; else: y = ""; return y

No flow (failure_category: "none"):
- x = param; y = "constant"; return y  (y NOT tainted by param)

**FOCUS**: Validate intra-procedural dataflow tracking:
- ✅ Direct assignments (a = b)
- ✅ Assignment chains (a = b; c = a)
- ✅ Operations preserving taint (x = a + "text")
- ✅ Container flows (list[0] = a; b = list[0])
- ✅ Control flow branches
- ✅ Independent variables (no flow)

Output ONLY the JSON, no additional text.`, sourceCode)
}
