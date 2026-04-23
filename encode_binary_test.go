package govdf_test

import (
	"bytes"
	"errors"
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func TestEncodeBinary_Node(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		node     *govdf.Node
		validate func(t *testing.T, data []byte)
	}{
		"simple app info": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"appid": {Type: govdf.NodeTypeScalar, Value: "730"},
							"name":  {Type: govdf.NodeTypeScalar, Value: "Counter-Strike 2"},
						},
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				var node govdf.Node
				require.NoError(t, govdf.UnmarshalBinary(data, &node))
				require.NotNil(t, node.Children["appinfo"])
				require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
				require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["name"].Value)
			},
		},
		"nested objects": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"common": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"name": {Type: govdf.NodeTypeScalar, Value: "Counter-Strike 2"},
									"type": {Type: govdf.NodeTypeScalar, Value: "Game"},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				var node govdf.Node
				require.NoError(t, govdf.UnmarshalBinary(data, &node))
				require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["common"].Children["name"].Value)
				require.Equal(t, "Game", node.Children["appinfo"].Children["common"].Children["type"].Value)
			},
		},
		"integer values written as int32": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"reviewscore": {Type: govdf.NodeTypeScalar, Value: "8"},
						},
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				require.Contains(t, string(data), "reviewscore")
				var node govdf.Node
				require.NoError(t, govdf.UnmarshalBinary(data, &node))
				require.Equal(t, "8", node.Children["appinfo"].Children["reviewscore"].Value)
			},
		},
		"non-integer string values": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"clienticon": {Type: govdf.NodeTypeScalar, Value: "324b323045b09bace182f928f4104dfcd93cb7f3"},
						},
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				var node govdf.Node
				require.NoError(t, govdf.UnmarshalBinary(data, &node))
				require.Equal(t, "324b323045b09bace182f928f4104dfcd93cb7f3", node.Children["appinfo"].Children["clienticon"].Value)
			},
		},
		"empty root": {
			node: &govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
			validate: func(t *testing.T, data []byte) {
				require.Equal(t, []byte{0x08}, data)
			},
		},
		"multiple root objects": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"appid": {Type: govdf.NodeTypeScalar, Value: "730"},
						},
					},
					"packageinfo": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"packaged": {Type: govdf.NodeTypeScalar, Value: "303386"},
						},
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				var node govdf.Node
				require.NoError(t, govdf.UnmarshalBinary(data, &node))
				require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
				require.Equal(t, "303386", node.Children["packageinfo"].Children["packaged"].Value)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := govdf.MarshalBinary(tc.node)
			require.NoError(t, err)
			tc.validate(t, data)
		})
	}
}

func TestEncodeBinary_ErrorHandling(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input any
	}{
		"nil value": {
			input: nil,
		},
		"nil node": {
			input: (*govdf.Node)(nil),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := govdf.MarshalBinary(tc.input)
			require.Error(t, err)
		})
	}
}

func TestEncodeBinary_Struct(t *testing.T) {
	t.Parallel()

	type Common struct {
		Name string `vdf:"name"`
		Type string `vdf:"type"`
	}
	type AppInfo struct {
		AppID  string `vdf:"appid"`
		Common Common `vdf:"common"`
	}
	type Root struct {
		AppInfo AppInfo `vdf:"appinfo"`
	}

	root := Root{
		AppInfo: AppInfo{
			AppID: "730",
			Common: Common{
				Name: "Counter-Strike 2",
				Type: "Game",
			},
		},
	}

	data, err := govdf.MarshalBinary(root)
	require.NoError(t, err)

	var node govdf.Node
	require.NoError(t, govdf.UnmarshalBinary(data, &node))
	require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
	require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["common"].Children["name"].Value)
	require.Equal(t, "Game", node.Children["appinfo"].Children["common"].Children["type"].Value)
}

func TestEncodeBinary_Roundtrip(t *testing.T) {
	t.Parallel()

	var original bytes.Buffer
	writeObject(&original, "appinfo")
	writeString(&original, "appid", "730")
	writeObject(&original, "common")
	writeString(&original, "name", "Counter-Strike 2")
	writeString(&original, "type", "Game")
	writeEnd(&original) // end common
	writeEnd(&original) // end appinfo
	writeEnd(&original) // end root

	var node govdf.Node
	require.NoError(t, govdf.UnmarshalBinary(original.Bytes(), &node))

	encoded, err := govdf.MarshalBinary(&node)
	require.NoError(t, err)

	var decoded govdf.Node
	require.NoError(t, govdf.UnmarshalBinary(encoded, &decoded))
	require.Equal(t, node.Children["appinfo"].Children["appid"].Value, decoded.Children["appinfo"].Children["appid"].Value)
	require.Equal(t, node.Children["appinfo"].Children["common"].Children["name"].Value, decoded.Children["appinfo"].Children["common"].Children["name"].Value)
}

func TestEncodeBinary_Encoder(t *testing.T) {
	t.Parallel()

	node := &govdf.Node{
		Type: govdf.NodeTypeMap,
		Children: map[string]*govdf.Node{
			"appinfo": {
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appid": {Type: govdf.NodeTypeScalar, Value: "730"},
				},
			},
		},
	}

	var buf bytes.Buffer
	encoder := govdf.NewBinaryEncoder(&buf)
	require.NoError(t, encoder.Encode(node))

	var decoded govdf.Node
	require.NoError(t, govdf.UnmarshalBinary(buf.Bytes(), &decoded))
	require.Equal(t, "730", decoded.Children["appinfo"].Children["appid"].Value)
}

func TestEncodeBinary_WriterErrors(t *testing.T) {
	t.Parallel()

	var writeErr = errors.New("write failed")

	var testCases = map[string]struct {
		node *govdf.Node
	}{
		"encodeObject end tag write error": {
			node: &govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
			},
		},
		"writeObjectTag type byte write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"child": {
						Type:     govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{},
					},
				},
			},
		},
		"writeObjectTag key write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"child": {
						Type:     govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{},
					},
				},
			},
		},
		"writeString type byte write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "non-integer-value"},
				},
			},
		},
		"writeString key write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "non-integer-value"},
				},
			},
		},
		"writeString value write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "non-integer-value"},
				},
			},
		},
		"writeInt32 type byte write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "42"},
				},
			},
		},
		"writeInt32 key write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "42"},
				},
			},
		},
		"writeInt32 value write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "42"},
				},
			},
		},
		"writeNullTerminatedString null byte write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fw := &failWriter{err: writeErr}
			encoder := govdf.NewBinaryEncoder(fw)
			err := encoder.Encode(tc.node)
			require.Error(t, err)
		})
	}
}

func TestEncodeBinary_WriterErrorsDeep(t *testing.T) {
	t.Parallel()

	var writeErr = errors.New("write failed")

	var testCases = map[string]struct {
		failAfter int
		node      *govdf.Node
	}{
		"writeString key null terminator error": {
			failAfter: 2,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "not-a-number"},
				},
			},
		},
		"writeString value write error": {
			failAfter: 3,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "not-a-number"},
				},
			},
		},
		"writeString value null terminator error": {
			failAfter: 4,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "not-a-number"},
				},
			},
		},
		"writeInt32 key null terminator error": {
			failAfter: 2,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "42"},
				},
			},
		},
		"writeInt32 binary write error": {
			failAfter: 3,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "42"},
				},
			},
		},
		"writeObjectTag key null terminator error": {
			failAfter: 2,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"child": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"key": {Type: govdf.NodeTypeScalar, Value: "v"},
						},
					},
				},
			},
		},
		"encodeObject nested end tag error": {
			failAfter: 5,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"child": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"key": {Type: govdf.NodeTypeScalar, Value: "v"},
						},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fw := &failAfterN{n: tc.failAfter, err: writeErr}
			encoder := govdf.NewBinaryEncoder(fw)
			err := encoder.Encode(tc.node)
			require.Error(t, err)
		})
	}
}

func TestEncodeBinary_NonNodeNonStructInput(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input any
	}{
		"string input": {
			input: "not a struct",
		},
		"int input": {
			input: 42,
		},
		"slice input": {
			input: []string{"a", "b"},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := govdf.MarshalBinary(tc.input)
			require.Error(t, err)
		})
	}
}

func TestEncodeBinary_UnknownNodeType(t *testing.T) {
	t.Parallel()

	node := &govdf.Node{
		Type: govdf.NodeTypeMap,
		Children: map[string]*govdf.Node{
			"child": {
				Type: govdf.NodeType(99),
			},
		},
	}

	_, err := govdf.MarshalBinary(node)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown node type")
}
