package govdf

import (
	"encoding/json"
	"errors"
	"fmt"
)

// NodeType represents the type of a VDF node.
type NodeType uint8

const (
	// NodeTypeMap represents a node that contains a map of key-value pairs.
	// For map nodes, the Children field contains the key-value mappings,
	// and the Value field is empty.
	NodeTypeMap NodeType = iota

	// NodeTypeScalar represents a node that contains a single scalar value.
	// For scalar nodes, the Value field contains the actual data,
	// and the Children field is nil.
	NodeTypeScalar
)

// Node represents a single node in a VDF document tree.
// Each node can be either a map (containing key-value pairs) or a scalar (containing a single value).
// Nodes also preserve position information and comments from the original VDF file.
//
// Example VDF structure:
//
//	"root_key" {
//	    "nested_key" "value"  // This is a line comment
//	}
//
// Would be represented as:
//   - Root node: Type=NodeTypeMap, Children={"root_key": childNode}
//   - Child node: Type=NodeTypeMap, Children={"nested_key": scalarNode}
//   - Scalar node: Type=NodeTypeScalar, Value="value", LineComment="This is a line comment"
type Node struct {
	// Type indicates whether this node is a map or scalar value
	Type NodeType

	// Value contains the scalar value for NodeTypeScalar nodes.
	// This field is empty for NodeTypeMap nodes.
	Value string

	// Children contains the key-value mappings for NodeTypeMap nodes.
	// This field is nil for NodeTypeScalar nodes.
	Children map[string]*Node

	// HeadComment contains any comment that appears before this node.
	// Comments are preserved during parsing and included in output.
	HeadComment string

	// LineComment contains any comment that appears on the same line as this node.
	// Line comments are typically used for inline documentation.
	LineComment string

	// Line and Column provide the position of this node in the original VDF file.
	// These are 1-indexed and useful for error reporting and debugging.
	Line   int
	Column int
}

// MarshalJSON returns the JSON encoding of the node.
// Map nodes are encoded as JSON objects, scalar nodes as JSON strings.
// This allows VDF nodes to be easily converted to JSON format.
func (n Node) MarshalJSON() ([]byte, error) {
	switch n.Type {
	case NodeTypeMap:
		return json.Marshal(n.Children)

	case NodeTypeScalar:
		return json.Marshal(n.Value)

	default:
		return nil, fmt.Errorf("unknown node type: %d", n.Type)
	}
}

// UnmarshalJSON parses JSON data into the node.
// JSON objects are converted to map nodes, JSON strings to scalar nodes.
// This allows VDF nodes to be created from JSON data.
func (n *Node) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a map first
	var m map[string]any
	if err := json.Unmarshal(data, &m); err == nil {
		n.Type = NodeTypeMap
		n.Children = convertMapToNodes(m)
		return nil
	}

	// Try to unmarshal as a scalar value
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n.Type = NodeTypeScalar
		n.Value = s
		return nil
	}

	return errors.New("cannot unmarshal JSON data into Node")
}

// convertMapToNodes converts a map[string]any to map[string]*Node.
// This is a helper function used during JSON unmarshaling to recursively
// convert JSON objects into VDF node structures.
func convertMapToNodes(m map[string]any) map[string]*Node {
	var result = make(map[string]*Node)
	for k, v := range m {
		result[k] = convertValueToNode(v)
	}
	return result
}

// convertValueToNode converts any value to a *Node.
// This is a helper function used during JSON unmarshaling to convert
// JSON values (objects, strings, primitives) into appropriate VDF nodes.
func convertValueToNode(v any) *Node {
	switch val := v.(type) {
	case map[string]any:
		return &Node{
			Type:     NodeTypeMap,
			Children: convertMapToNodes(val),
		}

	case string:
		return &Node{
			Type:  NodeTypeScalar,
			Value: val,
		}

	default:
		return &Node{
			Type:  NodeTypeScalar,
			Value: fmt.Sprintf("%v", val),
		}
	}
}
