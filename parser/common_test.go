package parser_test

import (
	"strings"
	"testing"

	"github.com/lukeod/gosmi/parser"
	"github.com/lukeod/gosmi/parser/testutil"
	"github.com/lukeod/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOidParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string // MIB snippet containing an OID assignment
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name: "Simple OID with names",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { iso org dod }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 1)
				node := mod.Body.Nodes[0]
				require.NotNil(t, node.Oid)
				oid := node.Oid
				require.Len(t, oid.SubIdentifiers, 3)
				assert.Equal(t, types.SmiIdentifier("iso"), *oid.SubIdentifiers[0].Name)
				assert.Nil(t, oid.SubIdentifiers[0].Number)
				assert.Equal(t, types.SmiIdentifier("org"), *oid.SubIdentifiers[1].Name)
				assert.Nil(t, oid.SubIdentifiers[1].Number)
				assert.Equal(t, types.SmiIdentifier("dod"), *oid.SubIdentifiers[2].Name)
				assert.Nil(t, oid.SubIdentifiers[2].Number)
			},
		},
		{
			name: "OID with numbers",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { 1 3 6 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 1)
				node := mod.Body.Nodes[0]
				require.NotNil(t, node.Oid)
				oid := node.Oid
				require.Len(t, oid.SubIdentifiers, 3)
				assert.Nil(t, oid.SubIdentifiers[0].Name)
				require.NotNil(t, oid.SubIdentifiers[0].Number)
				assert.Equal(t, types.SmiSubId(1), *oid.SubIdentifiers[0].Number)
				assert.Nil(t, oid.SubIdentifiers[1].Name)
				require.NotNil(t, oid.SubIdentifiers[1].Number)
				assert.Equal(t, types.SmiSubId(3), *oid.SubIdentifiers[1].Number)
				assert.Nil(t, oid.SubIdentifiers[2].Name)
				require.NotNil(t, oid.SubIdentifiers[2].Number)
				assert.Equal(t, types.SmiSubId(6), *oid.SubIdentifiers[2].Number)
			},
		},
		{
			name: "OID with mixed names and numbers",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { iso 3 6 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 1)
				node := mod.Body.Nodes[0]
				require.NotNil(t, node.Oid)
				oid := node.Oid
				require.Len(t, oid.SubIdentifiers, 4)
				assert.Equal(t, types.SmiIdentifier("iso"), *oid.SubIdentifiers[0].Name)
				assert.Nil(t, oid.SubIdentifiers[0].Number)
				assert.Nil(t, oid.SubIdentifiers[1].Name)
				require.NotNil(t, oid.SubIdentifiers[1].Number)
				assert.Equal(t, types.SmiSubId(3), *oid.SubIdentifiers[1].Number)
				assert.Nil(t, oid.SubIdentifiers[2].Name)
				require.NotNil(t, oid.SubIdentifiers[2].Number)
				assert.Equal(t, types.SmiSubId(6), *oid.SubIdentifiers[2].Number)
				assert.Nil(t, oid.SubIdentifiers[3].Name)
				require.NotNil(t, oid.SubIdentifiers[3].Number)
				assert.Equal(t, types.SmiSubId(1), *oid.SubIdentifiers[3].Number)
			},
		},
		{
			name: "OID with named numbers",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { iso org(3) dod(6) internet(1) }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 1)
				node := mod.Body.Nodes[0]
				require.NotNil(t, node.Oid)
				oid := node.Oid
				require.Len(t, oid.SubIdentifiers, 4)
				assert.Equal(t, types.SmiIdentifier("iso"), *oid.SubIdentifiers[0].Name)
				assert.Nil(t, oid.SubIdentifiers[0].Number)
				assert.Equal(t, types.SmiIdentifier("org"), *oid.SubIdentifiers[1].Name)
				require.NotNil(t, oid.SubIdentifiers[1].Number)
				assert.Equal(t, types.SmiSubId(3), *oid.SubIdentifiers[1].Number)
				assert.Equal(t, types.SmiIdentifier("dod"), *oid.SubIdentifiers[2].Name)
				require.NotNil(t, oid.SubIdentifiers[2].Number)
				assert.Equal(t, types.SmiSubId(6), *oid.SubIdentifiers[2].Number)
				assert.Equal(t, types.SmiIdentifier("internet"), *oid.SubIdentifiers[3].Name)
				require.NotNil(t, oid.SubIdentifiers[3].Number)
				assert.Equal(t, types.SmiSubId(1), *oid.SubIdentifiers[3].Number)
			},
		},
		{
			name: "Empty OID",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { }
					END`,
			wantErr: true, // Parser expects at least one sub-identifier
		},
		{
			name: "OID with trailing comma", // Participle might allow this depending on grammar
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { iso, }
					END`,
			wantErr: true, // Expect error, trailing comma usually invalid
		},
		{
			name: "OID with invalid token",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testOid OBJECT IDENTIFIER ::= { iso $ }
					END`,
			wantErr: true, // '$' is not a valid sub-identifier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// --- Tests for Range, SubType, NamedNumber, SyntaxType, Syntax ---
// These often appear within OBJECT-TYPE definitions, so we'll test them there.

func TestSyntaxParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string // MIB snippet containing an OBJECT-TYPE with syntax to test
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		// --- Basic SyntaxType Tests ---
		{
			name: "Simple Integer32",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				require.NotNil(t, node.ObjectType)
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("Integer32"), syntax.Type.Name)
				assert.Nil(t, syntax.Type.SubType)
				assert.Nil(t, syntax.Type.Enum)
				assert.Nil(t, syntax.Sequence)
			},
		},
		{
			name: "Simple OCTET STRING", // Test mapped keyword
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				require.NotNil(t, node.ObjectType)
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("OCTET STRING"), syntax.Type.Name)
				assert.Nil(t, syntax.Type.SubType)
				assert.Nil(t, syntax.Type.Enum)
				assert.Nil(t, syntax.Sequence)
			},
		},
		{
			name: "Simple OBJECT IDENTIFIER", // Test mapped keyword
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OBJECT IDENTIFIER MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				require.NotNil(t, node.ObjectType)
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("OBJECT IDENTIFIER"), syntax.Type.Name)
				assert.Nil(t, syntax.Type.SubType)
				assert.Nil(t, syntax.Type.Enum)
				assert.Nil(t, syntax.Sequence)
			},
		},
		{
			name: "Custom Type Name",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyCustomType ::= INTEGER
					testObj OBJECT-TYPE SYNTAX MyCustomType MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				require.NotNil(t, node.ObjectType)
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("MyCustomType"), syntax.Type.Name)
				assert.Nil(t, syntax.Type.SubType)
				assert.Nil(t, syntax.Type.Enum)
				assert.Nil(t, syntax.Sequence)
			},
		},

		// --- Range Tests (within Integer SubType) ---
		{
			name: "Integer with single range",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (0..255) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				require.NotNil(t, syntax.Type.SubType)
				subType := syntax.Type.SubType
				require.Len(t, subType.Integer, 1)
				assert.Equal(t, "0", subType.Integer[0].Start)
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present")
				assert.Equal(t, "255", subType.Integer[0].End)
				assert.Empty(t, subType.OctetString)
			},
		},
		{
			name: "Integer with multiple ranges",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (0..10 | 20..30 | 50) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.Integer, 3)
				assert.Equal(t, "0", subType.Integer[0].Start)
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present for range 0..10")
				assert.Equal(t, "10", subType.Integer[0].End)
				assert.Equal(t, "20", subType.Integer[1].Start)
				assert.NotEmpty(t, subType.Integer[1].End, "End value should be present for range 20..30")
				assert.Equal(t, "30", subType.Integer[1].End)
				assert.Equal(t, "50", subType.Integer[2].Start)
				assert.Empty(t, subType.Integer[2].End, "End value should be empty for single value range 50")
				assert.Empty(t, subType.OctetString)
			},
		},
		{
			name: "Integer with negative range",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (-10..10) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.Integer, 1)
				assert.Equal(t, "-10", subType.Integer[0].Start)
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present")
				assert.Equal(t, "10", subType.Integer[0].End)
			},
		},
		{
			name: "Integer range invalid order", // Syntactically valid, semantically invalid
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (10..0) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false, // Parser accepts, semantic check would fail
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.Integer, 1)
				assert.Equal(t, "10", subType.Integer[0].Start)
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present")
				assert.Equal(t, "0", subType.Integer[0].End)
			},
		},
		{
			name: "Integer range invalid syntax (missing ..)",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (0 10) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: true,
		},
		{
			name: "Integer range with MAX keyword",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (0..MAX) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false, // This should fail initially, then pass after the fix
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				require.NotNil(t, syntax.Type.SubType)
				subType := syntax.Type.SubType
				require.Len(t, subType.Integer, 1)
				assert.Equal(t, "0", subType.Integer[0].Start)
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present")
				assert.Equal(t, "MAX", subType.Integer[0].End) // Check for MAX keyword
				assert.Empty(t, subType.OctetString)
			},
		},
		{
			name: "Integer range with MIN keyword",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX Integer32 (MIN..100) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false, // This should fail initially, then pass after the fix
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				require.NotNil(t, syntax.Type.SubType)
				subType := syntax.Type.SubType
				require.Len(t, subType.Integer, 1)
				assert.Equal(t, "MIN", subType.Integer[0].Start) // Check for MIN keyword
				assert.NotEmpty(t, subType.Integer[0].End, "End value should be present")
				assert.Equal(t, "100", subType.Integer[0].End)
				assert.Empty(t, subType.OctetString)
			},
		},

		// --- Range Tests (within OctetString SIZE SubType) ---
		{
			name: "OCTET STRING with SIZE",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING (SIZE (0..255)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("OCTET STRING"), syntax.Type.Name)
				require.NotNil(t, syntax.Type.SubType)
				subType := syntax.Type.SubType
				require.Len(t, subType.OctetString, 1)
				assert.Equal(t, "0", subType.OctetString[0].Start)
				assert.NotEmpty(t, subType.OctetString[0].End, "End value should be present")
				assert.Equal(t, "255", subType.OctetString[0].End)
				assert.Empty(t, subType.Integer)
			},
		},
		{
			name: "OCTET STRING with multiple SIZE ranges",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING (SIZE (4 | 8 | 16..32)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.OctetString, 3)
				assert.Equal(t, "4", subType.OctetString[0].Start)
				assert.Empty(t, subType.OctetString[0].End, "End value should be empty for single value range 4")
				assert.Equal(t, "8", subType.OctetString[1].Start)
				assert.Empty(t, subType.OctetString[1].End, "End value should be empty for single value range 8")
				assert.Equal(t, "16", subType.OctetString[2].Start)
				assert.NotEmpty(t, subType.OctetString[2].End, "End value should be present for range 16..32")
				assert.Equal(t, "32", subType.OctetString[2].End)
				assert.Empty(t, subType.Integer)
			},
		},
		{
			name: "OCTET STRING SIZE with Hex range",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING (SIZE ('0A'H..'FF'H)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.OctetString, 1)
				assert.Equal(t, "'0A'H", subType.OctetString[0].Start)
				assert.NotEmpty(t, subType.OctetString[0].End, "End value should be present")
				assert.Equal(t, "'FF'H", subType.OctetString[0].End)
			},
		},
		{
			name: "OCTET STRING SIZE with Bin range",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING (SIZE ('1010'B..'1111'B)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.OctetString, 1)
				assert.Equal(t, "'1010'B", subType.OctetString[0].Start)
				assert.NotEmpty(t, subType.OctetString[0].End, "End value should be present")
				assert.Equal(t, "'1111'B", subType.OctetString[0].End)
			},
		},
		{
			name: "OCTET STRING with invalid SIZE syntax (missing SIZE keyword)",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING ((0..255)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: true,
		},
		{
			name: "OCTET STRING with invalid SIZE syntax (negative range)", // Syntactically valid, semantically invalid
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX OCTET STRING (SIZE (-10..10)) MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false, // Parser accepts, semantic check would fail
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				subType := node.ObjectType.Syntax.Type.SubType
				require.NotNil(t, subType)
				require.Len(t, subType.OctetString, 1)
				assert.Equal(t, "-10", subType.OctetString[0].Start)
				assert.NotEmpty(t, subType.OctetString[0].End, "End value should be present")
				assert.Equal(t, "10", subType.OctetString[0].End)
			},
		},

		// --- NamedNumber Tests (within Enum/BITS SyntaxType) ---
		{
			name: "Enum with named numbers",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX INTEGER { up(1), down(2), testing(3) } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), syntax.Type.Name)
				require.NotNil(t, syntax.Type.Enum)
				require.Len(t, syntax.Type.Enum, 3)
				assert.Equal(t, types.SmiIdentifier("up"), syntax.Type.Enum[0].Name)
				assert.Equal(t, "1", syntax.Type.Enum[0].Value)
				assert.Equal(t, types.SmiIdentifier("down"), syntax.Type.Enum[1].Name)
				assert.Equal(t, "2", syntax.Type.Enum[1].Value)
				assert.Equal(t, types.SmiIdentifier("testing"), syntax.Type.Enum[2].Name)
				assert.Equal(t, "3", syntax.Type.Enum[2].Value)
				assert.Nil(t, syntax.Type.SubType)
			},
		},
		{
			name: "BITS with named numbers",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX BITS { sunday(0), monday(1), tuesday(2), wednesday(3), thursday(4), friday(5), saturday(6) } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Type)
				assert.Equal(t, types.SmiIdentifier("BITS"), syntax.Type.Name)
				require.NotNil(t, syntax.Type.Enum)
				require.Len(t, syntax.Type.Enum, 7)
				assert.Equal(t, types.SmiIdentifier("sunday"), syntax.Type.Enum[0].Name)
				assert.Equal(t, "0", syntax.Type.Enum[0].Value)
				assert.Equal(t, types.SmiIdentifier("saturday"), syntax.Type.Enum[6].Name)
				assert.Equal(t, "6", syntax.Type.Enum[6].Value)
				assert.Nil(t, syntax.Type.SubType)
			},
		},
		{
			name: "Enum with negative value",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX INTEGER { error(-1), ok(0) } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				enum := node.ObjectType.Syntax.Type.Enum
				require.Len(t, enum, 2)
				assert.Equal(t, types.SmiIdentifier("error"), enum[0].Name)
				assert.Equal(t, "-1", enum[0].Value)
				assert.Equal(t, types.SmiIdentifier("ok"), enum[1].Name)
				assert.Equal(t, "0", enum[1].Value)
			},
		},
		{
			name: "Enum with trailing comma",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX INTEGER { up(1), } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false, // Parser allows trailing comma in list
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				enum := node.ObjectType.Syntax.Type.Enum
				require.Len(t, enum, 1)
				assert.Equal(t, types.SmiIdentifier("up"), enum[0].Name)
				assert.Equal(t, "1", enum[0].Value)
			},
		},
		{
			name: "Enum invalid syntax (missing value)",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX INTEGER { up } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: true,
		},
		{
			name: "Enum invalid syntax (missing parens)",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					testObj OBJECT-TYPE SYNTAX INTEGER { up 1 } MAX-ACCESS read-only STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: true,
		},

		// --- SEQUENCE OF Tests ---
		{
			name: "SEQUENCE OF",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyTableRow ::= SEQUENCE { col1 INTEGER }
					testObj OBJECT-TYPE SYNTAX SEQUENCE OF MyTableRow MAX-ACCESS not-accessible STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				node := testutil.FindNodeByName(t, mod, "testObj")
				syntax := node.ObjectType.Syntax
				require.NotNil(t, syntax.Sequence)
				assert.Equal(t, types.SmiIdentifier("MyTableRow"), *syntax.Sequence)
				assert.Nil(t, syntax.Type)
			},
		},
		{
			name: "SEQUENCE OF invalid syntax (missing OF)",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyTableRow ::= SEQUENCE { col1 INTEGER }
					testObj OBJECT-TYPE SYNTAX SEQUENCE MyTableRow MAX-ACCESS not-accessible STATUS current ::= { test 1 }
					test OBJECT IDENTIFIER ::= { iso 1 }
					END`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need Parse, not mustParseSnippet, because some tests expect errors
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
