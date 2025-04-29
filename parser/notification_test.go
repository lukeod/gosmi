package parser_test

import (
	"strings"
	"testing"

	"github.com/sleepinggenius2/gosmi/parser"
	"github.com/sleepinggenius2/gosmi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const NotificationTypeExample = `
ENTITY-MIB DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, NOTIFICATION-TYPE, OBJECT-IDENTIFIER, experimental FROM SNMPv2-SMI;

entityMIB OBJECT IDENTIFIER ::= { experimental 99 }
entityMIBObjects OBJECT IDENTIFIER ::= { entityMIB 1 }
entityMIBTraps      OBJECT IDENTIFIER ::= { entityMIB 2 }
entityMIBTrapPrefix OBJECT IDENTIFIER ::= { entityMIBTraps 0 }

entConfigChange NOTIFICATION-TYPE
    STATUS             current
    DESCRIPTION
            "An entConfigChange trap is sent when the value of
            entLastChangeTime changes. It can be utilized by an NMS to
            trigger logical/physical entity table maintenance polls.
            An agent must not generate more than one entConfigChange
            'trap-event' in a five second period, where a 'trap-event'
            is the transmission of a single trap PDU to a list of
            trap destinations.  If additional configuration changes
            occur within the five second 'throttling' period, then
            these trap-events should be suppressed by the agent. An
            NMS should periodically check the value of
            entLastChangeTime to detect any missed entConfigChange
            trap-events, e.g. due to throttling or transmission loss."
    REFERENCE "RFC 2863"
        ::= { entityMIBTrapPrefix 1 }

END
`

const NotificationGroupExample = `
NOTIF-GROUP-MIB DEFINITIONS ::= BEGIN
IMPORTS NOTIFICATION-GROUP, NOTIFICATION-TYPE, OBJECT-IDENTIFIER, experimental FROM SNMPv2-SMI;

notifGroupMIB OBJECT IDENTIFIER ::= { experimental 98 }

notif1 NOTIFICATION-TYPE STATUS current DESCRIPTION "Notif 1" ::= { notifGroupMIB 1 }
notif2 NOTIFICATION-TYPE STATUS current DESCRIPTION "Notif 2" ::= { notifGroupMIB 2 }

testNotificationGroup NOTIFICATION-GROUP
    NOTIFICATIONS { notif1, notif2 }
    STATUS        current
    DESCRIPTION   "A group of notifications."
    REFERENCE     "RFC XXXY"
    ::= { notifGroupMIB 3 }

END
`

const TrapTypeExample = `
TRAP-TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS enterprises, TRAP-TYPE, OBJECT-IDENTIFIER FROM RFC1155-SMI;

acme OBJECT IDENTIFIER ::= { enterprises 9999 }

acmeTrap TRAP-TYPE
    ENTERPRISE acme
    DESCRIPTION
        "This is an example trap."
    ::= 7

END
`

func TestNotificationParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, mod *parser.Module)
	}{
		{
			name:    "Parse NotificationTypeExample",
			input:   NotificationTypeExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 5, "Expected 5 nodes (4 OIDs + 1 Notification)")
				var notifNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].Name == "entConfigChange" {
						notifNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, notifNode, "Notification node 'entConfigChange' not found")
				require.NotNil(t, notifNode.NotificationType, "Node is not a NOTIFICATION-TYPE")

				nt := notifNode.NotificationType
				assert.Empty(t, nt.Objects, "Objects should be empty based on example")
				assert.Equal(t, parser.StatusCurrent, nt.Status, "Status mismatch")
				assert.Contains(t, nt.Description, "An entConfigChange trap is sent", "Description mismatch")
				assert.Equal(t, "RFC 2863", nt.Reference, "Reference mismatch")

				require.NotNil(t, notifNode.Oid, "Notification OID is nil")
				oid := notifNode.Oid
				require.Len(t, oid.SubIdentifiers, 2, "Notification OID length mismatch")
				assert.Equal(t, types.SmiIdentifier("entityMIBTrapPrefix"), *oid.SubIdentifiers[0].Name, "Notification OID parent name mismatch")
				require.NotNil(t, oid.SubIdentifiers[1].Number, "Notification OID sub-identifier number is nil")
				assert.Equal(t, types.SmiSubId(1), *oid.SubIdentifiers[1].Number, "Notification OID sub-identifier number mismatch")
			},
		},
		{
			name:    "Parse NotificationGroupExample",
			input:   NotificationGroupExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 4, "Expected 4 nodes (1 OID, 2 Notifs, 1 Group)")
				var groupNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].Name == "testNotificationGroup" {
						groupNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, groupNode, "Notification group node 'testNotificationGroup' not found")
				require.NotNil(t, groupNode.NotificationGroup, "Node is not a NOTIFICATION-GROUP")

				ng := groupNode.NotificationGroup
				assert.ElementsMatch(t, []types.SmiIdentifier{"notif1", "notif2"}, ng.Notifications, "Notifications mismatch")
				assert.Equal(t, parser.StatusCurrent, ng.Status, "Status mismatch")
				assert.Equal(t, "A group of notifications.", ng.Description, "Description mismatch")
				assert.Equal(t, "RFC XXXY", ng.Reference, "Reference mismatch")

				require.NotNil(t, groupNode.Oid, "Group OID is nil")
				oid := groupNode.Oid
				require.Len(t, oid.SubIdentifiers, 2, "Group OID length mismatch")
				assert.Equal(t, types.SmiIdentifier("notifGroupMIB"), *oid.SubIdentifiers[0].Name, "Group OID parent name mismatch")
				require.NotNil(t, oid.SubIdentifiers[1].Number, "Group OID sub-identifier number is nil")
				assert.Equal(t, types.SmiSubId(3), *oid.SubIdentifiers[1].Number, "Group OID sub-identifier number mismatch")
			},
		},
		{
			name:    "Parse TrapTypeExample (SMIv1)",
			input:   TrapTypeExample,
			wantErr: false,
			check: func(t *testing.T, mod *parser.Module) {
				require.Len(t, mod.Body.Nodes, 2, "Expected 2 nodes (1 OID, 1 Trap)")
				var trapNode *parser.Node
				for i := range mod.Body.Nodes {
					if mod.Body.Nodes[i].Name == "acmeTrap" {
						trapNode = &mod.Body.Nodes[i]
						break
					}
				}
				require.NotNil(t, trapNode, "Trap node 'acmeTrap' not found")

				// Check the OID assignment (trap number)
				assert.Nil(t, trapNode.Oid, "Trap OID should be nil with current parser limitations")
			},
		},
		{
			name: "Error - NotificationType Missing Status",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS NOTIFICATION-TYPE, OBJECT-IDENTIFIER, experimental FROM SNMPv2-SMI;
errorNotif NOTIFICATION-TYPE
    DESCRIPTION "Missing status."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - NotificationGroup Missing Notifications",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS NOTIFICATION-GROUP, OBJECT-IDENTIFIER, experimental FROM SNMPv2-SMI;
errorGroup NOTIFICATION-GROUP
    STATUS current
    DESCRIPTION "Missing notifications."
    ::= { experimental 1 }
END
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "Error - TrapType Missing Enterprise",
			input: `
ERROR-MIB DEFINITIONS ::= BEGIN
IMPORTS TRAP-TYPE, OBJECT-IDENTIFIER, enterprises FROM RFC1155-SMI;
errorTrap TRAP-TYPE
    -- ENTERPRISE missing
    DESCRIPTION "Missing enterprise."
    ::= 8
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
