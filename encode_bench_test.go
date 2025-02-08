package govdf_test

import (
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
)

func BenchmarkMarshal_SimpleStruct(b *testing.B) {
	type Person struct {
		Name string `vdf:"name"`
		Age  int    `vdf:"age"`
	}

	var person = Person{
		Name: "John Doe",
		Age:  30,
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := govdf.Marshal(person); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_ComplexStruct(b *testing.B) {
	type Address struct {
		Street string `vdf:"street"`
		City   string `vdf:"city"`
		State  string `vdf:"state"`
		Zip    string `vdf:"zip"`
	}

	type Company struct {
		Name    string  `vdf:"name"`
		Address Address `vdf:"address"`
		Active  bool    `vdf:"active"`
	}

	type Employee struct {
		ID      int     `vdf:"id"`
		Name    string  `vdf:"name"`
		Email   string  `vdf:"email"`
		Company Company `vdf:"company"`
		Active  bool    `vdf:"active"`
	}

	var employee = Employee{
		ID:    12345,
		Name:  "Jane Smith",
		Email: "jane.smith@example.com",
		Company: Company{
			Name: "Acme Corp",
			Address: Address{
				Street: "123 Main St",
				City:   "Anytown",
				State:  "CA",
				Zip:    "12345",
			},
			Active: true,
		},
		Active: true,
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := govdf.Marshal(employee); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_Node(b *testing.B) {
	var node = &govdf.Node{
		Type: govdf.NodeTypeMap,
		Children: map[string]*govdf.Node{
			"user": {
				Type: govdf.NodeTypeMap,
				Children: map[string]*govdf.Node{
					"name": {
						Type:  govdf.NodeTypeScalar,
						Value: "John Doe",
					},
					"age": {
						Type:  govdf.NodeTypeScalar,
						Value: "30",
					},
					"address": {
						Type: govdf.NodeTypeMap,
						Children: map[string]*govdf.Node{
							"street": {
								Type:  govdf.NodeTypeScalar,
								Value: "123 Main St",
							},
							"city": {
								Type:  govdf.NodeTypeScalar,
								Value: "Anytown",
							},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := govdf.Marshal(node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_Parallel(b *testing.B) {
	type Data struct {
		ID    int    `vdf:"id"`
		Name  string `vdf:"name"`
		Value string `vdf:"value"`
	}

	var data = Data{
		ID:    42,
		Name:  "test",
		Value: "benchmark data",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := govdf.Marshal(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
