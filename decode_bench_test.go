package govdf_test

import (
	"strings"
	"testing"

	govdf "github.com/lewisgibson/go-vdf"
)

func BenchmarkUnmarshal_SimpleStruct(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"name" "John Doe"`,
		`"age" "30"`,
	}, "\n")

	type Person struct {
		Name string `vdf:"name"`
		Age  int    `vdf:"age"`
	}

	b.ResetTimer()
	for b.Loop() {
		var person Person
		if err := govdf.Unmarshal([]byte(vdfData), &person); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_ComplexStruct(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"id" "12345"`,
		`"name" "Jane Smith"`,
		`"email" "jane.smith@example.com"`,
		`"company" {`,
		`	"name" "Acme Corp"`,
		`	"address" {`,
		`		"street" "123 Main St"`,
		`		"city" "Anytown"`,
		`		"state" "CA"`,
		`		"zip" "12345"`,
		`	}`,
		`	"active" "true"`,
		`}`,
		`"active" "true"`,
	}, "\n")

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

	b.ResetTimer()
	for b.Loop() {
		var employee Employee
		if err := govdf.Unmarshal([]byte(vdfData), &employee); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_Node(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"user" {`,
		`	"name" "John Doe"`,
		`	"age" "30"`,
		`	"address" {`,
		`		"street" "123 Main St"`,
		`		"city" "Anytown"`,
		`	}`,
		`}`,
	}, "\n")

	b.ResetTimer()
	for b.Loop() {
		var node govdf.Node
		if err := govdf.Unmarshal([]byte(vdfData), &node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_WithComments(b *testing.B) {
	var vdfData = strings.Join([]string{
		`// This is a head comment`,
		`"user" {`,
		`	"name" "John Doe"	// This is a line comment`,
		`	"age" "30"`,
		`	"address" {`,
		`		"street" "123 Main St"`,
		`		"city" "Anytown"`,
		`	}`,
		`}`,
	}, "\n")

	type User struct {
		Name    string `vdf:"name"`
		Age     int    `vdf:"age"`
		Address struct {
			Street string `vdf:"street"`
			City   string `vdf:"city"`
		} `vdf:"address"`
	}

	b.ResetTimer()
	for b.Loop() {
		var user User
		if err := govdf.Unmarshal([]byte(vdfData), &user); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_QuotedStrings(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"message" "Hello \"world\" with quotes"`,
		`"path" "C:\\Program Files\\Game"`,
		`"json" "{\"key\": \"value\"}"`,
	}, "\n")

	type Data struct {
		Message string `vdf:"message"`
		Path    string `vdf:"path"`
		JSON    string `vdf:"json"`
	}

	b.ResetTimer()
	for b.Loop() {
		var data Data
		if err := govdf.Unmarshal([]byte(vdfData), &data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_MixedTypes(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"string_field" "hello world"`,
		`"int_field" "42"`,
		`"bool_field" "true"`,
		`"float_field" "3.14159"`,
	}, "\n")

	type MixedData struct {
		String string  `vdf:"string_field"`
		Int    int     `vdf:"int_field"`
		Bool   bool    `vdf:"bool_field"`
		Float  float64 `vdf:"float_field"`
	}

	b.ResetTimer()
	for b.Loop() {
		var data MixedData
		if err := govdf.Unmarshal([]byte(vdfData), &data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_Parallel(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"id" "42"`,
		`"name" "test"`,
		`"value" "benchmark data"`,
	}, "\n")

	type Data struct {
		ID    int    `vdf:"id"`
		Name  string `vdf:"name"`
		Value string `vdf:"value"`
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var data Data
			if err := govdf.Unmarshal([]byte(vdfData), &data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUnmarshal_LargeNested(b *testing.B) {
	// Create a large nested VDF structure
	var vdfData = strings.Join([]string{
		`"level1" {`,
		`	"level2a" {`,
		`		"level3a" {`,
		`			"value1" "test1"`,
		`			"value2" "test2"`,
		`			"value3" "test3"`,
		`		}`,
		`		"level3b" {`,
		`			"value4" "test4"`,
		`			"value5" "test5"`,
		`			"value6" "test6"`,
		`		}`,
		`	}`,
		`	"level2b" {`,
		`		"level3c" {`,
		`			"value7" "test7"`,
		`			"value8" "test8"`,
		`			"value9" "test9"`,
		`		}`,
		`		"level3d" {`,
		`			"value10" "test10"`,
		`			"value11" "test11"`,
		`			"value12" "test12"`,
		`		}`,
		`	}`,
		`}`,
	}, "\n")

	type Level3 struct {
		Value1 string `vdf:"value1"`
		Value2 string `vdf:"value2"`
		Value3 string `vdf:"value3"`
	}

	type Level2 struct {
		Level3A Level3 `vdf:"level3a"`
		Level3B Level3 `vdf:"level3b"`
		Level3C Level3 `vdf:"level3c"`
		Level3D Level3 `vdf:"level3d"`
	}

	type Level1 struct {
		Level2A Level2 `vdf:"level2a"`
		Level2B Level2 `vdf:"level2b"`
	}

	b.ResetTimer()
	for b.Loop() {
		var level1 Level1
		if err := govdf.Unmarshal([]byte(vdfData), &level1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_CustomUnmarshaler(b *testing.B) {
	var vdfData = `"custom_field" "test_value"`

	type CustomUnmarshalerStruct struct {
		CustomField mockUnmarshaler `vdf:"custom_field"`
	}

	b.ResetTimer()
	for b.Loop() {
		var data CustomUnmarshalerStruct
		if err := govdf.Unmarshal([]byte(vdfData), &data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_OptionalFields(b *testing.B) {
	var vdfData = strings.Join([]string{
		`"required" {`,
		`	"name" "John"`,
		`	"age" "30"`,
		`}`,
		`"optional" {`,
		`	"name" "Jane"`,
		`	"age" "25"`,
		`}`,
	}, "\n")

	type Person struct {
		Name string `vdf:"name"`
		Age  int    `vdf:"age"`
	}

	type TestStruct struct {
		Required Person  `vdf:"required"`
		Optional *Person `vdf:"optional"`
		Nil      *Person `vdf:"nil"`
	}

	b.ResetTimer()
	for b.Loop() {
		var data TestStruct
		if err := govdf.Unmarshal([]byte(vdfData), &data); err != nil {
			b.Fatal(err)
		}
	}
}
