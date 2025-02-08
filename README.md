# go-vdf

[![Build Workflow](https://github.com/lewisgibson/go-vdf/actions/workflows/build.yaml/badge.svg)](https://github.com/lewisgibson/go-vdf/actions/workflows/build.yaml)
[![Pkg Go Dev](https://pkg.go.dev/badge/github.com/lewisgibson/go-vdf)](https://pkg.go.dev/github.com/lewisgibson/go-vdf)

A high-performance parser and encoder for the [VDF (Valve Data Format)](https://developer.valvesoftware.com/wiki/KeyValues) in Go. VDF is commonly used in Valve games like Counter-Strike, Dota 2, and Team Fortress 2 for configuration files, item definitions, and game data.

## Features

-   ✅ **Complete VDF Support**: Parse and encode VDF files with full feature support
-   ✅ **Struct Mapping**: Direct unmarshaling to Go structs with `vdf` tags
-   ✅ **Node Tree API**: Work with VDF data as a tree of nodes
-   ✅ **Comment Preservation**: Maintains head and line comments during parsing
-   ✅ **Position Tracking**: Line and column information for error reporting
-   ✅ **Custom Marshalers**: Implement `MarshalVDF` and `UnmarshalVDF` interfaces
-   ✅ **JSON Compatibility**: Nodes can be marshaled/unmarshaled to/from JSON
-   ✅ **High Performance**: Optimized with efficient parsing and minimal allocations
-   ✅ **Concurrent Safe**: Thread-safe operations with proper synchronization
-   ✅ **Escaped Quote Support**: Properly handles escaped quotes in string values
-   ✅ **Robust Error Handling**: Detailed error messages with line/column information
-   ✅ **Clean Architecture**: Well-structured, maintainable code with comprehensive tests

## Resources

-   [Discussions](https://github.com/lewisgibson/go-vdf/discussions)
-   [Reference](https://pkg.go.dev/github.com/lewisgibson/go-vdf)
-   [Examples](https://pkg.go.dev/github.com/lewisgibson/go-vdf#pkg-examples)

## Installation

```sh
go get github.com/lewisgibson/go-vdf
```

## Quickstart

### Basic Usage

```go
package main

import (
	"fmt"
	"log"

	govdf "github.com/lewisgibson/go-vdf"
)

func main() {
	vdfData := []byte(`
"items_game"
{
    "game_info"
    {
        "first_valid_class" "2"
        "last_valid_class" "3"
        "max_num_stickers" "5"
    }
}`)

	// Parse into a Node tree
	var node govdf.Node
	if err := govdf.Unmarshal(vdfData, &node); err != nil {
		log.Fatal(err)
	}

	// Access nested values
	gameInfo := node.Children["items_game"].Children["game_info"]
	fmt.Printf("First valid class: %s\n", gameInfo.Children["first_valid_class"].Value)
}
```

### Struct Mapping

```go
type GameInfo struct {
	FirstValidClass string `vdf:"first_valid_class"`
	LastValidClass  string `vdf:"last_valid_class"`
	MaxNumStickers  int    `vdf:"max_num_stickers"`
}

type ItemsGame struct {
	GameInfo GameInfo `vdf:"game_info"`
}

// Parse directly into structs
var itemsGame ItemsGame
if err := govdf.Unmarshal(vdfData, &itemsGame); err != nil {
	log.Fatal(err)
}

fmt.Printf("Max stickers: %d\n", itemsGame.GameInfo.MaxNumStickers)
```

### Encoding to VDF

```go
// Create a Node tree
node := &govdf.Node{
	Type: govdf.NodeTypeMap,
	Children: map[string]*govdf.Node{
		"player": {
			Type: govdf.NodeTypeMap,
			Children: map[string]*govdf.Node{
				"name": {Type: govdf.NodeTypeScalar, Value: "John Doe"},
				"level": {Type: govdf.NodeTypeScalar, Value: "42"},
			},
		},
	},
}

// Encode to VDF
vdfBytes, err := govdf.Marshal(node)
if err != nil {
	log.Fatal(err)
}

fmt.Println(string(vdfBytes))
```

### Custom Marshalers

```go
type Player struct {
	Name  string
	Level int
}

func (p Player) MarshalVDF() ([]byte, error) {
	return govdf.Marshal(&govdf.Node{
		Type: govdf.NodeTypeMap,
		Children: map[string]*govdf.Node{
			"name":  {Type: govdf.NodeTypeScalar, Value: p.Name},
			"level": {Type: govdf.NodeTypeScalar, Value: fmt.Sprintf("%d", p.Level)},
		},
	})
}

func (p *Player) UnmarshalVDF(node *govdf.Node) error {
	if nameNode, ok := node.Children["name"]; ok {
		p.Name = nameNode.Value
	}
	if levelNode, ok := node.Children["level"]; ok {
		level, err := strconv.Atoi(levelNode.Value)
		if err != nil {
			return err
		}
		p.Level = level
	}
	return nil
}
```

## Performance

Benchmark results on AMD Ryzen 9 5900X:

```
BenchmarkUnmarshal_SimpleStruct-24         676,318 ops/sec    1,775 ns/op    5,296 B/op     18 allocs/op
BenchmarkUnmarshal_ComplexStruct-24        181,114 ops/sec    6,168 ns/op    8,424 B/op     71 allocs/op
BenchmarkUnmarshal_Node-24                 575,926 ops/sec    2,774 ns/op    5,816 B/op     31 allocs/op
BenchmarkMarshal_SimpleStruct-24         1,528,666 ops/sec      804 ns/op      680 B/op     26 allocs/op
BenchmarkMarshal_ComplexStruct-24          260,749 ops/sec    4,327 ns/op    3,131 B/op    135 allocs/op
BenchmarkMarshal_Node-24                 1,092,033 ops/sec    1,044 ns/op      480 B/op     52 allocs/op
```

### Running Benchmarks

Use the provided Makefile target for benchmarking:

```bash
make bench
```

## API Reference

### Core Functions

-   `Unmarshal(data []byte, v any) error` - Parse VDF data into a struct or Node
-   `Marshal(v any) ([]byte, error)` - Encode a struct or Node to VDF format
-   `NewDecoder(r io.Reader) *Decoder` - Create a streaming decoder
-   `NewEncoder(w io.Writer) *Encoder` - Create a streaming encoder

### Node Structure

```go
type Node struct {
    Type         NodeType              // NodeTypeMap or NodeTypeScalar
    Value        string                // Value for scalar nodes
    Children     map[string]*Node      // Child nodes for map nodes
    HeadComment  string                // Comment before the node
    LineComment  string                // Comment on the same line
    Line         int                   // Line number in source
    Column       int                   // Column number in source
}
```

### Interfaces

-   `Marshaler` - Implement `MarshalVDF() ([]byte, error)` for custom encoding
-   `Unmarshaler` - Implement `UnmarshalVDF(*Node) error` for custom decoding

## Examples

See the [examples directory](examples/) for more detailed usage examples.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
