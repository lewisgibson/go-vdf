# go-vdf

[![Build Workflow](https://github.com/lewisgibson/go-vdf/actions/workflows/build.yaml/badge.svg)](https://github.com/lewisgibson/go-vdf/actions/workflows/build.yaml)
[![Pkg Go Dev](https://pkg.go.dev/badge/github.com/lewisgibson/go-vdf)](https://pkg.go.dev/github.com/lewisgibson/go-vdf)

A parser for the [vdf](https://developer.valvesoftware.com/wiki/KeyValues) format in go.

## Resources

-   [Discussions](https://github.com/lewisgibson/go-vdf/discussions)
-   [Reference](https://pkg.go.dev/github.com/lewisgibson/go-vdf)
-   [Examples](https://pkg.go.dev/github.com/lewisgibson/go-vdf#pkg-examples)

## Installation

```sh
go get github.com/lewisgibson/go-vdf
```

## Quickstart

```go
import (
	"fmt"
	"strings"

	govdf "github.com/lewisgibson/go-vdf"
)

// Unmarshal the vdf bytes into a node.
var node govdf.Node
if err := govdf.Unmarshal(vdfBytes, &node); err != nil {
    panic(err)
}

// Print the root node and it's children.
type traversal struct {
    keys []string
    node *govdf.Node
}
var current *traversal
var stack = []*traversal{{keys: []string{""}, node: &node}}
for len(stack) != 0 {
    // Pop the last node from the stack.
    current, stack = stack[len(stack)-1], stack[:len(stack)-1]

    // Print the current node.
    if current.node.Type == govdf.NodeTypeMap {
        fmt.Printf("Map node at line %d, column %d\n", current.node.Line, current.node.Column)

        // Add the children to the stack.
        for key, child := range current.node.Children {
            stack = append(stack, &traversal{
                keys: append(append([]string{}, current.keys...), key),
                node: child,
            })
        }
    } else {
        fmt.Printf("Scalar node at line %d, column %d: %s -> %s\n", current.node.Line, current.node.Column, strings.Join(current.keys, "."), current.node.Value)
    }
}
```

## Todo

-   [x] Decoding to node
-   [ ] Decoding to struct
-   [ ] Encoding from node
-   [ ] Encoding from struct
