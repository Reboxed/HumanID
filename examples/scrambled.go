package examples

import (
	"fmt"
	"log"

	"github.com/Reboxed/HumanID"
)

func Scrambled() {
	key := [4]uint32{0x12345678, 0x9abcdef0, 0x0fedcba9, 0x87654321}
	generator, err := HumanID.Load(100, key)
	if err != nil {
		log.Fatal(err)
	}
	id, err := generator.EncodeScrambled(12345, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Scrambled Human ID for 12345: %s\n", id)
	decoded, err := generator.DecodeFromScrambled(id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded back: %d\n", decoded)
}
