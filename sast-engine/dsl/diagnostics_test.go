package dsl

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnosticCollector_Add(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Add("warning", "ir_validation", "test message", map[string]string{"key": "val"})

	entries := dc.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "warning", entries[0].Level)
	assert.Equal(t, "ir_validation", entries[0].Component)
	assert.Equal(t, "test message", entries[0].Message)
	assert.Equal(t, "val", entries[0].Context["key"])
}

func TestDiagnosticCollector_Addf(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Addf("error", "executor", "panic recovered: %v", "nil pointer")

	entries := dc.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "panic recovered: nil pointer", entries[0].Message)
}

func TestDiagnosticCollector_NilSafe(t *testing.T) {
	var dc *DiagnosticCollector

	// All methods should be no-ops on nil receiver.
	dc.Add("error", "test", "msg", nil)
	dc.Addf("error", "test", "msg %s", "arg")
	assert.Nil(t, dc.Entries())
	assert.False(t, dc.HasErrors())
	assert.False(t, dc.HasWarnings())
	assert.Nil(t, dc.FilterByLevel("error"))
	assert.Nil(t, dc.FilterByComponent("test"))
	assert.Equal(t, 0, dc.Count())
}

func TestDiagnosticCollector_ConcurrentSafe(t *testing.T) {
	dc := NewDiagnosticCollector()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			dc.Addf("debug", "test", "entry %d", n)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 100, dc.Count())
}

func TestDiagnosticCollector_HasErrors(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.False(t, dc.HasErrors())

	dc.Add("warning", "test", "warn", nil)
	assert.False(t, dc.HasErrors())

	dc.Add("error", "test", "err", nil)
	assert.True(t, dc.HasErrors())
}

func TestDiagnosticCollector_HasWarnings(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.False(t, dc.HasWarnings())

	dc.Add("error", "test", "err", nil)
	assert.False(t, dc.HasWarnings())

	dc.Add("warning", "test", "warn", nil)
	assert.True(t, dc.HasWarnings())
}

func TestDiagnosticCollector_FilterByLevel(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Add("error", "a", "e1", nil)
	dc.Add("warning", "b", "w1", nil)
	dc.Add("error", "c", "e2", nil)
	dc.Add("debug", "d", "d1", nil)

	errors := dc.FilterByLevel("error")
	assert.Len(t, errors, 2)

	warnings := dc.FilterByLevel("warning")
	assert.Len(t, warnings, 1)

	debugs := dc.FilterByLevel("debug")
	assert.Len(t, debugs, 1)

	skips := dc.FilterByLevel("skip")
	assert.Empty(t, skips)
}

func TestDiagnosticCollector_FilterByComponent(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Add("error", "ir_validation", "e1", nil)
	dc.Add("warning", "type_match", "w1", nil)
	dc.Add("error", "ir_validation", "e2", nil)

	irEntries := dc.FilterByComponent("ir_validation")
	assert.Len(t, irEntries, 2)

	typeEntries := dc.FilterByComponent("type_match")
	assert.Len(t, typeEntries, 1)

	noEntries := dc.FilterByComponent("nonexistent")
	assert.Empty(t, noEntries)
}

func TestDiagnosticCollector_Count(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0, dc.Count())

	dc.Add("error", "test", "msg", nil)
	assert.Equal(t, 1, dc.Count())

	dc.Add("warning", "test", "msg2", nil)
	assert.Equal(t, 2, dc.Count())
}

func TestDiagnosticCollector_EntriesReturnsCopy(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Add("error", "test", "msg", nil)

	entries := dc.Entries()
	entries[0].Message = "modified"

	// Original should be unchanged.
	original := dc.Entries()
	assert.Equal(t, "msg", original[0].Message)
}

func TestDiagnosticCollector_AddWithNilContext(t *testing.T) {
	dc := NewDiagnosticCollector()
	dc.Add("debug", "test", "no context", nil)

	entries := dc.Entries()
	require.Len(t, entries, 1)
	assert.Nil(t, entries[0].Context)
}
