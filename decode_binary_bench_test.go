package govdf_test

import (
	"bytes"
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

func buildSimpleBinaryVDF() []byte {
	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "appid", "730")
	writeString(&buf, "name", "Counter-Strike 2")
	writeInt32(&buf, "type", 1)
	writeEnd(&buf)
	writeEnd(&buf)
	return buf.Bytes()
}

func buildComplexBinaryVDF() []byte {
	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "appid", "730")
	writeObject(&buf, "common")
	writeString(&buf, "name", "Counter-Strike 2")
	writeString(&buf, "type", "Game")
	writeString(&buf, "clienticon", "324b323045b09bace182f928f4104dfcd93cb7f3")
	writeInt32(&buf, "reviewscore", 8)
	writeUint64(&buf, "steam_deck_compat_tested", 1700000000)
	writeObject(&buf, "languages")
	for _, lang := range []string{"english", "german", "french", "italian", "koreana", "spanish", "schinese", "tchinese", "russian", "thai", "japanese", "portuguese", "polish", "danish", "dutch", "finnish", "norwegian", "swedish", "hungarian", "czech", "romanian", "turkish", "brazilian", "bulgarian", "greek", "ukrainian", "latam", "vietnamese", "indonesian"} {
		writeString(&buf, lang, "1")
	}
	writeEnd(&buf) // languages
	writeEnd(&buf) // common
	writeObject(&buf, "depots")
	for _, depot := range []string{"731", "732", "733", "734", "735"} {
		writeObject(&buf, depot)
		writeObject(&buf, "manifests")
		writeObject(&buf, "public")
		writeString(&buf, "gid", "7908178339897559225")
		writeString(&buf, "size", "42949672960")
		writeEnd(&buf) // public
		writeEnd(&buf) // manifests
		writeEnd(&buf) // depot
	}
	writeEnd(&buf) // depots
	writeEnd(&buf) // appinfo
	writeEnd(&buf) // root
	return buf.Bytes()
}

func BenchmarkUnmarshalBinary_Simple(b *testing.B) {
	data := buildSimpleBinaryVDF()
	b.ResetTimer()
	for b.Loop() {
		var node govdf.Node
		require.NoError(b, govdf.UnmarshalBinary(data, &node))
	}
}

func BenchmarkUnmarshalBinary_Complex(b *testing.B) {
	data := buildComplexBinaryVDF()
	b.ResetTimer()
	for b.Loop() {
		var node govdf.Node
		require.NoError(b, govdf.UnmarshalBinary(data, &node))
	}
}

func BenchmarkUnmarshalBinary_Struct(b *testing.B) {
	data := buildSimpleBinaryVDF()

	type AppInfo struct {
		AppID string `vdf:"appid"`
		Name  string `vdf:"name"`
	}
	type Root struct {
		AppInfo AppInfo `vdf:"appinfo"`
	}

	b.ResetTimer()
	for b.Loop() {
		var root Root
		require.NoError(b, govdf.UnmarshalBinary(data, &root))
	}
}

func BenchmarkUnmarshalBinary_AllTypes(b *testing.B) {
	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "name", "Counter-Strike 2")
	writeInt32(&buf, "reviewscore", 8)
	writeFloat32(&buf, "controller_support", 3.14)
	writeUint64(&buf, "steamid", 76561198065346589)
	writeInt64(&buf, "last_update", -12345)
	writeEnd(&buf)
	writeEnd(&buf)
	data := buf.Bytes()

	b.ResetTimer()
	for b.Loop() {
		var node govdf.Node
		require.NoError(b, govdf.UnmarshalBinary(data, &node))
	}
}
