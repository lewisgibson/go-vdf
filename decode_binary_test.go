package govdf_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
	"github.com/stretchr/testify/require"
)

// writeBinaryVDF helpers for building test data.
func writeObject(buf *bytes.Buffer, key string) {
	buf.WriteByte(0x00) // object tag
	buf.WriteString(key)
	buf.WriteByte(0x00) // null terminator
}

func writeString(buf *bytes.Buffer, key, value string) {
	buf.WriteByte(0x01) // string tag
	buf.WriteString(key)
	buf.WriteByte(0x00)
	buf.WriteString(value)
	buf.WriteByte(0x00)
}

func writeInt32(buf *bytes.Buffer, key string, value int32) {
	buf.WriteByte(0x02) // int32 tag
	buf.WriteString(key)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.LittleEndian, value)
}

func writeUint64(buf *bytes.Buffer, key string, value uint64) {
	buf.WriteByte(0x07) // uint64 tag
	buf.WriteString(key)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.LittleEndian, value)
}

func writeFloat32(buf *bytes.Buffer, key string, value float32) {
	buf.WriteByte(0x03) // float32 tag
	buf.WriteString(key)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.LittleEndian, value)
}

func writeInt64(buf *bytes.Buffer, key string, value int64) {
	buf.WriteByte(0x0A) // int64 tag
	buf.WriteString(key)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.LittleEndian, value)
}

func writeEnd(buf *bytes.Buffer) {
	buf.WriteByte(0x08) // end tag
}

func buildAppInfoBinaryVDF() []byte {
	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "appid", "730")
	writeString(&buf, "name", "Counter-Strike 2")
	writeEnd(&buf) // end appinfo
	writeEnd(&buf) // end root
	return buf.Bytes()
}

func TestDecodeBinary_Node(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input    func() []byte
		validate func(t *testing.T, node govdf.Node)
	}{
		"simple app info object": {
			input: buildAppInfoBinaryVDF,
			validate: func(t *testing.T, node govdf.Node) {
				require.NotNil(t, node.Children["appinfo"])
				require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
				require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["name"].Value)
			},
		},
		"nested objects": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeObject(&buf, "common")
				writeString(&buf, "name", "Counter-Strike 2")
				writeString(&buf, "type", "Game")
				writeEnd(&buf) // end common
				writeEnd(&buf) // end appinfo
				writeEnd(&buf) // end root
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.NotNil(t, node.Children["appinfo"])
				require.NotNil(t, node.Children["appinfo"].Children["common"])
				require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["common"].Children["name"].Value)
				require.Equal(t, "Game", node.Children["appinfo"].Children["common"].Children["type"].Value)
			},
		},
		"int32 value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeInt32(&buf, "reviewscore", 8)
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "8", node.Children["appinfo"].Children["reviewscore"].Value)
			},
		},
		"uint64 value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeUint64(&buf, "steamid", 76561198065346589)
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "76561198065346589", node.Children["appinfo"].Children["steamid"].Value)
			},
		},
		"float32 value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				buf.WriteByte(0x03) // float32 tag
				buf.WriteString("controller_support")
				buf.WriteByte(0x00)
				binary.Write(&buf, binary.LittleEndian, float32(3.14))
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Contains(t, node.Children["appinfo"].Children["controller_support"].Value, "3.14")
			},
		},
		"int64 value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeInt64(&buf, "last_update", -9876543210)
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "-9876543210", node.Children["appinfo"].Children["last_update"].Value)
			},
		},
		"color value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				buf.WriteByte(0x06) // color tag
				buf.WriteString("clienttga")
				buf.WriteByte(0x00)
				binary.Write(&buf, binary.LittleEndian, int32(0xFF0000))
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "16711680", node.Children["appinfo"].Children["clienttga"].Value)
			},
		},
		"pointer value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				buf.WriteByte(0x04) // pointer tag
				buf.WriteString("header_image")
				buf.WriteByte(0x00)
				binary.Write(&buf, binary.LittleEndian, int32(12345))
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "12345", node.Children["appinfo"].Children["header_image"].Value)
			},
		},
		"wstring value": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				buf.WriteByte(0x05) // wstring tag
				buf.WriteString("localized_name")
				buf.WriteByte(0x00)
				buf.WriteString("Counter-Strike 2")
				buf.WriteByte(0x00)
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "Counter-Strike 2", node.Children["appinfo"].Children["localized_name"].Value)
			},
		},
		"empty object": {
			input: func() []byte {
				var buf bytes.Buffer
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Empty(t, node.Children)
			},
		},
		"empty input": {
			input: func() []byte { return []byte{} },
			validate: func(t *testing.T, node govdf.Node) {
				require.Empty(t, node.Children)
			},
		},
		"multiple root objects": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeString(&buf, "appid", "730")
				writeEnd(&buf)
				writeObject(&buf, "packageinfo")
				writeString(&buf, "packaged", "303386")
				writeEnd(&buf)
				writeEnd(&buf)
				return buf.Bytes()
			},
			validate: func(t *testing.T, node govdf.Node) {
				require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
				require.Equal(t, "303386", node.Children["packageinfo"].Children["packaged"].Value)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node govdf.Node
			err := govdf.UnmarshalBinary(tc.input(), &node)
			require.NoError(t, err)
			tc.validate(t, node)
		})
	}
}

func TestDecodeBinary_ErrorHandling(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		input       func() []byte
		expectedErr string
	}{
		"invalid tag at root": {
			input: func() []byte {
				var buf bytes.Buffer
				buf.WriteByte(0xFF)
				return buf.Bytes()
			},
			expectedErr: "expected object tag",
		},
		"unknown type tag inside object": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				buf.WriteByte(0x0B)
				buf.WriteString("key")
				buf.WriteByte(0x00)
				return buf.Bytes()
			},
			expectedErr: "unknown binary VDF tag",
		},
		"truncated data": {
			input: func() []byte {
				var buf bytes.Buffer
				writeObject(&buf, "appinfo")
				writeString(&buf, "key", "value")
				// missing writeEnd
				return buf.Bytes()
			},
			expectedErr: "failed to read tag",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node govdf.Node
			err := govdf.UnmarshalBinary(tc.input(), &node)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestDecodeBinary_Struct(t *testing.T) {
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

	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "appid", "730")
	writeObject(&buf, "common")
	writeString(&buf, "name", "Counter-Strike 2")
	writeString(&buf, "type", "Game")
	writeEnd(&buf) // end common
	writeEnd(&buf) // end appinfo
	writeEnd(&buf) // end root

	var root Root
	err := govdf.UnmarshalBinary(buf.Bytes(), &root)
	require.NoError(t, err)
	require.Equal(t, "730", root.AppInfo.AppID)
	require.Equal(t, "Counter-Strike 2", root.AppInfo.Common.Name)
	require.Equal(t, "Game", root.AppInfo.Common.Type)
}

func TestDecodeBinary_Decoder(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writeObject(&buf, "appinfo")
	writeString(&buf, "appid", "730")
	writeEnd(&buf)
	writeEnd(&buf)

	decoder := govdf.NewBinaryDecoder(&buf)
	var node govdf.Node
	err := decoder.Decode(&node)
	require.NoError(t, err)
	require.Equal(t, "730", node.Children["appinfo"].Children["appid"].Value)
}
