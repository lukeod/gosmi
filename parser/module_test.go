package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sleepinggenius2/gosmi/parser"
	"github.com/sleepinggenius2/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ModuleExample = `
FIZBIN-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, experimental
        FROM SNMPv2-SMI;

fizbin MODULE-IDENTITY
    LAST-UPDATED "199505241811Z"
    ORGANIZATION "IETF SNMPv2 Working Group"
    CONTACT-INFO
            "        Marshall T. Rose

             Postal: Dover Beach Consulting, Inc.
                     420 Whisman Court
                     Mountain View, CA  94043-2186
                     US

                Tel: +1 415 968 1052
                Fax: +1 415 968 2510

             E-mail: mrose@dbc.mtview.ca.us"

    DESCRIPTION
            "The MIB module for entities implementing the xxxx
            protocol."
    REVISION      "9505241811Z"
    DESCRIPTION
            "The latest version of this MIB module."
    REVISION      "9210070433Z"
    DESCRIPTION
            "The initial version of this MIB module, published in
            RFC yyyy."
    ::= { experimental 101 }

END
`

func TestModuleParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Parse ModuleExample",
			input:   ModuleExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, types.SmiIdentifier("FIZBIN-MIB"), mod.Name, "Module name mismatch")
				require.NotNil(t, mod.Body, "Module body is nil")

				require.Len(t, mod.Body.Imports, 1, "Expected 1 import statement")
				imp := mod.Body.Imports[0]
				assert.Equal(t, types.SmiIdentifier("SNMPv2-SMI"), imp.Module, "Import module name mismatch")
				expectedImports := []types.SmiIdentifier{"MODULE-IDENTITY", "OBJECT-TYPE", "experimental"}
				assert.ElementsMatch(t, expectedImports, imp.Names, "Imported names mismatch")

				require.NotNil(t, mod.Body.Identity, "Module identity is nil")
				identity := mod.Body.Identity
				assert.Equal(t, types.SmiIdentifier("fizbin"), identity.Name, "Identity name mismatch")

				expectedLastUpdated, _ := time.Parse(time.RFC3339, "1995-05-24T18:11:00Z")
				assert.Equal(t, expectedLastUpdated, identity.LastUpdated.ToTime(), "LastUpdated mismatch")

				assert.Equal(t, "IETF SNMPv2 Working Group", identity.Organization, "Organization mismatch")
				assert.Contains(t, identity.ContactInfo, "Marshall T. Rose", "ContactInfo missing name")
				assert.Contains(t, identity.ContactInfo, "mrose@dbc.mtview.ca.us", "ContactInfo missing email")
				assert.Contains(t, identity.Description, "The MIB module for entities implementing the xxxx", "Description mismatch")

				require.Len(t, identity.Revisions, 2, "Expected 2 revisions")
				rev1 := identity.Revisions[0]
				rev2 := identity.Revisions[1]
				expectedRev1Date, _ := time.Parse(time.RFC3339, "1995-05-24T18:11:00Z")
				expectedRev2Date, _ := time.Parse(time.RFC3339, "1992-10-07T04:33:00Z")
				assert.Equal(t, expectedRev1Date, rev1.Date.ToTime(), "Revision 1 date mismatch")
				assert.Equal(t, "The latest version of this MIB module.", rev1.Description, "Revision 1 description mismatch")
				assert.Equal(t, expectedRev2Date, rev2.Date.ToTime(), "Revision 2 date mismatch")
				assert.Contains(t, rev2.Description, "The initial version of this MIB module", "Revision 2 description mismatch")

				require.Len(t, identity.Oid.SubIdentifiers, 2, "Identity OID length mismatch")
				assert.Equal(t, types.SmiIdentifier("experimental"), *identity.Oid.SubIdentifiers[0].Name, "Identity OID parent name mismatch")
				assert.Nil(t, identity.Oid.SubIdentifiers[0].Number, "Identity OID parent should not have number")
				assert.Nil(t, identity.Oid.SubIdentifiers[1].Name, "Identity OID sub-identifier should not have name")
				require.NotNil(t, identity.Oid.SubIdentifiers[1].Number, "Identity OID sub-identifier number is nil")
				assert.Equal(t, types.SmiSubId(101), *identity.Oid.SubIdentifiers[1].Number, "Identity OID sub-identifier number mismatch")
			},
		},
		{
			name: "Module No Imports",
			input: `
NOIMPORTS-MIB DEFINITIONS ::= BEGIN
    noImportsIdentity MODULE-IDENTITY
        LAST-UPDATED "202301010000Z"
        ORGANIZATION "No Imports Inc."
        CONTACT-INFO "ni@example.com"
        DESCRIPTION  "Module without imports."
    ::= { enterprises 12345 }
END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, types.SmiIdentifier("NOIMPORTS-MIB"), mod.Name)
				assert.Empty(t, mod.Body.Imports, "Should have no imports")
				require.NotNil(t, mod.Body.Identity)
				assert.Equal(t, types.SmiIdentifier("noImportsIdentity"), mod.Body.Identity.Name)
			},
		},
		{
			name: "Module with Exports (less common)",
			input: `
EXPORTS-MIB DEFINITIONS ::= BEGIN
			 EXPORTS myExportedObject;

			 exportsIdentity MODULE-IDENTITY
        LAST-UPDATED "202301010000Z"
        ORGANIZATION "Exports Inc."
        CONTACT-INFO "ex@example.com"
        DESCRIPTION  "Module with exports."
    ::= { enterprises 12346 }

    myExportedObject OBJECT IDENTIFIER ::= { exportsIdentity 1 }
END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Equal(t, types.SmiIdentifier("EXPORTS-MIB"), mod.Name)
				require.Len(t, mod.Body.Exports, 1, "Expected 1 export")
				assert.Equal(t, types.SmiIdentifier("myExportedObject"), mod.Body.Exports[0])
				require.NotNil(t, mod.Body.Identity)
				assert.Equal(t, types.SmiIdentifier("exportsIdentity"), mod.Body.Identity.Name)
				require.Len(t, mod.Body.Nodes, 1)
				assert.Equal(t, types.SmiIdentifier("myExportedObject"), mod.Body.Nodes[0].Name)
			},
		},
		{
			name: "Syntax Error - Missing Module Identity",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
    IMPORTS OBJECT-TYPE FROM SNMPv2-SMI;
END
`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				assert.Nil(t, mod.Body.Identity, "Module identity should be nil")
				assert.Empty(t, mod.Body.Nodes, "Should have no nodes")
			},
		},
		{
			name: "Syntax Error - Invalid OID Assignment",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
    errorIdentity MODULE-IDENTITY
        LAST-UPDATED "202301010000Z"
        ORGANIZATION "Error Inc."
        CONTACT-INFO "err@example.com"
        DESCRIPTION  "Invalid OID."
    ::= { enterprises .1 }
END
`,
			wantErr: true,
			check:   nil,
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
