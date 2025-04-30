package main

import (
	"time" // Added for timing in DirComparisonResult

	// Added for models used in resolved data
	// Added for mainline models
	"github.com/lukeod/gosmi/types"                         // Added for types used in resolved data
	mainline_types "github.com/sleepinggenius2/gosmi/types" // Added for mainline types
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

// ComparisonResults holds the results of the semantic comparison for a single file.
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

// DirComparisonResult holds the comparison result for a single file within a directory scan.
type DirComparisonResult struct {
	FilePath         string
	Same             bool
	ForkError        error
	MainlineError    error
	ForkDuration     time.Duration
	MainlineDuration time.Duration
}

// --- Helper functions related to types (moved from compare.go for locality) ---

// Helper to safely get string representation of NodeKind
func getNodeKindString(kind interface{}) string {
	switch k := kind.(type) {
	case types.NodeKind:
		return k.String()
	case mainline_types.NodeKind:
		return k.String()
	default:
		// Use fmt.Sprintf which requires importing "fmt"
		// Let's assume fmt is imported in the main package or add it here if needed.
		// For now, returning a simpler string to avoid adding fmt import just for this.
		return "unknown_kind"
		// return fmt.Sprintf("unknown_kind(%T)", kind) // Original requires fmt
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
		// Use fmt.Sprintf which requires importing "fmt"
		// Let's assume fmt is imported in the main package or add it here if needed.
		// For now, returning a simpler string.
		return "unknown_basetype"
		// return fmt.Sprintf("unknown_basetype(%T)", basetype) // Original requires fmt
	}
}
