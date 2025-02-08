package govdf_test

import (
	"encoding/json"
	"strings"
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_TypeConstants(t *testing.T) {
	t.Parallel()

	// Assert: The node type constants should not change.
	assert.Equalf(t, govdf.NodeType(0), govdf.NodeTypeMap, "This value should not be changed")    //nolint:testifylint // The values are in the correct order.
	assert.Equalf(t, govdf.NodeType(1), govdf.NodeTypeScalar, "This value should not be changed") //nolint:testifylint // The values are in the correct order.
}

func TestNode_MarshalJSON(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		node     govdf.Node
		expected string
	}{
		"map node": {
			node: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"name": {Type: govdf.NodeTypeScalar, Value: "John"},
					"age":  {Type: govdf.NodeTypeScalar, Value: "30"},
				},
			},
			expected: `{"age":"30","name":"John"}`,
		},
		"scalar node": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello world",
			},
			expected: `"hello world"`,
		},
		"empty map": {
			node: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
			expected: `{}`,
		},
		"empty string": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "",
			},
			expected: `""`,
		},
		"nested map": {
			node: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"user": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"name": {Type: govdf.NodeTypeScalar, Value: "Jane"},
							"age":  {Type: govdf.NodeTypeScalar, Value: "25"},
						},
					},
				},
			},
			expected: `{"user":{"age":"25","name":"Jane"}}`,
		},
		"string with quotes": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: `hello "world"`,
			},
			expected: `"hello \"world\""`,
		},
		"string with special characters": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello\nworld\twith\ttabs",
			},
			expected: `"hello\nworld\twith\ttabs"`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Marshal the node into JSON.
			bytes, err := json.Marshal(tc.node)
			require.NoError(t, err)

			// Assert: The node should match the expected node.
			require.Equal(t, tc.expected, string(bytes))
		})
	}
}

func TestNode_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input    string
		expected govdf.Node
	}{
		"map node": {
			input: `{"name": "John", "age": 30}`,
			expected: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"name": {Type: govdf.NodeTypeScalar, Value: "John"},
					"age":  {Type: govdf.NodeTypeScalar, Value: "30"},
				},
			},
		},
		"nested map": {
			input: `{"user": {"name": "Jane", "details": {"age": 25}}}`,
			expected: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"user": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"name": {Type: govdf.NodeTypeScalar, Value: "Jane"},
							"details": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"age": {Type: govdf.NodeTypeScalar, Value: "25"},
								},
							},
						},
					},
				},
			},
		},
		"scalar node": {
			input: `"hello world"`,
			expected: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello world",
			},
		},
		"empty map": {
			input: `{}`,
			expected: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
		"empty string": {
			input: `""`,
			expected: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "",
			},
		},
		"null (treated as empty map)": {
			input: `null`,
			expected: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Unmarshal the input into a node.
			node := govdf.Node{}
			require.NoError(t, json.Unmarshal([]byte(tc.input), &node))

			// Assert: The node should match the expected node.
			require.Equal(t, tc.expected, node)
		})
	}
}

func TestNode_JSONEdgeCases(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input    string
		expected govdf.Node
	}{
		"invalid JSON": {
			input: `{invalid json}`,
			expected: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
		"null value": {
			input: `null`,
			expected: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
		"empty object": {
			input: `{}`,
			expected: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
		"empty string": {
			input: `""`,
			expected: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "",
			},
		},
		"string with special characters": {
			input: `"hello\nworld\twith\ttabs"`,
			expected: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello\nworld\twith\ttabs",
			},
		},
		"string with unicode": {
			input: `"hello 世界 🌍"`,
			expected: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello 世界 🌍",
			},
		},
		"deeply nested object": {
			input: `{"level1": {"level2": {"level3": {"level4": "deep value"}}}}`,
			expected: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"level1": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"level2": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"level3": {
										Type: govdf.NodeTypeMap,
										Children: map[string]*govdf.Node{
											"level4": {Type: govdf.NodeTypeScalar, Value: "deep value"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node govdf.Node
			err := json.Unmarshal([]byte(tc.input), &node)

			if strings.Contains(tc.input, "invalid json") {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, node)
		})
	}
}

func TestNode_MarshalJSONEdgeCases(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		node     govdf.Node
		expected string
	}{
		"unknown node type": {
			node: govdf.Node{
				Type: govdf.NodeType(99), // Invalid node type
			},
			expected: `null`,
		},
		"map with nil children": {
			node: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: nil,
			},
			expected: `null`,
		},
		"scalar with empty value": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "",
			},
			expected: `""`,
		},
		"scalar with special characters": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello\nworld\twith\ttabs",
			},
			expected: `"hello\nworld\twith\ttabs"`,
		},
		"scalar with unicode": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello 世界 🌍",
			},
			expected: `"hello 世界 🌍"`,
		},
		"scalar with quotes": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: `hello "world"`,
			},
			expected: `"hello \"world\""`,
		},
		"scalar with backslashes": {
			node: govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: `hello\world`,
			},
			expected: `"hello\\world"`,
		},
		"map with empty children": {
			node: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
			expected: `{}`,
		},
		"map with nil child": {
			node: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key":  {Type: govdf.NodeTypeScalar, Value: "value"},
					"nil":  nil,
					"key2": {Type: govdf.NodeTypeScalar, Value: "value2"},
				},
			},
			expected: `{"key":"value","key2":"value2","nil":null}`,
		},
		"nested map": {
			node: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"nested": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"inner": {Type: govdf.NodeTypeScalar, Value: "value"},
						},
					},
				},
			},
			expected: `{"nested":{"inner":"value"}}`,
		},
		"deeply nested map": {
			node: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"level1": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"level2": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"level3": {Type: govdf.NodeTypeScalar, Value: "deep value"},
								},
							},
						},
					},
				},
			},
			expected: `{"level1":{"level2":{"level3":"deep value"}}}`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Marshal the node into JSON.
			result, err := json.Marshal(tc.node)

			// Assert: The node should match the expected node.
			if name == "unknown node type" {
				require.Error(t, err)
				return
			}

			// Assert: The node should match the expected node.
			require.NoError(t, err)
			require.Equal(t, tc.expected, string(result))
		})
	}
}
