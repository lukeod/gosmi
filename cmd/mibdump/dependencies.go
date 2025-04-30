package main

// --- Dependency Tracking Functions ---

// trackDependency adds a dependency parsing result to the tracking list.
// It uses the DependencyParseResult type defined in types.go.
func trackDependency(dependencies *[]DependencyParseResult, moduleName, path string, success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error() // Convert error to string if it exists
	}
	*dependencies = append(*dependencies, DependencyParseResult{
		ModuleName: moduleName,
		Path:       path,
		Success:    success,
		Error:      errStr,
	})
}

// Note: The compareDependencyResults function remains in compare.go as it's part of the comparison logic,
// even though it operates on dependency results.
