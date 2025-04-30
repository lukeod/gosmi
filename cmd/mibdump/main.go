package main

import (
	"encoding/json" // Added for error handling
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect" // Added for DeepEqual
	"strings"

	"github.com/lukeod/gosmi" // Added for models used in resolved data
	"github.com/lukeod/gosmi/parser"
	"github.com/lukeod/gosmi/types" // Added for types used in resolved data

	mainline_gosmi "github.com/sleepinggenius2/gosmi" // Added for mainline models
	mainline_parser "github.com/sleepinggenius2/gosmi/parser"
	mainline_types "github.com/sleepinggenius2/gosmi/types" // Added for mainline types

	"github.com/pmezard/go-difflib/difflib" // Added for diffing
)

// --- Semantic Comparison Data Structures ---

const maxExamplesPerCategory = 3 // Limit the number of examples stored

// DependencyParseResult tracks parsing results for a dependency
type DependencyParseResult struct {
	ModuleName string `json:"moduleName"`
	Path       string `json:"path"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// DependencyResults tracks parsing results for all dependencies
type DependencyResults struct {
	ForkDependencies     []DependencyParseResult `json:"forkDependencies"`
	MainlineDependencies []DependencyParseResult `json:"mainlineDependencies"`
	HasDifferences       bool                    `json:"hasDifferences"`
}

// ValuePair stores the value from both fork and mainline for a differing field.
type ValuePair struct {
	Fork     interface{} `json:"fork"`
	Mainline interface{} `json:"mainline"`
}

// ModuleInfoDifference holds differences found in the SmiModule attributes.
type ModuleInfoDifference struct {
	FieldName string    `json:"fieldName"`
	Diff      ValuePair `json:"diff"`
}

// NodeDifference holds details about a modified SmiNode.
type NodeDifference struct {
	Name     string                 `json:"name"`
	Oid      string                 `json:"oid"` // Use OID for reliable identification if names clash
	KindDiff *ValuePair             `json:"kindDiff,omitempty"`
	Diffs    []ModuleInfoDifference `json:"diffs"` // Generic list of field differences
}

// TypeDifference holds details about a modified SmiType.
type TypeDifference struct {
	Name     string                 `json:"name"`
	BaseType string                 `json:"baseType"` // Use BaseType for identification
	KindDiff *ValuePair             `json:"kindDiff,omitempty"`
	Diffs    []ModuleInfoDifference `json:"diffs"` // Generic list of field differences
}

// SimplifiedNode represents a node for addition/removal reporting.
type SimplifiedNode struct {
	Name string `json:"name"`
	Oid  string `json:"oid"`
	Kind string `json:"kind"` // Kind as string for easier comparison
}

// SimplifiedType represents a type for addition/removal reporting.
type SimplifiedType struct {
	Name     string `json:"name"`
	BaseType string `json:"baseType"` // BaseType as string
	Kind     string `json:"kind"`     // Kind as string
}

// ComparisonResults holds the results of the semantic comparison.
type ComparisonResults struct {
	ModuleInfoDiffs []ModuleInfoDifference `json:"moduleInfoDiffs,omitempty"`

	NodesAdded         []SimplifiedNode `json:"nodesAdded,omitempty"`
	NodesRemoved       []SimplifiedNode `json:"nodesRemoved,omitempty"`
	NodesModified      []NodeDifference `json:"nodesModified,omitempty"`
	NodesCompared      int              `json:"nodesCompared"`   // Total nodes considered from both sides
	NodesOk            int              `json:"nodesOk"`         // Nodes present in both and identical
	NodesAddedCount    int              `json:"nodesAddedCount"` // Total count if > maxExamples
	NodesRemovedCount  int              `json:"nodesRemovedCount"`
	NodesModifiedCount int              `json:"nodesModifiedCount"`

	TypesAdded         []SimplifiedType `json:"typesAdded,omitempty"`
	TypesRemoved       []SimplifiedType `json:"typesRemoved,omitempty"`
	TypesModified      []TypeDifference `json:"typesModified,omitempty"`
	TypesCompared      int              `json:"typesCompared"`
	TypesOk            int              `json:"typesOk"`
	TypesAddedCount    int              `json:"typesAddedCount"`
	TypesRemovedCount  int              `json:"typesRemovedCount"`
	TypesModifiedCount int              `json:"typesModifiedCount"`

	// TODO: Add fields for Imports, Revisions, Identity Node differences
	// ImportsAdded/Removed/Modified
	// RevisionsAdded/Removed/Modified
	// IdentityNodeDiff *NodeDifference
}

// --- Semantic Comparison Logic ---

// Helper to safely get string representation of NodeKind
func getNodeKindString(kind interface{}) string {
	switch k := kind.(type) {
	case types.NodeKind:
		return k.String()
	case mainline_types.NodeKind:
		return k.String()
	default:
		return fmt.Sprintf("unknown_kind(%T)", kind)
	}
}

// Helper to safely get string representation of BaseType
func getBaseTypeString(basetype interface{}) string {
	switch bt := basetype.(type) {
	case types.BaseType:
		return bt.String()
	case mainline_types.BaseType:
		return bt.String()
	default:
		return fmt.Sprintf("unknown_basetype(%T)", basetype)
	}
}

// compareModuleInfo compares attributes of the main SmiModule.
func compareModuleInfo(forkModule gosmi.SmiModule, mainlineModule mainline_gosmi.SmiModule) []ModuleInfoDifference {
	diffs := []ModuleInfoDifference{}

	// Compare basic fields
	if forkModule.Name != mainlineModule.Name {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Name", Diff: ValuePair{Fork: forkModule.Name, Mainline: mainlineModule.Name}})
	}
	if forkModule.Path != mainlineModule.Path {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Path", Diff: ValuePair{Fork: forkModule.Path, Mainline: mainlineModule.Path}})
	}
	if forkModule.Organization != mainlineModule.Organization {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Organization", Diff: ValuePair{Fork: forkModule.Organization, Mainline: mainlineModule.Organization}})
	}
	if forkModule.ContactInfo != mainlineModule.ContactInfo {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "ContactInfo", Diff: ValuePair{Fork: forkModule.ContactInfo, Mainline: mainlineModule.ContactInfo}})
	}
	if forkModule.Description != mainlineModule.Description {
		// Normalize whitespace for description comparison? Maybe too aggressive.
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Description", Diff: ValuePair{Fork: forkModule.Description, Mainline: mainlineModule.Description}})
	}
	if forkModule.Language.String() != mainlineModule.Language.String() { // Compare string representations of Language enum
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Language", Diff: ValuePair{Fork: forkModule.Language.String(), Mainline: mainlineModule.Language.String()}})
	}
	// Conformance field doesn't exist in either SmiModule type, so removing this comparison

	// TODO: Compare Imports, Revisions, Identity Node (or handle them in separate functions)

	return diffs
}

// compareNodes compares slices of SmiNode from both versions.
func compareNodes(forkNodes []gosmi.SmiNode, mainlineNodes []mainline_gosmi.SmiNode) (
	added []SimplifiedNode, removed []SimplifiedNode, modified []NodeDifference,
	comparedCount, okCount, addedTotal, removedTotal, modifiedTotal int) {

	forkNodeMap := make(map[string]gosmi.SmiNode) // Keyed by OID string
	for _, node := range forkNodes {
		forkNodeMap[node.Oid.String()] = node
	}

	mainlineNodeMap := make(map[string]mainline_gosmi.SmiNode) // Keyed by OID string
	for _, node := range mainlineNodes {
		mainlineNodeMap[node.Oid.String()] = node
	}

	comparedCount = len(forkNodeMap) + len(mainlineNodeMap) // Initial estimate, adjusted later

	processedMainlineOids := make(map[string]bool)

	for oid, forkNode := range forkNodeMap {
		mainlineNode, existsInMainline := mainlineNodeMap[oid]
		processedMainlineOids[oid] = true // Mark this OID as seen

		if existsInMainline {
			// Compare the nodes
			nodeDiffs := []ModuleInfoDifference{}
			var kindDiff *ValuePair

			// Compare Kind first
			forkKindStr := getNodeKindString(forkNode.Kind)
			mainlineKindStr := getNodeKindString(mainlineNode.Kind)
			if forkKindStr != mainlineKindStr {
				kindDiff = &ValuePair{Fork: forkKindStr, Mainline: mainlineKindStr}
			}

			// Compare other relevant fields based on kind? Or generically?
			// Generic approach: Compare common fields first.
			if forkNode.Name != mainlineNode.Name {
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Name", Diff: ValuePair{Fork: forkNode.Name, Mainline: mainlineNode.Name}})
			}
			if forkNode.Status.String() != mainlineNode.Status.String() { // Compare string representation for enum
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Status", Diff: ValuePair{Fork: forkNode.Status.String(), Mainline: mainlineNode.Status.String()}})
			}
			if forkNode.Description != mainlineNode.Description {
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Description", Diff: ValuePair{Fork: forkNode.Description, Mainline: mainlineNode.Description}})
			}
			// Reference field doesn't exist in SmiNode, so removing this comparison

			// Format field doesn't exist in SmiNode, so removing this comparison

			// Units field doesn't exist in SmiNode, so removing this comparison
			if forkNode.Access.String() != mainlineNode.Access.String() { // Access enum
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Access", Diff: ValuePair{Fork: forkNode.Access.String(), Mainline: mainlineNode.Access.String()}})
			}
			if forkNode.Decl.String() != mainlineNode.Decl.String() { // Decl enum
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Decl", Diff: ValuePair{Fork: forkNode.Decl.String(), Mainline: mainlineNode.Decl.String()}})
			}
			// TODO: Compare Type? This requires comparing SmiType objects, might need a dedicated helper or compare base type name.
			// Compare forkNode.Type.Name vs mainlineNode.Type.Name ?
			// Compare forkNode.Type.BaseType vs mainlineNode.Type.BaseType ?

			// IndexKind field doesn't exist in SmiNode, so removing this comparison

			// TODO: Compare Create (bool?)
			// TODO: Compare Elements (slice of SmiElement) - needs deeper comparison

			if kindDiff != nil || len(nodeDiffs) > 0 {
				modifiedTotal++
				if len(modified) < maxExamplesPerCategory {
					modified = append(modified, NodeDifference{
						Name:     forkNode.Name, // Use fork\'s name as primary identifier in report
						Oid:      oid,
						KindDiff: kindDiff,
						Diffs:    nodeDiffs,
					})
				}
			} else {
				okCount++
			}
		} else {
			// Node exists in Fork but not in Mainline -> Added
			addedTotal++
			if len(added) < maxExamplesPerCategory {
				added = append(added, SimplifiedNode{
					Name: forkNode.Name,
					Oid:  oid,
					Kind: getNodeKindString(forkNode.Kind),
				})
			}
		}
	}

	// Check for nodes in Mainline but not processed (i.e., not in Fork) -> Removed
	for oid, mainlineNode := range mainlineNodeMap {
		if !processedMainlineOids[oid] {
			removedTotal++
			if len(removed) < maxExamplesPerCategory {
				removed = append(removed, SimplifiedNode{
					Name: mainlineNode.Name,
					Oid:  oid,
					Kind: getNodeKindString(mainlineNode.Kind),
				})
			}
		}
	}

	// Adjust compared count - it's the number of unique OIDs across both maps
	comparedCount = okCount + addedTotal + removedTotal + modifiedTotal

	return added, removed, modified, comparedCount, okCount, addedTotal, removedTotal, modifiedTotal
}

// compareTypes compares slices of SmiType from both versions.
func compareTypes(forkTypes []gosmi.SmiType, mainlineTypes []mainline_gosmi.SmiType) (
	added []SimplifiedType, removed []SimplifiedType, modified []TypeDifference,
	comparedCount, okCount, addedTotal, removedTotal, modifiedTotal int) {

	// Key: ModuleName.TypeName (assuming types are module-scoped, need confirmation)
	// Alternative key: Just TypeName if globally unique within comparison context. Let's try Name first.
	forkTypeMap := make(map[string]gosmi.SmiType)
	for _, t := range forkTypes {
		forkTypeMap[t.Name] = t
	}

	mainlineTypeMap := make(map[string]mainline_gosmi.SmiType)
	for _, t := range mainlineTypes {
		mainlineTypeMap[t.Name] = t
	}

	comparedCount = len(forkTypeMap) + len(mainlineTypeMap) // Initial estimate

	processedMainlineNames := make(map[string]bool)

	for name, forkType := range forkTypeMap {
		mainlineType, existsInMainline := mainlineTypeMap[name]
		processedMainlineNames[name] = true

		if existsInMainline {
			// Compare the types
			typeDiffs := []ModuleInfoDifference{}
			var kindDiff *ValuePair // If kind concept applies to types

			forkBaseTypeStr := getBaseTypeString(forkType.BaseType)
			mainlineBaseTypeStr := getBaseTypeString(mainlineType.BaseType)

			if forkBaseTypeStr != mainlineBaseTypeStr {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "BaseType", Diff: ValuePair{Fork: forkBaseTypeStr, Mainline: mainlineBaseTypeStr}})
			}
			if forkType.Format != mainlineType.Format {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Format", Diff: ValuePair{Fork: forkType.Format, Mainline: mainlineType.Format}})
			}
			if forkType.Status.String() != mainlineType.Status.String() {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Status", Diff: ValuePair{Fork: forkType.Status.String(), Mainline: mainlineType.Status.String()}})
			}
			if forkType.Description != mainlineType.Description {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Description", Diff: ValuePair{Fork: forkType.Description, Mainline: mainlineType.Description}})
			}
			if forkType.Reference != mainlineType.Reference {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Reference", Diff: ValuePair{Fork: forkType.Reference, Mainline: mainlineType.Reference}})
			}
			if forkType.Units != mainlineType.Units {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Units", Diff: ValuePair{Fork: forkType.Units, Mainline: mainlineType.Units}})
			}
			if forkType.Decl.String() != mainlineType.Decl.String() {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Decl", Diff: ValuePair{Fork: forkType.Decl.String(), Mainline: mainlineType.Decl.String()}})
			}

			// TODO: Compare NamedNumbers (slice) - needs deeper comparison (name, value)
			// TODO: Compare Ranges (slice) - needs deeper comparison (min, max, baseType)

			if kindDiff != nil || len(typeDiffs) > 0 {
				modifiedTotal++
				if len(modified) < maxExamplesPerCategory {
					modified = append(modified, TypeDifference{
						Name:     name,
						BaseType: forkBaseTypeStr, // Use fork's base type string
						KindDiff: kindDiff,        // If applicable
						Diffs:    typeDiffs,
					})
				}
			} else {
				okCount++
			}
		} else {
			// Type exists in Fork but not in Mainline -> Added
			addedTotal++
			if len(added) < maxExamplesPerCategory {
				added = append(added, SimplifiedType{
					Name:     name,
					BaseType: getBaseTypeString(forkType.BaseType),
					Kind:     "", // Add kind if relevant for types
				})
			}
		}
	}

	// Check for types in Mainline but not processed -> Removed
	for name, mainlineType := range mainlineTypeMap {
		if !processedMainlineNames[name] {
			removedTotal++
			if len(removed) < maxExamplesPerCategory {
				removed = append(removed, SimplifiedType{
					Name:     name,
					BaseType: getBaseTypeString(mainlineType.BaseType),
					Kind:     "", // Add kind if relevant
				})
			}
		}
	}

	comparedCount = okCount + addedTotal + removedTotal + modifiedTotal

	return added, removed, modified, comparedCount, okCount, addedTotal, removedTotal, modifiedTotal
}

// compareResolvedResults performs the main semantic comparison.
func compareResolvedResults(forkResolved, mainlineResolved map[string]interface{}) (*ComparisonResults, error) {
	results := &ComparisonResults{}

	// --- Type Assertions and Data Extraction ---
	var forkModule gosmi.SmiModule
	var mainlineModule mainline_gosmi.SmiModule
	var forkNodes []gosmi.SmiNode
	var mainlineNodes []mainline_gosmi.SmiNode
	var forkTypes []gosmi.SmiType
	var mainlineTypes []mainline_gosmi.SmiType
	// TODO: Extract Imports, Revisions, Identity Node

	// Fork Data
	fm, ok := forkResolved["moduleInfo"].(gosmi.SmiModule)
	if !ok {
		// Handle case where fork failed to resolve module info
		// return nil, errors.New("fork result missing or invalid 'moduleInfo'")
		// Allow comparison to proceed, but ModuleInfoDiffs will be empty/based on nil vs mainline
	} else {
		forkModule = fm
	}

	fn, ok := forkResolved["nodes"].([]gosmi.SmiNode)
	if !ok {
		// Allow comparison with empty slice
	} else {
		forkNodes = fn
	}

	ft, ok := forkResolved["types"].([]gosmi.SmiType)
	if !ok {
		// Allow comparison with empty slice
	} else {
		forkTypes = ft
	}

	// Mainline Data
	mm, ok := mainlineResolved["moduleInfo"].(mainline_gosmi.SmiModule)
	if !ok {
		// Handle case where mainline failed to resolve module info
		// return nil, errors.New("mainline result missing or invalid 'moduleInfo'")
	} else {
		mainlineModule = mm
	}

	mn, ok := mainlineResolved["nodes"].([]mainline_gosmi.SmiNode)
	if !ok {
		// Allow comparison with empty slice
	} else {
		mainlineNodes = mn
	}

	mt, ok := mainlineResolved["types"].([]mainline_gosmi.SmiType)
	if !ok {
		// Allow comparison with empty slice
	} else {
		mainlineTypes = mt
	}

	// --- Perform Comparisons ---

	// Compare Module Info
	// Need to handle cases where one or both might be zero-value structs if resolution failed
	if forkModule.Name != "" || mainlineModule.Name != "" { // Only compare if at least one module was resolved
		results.ModuleInfoDiffs = compareModuleInfo(forkModule, mainlineModule)
	}

	// Compare Nodes
	results.NodesAdded, results.NodesRemoved, results.NodesModified,
		results.NodesCompared, results.NodesOk, results.NodesAddedCount, results.NodesRemovedCount, results.NodesModifiedCount = compareNodes(forkNodes, mainlineNodes)

	// Compare Types
	results.TypesAdded, results.TypesRemoved, results.TypesModified,
		results.TypesCompared, results.TypesOk, results.TypesAddedCount, results.TypesRemovedCount, results.TypesModifiedCount = compareTypes(forkTypes, mainlineTypes)

	// TODO: Call comparison functions for Imports, Revisions, Identity Node

	return results, nil
}

// --- Dependency Tracking Functions ---

// trackDependency adds a dependency parsing result to the tracking list
func trackDependency(dependencies *[]DependencyParseResult, moduleName, path string, success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	*dependencies = append(*dependencies, DependencyParseResult{
		ModuleName: moduleName,
		Path:       path,
		Success:    success,
		Error:      errStr,
	})
}

// compareDependencyResults compares fork and mainline dependency parsing results
func compareDependencyResults(forkDeps, mainlineDeps []DependencyParseResult) *DependencyResults {
	result := &DependencyResults{
		ForkDependencies:     forkDeps,
		MainlineDependencies: mainlineDeps,
		HasDifferences:       false,
	}

	// Create maps for quick lookup
	forkMap := make(map[string]DependencyParseResult)
	for _, dep := range forkDeps {
		forkMap[dep.ModuleName] = dep
	}

	mainlineMap := make(map[string]DependencyParseResult)
	for _, dep := range mainlineDeps {
		mainlineMap[dep.ModuleName] = dep
	}

	// Check for differences
	for name, forkDep := range forkMap {
		if mainlineDep, exists := mainlineMap[name]; exists {
			if forkDep.Success != mainlineDep.Success {
				result.HasDifferences = true
				break
			}
		} else {
			// Dependency exists in fork but not in mainline
			result.HasDifferences = true
			break
		}
	}

	// Check for dependencies in mainline but not in fork
	for name := range mainlineMap {
		if _, exists := forkMap[name]; !exists {
			result.HasDifferences = true
			break
		}
	}

	return result
}

// --- Main Function ---

func main() {
	log.SetFlags(0) // Disable log prefixes

	mibFilePath := flag.String("mibfile", "", "Path to the MIB file to parse")
	outputType := flag.String("output", "all", "Type of output: ast, resolved, or all (default)")
	dumpOutput := flag.Bool("dump", false, "Dump the full JSON output instead of a diff summary")
	flag.Parse()

	if *mibFilePath == "" {
		log.Fatal("Error: -mibfile flag is required")
	}

	if *outputType != "ast" && *outputType != "resolved" && *outputType != "all" {
		log.Fatalf("Error: invalid -output type %q. Must be 'ast', 'resolved', or 'all'", *outputType)
	}

	// --- Process with Fork (lukeod/gosmi) ---
	log.Println("--- Processing with Fork (lukeod/gosmi) ---")
	var forkAstModule *parser.Module
	var forkParseErr error
	if *outputType == "ast" || *outputType == "all" {
		log.Printf("[Fork] Parsing AST from %s...", *mibFilePath)
		forkAstModule, forkParseErr = parser.ParseFile(*mibFilePath)
		if forkParseErr != nil {
			log.Printf("[Fork] Error parsing AST: %v", forkParseErr)
		} else {
			log.Println("[Fork] AST parsing successful.")
		}
	}

	var forkResolvedModule gosmi.SmiModule
	var forkLoadErr error
	forkResolvedMap := make(map[string]interface{}) // Store resolved data here
	if *outputType == "resolved" || *outputType == "all" {
		log.Printf("[Fork] Loading and resolving MIB %s...", *mibFilePath)
		gosmi.Init()       // Initialize fork gosmi
		defer gosmi.Exit() // Ensure fork exits eventually

		dir := filepath.Dir(*mibFilePath)
		gosmi.PrependPath(dir)

		// Track dependency parsing results
		var forkDependencies []DependencyParseResult

		// Simplified dependency loading (assuming they are in the same dir)
		deps := []string{"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "RFC1155-SMI", "RFC1213-MIB"} // Common base MIBs
		for _, dep := range deps {
			depPath := filepath.Join(dir, dep+".mib")
			if _, err := os.Stat(depPath); err == nil {
				log.Printf("[Fork] Pre-loading dependency: %s", dep)
				_, loadErr := gosmi.LoadModule(dep)
				trackDependency(&forkDependencies, dep, depPath, loadErr == nil, loadErr)
			}
		}

		baseName := filepath.Base(*mibFilePath)
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
	if *outputType == "ast" || *outputType == "all" {
		log.Printf("[Mainline] Parsing AST from %s...", *mibFilePath)
		mainlineAstModule, mainlineParseErr = mainline_parser.ParseFile(*mibFilePath)
		if mainlineParseErr != nil {
			log.Printf("[Mainline] Error parsing AST: %v", mainlineParseErr)
		} else {
			log.Println("[Mainline] AST parsing successful.")
		}
	}

	var mainlineResolvedModule mainline_gosmi.SmiModule
	var mainlineLoadErr error
	mainlineResolvedMap := make(map[string]interface{}) // Store resolved data here
	if *outputType == "resolved" || *outputType == "all" {
		log.Printf("[Mainline] Loading and resolving MIB %s...", *mibFilePath)
		mainline_gosmi.Init()       // Initialize mainline gosmi
		defer mainline_gosmi.Exit() // Ensure mainline exits eventually

		dir := filepath.Dir(*mibFilePath)
		mainline_gosmi.PrependPath(dir)
		// Track dependency parsing results
		var mainlineDependencies []DependencyParseResult

		// Simplified dependency loading
		deps := []string{"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "RFC1155-SMI", "RFC1213-MIB"}
		for _, dep := range deps {
			depPath := filepath.Join(dir, dep+".mib")
			if _, err := os.Stat(depPath); err == nil {
				log.Printf("[Mainline] Pre-loading dependency: %s", dep)
				_, loadErr := mainline_gosmi.LoadModule(dep)
				trackDependency(&mainlineDependencies, dep, depPath, loadErr == nil, loadErr)
			}
		}

		baseName := filepath.Base(*mibFilePath)
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
	if *outputType == "resolved" || *outputType == "all" {
		// Only compare dependencies if we're doing resolution
		forkDeps := []DependencyParseResult{}
		mainlineDeps := []DependencyParseResult{}

		// Extract dependencies from the resolved maps if they exist
		if deps, ok := forkResolvedMap["dependencies"].([]DependencyParseResult); ok {
			forkDeps = deps
		}
		if deps, ok := mainlineResolvedMap["dependencies"].([]DependencyParseResult); ok {
			mainlineDeps = deps
		}

		dependencyResults = compareDependencyResults(forkDeps, mainlineDeps)
	} else {
		// Create empty dependency results if not doing resolution
		dependencyResults = &DependencyResults{
			HasDifferences: false,
		}
	}

	// --- Prepare Aggregate Results (for potential dump or old diff) ---
	forkResults := make(map[string]interface{})
	if forkAstModule != nil && (*outputType == "ast" || *outputType == "all") {
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
	if mainlineAstModule != nil && (*outputType == "ast" || *outputType == "all") {
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
	outputResolved := (*outputType == "resolved" || *outputType == "all") && (len(forkResolvedMap) > 0 || len(mainlineResolvedMap) > 0)

	if *dumpOutput {
		// Dump full JSON including AST and resolved maps
		log.Println("Dumping full JSON output...")
		output := map[string]interface{}{
			"inputFile":        *mibFilePath,
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
		comparisonResults, err := compareResolvedResults(forkResolvedMap, mainlineResolvedMap)
		if err != nil {
			log.Printf("Error during semantic comparison: %v", err)
			// Fallback to old diff? Or just report error?
			fmt.Println("❌ Semantic comparison failed:", err)

		} else {
			// Check if there are any differences found by the semantic comparison
			hasDiffs := len(comparisonResults.ModuleInfoDiffs) > 0 ||
				comparisonResults.NodesAddedCount > 0 || comparisonResults.NodesRemovedCount > 0 || comparisonResults.NodesModifiedCount > 0 ||
				comparisonResults.TypesAddedCount > 0 || comparisonResults.TypesRemovedCount > 0 || comparisonResults.TypesModifiedCount > 0 // || Add checks for Imports, Revisions, Identity

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
					fmt.Println("✅ No differences found (AST diff string empty).") // Fallback
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
		log.Fatalf("Both fork and mainline processing failed completely.")
	}
}
