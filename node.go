package govdf

import (
	"encoding/json"
	"fmt"
)

// NodeType represents the type of a node.
type NodeType uint8

const (
	// NodeTypeMap represents a node that contains a map of key-value pairs.
	// The key-value pairs are stored in the Value field of the node.
	NodeTypeMap NodeType = iota
	// NodeTypeScalar represents a node that contains a single value.
	NodeTypeScalar
)

// Node represents a node in the
type Node struct {
	Type     NodeType
	Value    string
	Children map[string]*Node

	HeadComment string
	LineComment string

	Line   int
	Column int
}

// Encode writes the VDF encoding of v to the stream.
func (n *Node) Encode(v any) error {
	return fmt.Errorf("not implemented")
}

// Decode reads the next VDF-encoded value from its input and stores it in the value pointed to by v.
func (n *Node) Decode(v any) error {
	return fmt.Errorf("not implemented")
}

// MarshalJSON returns the JSON encoding of the node.
func (n *Node) MarshalJSON() ([]byte, error) {
	switch n.Type {
	case NodeTypeMap:
		return json.Marshal(n.Children)

	case NodeTypeScalar:
		return json.Marshal(n.Value)

	default:
		return nil, fmt.Errorf("unknown node type: %d", n.Type)
	}
}
