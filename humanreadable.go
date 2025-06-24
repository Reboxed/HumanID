package humanreadable

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Generator struct {
	loaded     bool
	Adjectives []string
	Nouns      []string
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

var ADJECTIVES_FILE_NOT_FOUND = errors.New("Adjectives file not found")
var NOUNS_FILE_NOT_FOUND = errors.New("Adjectives file not found")

func Load() (*Generator, error) {
	// nouns
	adjBytes, err := os.ReadFile(filepath.Join(basepath, "adjectives.txt"))
	if err != nil {
		return nil, ADJECTIVES_FILE_NOT_FOUND
	}
	adjectivesStr := string(adjBytes)
	adjectives := strings.Split(adjectivesStr, "\n")

	// nouns
	nounsBytes, err := os.ReadFile(filepath.Join(basepath, "nouns.txt"))
	if err != nil {
		return nil, NOUNS_FILE_NOT_FOUND
	}
	nounsStr := string(nounsBytes)
	nouns := strings.Split(nounsStr, "\n")

	return &Generator{
		loaded:     true,
		Adjectives: adjectives[:],
		Nouns:      append(nouns, adjectives...),
	}, nil
}

var INVALID_PIECES_LENGTH = errors.New("Pieces have to be at least two")
var GENERATOR_NOT_LOADED = errors.New("The generator data is not loaded")

func (generator *Generator) Generate(pieces int, highestNumber int) (string, error) {
	if !generator.loaded {
		return "", GENERATOR_NOT_LOADED
	}
	if pieces < 2 {
		return "", INVALID_PIECES_LENGTH
	}

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
	randomNr := rand.Intn(highestNumber)
	if highestNumber > 0 && randomNr > 0 {
		finalStr := fmt.Sprintf("%s-%s-%d", adjectiveStr, generator.Nouns[randomNoun], randomNr)
		return finalStr, nil
	} else {
		finalStr := fmt.Sprintf("%s-%s", adjectiveStr, generator.Nouns[randomNoun])
		return finalStr, nil
	}
}
