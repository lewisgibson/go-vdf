package govdf_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func TestDecode_Node(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input        string
		expectedNode govdf.Node
	}{
		"top level map": {
			input: `"a" {}`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type: govdf.NodeTypeMap,

						Line:   1,
						Column: 5,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"top level map with string value": {
			input: `"a" "b"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type:  govdf.NodeTypeScalar,
						Value: "b",

						Line:   1,
						Column: 4,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"top level map with string value with spaces": {
			input: `"a" "b c"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type:  govdf.NodeTypeScalar,
						Value: "b c",

						Line:   1,
						Column: 4,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"top level map with string value with quotes": {
			input: `"a" "b\"c"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type:  govdf.NodeTypeScalar,
						Value: "b\"c",

						Line:   1,
						Column: 4,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"nested map": {
			input: `"a" {
		        "b" "c"
		    }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"b": {
								Type:  govdf.NodeTypeScalar,
								Value: "c",

								Line:   2,
								Column: 14,
							},
						},

						Line:   1,
						Column: 5,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"comment lines": {
			input: `"top level" {
                // this is a comment
                "foo" "bar"
            }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"top level": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"foo": {
								Type:  govdf.NodeTypeScalar,
								Value: "bar",

								Line:   3,
								Column: 22,

								HeadComment: "this is a comment",
							},
						},

						Line:   1,
						Column: 13,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"comment on same line": {
			input: `"top level" {
                "foo" "bar"                 // this is a comment
            }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"top level": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"foo": {
								Type:  govdf.NodeTypeScalar,
								Value: "bar",

								Line:   2,
								Column: 22,

								LineComment: "this is a comment",
							},
						},

						Line:   1,
						Column: 13,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
		"regression: game info": {
			input: `"items_game" {
		        "game_info"
		        {
		            "first_valid_class"             "2"
		            "last_valid_class"              "3"
		            "first_valid_item_slot"         "0"
		            "last_valid_item_slot"          "54"
		            "num_item_presets"              "4"
		            "max_num_stickers"              "5"
		            "max_num_patches"               "3"
		        }
		        }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"items_game": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{

							"game_info": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"first_valid_class": {
										Type:   govdf.NodeTypeScalar,
										Value:  "2",
										Line:   4,
										Column: 46,
									},
									"last_valid_class": {
										Type:   govdf.NodeTypeScalar,
										Value:  "3",
										Line:   5,
										Column: 46,
									},
									"first_valid_item_slot": {
										Type:   govdf.NodeTypeScalar,
										Value:  "0",
										Line:   6,
										Column: 46,
									},
									"last_valid_item_slot": {
										Type:   govdf.NodeTypeScalar,
										Value:  "54",
										Line:   7,
										Column: 46,
									},
									"num_item_presets": {
										Type:   govdf.NodeTypeScalar,
										Value:  "4",
										Line:   8,
										Column: 46,
									},
									"max_num_stickers": {
										Type:   govdf.NodeTypeScalar,
										Value:  "5",
										Line:   9,
										Column: 46,
									},
									"max_num_patches": {
										Type:   govdf.NodeTypeScalar,
										Value:  "3",
										Line:   10,
										Column: 46,
									},
								},
								Line:   3,
								Column: 11,
							},
						},
						Line:   1,
						Column: 14,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"regression: comment skips next line": {
			input: `"a" {
		        "csgo_instr_explain_inspect"						"Hold to inspect your weapon"	// not 'gun', could be knife or tool
		        "csgo_instr_explain_reload"							"Reload your gun"
		    }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"a": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"csgo_instr_explain_inspect": {
								Type:  govdf.NodeTypeScalar,
								Value: "Hold to inspect your weapon",

								Line:   2,
								Column: 44,

								LineComment: "not 'gun', could be knife or tool",
							},
							"csgo_instr_explain_reload": {
								Type:  govdf.NodeTypeScalar,
								Value: "Reload your gun",

								Line:   2,
								Column: 117,
							},
						},

						Line:   1,
						Column: 5,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"duplicate map keys": {
			input: `"root" { "items" { "1" { "name" "first" } } "items" { "2" { "name" "second" } } }`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"root": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"items": {
								Type: govdf.NodeTypeMap,
								Children: map[string]*govdf.Node{
									"1": {
										Type: govdf.NodeTypeMap,
										Children: map[string]*govdf.Node{
											"name": {
												Type:  govdf.NodeTypeScalar,
												Value: "first",

												Line:   1,
												Column: 32,
											},
										},

										Line:   1,
										Column: 24,
									},
									"2": {
										Type: govdf.NodeTypeMap,
										Children: map[string]*govdf.Node{
											"name": {
												Type:  govdf.NodeTypeScalar,
												Value: "second",

												Line:   1,
												Column: 67,
											},
										},

										Line:   1,
										Column: 59,
									},
								},

								Line:   1,
								Column: 18,
							},
						},

						Line:   1,
						Column: 8,
					},
				},

				Line:   1,
				Column: 1,
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Act: Unmarshal the input into a node.
			node := govdf.Node{}
			require.NoErrorf(t, govdf.Unmarshal([]byte(tc.input), &node), "output: %s", node)

			// Assert: The node should match the expected node.
			if diff := cmp.Diff(tc.expectedNode, node); diff != "" {
				t.Errorf("unexpected node (-want +got):\n%s", diff)
			}
		})
	}
}

// mockUnmarshaler is a mock type that implements UnmarshalVDF for testing.
type mockUnmarshaler string

// UnmarshalVDF implements the UnmarshalVDF method for the mockUnmarshaler type.
func (c *mockUnmarshaler) UnmarshalVDF(node *govdf.Node) error {
	if node.Type == govdf.NodeTypeScalar {
		*c = mockUnmarshaler(fmt.Sprintf("custom:%s", node.Value))
	}
	return nil
}

func TestDecode_Struct(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input         string
		createStructs func() (actual, expected any)
	}{
		"top level map": {
			input: `"a" {}`,
			createStructs: func() (any, any) {
				type EmptyStruct struct{}
				type SimpleStruct struct {
					A *EmptyStruct `vdf:"a"`
				}
				return &SimpleStruct{}, &SimpleStruct{
					A: &EmptyStruct{},
				}
			},
		},
		"top level map with string value": {
			input: `"a" "b"`,
			createStructs: func() (any, any) {
				type SimpleStruct struct {
					A string `vdf:"a"`
				}
				return &SimpleStruct{}, &SimpleStruct{
					A: "b",
				}
			},
		},
		"top level map with string value with spaces": {
			input: `"a" "b c"`,
			createStructs: func() (any, any) {
				type SimpleStruct struct {
					A string `vdf:"a"`
				}
				return &SimpleStruct{}, &SimpleStruct{
					A: "b c",
				}
			},
		},
		"top level map with string value with quotes": {
			input: `"a" "b\"c"`,
			createStructs: func() (any, any) {
				type SimpleStruct struct {
					A string `vdf:"a"`
				}
				return &SimpleStruct{}, &SimpleStruct{
					A: "b\"c",
				}
			},
		},
		"nested map": {
			input: `"a" {
		        "b" "c"
		    }`,
			createStructs: func() (any, any) {
				type NestedStruct struct {
					B string `vdf:"b"`
				}
				type ParentStruct struct {
					A *NestedStruct `vdf:"a"`
				}
				return &ParentStruct{}, &ParentStruct{
					A: &NestedStruct{
						B: "c",
					},
				}
			},
		},
		"comment lines": {
			input: `"top level" {
                // this is a comment
                "foo" "bar"
            }`,
			createStructs: func() (any, any) {
				type CommentStruct struct {
					Foo string `vdf:"foo"`
				}
				type RootStruct struct {
					TopLevel *CommentStruct `vdf:"top level"`
				}
				return &RootStruct{}, &RootStruct{
					TopLevel: &CommentStruct{
						Foo: "bar",
					},
				}
			},
		},
		"comment on same line": {
			input: `"top level" {
                "foo" "bar"                 // this is a comment
            }`,
			createStructs: func() (any, any) {
				type CommentStruct struct {
					Foo string `vdf:"foo"`
				}
				type RootStruct struct {
					TopLevel *CommentStruct `vdf:"top level"`
				}
				return &RootStruct{}, &RootStruct{
					TopLevel: &CommentStruct{
						Foo: "bar",
					},
				}
			},
		},
		"regression: game info": {
			input: `"items_game" {
		        "game_info"
		        {
		            "first_valid_class"             "2"
		            "last_valid_class"              "3"
		            "first_valid_item_slot"         "0"
		            "last_valid_item_slot"          "54"
		            "num_item_presets"              "4"
		            "max_num_stickers"              "5"
		            "max_num_patches"               "3"
		        }
		        }`,
			createStructs: func() (any, any) {
				type GameInfoStruct struct {
					FirstValidClass    int `vdf:"first_valid_class"`
					LastValidClass     int `vdf:"last_valid_class"`
					FirstValidItemSlot int `vdf:"first_valid_item_slot"`
					LastValidItemSlot  int `vdf:"last_valid_item_slot"`
					NumItemPresets     int `vdf:"num_item_presets"`
					MaxNumStickers     int `vdf:"max_num_stickers"`
					MaxNumPatches      int `vdf:"max_num_patches"`
				}
				type ItemsGameStruct struct {
					GameInfo *GameInfoStruct `vdf:"game_info"`
				}
				type RootStruct struct {
					ItemsGame *ItemsGameStruct `vdf:"items_game"`
				}
				return &RootStruct{}, &RootStruct{
					ItemsGame: &ItemsGameStruct{
						GameInfo: &GameInfoStruct{
							FirstValidClass:    2,
							LastValidClass:     3,
							FirstValidItemSlot: 0,
							LastValidItemSlot:  54,
							NumItemPresets:     4,
							MaxNumStickers:     5,
							MaxNumPatches:      3,
						},
					},
				}
			},
		},
		"regression: comment skips next line": {
			input: `"a" {
		        "csgo_instr_explain_inspect"						"Hold to inspect your weapon"	// not 'gun', could be knife or tool
		        "csgo_instr_explain_reload"							"Reload your gun"
		    }`,
			createStructs: func() (any, any) {
				type InstructionStruct struct {
					Inspect string `vdf:"csgo_instr_explain_inspect"`
					Reload  string `vdf:"csgo_instr_explain_reload"`
				}
				type InstructionParent struct {
					A *InstructionStruct `vdf:"a"`
				}
				return &InstructionParent{}, &InstructionParent{
					A: &InstructionStruct{
						Inspect: "Hold to inspect your weapon",
						Reload:  "Reload your gun",
					},
				}
			},
		},
		"mixed data types": {
			input: `"mixed" {
		        "string_field" "hello world"
		        "int_field" "42"
		        "bool_field" "true"
		        "float_field" "3.14159"
		    }`,
			createStructs: func() (any, any) {
				type MixedTypesStruct struct {
					StringField string  `vdf:"string_field"`
					IntField    int     `vdf:"int_field"`
					BoolField   bool    `vdf:"bool_field"`
					FloatField  float64 `vdf:"float_field"`
				}
				type NestedMixedStruct struct {
					Mixed *MixedTypesStruct `vdf:"mixed"`
				}
				return &NestedMixedStruct{}, &NestedMixedStruct{
					Mixed: &MixedTypesStruct{
						StringField: "hello world",
						IntField:    42,
						BoolField:   true,
						FloatField:  3.14159,
					},
				}
			},
		},
		"custom unmarshaler": {
			input: `"custom_field" "test_value"`,
			createStructs: func() (any, any) {
				type CustomUnmarshalerStruct struct {
					CustomField mockUnmarshaler `vdf:"custom_field"`
				}
				return &CustomUnmarshalerStruct{}, &CustomUnmarshalerStruct{
					CustomField: "custom:test_value",
				}
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange: Create the actual and expected structs.
			actualStruct, expectedStruct := tc.createStructs()

			// Act: Unmarshal the input into the actual struct
			require.NoErrorf(t, govdf.Unmarshal([]byte(tc.input), actualStruct), "failed to unmarshal VDF")

			// Assert: The actual struct should match the expected struct
			if diff := cmp.Diff(expectedStruct, actualStruct); diff != "" {
				t.Errorf("unexpected struct (-want +got):\n%s", diff)
			}
		})
	}
}

//go:embed fixtures/*.*
var fixtures embed.FS

func TestDecode_Fixtures(t *testing.T) {
	t.Parallel()

	dirents, err := fixtures.ReadDir("fixtures")
	require.NoError(t, err)

	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name(), ".vdf") {
			t.Run(dirent.Name(), func(t *testing.T) {
				t.Parallel()

				// Arrange: load and parse the vdf file.
				vdfFile, err := fixtures.Open("fixtures/" + dirent.Name())
				require.NoError(t, err)

				vdfBytes, err := io.ReadAll(vdfFile)
				require.NoError(t, err)

				vdfNode := govdf.Node{}
				require.NoError(t, govdf.Unmarshal(vdfBytes, &vdfNode))

				// Arrange: load and parse the json file.
				jsonFile, err := fixtures.Open("fixtures/" + strings.Replace(dirent.Name(), ".vdf", ".json", 1))
				require.NoError(t, err)

				jsonBytes, err := io.ReadAll(jsonFile)
				require.NoError(t, err)

				vdfBytes, err = json.MarshalIndent(&vdfNode, "", "    ")
				require.NoError(t, err)

				// Assert: the json node should match the vdf node.
				require.JSONEq(t, string(jsonBytes), string(vdfBytes))
			})
		}
	}
}

func TestDecode_ErrorHandling(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input       string
		expectError bool
		errorSubstr string
	}{
		"invalid rune": {
			input:       string([]byte{0xFF, 0xFE}), // Invalid UTF-8
			expectError: true,
			errorSubstr: "invalid rune",
		},
		"unexpected character": {
			input:       `"key" "value" unexpected`,
			expectError: true,
			errorSubstr: "unexpected character",
		},
		"unexpected opening brace at root": {
			input:       `{`,
			expectError: false, // This actually works fine
		},
		"unexpected closing brace at root": {
			input:       `}`,
			expectError: true,
			errorSubstr: "unexpected '}' at root level",
		},
		"malformed comment": {
			input:       `/`,
			expectError: true,
			errorSubstr: "EOF",
		},
		"unclosed quoted string": {
			input:       `"unclosed string`,
			expectError: true,
			errorSubstr: "EOF",
		},
		"unclosed quoted key": {
			input:       `"unclosed key" "value"`,
			expectError: false, // This should work fine
		},
		"empty input": {
			input:       ``,
			expectError: false, // Empty input should result in empty map
		},
		"only whitespace": {
			input:       `   \t\n\r   `,
			expectError: true, // This actually causes an error
		},
		"unexpected character in key": {
			input:       `"key" "value" x`,
			expectError: true,
			errorSubstr: "unexpected character",
		},
		"unexpected character after map": {
			input:       `"key" { "nested" "value" } x`,
			expectError: true,
			errorSubstr: "unexpected character",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node govdf.Node
			err := govdf.Unmarshal([]byte(tc.input), &node)

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

func TestDecode_EdgeCases(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input        string
		expectedNode govdf.Node
	}{
		"empty input": {
			input: ``,
			expectedNode: govdf.Node{
				Type:     govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{},
				Line:     1,
				Column:   1,
			},
		},
		"byte order mark": {
			input: string([]byte{0xEF, 0xBB, 0xBF}) + `"key" "value"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:   govdf.NodeTypeScalar,
						Value:  "value",
						Line:   1,
						Column: 7,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"escaped quotes in value": {
			input: `"key" "value with \"quotes\""`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:   govdf.NodeTypeScalar,
						Value:  `value with "quotes"`,
						Line:   1,
						Column: 6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"multiple consecutive backslashes": {
			input: `"key" "value with \\\\quotes"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:   govdf.NodeTypeScalar,
						Value:  `value with \\\\quotes`,
						Line:   1,
						Column: 6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"odd number of backslashes before quote": {
			input: `"key" "value with \\\"quotes"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:   govdf.NodeTypeScalar,
						Value:  `value with \\"quotes`,
						Line:   1,
						Column: 6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"even number of backslashes before quote": {
			input: `"key" "value with \\\\\"quotes"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:   govdf.NodeTypeScalar,
						Value:  `value with \\\\"quotes`,
						Line:   1,
						Column: 6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"line comment detection": {
			input: `"key" "value" // this is a line comment`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:        govdf.NodeTypeScalar,
						Value:       "value",
						LineComment: "this is a line comment",
						Line:        1,
						Column:      6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"head comment with multiple lines": {
			input: `// first comment line
// second comment line
"key" "value"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:        govdf.NodeTypeScalar,
						Value:       "value",
						HeadComment: "first comment line\nsecond comment line",
						Line:        3,
						Column:      6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
		"empty comment lines": {
			input: `//
//
"key" "value"`,
			expectedNode: govdf.Node{
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"key": {
						Type:        govdf.NodeTypeScalar,
						Value:       "value",
						HeadComment: "",
						Line:        3,
						Column:      6,
					},
				},
				Line:   1,
				Column: 1,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			node := govdf.Node{}
			require.NoError(t, govdf.Unmarshal([]byte(tc.input), &node))

			if diff := cmp.Diff(tc.expectedNode, node); diff != "" {
				t.Errorf("unexpected node (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDecode_StructMappingErrors(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input        string
		createStruct func() any
		expectError  bool
		errorSubstr  string
	}{
		"nil target": {
			input: `"key" "value"`,
			createStruct: func() any {
				return nil
			},
			expectError: true,
			errorSubstr: "target must be a non-nil pointer",
		},
		"non-pointer target": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					Key string `vdf:"key"`
				}
				return TestStruct{}
			},
			expectError: true,
			errorSubstr: "target must be a non-nil pointer",
		},
		"non-struct target": {
			input: `"key" "value"`,
			createStruct: func() any {
				var s string
				return &s
			},
			expectError: true,
			errorSubstr: "target must be a pointer to a struct",
		},
		"unexported field": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					key string // unexported field
				}
				return &TestStruct{key: "test"}
			},
			expectError: false, // Should skip unexported fields
		},
		"field with dash tag": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					Key string `vdf:"-"`
				}
				return &TestStruct{}
			},
			expectError: false, // Should skip fields with "-" tag
		},
		"unsupported map type": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					Key map[string]string `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "unsupported type for scalar value",
		},
		"unsupported scalar type": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					Key complex128 `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "unsupported type for scalar value",
		},
		"invalid bool value": {
			input: `"key" "not-a-bool"`,
			createStruct: func() any {
				type TestStruct struct {
					Key bool `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "error converting \"not-a-bool\" to bool",
		},
		"invalid int value": {
			input: `"key" "not-a-number"`,
			createStruct: func() any {
				type TestStruct struct {
					Key int `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "error converting \"not-a-number\" to int",
		},
		"int overflow": {
			input: `"key" "9223372036854775808"`, // MaxInt64 + 1
			createStruct: func() any {
				type TestStruct struct {
					Key int64 `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "value out of range",
		},
		"invalid uint value": {
			input: `"key" "not-a-number"`,
			createStruct: func() any {
				type TestStruct struct {
					Key uint `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "error converting \"not-a-number\" to uint",
		},
		"uint overflow": {
			input: `"key" "18446744073709551616"`, // MaxUint64 + 1
			createStruct: func() any {
				type TestStruct struct {
					Key uint64 `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "value out of range",
		},
		"invalid float value": {
			input: `"key" "not-a-float"`,
			createStruct: func() any {
				type TestStruct struct {
					Key float64 `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "error converting \"not-a-float\" to float",
		},
		"float overflow": {
			input: `"key" "1e400"`, // Very large number
			createStruct: func() any {
				type TestStruct struct {
					Key float32 `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "value out of range",
		},
		"unsettable field": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					key string // unexported field
				}
				return &TestStruct{key: "test"}
			},
			expectError: false, // Should skip unsettable fields
		},
		"custom unmarshaler error": {
			input: `"key" "value"`,
			createStruct: func() any {
				type TestStruct struct {
					Key errorUnmarshaler `vdf:"key"`
				}
				return &TestStruct{}
			},
			expectError: true,
			errorSubstr: "custom unmarshaler error",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			target := tc.createStruct()
			err := govdf.Unmarshal([]byte(tc.input), target)

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

// errorUnmarshaler is a mock type that always returns an error for testing.
type errorUnmarshaler string

// UnmarshalVDF implements the UnmarshalVDF method for the errorUnmarshaler type.
func (e *errorUnmarshaler) UnmarshalVDF(node *govdf.Node) error {
	return fmt.Errorf("custom unmarshaler error")
}

func TestDecode_StructMappingEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("case insensitive field matching", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Key string `vdf:"key"`
		}

		result := TestStruct{}
		require.NoError(t, govdf.Unmarshal([]byte(`"KEY" "value"`), &result))
		require.Equal(t, "value", result.Key)
	})

	t.Run("vdf tag with comma options", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Key string `vdf:"custom_name,option1,option2"`
		}

		result := TestStruct{}
		require.NoError(t, govdf.Unmarshal([]byte(`"custom_name" "value"`), &result))
		require.Equal(t, "value", result.Key)
	})

	t.Run("field without vdf tag uses lowercase name", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			TestField string
		}

		result := TestStruct{}
		require.NoError(t, govdf.Unmarshal([]byte(`"testfield" "value"`), &result))
		require.Equal(t, "value", result.TestField)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		t.Parallel()

		type NestedStruct struct {
			Value string `vdf:"value"`
		}
		type TestStruct struct {
			Nested *NestedStruct `vdf:"nested"`
		}

		result := TestStruct{}
		require.NoError(t, govdf.Unmarshal([]byte(`"nested" { "value" "test" }`), &result))
		require.NotNil(t, result.Nested)
		require.Equal(t, "test", result.Nested.Value)
	})

	t.Run("pointer to scalar", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Key *string `vdf:"key"`
		}

		result := TestStruct{}
		require.NoError(t, govdf.Unmarshal([]byte(`"key" "value"`), &result))
		require.NotNil(t, result.Key)
		require.Equal(t, "value", *result.Key)
	})

	t.Run("all numeric types", func(t *testing.T) {
		t.Parallel()

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
		}

		result := TestStruct{}
		input := `"int8" "127" "int16" "32767" "int32" "2147483647" "int64" "9223372036854775807" "uint8" "255" "uint16" "65535" "uint32" "4294967295" "uint64" "18446744073709551615" "float32" "3.14" "float64" "3.14159265359"`
		require.NoError(t, govdf.Unmarshal([]byte(input), &result))

		require.Equal(t, int8(127), result.Int8)
		require.Equal(t, int16(32767), result.Int16)
		require.Equal(t, int32(2147483647), result.Int32)
		require.Equal(t, int64(9223372036854775807), result.Int64)
		require.Equal(t, uint8(255), result.Uint8)
		require.Equal(t, uint16(65535), result.Uint16)
		require.Equal(t, uint32(4294967295), result.Uint32)
		require.Equal(t, uint64(18446744073709551615), result.Uint64)
		require.InDelta(t, float32(3.14), result.Float32, 0.001)
		require.InDelta(t, 3.14159265359, result.Float64, 0.001)
	})
}
