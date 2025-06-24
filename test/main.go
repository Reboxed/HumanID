package main

import (
	"fmt"
	"log"

	humanreadable "github.com/Reboxed/HumanReadable"
)

func main() {
	const iterations = 50_000_000
	const maxLength = 100

	generator, err := humanreadable.Load()
	if err != nil {
		log.Fatal(err)
	}

	seen := make(map[string]int)              // Map to store first occurrence index
	duplicates := make(map[string][]int)      // Map to store duplicate IDs with all their indexes

	for i := 0; i < iterations; i++ {
		id, err := generator.Encode(uint64(i), 2, true)
		if err != nil {
			log.Fatalf("Generation failed at iteration %d: %v", i, err)
		}

		if firstIndex, exists := seen[id]; exists {
			// Found a duplicate
			if _, ok := duplicates[id]; !ok {
				duplicates[id] = []int{firstIndex}
			}
			duplicates[id] = append(duplicates[id], i)
		} else {
			seen[id] = i
		}

		// Optional: periodically log progress
		if i%10_000_000 == 0 {
			fmt.Printf("Checked %d IDs... Found %d duplicates so far. Example: \"%s\"\n", i, len(duplicates), id)
		}
	}

	if len(duplicates) == 0 {
		fmt.Println("No duplicate IDs found.")
	} else {
		fmt.Printf("Found %d duplicate IDs:\n", len(duplicates))
		for id, indices := range duplicates {
			fmt.Printf("ID: %s — Occurred at indexes: %v\n", id, indices)
		}
	}
}
