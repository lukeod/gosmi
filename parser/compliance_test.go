package parser_test

import (
	"strings"
	"testing"

	"github.com/sleepinggenius2/gosmi/parser"
	"github.com/sleepinggenius2/gosmi/parser/testutil"
	"github.com/sleepinggenius2/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentCapabilitiesExample = `
AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, OBJECT-IDENTITY, OBJECT-GROUP, NOTIFICATION-GROUP, AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;

testAgentCaps AGENT-CAPABILITIES
    PRODUCT-RELEASE "Test Agent v1.0"
    STATUS          current
    DESCRIPTION
        "Capabilities of the test agent."
    REFERENCE       "Test Ref Doc"
    SUPPORTS        TEST-MODULE -- Implicit module name
        INCLUDES    { testGroup1, testGroup2 }
        VARIATION   testObject1
            SYNTAX      Integer32 (0..100)
            WRITE-SYNTAX Integer32 (10..20)
            ACCESS      read-write
            CREATION-REQUIRES { testObject2 }
            DEFVAL      { 50 }
            DESCRIPTION "Variation for testObject1"
        VARIATION   testObject2
            ACCESS      read-only
            DESCRIPTION "Variation for testObject2"
    SUPPORTS        OTHER-MODULE
        INCLUDES    { otherGroup }
        VARIATION   otherObject
            DESCRIPTION "Variation for otherObject"
          ::= { experimental 0 }
         
         testGroup1 OBJECT-GROUP OBJECTS { testObject1 } STATUS current DESCRIPTION "Group 1" ::= { experimental 1 }
testGroup2 NOTIFICATION-GROUP NOTIFICATIONS { testNotif1 } STATUS current DESCRIPTION "Group 2" ::= { experimental 2 }
testObject1 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-create STATUS current DESCRIPTION "Object 1" ::= { experimental 3 }
testObject2 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-create STATUS current DESCRIPTION "Object 2" ::= { experimental 4 }
testNotif1 NOTIFICATION-TYPE STATUS current DESCRIPTION "Notif 1" ::= { experimental 5 }
otherGroup OBJECT-GROUP OBJECTS { otherObject } STATUS current DESCRIPTION "Other Group" ::= { experimental 6 }
otherObject OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Other Object" ::= { experimental 7 }

END
`

func TestAgentCapabilitiesParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Full AgentCapabilities Example",
			input:   agentCapabilitiesExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				var capsNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].AgentCapabilities != nil {
						capsNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, capsNode, "AGENT-CAPABILITIES node not found")
				require.NotNil(t, capsNode.AgentCapabilities)
				caps := capsNode.AgentCapabilities

				assert.Equal(t, "testAgentCaps", string(capsNode.Name))
				assert.Equal(t, "Test Agent v1.0", caps.ProductRelease)
				assert.Equal(t, parser.StatusCurrent, caps.Status)
				assert.Contains(t, caps.Description, "Capabilities of the test agent.")
				assert.Equal(t, "Test Ref Doc", caps.Reference)

				require.Len(t, caps.Modules, 2)

				mod1 := caps.Modules[0]
				assert.Equal(t, types.SmiIdentifier("TEST-MODULE"), mod1.Module)
				require.Len(t, mod1.Includes, 2)
				assert.Equal(t, types.SmiIdentifier("testGroup1"), mod1.Includes[0])
				assert.Equal(t, types.SmiIdentifier("testGroup2"), mod1.Includes[1])
				require.Len(t, mod1.Variations, 2)
				var1 := mod1.Variations[0]
				assert.Equal(t, types.SmiIdentifier("testObject1"), var1.Name)
				require.NotNil(t, var1.Syntax)
				require.NotNil(t, var1.Syntax.Type)
				assert.Equal(t, types.SmiIdentifier("Integer32"), var1.Syntax.Type.Name)
				require.NotNil(t, var1.Syntax.Type.SubType)
				require.Len(t, var1.Syntax.Type.SubType.Integer, 1)
				assert.Equal(t, "0", var1.Syntax.Type.SubType.Integer[0].Start)
				assert.Equal(t, "100", var1.Syntax.Type.SubType.Integer[0].End)

				require.NotNil(t, var1.WriteSyntax)
				require.NotNil(t, var1.WriteSyntax.Type)
				assert.Equal(t, types.SmiIdentifier("Integer32"), var1.WriteSyntax.Type.Name)
				require.NotNil(t, var1.WriteSyntax.Type.SubType)
				require.Len(t, var1.WriteSyntax.Type.SubType.Integer, 1)
				assert.Equal(t, "10", var1.WriteSyntax.Type.SubType.Integer[0].Start)
				assert.Equal(t, "20", var1.WriteSyntax.Type.SubType.Integer[0].End)

				require.NotNil(t, var1.Access)
				assert.Equal(t, parser.AccessReadWrite, *var1.Access)
				require.Len(t, var1.Creation, 1)
				assert.Equal(t, types.SmiIdentifier("testObject2"), var1.Creation[0])
				require.NotNil(t, var1.Defval)
				assert.Equal(t, "50", *var1.Defval)
				assert.Contains(t, var1.Description, "Variation for testObject1")

				var2 := mod1.Variations[1]
				assert.Equal(t, types.SmiIdentifier("testObject2"), var2.Name)
				assert.Nil(t, var2.Syntax)
				assert.Nil(t, var2.WriteSyntax)
				require.NotNil(t, var2.Access)
				assert.Equal(t, parser.AccessReadOnly, *var2.Access)
				assert.Empty(t, var2.Creation)
				assert.Nil(t, var2.Defval)
				assert.Contains(t, var2.Description, "Variation for testObject2")

				mod2 := caps.Modules[1]
				assert.Equal(t, types.SmiIdentifier("OTHER-MODULE"), mod2.Module)
				require.Len(t, mod2.Includes, 1)
				assert.Equal(t, types.SmiIdentifier("otherGroup"), mod2.Includes[0])
				require.Len(t, mod2.Variations, 1)
				var3 := mod2.Variations[0]
				assert.Equal(t, types.SmiIdentifier("otherObject"), var3.Name)
				assert.Nil(t, var3.Syntax)
				assert.Nil(t, var3.WriteSyntax)
				assert.Nil(t, var3.Access)
				assert.Empty(t, var3.Creation)
				assert.Nil(t, var3.Defval)
				assert.Contains(t, var3.Description, "Variation for otherObject")
			},
		},
		{
			name: "Minimal AgentCapabilities",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					minCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Min Agent"
						STATUS current
						DESCRIPTION "Minimal"
						SUPPORTS MIN-MOD INCLUDES { minGroup }
					::= { experimental 10 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				capsNode := testutil.FindNodeByName(t, mod, "minCaps")
				require.NotNil(t, capsNode.AgentCapabilities)
				caps := capsNode.AgentCapabilities
				assert.Equal(t, "Min Agent", caps.ProductRelease)
				assert.Equal(t, parser.StatusCurrent, caps.Status)
				assert.Equal(t, "Minimal", caps.Description)
				require.Len(t, caps.Modules, 1)
				assert.Equal(t, types.SmiIdentifier("MIN-MOD"), caps.Modules[0].Module)
				require.Len(t, caps.Modules[0].Includes, 1)
				assert.Equal(t, types.SmiIdentifier("minGroup"), caps.Modules[0].Includes[0])
				assert.Empty(t, caps.Modules[0].Variations)
			},
		},
		{
			name: "AgentCapabilities Missing PRODUCT-RELEASE",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						STATUS current
						DESCRIPTION "Missing Product Release"
						SUPPORTS MIN-MOD INCLUDES { minGroup }
					::= { experimental 10 }
					END`,
			wantErr: true,
		},
		{
			name: "AgentCapabilities Missing STATUS",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Bad Agent"
						DESCRIPTION "Missing Status"
						SUPPORTS MIN-MOD INCLUDES { minGroup }
					::= { experimental 10 }
					END`,
			wantErr: true,
		},
		{
			name: "AgentCapabilities Missing DESCRIPTION",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Bad Agent"
						STATUS current
						SUPPORTS MIN-MOD INCLUDES { minGroup }
					::= { experimental 10 }
					END`,
			wantErr: true,
		},
		{
			name: "AgentCapabilities Missing SUPPORTS",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Bad Agent"
						STATUS current
						DESCRIPTION "Missing Supports"
					::= { experimental 10 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				capsNode := testutil.FindNodeByName(t, mod, "badCaps")
				require.NotNil(t, capsNode.AgentCapabilities)
				assert.Empty(t, capsNode.AgentCapabilities.Modules)
			},
		},
		{
			name: "AgentCapabilityModule Missing INCLUDES",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Bad Agent"
						STATUS current
						DESCRIPTION "Missing Includes"
						SUPPORTS MIN-MOD
					::= { experimental 10 }
					END`,
			wantErr: true,
		},
		{
			name: "AgentCapabilityVariation Missing DESCRIPTION",
			input: `AGENT-CAPS-MIB DEFINITIONS ::= BEGIN
					IMPORTS AGENT-CAPABILITIES, experimental FROM SNMPv2-SMI;
					badCaps AGENT-CAPABILITIES
						PRODUCT-RELEASE "Bad Agent"
						STATUS current
						DESCRIPTION "Var Missing Desc"
						SUPPORTS MIN-MOD INCLUDES { minGroup }
							VARIATION badVar ACCESS read-only
					::= { experimental 10 }
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

const moduleComplianceExample = `
MOD-COMP-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, OBJECT-IDENTITY, OBJECT-GROUP, NOTIFICATION-GROUP, MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;

testCompliance MODULE-COMPLIANCE
    STATUS      current
    DESCRIPTION "Compliance statement for TEST-MIB."
    REFERENCE   "Test Ref"
    MODULE      -- Implicit module name (TEST-MIB assumed)
        MANDATORY-GROUPS { testGroup1 }
        GROUP   testGroup2
            DESCRIPTION "Optional group 2."
        OBJECT  testObject1
            SYNTAX      Integer32 (0..10)
            WRITE-SYNTAX Integer32 (1..5)
            MIN-ACCESS  read-only
            DESCRIPTION "Refinement for testObject1."
        OBJECT  testObject2
            MIN-ACCESS  not-accessible
            DESCRIPTION "Refinement for testObject2."
    MODULE      OTHER-MIB
        MANDATORY-GROUPS { otherGroup }
        OBJECT otherObject
        	DESCRIPTION "Refinement for otherObject."
       ::= { experimental 0 }
      
      testGroup1 OBJECT-GROUP OBJECTS { testObject1 } STATUS current DESCRIPTION "Group 1" ::= { experimental 1 }
testGroup2 OBJECT-GROUP OBJECTS { testObject2 } STATUS current DESCRIPTION "Group 2" ::= { experimental 2 }
testObject1 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-create STATUS current DESCRIPTION "Object 1" ::= { experimental 3 }
testObject2 OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Object 2" ::= { experimental 4 }
otherGroup OBJECT-GROUP OBJECTS { otherObject } STATUS current DESCRIPTION "Other Group" ::= { experimental 5 }
otherObject OBJECT-TYPE SYNTAX Integer32 MAX-ACCESS read-only STATUS current DESCRIPTION "Other Object" ::= { experimental 6 }

END
`

func TestModuleComplianceParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Full ModuleCompliance Example",
			input:   moduleComplianceExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				var compNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].ModuleCompliance != nil {
						compNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, compNode, "MODULE-COMPLIANCE node not found")
				require.NotNil(t, compNode.ModuleCompliance)
				comp := compNode.ModuleCompliance

				assert.Equal(t, "testCompliance", string(compNode.Name))
				assert.Equal(t, parser.StatusCurrent, comp.Status)
				assert.Contains(t, comp.Description, "Compliance statement for TEST-MIB.")
				assert.Equal(t, "Test Ref", comp.Reference)

				require.Len(t, comp.Modules, 2)

				mod1 := comp.Modules[0]
				assert.Equal(t, parser.ComplianceModuleName(""), mod1.Name)
				require.Len(t, mod1.MandatoryGroups, 1)
				assert.Equal(t, types.SmiIdentifier("testGroup1"), mod1.MandatoryGroups[0])
				require.Len(t, mod1.Compliances, 3)
				assert.NotNil(t, mod1.Compliances[0].Group)
				assert.Nil(t, mod1.Compliances[0].Object)
				grp1 := mod1.Compliances[0].Group
				assert.Equal(t, types.SmiIdentifier("testGroup2"), grp1.Name)
				assert.Contains(t, grp1.Description, "Optional group 2.")

				assert.Nil(t, mod1.Compliances[1].Group)
				assert.NotNil(t, mod1.Compliances[1].Object)
				obj1 := mod1.Compliances[1].Object
				assert.Equal(t, types.SmiIdentifier("testObject1"), obj1.Name)
				require.NotNil(t, obj1.Syntax)
				require.NotNil(t, obj1.Syntax.Type)
				assert.Equal(t, types.SmiIdentifier("Integer32"), obj1.Syntax.Type.Name)
				require.NotNil(t, obj1.Syntax.Type.SubType)
				require.Len(t, obj1.Syntax.Type.SubType.Integer, 1)
				assert.Equal(t, "0", obj1.Syntax.Type.SubType.Integer[0].Start)
				assert.Equal(t, "10", obj1.Syntax.Type.SubType.Integer[0].End)

				require.NotNil(t, obj1.WriteSyntax)
				require.NotNil(t, obj1.WriteSyntax.Type)
				assert.Equal(t, types.SmiIdentifier("Integer32"), obj1.WriteSyntax.Type.Name)
				require.NotNil(t, obj1.WriteSyntax.Type.SubType)
				require.Len(t, obj1.WriteSyntax.Type.SubType.Integer, 1)
				assert.Equal(t, "1", obj1.WriteSyntax.Type.SubType.Integer[0].Start)
				assert.Equal(t, "5", obj1.WriteSyntax.Type.SubType.Integer[0].End)

				require.NotNil(t, obj1.MinAccess)
				assert.Equal(t, parser.AccessReadOnly, *obj1.MinAccess)
				assert.Contains(t, obj1.Description, "Refinement for testObject1.")

				assert.Nil(t, mod1.Compliances[2].Group)
				assert.NotNil(t, mod1.Compliances[2].Object)
				obj2 := mod1.Compliances[2].Object
				assert.Equal(t, types.SmiIdentifier("testObject2"), obj2.Name)
				assert.Nil(t, obj2.Syntax)
				assert.Nil(t, obj2.WriteSyntax)
				require.NotNil(t, obj2.MinAccess)
				assert.Equal(t, parser.AccessNotAccessible, *obj2.MinAccess)
				assert.Contains(t, obj2.Description, "Refinement for testObject2.")

				mod2 := comp.Modules[1]
				assert.Equal(t, parser.ComplianceModuleName("OTHER-MIB"), mod2.Name)
				require.Len(t, mod2.MandatoryGroups, 1)
				assert.Equal(t, types.SmiIdentifier("otherGroup"), mod2.MandatoryGroups[0])
				require.Len(t, mod2.Compliances, 1)
				assert.Nil(t, mod2.Compliances[0].Group)
				assert.NotNil(t, mod2.Compliances[0].Object)
				obj3 := mod2.Compliances[0].Object
				assert.Equal(t, types.SmiIdentifier("otherObject"), obj3.Name)
				assert.Nil(t, obj3.Syntax)
				assert.Nil(t, obj3.WriteSyntax)
				assert.Nil(t, obj3.MinAccess)
				assert.Contains(t, obj3.Description, "Refinement for otherObject.")
			},
		},
		{
			name: "Minimal ModuleCompliance",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					minComp MODULE-COMPLIANCE
						STATUS current
						DESCRIPTION "Minimal"
						MODULE MIN-MOD
					::= { experimental 20 }
					END`,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				compNode := testutil.FindNodeByName(t, mod, "minComp")
				require.NotNil(t, compNode.ModuleCompliance)
				comp := compNode.ModuleCompliance
				assert.Equal(t, parser.StatusCurrent, comp.Status)
				assert.Equal(t, "Minimal", comp.Description)
				require.Len(t, comp.Modules, 1)
				assert.Equal(t, parser.ComplianceModuleName("MIN-MOD"), comp.Modules[0].Name)
				assert.Empty(t, comp.Modules[0].MandatoryGroups)
				assert.Empty(t, comp.Modules[0].Compliances)
			},
		},
		{
			name: "ModuleCompliance Missing STATUS",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					badComp MODULE-COMPLIANCE
						DESCRIPTION "Missing Status"
						MODULE MIN-MOD
					::= { experimental 20 }
					END`,
			wantErr: true,
		},
		{
			name: "ModuleCompliance Missing DESCRIPTION",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					badComp MODULE-COMPLIANCE
						STATUS current
						MODULE MIN-MOD
					::= { experimental 20 }
					END`,
			wantErr: true,
		},
		{
			name: "ModuleCompliance Missing MODULE clause",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					badComp MODULE-COMPLIANCE
						STATUS current
						DESCRIPTION "Missing Module"
					::= { experimental 20 }
					END`,
			wantErr: true,
		},
		{
			name: "ComplianceGroup Missing DESCRIPTION",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					badComp MODULE-COMPLIANCE
						STATUS current
						DESCRIPTION "Bad Group"
						MODULE MIN-MOD
							GROUP badGroup
					::= { experimental 20 }
					END`,
			wantErr: true,
		},
		{
			name: "ComplianceObject Missing DESCRIPTION",
			input: `MOD-COMP-MIB DEFINITIONS ::= BEGIN
					IMPORTS MODULE-COMPLIANCE, experimental FROM SNMPv2-SMI;
					badComp MODULE-COMPLIANCE
						STATUS current
						DESCRIPTION "Bad Object"
						MODULE MIN-MOD
							OBJECT badObject MIN-ACCESS read-only
					::= { experimental 20 }
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
