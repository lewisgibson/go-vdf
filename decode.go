package govdf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Unmarshaler is the interface implemented by types that can unmarshal a VDF description of themselves.
type Unmarshaler interface {
	UnmarshalVDF(value *Node) error
}

// Unmarshal parses the VDF-encoded data and stores the result in the value pointed to by v.
func Unmarshal(in []byte, out any) error {
	return NewDecoder(bytes.NewReader(in)).Decode(out)
}

// Decoder reads and decodes VDF values from an input stream.
type Decoder struct {
	reader *bufio.Reader
	line   int
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		// The reader must have a size of 4 to support peeking utf8 runes.
		reader: bufio.NewReaderSize(r, 4),
	}
}

// Decode reads the next VDF-encoded value from its input and stores it in the value pointed to by v.
func (d *Decoder) Decode(v any) error {
	// Decode the VDF data into a Node struct.
	node, err := d.parse()
	if err != nil {
		return fmt.Errorf("line %d: %w", d.line, err)
	}

	// If the target is a node pointer, return the root node itself.
	if _, ok := v.(*Node); ok && v != nil {
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(node).Elem())
		return nil
	}

	// Else, map the node to the target struct.
	return mapNodeToStruct(node, v)
}

// parse parses the VDF data into a Node struct.
func (d *Decoder) parse() (*Node, error) {
	// root is the top-level map.
	var root = &Node{
		Type:   NodeTypeMap,
		Line:   1,
		Column: 1,
	}

	// stack is a stack of maps that are being parsed.
	var stack = []*Node{root}

	// isReadingKey is a boolean that indicates if the parser is reading a key.
	var isReadingKey bool
	// key is the key that is being read.
	var key string

	// isReadingValue is a boolean that indicates if the parser is reading a value.
	var isReadingValue bool
	// value is the value that is being read.
	var value string

	// headComment is the comment above the node.
	var headComment string

	// Iterate over each rune in the scanner.
	var line, column = 1, 1
	for {
		r, size, err := d.reader.ReadRune()
		switch {
		case r == unicode.ReplacementChar && size == 1:
			return root, fmt.Errorf("invalid rune: %v", r)

		case errors.Is(err, io.EOF):
			return root, nil

		case err != nil:
			return root, err
		}

		switch {
		// Start of a new map.
		case !isReadingKey && !isReadingValue && r == '{':
			// current is the current map in the stack.
			var current = stack[len(stack)-1]
			if current.Children == nil {
				current.Children = make(map[string]*Node)
			}

			current.Children[key] = &Node{
				Type:   NodeTypeMap,
				Column: column,
				Line:   line,
			}

			// Add it to the stack to be picked up for values, and then reset
			stack = append(stack, current.Children[key])

			// Reset the key and value.
			key = ""
			isReadingKey = false

			// Increment the column.
			column++

		// End of the current map.
		case !isReadingKey && !isReadingValue && r == '}':
			// Pop the current map off the stack.
			stack = stack[:len(stack)-1]
			column++

			// Skip Byte Order Mark
		case !isReadingKey && !isReadingValue && r == 65279:
			column++

		// Skip Comments
		case !isReadingKey && !isReadingValue && r == '/':
			str, err := d.reader.ReadString('\n')
			switch {
			case errors.Is(err, io.EOF):
				return root, nil

			case err != nil:
				return root, err
			}

			if comment := strings.TrimSpace(strings.TrimPrefix(str, "/")); comment != "" {
				headComment += comment + "\n"
			}

			line++
			column = 1

		// Skip column whitespace.
		case !isReadingKey && !isReadingValue && (r == ' ' || r == '\t'):
			column++

			// Skip line whitespace.
		case !isReadingKey && !isReadingValue && unicode.IsSpace(r):
			line++
			column = 1

			// Start Reading
		case !isReadingKey && !isReadingValue && r == '"':
			if len(key) == 0 {
				isReadingKey = true
			} else {
				isReadingValue = true
			}
			column++

			// End Reading Key
		case isReadingKey && r == '"':
			isReadingKey = false
			column++

			// Read Key
		case isReadingKey:
			key += strings.ToLower(string(r))
			column++

			// Read Value
		case isReadingValue:
			// Check if its the end
			if r == '"' {
				isEnd, err := isValueEnd(d.reader)
				switch {
				case err != nil:
					return root, err

				case isEnd:
					// Consume the rest of the line until a newline.
					rest, err := d.reader.ReadString('\n')
					if err != nil && !errors.Is(err, io.EOF) {
						return root, err
					}

					// Get the current map in the stack.
					var current = stack[len(stack)-1]
					if current.Children == nil {
						current.Children = make(map[string]*Node)
					}

					current.Children[key] = &Node{
						Type:        NodeTypeScalar,
						Value:       value,
						Column:      column - len(value) - 2, // The column is the starting column of the value.
						Line:        line,
						HeadComment: strings.TrimSpace(headComment),
						LineComment: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rest), "//")),
					}

					// Reset the key and value.
					key = ""
					value = ""
					headComment = ""
					isReadingValue = false

					// Move to the next line.
					line++
					column = 1
					continue
				}
			}

			value += string(r)
			column++

		default:
			return root, fmt.Errorf("unexpected rune: %v", r)
		}
	}
}

// isValueEnd checks if the value has ended.
// It does this by peeking all of the next runes, skipping comments and whitespace, until a newline, closing bracket, or EOF is found.
func isValueEnd(reader *bufio.Reader) (bool, error) {
	// If a newline or closing bracket is found, the value has ended.
	r, err := peekRune(reader, 0)
	switch {
	case errors.Is(err, io.EOF):
		return true, nil

	case err != nil:
		return false, err

	case r == '\n' || r == '}':
		return true, nil
	}

	// Else, it must be a comment.
	var offset int
	for ; ; offset += 4 {
		r, err := peekRune(reader, offset)
		switch {
		case errors.Is(err, io.EOF):
			return true, nil

		case err != nil:
			return false, err

			// A comment can have an arbitrary number of spaces before it.
		case unicode.IsSpace(r):
			continue

			// A comment must start with a forward slash.
		case r == '/':
			return true, nil

		default:
			return false, nil
		}
	}
}

// peekRune peeks the next rune at the given offset.
func peekRune(reader *bufio.Reader, offset int) (rune, error) {
	for size := 4; size > 0; size-- {
		b, err := reader.Peek(size + 4*offset)
		if err == nil {
			r, _ := utf8.DecodeRune(b[4*offset:])
			if r == utf8.RuneError {
				return r, fmt.Errorf("invalid rune: %v", b)
			}
			return r, nil
		}
	}
	return -1, io.EOF
}

// mapNodeToStruct maps the contents of a Node to a user-defined struct.
func mapNodeToStruct(node *Node, target any) error {
	var targetValue = reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to a struct")
	}
	targetValue = targetValue.Elem()

	for key, child := range node.Children {
		var field = targetValue.FieldByName(strings.Title(key))
		if field.IsValid() && field.CanSet() {
			switch child.Type {
			case NodeTypeMap:
				var nestedStruct = reflect.New(field.Type()).Interface()
				if err := mapNodeToStruct(child, nestedStruct); err != nil {
					return err
				}
				field.Set(reflect.ValueOf(nestedStruct).Elem())

			case NodeTypeScalar:
				field.SetString(child.Value)
			}
		}
	}

	return nil
}
