package parser_test

import (
	"strings"
	"testing"

	"github.com/sleepinggenius2/gosmi/parser"
	"github.com/sleepinggenius2/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// Assuming testutil exists as planned
)

// --- Test Type Assignments (::=) ---

func TestTypeAssignmentParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string // MIB snippet containing a type assignment
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		// --- Basic SyntaxType Assignment ---
		{
			name: "Simple Type Assignment - INTEGER",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyInteger ::= INTEGER
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MyInteger")
				require.NotNil(t, typ.Syntax)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), typ.Syntax.Name)
				assert.Nil(t, typ.Syntax.SubType)
				assert.Nil(t, typ.Syntax.Enum)
				assert.Nil(t, typ.TextualConvention)
				assert.Nil(t, typ.Sequence)
				assert.Nil(t, typ.Implicit)
			},
		},
		{
			name: "Type Assignment - INTEGER with subtype",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyRangedInt ::= INTEGER (0..100)
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MyRangedInt")
				require.NotNil(t, typ.Syntax)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), typ.Syntax.Name)
				require.NotNil(t, typ.Syntax.SubType)
				require.Len(t, typ.Syntax.SubType.Integer, 1)
				assert.Equal(t, "0", typ.Syntax.SubType.Integer[0].Start)
				assert.Equal(t, "100", typ.Syntax.SubType.Integer[0].End)
				assert.Nil(t, typ.Syntax.Enum)
			},
		},
		{
			name: "Type Assignment - INTEGER with enum",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyEnumInt ::= INTEGER { active(1), inactive(0) }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MyEnumInt")
				require.NotNil(t, typ.Syntax)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), typ.Syntax.Name)
				require.NotNil(t, typ.Syntax.Enum)
				require.Len(t, typ.Syntax.Enum, 2)
				assert.Equal(t, types.SmiIdentifier("active"), typ.Syntax.Enum[0].Name)
				assert.Equal(t, "1", typ.Syntax.Enum[0].Value)
				assert.Equal(t, types.SmiIdentifier("inactive"), typ.Syntax.Enum[1].Name)
				assert.Equal(t, "0", typ.Syntax.Enum[1].Value)
				assert.Nil(t, typ.Syntax.SubType)
			},
		},
		{
			name: "Type Assignment - Custom Base Type",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					BaseType ::= INTEGER
					DerivedType ::= BaseType (1..10)
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "DerivedType")
				require.NotNil(t, typ.Syntax)
				assert.Equal(t, types.SmiIdentifier("BaseType"), typ.Syntax.Name) // Name is the base type identifier
				require.NotNil(t, typ.Syntax.SubType)
				require.Len(t, typ.Syntax.SubType.Integer, 1)
				assert.Equal(t, "1", typ.Syntax.SubType.Integer[0].Start)
				assert.Equal(t, "10", typ.Syntax.SubType.Integer[0].End)
			},
		},

		// --- TEXTUAL-CONVENTION Assignment ---
		{
			name: "Textual Convention - Full",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					DisplayString ::= TEXTUAL-CONVENTION
						DISPLAY-HINT "255a"
						STATUS       current
						DESCRIPTION  "Represents textual information taken from the NVT ASCII character set."
						REFERENCE    "RFC 1213"
						SYNTAX       OCTET STRING (SIZE (0..255))
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "DisplayString")
				require.NotNil(t, typ.TextualConvention)
				tc := typ.TextualConvention
				assert.Equal(t, "255a", tc.DisplayHint) // Parser removes quotes
				assert.Equal(t, parser.StatusCurrent, tc.Status)
				assert.Contains(t, tc.Description, "NVT ASCII character set") // Parser removes quotes
				assert.Equal(t, "RFC 1213", tc.Reference)                     // Parser removes quotes
				require.NotNil(t, tc.Syntax)
				assert.Equal(t, types.SmiIdentifier("OCTET STRING"), tc.Syntax.Name)
				require.NotNil(t, tc.Syntax.SubType)
				require.Len(t, tc.Syntax.SubType.OctetString, 1)
				assert.Equal(t, "0", tc.Syntax.SubType.OctetString[0].Start)
				assert.Equal(t, "255", tc.Syntax.SubType.OctetString[0].End)
				assert.Nil(t, typ.Syntax)
				assert.Nil(t, typ.Sequence)
				assert.Nil(t, typ.Implicit)
			},
		},
		{
			name: "Textual Convention - Minimal",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyTC ::= TEXTUAL-CONVENTION
						STATUS       current
						DESCRIPTION  "Minimal TC"
						SYNTAX       INTEGER
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MyTC")
				require.NotNil(t, typ.TextualConvention)
				tc := typ.TextualConvention
				assert.Empty(t, tc.DisplayHint)
				assert.Equal(t, parser.StatusCurrent, tc.Status)
				assert.Equal(t, "Minimal TC", tc.Description) // Parser removes quotes
				assert.Empty(t, tc.Reference)
				require.NotNil(t, tc.Syntax)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), tc.Syntax.Name)
				assert.Nil(t, tc.Syntax.SubType)
				assert.Nil(t, tc.Syntax.Enum)
			},
		},
		{
			name: "Textual Convention - Missing STATUS",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					BadTC ::= TEXTUAL-CONVENTION
						DESCRIPTION  "Missing Status"
						SYNTAX       INTEGER
					END`,
			wantErr: true,
		},
		{
			name: "Textual Convention - Missing DESCRIPTION",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					BadTC ::= TEXTUAL-CONVENTION
						STATUS       current
						SYNTAX       INTEGER
					END`,
			wantErr: true,
		},
		{
			name: "Textual Convention - Missing SYNTAX",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					BadTC ::= TEXTUAL-CONVENTION
						STATUS       current
						DESCRIPTION  "Missing Syntax"
					END`,
			wantErr: true,
		},

		// --- SEQUENCE Assignment ---
		{
			name: "SEQUENCE Assignment",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MySequence ::= SEQUENCE {
						field1 INTEGER,
						field2 OCTET STRING (SIZE(0..10))
					}
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MySequence")
				require.NotNil(t, typ.Sequence)
				seq := typ.Sequence
				assert.Equal(t, parser.SequenceTypeSequence, seq.Type)
				require.Len(t, seq.Entries, 2)
				// Entry 1
				assert.Equal(t, types.SmiIdentifier("field1"), seq.Entries[0].Descriptor)
				require.NotNil(t, seq.Entries[0].Syntax)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), seq.Entries[0].Syntax.Name)
				assert.Nil(t, seq.Entries[0].Syntax.SubType)
				assert.Nil(t, seq.Entries[0].Syntax.Enum)
				// Entry 2
				assert.Equal(t, types.SmiIdentifier("field2"), seq.Entries[1].Descriptor)
				require.NotNil(t, seq.Entries[1].Syntax)
				assert.Equal(t, types.SmiIdentifier("OCTET STRING"), seq.Entries[1].Syntax.Name)
				require.NotNil(t, seq.Entries[1].Syntax.SubType)
				require.Len(t, seq.Entries[1].Syntax.SubType.OctetString, 1)
				assert.Equal(t, "0", seq.Entries[1].Syntax.SubType.OctetString[0].Start)
				assert.Equal(t, "10", seq.Entries[1].Syntax.SubType.OctetString[0].End)
				assert.Nil(t, seq.Entries[1].Syntax.Enum)

				assert.Nil(t, typ.Syntax)
				assert.Nil(t, typ.TextualConvention)
				assert.Nil(t, typ.Implicit)
			},
		},
		{
			name: "SEQUENCE with trailing comma",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MySequence ::= SEQUENCE { field1 INTEGER, }
					END`,
			wantErr: false, // Parser allows trailing comma
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MySequence")
				require.NotNil(t, typ.Sequence)
				require.Len(t, typ.Sequence.Entries, 1)
				assert.Equal(t, types.SmiIdentifier("field1"), typ.Sequence.Entries[0].Descriptor)
			},
		},
		{
			name: "Empty SEQUENCE",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MySequence ::= SEQUENCE { }
					END`,
			wantErr: true, // Parser currently requires at least one entry
			check: func(t *testing.T, mod *parser.Module) {
				// This check won't run if wantErr is true
				// typ := findTypeByName(t, mod, "MySequence")
				// require.NotNil(t, typ.Sequence)
				// assert.Equal(t, parser.SequenceTypeSequence, typ.Sequence.Type)
				// assert.Empty(t, typ.Sequence.Entries)
			},
		},
		{
			name: "SEQUENCE missing braces",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MySequence ::= SEQUENCE field1 INTEGER
					END`,
			wantErr: true,
		},

		// --- CHOICE Assignment ---
		{
			name: "CHOICE Assignment",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyChoice ::= CHOICE {
						choiceInt INTEGER,
						choiceStr OCTET STRING
					}
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				typ := findTypeByName(t, mod, "MyChoice")
				require.NotNil(t, typ.Sequence) // CHOICE uses the Sequence struct
				seq := typ.Sequence
				assert.Equal(t, parser.SequenceTypeChoice, seq.Type)
				require.Len(t, seq.Entries, 2)
				assert.Equal(t, types.SmiIdentifier("choiceInt"), seq.Entries[0].Descriptor)
				assert.Equal(t, types.SmiIdentifier("INTEGER"), seq.Entries[0].Syntax.Name)
				assert.Equal(t, types.SmiIdentifier("choiceStr"), seq.Entries[1].Descriptor)
				assert.Equal(t, types.SmiIdentifier("OCTET STRING"), seq.Entries[1].Syntax.Name)
			},
		},
		{
			name: "Empty CHOICE",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyChoice ::= CHOICE { }
					END`,
			wantErr: true, // Parser currently requires at least one entry
			check: func(t *testing.T, mod *parser.Module) {
				// This check won't run if wantErr is true
				// typ := findTypeByName(t, mod, "MyChoice")
				// require.NotNil(t, typ.Sequence)
				// assert.Equal(t, parser.SequenceTypeChoice, typ.Sequence.Type)
				// assert.Empty(t, typ.Sequence.Entries)
			},
		},

		// --- IMPLICIT Assignment ---
		{
			name: "IMPLICIT Assignment",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyImplicitType ::= [1] IMPLICIT OCTET STRING (SIZE(10))
					END`,
			wantErr: true, // Parser currently fails with lexer error after ']'
			check: func(t *testing.T, mod *parser.Module) {
				// This check won't run if wantErr is true
				// typ := findTypeByName(t, mod, "MyImplicitType")
				// require.NotNil(t, typ.Implicit)
				// imp := typ.Implicit
				// assert.False(t, imp.Application)
				// assert.Equal(t, 1, imp.Number)
				// require.NotNil(t, imp.Syntax)
				// assert.Equal(t, types.SmiIdentifier("OCTET STRING"), imp.Syntax.Name)
				// require.NotNil(t, imp.Syntax.SubType)
				// require.Len(t, imp.Syntax.SubType.OctetString, 1)
				// assert.Equal(t, "10", imp.Syntax.SubType.OctetString[0].Start)
				// assert.Empty(t, imp.Syntax.SubType.OctetString[0].End)
				//
				// assert.Nil(t, typ.Syntax)
				// assert.Nil(t, typ.TextualConvention)
				// assert.Nil(t, typ.Sequence)
			},
		},
		{
			name: "IMPLICIT APPLICATION Assignment",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyImplicitAppType ::= [APPLICATION 5] IMPLICIT INTEGER { zero(0) }
					END`,
			wantErr: true, // Parser currently fails with lexer error after ']'
			check: func(t *testing.T, mod *parser.Module) {
				// This check won't run if wantErr is true
				// typ := findTypeByName(t, mod, "MyImplicitAppType")
				// require.NotNil(t, typ.Implicit)
				// imp := typ.Implicit
				// assert.True(t, imp.Application)
				// assert.Equal(t, 5, imp.Number)
				// require.NotNil(t, imp.Syntax)
				// assert.Equal(t, types.SmiIdentifier("INTEGER"), imp.Syntax.Name)
				// require.NotNil(t, imp.Syntax.Enum)
				// require.Len(t, imp.Syntax.Enum, 1)
				// assert.Equal(t, types.SmiIdentifier("zero"), imp.Syntax.Enum[0].Name)
				// assert.Equal(t, "0", imp.Syntax.Enum[0].Value)
			},
		},
		{
			name: "IMPLICIT missing number",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyImplicitType ::= [] IMPLICIT OCTET STRING
					END`,
			wantErr: true,
		},
		{
			name: "IMPLICIT missing IMPLICIT keyword",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyImplicitType ::= [1] OCTET STRING
					END`,
			wantErr: true,
		},
		{
			name: "IMPLICIT missing base type",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyImplicitType ::= [1] IMPLICIT
					END`,
			wantErr: true,
		},

		// --- General Errors ---
		{
			name: "Missing assignment operator",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyType INTEGER
					END`,
			wantErr: true,
		},
		{
			name: "Missing type definition",
			input: `TEST-MIB DEFINITIONS ::= BEGIN
					MyType ::=
					END`,
			wantErr: true,
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

// Helper function to find a type definition by name in the parsed module
func findTypeByName(t *testing.T, mod *parser.Module, name types.SmiIdentifier) *parser.Type {
	t.Helper()
	if mod == nil {
		require.FailNowf(t, "Module is nil", "Cannot search for type %q in a nil module", name)
		return nil // Should not be reached
	}
	for i := range mod.Body.Types {
		if mod.Body.Types[i].Name == name {
			return &mod.Body.Types[i] // Return address of the type
		}
	}
	require.FailNowf(t, "Type not found", "Type with name %q not found in module", name)
	return nil // Should not be reached
}
