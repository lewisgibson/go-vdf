package govdf

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDuplicateKeysNestedMaps(t *testing.T) {
	var node Node
	require.NoError(t, Unmarshal([]byte(`"item" { "name" "first" }
"item" { "name" "second" }
`), &node))
	require.Len(t, node.Children["item"], 2)
	require.Equal(t, "first", node.Children["item"][0].Children["name"][0].Value)
	require.Equal(t, "second", node.Children["item"][1].Children["name"][0].Value)
}

func TestDuplicateNumericKeysIntoSlice(t *testing.T) {
	type config struct {
		Ports []int `vdf:"port"`
	}
	var got config
	require.NoError(t, Unmarshal([]byte(`"port" "80"
"port" "443"
`), &got))
	require.Equal(t, []int{80, 443}, got.Ports)
}

func TestDuplicateKeysJSONRoundTrip(t *testing.T) {
	var node Node
	require.NoError(t, Unmarshal([]byte(`"tag" "one"
"tag" "two"
`), &node))
	b, err := json.Marshal(node)
	require.NoError(t, err)
	var raw map[string][]string
	require.NoError(t, json.Unmarshal(b, &raw))
	require.Equal(t, []string{"one", "two"}, raw["tag"])
}

func TestDuplicateKeysEncodeDecodeRoundTrip(t *testing.T) {
	type config struct {
		Tags []string `vdf:"tag"`
	}
	want := config{Tags: []string{"one", "two", "three"}}
	b, err := Marshal(want)
	require.NoError(t, err)
	var got config
	require.NoError(t, Unmarshal(b, &got))
	require.Equal(t, want, got)
}

func TestDuplicateKeysStreamingDecoder(t *testing.T) {
	dec := NewDecoder(bytes.NewBufferString(`"x" "a"
"x" "b"
`))
	var node Node
	require.NoError(t, dec.Decode(&node))
	require.Equal(t, []string{"a", "b"}, []string{node.Children["x"][0].Value, node.Children["x"][1].Value})
}

func TestDuplicateScalarKeys(t *testing.T) {
	input := []byte(`"tag" "one"
"tag" "two"
"tag" "three"
`)

	var node Node
	require.NoError(t, Unmarshal(input, &node))
	require.Len(t, node.Children["tag"], 3)
	require.Equal(t, "one", node.Children["tag"][0].Value)
	require.Equal(t, "two", node.Children["tag"][1].Value)
	require.Equal(t, "three", node.Children["tag"][2].Value)
}

func TestDuplicateKeysIntoSlice(t *testing.T) {
	type config struct {
		Tags []string `vdf:"tag"`
	}

	var got config
	require.NoError(t, Unmarshal([]byte(`"tag" "one"
"tag" "two"
`), &got))
	require.Equal(t, []string{"one", "two"}, got.Tags)
}

func TestDuplicateKeysMarshal(t *testing.T) {
	type config struct {
		Tags []string `vdf:"tag"`
	}

	data, err := Marshal(config{Tags: []string{"one", "two"}})
	require.NoError(t, err)
	require.Equal(t, "\"tag\" \"one\"\n\"tag\" \"two\"\n", string(data))
}
