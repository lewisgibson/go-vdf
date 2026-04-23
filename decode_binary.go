package govdf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// Binary VDF type tags.
const (
	binaryTypeObject  byte = 0x00
	binaryTypeString  byte = 0x01
	binaryTypeInt32   byte = 0x02
	binaryTypeFloat32 byte = 0x03
	binaryTypePointer byte = 0x04
	binaryTypeWString byte = 0x05
	binaryTypeColor   byte = 0x06
	binaryTypeUint64  byte = 0x07
	binaryTypeEnd     byte = 0x08
	binaryTypeInt64   byte = 0x0A
)

// UnmarshalBinary parses binary VDF-encoded data and stores the result
// in the value pointed to by v. Binary VDF is Valve's binary serialization
// of the KeyValues format, using type-tagged fields with null-terminated strings.
func UnmarshalBinary(in []byte, out any) error {
	return NewBinaryDecoder(bytes.NewReader(in)).Decode(out)
}

// BinaryDecoder decodes binary VDF data into Node structures.
type BinaryDecoder struct {
	reader *bufio.Reader
	buf    bytes.Buffer
}

// NewBinaryDecoder returns a new binary VDF decoder that reads from r.
func NewBinaryDecoder(r io.Reader) *BinaryDecoder {
	return &BinaryDecoder{reader: bufio.NewReader(r)}
}

// Decode reads the binary VDF-encoded value and stores it in v.
// The target value v must be a pointer to a *Node or a struct.
func (d *BinaryDecoder) Decode(v any) error {
	node, err := d.parseRoot()
	if err != nil {
		return err
	}

	if _, ok := v.(*Node); ok {
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(node).Elem())
		return nil
	}

	return mapNodeToStruct(node, v)
}

// parseRoot reads the top-level binary VDF object.
func (d *BinaryDecoder) parseRoot() (*Node, error) {
	var root = &Node{
		Type:     NodeTypeMap,
		Children: make(map[string]*Node),
	}
	for {
		tag, err := d.readByte()
		switch {
		case errors.Is(err, io.EOF):
			return root, nil

		case err != nil:
			return nil, fmt.Errorf("failed to read tag: %w", err)

		case tag == binaryTypeEnd:
			return root, nil

		case tag != binaryTypeObject:
			return nil, fmt.Errorf("expected object tag (0x00) at root, got 0x%02X", tag)
		}

		key, err := d.readNullTerminatedString()
		if err != nil {
			return nil, fmt.Errorf("failed to read root key: %w", err)
		}

		child, err := d.parseObject()
		if err != nil {
			return nil, fmt.Errorf("failed to parse root object %q: %w", key, err)
		}

		root.Children[key] = child
	}
}

// parseObject reads an object's children until the end tag (0x08).
func (d *BinaryDecoder) parseObject() (*Node, error) {
	var node = &Node{
		Type:     NodeTypeMap,
		Children: make(map[string]*Node),
	}
	for {
		tag, err := d.readByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read tag: %w", err)
		}

		if tag == binaryTypeEnd {
			return node, nil
		}

		key, err := d.readNullTerminatedString()
		if err != nil {
			return nil, fmt.Errorf("failed to read key: %w", err)
		}

		switch tag {
		case binaryTypeObject:
			child, err := d.parseObject()
			if err != nil {
				return nil, fmt.Errorf("failed to parse object %q: %w", key, err)
			}
			node.Children[key] = child

		case binaryTypeString, binaryTypeWString:
			value, err := d.readNullTerminatedString()
			if err != nil {
				return nil, fmt.Errorf("failed to read string value for %q: %w", key, err)
			}
			node.Children[key] = &Node{Type: NodeTypeScalar, Value: value}

		case binaryTypeInt32, binaryTypeColor, binaryTypePointer:
			var v int32
			if err := binary.Read(d.reader, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("failed to read int32 value for %q: %w", key, err)
			}
			node.Children[key] = &Node{Type: NodeTypeScalar, Value: strconv.Itoa(int(v))}

		case binaryTypeFloat32:
			var v float32
			if err := binary.Read(d.reader, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("failed to read float32 value for %q: %w", key, err)
			}
			node.Children[key] = &Node{Type: NodeTypeScalar, Value: fmt.Sprintf("%g", v)}

		case binaryTypeUint64:
			var v uint64
			if err := binary.Read(d.reader, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("failed to read uint64 value for %q: %w", key, err)
			}
			node.Children[key] = &Node{Type: NodeTypeScalar, Value: strconv.FormatUint(v, 10)}

		case binaryTypeInt64:
			var v int64
			if err := binary.Read(d.reader, binary.LittleEndian, &v); err != nil {
				return nil, fmt.Errorf("failed to read int64 value for %q: %w", key, err)
			}
			node.Children[key] = &Node{Type: NodeTypeScalar, Value: strconv.FormatInt(v, 10)}

		default:
			return nil, fmt.Errorf("unknown binary VDF tag 0x%02X for key %q", tag, key)
		}
	}
}

// readByte reads a single byte from the reader.
func (d *BinaryDecoder) readByte() (byte, error) {
	return d.reader.ReadByte()
}

// readNullTerminatedString reads bytes until a null terminator (0x00).
func (d *BinaryDecoder) readNullTerminatedString() (string, error) {
	d.buf.Reset()
	for {
		b, err := d.readByte()
		switch {
		case err != nil:
			return "", err

		case b == 0x00:
			return d.buf.String(), nil
		}
		d.buf.WriteByte(b)
	}
}
