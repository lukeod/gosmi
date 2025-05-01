package internal

import (
	"testing"

	"github.com/lukeod/gosmi/types"
)

func TestObjectGetSmiNode(t *testing.T) {
	// Test that OidLen is properly set from Node
	node := &Node{
		OidLen: 5,
		Oid:    types.Oid{1, 2, 3, 4, 5},
	}
	
	obj := &Object{
		SmiNode: types.SmiNode{
			Name: "testObject",
		},
		Node: node,
	}
	
	smiNode := obj.GetSmiNode()
	
	if smiNode.OidLen != 5 {
		t.Errorf("Expected OidLen to be 5, got %d", smiNode.OidLen)
	}
	
	if len(smiNode.Oid) != 5 {
		t.Errorf("Expected Oid length to be 5, got %d", len(smiNode.Oid))
	}
}
