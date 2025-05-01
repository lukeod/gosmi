package parser_test

import (
	"strings"
	"testing"

	"github.com/lukeod/gosmi/parser"
	"github.com/lukeod/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Existing examples wrapped in valid MIB structures
const ObjectIdentityExample = `
FIZBIN-IDENTITY-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-IDENTITY, OBJECT-IDENTIFIER, experimental FROM SNMPv2-SMI;

fizbinChipSets OBJECT IDENTIFIER ::= { experimental 69 } -- Define parent

fizbin69 OBJECT-IDENTITY
    STATUS  current
    DESCRIPTION
        "The authoritative identity of the Fizbin 69 chipset."
    REFERENCE "Fizbin Datasheet v6.9" -- Optional REFERENCE
    ::= { fizbinChipSets 1 }

END
`

// ObjectDefvalExample is descriptive, not directly parsable as a MIB.
// DEFVAL parsing is tested within ObjectTypeExample.

const ObjectTypeExample = `
EVAL-MIB DEFINITIONS ::= BEGIN

IMPORTS
    OBJECT-TYPE, OBJECT-IDENTIFIER, experimental, Integer32, DisplayString
        FROM SNMPv2-SMI
    RowStatus
        FROM SNMPv2-TC; -- Merged imports

eval OBJECT IDENTIFIER ::= { experimental 42 } -- Define parent

-- Define EvalEntry type used in SEQUENCE OF
EvalEntry ::= SEQUENCE {
    evalIndex       Integer32,
    evalString      DisplayString,
    evalValue       Integer32,
    evalStatus      RowStatus
}

evalSlot OBJECT-TYPE
    SYNTAX      Integer32 (0..2147483647)
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION
            "The index number of the first unassigned entry in the
            evaluation table, or the value of zero indicating that
            all entries are assigned.

            A management station should create new entries in the
            evaluation table using this algorithm:  first, issue a
            management protocol retrieval operation to determine the
            value of evalSlot; and, second, issue a management
            protocol set operation to create an instance of the
            evalStatus object setting its value to createAndGo(4) or
            createAndWait(5).  If this latter operation succeeds,
            then the management station may continue modifying the
            instances corresponding to the newly created conceptual
            row, without fear of collision with other management
            stations."
    ::= { eval 1 }

evalTable OBJECT-TYPE
    SYNTAX      SEQUENCE OF EvalEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION
            "The (conceptual) evaluation table."
    ::= { eval 2 }

evalEntry OBJECT-TYPE
    SYNTAX      EvalEntry -- Use the defined type
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION
            "An entry (conceptual row) in the evaluation table."
    INDEX   { evalIndex }
    ::= { evalTable 1 }

evalIndex OBJECT-TYPE
    SYNTAX      Integer32 (1..2147483647)
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION
            "The auxiliary variable used for identifying instances of
            the columnar objects in the evaluation table."
    ::= { evalEntry 1 }

evalString OBJECT-TYPE
    SYNTAX      DisplayString (SIZE (0..255)) -- Added SIZE constraint for example
    MAX-ACCESS  read-create
    STATUS      current
    DESCRIPTION
            "The string to evaluate."
    ::= { evalEntry 2 }

evalValue OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION
            "The value when evalString was last evaluated, or zero if
             no such value is available."
    DEFVAL  { 0 }
    ::= { evalEntry 3 }

evalStatus OBJECT-TYPE
    SYNTAX      RowStatus -- Use imported type
    MAX-ACCESS  read-create
    STATUS      current
    DESCRIPTION
            "The status column used for creating, modifying, and
            deleting instances of the columnar objects in the
            evaluation table."
    DEFVAL  { active } -- Use named number from RowStatus TC
    ::= { evalEntry 4 }

END
`

const ObjectGroupExample = `
OBJGROUP-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-GROUP, OBJECT-TYPE, OBJECT-IDENTIFIER, experimental, Integer32 FROM SNMPv2-SMI;

objGroupRoot OBJECT IDENTIFIER ::= { experimental 77 }

obj1 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Obj 1" ::= { objGroupRoot 1 }
obj2 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Obj 2" ::= { objGroupRoot 2 }

testObjectGroup OBJECT-GROUP
    OBJECTS       { obj1, obj2 }
    STATUS        current
    DESCRIPTION   "A group of objects."
    REFERENCE     "RFC ABC"
    ::= { objGroupRoot 3 }

END
`

func TestObjectParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Parse ObjectIdentityExample",
			input:   ObjectIdentityExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 2, "Expected 2 nodes (parent OID + identity)")
				var identNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].Name == "fizbin69" {
						identNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, identNode, "Object identity node 'fizbin69' not found")
				require.NotNil(t, identNode.ObjectIdentity, "Node is not an OBJECT-IDENTITY")

				oi := identNode.ObjectIdentity
				assert.Equal(t, parser.StatusCurrent, oi.Status, "Status mismatch")
				assert.Contains(t, oi.Description, "Fizbin 69 chipset", "Description mismatch")
				assert.Equal(t, "Fizbin Datasheet v6.9", oi.Reference, "Reference mismatch")

				require.NotNil(t, identNode.Oid, "Identity OID is nil")
				oid := identNode.Oid
				require.Len(t, oid.SubIdentifiers, 2, "Identity OID length mismatch")
				assert.Equal(t, types.SmiIdentifier("fizbinChipSets"), *oid.SubIdentifiers[0].Name, "Identity OID parent name mismatch")
				require.NotNil(t, oid.SubIdentifiers[1].Number, "Identity OID sub-identifier number is nil")
				assert.Equal(t, types.SmiSubId(1), *oid.SubIdentifiers[1].Number, "Identity OID sub-identifier number mismatch")
			},
		},
		{
			name:    "Parse ObjectTypeExample",
			input:   ObjectTypeExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				// Find nodes by name for easier checking
				nodes := make(map[types.SmiIdentifier]*parser.Node)
				for i := range mod.Body.Nodes {
					nodes[mod.Body.Nodes[i].Name] = &mod.Body.Nodes[i]
				}
				require.Len(t, mod.Body.Types, 1, "Expected 1 type definition (EvalEntry)")
				assert.Equal(t, types.SmiIdentifier("EvalEntry"), mod.Body.Types[0].Name)

				// Check evalSlot (Scalar)
				slotNode, ok := nodes["evalSlot"]
				require.True(t, ok, "evalSlot node not found")
				require.NotNil(t, slotNode.ObjectType, "evalSlot is not an OBJECT-TYPE")
				slotOT := slotNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("Integer32"), slotOT.Syntax.Type.Name, "evalSlot SYNTAX name mismatch")
				require.NotNil(t, slotOT.Syntax.Type.SubType, "evalSlot SYNTAX SubType is nil")
				require.Len(t, slotOT.Syntax.Type.SubType.Integer, 1, "evalSlot SYNTAX range count mismatch")
				assert.Equal(t, "0", slotOT.Syntax.Type.SubType.Integer[0].Start, "evalSlot SYNTAX range start mismatch")
				assert.Equal(t, "2147483647", slotOT.Syntax.Type.SubType.Integer[0].End, "evalSlot SYNTAX range end mismatch")
				assert.Equal(t, parser.AccessReadOnly, slotOT.Access, "evalSlot MAX-ACCESS mismatch")
				assert.Equal(t, parser.StatusCurrent, slotOT.Status, "evalSlot STATUS mismatch")
				assert.Contains(t, slotOT.Description, "index number of the first unassigned entry", "evalSlot DESCRIPTION mismatch")
				assert.Nil(t, slotOT.Defval, "evalSlot DEFVAL should be nil") // Corrected case: Defval

				// Check evalTable (Table)
				tableNode, ok := nodes["evalTable"]
				require.True(t, ok, "evalTable node not found")
				require.NotNil(t, tableNode.ObjectType, "evalTable is not an OBJECT-TYPE")
				tableOT := tableNode.ObjectType
				require.NotNil(t, tableOT.Syntax.Sequence, "evalTable SYNTAX should be SEQUENCE OF")
				assert.Equal(t, types.SmiIdentifier("EvalEntry"), *tableOT.Syntax.Sequence, "evalTable SYNTAX sequence type mismatch")
				assert.Equal(t, parser.AccessNotAccessible, tableOT.Access, "evalTable MAX-ACCESS mismatch")
				assert.Equal(t, parser.StatusCurrent, tableOT.Status, "evalTable STATUS mismatch")

				// Check evalEntry (Row)
				entryNode, ok := nodes["evalEntry"]
				require.True(t, ok, "evalEntry node not found")
				require.NotNil(t, entryNode.ObjectType, "evalEntry is not an OBJECT-TYPE")
				entryOT := entryNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("EvalEntry"), entryOT.Syntax.Type.Name, "evalEntry SYNTAX mismatch")
				assert.Equal(t, parser.AccessNotAccessible, entryOT.Access, "evalEntry MAX-ACCESS mismatch")
				assert.Equal(t, parser.StatusCurrent, entryOT.Status, "evalEntry STATUS mismatch")
				require.Len(t, entryOT.Index, 1, "evalEntry INDEX count mismatch")
				assert.Equal(t, types.SmiIdentifier("evalIndex"), entryOT.Index[0].Name, "evalEntry INDEX name mismatch")
				assert.False(t, entryOT.Index[0].Implied, "evalEntry INDEX should not be implied")
				assert.Nil(t, entryOT.Augments, "evalEntry AUGMENTS should be nil")

				// Check evalIndex (Column, Index)
				indexNode, ok := nodes["evalIndex"]
				require.True(t, ok, "evalIndex node not found")
				require.NotNil(t, indexNode.ObjectType, "evalIndex is not an OBJECT-TYPE")
				indexOT := indexNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("Integer32"), indexOT.Syntax.Type.Name, "evalIndex SYNTAX name mismatch")
				assert.Equal(t, parser.AccessNotAccessible, indexOT.Access, "evalIndex MAX-ACCESS mismatch")

				// Check evalString (Column)
				stringNode, ok := nodes["evalString"]
				require.True(t, ok, "evalString node not found")
				require.NotNil(t, stringNode.ObjectType, "evalString is not an OBJECT-TYPE")
				stringOT := stringNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("DisplayString"), stringOT.Syntax.Type.Name, "evalString SYNTAX name mismatch")
				require.NotNil(t, stringOT.Syntax.Type.SubType, "evalString SYNTAX SubType is nil")
				require.Len(t, stringOT.Syntax.Type.SubType.OctetString, 1, "evalString SYNTAX SIZE range count mismatch")
				assert.Equal(t, "0", stringOT.Syntax.Type.SubType.OctetString[0].Start, "evalString SYNTAX SIZE start mismatch")
				assert.Equal(t, "255", stringOT.Syntax.Type.SubType.OctetString[0].End, "evalString SYNTAX SIZE end mismatch")
				assert.Equal(t, parser.AccessReadCreate, stringOT.Access, "evalString MAX-ACCESS mismatch")

				// Check evalValue (Column with DEFVAL)
				valueNode, ok := nodes["evalValue"]
				require.True(t, ok, "evalValue node not found")
				require.NotNil(t, valueNode.ObjectType, "evalValue is not an OBJECT-TYPE")
				valueOT := valueNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("Integer32"), valueOT.Syntax.Type.Name, "evalValue SYNTAX name mismatch")
				assert.Equal(t, parser.AccessReadOnly, valueOT.Access, "evalValue MAX-ACCESS mismatch")
				require.NotNil(t, valueOT.Defval, "evalValue DEFVAL is nil")
				assert.Equal(t, "0", *valueOT.Defval, "evalValue DEFVAL value mismatch")

				// Check evalStatus (Column with DEFVAL named number)
				statusNode, ok := nodes["evalStatus"]
				require.True(t, ok, "evalStatus node not found")
				require.NotNil(t, statusNode.ObjectType, "evalStatus is not an OBJECT-TYPE")
				statusOT := statusNode.ObjectType
				assert.Equal(t, types.SmiIdentifier("RowStatus"), statusOT.Syntax.Type.Name, "evalStatus SYNTAX name mismatch")
				assert.Equal(t, parser.AccessReadCreate, statusOT.Access, "evalStatus MAX-ACCESS mismatch")
				require.NotNil(t, statusOT.Defval, "evalStatus DEFVAL is nil")                  // Corrected case: Defval
				assert.Equal(t, "active", *statusOT.Defval, "evalStatus DEFVAL value mismatch") // Compare string value
			},
		},
		{
			name:    "Parse ObjectGroupExample",
			input:   ObjectGroupExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 4, "Expected 4 nodes (root OID, 2 objects, 1 group)")
				var groupNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].Name == "testObjectGroup" {
						groupNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, groupNode, "Object group node 'testObjectGroup' not found")
				require.NotNil(t, groupNode.ObjectGroup, "Node is not an OBJECT-GROUP")

				og := groupNode.ObjectGroup
				assert.ElementsMatch(t, []types.SmiIdentifier{"obj1", "obj2"}, og.Objects, "Objects mismatch")
				assert.Equal(t, parser.StatusCurrent, og.Status, "Status mismatch")
				assert.Equal(t, "A group of objects.", og.Description, "Description mismatch")
				assert.Equal(t, "RFC ABC", og.Reference, "Reference mismatch")

				require.NotNil(t, groupNode.Oid, "Group OID is nil")
				oid := groupNode.Oid
				require.Len(t, oid.SubIdentifiers, 2, "Group OID length mismatch")
				assert.Equal(t, types.SmiIdentifier("objGroupRoot"), *oid.SubIdentifiers[0].Name, "Group OID parent name mismatch")
				require.NotNil(t, oid.SubIdentifiers[1].Number, "Group OID sub-identifier number is nil")
				assert.Equal(t, types.SmiSubId(3), *oid.SubIdentifiers[1].Number, "Group OID sub-identifier number mismatch")
			},
		},
		{
			name: "Error - ObjectType Missing SYNTAX",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, experimental FROM SNMPv2-SMI;
errorObject OBJECT-TYPE
    -- SYNTAX missing
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "Missing syntax."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - ObjectType Missing MAX-ACCESS",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, experimental, Integer32 FROM SNMPv2-SMI;
errorObject OBJECT-TYPE
    SYNTAX Integer32
    -- MAX-ACCESS missing
    STATUS current
    DESCRIPTION "Missing access."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - ObjectType Missing STATUS",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, experimental, Integer32 FROM SNMPv2-SMI;
errorObject OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    -- STATUS missing
    DESCRIPTION "Missing status."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - ObjectIdentity Missing STATUS",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-IDENTITY, experimental FROM SNMPv2-SMI;
errorIdentity OBJECT-IDENTITY
    -- STATUS missing
    DESCRIPTION "Missing status."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - ObjectGroup Missing OBJECTS",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-GROUP, experimental FROM SNMPv2-SMI;
errorGroup OBJECT-GROUP
    -- OBJECTS missing
    STATUS current
    DESCRIPTION "Missing objects."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need to parse the type definition for EvalEntry before parsing the main MIB
			// This simulates how imports would work in a real scenario.
			// We parse the type definition separately first (though ideally, the parser handles this resolution).
			// For this test setup, we include it directly in the ObjectTypeExample MIB string.
			mod, err := parser.Parse(tt.name+".mib", strings.NewReader(tt.input))

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

// TestObjectGroupTrailingComma verifies that parsing an OBJECT-GROUP
// with a trailing comma in the OBJECTS list does not panic.
// This test is expected to PANIC before the grammar fix.
func TestObjectGroupTrailingComma(t *testing.T) {
	input := `
OBJGROUP-COMMA-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-GROUP, OBJECT-TYPE, OBJECT-IDENTIFIER, experimental, Integer32 FROM SNMPv2-SMI;

objGroupRoot OBJECT IDENTIFIER ::= { experimental 78 }

obj1 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Obj 1" ::= { objGroupRoot 1 }
obj2 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Obj 2" ::= { objGroupRoot 2 }

testObjectGroupComma OBJECT-GROUP
    OBJECTS       { obj1, obj2 } -- Trailing comma removed
    STATUS        current
    DESCRIPTION   "A group with a trailing comma."
    ::= { objGroupRoot 3 }

END
`
	// Before the fix, this call is expected to panic.
	// After the fix, it should parse without error.
	mod, err := parser.Parse("OBJGROUP-COMMA-MIB.mib", strings.NewReader(input))

	// After the fix, we expect no error and a valid module.
	require.NoError(t, err, "Parsing failed unexpectedly")
	require.NotNil(t, mod, "Parsed module should not be nil")

	// Find the group and verify its contents after the fix
	var groupNode *parser.Node
	for i := range mod.Body.Nodes {
		if mod.Body.Nodes[i].Name == "testObjectGroupComma" {
			groupNode = &mod.Body.Nodes[i] // Take address of slice element directly
			break
		}
	}
	require.NotNil(t, groupNode, "Could not find node 'testObjectGroupComma'")
	require.NotNil(t, groupNode.ObjectGroup, "Node 'testObjectGroupComma' is not an OBJECT-GROUP")
	assert.Len(t, groupNode.ObjectGroup.Objects, 2, "Expected 2 objects in the group")
	assert.Equal(t, types.SmiIdentifier("obj1"), groupNode.ObjectGroup.Objects[0])
	assert.Equal(t, types.SmiIdentifier("obj2"), groupNode.ObjectGroup.Objects[1])
}
