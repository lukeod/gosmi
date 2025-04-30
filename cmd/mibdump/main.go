package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath" // Added import
	"strings"

	"github.com/lukeod/gosmi"
	"github.com/lukeod/gosmi/parser"
	// smi internal types might be needed if gosmi public API isn't sufficient for resolved objects
)

func main() {
	log.SetFlags(0) // Disable log prefixes

	mibFilePath := flag.String("mibfile", "", "Path to the MIB file to parse")
	outputType := flag.String("output", "all", "Type of output: ast, resolved, or all (default)")
	flag.Parse()

	if *mibFilePath == "" {
		log.Fatal("Error: -mibfile flag is required")
	}

	if *outputType != "ast" && *outputType != "resolved" && *outputType != "all" {
		log.Fatalf("Error: invalid -output type %q. Must be 'ast', 'resolved', or 'all'", *outputType)
	}

	// --- Parse AST ---
	var astModule *parser.Module
	var parseErr error
	if *outputType == "ast" || *outputType == "all" {
		log.Printf("Parsing AST from %s...", *mibFilePath)
		astModule, parseErr = parser.ParseFile(*mibFilePath)
		if parseErr != nil {
			log.Printf("Error parsing AST: %v", parseErr)
			// Continue if possible, maybe only resolved output was requested or possible
		} else {
			log.Println("AST parsing successful.")
		}
	}

	// --- Load/Resolve MIB ---
	var resolvedModule gosmi.SmiModule
	var loadErr error
	if *outputType == "resolved" || *outputType == "all" {
		log.Printf("Loading and resolving MIB %s...", *mibFilePath)

		// Initialize gosmi first
		gosmi.Init()
		defer gosmi.Exit()

		// Add the directory of the input MIB to the search path *after* initializing gosmi
		dir := filepath.Dir(*mibFilePath)
		gosmi.PrependPath(dir) // Use PrependPath after Init

		// Explicitly load known base MIBs from the same directory first
		// Ignore errors here, as they might already be loaded or not strictly needed for the target MIB's AST vs. resolution
		baseSmiPath := filepath.Join(dir, "SNMPv2-SMI.mib")
		if _, err := os.Stat(baseSmiPath); err == nil {
			log.Printf("Pre-loading dependency: %s", baseSmiPath)
			_, _ = gosmi.LoadModule(baseSmiPath)
		}
		baseTcPath := filepath.Join(dir, "SNMPv2-TC.mib")
		if _, err := os.Stat(baseTcPath); err == nil {
			log.Printf("Pre-loading dependency: %s", baseTcPath)
			_, _ = gosmi.LoadModule(baseTcPath)
		}

		// Extract module name from file path
		baseName := filepath.Base(*mibFilePath)
		moduleNameFromName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		// Now load the target MIB by its name
		log.Printf("Attempting to load target MIB by name: %s (from path: %s)", moduleNameFromName, *mibFilePath)
		// LoadModule returns the path it found, or an error
		foundPath, loadErr := gosmi.LoadModule(moduleNameFromName)
		if loadErr != nil {
			log.Printf("Error loading/resolving target MIB '%s': %v", moduleNameFromName, loadErr)
			// Continue if possible, maybe only AST output was requested or possible
		} else {
			log.Printf("MIB loading successful (Found at: %s). Fetching resolved module '%s'...", foundPath, moduleNameFromName)
			var getErr error
			// Use the extracted module name to get the resolved module
			resolvedModule, getErr = gosmi.GetModule(moduleNameFromName)
			if getErr != nil {
				log.Printf("Error getting resolved module %s: %v", moduleNameFromName, getErr)
				loadErr = getErr // Combine errors for final check
			} else {
				log.Println("Resolved module fetched successfully.")
			}
		}
	}

	// --- Output JSON ---
	output := make(map[string]interface{})
	output["inputFile"] = *mibFilePath

	if astModule != nil && (*outputType == "ast" || *outputType == "all") {
		output["ast"] = astModule
	}
	if parseErr != nil {
		output["astError"] = parseErr.Error()
	}

	// Check if resolvedModule has been populated (check Name or smiModule pointer)
	if resolvedModule.Name != "" && (*outputType == "resolved" || *outputType == "all") {
		resolvedOutput := make(map[string]interface{})
		resolvedOutput["moduleInfo"] = resolvedModule // Basic module info

		// Fetch and add resolved nodes and types
		log.Println("Fetching resolved nodes...")
		resolvedOutput["nodes"] = resolvedModule.GetNodes() // Get all kinds of nodes
		log.Println("Fetching resolved types...")
		resolvedOutput["types"] = resolvedModule.GetTypes()
		log.Println("Fetching resolved imports...")
		resolvedOutput["imports"] = resolvedModule.GetImports()
		log.Println("Fetching resolved revisions...")
		resolvedOutput["revisions"] = resolvedModule.GetRevisions()
		identityNode, ok := resolvedModule.GetIdentityNode()
		if ok {
			resolvedOutput["identityNode"] = identityNode
		}

		output["resolved"] = resolvedOutput
	}
	if loadErr != nil {
		output["resolvedError"] = loadErr.Error()
	}

	// Check if we actually got any output before failing
	hasOutput := output["ast"] != nil || output["resolved"] != nil
	if !hasOutput && parseErr != nil && loadErr != nil {
		log.Fatalf("Both AST parsing and MIB loading failed.")
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling output to JSON: %v", err)
	}

	fmt.Println(string(jsonOutput))
}
