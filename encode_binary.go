package govdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/lewisgibson/go-vdf/internal"
)

// binaryBufferPool is a type-safe pool of bytes.Buffer for reuse in MarshalBinary.
var binaryBufferPool = internal.NewPool(func() *bytes.Buffer {
	return &bytes.Buffer{}
})

// getBinaryBuffer returns a bytes.Buffer from the pool.
func getBinaryBuffer() *bytes.Buffer {
	b := binaryBufferPool.Get()
	b.Reset()
	return b
}

// putBinaryBuffer returns a bytes.Buffer to the pool.
func putBinaryBuffer(b *bytes.Buffer) {
	binaryBufferPool.Put(b)
}

// MarshalBinary returns the binary VDF encoding of v.
// The input v can be a *Node or a struct with vdf struct tags.
// Binary VDF is Valve's binary serialization of the KeyValues format,
// using type-tagged fields with null-terminated strings.
//
// Example:
//
//	type AppInfo struct {
//	    AppID string `vdf:"appid"`
//	    Name  string `vdf:"name"`
//	}
//	type Root struct {
//	    AppInfo `vdf:"appinfo"`
//	}
//	root := Root{AppInfo: AppInfo{AppID: "730", Name: "Counter-Strike 2"}}
//	data, err := govdf.MarshalBinary(root)
func MarshalBinary(in any) ([]byte, error) {
	var buffer = getBinaryBuffer()
	defer putBinaryBuffer(buffer)
	if err := NewBinaryEncoder(buffer).Encode(in); err != nil {
		return nil, err
	}
	// Copy the data to avoid race condition when buffer is reused
	return append([]byte(nil), buffer.Bytes()...), nil
}

// BinaryEncoder writes binary VDF values to an output stream.
// It provides streaming encoding capabilities and is not safe for concurrent use.
type BinaryEncoder struct {
	w io.Writer
}

// NewBinaryEncoder returns a new binary VDF encoder that writes to w.
func NewBinaryEncoder(w io.Writer) *BinaryEncoder {
	return &BinaryEncoder{w: w}
}

// Encode writes the binary VDF encoding of v to the stream.
// The input v can be a *Node or a struct with vdf struct tags.
// The output uses Valve's binary type-tagged format with null-terminated strings.
func (e *BinaryEncoder) Encode(v any) error {
	if v == nil {
		return ErrNilValue
	}

	if node, ok := v.(*Node); ok {
		return e.encodeRoot(node)
	}

	var node, err = structToNode(v)
	if err != nil {
		return err
	}

	return e.encodeRoot(node)
}

// encodeRoot writes the root-level Node as a binary VDF object.
func (e *BinaryEncoder) encodeRoot(node *Node) error {
	if node == nil {
		return ErrNilNode
	}

	return e.encodeObject(node)
}

// encodeObject writes a map Node's children as binary VDF fields.
func (e *BinaryEncoder) encodeObject(node *Node) error {
	// Get sorted keys for deterministic output
	var keys = make([]string, 0, len(node.Children))
	for key := range node.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Write each key-value pair
	for _, key := range keys {
		var child = node.Children[key]
		if child == nil {
			continue
		}

		switch child.Type {
		case NodeTypeMap:
			if err := e.writeObjectTag(key); err != nil {
				return err
			}
			if err := e.encodeObject(child); err != nil {
				return err
			}

		case NodeTypeScalar:
			if err := e.writeScalar(key, child.Value); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown node type: %d", child.Type)
		}
	}

	return e.writeByte(binaryTypeEnd)
}

// writeScalar writes a scalar value with the appropriate binary VDF type tag.
// Integer values are written as int32, all others as strings.
func (e *BinaryEncoder) writeScalar(key, value string) error {
	if v, err := strconv.ParseInt(value, 10, 32); err == nil {
		return e.writeInt32(key, int32(v))
	}
	return e.writeString(key, value)
}

// writeObjectTag writes an object type tag followed by the null-terminated key.
func (e *BinaryEncoder) writeObjectTag(key string) error {
	if err := e.writeByte(binaryTypeObject); err != nil {
		return err
	}
	return e.writeNullTerminatedString(key)
}

// writeString writes a string type tag, null-terminated key, and null-terminated value.
func (e *BinaryEncoder) writeString(key, value string) error {
	if err := e.writeByte(binaryTypeString); err != nil {
		return err
	}
	if err := e.writeNullTerminatedString(key); err != nil {
		return err
	}
	return e.writeNullTerminatedString(value)
}

// writeInt32 writes an int32 type tag, null-terminated key, and little-endian int32 value.
func (e *BinaryEncoder) writeInt32(key string, value int32) error {
	if err := e.writeByte(binaryTypeInt32); err != nil {
		return err
	}
	if err := e.writeNullTerminatedString(key); err != nil {
		return err
	}
	return binary.Write(e.w, binary.LittleEndian, value)
}

// writeByte writes a single byte to the writer.
func (e *BinaryEncoder) writeByte(b byte) error {
	_, err := e.w.Write([]byte{b})
	return err
}

// writeNullTerminatedString writes a string followed by a null terminator.
func (e *BinaryEncoder) writeNullTerminatedString(s string) error {
	if _, err := io.WriteString(e.w, s); err != nil {
		return err
	}
	return e.writeByte(0x00)
}
