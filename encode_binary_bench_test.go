package govdf_test

import (
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func BenchmarkMarshalBinary_Simple(b *testing.B) {
	node := &govdf.Node{
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
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := govdf.MarshalBinary(node)
		require.NoError(b, err)
	}
}

func BenchmarkMarshalBinary_Complex(b *testing.B) {
	var languages = make(map[string]*govdf.Node)
	for _, lang := range []string{"english", "german", "french", "italian", "koreana", "spanish", "schinese", "tchinese", "russian", "thai", "japanese", "portuguese", "polish", "danish", "dutch", "finnish", "norwegian", "swedish", "hungarian", "czech", "romanian", "turkish", "brazilian", "bulgarian", "greek", "ukrainian", "latam", "vietnamese", "indonesian"} {
		languages[lang] = &govdf.Node{Type: govdf.NodeTypeScalar, Value: "1"}
	}

	var depots = make(map[string]*govdf.Node)
	for _, depot := range []string{"731", "732", "733", "734", "735"} {
		depots[depot] = &govdf.Node{
			Type: govdf.NodeTypeMap,
			Children: map[string]*govdf.Node{
				"manifests": {
					Type: govdf.NodeTypeMap,
					Children: map[string]*govdf.Node{
						"public": {
							Type: govdf.NodeTypeMap,
							Children: map[string]*govdf.Node{
								"gid":  {Type: govdf.NodeTypeScalar, Value: "7908178339897559225"},
								"size": {Type: govdf.NodeTypeScalar, Value: "42949672960"},
							},
						},
					},
				},
			},
		}
	}

	node := &govdf.Node{
		Type: govdf.NodeTypeMap,
		Children: map[string]*govdf.Node{
			"appinfo": {
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"appid": {Type: govdf.NodeTypeScalar, Value: "730"},
					"common": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"name":       {Type: govdf.NodeTypeScalar, Value: "Counter-Strike 2"},
							"type":       {Type: govdf.NodeTypeScalar, Value: "Game"},
							"clienticon": {Type: govdf.NodeTypeScalar, Value: "324b323045b09bace182f928f4104dfcd93cb7f3"},
							"languages":  {Type: govdf.NodeTypeMap, Children: languages},
						},
					},
					"depots": {Type: govdf.NodeTypeMap, Children: depots},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := govdf.MarshalBinary(node)
		require.NoError(b, err)
	}
}

func BenchmarkMarshalBinary_Struct(b *testing.B) {
	type AppInfo struct {
		AppID string `vdf:"appid"`
		Name  string `vdf:"name"`
	}

	appInfo := AppInfo{
		AppID: "730",
		Name:  "Counter-Strike 2",
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := govdf.MarshalBinary(appInfo)
		require.NoError(b, err)
	}
}
