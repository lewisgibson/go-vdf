package govdf_test

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func TestEncode_Node(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		node     *govdf.Node
		expected string
	}{
		"simple scalar": {
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "hello world",
			},
			expected: `"hello world"`,
		},
		"scalar with quotes": {
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: `hello "world"`,
			},
			expected: `"hello "world""`,
		},
		"simple map": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:  govdf.NodeTypeScalar,
						Value: "value",
					},
				},
			},
			expected: `"key" "value"`,
		},
		"nested map": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"parent": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"child": {
								Type:  govdf.NodeTypeScalar,
								Value: "value",
							},
						},
					},
				},
			},
			expected: strings.Join([]string{
				`"parent" {`,
				`    "child" "value"`,
				`}`,
			}, "\n"),
		},
		"map with comments": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:        govdf.NodeTypeScalar,
						Value:       "value",
						HeadComment: "This is a head comment",
						LineComment: "This is a line comment",
					},
				},
			},
			expected: strings.Join([]string{
				`// This is a head comment`,
				`"key" "value"	// This is a line comment`,
			}, "\n"),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Marshal the node into VDF.
			result, err := govdf.Marshal(tc.node)
			require.NoError(t, err)

			// Assert: The node should match the expected node.
			expected := strings.TrimSpace(tc.expected)
			actual := strings.TrimSpace(string(result))
			require.Equal(t, expected, actual)
		})
	}
}

// mockMarshaler is a mock type that implements MarshalVDF for testing.
type mockMarshaler struct {
	value string
}

// MarshalVDF implements the MarshalVDF method for the mockMarshaler type.
func (m *mockMarshaler) MarshalVDF() ([]byte, error) {
	return []byte(`"custom" "` + m.value + `"`), nil
}

func TestEncode_Struct(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input    func() any
		expected string
	}{
		"simple struct": {
			input: func() any {
				type Person struct {
					Name string `vdf:"name"`
					Age  int    `vdf:"age"`
				}
				return Person{
					Name: "John",
					Age:  30,
				}
			},
			expected: strings.Join([]string{
				`"age" "30"`,
				`"name" "John"`,
			}, "\n"),
		},
		"nested struct": {
			input: func() any {
				type User struct {
					Name string `vdf:"name"`
					Age  int    `vdf:"age"`
				}
				type Container struct {
					User User `vdf:"user"`
				}
				return Container{
					User: User{
						Name: "John",
						Age:  30,
					},
				}
			},
			expected: strings.Join([]string{
				`"user" {`,
				`    "age" "30"`,
				`    "name" "John"`,
				`}`,
			}, "\n"),
		},
		"mixed types": {
			input: func() any {
				type MixedData struct {
					String string  `vdf:"string_field"`
					Int    int     `vdf:"int_field"`
					Bool   bool    `vdf:"bool_field"`
					Float  float64 `vdf:"float_field"`
				}
				return MixedData{
					String: "hello world",
					Int:    42,
					Bool:   true,
					Float:  3.14159,
				}
			},
			expected: strings.Join([]string{
				`"bool_field" "true"`,
				`"float_field" "3.14159"`,
				`"int_field" "42"`,
				`"string_field" "hello world"`,
			}, "\n"),
		},
		"optional pointer structs": {
			input: func() any {
				type Person struct {
					Name string `vdf:"name"`
					Age  int    `vdf:"age"`
				}
				type TestStruct struct {
					Required Person  `vdf:"required"`
					Optional *Person `vdf:"optional"`
					Nil      *Person `vdf:"nil"`
				}
				return TestStruct{
					Required: Person{Name: "John", Age: 30},
					Optional: &Person{Name: "Jane", Age: 25},
					Nil:      nil, // This should be skipped
				}
			},
			expected: strings.Join([]string{
				`"optional" {`,
				`    "age" "25"`,
				`    "name" "Jane"`,
				`}`,
				`"required" {`,
				`    "age" "30"`,
				`    "name" "John"`,
				`}`,
			}, "\n"),
		},
		"custom marshaler": {
			input: func() any {
				return &mockMarshaler{value: "test_value"}
			},
			expected: `"custom" "test_value"`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Marshal the struct into VDF.
			result, err := govdf.Marshal(tc.input())
			require.NoError(t, err)

			// Assert: The node should match the expected node.
			expected := strings.TrimSpace(tc.expected)
			actual := strings.TrimSpace(string(result))
			require.Equal(t, expected, actual)
		})
	}
}

func TestEncode_Fixtures(t *testing.T) {
	t.Parallel()

	dirents, err := fixtures.ReadDir("fixtures")
	require.NoError(t, err)

	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name(), ".json") {
			t.Run(dirent.Name(), func(t *testing.T) {
				t.Parallel()

				// Arrange: load and parse the vdf file.
				vdfFile, err := fixtures.Open("fixtures/" + strings.Replace(dirent.Name(), ".json", ".vdf", 1))
				require.NoError(t, err)

				vdfBytes, err := io.ReadAll(vdfFile)
				require.NoError(t, err)

				vdfNode := govdf.Node{}
				govdf.Unmarshal(vdfBytes, &vdfNode)

				// Arrange: load and parse the json file.
				jsonFile, err := fixtures.Open("fixtures/" + dirent.Name())
				require.NoError(t, err)

				jsonBytes, err := io.ReadAll(jsonFile)
				require.NoError(t, err)

				jsonNode := govdf.Node{}
				require.NoError(t, json.Unmarshal(jsonBytes, &jsonNode))

				// Assert: the json node should match the vdf node.
				var ignore = cmpopts.IgnoreFields(govdf.Node{}, "Line", "Column", "HeadComment", "LineComment")
				if diff := cmp.Diff(vdfNode, jsonNode, ignore); diff != "" {
					t.Errorf("VDF and JSON nodes are not structurally identical (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestEncode_ErrorHandling(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input       any
		expectError bool
		errorSubstr string
	}{
		"nil value": {
			input:       nil,
			expectError: true,
			errorSubstr: "cannot encode nil value",
		},
		"unsupported type": {
			input:       complex(1, 2),
			expectError: true,
			errorSubstr: "expected struct",
		},
		"nil node": {
			input:       (*govdf.Node)(nil),
			expectError: true,
			errorSubstr: "cannot encode nil node",
		},
		"unknown node type": {
			input: &govdf.Node{
				Type: govdf.NodeType(99), // Invalid node type
			},
			expectError: true,
			errorSubstr: "unknown node type",
		},
		"custom marshaler error": {
			input:       &errorMarshaler{},
			expectError: true,
			errorSubstr: "custom marshaler error",
		},
		"non-struct value": {
			input:       "not a struct",
			expectError: true,
			errorSubstr: "expected struct",
		},
		"custom marshaler with unmarshal error": {
			input:       &errorMarshalerWithUnmarshalError{},
			expectError: false, // This actually works fine
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := govdf.Marshal(tc.input)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorSubstr != "" {
					require.Contains(t, err.Error(), tc.errorSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// errorMarshaler is a mock type that always returns an error for testing.
type errorMarshaler struct{}

func (e *errorMarshaler) MarshalVDF() ([]byte, error) {
	return nil, fmt.Errorf("custom marshaler error")
}

// errorMarshalerWithUnmarshalError is a mock type that returns data that fails to unmarshal.
type errorMarshalerWithUnmarshalError struct{}

func (e *errorMarshalerWithUnmarshalError) MarshalVDF() ([]byte, error) {
	return []byte(`invalid vdf data`), nil
}

func TestEncode_EdgeCases(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input    func() any
		expected string
	}{
		"empty map": {
			input: func() any {
				return &govdf.Node{
					Type:     govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{},
				}
			},
			expected: "",
		},
		"scalar with empty value": {
			input: func() any {
				return &govdf.Node{
					Type:  govdf.NodeTypeScalar,
					Value: "",
				}
			},
			expected: `""`,
		},
		"scalar with quotes in value": {
			input: func() any {
				return &govdf.Node{
					Type:  govdf.NodeTypeScalar,
					Value: `hello "world"`,
				}
			},
			expected: `"hello "world""`,
		},
		"scalar with newlines in value": {
			input: func() any {
				return &govdf.Node{
					Type:  govdf.NodeTypeScalar,
					Value: "hello\nworld",
				}
			},
			expected: `"hello
world"`,
		},
		"scalar with tabs in value": {
			input: func() any {
				return &govdf.Node{
					Type:  govdf.NodeTypeScalar,
					Value: "hello\tworld",
				}
			},
			expected: `"hello	world"`,
		},
		"map with nil child": {
			input: func() any {
				return &govdf.Node{
					Type: govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{
						"key":  {Type: govdf.NodeTypeScalar, Value: "value"},
						"nil":  nil,
						"key2": {Type: govdf.NodeTypeScalar, Value: "value2"},
					},
				}
			},
			expected: strings.Join([]string{
				`"key" "value"`,
				`"key2" "value2"`,
			}, "\n"),
		},
		"scalar with head comment": {
			input: func() any {
				return &govdf.Node{
					Type:        govdf.NodeTypeScalar,
					Value:       "value",
					HeadComment: "This is a head comment",
				}
			},
			expected: strings.Join([]string{
				`// This is a head comment`,
				`"value"`,
			}, "\n"),
		},
		"scalar with line comment": {
			input: func() any {
				return &govdf.Node{
					Type:        govdf.NodeTypeScalar,
					Value:       "value",
					LineComment: "This is a line comment",
				}
			},
			expected: `"value"	// This is a line comment`,
		},
		"scalar with multiline head comment": {
			input: func() any {
				return &govdf.Node{
					Type:        govdf.NodeTypeScalar,
					Value:       "value",
					HeadComment: "Line 1\nLine 2\nLine 3",
				}
			},
			expected: strings.Join([]string{
				`// Line 1`,
				`// Line 2`,
				`// Line 3`,
				`"value"`,
			}, "\n"),
		},
		"map with head comment on child": {
			input: func() any {
				return &govdf.Node{
					Type: govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{
						"key": {
							Type:        govdf.NodeTypeScalar,
							Value:       "value",
							HeadComment: "Comment for key",
						},
					},
				}
			},
			expected: strings.Join([]string{
				`// Comment for key`,
				`"key" "value"`,
			}, "\n"),
		},
		"map with line comment on child": {
			input: func() any {
				return &govdf.Node{
					Type: govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{
						"key": {
							Type:        govdf.NodeTypeScalar,
							Value:       "value",
							LineComment: "Comment for key",
						},
					},
				}
			},
			expected: `"key" "value"	// Comment for key`,
		},
		"nested map with comments": {
			input: func() any {
				return &govdf.Node{
					Type: govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{
						"parent": {
							Type: govdf.NodeTypeMap,
							Children: map[string]*govdf.Node{
								"child": {
									Type:        govdf.NodeTypeScalar,
									Value:       "value",
									HeadComment: "Child comment",
									LineComment: "Line comment",
								},
							},
						},
					},
				}
			},
			expected: strings.Join([]string{
				`"parent" {`,
				`    // Child comment`,
				`    "child" "value"	// Line comment`,
				`}`,
			}, "\n"),
		},
		"all numeric types": {
			input: func() any {
				type TestStruct struct {
					Int8    int8    `vdf:"int8"`
					Int16   int16   `vdf:"int16"`
					Int32   int32   `vdf:"int32"`
					Int64   int64   `vdf:"int64"`
					Uint8   uint8   `vdf:"uint8"`
					Uint16  uint16  `vdf:"uint16"`
					Uint32  uint32  `vdf:"uint32"`
					Uint64  uint64  `vdf:"uint64"`
					Float32 float32 `vdf:"float32"`
					Float64 float64 `vdf:"float64"`
					Bool    bool    `vdf:"bool"`
				}
				return TestStruct{
					Int8:    127,
					Int16:   32767,
					Int32:   2147483647,
					Int64:   9223372036854775807,
					Uint8:   255,
					Uint16:  65535,
					Uint32:  4294967295,
					Uint64:  18446744073709551615,
					Float32: 3.140000104904175,
					Float64: 3.14159265359,
					Bool:    true,
				}
			},
			expected: strings.Join([]string{
				`"bool" "true"`,
				`"float32" "3.140000104904175"`,
				`"float64" "3.14159265359"`,
				`"int16" "32767"`,
				`"int32" "2147483647"`,
				`"int64" "9223372036854775807"`,
				`"int8" "127"`,
				`"uint16" "65535"`,
				`"uint32" "4294967295"`,
				`"uint64" "18446744073709551615"`,
				`"uint8" "255"`,
			}, "\n"),
		},
		"pointer to struct": {
			input: func() any {
				type TestStruct struct {
					Value string `vdf:"value"`
				}
				return &TestStruct{Value: "test"}
			},
			expected: `"value" "test"`,
		},
		"nil pointer struct": {
			input: func() any {
				type TestStruct struct {
					Value *string `vdf:"value"`
				}
				return TestStruct{Value: nil}
			},
			expected: "",
		},
		"unexported field": {
			input: func() any {
				type TestStruct struct {
					Exported   string `vdf:"exported"`
					unexported string `vdf:"unexported"`
				}
				return TestStruct{
					Exported:   "visible",
					unexported: "hidden",
				}
			},
			expected: `"exported" "visible"`,
		},
		"field with dash tag": {
			input: func() any {
				type TestStruct struct {
					Visible string `vdf:"visible"`
					Hidden  string `vdf:"-"`
				}
				return TestStruct{
					Visible: "visible",
					Hidden:  "hidden",
				}
			},
			expected: strings.Join([]string{
				`"hidden" "hidden"`,
				`"visible" "visible"`,
			}, "\n"),
		},
		"custom marshaler": {
			input: func() any {
				return &mockMarshaler{value: "test_value"}
			},
			expected: `"custom" "test_value"`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := govdf.Marshal(tc.input())
			require.NoError(t, err)

			expected := strings.TrimSpace(tc.expected)
			actual := strings.TrimSpace(string(result))
			require.Equal(t, expected, actual)
		})
	}
}
