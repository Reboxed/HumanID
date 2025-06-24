package humanreadable

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Generator struct {
	Adjectives []string
	Nouns      []string 
	baseA      int
	baseN      int
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
		Adjectives: adjectives[:],
		Nouns:      append(nouns, adjectives...),
		baseA:      len(adjectives),
		baseN:      len(nouns),
	}, nil
}

var INVALID_PIECES_LENGTH = errors.New("Pieces have to be at least two")
var GENERATOR_NOT_LOADED = errors.New("The generator data is not loaded")

func filterEmpty(input []string) []string {
	var result []string
	for _, s := range input {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// Encode converts a number into a unique human-readable string
func (g *Generator) Encode(index uint64, adjectivesCount int, addSuffix bool) (string, error) {
	if adjectivesCount < 1 {
		return "", errors.New("must use at least 1 adjective")
	}
	if g.baseA == 0 || g.baseN == 0 {
		return "", errors.New("adjective or noun list is empty")
	}

	originalIndex := index // keep the original index

	pieces := make([]string, adjectivesCount+1)

	// Get noun
	nounIndex := index % uint64(g.baseN)
	pieces[adjectivesCount] = g.Nouns[nounIndex]
	index /= uint64(g.baseN)

	// Get adjectives
	for i := adjectivesCount - 1; i >= 0; i-- {
		adjIndex := index % uint64(g.baseA)
		pieces[i] = g.Adjectives[adjIndex]
		index /= uint64(g.baseA)
	}

	result := strings.Join(pieces, "-")

	if addSuffix {
		const prime uint64 = 2654435761
		suffix := (originalIndex * prime) % 100
		result = fmt.Sprintf("%s-%d", result, suffix)
	}

	return result, nil
}

// Decode converts a human-readable string back into a number
func (g *Generator) Decode(input string, adjectivesCount int) (uint64, error) {
	parts := strings.Split(input, "-")

	var suffixValue *int
	// Check if last part is a number suffix in [0, 100)
	if len(parts) > adjectivesCount+1 {
		lastPart := parts[len(parts)-1]
		if n, err := strconv.Atoi(lastPart); err == nil && n >= 0 && n < 100 {
			suffixValue = &n
			parts = parts[:len(parts)-1] // strip suffix
		}
	}

	if len(parts) != adjectivesCount+1 {
		return 0, fmt.Errorf("expected %d adjectives and 1 noun, got %d parts", adjectivesCount, len(parts))
	}

	var num uint64 = 0

	// Decode adjectives
	for i := 0; i < adjectivesCount; i++ {
		idx := indexOf(g.Adjectives, parts[i])
		if idx == -1 {
			return 0, fmt.Errorf("invalid adjective: %s", parts[i])
		}
		num = num*uint64(g.baseA) + uint64(idx)
	}

	// Decode noun
	nounIdx := indexOf(g.Nouns, parts[adjectivesCount])
	if nounIdx == -1 {
		return 0, fmt.Errorf("invalid noun: %s", parts[adjectivesCount])
	}
	num = num*uint64(g.baseN) + uint64(nounIdx)

	// Validate suffix if present
	if suffixValue != nil {
		const prime uint64 = 2654435761
		expectedSuffix := (num * prime) % 100
		if expectedSuffix != uint64(*suffixValue) {
			return 0, fmt.Errorf("suffix mismatch: expected %d, got %d", expectedSuffix, *suffixValue)
		}
	}

	return num, nil
}

func indexOf(list []string, target string) int {
	for i, val := range list {
		if val == target {
			return i
		}
	}
	return -1
}

