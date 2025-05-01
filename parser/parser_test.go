package parser_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukeod/gosmi/parser"
	"github.com/lukeod/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string // Input for parser.Parse
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name: "Module Identity Strings",
			input: `
TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS MODULE-IDENTITY, experimental FROM SNMPv2-SMI;

testIdentity MODULE-IDENTITY
    LAST-UPDATED "202404300000Z"
    ORGANIZATION "Test Org"
    CONTACT-INFO
            "Contact Name
             Email: test@example.com
             Phone: +1234567890"
    DESCRIPTION
            "This is a multi-line
             description for testing.
             It includes special chars like ' and \"."
    REVISION      "202404300000Z"
    DESCRIPTION
            "Initial revision."
    ::= { experimental 999 }

END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.NotNil(t, mod.Body.Identity, "Parsed module identity is nil")
				identity := mod.Body.Identity
				assert.Equal(t, parser.Date("202404300000Z"), identity.LastUpdated, "LastUpdated mismatch")
				assert.Equal(t, "Test Org", identity.Organization, "Organization mismatch")
				assert.Contains(t, identity.ContactInfo, "Contact Name", "ContactInfo missing name")
				assert.Contains(t, identity.ContactInfo, "Email: test@example.com", "ContactInfo missing email")
				assert.Contains(t, identity.ContactInfo, "Phone: +1234567890", "ContactInfo missing phone")
				assert.Contains(t, identity.Description, "This is a multi-line", "Description missing first part")
				assert.Contains(t, identity.Description, "description for testing.", "Description missing second part")
				assert.Contains(t, identity.Description, "special chars like ' and \".", "Description missing third part")
				require.Len(t, identity.Revisions, 1, "Expected 1 revision")
				assert.Equal(t, "Initial revision.", identity.Revisions[0].Description, "Revision description mismatch")
			},
		},
		{
			name: "Object Type Strings",
			input: `
TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS OBJECT-TYPE, experimental FROM SNMPv2-SMI;

testObject OBJECT-TYPE
    SYNTAX      INTEGER
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION
            "This object has a description.
             It spans multiple lines.
             Includes 'single' and \"double\" quotes."
    REFERENCE   "RFC XXXX Section Y"
    ::= { experimental 998 }

END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 1, "Expected 1 node")
				node := mod.Body.Nodes[0]
				require.NotNil(t, node.ObjectType, "Parsed object type is nil")
				objectType := node.ObjectType
				assert.Contains(t, objectType.Description, "This object has a description.", "ObjectType Description missing first part")
				assert.Contains(t, objectType.Description, "spans multiple lines.", "ObjectType Description missing second part")
				assert.Contains(t, objectType.Description, "Includes 'single' and \"double\" quotes.", "ObjectType Description missing third part")
				assert.Equal(t, "RFC XXXX Section Y", objectType.Reference, "Reference mismatch")
			},
		},
		{
			name: "Minimal Valid Module",
			input: `
MINIMAL-MIB DEFINITIONS ::= BEGIN
    minimalIdentity MODULE-IDENTITY
        LAST-UPDATED "202401010000Z"
        ORGANIZATION "Minimal Org"
        CONTACT-INFO "min@example.com"
        DESCRIPTION  "Minimal desc."
    ::= { enterprises 99999 }
END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, "MINIMAL-MIB", string(mod.Name))
				require.NotNil(t, mod.Body.Identity)
				assert.Equal(t, "minimalIdentity", string(mod.Body.Identity.Name))
			},
		},
		{
			name: "Module with Imports",
			input: `
IMPORTS-MIB DEFINITIONS ::= BEGIN
    IMPORTS
        MODULE-IDENTITY, OBJECT-TYPE, Counter32, enterprises
            FROM SNMPv2-SMI
        DisplayString
            FROM SNMPv2-TC;

    importsIdentity MODULE-IDENTITY
        LAST-UPDATED "202401010000Z"
        ORGANIZATION "Imports Org"
        CONTACT-INFO "imp@example.com"
        DESCRIPTION  "Imports desc."
    ::= { enterprises 99998 }

    testCounter OBJECT-TYPE
        SYNTAX Counter32
        MAX-ACCESS read-only
        STATUS current
        DESCRIPTION "A counter."
    ::= { importsIdentity 1 }

END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, "IMPORTS-MIB", string(mod.Name))
				// Check Imports (expecting two separate Import structs)
				require.Len(t, mod.Body.Imports, 2, "Expected 2 import statements (one per FROM clause)")

				// Find the imports by module name as order isn't guaranteed
				var smiImport, tcImport *parser.Import
				for i := range mod.Body.Imports {
					if mod.Body.Imports[i].Module == "SNMPv2-SMI" {
						smiImport = &mod.Body.Imports[i]
					} else if mod.Body.Imports[i].Module == "SNMPv2-TC" {
						tcImport = &mod.Body.Imports[i]
					}
				}
				require.NotNil(t, smiImport, "SNMPv2-SMI import not found")
				require.NotNil(t, tcImport, "SNMPv2-TC import not found")

				// Assert SNMPv2-SMI imports
				expectedSmiImports := []types.SmiIdentifier{"MODULE-IDENTITY", "OBJECT-TYPE", "Counter32", "enterprises"}
				assert.ElementsMatch(t, expectedSmiImports, smiImport.Names, "SNMPv2-SMI imported names mismatch")

				// Assert SNMPv2-TC imports
				expectedTcImports := []types.SmiIdentifier{"DisplayString"}
				assert.ElementsMatch(t, expectedTcImports, tcImport.Names, "SNMPv2-TC imported names mismatch")

				// Check Node
				require.Len(t, mod.Body.Nodes, 1)
				assert.Equal(t, "testCounter", string(mod.Body.Nodes[0].Name))
			},
		},
		{
			name:    "Syntax Error - Missing BEGIN",
			input:   `TEST-MIB DEFINITIONS ::= testIdentity MODULE-IDENTITY ... END`,
			wantErr: true, // Expecting a parsing error
			check:   nil,  // No checks needed if error is expected
		},
		{
			name:    "Syntax Error - Invalid Keyword",
			input:   `TEST-MIB DEFINITIONS ::= BEGIN INVALID-KEYWORD END`,
			wantErr: true,
			check:   nil,
		},
		{
			name:    "Empty Input",
			input:   ``,
			wantErr: true, // Participle usually errors on empty input for the top-level rule
			check:   nil,
		},
		{
			name: "Comments Only",
			input: `
-- This is a comment MIB
-- It should be ignored by the parser, resulting in an error
-- because there's no actual module definition.
`,
			wantErr: true, // Expect error as no module definition found
			check:   nil,
		},
		{
			name: "Valid Module with Various Definitions",
			input: `
COMPLEX-MIB DEFINITIONS ::= BEGIN
    IMPORTS OBJECT-TYPE, NOTIFICATION-TYPE, MODULE-IDENTITY, enterprises FROM SNMPv2-SMI;

    complexIdentity MODULE-IDENTITY
        LAST-UPDATED "202401010000Z"
        ORGANIZATION "Complex Org"
        CONTACT-INFO "complex@example.com"
        DESCRIPTION  "Complex desc."
    ::= { enterprises 99997 }

    complexScalar OBJECT-TYPE
        SYNTAX INTEGER (0..100)
        MAX-ACCESS read-only
        STATUS current
        DESCRIPTION "A scalar."
    ::= { complexIdentity 1 }

    complexNotification NOTIFICATION-TYPE
        STATUS current
        DESCRIPTION "A notification."
    ::= { complexIdentity 2 }

END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, "COMPLEX-MIB", string(mod.Name))
				require.NotNil(t, mod.Body.Identity)
				assert.Equal(t, "complexIdentity", string(mod.Body.Identity.Name))
				require.Len(t, mod.Body.Nodes, 2) // scalar + notification
				assert.Equal(t, "complexScalar", string(mod.Body.Nodes[0].Name))
				assert.NotNil(t, mod.Body.Nodes[0].ObjectType)
				assert.Equal(t, "complexNotification", string(mod.Body.Nodes[1].Name))
				assert.NotNil(t, mod.Body.Nodes[1].NotificationType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use parser.Parse directly to test error handling
			mod, err := parser.Parse(tt.name+".mib", strings.NewReader(tt.input)) // Provide a dummy filename

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

func TestParseFile(t *testing.T) {
	validMibContent := `
VALID-FILE-MIB DEFINITIONS ::= BEGIN
    IMPORTS MODULE-IDENTITY, enterprises FROM SNMPv2-SMI;
    validIdentity MODULE-IDENTITY
        LAST-UPDATED "202401010000Z"
        ORGANIZATION "Valid File Org"
        CONTACT-INFO "valid@example.com"
        DESCRIPTION  "Valid file desc."
    ::= { enterprises 99996 }
END
`
	invalidMibContent := `
INVALID-FILE-MIB DEFINITIONS ::= BEGIN
    -- Missing required fields for MODULE-IDENTITY
    invalidIdentity MODULE-IDENTITY
    ::= { enterprises 99995 }
END
`

	tempDir := t.TempDir()

	validMibPath := filepath.Join(tempDir, "valid.mib")
	err := os.WriteFile(validMibPath, []byte(validMibContent), 0600)
	require.NoError(t, err, "Failed to write valid temp MIB file")

	invalidMibPath := filepath.Join(tempDir, "invalid.mib")
	err = os.WriteFile(invalidMibPath, []byte(invalidMibContent), 0600)
	require.NoError(t, err, "Failed to write invalid temp MIB file")

	nonExistentPath := filepath.Join(tempDir, "nonexistent.mib")

	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Parse Valid File",
			path:    validMibPath,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, "VALID-FILE-MIB", string(mod.Name))
				require.NotNil(t, mod.Body.Identity)
				assert.Equal(t, "validIdentity", string(mod.Body.Identity.Name))
				assert.Equal(t, "Valid File Org", mod.Body.Identity.Organization)
			},
		},
		{
			name:    "Parse Invalid File (Syntax Error)",
			path:    invalidMibPath,
			wantErr: true, // Expect parser error due to missing fields
			check:   nil,
		},
		{
			name:    "Parse Non-Existent File",
			path:    nonExistentPath,
			wantErr: true, // Expect file system error
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod, err := parser.ParseFile(tt.path)

			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
				// For non-existent file, check for os error specifically if needed
				if tt.name == "Parse Non-Existent File" {
					require.True(t, errors.Is(err, os.ErrNotExist), "Expected file not found error, got: %v", err)
				}
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
