package testutil

import (
	"testing"

	"github.com/lukeod/gosmi/parser"
	"github.com/lukeod/gosmi/types"
	"github.com/stretchr/testify/require"
)

// FindNodeByName searches the parsed module's top-level nodes (Nodes and Types)
// for a node with the given name. It uses require.FailNowf if the node is not found.
// Note: This primarily checks mod.Body.Nodes as most test targets (OBJECT-TYPE etc.) reside there.
// It includes a basic check for mod.Body.Types but fails if the match is found there,
// as helper usage in current tests expects the target within mod.Body.Nodes.
func FindNodeByName(t *testing.T, mod *parser.Module, name types.SmiIdentifier) *parser.Node {
	t.Helper()
	if mod == nil { // Only check if the module pointer itself is nil
		require.FailNowf(t, "Module is nil", "Cannot search for node %q in a nil module", name)
		return nil // Should not be reached
	}

	// Primarily search within Body.Nodes
	for i := range mod.Body.Nodes {
		// Check if the node has a Name field before accessing it
		// This covers OBJECT-TYPE, OBJECT-IDENTITY, MODULE-COMPLIANCE, etc.
		if mod.Body.Nodes[i].Name == name {
			return &mod.Body.Nodes[i] // Return address of the node
		}
	}

	// Check Body.Types as a fallback, but fail if found here for current test expectations
	for i := range mod.Body.Types {
		if mod.Body.Types[i].Name == name {
			// This case is less likely for the current tests, but included for completeness
			// We can't return a *parser.Node here directly.
			// For syntax tests, we expect the target to be within a Node.
			require.FailNowf(t, "Target found as Type, not Node", "Identifier %q found as Type, expected within Node for current tests", name)
		}
	}

	require.FailNowf(t, "Node not found", "Node with name %q not found in module", name)
	return nil // Should not be reached
}

/*
Standard Table-Driven Test Structure Recommendation:

For testing parser components, a table-driven approach is recommended for clarity and maintainability.
Each test case should be defined in a struct, typically including:
- name: A descriptive name for the test case.
- input: The MIB snippet or input data for the test.
- wantErr: A boolean indicating if an error is expected.
- check: An optional function `func(t *testing.T, result *parser.Module)` or similar
         to perform specific assertions on the parsed result if no error is expected.
         This function can use helpers like assertNodeType or testify assertions.

Example Structure:

func TestSomeParserFeature(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module) // Adjust result type as needed
	}{
		{
			name:  "Simple valid case",
			input: `MODULE-IDENTITY ... ::= { ... }`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.NotNil(t, mod.Identity)
				assert.Equal(t, "EXPECTED_NAME", mod.Identity.Name)
				// ... more assertions ...
			},
		},
		{
			name:  "Case with expected parsing error",
			input: `MODULE-IDENTITY ::=`, // Invalid syntax
			wantErr: true,
		},
		// ... more test cases ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mustParseSnippet if the test should fail immediately on parse error
			// and only proceeds if parsing is successful.
			// mod := mustParseSnippet(t, tt.input)
			// if tt.check != nil {
			// 	tt.check(t, mod)
			// }

			// Or, use parser.Parse directly if you need to check for expected errors.
			mod, err := parser.Parse(tt.name, strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
				require.NotNil(t, mod, "Expected a non-nil module on success")
				if tt.check != nil {
					tt.check(t, mod)
				}
			}
		})
	}
}

*/
