package parser_test

import (
	"strings"
	"testing"

	"github.com/sleepinggenius2/gosmi/parser"
	"github.com/stretchr/testify/assert"
)

func TestParseModuleIdentityStrings(t *testing.T) {
	mibSnippet := `
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
             It includes special chars like ' and \". "
    REVISION      "202404300000Z"
    DESCRIPTION
            "Initial revision."
    ::= { experimental 999 }

END
`
	module, err := parser.Parse("test.mib", strings.NewReader(mibSnippet))
	assert.NoError(t, err, "Parsing MODULE-IDENTITY snippet failed")
	assert.NotNil(t, module, "Parsed module is nil")
	assert.NotNil(t, module.Body.Identity, "Parsed module identity is nil")

	identity := module.Body.Identity
	assert.Equal(t, parser.Date("202404300000Z"), identity.LastUpdated, "LastUpdated mismatch") // Compare with parser.Date type
	assert.Equal(t, "Test Org", identity.Organization, "Organization mismatch")
	// Check content, less strict on exact whitespace
	assert.Contains(t, identity.ContactInfo, "Contact Name", "ContactInfo missing name")
	assert.Contains(t, identity.ContactInfo, "Email: test@example.com", "ContactInfo missing email")
	assert.Contains(t, identity.ContactInfo, "Phone: +1234567890", "ContactInfo missing phone")
	// Check content, less strict on exact whitespace
	assert.Contains(t, identity.Description, "This is a multi-line", "Description missing first part")
	assert.Contains(t, identity.Description, "description for testing.", "Description missing second part")
	assert.Contains(t, identity.Description, "special chars like ' and \".", "Description missing third part")

	// Check revision description as well
	assert.Len(t, identity.Revisions, 1, "Expected 1 revision")
	if len(identity.Revisions) == 1 {
		assert.Equal(t, "Initial revision.", identity.Revisions[0].Description, "Revision description mismatch")
	}
}

func TestParseObjectTypeStrings(t *testing.T) {
	mibSnippet := `
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
`
	module, err := parser.Parse("test.mib", strings.NewReader(mibSnippet))
	assert.NoError(t, err, "Parsing OBJECT-TYPE snippet failed")
	assert.NotNil(t, module, "Parsed module is nil")
	assert.Len(t, module.Body.Nodes, 1, "Expected 1 node")

	node := module.Body.Nodes[0]
	assert.NotNil(t, node.ObjectType, "Parsed object type is nil")
	objectType := node.ObjectType
	// Check content, less strict on exact whitespace
	assert.Contains(t, objectType.Description, "This object has a description.", "ObjectType Description missing first part")
	assert.Contains(t, objectType.Description, "spans multiple lines.", "ObjectType Description missing second part")
	assert.Contains(t, objectType.Description, "Includes 'single' and \"double\" quotes.", "ObjectType Description missing third part")
	assert.Equal(t, "RFC XXXX Section Y", objectType.Reference, "Reference mismatch")
}
