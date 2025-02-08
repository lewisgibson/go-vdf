package govdf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Unmarshaler is the interface implemented by types that can unmarshal a VDF description of themselves.
// Types implementing this interface can provide custom logic for converting VDF nodes into Go values.
//
// Example:
//
//	type CustomType struct {
//	    Value string
//	}
//
//	func (c *CustomType) UnmarshalVDF(node *govdf.Node) error {
//	    if node.Type != govdf.NodeTypeScalar {
//	        return fmt.Errorf("expected scalar, got %v", node.Type)
//	    }
//	    c.Value = node.Value
//	    return nil
//	}
type Unmarshaler interface {
	UnmarshalVDF(value *Node) error
}

// Unmarshal parses the VDF-encoded data and stores the result in the value pointed to by v.
// The target value v must be a pointer to a struct or a *Node.
//
// Example:
//
//	// Parse into a struct
//	type Config struct {
//	    Name string `vdf:"name"`
//	    Port int    `vdf:"port"`
//	}
//	var config Config
//	err := govdf.Unmarshal(vdfData, &config)
//
//	// Parse into a Node for manual processing
//	var node govdf.Node
//	err := govdf.Unmarshal(vdfData, &node)
func Unmarshal(in []byte, out any) error {
	return NewDecoder(bytes.NewReader(in)).Decode(out)
}

// Decoder is a VDF decoder that parses VDF data into Node structures.
// It provides streaming parsing capabilities and maintains position information
// for accurate error reporting. The decoder is not safe for concurrent use.
type Decoder struct {
	reader *bufio.Reader
	line   int
	column int

	// Reusable buffers to avoid allocations during parsing
	keyBuilder   *strings.Builder
	valueBuilder *strings.Builder
}

// NewDecoder returns a new decoder that reads from r.
// The decoder uses a buffered reader for efficient parsing of large VDF files.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		reader:       bufio.NewReaderSize(r, 4096),
		line:         1,
		column:       1,
		keyBuilder:   &strings.Builder{},
		valueBuilder: &strings.Builder{},
	}
}

// Decode reads the next VDF-encoded value from its input and stores it in the value pointed to by v.
// The target value v must be a pointer to a struct or a *Node.
// This method parses the entire VDF document from the input stream.
func (d *Decoder) Decode(v any) error {
	// Decode the VDF data into a Node struct.
	node, err := d.parse()
	if err != nil {
		return fmt.Errorf("line %d, column %d: %w", d.line, d.column, err)
	}

	// If the target is a node pointer, return the root node itself.
	if _, ok := v.(*Node); ok {
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(node).Elem())
		return nil
	}

	// For struct targets, always map the root node to the struct.
	return mapNodeToStruct(node, v)
}

// parse parses the VDF data into a Node struct.
// This is the main parsing method that processes the entire VDF document.
func (d *Decoder) parse() (*Node, error) {
	// Reset state
	d.line, d.column = 1, 1
	d.keyBuilder.Reset()
	d.valueBuilder.Reset()

	// Create root node
	root := &Node{
		Type:     NodeTypeMap,
		Line:     1,
		Column:   1,
		Children: make(map[string]*Node),
	}

	// Parse the input
	var stack = []*Node{root}
	var currentKey string
	var headComment string

	for {
		r, size, err := d.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return root, err
		}

		// Update position
		if r == '\n' {
			d.line++
			d.column = 1
		} else {
			d.column++
		}

		// Handle invalid runes
		if r == unicode.ReplacementChar && size == 1 {
			return root, newPositionError(d.line, d.column, errors.New("invalid rune"))
		}

		// Process the rune
		if err := d.processRune(r, &stack, &currentKey, &headComment); err != nil {
			return root, err
		}
	}

	return root, nil
}

// processRune processes a single rune from the VDF input.
// This method handles the state machine logic for parsing VDF syntax.
func (d *Decoder) processRune(r rune, stack *[]*Node, currentKey *string, headComment *string) error {
	switch r {
	case ' ', '\t', '\n', '\r':
		// Skip whitespace
		return nil

	case '/':
		// Handle comments
		return d.handleComment(headComment)

	case '{':
		// Start of a new map
		if len(*stack) == 0 {
			return newParseError(d.line, d.column, "unexpected '{' at root level")
		}

		var current = (*stack)[len(*stack)-1]
		if current.Children == nil {
			current.Children = make(map[string]*Node)
		}

		var newNode = &Node{
			Type:        NodeTypeMap,
			Column:      d.column - 1, // Position of the '{' character
			Line:        d.line,
			HeadComment: strings.TrimSpace(*headComment),
		}

		current.Children[*currentKey] = newNode
		*stack = append(*stack, newNode)
		*currentKey = ""
		*headComment = ""
		return nil

	case '}':
		// End of current map
		if len(*stack) <= 1 {
			return newParseError(d.line, d.column, "unexpected '}' at root level")
		}
		*stack = (*stack)[:len(*stack)-1]
		return nil

	case 65279: // Byte Order Mark
		return nil

	case '"':
		// Handle quoted strings (keys or values)
		return d.handleQuotedString(stack, currentKey, headComment)
	}

	if unicode.IsSpace(r) {
		return nil
	}

	return newParseErrorWithExpected(d.line, d.column, "unexpected character", "valid VDF character", string(r))
}

// handleComment processes comment lines starting with "//".
// Comments are preserved and attached to the next VDF element.
func (d *Decoder) handleComment(headComment *string) error {
	// Read the second '/' character to confirm this is a comment
	r, _, err := d.reader.ReadRune()
	if err != nil {
		return err
	}
	if r != '/' {
		return newParseErrorWithExpected(d.line, d.column, "expected '//' for comment", "//", "/"+string(r))
	}

	// Read the rest of the line
	line, err := d.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// Extract comment text
	if comment := strings.TrimSpace(strings.TrimRight(line, "\n\r")); comment != "" {
		if *headComment != "" {
			*headComment += "\n"
		}
		*headComment += comment
	}

	// Update position
	d.line++
	d.column = 1
	return nil
}

// handleQuotedString processes quoted strings (keys or values).
// This method determines whether the quoted string is a key or value based on context.
func (d *Decoder) handleQuotedString(stack *[]*Node, currentKey *string, headComment *string) error {
	if *currentKey == "" {
		return d.readKey(currentKey)
	}
	return d.readValue(stack, currentKey, headComment)
}

// readKey reads a quoted key from the VDF input.
// Keys are used to identify values in VDF key-value pairs.
func (d *Decoder) readKey(currentKey *string) error {
	d.keyBuilder.Reset()

	for {
		r, _, err := d.reader.ReadRune()
		if err != nil {
			return err
		}

		if r == '\n' {
			d.line++
			d.column = 1
		} else {
			d.column++
		}

		if r == '"' {
			*currentKey = d.keyBuilder.String()
			return nil
		}

		d.keyBuilder.WriteRune(r)
	}
}

// readValue reads a quoted value from the VDF input.
// Values can be scalar strings or nested VDF structures.
func (d *Decoder) readValue(stack *[]*Node, currentKey *string, headComment *string) error {
	d.valueBuilder.Reset()

	for {
		r, _, err := d.reader.ReadRune()
		if err != nil {
			return err
		}

		if r == '\n' {
			d.line++
			d.column = 1
		} else {
			d.column++
		}

		if r == '"' {
			// Check if this is an escaped quote
			if d.valueBuilder.Len() > 0 {
				value := d.valueBuilder.String()

				// Count consecutive backslashes from the end
				var backslashCount int
				for i := len(value) - 1; i >= 0 && value[i] == '\\'; i-- {
					backslashCount++
				}

				// If odd number of backslashes, this quote is escaped
				if backslashCount%2 == 1 {
					// This is an escaped quote - remove the backslash and add the quote
					currentValue := d.valueBuilder.String()
					d.valueBuilder.Reset()
					d.valueBuilder.WriteString(currentValue[:len(currentValue)-1]) // Remove the backslash
					d.valueBuilder.WriteRune(r)                                    // Add the unescaped quote
					continue
				}
			}

			// End of value - create scalar node
			value := d.valueBuilder.String()

			// Check if there's a line comment after the value
			var lineComment string
			if d.hasLineComment() {
				lineComment = d.extractLineComment()
			}

			var current = (*stack)[len(*stack)-1]
			if current.Children == nil {
				current.Children = make(map[string]*Node)
			}

			current.Children[*currentKey] = &Node{
				Type:        NodeTypeScalar,
				Value:       value,
				Column:      d.column - len(value) - 3 - strings.Count(value, "\""),
				Line:        d.line,
				HeadComment: strings.TrimSpace(*headComment),
				LineComment: lineComment,
			}

			// Reset for next key-value pair
			*currentKey = ""
			*headComment = ""
			return nil
		}

		d.valueBuilder.WriteRune(r)
	}
}

// hasLineComment checks if there's a line comment after the current value.
// Line comments appear on the same line as a VDF value.
func (d *Decoder) hasLineComment() bool {
	peeked, err := d.reader.Peek(20)
	if err != nil {
		return false
	}

	// Look for whitespace followed by //
	for i, b := range peeked {
		if unicode.IsSpace(rune(b)) {
			continue
		}
		if b == '/' && i+1 < len(peeked) && peeked[i+1] == '/' {
			return true
		}
		return false
	}

	return false
}

// extractLineComment extracts any line comment after the current value.
// This method reads and parses the comment text following a VDF value.
func (d *Decoder) extractLineComment() string {
	line, err := d.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return ""
	}

	var commentStart = strings.Index(line, "//")
	if commentStart == -1 {
		return ""
	}

	return strings.TrimSpace(line[commentStart+2:]) // +2 to skip the "//" prefix
}

// mapNodeToStruct maps the contents of a Node to a user-defined struct.
// This function uses reflection to map VDF key-value pairs to struct fields
// using the "vdf" struct tag for field name mapping.
func mapNodeToStruct(node *Node, target any) error {
	var targetValue = reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
		return newValidationError("target must be a non-nil pointer to a struct")
	}
	if targetValue = targetValue.Elem(); targetValue.Kind() != reflect.Struct {
		return newValidationError("target must be a pointer to a struct")
	}

	// Build a map of field names to field info for efficient lookup.
	var fieldMap = buildFieldMap(targetValue.Type())
	for key, child := range node.Children {
		var fieldInfo, exists = fieldMap[key]
		if !exists {
			// Try case-insensitive match.
			for vdfKey, info := range fieldMap {
				if strings.EqualFold(vdfKey, key) {
					fieldInfo = info
					exists = true
					break
				}
			}
		}
		if !exists {
			continue // Skip unknown fields.
		}

		var field = targetValue.FieldByIndex(fieldInfo.Index)
		if !field.CanSet() {
			continue
		}

		// Check if field implements Unmarshaler interface.
		if field.CanAddr() && field.Addr().Type().Implements(reflect.TypeOf((*Unmarshaler)(nil)).Elem()) {
			var unmarshaler = field.Addr().Interface().(Unmarshaler)
			if err := unmarshaler.UnmarshalVDF(child); err != nil {
				return err
			}
			continue
		}

		switch child.Type {
		case NodeTypeMap:
			if err := setMapValue(field, child); err != nil {
				return err
			}

		case NodeTypeScalar:
			if err := setScalarValue(field, child.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

// fieldInfo contains information about a struct field.
// This is used internally for efficient field mapping during struct conversion.
type fieldInfo struct {
	Index []int  // Field index for reflection access
	Tag   string // VDF tag value for field name mapping
}

// buildFieldMap creates a map of field names to field information.
// This function processes struct tags to build an efficient lookup table for field mapping.
func buildFieldMap(structType reflect.Type) map[string]fieldInfo {
	var fieldMap = make(map[string]fieldInfo)
	for i := 0; i < structType.NumField(); i++ {
		// Skip unexported fields.
		var field = structType.Field(i)
		if !field.IsExported() {
			continue
		}

		// Determine the field name.
		var fieldName string
		var vdfTag = field.Tag.Get("vdf")
		if vdfTag != "" && vdfTag != "-" {
			// Use the vdf tag value, but handle comma-separated options.
			fieldName = strings.Split(vdfTag, ",")[0]
		} else {
			// Use the field name (lowercased for consistency with VDF keys).
			fieldName = strings.ToLower(field.Name)
		}

		// Skip fields with "-" tag.
		if fieldName == "-" {
			continue
		}

		fieldMap[fieldName] = fieldInfo{
			Index: field.Index,
			Tag:   vdfTag,
		}
	}
	return fieldMap
}

// setMapValue sets a map/struct value from a Node.
// This function handles the conversion of VDF map nodes to Go struct or map types.
func setMapValue(field reflect.Value, node *Node) error {
	switch field.Kind() {
	case reflect.Struct:
		// Create a new instance of the struct type.
		if field.CanAddr() {
			return mapNodeToStruct(node, field.Addr().Interface())
		}
		return newValidationError("cannot set struct field")

	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return mapNodeToStruct(node, field.Interface())

	case reflect.Map:
		if field.IsNil() {
			field.Set(reflect.MakeMap(field.Type()))
		}

		// For maps, we need to determine the key and value types.
		// This is a simplified implementation - in practice, VDF maps are usually structs.
		return newValidationError("map type not supported for VDF decoding")

	default:
		return newValidationError(fmt.Sprintf("unsupported type for map value: %v", field.Kind()))
	}
}

// setScalarValue sets a scalar value from a string.
// This function handles the conversion of VDF scalar values to Go primitive types.
func setScalarValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return newValidationError("field cannot be set")
	}

	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setScalarValue(field.Elem(), value)

	case reflect.String:
		field.SetString(value)

	case reflect.Bool:
		var boolVal, err = strconv.ParseBool(value)
		if err != nil {
			return newTypeError("bool", value, err)
		}
		field.SetBool(boolVal)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return newTypeError("int", value, err)
		}
		if field.OverflowInt(intVal) {
			return newOverflowError("int", value)
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return newTypeError("uint", value, err)
		}
		if field.OverflowUint(uintVal) {
			return newOverflowError("uint", value)
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		var floatVal, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return newTypeError("float", value, err)
		}
		if field.OverflowFloat(floatVal) {
			return newOverflowError("float", value)
		}
		field.SetFloat(floatVal)

	default:
		return newValidationError(fmt.Sprintf("unsupported type for scalar value: %v", field.Kind()))
	}

	return nil
}
