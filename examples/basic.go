package examples

import (
	"fmt"
	"log"

	"github.com/Reboxed/HumanID"
)

func Basic() {
	generator, err := HumanID.Load(100)
	if err != nil {
		log.Fatal(err)
	}
	id, err := generator.Encode(12345, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Human ID for 12345: %s\n", id)
	decoded, err := generator.Decode(id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded back: %d\n", decoded)
}
