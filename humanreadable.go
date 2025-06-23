package huanreadable

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

type Generator struct {
	loaded     bool
	Adjectives []string
	Nouns      []string
}

var ADJECTIVES_FILE_NOT_FOUND = errors.New("Adjectives file not found")
var NOUNS_FILE_NOT_FOUND = errors.New("Adjectives file not found")

func Load() (*Generator, error) {
	// nouns
	adjBytes, err := os.ReadFile("./adjectives.txt")
	if err != nil {
		return nil, ADJECTIVES_FILE_NOT_FOUND
	}
	adjectivesStr := string(adjBytes)
	adjectives := strings.Split(adjectivesStr, "\n")

	// nouns
	nounsBytes, err := os.ReadFile("./nouns.txt")
	if err != nil {
		return nil, NOUNS_FILE_NOT_FOUND
	}
	nounsStr := string(nounsBytes)
	nouns := strings.Split(nounsStr, "\n")

	return &Generator {
		loaded: true,
		Adjectives: adjectives[:],
		Nouns: append(nouns, adjectives...),
	}, nil
}

var INVALID_PIECES_LENGTH = errors.New("Pieces have to be at least two")
var GENERATOR_NOT_LOADED = errors.New("The generator data is not loaded")

func (generator *Generator) Generate(pieces int, numbers bool) (string, error) {
	if !generator.loaded { return "", GENERATOR_NOT_LOADED }
	if pieces < 2 { return "", INVALID_PIECES_LENGTH }

	// Generate adjective
	var adjectiveStr string = ""
	for range pieces - 1 {
		randomAdjective := rand.Intn(len(generator.Adjectives))
		if len(adjectiveStr) == 0 {
			adjectiveStr += generator.Adjectives[randomAdjective]
			continue
		}
		adjectiveStr += fmt.Sprintf("-%s", generator.Adjectives[randomAdjective])
	}

	// Generate noun
	randomNoun := rand.Intn(len(generator.Nouns))
	finalStr := fmt.Sprintf("%s-%s-%d", adjectiveStr, generator.Nouns[randomNoun], rand.Intn(1000))
	return finalStr, nil
}
