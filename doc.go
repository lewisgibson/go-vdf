// Package govdf provides a comprehensive Go library for encoding and decoding VDF (Valve Data Format) files.
// VDF is a text-based data format used by Valve Software in games like Counter-Strike, Dota 2, and Team Fortress 2.
//
// The library supports:
//   - Parsing VDF files into Go structs using struct tags
//   - Converting Go structs to VDF format
//   - Working with VDF data as a tree of Node structures
//   - Preserving comments and formatting during round-trip operations
//   - Comprehensive error handling with detailed position information
//
// Basic usage:
//
//	// Parse VDF data into a struct
//	type GameInfo struct {
//		FirstValidClass string `vdf:"first_valid_class"`
//		LastValidClass  string `vdf:"last_valid_class"`
//	}
//
//	var info GameInfo
//	err := govdf.Unmarshal(vdfData, &info)
//
//	// Convert struct to VDF
//	vdfData, err := govdf.Marshal(info)
//
//	// Work with raw VDF nodes
//	var node govdf.Node
//	err := govdf.Unmarshal(vdfData, &node)
//	fmt.Println(node.Children["game_info"].Children["first_valid_class"].Value)
//
// All error types implement the error interface and can be used with standard Go error handling
// patterns including errors.Is, errors.As, and error wrapping.
package govdf
