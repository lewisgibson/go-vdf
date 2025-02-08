package govdf

import (
	"errors"
	"fmt"
)

// Sentinel errors for common VDF operations.
// These are predefined errors that can be used for error checking with errors.Is.
var (
	// ErrNilValue is returned when attempting to encode a nil value.
	// This typically occurs when passing nil to the Marshal function.
	ErrNilValue = errors.New("cannot encode nil value")

	// ErrNilNode is returned when attempting to encode a nil Node.
	// This occurs when a Node pointer is nil during encoding operations.
	ErrNilNode = errors.New("cannot encode nil node")
)

// PositionError represents an error that occurred at a specific line and column
// in the VDF input. This is useful for debugging parsing issues as it provides
// exact location information.
//
// Example:
//
//	var posErr *PositionError
//	if errors.As(err, &posErr) {
//	    fmt.Printf("Error at line %d, column %d: %v\n", posErr.Line, posErr.Column, posErr.Err)
//	}
type PositionError struct {
	Line   int   // Line number where the error occurred (1-indexed)
	Column int   // Column number where the error occurred (1-indexed)
	Err    error // The underlying error that caused this position error
}

// Error returns a formatted error message including line and column information.
func (e *PositionError) Error() string {
	return fmt.Sprintf("line %d, column %d: %v", e.Line, e.Column, e.Err)
}

// Unwrap returns the underlying error, allowing PositionError to work with
// error wrapping and unwrapping operations.
func (e *PositionError) Unwrap() error {
	return e.Err
}

// newPositionError creates a new PositionError with the specified location and error.
// This is an internal function used by the decoder to create position-aware errors.
func newPositionError(line, column int, err error) *PositionError {
	return &PositionError{
		Line:   line,
		Column: column,
		Err:    err,
	}
}

// ParseError represents a VDF parsing error with detailed context about what was
// expected versus what was found. This is the most common error type during
// VDF parsing operations.
//
// Example:
//
//	var parseErr *ParseError
//	if errors.As(err, &parseErr) {
//	    fmt.Printf("Parse error at line %d, column %d: %s\n", parseErr.Line, parseErr.Column, parseErr.Message)
//	    if parseErr.Expected != "" {
//	        fmt.Printf("Expected: %q, Found: %q\n", parseErr.Expected, parseErr.Found)
//	    }
//	}
type ParseError struct {
	Line     int    // Line number where the parse error occurred (1-indexed)
	Column   int    // Column number where the parse error occurred (1-indexed)
	Message  string // Human-readable description of the parse error
	Expected string // What was expected at this location (may be empty)
	Found    string // What was actually found at this location (may be empty)
}

// Error returns a formatted error message. If both Expected and Found are provided,
// it includes them in the message for better debugging context.
func (e *ParseError) Error() string {
	if e.Expected != "" && e.Found != "" {
		return fmt.Sprintf("line %d, column %d: %s (expected %q, found %q)", e.Line, e.Column, e.Message, e.Expected, e.Found)
	}
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// newParseError creates a new ParseError with the specified location and message.
// This is an internal function used by the decoder for basic parse errors.
func newParseError(line, column int, message string) *ParseError {
	return &ParseError{
		Line:    line,
		Column:  column,
		Message: message,
	}
}

// newParseErrorWithExpected creates a new ParseError with expected/found information.
// This is an internal function used by the decoder for parse errors where we know
// what was expected versus what was found.
func newParseErrorWithExpected(line, column int, message, expected, found string) *ParseError {
	return &ParseError{
		Line:     line,
		Column:   column,
		Message:  message,
		Expected: expected,
		Found:    found,
	}
}

// TypeError represents an error that occurred during type conversion when mapping
// VDF data to Go structs. This includes parsing errors for numeric types, boolean
// conversion failures, and other type-related issues.
//
// Example:
//
//	var typeErr *TypeError
//	if errors.As(err, &typeErr) {
//	    fmt.Printf("Type conversion error: %s\n", typeErr.Error())
//	    fmt.Printf("Failed to convert %q to %s\n", typeErr.Value, typeErr.Type)
//	}
type TypeError struct {
	Type     string // The Go type that conversion was attempted to
	Value    string // The string value that failed to convert
	Original error  // The underlying error from the conversion attempt
}

// Error returns a formatted error message describing the type conversion failure.
func (e *TypeError) Error() string {
	return fmt.Sprintf("error converting %q to %s: %v", e.Value, e.Type, e.Original)
}

// Unwrap returns the underlying error, allowing TypeError to work with
// error wrapping and unwrapping operations.
func (e *TypeError) Unwrap() error {
	return e.Original
}

// newTypeError creates a new TypeError with the specified type, value, and underlying error.
// This is an internal function used by the decoder during struct field mapping.
func newTypeError(valueType, value string, err error) *TypeError {
	return &TypeError{
		Type:     valueType,
		Value:    value,
		Original: err,
	}
}

// OverflowError represents a numeric overflow error that occurred when a value
// is too large to fit in the target numeric type. This can happen with int,
// uint, float32, or float64 conversions.
//
// Example:
//
//	var overflowErr *OverflowError
//	if errors.As(err, &overflowErr) {
//	    fmt.Printf("Overflow error: %s\n", overflowErr.Error())
//	    fmt.Printf("Value %q overflows %s type\n", overflowErr.Value, overflowErr.Type)
//	}
type OverflowError struct {
	Type  string // The Go numeric type that the value overflows
	Value string // The string value that caused the overflow
}

// Error returns a formatted error message describing the overflow condition.
func (e *OverflowError) Error() string {
	return fmt.Sprintf("%s value %s overflows", e.Type, e.Value)
}

// newOverflowError creates a new OverflowError with the specified type and value.
// This is an internal function used by the decoder during numeric type validation.
func newOverflowError(valueType, value string) *OverflowError {
	return &OverflowError{
		Type:  valueType,
		Value: value,
	}
}

// ValidationError represents a general validation error that occurs when
// the input data or target structure doesn't meet the expected requirements.
// This includes issues like invalid target types, unsupported operations,
// and other validation failures.
//
// Example:
//
//	var validationErr *ValidationError
//	if errors.As(err, &validationErr) {
//	    fmt.Printf("Validation error: %s\n", validationErr.Message)
//	}
type ValidationError struct {
	Message string // Human-readable description of the validation failure
}

// Error returns a formatted error message describing the validation failure.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// newValidationError creates a new ValidationError with the specified message.
// This is an internal function used throughout the library for validation failures.
func newValidationError(message string) *ValidationError {
	return &ValidationError{
		Message: message,
	}
}
