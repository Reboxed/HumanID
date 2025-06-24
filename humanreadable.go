package humanreadable

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Generator holds the word lists and configuration for generating IDs.
type Generator struct {
	adjectives      []string
	nouns           []string
	baseA           int            // Number of unique adjectives
	baseN           int            // Number of unique nouns
	maxCombinations map[int]uint64 // Cache for combination calculations
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// Pre-defined errors for file loading and input validation.
var (
	ADJECTIVES_FILE_NOT_FOUND = errors.New("adjectives file not found")
	NOUNS_FILE_NOT_FOUND      = errors.New("nouns file not found")
	INVALID_PIECES_LENGTH     = errors.New("ID must have at least one adjective and one noun")
	GENERATOR_NOT_LOADED      = errors.New("the generator data is not loaded")
)

// Load initializes a new Generator. It reads adjectives and nouns from text files,
// filters them to ensure they are simple alphanumeric words, removes duplicates,
// and shuffles them based on the provided seed.
func Load(seed int64) (*Generator, error) {
	// **FIX**: This regex ensures we only load words that are purely alphanumeric.
	// This prevents words with hyphens or other special characters from breaking
	// the Decode logic, which relies on splitting by '-'.
	alphaNumRegex := regexp.MustCompile(`^[a-z0-9]+$`)

	// Load adjectives from file
	adjBytes, err := os.ReadFile(filepath.Join(basepath, "adjectives.txt"))
	if err != nil {
		return nil, ADJECTIVES_FILE_NOT_FOUND
	}
	initialAdjectives := strings.Split(string(adjBytes), "\n")
	var filteredAdjectives []string
	for _, adj := range initialAdjectives {
		processedWord := strings.TrimSpace(strings.ToLower(adj))
		if alphaNumRegex.MatchString(processedWord) {
			filteredAdjectives = append(filteredAdjectives, processedWord)
		}
	}
	adjectives := unique(filteredAdjectives)

	// Load nouns from file
	nounsBytes, err := os.ReadFile(filepath.Join(basepath, "nouns.txt"))
	if err != nil {
		return nil, NOUNS_FILE_NOT_FOUND
	}
	initialNouns := strings.Split(string(nounsBytes), "\n")
	var filteredNouns []string
	for _, noun := range initialNouns {
		processedWord := strings.TrimSpace(strings.ToLower(noun))
		if alphaNumRegex.MatchString(processedWord) {
			filteredNouns = append(filteredNouns, processedWord)
		}
	}
	nouns := unique(filteredNouns)

	// If no seed is provided, use current time
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	// Create a new pseudo-random number generator
	r := rand.New(rand.NewSource(seed))

	// Shuffle both adjectives and nouns to ensure non-sequential IDs
	shuffledAdjectives := make([]string, len(adjectives))
	copy(shuffledAdjectives, adjectives)
	r.Shuffle(len(shuffledAdjectives), func(i, j int) {
		shuffledAdjectives[i], shuffledAdjectives[j] = shuffledAdjectives[j], shuffledAdjectives[i]
	})

	shuffledNouns := make([]string, len(nouns))
	copy(shuffledNouns, nouns)
	r.Shuffle(len(shuffledNouns), func(i, j int) {
		shuffledNouns[i], shuffledNouns[j] = shuffledNouns[j], shuffledNouns[i]
	})

	return &Generator{
		adjectives:      shuffledAdjectives,
		nouns:           shuffledNouns,
		baseA:           len(adjectives),
		baseN:           len(nouns),
		maxCombinations: make(map[int]uint64),
	}, nil
}

// MaxCombinations calculates the total number of unique combinations with exactly n adjectives.
// This number does not include the numeric suffix.
func (g *Generator) MaxCombinations(adjectivesCount int) uint64 {
	if adjectivesCount < 1 {
		return 0
	}
	// Protect against adjectivesCount being larger than the available adjectives, which would cause an infinite loop in some scenarios.
	// While the adjective list itself isn't directly indexed by this, it's a logical constraint.
	if g.baseA > 0 && adjectivesCount > g.baseA {
		// This check is more semantic, the math would still work but the IDs would have repeating adjectives
	}

	// Use a cached value if available
	if val, ok := g.maxCombinations[adjectivesCount]; ok {
		return val
	}

	var combos uint64 = 1
	// Calculate base combinations from words (adjectives^count * nouns)
	for i := 0; i < adjectivesCount; i++ {
		// Prevent overflow by checking before multiplication
		if combos > (1<<64-1)/uint64(g.baseA) {
			return 0 // Represents a number too large to fit in uint64
		}
		combos *= uint64(g.baseA)
	}

	if combos > (1<<64-1)/uint64(g.baseN) {
		return 0 // Represents a number too large to fit in uint64
	}
	combos *= uint64(g.baseN)

	// Cache the result for future use
	g.maxCombinations[adjectivesCount] = combos
	return combos
}

// Encode converts a number into a unique human-readable string.
// The mapping is reversible by the Decode function.
func (g *Generator) Encode(index uint64, adjectivesCount int) (string, error) {
	if adjectivesCount < 1 {
		return "", errors.New("must use at least 1 adjective")
	}
	if g.baseA == 0 || g.baseN == 0 {
		return "", errors.New("adjective or noun list is empty")
	}

	// Get the number of combinations from words alone
	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return "", errors.New("adjective count is too high or lists are empty, or combinations overflowed uint64")
	}

	// Total combinations are baseCombos * 100 (1 for no suffix, 99 for suffixes 1-99)
	maxTotalCombos := baseCombos * 100
	// Check for overflow in the total combinations calculation
	if baseCombos > (1<<64-1)/100 {
		maxTotalCombos = 1<<64 - 1 // Clamp to max uint64
	}

	if index >= maxTotalCombos && maxTotalCombos != 1<<64-1 {
		return "", fmt.Errorf("index %d out of bounds (max %d)", index, maxTotalCombos-1)
	}

	// The word part of the ID is based on the remainder (`num`)
	num := index % baseCombos
	// The suffix is based on which block of `baseCombos` the index falls into
	suffix := int(index / baseCombos)

	pieces := make([]string, adjectivesCount+1)

	// Derive the noun from the `num`
	nounIndex := num % uint64(g.baseN)
	pieces[adjectivesCount] = g.nouns[nounIndex]
	num /= uint64(g.baseN)

	// Derive the adjectives from the `num`
	for i := adjectivesCount - 1; i >= 0; i-- {
		adjIndex := num % uint64(g.baseA)
		pieces[i] = g.adjectives[adjIndex]
		num /= uint64(g.baseA)
	}

	result := strings.Join(pieces, "-")

	// Suffix 0 means no suffix is appended. Suffixes 1-99 are appended.
	if suffix > 0 {
		result = fmt.Sprintf("%s-%d", result, suffix)
	}

	return result, nil
}

// Decode converts a human-readable string back into the original number.
func (g *Generator) Decode(input string) (uint64, error) {
	if g.baseA == 0 || g.baseN == 0 {
		return 0, GENERATOR_NOT_LOADED
	}
	parts := strings.Split(input, "-")
	if len(parts) < 2 {
		return 0, INVALID_PIECES_LENGTH
	}

	var suffix int
	last := parts[len(parts)-1]
	// Check if the last part is a numeric suffix
	if s, err := strconv.Atoi(last); err == nil && s >= 1 && s <= 99 {
		suffix = s
		parts = parts[:len(parts)-1] // strip suffix from parts
	}

	adjectivesCount := len(parts) - 1
	if adjectivesCount < 1 {
		return 0, INVALID_PIECES_LENGTH
	}

	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return 0, errors.New("could not calculate combinations for decoding")
	}

	// Reconstruct the number (`num`) from the adjective and noun parts
	var num uint64

	// Noun is the last part of the word list
	noun := parts[len(parts)-1]
	nounIndex := indexOf(g.nouns, noun)
	if nounIndex < 0 {
		return 0, fmt.Errorf("noun %q not found", noun)
	}

	// Adjectives are the preceding parts
	var adjectiveNumPart uint64
	for i := 0; i < adjectivesCount; i++ {
		adj := parts[i]
		idx := indexOf(g.adjectives, adj)
		if idx < 0 {
			return 0, fmt.Errorf("adjective %q not found", adj)
		}
		adjectiveNumPart = adjectiveNumPart*uint64(g.baseA) + uint64(idx)
	}

	// Combine adjective and noun parts to get the intermediate number
	num = adjectiveNumPart*uint64(g.baseN) + uint64(nounIndex)

	// Reconstruct the original index from the num and suffix
	index := uint64(suffix)*baseCombos + num

	return index, nil
}

// indexOf finds the index of a target string in a slice of strings.
// Returns -1 if the target is not found.
func indexOf(list []string, target string) int {
	for i, val := range list {
		if val == target {
			return i
		}
	}
	return -1
}

// unique returns a new slice containing only the unique non-empty strings from the input.
func unique(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range input {
		// The input to this function is already processed (trimmed, lowercased)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}
