package dsl

import (
	"fmt"
	"sync"
)

// DiagnosticCollector collects structured diagnostics during rule execution.
// All methods are nil-safe — calling on a nil receiver is a no-op.
// Thread-safe: multiple goroutines can append concurrently.
type DiagnosticCollector struct {
	entries []DiagnosticEntry
	mu      sync.Mutex
}

// DiagnosticEntry represents a single diagnostic event.
type DiagnosticEntry struct {
	Level     string            // "warning", "error", "debug", "skip"
	Component string            // "ir_validation", "type_match", "arg_match", "fqn_bridge", "dataflow", "executor"
	Message   string            // Human-readable description
	Context   map[string]string // Structured data: rule_id, call_site, file, line, etc.
}

// NewDiagnosticCollector creates a new collector.
func NewDiagnosticCollector() *DiagnosticCollector {
	return &DiagnosticCollector{}
}

// Add appends a diagnostic entry. Nil-safe.
func (dc *DiagnosticCollector) Add(level, component, message string, ctx map[string]string) {
	if dc == nil {
		return
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.entries = append(dc.entries, DiagnosticEntry{
		Level:     level,
		Component: component,
		Message:   message,
		Context:   ctx,
	})
}

// Addf appends a formatted diagnostic entry. Nil-safe.
func (dc *DiagnosticCollector) Addf(level, component, format string, args ...any) {
	if dc == nil {
		return
	}
	dc.Add(level, component, fmt.Sprintf(format, args...), nil)
}

// Entries returns a copy of all collected entries. Nil-safe.
func (dc *DiagnosticCollector) Entries() []DiagnosticEntry {
	if dc == nil {
		return nil
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	result := make([]DiagnosticEntry, len(dc.entries))
	copy(result, dc.entries)
	return result
}

// HasErrors returns true if any entry has level "error". Nil-safe.
func (dc *DiagnosticCollector) HasErrors() bool {
	if dc == nil {
		return false
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	for _, e := range dc.entries {
		if e.Level == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any entry has level "warning". Nil-safe.
func (dc *DiagnosticCollector) HasWarnings() bool {
	if dc == nil {
		return false
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	for _, e := range dc.entries {
		if e.Level == "warning" {
			return true
		}
	}
	return false
}

// FilterByLevel returns entries matching the given level. Nil-safe.
func (dc *DiagnosticCollector) FilterByLevel(level string) []DiagnosticEntry {
	if dc == nil {
		return nil
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	var result []DiagnosticEntry
	for _, e := range dc.entries {
		if e.Level == level {
			result = append(result, e)
		}
	}
	return result
}

// FilterByComponent returns entries matching the given component. Nil-safe.
func (dc *DiagnosticCollector) FilterByComponent(component string) []DiagnosticEntry {
	if dc == nil {
		return nil
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	var result []DiagnosticEntry
	for _, e := range dc.entries {
		if e.Component == component {
			result = append(result, e)
		}
	}
	return result
}

// Count returns the total number of entries. Nil-safe.
func (dc *DiagnosticCollector) Count() int {
	if dc == nil {
		return 0
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return len(dc.entries)
}
