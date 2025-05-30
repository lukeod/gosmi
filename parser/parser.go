package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath" // Added for file path manipulation

	"github.com/alecthomas/participle/v2"
	gosmilexer "github.com/lukeod/gosmi/parser/lexer" // Import the refactored lexer package
)

var (
	// Removed: compressSpace = regexp.MustCompile(`(?:\r?\n *)+`)
	smiParser = participle.MustBuild[Module](
		participle.Lexer(&gosmilexer.LexerDefinition{}), // Pass a pointer to the refactored lexer definition
		participle.Unquote("ExtUTCTime"),                // Keep for now, might be redundant depending on parser needs
		// Removed Map for ObjectIdentifier - handled by handwritten lexer
		// Removed Map for OctetString - handled by handwritten lexer
		// Removed Map for Text - whitespace compression and unquoting now handled directly in lexer's lexText function
		participle.Upper("ExtUTCTime", "BinString", "HexString"), // Keep for consistency if parser expects uppercase
		// Removed Elide("Whitespace", "Comment") - handled by handwritten lexer's NextToken loop
	)
)

// Parse function needs filename argument for v2
func Parse(filename string, r io.Reader) (*Module, error) {
	// Update Parse call signature - Parse now returns the struct and error
	return smiParser.Parse(filename, r)
}

// ParseFile already has filename, update Parse call inside
func ParseFile(path string) (*Module, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Open file: %w", err)
	}
	defer r.Close()
	// Pass filename to Parse
	module, err := Parse(path, r)
	if err != nil {
		// Add filename to error context if helpful
		return module, fmt.Errorf("Parse file %q: %w", path, err)
	}
	return module, nil
}

// LoadMibTree parses the MIB file at rootMibPath and recursively loads all its dependencies
// found within the specified mibDirs. It returns a map of module names to their parsed ASTs.
func LoadMibTree(rootMibPath string, mibDirs []string) (map[string]*Module, error) {
	parsedModules := make(map[string]*Module)
	// Resolve the root MIB path to absolute to handle relative paths consistently
	absRootMibPath, err := filepath.Abs(rootMibPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootMibPath, err)
	}

	// Add the directory of the root MIB to the search paths automatically
	rootMibDir := filepath.Dir(absRootMibPath)
	effectiveMibDirs := append([]string{rootMibDir}, mibDirs...) // Prepend root dir

	err = loadMibRecursive(absRootMibPath, effectiveMibDirs, parsedModules)
	if err != nil {
		return nil, err
	}
	return parsedModules, nil
}

// loadMibRecursive is the helper function for LoadMibTree.
// It parses a single MIB, finds its dependencies, and recursively calls itself.
// It now expects an absolute path for mibPath.
func loadMibRecursive(mibPath string, mibDirs []string, parsedModules map[string]*Module) error {
	// Ensure path is absolute (should be guaranteed by caller LoadMibTree and recursive calls)
	if !filepath.IsAbs(mibPath) {
		return fmt.Errorf("internal error: loadMibRecursive called with relative path %s", mibPath)
	}

	// 1. Parse the current MIB file
	module, err := ParseFile(mibPath)
	if err != nil {
		// Don't treat parse errors as fatal for the whole tree, maybe log?
		// For baseline generation, we probably want it to be fatal.
		return fmt.Errorf("failed to parse MIB %s: %w", mibPath, err)
	}

	moduleName := string(module.Name)
	// Check if already parsed *after* successful parsing to get the canonical module name
	if _, exists := parsedModules[moduleName]; exists {
		return nil // Already parsed this module (cycle or shared dependency)
	}
	parsedModules[moduleName] = module

	// 2. Process imports
	if module.Body.Imports == nil {
		return nil // No imports in this module
	}

	for _, importStmt := range module.Body.Imports {
		dependencyModuleName := string(importStmt.Module)
		if _, exists := parsedModules[dependencyModuleName]; exists {
			continue // Already processed this dependency
		}

		// 3. Find the dependency MIB file
		dependencyMibPath, found := findMibFile(dependencyModuleName, mibDirs)
		if !found {
			// Option: Treat missing dependencies as non-fatal?
			// For baseline, it should probably be an error.
			// Check if it's a known standard MIB that might be implicitly available
			// For now, return error.
			return fmt.Errorf("could not find MIB file for imported module '%s' (needed by '%s') in search paths %v", dependencyModuleName, moduleName, mibDirs)
		}

		// 4. Recursively load the dependency
		// Ensure the found path is absolute before recursing
		absDependencyMibPath, err := filepath.Abs(dependencyMibPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for dependency %s: %w", dependencyMibPath, err)
		}

		err = loadMibRecursive(absDependencyMibPath, mibDirs, parsedModules)
		if err != nil {
			// Make error message more informative
			return fmt.Errorf("failed to load dependency '%s' (from %s, imported by '%s'): %w", dependencyModuleName, absDependencyMibPath, moduleName, err)
		}
	}

	return nil
}

// findMibFile searches for a MIB file corresponding to a module name in the given directories.
// It checks for common extensions like .mib, .txt, or no extension.
// It returns the absolute path if found.
func findMibFile(moduleName string, mibDirs []string) (string, bool) {
	// Common variations of MIB filenames - case might matter on some systems
	possibleFilenames := []string{
		moduleName + ".mib",
		moduleName + ".txt",
		moduleName,
		// Add case variations if needed, e.g., strings.ToUpper(moduleName) + ".MIB"
	}

	for _, dir := range mibDirs {
		// Ensure dir is absolute for consistent results
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fmt.Printf("Warning: could not get absolute path for search directory %s: %v\n", dir, err)
			continue // Skip this directory if path is invalid
		}

		for _, fname := range possibleFilenames {
			path := filepath.Join(absDir, fname)
			if _, err := os.Stat(path); err == nil {
				// Found the file
				return path, true // Return the absolute path
			}
		}
	}

	// Optional: Add search in standard system MIB locations if relevant
	// e.g., /usr/share/snmp/mibs

	return "", false
}
