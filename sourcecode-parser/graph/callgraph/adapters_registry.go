package callgraph

// registerAdapters registers all language adapters with the global registry.
// This is called from init() in builder.go after the registry is created.
// To avoid import cycles, actual registration happens via RegisterPythonAdapter.
func registerAdapters() {
	// NOTE: Python adapter will register itself via init() in adapters/python package
	// This function exists as a placeholder for future manual registration if needed
}
