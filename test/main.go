package main

import (
	"fmt"

	humanreadable "github.com/Reboxed/HumanReadable"
)

func main() {
	generator, err := humanreadable.Load();
	if err != nil {
		panic(err)
	}
	for range 10 {
		generated, _ := generator.Generate(2, 100)
		fmt.Println(generated)
	}
}


