package main

import (
	"github.com/lukeod/gosmi"
	mainline_gosmi "github.com/sleepinggenius2/gosmi"
	// Note: types and mainline_types are implicitly used via the structs defined in types.go
	// We might need "fmt" if we uncomment the detailed unknown type logging in helpers (moved to types.go)
)

// --- Semantic Comparison Logic ---

// compareModuleInfo compares attributes of the main SmiModule.
func compareModuleInfo(forkModule gosmi.SmiModule, mainlineModule mainline_gosmi.SmiModule) []ModuleInfoDifference {
	diffs := []ModuleInfoDifference{}

	// Compare basic fields
	if forkModule.Name != mainlineModule.Name {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Name", Diff: ValuePair{Fork: forkModule.Name, Mainline: mainlineModule.Name}})
	}
	// Path comparison might be noisy if libs load from different places, maybe skip?
	// if forkModule.Path != mainlineModule.Path {
	// 	diffs = append(diffs, ModuleInfoDifference{FieldName: "Path", Diff: ValuePair{Fork: forkModule.Path, Mainline: mainlineModule.Path}})
	// }
	if forkModule.Organization != mainlineModule.Organization {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "Organization", Diff: ValuePair{Fork: forkModule.Organization, Mainline: mainlineModule.Organization}})
	}
	if forkModule.ContactInfo != mainlineModule.ContactInfo {
		diffs = append(diffs, ModuleInfoDifference{FieldName: "ContactInfo", Diff: ValuePair{Fork: forkModule.ContactInfo, Mainline: mainlineModule.ContactInfo}})
	}
	if forkModule.Description != mainlineModule.Description {
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

	// Use maxExamplesPerCategory constant (defined in types.go)
	// comparedCount = len(forkNodeMap) + len(mainlineNodeMap) // Initial estimate, adjusted later

	processedMainlineOids := make(map[string]bool)

	for oid, forkNode := range forkNodeMap {
		mainlineNode, existsInMainline := mainlineNodeMap[oid]
		processedMainlineOids[oid] = true // Mark this OID as seen

		if existsInMainline {
			// Compare the nodes
			nodeDiffs := []ModuleInfoDifference{}
			var kindDiff *ValuePair

			// Compare Kind first using helper from types.go
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
			// Reference field doesn't exist in SmiNode
			// Format field doesn't exist in SmiNode
			// Units field doesn't exist in SmiNode
			if forkNode.Access.String() != mainlineNode.Access.String() { // Access enum
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Access", Diff: ValuePair{Fork: forkNode.Access.String(), Mainline: mainlineNode.Access.String()}})
			}
			if forkNode.Decl.String() != mainlineNode.Decl.String() { // Decl enum
				nodeDiffs = append(nodeDiffs, ModuleInfoDifference{FieldName: "Decl", Diff: ValuePair{Fork: forkNode.Decl.String(), Mainline: mainlineNode.Decl.String()}})
			}
			// TODO: Compare Type? This requires comparing SmiType objects, might need a dedicated helper or compare base type name.
			// Compare forkNode.Type.Name vs mainlineNode.Type.Name ?
			// Compare forkNode.Type.BaseType vs mainlineNode.Type.BaseType ?

			// IndexKind field doesn't exist in SmiNode
			// TODO: Compare Create (bool?)
			// TODO: Compare Elements (slice of SmiElement) - needs deeper comparison

			if kindDiff != nil || len(nodeDiffs) > 0 {
				modifiedTotal++
				if len(modified) < maxExamplesPerCategory {
					modified = append(modified, NodeDifference{
						Name:     forkNode.Name, // Use fork's name as primary identifier in report
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
					Kind: getNodeKindString(forkNode.Kind), // Use helper
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
					Kind: getNodeKindString(mainlineNode.Kind), // Use helper
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

	// Use maxExamplesPerCategory constant (defined in types.go)
	// comparedCount = len(forkTypeMap) + len(mainlineTypeMap) // Initial estimate

	processedMainlineNames := make(map[string]bool)

	for name, forkType := range forkTypeMap {
		mainlineType, existsInMainline := mainlineTypeMap[name]
		processedMainlineNames[name] = true

		if existsInMainline {
			// Compare the types
			typeDiffs := []ModuleInfoDifference{}
			var kindDiff *ValuePair // If kind concept applies to types

			// Use helper from types.go
			forkBaseTypeStr := getBaseTypeString(forkType.BaseType)
			mainlineBaseTypeStr := getBaseTypeString(mainlineType.BaseType)

			if forkBaseTypeStr != mainlineBaseTypeStr {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "BaseType", Diff: ValuePair{Fork: forkBaseTypeStr, Mainline: mainlineBaseTypeStr}})
			}
			if forkType.Format != mainlineType.Format {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Format", Diff: ValuePair{Fork: forkType.Format, Mainline: mainlineType.Format}})
			}
			// Description comparison was missing, adding it back
			if forkType.Description != mainlineType.Description {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Description", Diff: ValuePair{Fork: forkType.Description, Mainline: mainlineType.Description}})
			}
			// Status comparison was incorrect (comparing Description), fixing it
			if forkType.Status.String() != mainlineType.Status.String() {
				typeDiffs = append(typeDiffs, ModuleInfoDifference{FieldName: "Status", Diff: ValuePair{Fork: forkType.Status.String(), Mainline: mainlineType.Status.String()}})
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
					BaseType: getBaseTypeString(forkType.BaseType), // Use helper
					Kind:     "",                                   // Add kind if relevant for types
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
					BaseType: getBaseTypeString(mainlineType.BaseType), // Use helper
					Kind:     "",                                       // Add kind if relevant
				})
			}
		}
	}

	comparedCount = okCount + addedTotal + removedTotal + modifiedTotal

	return added, removed, modified, comparedCount, okCount, addedTotal, removedTotal, modifiedTotal
}

// compareResolvedResults performs the main semantic comparison.
func compareResolvedResults(forkResolved, mainlineResolved map[string]interface{}) (*ComparisonResults, error) {
	results := &ComparisonResults{} // Uses type from types.go

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
		// Handle case where fork failed to resolve module info (e.g., log or set default)
	} else {
		forkModule = fm
	}
	fn, ok := forkResolved["nodes"].([]gosmi.SmiNode)
	if !ok {
		forkNodes = []gosmi.SmiNode{} // Ensure slice is not nil
	} else {
		forkNodes = fn
	}
	ft, ok := forkResolved["types"].([]gosmi.SmiType)
	if !ok {
		forkTypes = []gosmi.SmiType{} // Ensure slice is not nil
	} else {
		forkTypes = ft
	}

	// Mainline Data
	mm, ok := mainlineResolved["moduleInfo"].(mainline_gosmi.SmiModule)
	if !ok {
		// Handle case where mainline failed to resolve module info
	} else {
		mainlineModule = mm
	}
	mn, ok := mainlineResolved["nodes"].([]mainline_gosmi.SmiNode)
	if !ok {
		mainlineNodes = []mainline_gosmi.SmiNode{} // Ensure slice is not nil
	} else {
		mainlineNodes = mn
	}
	mt, ok := mainlineResolved["types"].([]mainline_gosmi.SmiType)
	if !ok {
		mainlineTypes = []mainline_gosmi.SmiType{} // Ensure slice is not nil
	} else {
		mainlineTypes = mt
	}

	// --- Perform Comparisons ---

	// Compare Module Info
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

// hasSemanticDifferences checks the ComparisonResults struct for any reported differences.
func hasSemanticDifferences(results *ComparisonResults) bool {
	if results == nil {
		return false // Or true, depending on how you want to handle comparison errors
	}
	return len(results.ModuleInfoDiffs) > 0 ||
		results.NodesAddedCount > 0 || results.NodesRemovedCount > 0 || results.NodesModifiedCount > 0 ||
		results.TypesAddedCount > 0 || results.TypesRemovedCount > 0 || results.TypesModifiedCount > 0
	// || Add checks for Imports, Revisions, Identity
}

// compareDependencyResults compares fork and mainline dependency parsing results
func compareDependencyResults(forkDeps, mainlineDeps []DependencyParseResult) *DependencyResults {
	result := &DependencyResults{ // Uses type from types.go
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
			// Compare success status and potentially error messages if needed
			if forkDep.Success != mainlineDep.Success {
				result.HasDifferences = true
				break
			}
			// Optionally compare error strings for more detail, but be wary of minor variations
			// if (forkDep.Error != "") != (mainlineDep.Error != "") { // Check if one has error and other doesn't
			// 	result.HasDifferences = true
			// 	break
			// }
		} else {
			// Dependency exists in fork but not in mainline
			result.HasDifferences = true
			break
		}
	}

	// Check for dependencies in mainline but not in fork (only if no diff found yet)
	if !result.HasDifferences {
		for name := range mainlineMap {
			if _, exists := forkMap[name]; !exists {
				result.HasDifferences = true
				break
			}
		}
	}

	return result
}
