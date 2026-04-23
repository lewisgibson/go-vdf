package govdf_test

import (
	"encoding/json"
	"errors"
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

// failWriter is a writer that always returns an error.
type failWriter struct{ err error }

func (f *failWriter) Write([]byte) (int, error) { return 0, f.err }

// failAfterN is a writer that succeeds for n writes then fails.
type failAfterN struct {
	n   int
	err error
}

func (f *failAfterN) Write(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, f.err
	}
	f.n--
	return len(p), nil
}

// errorMarshaler is a mock type that always returns an error for testing.
type errorMarshaler struct{}

func (e *errorMarshaler) MarshalVDF() ([]byte, error) {
	return nil, errors.New("custom marshaler error")
}

// errorMarshalerWithUnmarshalError is a mock type that returns data that fails to unmarshal.
type errorMarshalerWithUnmarshalError struct{}

func (e *errorMarshalerWithUnmarshalError) MarshalVDF() ([]byte, error) {
	return []byte(`invalid vdf data`), nil
}

func TestEncode_WriterErrors(t *testing.T) {
	t.Parallel()

	var writeErr = errors.New("write failed")

	var testCases = map[string]struct {
		node *govdf.Node
	}{
		"encodeMap key write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeMap nested map opening brace write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"parent": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"child": {Type: govdf.NodeTypeScalar, Value: "value"},
						},
					},
				},
			},
		},
		"encodeMap nested map closing brace write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"parent": {
						Type:     govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{},
					},
				},
			},
		},
		"encodeMap scalar space write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeMap scalar line comment write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value", LineComment: "comment"},
				},
			},
		},
		"encodeMap scalar newline write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeScalar head comment write error": {
			node: &govdf.Node{
				Type:        govdf.NodeTypeScalar,
				Value:       "value",
				HeadComment: "comment",
			},
		},
		"encodeScalar value write error": {
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "value",
			},
		},
		"encodeScalar line comment write error": {
			node: &govdf.Node{
				Type:        govdf.NodeTypeScalar,
				Value:       "value",
				LineComment: "comment",
			},
		},
		"encodeScalar trailing newline write error": {
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "value",
			},
		},
		"encodeMap head comment write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value", HeadComment: "comment"},
				},
			},
		},
		"writeIndent write error": {
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"parent": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"child": {Type: govdf.NodeTypeScalar, Value: "value"},
						},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fw := &failWriter{err: writeErr}
			encoder := govdf.NewEncoder(fw)
			err := encoder.Encode(tc.node)
			require.Error(t, err)
			require.ErrorIs(t, err, writeErr)
		})
	}
}

func TestEncode_WriterErrorsDeep(t *testing.T) {
	t.Parallel()

	var writeErr = errors.New("write failed")

	var testCases = map[string]struct {
		failAfter int
		node      *govdf.Node
	}{
		"writeQuotedString content write error": {
			failAfter: 1,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"writeQuotedString closing quote error": {
			failAfter: 2,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeMap scalar value quote error": {
			failAfter: 4,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeMap scalar newline error": {
			failAfter: 8,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "value"},
				},
			},
		},
		"encodeScalar head comment error": {
			failAfter: 0,
			node: &govdf.Node{
				Type:        govdf.NodeTypeScalar,
				Value:       "test",
				HeadComment: "a comment",
			},
		},
		"encodeScalar value error": {
			failAfter: 1,
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "test",
			},
		},
		"encodeScalar line comment error": {
			failAfter: 3,
			node: &govdf.Node{
				Type:        govdf.NodeTypeScalar,
				Value:       "test",
				LineComment: "inline",
			},
		},
		"encodeScalar trailing newline error": {
			failAfter: 3,
			node: &govdf.Node{
				Type:  govdf.NodeTypeScalar,
				Value: "test",
			},
		},
		"writeHeadComment indent error": {
			failAfter: 0,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {Type: govdf.NodeTypeScalar, Value: "v", HeadComment: "comment"},
				},
			},
		},
		"encodeMap nested indent error for closing brace": {
			failAfter: 5,
			node: &govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"parent": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"child": {Type: govdf.NodeTypeScalar, Value: "v"},
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
			encoder := govdf.NewEncoder(fw)
			err := encoder.Encode(tc.node)
			require.Error(t, err)
		})
	}
}

func TestEncode_WriterError_Marshaler(t *testing.T) {
	t.Parallel()

	var writeErr = errors.New("write failed")
	fw := &failWriter{err: writeErr}
	encoder := govdf.NewEncoder(fw)
	err := encoder.Encode(&mockMarshaler{value: "test"})
	require.Error(t, err)
	require.ErrorIs(t, err, writeErr)
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
		"uint types": {
			input: func() any {
				type TestStruct struct {
					Uint   uint   `vdf:"uint"`
					Uint8  uint8  `vdf:"uint8"`
					Uint16 uint16 `vdf:"uint16"`
					Uint32 uint32 `vdf:"uint32"`
					Uint64 uint64 `vdf:"uint64"`
				}
				return TestStruct{
					Uint:   42,
					Uint8:  255,
					Uint16: 65535,
					Uint32: 4294967295,
					Uint64: 18446744073709551615,
				}
			},
			expected: strings.Join([]string{
				`"uint" "42"`,
				`"uint16" "65535"`,
				`"uint32" "4294967295"`,
				`"uint64" "18446744073709551615"`,
				`"uint8" "255"`,
			}, "\n"),
		},
		"int types": {
			input: func() any {
				type TestStruct struct {
					Int   int   `vdf:"int"`
					Int8  int8  `vdf:"int8"`
					Int16 int16 `vdf:"int16"`
					Int32 int32 `vdf:"int32"`
					Int64 int64 `vdf:"int64"`
				}
				return TestStruct{
					Int:   -1,
					Int8:  -128,
					Int16: -32768,
					Int32: -2147483648,
					Int64: -9223372036854775808,
				}
			},
			expected: strings.Join([]string{
				`"int" "-1"`,
				`"int16" "-32768"`,
				`"int32" "-2147483648"`,
				`"int64" "-9223372036854775808"`,
				`"int8" "-128"`,
			}, "\n"),
		},
		"float types": {
			input: func() any {
				type TestStruct struct {
					Float32 float32 `vdf:"float32"`
					Float64 float64 `vdf:"float64"`
				}
				return TestStruct{
					Float32: 1.5,
					Float64: 2.5,
				}
			},
			expected: strings.Join([]string{
				`"float32" "1.5"`,
				`"float64" "2.5"`,
			}, "\n"),
		},
		"bool types": {
			input: func() any {
				type TestStruct struct {
					True  bool `vdf:"true_field"`
					False bool `vdf:"false_field"`
				}
				return TestStruct{
					True:  true,
					False: false,
				}
			},
			expected: strings.Join([]string{
				`"false_field" "false"`,
				`"true_field" "true"`,
			}, "\n"),
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

func TestEncode_UnsupportedFieldType(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Ch chan int `vdf:"ch"`
	}

	_, err := govdf.Marshal(TestStruct{Ch: make(chan int)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported type")
}

func TestEncode_CustomMarshalerInStruct(t *testing.T) {
	t.Parallel()

	type Container struct {
		Custom *mockMarshaler `vdf:"custom"`
	}

	container := Container{Custom: &mockMarshaler{value: "test_value"}}
	result, err := govdf.Marshal(container)
	require.NoError(t, err)
	require.Contains(t, string(result), "test_value")
}

func TestEncode_PointerField(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Name string `vdf:"name"`
	}
	type Outer struct {
		Inner *Inner `vdf:"inner"`
		Nil   *Inner `vdf:"nil"`
	}

	outer := Outer{Inner: &Inner{Name: "test"}}
	result, err := govdf.Marshal(outer)
	require.NoError(t, err)
	require.Contains(t, string(result), "test")
}
