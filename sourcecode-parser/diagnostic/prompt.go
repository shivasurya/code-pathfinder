package diagnostic

import (
	"fmt"
)

// BuildAnalysisPrompt constructs the prompt for LLM pattern discovery and test generation.
// Based on diagnostic-tech-proposal.md Section 3.3 (LLM Prompt Design).
func BuildAnalysisPrompt(sourceCode string) string {
	return fmt.Sprintf(`You are a dataflow analysis expert. Analyze this Python function to discover all dataflow patterns and generate test cases.

**FUNCTION TO ANALYZE**:
`+"```python\n%s\n```"+`

**YOUR TASK**:

1. **DISCOVER PATTERNS** - Identify all dataflow patterns in THIS function:
   - **Sources**: Any operation that introduces new data (user input, file reads, network, env vars, function params, etc.)
   - **Sinks**: Any operation that consumes data (output, storage, exec, system calls, returns, etc.)
   - **Sanitizers**: Any operation that transforms/validates data (escape, quote, validate, cast, etc.)
   - **Propagators**: Operations that pass data along (assignments, calls, returns)

2. **TRACE DATAFLOWS** - For each discovered source:
   - Track where the data flows (which variables, which lines)
   - Identify if it reaches any sinks
   - Note if any sanitizers are applied
   - Track through: assignments, calls, returns, branches, containers, attributes

3. **GENERATE TEST CASES** - Create test cases our tool should pass:
   - Expected flows (source → sink paths)
   - Expected sanitizer detection
   - Expected variable tracking

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
      "reasoning": "Direct flow from user input to OS command without sanitization"
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

1. **NO PREDEFINED PATTERNS**: Discover patterns from the code itself, don't assume
2. **BE SPECIFIC**: Include exact line numbers, variable names, code snippets
3. **TRACK EVERYTHING**: Even non-security dataflows (var assignments, returns, etc.)
4. **SANITIZER EFFECTIVENESS**: Note what each sanitizer actually blocks
5. **GENERATE TESTS**: Each test case should be independently verifiable
6. **CONFIDENCE SCORES**: Rate how confident you are (0.0-1.0)
7. **EXPLAIN REASONING**: Why you think a flow exists or doesn't exist

**EXAMPLE PATTERNS TO DISCOVER**:

Security:
- request.GET/POST/COOKIES → eval/exec/os.system
- input() → open()
- socket.recv() → subprocess.call()

Generic Dataflow:
- function_param → return value
- config['key'] → database.save()
- user.name → logger.info()
- x = calculate() → result = process(x)

**FOCUS**: Validate dataflow tracking algorithm:
- ✅ Track variables through assignments
- ✅ Detect def-use chains correctly
- ✅ Identify taint propagation paths
- ✅ Recognize sanitizers
- ✅ Handle control flow (if/else)
- ✅ Track container operations
- ✅ Track attribute access

Output ONLY the JSON, no additional text.`, sourceCode)
}
