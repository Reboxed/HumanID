package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Reboxed/HumanID"
)

const (
	iterations = 50_000_000_000
)

func main() {
	start := time.Now()

	generator, err := HumanID.Load(100, [4]uint32{0x12345678, 0x9abcdef0, 0x0fedcba9, 0x87654321})
	if err != nil {
		log.Fatal(err)
	}
	combinations := generator.MaxCombinations(2)
	fmt.Printf("Max combinations of %d\n", combinations)

	seen := make(map[string]int)
	duplicates := make(map[string][]int)

	for i := 0; i < iterations; i++ {
		id, err := generator.EncodeScrambled(uint64(i), 2)
		dec, errDec := generator.DecodeFromScrambled(id)

		if err != nil {
			fmt.Printf("Generation failed at iteration %d: %v\n", i, err)
		}
		if dec != uint64(i) {
			fmt.Printf("Decoding invalid at iteration %d with `%s`: found %d: %v\n", i, id, dec, errDec)
		} /*  else {
			fmt.Printf("`%s` with %d decodes %d\n", id, i, dec)
		} */

		if firstIndex, exists := seen[id]; exists {
			if _, ok := duplicates[id]; !ok {
				duplicates[id] = []int{firstIndex}
			}
			duplicates[id] = append(duplicates[id], i)
		} else {
			seen[id] = i
		}

		// Optional progress log
		if i%15_000_000 == 0 {
			ranNum := uint64(rand.Intn(int(combinations)))

			example, _ := generator.EncodeScrambled(ranNum, 2)
			reverse, err := generator.DecodeFromScrambled(example)

			fmt.Printf("checked %d... duplicates: %d, example: %s\n", i, len(duplicates), example)
			if reverse != ranNum {
				fmt.Printf("INVALID DECODING. Expected %d, got %d: %v\n", ranNum, reverse, err)
			}
		}
	}

	elapsed := time.Since(start)
	if len(duplicates) == 0 {
		fmt.Println("No duplicate IDs found.")
	} else {
		fmt.Printf("Found %d duplicate IDs:\n", len(duplicates))
		for id, indices := range duplicates {
			fmt.Printf("ID: %s — Occurred at indexes: %v\n", id, indices)
		}
	}
	fmt.Printf("Completed in %s.\n", elapsed)
}
