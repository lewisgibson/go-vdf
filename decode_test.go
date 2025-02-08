package govdf_test

import (
	"embed"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	type testCase struct {
		input        string
		expectedNode govdf.Node
	}
	var testCases = map[string]testCase{
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
			input: `"a" "b"c"`,
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

								Line:   3,
								Column: 44,
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
	}
	for name, tc := range testCases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Node
			node := govdf.Node{}
			err := govdf.Unmarshal([]byte(tc.input), &node)
			require.NoErrorf(t, err, "output: %s", node)
			if diff := cmp.Diff(tc.expectedNode, node); diff != "" {
				t.Errorf("unexpected node (-want +got):\n%s", diff)
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

				vdfFile, err := fixtures.Open("fixtures/" + dirent.Name())
				require.NoError(t, err)

				vdfBytes, err := io.ReadAll(vdfFile)
				require.NoError(t, err)

				jsonFile, err := fixtures.Open("fixtures/" + strings.Replace(dirent.Name(), ".vdf", ".json", 1))
				require.NoError(t, err)

				jsonBytes, err := io.ReadAll(jsonFile)
				require.NoError(t, err)

				node := govdf.Node{}
				err = govdf.Unmarshal(vdfBytes, &node)
				require.NoError(t, err)

				vdfBytes, err = json.MarshalIndent(&node, "", "    ")
				require.NoError(t, err)

				require.JSONEq(t, string(jsonBytes), string(vdfBytes))
			})
		}
	}
}
