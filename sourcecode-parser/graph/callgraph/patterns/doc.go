// Package patterns provides security and framework pattern detection.
//
// This package handles:
//   - Security pattern matching (SQL injection, XSS, etc.)
//   - Framework detection (Django, Flask, FastAPI)
//   - Code quality pattern detection
//
// # Pattern Matching
//
//	registry := patterns.NewPatternRegistry()
//	registry.AddPattern(&patterns.Pattern{
//	    ID:         "SQL-INJECTION-001",
//	    Name:       "SQL Injection",
//	    Type:       patterns.PatternTypeMissingSanitizer,
//	    Sources:    []string{"request.GET", "request.POST"},
//	    Sinks:      []string{"execute", "executemany"},
//	    Sanitizers: []string{"escape_sql"},
//	})
//
//	match := registry.MatchPattern(pattern, callGraph)
//	if match.Matched {
//	    fmt.Printf("Found vulnerability: %s -> %s\n",
//	        match.SourceFQN, match.SinkFQN)
//	}
//
// # Framework Detection
//
//	framework := patterns.DetectFramework(importMap)
//	if framework != nil {
//	    fmt.Printf("Using %s (%s)\n",
//	        framework.Name, framework.Category)
//	}
package patterns
