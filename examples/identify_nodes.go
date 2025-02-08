package main

import (
	"fmt"
	"strings"

	govdf "github.com/lewisgibson/go-vdf"
)

func main() {
	vdfBytes := []byte(`
    "items_game"
    {
        "game_info"
        {
            "first_valid_class"             "2"
            "last_valid_class"              "3"
            "first_valid_item_slot"         "0"
            "last_valid_item_slot"          "54"
            "num_item_presets"              "4"
            "max_num_stickers"              "5"
            "max_num_patches"               "3"
        }
    }
    `)

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
}
