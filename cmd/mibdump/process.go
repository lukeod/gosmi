package main

import (
	"encoding/json"
	"errors" // Added for panic recovery
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug" // Added for panic recovery stack trace
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lukeod/gosmi"
	"github.com/lukeod/gosmi/parser"
	mainline_gosmi "github.com/sleepinggenius2/gosmi"
	mainline_parser "github.com/sleepinggenius2/gosmi/parser"

	"github.com/pmezard/go-difflib/difflib"
)

// processSingleMibFile handles the original logic for a single MIB file
// Note: Parameters are now values, not pointers
func processSingleMibFile(mibFilePath, outputType string, dumpOutput bool) {
	log.Printf("Processing single MIB file: %s\n", mibFilePath)

	// --- Process with Fork (lukeod/gosmi) ---
	log.Println("--- Processing with Fork (lukeod/gosmi) ---")
	var forkAstModule *parser.Module
	var forkParseErr error
	// Use outputType directly (no *)
	if outputType == "ast" || outputType == "all" {
		log.Printf("[Fork] Parsing AST from %s...", mibFilePath)    // Use mibFilePath directly
		forkAstModule, forkParseErr = parser.ParseFile(mibFilePath) // Use mibFilePath directly
		if forkParseErr != nil {
			log.Printf("[Fork] Error parsing AST: %v", forkParseErr)
		} else {
			log.Println("[Fork] AST parsing successful.")
		}
	}

	var forkResolvedModule gosmi.SmiModule
	var forkLoadErr error
	forkResolvedMap := make(map[string]interface{}) // Store resolved data here
	var forkDependencies []DependencyParseResult    // Track dependencies here

	// Use outputType directly (no *)
	if outputType == "resolved" || outputType == "all" {
		log.Printf("[Fork] Loading and resolving MIB %s...", mibFilePath) // Use mibFilePath directly
		gosmi.Init()                                                      // Initialize fork gosmi
		defer gosmi.Exit()                                                // Ensure fork exits eventually

		dir := filepath.Dir(mibFilePath) // Use mibFilePath directly
		gosmi.PrependPath(dir)

		// Simplified dependency loading (assuming they are in the same dir)
		deps := []string{"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "RFC1155-SMI", "RFC1213-MIB"} // Common base MIBs
		for _, dep := range deps {
			depPath := filepath.Join(dir, dep+".mib") // Assuming .mib extension, adjust if needed
			// Check if file exists before trying to load
			if _, statErr := os.Stat(depPath); statErr == nil {
				log.Printf("[Fork] Pre-loading dependency: %s", dep)
				_, loadErr := gosmi.LoadModule(dep)
				// Use the trackDependency function (assuming it's defined in dependencies.go)
				trackDependency(&forkDependencies, dep, depPath, loadErr == nil, loadErr)
			} else if !os.IsNotExist(statErr) {
				// Log other errors like permission issues
				log.Printf("[Fork] Error checking dependency file %s: %v", depPath, statErr)
			}
		}

		baseName := filepath.Base(mibFilePath) // Use mibFilePath directly
		moduleNameFromName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		log.Printf("[Fork] Attempting to load target MIB by name: %s", moduleNameFromName)
		_, loadErr := gosmi.LoadModule(moduleNameFromName) // Load by name
		if loadErr != nil {
			log.Printf("[Fork] Error loading/resolving target MIB %q: %v", moduleNameFromName, loadErr)
			forkLoadErr = loadErr
		} else {
			// Use actual path found or module name to get module data
			log.Printf("[Fork] MIB loading successful. Fetching resolved module %q...", moduleNameFromName)
			var getErr error
			forkResolvedModule, getErr = gosmi.GetModule(moduleNameFromName)
			if getErr != nil {
				log.Printf("[Fork] Error getting resolved module %s: %v", moduleNameFromName, getErr)
				forkLoadErr = getErr // Treat GetModule error as a load error for simplicity
			} else {
				log.Println("[Fork] Resolved module fetched successfully.")
				// Populate the map for comparison/output
				forkResolvedMap["moduleInfo"] = forkResolvedModule
				forkResolvedMap["nodes"] = forkResolvedModule.GetNodes()
				forkResolvedMap["types"] = forkResolvedModule.GetTypes()
				forkResolvedMap["imports"] = forkResolvedModule.GetImports()     // Assuming models.Import
				forkResolvedMap["revisions"] = forkResolvedModule.GetRevisions() // Assuming models.Revision
				identityNode, ok := forkResolvedModule.GetIdentityNode()
				if ok {
					forkResolvedMap["identityNode"] = identityNode
				}
			}
		}
		// gosmi.Exit() is deferred
	}

	// --- Process with Mainline (sleepinggenius2/gosmi) ---
	log.Println("--- Processing with Mainline (sleepinggenius2/gosmi) ---")
	var mainlineAstModule *mainline_parser.Module
	var mainlineParseErr error
	// Use outputType directly (no *)
	if outputType == "ast" || outputType == "all" {
		log.Printf("[Mainline] Parsing AST from %s...", mibFilePath)                 // Use mibFilePath directly
		mainlineAstModule, mainlineParseErr = mainline_parser.ParseFile(mibFilePath) // Use mibFilePath directly
		if mainlineParseErr != nil {
			log.Printf("[Mainline] Error parsing AST: %v", mainlineParseErr)
		} else {
			log.Println("[Mainline] AST parsing successful.")
		}
	}

	var mainlineResolvedModule mainline_gosmi.SmiModule
	var mainlineLoadErr error
	mainlineResolvedMap := make(map[string]interface{}) // Store resolved data here
	var mainlineDependencies []DependencyParseResult    // Track dependencies here

	// Use outputType directly (no *)
	if outputType == "resolved" || outputType == "all" {
		log.Printf("[Mainline] Loading and resolving MIB %s...", mibFilePath) // Use mibFilePath directly
		mainline_gosmi.Init()                                                 // Initialize mainline gosmi
		defer mainline_gosmi.Exit()                                           // Ensure mainline exits eventually

		dir := filepath.Dir(mibFilePath) // Use mibFilePath directly
		mainline_gosmi.PrependPath(dir)

		// Simplified dependency loading
		deps := []string{"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "RFC1155-SMI", "RFC1213-MIB"}
		for _, dep := range deps {
			depPath := filepath.Join(dir, dep+".mib") // Assuming .mib extension
			if _, statErr := os.Stat(depPath); statErr == nil {
				log.Printf("[Mainline] Pre-loading dependency: %s", dep)
				_, loadErr := mainline_gosmi.LoadModule(dep)
				trackDependency(&mainlineDependencies, dep, depPath, loadErr == nil, loadErr)
			} else if !os.IsNotExist(statErr) {
				log.Printf("[Mainline] Error checking dependency file %s: %v", depPath, statErr)
			}
		}

		baseName := filepath.Base(mibFilePath) // Use mibFilePath directly
		moduleNameFromName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		log.Printf("[Mainline] Attempting to load target MIB by name: %s", moduleNameFromName)
		_, loadErr := mainline_gosmi.LoadModule(moduleNameFromName)
		if loadErr != nil {
			log.Printf("[Mainline] Error loading/resolving target MIB %q: %v", moduleNameFromName, loadErr)
			mainlineLoadErr = loadErr
		} else {
			log.Printf("[Mainline] MIB loading successful. Fetching resolved module %q...", moduleNameFromName)
			var getErr error
			mainlineResolvedModule, getErr = mainline_gosmi.GetModule(moduleNameFromName)
			if getErr != nil {
				log.Printf("[Mainline] Error getting resolved module %s: %v", moduleNameFromName, getErr)
				mainlineLoadErr = getErr
			} else {
				log.Println("[Mainline] Resolved module fetched successfully.")
				// Populate the map
				mainlineResolvedMap["moduleInfo"] = mainlineResolvedModule
				mainlineResolvedMap["nodes"] = mainlineResolvedModule.GetNodes()
				mainlineResolvedMap["types"] = mainlineResolvedModule.GetTypes()
				mainlineResolvedMap["imports"] = mainlineResolvedModule.GetImports()     // Assuming mainline_models.Import
				mainlineResolvedMap["revisions"] = mainlineResolvedModule.GetRevisions() // Assuming mainline_models.Revision
				identityNode, ok := mainlineResolvedModule.GetIdentityNode()
				if ok {
					mainlineResolvedMap["identityNode"] = identityNode
				}
			}
		}
		// mainline_gosmi.Exit() is deferred
	}

	// --- Compare Dependencies ---
	var dependencyResults *DependencyResults
	// Use outputType directly (no *)
	if outputType == "resolved" || outputType == "all" {
		// Only compare dependencies if we're doing resolution
		dependencyResults = compareDependencyResults(forkDependencies, mainlineDependencies)
	} else {
		// Create empty dependency results if not doing resolution
		dependencyResults = &DependencyResults{
			HasDifferences: false,
		}
	}

	// --- Prepare Aggregate Results (for potential dump or old diff) ---
	forkResults := make(map[string]interface{})
	// Use outputType directly (no *)
	if forkAstModule != nil && (outputType == "ast" || outputType == "all") {
		forkResults["ast"] = forkAstModule
	}
	if forkParseErr != nil {
		forkResults["astError"] = forkParseErr.Error()
	}
	// Only include 'resolved' if it was actually populated
	if len(forkResolvedMap) > 0 {
		forkResults["resolved"] = forkResolvedMap
	}
	if forkLoadErr != nil {
		forkResults["resolvedError"] = forkLoadErr.Error()
	}

	mainlineResults := make(map[string]interface{})
	// Use outputType directly (no *)
	if mainlineAstModule != nil && (outputType == "ast" || outputType == "all") {
		mainlineResults["ast"] = mainlineAstModule
	}
	if mainlineParseErr != nil {
		mainlineResults["astError"] = mainlineParseErr.Error()
	}
	// Only include 'resolved' if it was actually populated
	if len(mainlineResolvedMap) > 0 {
		mainlineResults["resolved"] = mainlineResolvedMap
	}
	if mainlineLoadErr != nil {
		mainlineResults["resolvedError"] = mainlineLoadErr.Error()
	}

	// --- Output ---
	log.Println("--- Comparison Results ---")

	// Decide output based on flags
	// Use outputType directly (no *)
	outputResolved := (outputType == "resolved" || outputType == "all") && (len(forkResolvedMap) > 0 || len(mainlineResolvedMap) > 0)

	// Use dumpOutput directly (no *)
	if dumpOutput {
		// Dump full JSON including AST and resolved maps
		log.Println("Dumping full JSON output...")
		output := map[string]interface{}{
			"inputFile":        mibFilePath,     // Use mibFilePath directly
			"fork_results":     forkResults,     // Contains potentially ast/resolved/errors
			"mainline_results": mainlineResults, // Contains potentially ast/resolved/errors
		}
		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling output to JSON: %v", err)
		}
		fmt.Println(string(jsonOutput))

	} else if outputResolved {
		// Perform SEMANTIC comparison only if resolved data is present
		log.Println("Performing semantic comparison of resolved MIB data...")
		// Use compareResolvedResults (assuming it's defined in compare.go)
		comparisonResults, err := compareResolvedResults(forkResolvedMap, mainlineResolvedMap)
		if err != nil {
			log.Printf("Error during semantic comparison: %v", err)
			// Fallback to old diff? Or just report error?
			fmt.Println("❌ Semantic comparison failed:", err)

		} else {
			// Check if there are any differences found by the semantic comparison
			// Use hasSemanticDifferences (assuming it's defined in compare.go)
			hasDiffs := hasSemanticDifferences(comparisonResults)

			if !hasDiffs {
				fmt.Println("✅ No semantic differences found in resolved MIB data.")
				fmt.Printf("   Compared: %d Nodes, %d Types\n", comparisonResults.NodesCompared, comparisonResults.TypesCompared) // Provide some context
			} else {
				fmt.Println("⚠️ Semantic differences found in resolved MIB data:")
				// TODO: Implement nice printing of the comparisonResults struct here!
				// For now, just dump the comparison result struct as JSON
				jsonOutput, err := json.MarshalIndent(comparisonResults, "", "  ")
				if err != nil {
					log.Printf("Error marshalling semantic comparison results: %v", err)
					fmt.Println("Could not display semantic differences.")
				} else {
					fmt.Println(string(jsonOutput))
				}
			}
		}

	} else {
		// If only AST was requested or resolution failed for both, use DeepEqual/JSON diff on ASTs if available
		log.Println("Comparing ASTs (or showing errors if resolution failed)...")

		// Check for dependency differences first
		if dependencyResults != nil && dependencyResults.HasDifferences {
			fmt.Println("⚠️ Differences found in dependency parsing:")
			fmt.Println("--- Fork Dependencies ---")
			for _, dep := range dependencyResults.ForkDependencies {
				status := "✅ Success"
				if !dep.Success {
					status = "❌ Failed: " + dep.Error
				}
				fmt.Printf("  %s: %s\n", dep.ModuleName, status)
			}
			fmt.Println("--- Mainline Dependencies ---")
			for _, dep := range dependencyResults.MainlineDependencies {
				status := "✅ Success"
				if !dep.Success {
					status = "❌ Failed: " + dep.Error
				}
				fmt.Printf("  %s: %s\n", dep.ModuleName, status)
			}
		}

		// Select only the AST parts for comparison if possible
		astForkResults := make(map[string]interface{})
		if forkResults["ast"] != nil {
			astForkResults["ast"] = forkResults["ast"]
		}
		if forkResults["astError"] != nil {
			astForkResults["astError"] = forkResults["astError"]
		}

		astMainlineResults := make(map[string]interface{})
		if mainlineResults["ast"] != nil {
			astMainlineResults["ast"] = mainlineResults["ast"]
		}
		if mainlineResults["astError"] != nil {
			astMainlineResults["astError"] = mainlineResults["astError"]
		}

		if reflect.DeepEqual(astForkResults, astMainlineResults) && (dependencyResults == nil || !dependencyResults.HasDifferences) {
			fmt.Println("✅ No differences found in AST, parsing errors, or dependencies.")
		} else {
			fmt.Println("⚠️ Differences found in AST or parsing errors:")
			// Use the old JSON diff method for AST differences or errors
			forkJSON, errFork := json.MarshalIndent(astForkResults, "", "  ")
			mainlineJSON, errMainline := json.MarshalIndent(astMainlineResults, "", "  ")

			if errFork != nil || errMainline != nil {
				log.Printf("Error marshalling AST results for diffing: ForkErr=%v, MainlineErr=%v", errFork, errMainline)
				fmt.Println("Could not generate AST diff due to marshalling errors.")
			} else {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(string(mainlineJSON)),
					B:        difflib.SplitLines(string(forkJSON)),
					FromFile: "Mainline AST/Error",
					ToFile:   "Fork AST/Error",
					Context:  3,
				}
				diffStr, err := difflib.GetUnifiedDiffString(diff)
				if err != nil {
					log.Printf("Error generating AST diff string: %v", err)
					fmt.Println("Could not generate AST diff string.")
				} else if diffStr == "" {
					// This might happen if the only difference is in non-AST parts but we fell into this 'else' block
					fmt.Println("✅ No differences found in AST/Error data (diff string empty).")
				} else {
					fmt.Println("--- AST/Error Diff Summary (Mainline vs Fork) ---")
					fmt.Println(diffStr)
				}
			}
		}
	}

	// Final check for complete failure
	hasForkOutput := forkResults["ast"] != nil || forkResults["resolved"] != nil
	hasMainlineOutput := mainlineResults["ast"] != nil || mainlineResults["resolved"] != nil
	if !hasForkOutput && !hasMainlineOutput && forkParseErr != nil && forkLoadErr != nil && mainlineParseErr != nil && mainlineLoadErr != nil {
		// Use log.Fatal to exit with non-zero status
		log.Fatal("Both fork and mainline processing failed completely.")
	}
}

// compareSingleMibForDir processes a single MIB file for directory comparison mode.
// It initializes gosmi, loads the MIB, performs comparison, and returns results including timing.
// This function includes panic recovery to prevent halting the directory scan.
func compareSingleMibForDir(mibFilePath string) (result DirComparisonResult) {
	// Initialize result struct
	result = DirComparisonResult{FilePath: mibFilePath, Same: false} // Default to not same

	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC recovered while processing %s: %v\nStack trace:\n%s", mibFilePath, r, debug.Stack())
			// Record the panic as an error for both fork and mainline, as we don't know where it occurred.
			// Mark as different.
			errMsg := fmt.Sprintf("panic: %v", r)
			result.ForkError = errors.New(errMsg)
			result.MainlineError = errors.New(errMsg)
			result.Same = false
			// Ensure gosmi instances are cleaned up if they were initialized before panic
			// Note: This relies on gosmi.Exit() being safe to call multiple times or if not initialized.
			// Consider adding checks or more robust cleanup if needed.
			gosmi.Exit()
			mainline_gosmi.Exit()
		}
	}()

	dir := filepath.Dir(mibFilePath)
	baseName := filepath.Base(mibFilePath)
	moduleNameFromName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	var forkResolvedMap map[string]interface{}
	var mainlineResolvedMap map[string]interface{}
	var comparisonResults *ComparisonResults
	var comparisonErr error

	// --- Process with Fork (lukeod/gosmi) ---
	forkStart := time.Now()
	gosmi.Init() // Potential panic point
	gosmi.PrependPath(dir)
	deps := []string{"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "RFC1155-SMI", "RFC1213-MIB"}
	for _, dep := range deps {
		gosmi.LoadModule(dep) // Potential panic point
	}
	_, result.ForkError = gosmi.LoadModule(moduleNameFromName) // Potential panic point
	if result.ForkError == nil {
		var forkModule gosmi.SmiModule
		forkModule, result.ForkError = gosmi.GetModule(moduleNameFromName) // Potential panic point
		if result.ForkError == nil {
			forkResolvedMap = map[string]interface{}{
				"moduleInfo": forkModule,
				"nodes":      forkModule.GetNodes(), // Potential panic point (if module invalid?)
				"types":      forkModule.GetTypes(), // Potential panic point
			}
		}
	}
	result.ForkDuration = time.Since(forkStart)
	gosmi.Exit() // Clean up fork instance

	// --- Process with Mainline (sleepinggenius2/gosmi) ---
	mainlineStart := time.Now()
	mainline_gosmi.Init() // Potential panic point
	mainline_gosmi.PrependPath(dir)
	for _, dep := range deps {
		mainline_gosmi.LoadModule(dep) // Potential panic point
	}
	_, result.MainlineError = mainline_gosmi.LoadModule(moduleNameFromName) // Potential panic point
	if result.MainlineError == nil {
		var mainlineModule mainline_gosmi.SmiModule
		mainlineModule, result.MainlineError = mainline_gosmi.GetModule(moduleNameFromName) // Potential panic point
		if result.MainlineError == nil {
			mainlineResolvedMap = map[string]interface{}{
				"moduleInfo": mainlineModule,
				"nodes":      mainlineModule.GetNodes(), // Potential panic point
				"types":      mainlineModule.GetTypes(), // Potential panic point
			}
		}
	}
	result.MainlineDuration = time.Since(mainlineStart)
	mainline_gosmi.Exit() // Clean up mainline instance

	// --- Compare Results (only if both processed without error/panic) ---
	if result.ForkError == nil && result.MainlineError == nil {
		// Use compareResolvedResults (defined in compare.go)
		comparisonResults, comparisonErr = compareResolvedResults(forkResolvedMap, mainlineResolvedMap) // Potential panic point
		if comparisonErr != nil {
			// Treat comparison error as a difference
			result.Same = false
			// Log comparison error specifically
			log.Printf("Comparison error for %s: %v", mibFilePath, comparisonErr)
			// Store comparison error in a way that distinguishes it from load errors?
			// For now, just marking as different is sufficient.
		} else {
			// Use hasSemanticDifferences (defined in compare.go)
			result.Same = !hasSemanticDifferences(comparisonResults) // Potential panic point
		}
	} else {
		// If either had a load error (or a panic was caught and assigned error), they are not the same
		result.Same = false
	}

	// Result is returned automatically due to named return variable
	return
}

// processDirectory handles the recursive directory processing
func processDirectory(dirPath string) {
	log.Printf("Processing directory recursively: %s\n", dirPath)
	var results []DirComparisonResult
	var mibFilesFound int

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %q: %v\n", path, err)
			return err // Prevent further processing if path is inaccessible
		}
		if !d.IsDir() {
			// Basic MIB file check (can be refined)
			ext := strings.ToLower(filepath.Ext(path))
			// Consider .mib, .txt, and files with no extension as potential MIBs
			if ext == ".mib" || ext == ".txt" || ext == "" {
				log.Printf("Found potential MIB: %s", path)
				mibFilesFound++
				// Call the comparison function for this file
				comparisonResult := compareSingleMibForDir(path)
				results = append(results, comparisonResult)
			}
		}
		return nil // Continue walking
	})

	if err != nil {
		log.Fatalf("Error walking directory %q: %v", dirPath, err)
	}

	log.Printf("Finished processing. Found %d potential MIB files.", mibFilesFound)

	// --- Print Summary Table ---
	if len(results) > 0 {
		fmt.Println("\n--- Directory Comparison Summary ---")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0) // Use spaces for alignment
		// Print header
		fmt.Fprintln(w, "File\tSame?\tFork Error\tMainline Error\tFork Time (ms)\tMainline Time (ms)")
		fmt.Fprintln(w, "----\t-----\t----------\t--------------\t--------------\t------------------")

		// Print results for each file
		for _, res := range results {
			forkErrStr := "nil"
			if res.ForkError != nil {
				forkErrStr = res.ForkError.Error()
				// Truncate long errors for table readability
				if len(forkErrStr) > 50 {
					forkErrStr = forkErrStr[:47] + "..."
				}
			}
			mainlineErrStr := "nil"
			if res.MainlineError != nil {
				mainlineErrStr = res.MainlineError.Error()
				if len(mainlineErrStr) > 50 {
					mainlineErrStr = mainlineErrStr[:47] + "..."
				}
			}
			// Attempt to get relative path for cleaner output
			relPath, err := filepath.Rel(dirPath, res.FilePath)
			if err != nil {
				relPath = res.FilePath // Fallback to absolute path if Rel fails
			}

			// Print formatted row
			fmt.Fprintf(w, "%s\t%t\t%s\t%s\t%d\t%d\n",
				relPath,
				res.Same,
				forkErrStr,
				mainlineErrStr,
				res.ForkDuration.Milliseconds(),
				res.MainlineDuration.Milliseconds(),
			)
		}
		w.Flush() // Ensure all buffered output is written
	} else {
		log.Println("No MIB files processed in the directory.")
	}
}
