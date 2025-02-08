package govdf

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/lewisgibson/go-vdf/internal"
)

// bufferPool is a type-safe pool of bytes.Buffer for reuse in Marshal.
var bufferPool = internal.NewPool(func() *bytes.Buffer {
	return &bytes.Buffer{}
})

// getBuffer returns a bytes.Buffer from the pool.
func getBuffer() *bytes.Buffer {
	b := bufferPool.Get()
	b.Reset()
	return b
}

// putBuffer returns a bytes.Buffer to the pool.
func putBuffer(b *bytes.Buffer) {
	bufferPool.Put(b)
}

// Marshaler is the interface implemented by types that can marshal themselves into a VDF description.
// Types implementing this interface can provide custom logic for converting Go values into VDF format.
//
// Example:
//
//	type CustomType struct {
//	    Value string
//	}
//
//	func (c CustomType) MarshalVDF() ([]byte, error) {
//	    return govdf.Marshal(map[string]string{"value": c.Value})
//	}
type Marshaler interface {
	MarshalVDF() ([]byte, error)
}

// Marshal returns the VDF encoding of v.
// The input v can be a struct, a *Node, or any type implementing Marshaler.
// Struct fields are mapped to VDF keys using the "vdf" struct tag.
//
// Example:
//
//	type Config struct {
//	    Name string `vdf:"name"`
//	    Port int    `vdf:"port"`
//	}
//	config := Config{Name: "server", Port: 8080}
//	vdfData, err := govdf.Marshal(config)
func Marshal(in any) ([]byte, error) {
	var buffer = getBuffer()
	defer putBuffer(buffer)
	if err := NewEncoder(buffer).Encode(in); err != nil {
		return nil, err
	}
	// Copy the data to avoid race condition when buffer is reused
	return append([]byte(nil), buffer.Bytes()...), nil
}

// Encoder writes VDF values to an output stream.
// It provides streaming encoding capabilities and is not safe for concurrent use.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
// The encoder will write properly formatted VDF data to the provided writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Encode writes the VDF encoding of v to the stream.
// The input v can be a struct, a *Node, or any type implementing Marshaler.
// The output is properly formatted with indentation and preserved comments.
func (e *Encoder) Encode(v any) error {
	if v == nil {
		return ErrNilValue
	}

	// Handle Node types directly
	if node, ok := v.(*Node); ok {
		return e.encodeNode(node, 0)
	}

	// Handle custom Marshaler interface
	if marshaler, ok := v.(Marshaler); ok {
		data, err := marshaler.MarshalVDF()
		if err != nil {
			return err
		}
		if _, err = e.w.Write(data); err != nil {
			return err
		}
		return nil
	}

	// Handle structs by converting to Node first
	var node, err = structToNode(v)
	if err != nil {
		return err
	}

	return e.encodeNode(node, 0)
}

// encodeNode writes a Node to the output stream with proper indentation.
// This is an internal method that handles the actual VDF formatting and output.
func (e *Encoder) encodeNode(node *Node, indent int) error {
	if node == nil {
		return ErrNilNode
	}

	switch node.Type {
	case NodeTypeMap:
		return e.encodeMap(node, indent)

	case NodeTypeScalar:
		return e.encodeScalar(node)

	default:
		return newValidationError(fmt.Sprintf("unknown node type: %d", node.Type))
	}
}

// encodeMap writes a map node to the output stream.
// This method handles the formatting of VDF key-value pairs and nested structures.
func (e *Encoder) encodeMap(node *Node, indent int) error {
	if len(node.Children) == 0 {
		return nil
	}

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

		// Write head comment if present for this child
		if child.HeadComment != "" {
			if err := e.writeHeadComment(child.HeadComment, indent); err != nil {
				return err
			}
		}

		// Write the key
		if err := e.writeIndent(indent); err != nil {
			return err
		}
		if err := e.writeQuotedString(key); err != nil {
			return err
		}

		// Write the value based on its type
		switch child.Type {
		case NodeTypeMap:
			// Write opening brace
			if _, err := e.w.Write([]byte(" {\n")); err != nil {
				return err
			}
			// Write map contents
			if err := e.encodeMap(child, indent+1); err != nil {
				return err
			}
			// Write closing brace
			if err := e.writeIndent(indent); err != nil {
				return err
			}
			if _, err := e.w.Write([]byte("}\n")); err != nil {
				return err
			}

		case NodeTypeScalar:
			// Write space before value
			if _, err := e.w.Write([]byte(" ")); err != nil {
				return err
			}
			// Write scalar value
			if err := e.writeQuotedString(child.Value); err != nil {
				return err
			}
			// Write line comment if present
			if child.LineComment != "" {
				if _, err := e.w.Write([]byte("\t// " + child.LineComment)); err != nil {
					return err
				}
			}
			if _, err := e.w.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}

	return nil
}

// encodeScalar writes a scalar node to the output stream.
// This method handles the formatting of VDF scalar values with proper quoting.
func (e *Encoder) encodeScalar(node *Node) error {
	// Write head comment if present
	if node.HeadComment != "" {
		if err := e.writeHeadComment(node.HeadComment, 0); err != nil {
			return err
		}
	}

	// Write the value
	if err := e.writeQuotedString(node.Value); err != nil {
		return err
	}

	// Write line comment if present
	if node.LineComment != "" {
		if _, err := e.w.Write([]byte("\t// " + node.LineComment)); err != nil {
			return err
		}
	}

	// Add newline for scalar nodes
	if _, err := e.w.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// writeQuotedString writes a string with proper VDF quoting and escaping.
// VDF strings are enclosed in double quotes and handle escaping internally.
func (e *Encoder) writeQuotedString(s string) error {
	if _, err := e.w.Write([]byte(`"`)); err != nil {
		return err
	}

	// In VDF, quotes inside strings are not escaped - they're included as-is
	// The only escaping is when a quote appears at the end of a value
	// but is not actually the end of the value (which is handled by the parser)
	if _, err := e.w.Write([]byte(s)); err != nil {
		return err
	}

	if _, err := e.w.Write([]byte(`"`)); err != nil {
		return err
	}

	return nil
}

// writeIndent writes the appropriate indentation spaces.
// VDF uses 4 spaces per indentation level for consistent formatting.
func (e *Encoder) writeIndent(indent int) error {
	var spaces = strings.Repeat("    ", indent) // 4 spaces per indent level
	if _, err := e.w.Write([]byte(spaces)); err != nil {
		return err
	}
	return nil
}

// writeHeadComment writes a head comment with proper indentation.
// Head comments appear before a VDF key-value pair and are preserved during encoding.
func (e *Encoder) writeHeadComment(comment string, indent int) error {
	var lines = strings.Split(strings.TrimSpace(comment), "\n")
	for _, line := range lines {
		if err := e.writeIndent(indent); err != nil {
			return err
		}
		if _, err := e.w.Write([]byte("// " + line + "\n")); err != nil {
			return err
		}
	}
	return nil
}

// structToNode converts a struct to a Node for encoding.
// This function recursively converts Go struct fields to VDF nodes using struct tags.
func structToNode(v any) (*Node, error) {
	var val = reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil // Skip nil pointer structs (optional fields)
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, newValidationError(fmt.Sprintf("expected struct, got %v", val.Kind()))
	}

	var node = &Node{
		Type:     NodeTypeMap,
		Children: make(map[string]*Node),
	}

	// Process each field
	for i := 0; i < val.NumField(); i++ {
		// Skip unexported fields
		var fieldType = val.Type().Field(i)
		if !fieldType.IsExported() {
			continue
		}

		// Determine the field name
		var fieldName string
		if vdfTag := fieldType.Tag.Get("vdf"); vdfTag != "" && vdfTag != "-" {
			fieldName = strings.Split(vdfTag, ",")[0]
		} else {
			fieldName = strings.ToLower(fieldType.Name)
		}

		// Skip fields with "-" tag
		if fieldName == "-" {
			continue
		}

		// Convert field value to node
		childNode, err := valueToNode(val.Field(i))
		switch {
		case err != nil:
			return nil, err

		case childNode != nil:
			node.Children[fieldName] = childNode
		}
	}

	return node, nil
}

// valueToNode converts a reflect.Value to a Node.
// This function handles the conversion of Go values to VDF nodes, including
// custom Marshaler implementations and basic type conversions.
func valueToNode(val reflect.Value) (*Node, error) {
	// Handle nil pointers
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil, nil
	}

	// Dereference pointers
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check for custom Marshaler interface
	if val.CanAddr() && val.Addr().Type().Implements(reflect.TypeOf((*Marshaler)(nil)).Elem()) {
		marshaler := val.Addr().Interface().(Marshaler)
		data, err := marshaler.MarshalVDF()
		if err != nil {
			return nil, err
		}

		// Parse the marshaled data back into a node
		var tempNode Node
		if err := Unmarshal(data, &tempNode); err != nil {
			return nil, err
		}

		return &tempNode, nil
	}

	switch val.Kind() {
	case reflect.Struct:
		return structToNode(val.Interface())

	case reflect.String:
		return &Node{
			Type:  NodeTypeScalar,
			Value: val.String(),
		}, nil

	case reflect.Bool:
		return &Node{
			Type:  NodeTypeScalar,
			Value: strconv.FormatBool(val.Bool()),
		}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Node{
			Type:  NodeTypeScalar,
			Value: strconv.FormatInt(val.Int(), 10),
		}, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Node{
			Type:  NodeTypeScalar,
			Value: strconv.FormatUint(val.Uint(), 10),
		}, nil

	case reflect.Float32, reflect.Float64:
		return &Node{
			Type:  NodeTypeScalar,
			Value: strconv.FormatFloat(val.Float(), 'g', -1, 64),
		}, nil

	default:
		return nil, newValidationError(fmt.Sprintf("unsupported type for encoding: %v", val.Kind()))
	}
}
