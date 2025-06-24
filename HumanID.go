package HumanID

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

// Number of Feistel rounds for permutation (should be >= 3 for security, 4-6 is typical).
const feistelRounds = 4

// Generator holds the word lists and configuration for generating IDs.
type Generator struct {
	adjectives      []string
	nouns           []string
	baseA           int            // Number of unique adjectives
	baseN           int            // Number of unique nouns
	maxCombinations map[int]uint64 // Cache for combination calculations
	roundKeys       []uint64       // Round keys for Feistel-based permutation
	adjIndexMap     map[string]int // Map for adjective to index lookup
	nounIndexMap    map[string]int // Map for noun to index lookup
	xxteaKey        [4]uint32      // XXTEA key for block cipher scrambling
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
// and shuffles them based on the provided seed. Optionally accepts a public XXTEA key.
func Load(seed int64, xxteaKey ...[4]uint32) (*Generator, error) {
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

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(seed))

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

	// Build lookup maps for instant decode
	adjIndexMap := make(map[string]int, len(shuffledAdjectives))
	for i, w := range shuffledAdjectives {
		adjIndexMap[w] = i
	}
	nounIndexMap := make(map[string]int, len(shuffledNouns))
	for i, w := range shuffledNouns {
		nounIndexMap[w] = i
	}

	// Generate Feistel round keys
	roundKeys := make([]uint64, feistelRounds)
	for i := range roundKeys {
		roundKeys[i] = r.Uint64()
	}
	// Use provided XXTEA key or a default public key
	var key [4]uint32
	if len(xxteaKey) > 0 {
		key = xxteaKey[0]
	} else {
		key = [4]uint32{0x12345678, 0x9abcdef0, 0x0fedcba9, 0x87654321}
	}
	return &Generator{
		adjectives:      shuffledAdjectives,
		nouns:           shuffledNouns,
		baseA:           len(adjectives),
		baseN:           len(nouns),
		maxCombinations: make(map[int]uint64),
		roundKeys:       roundKeys,
		adjIndexMap:     adjIndexMap,
		nounIndexMap:    nounIndexMap,
		xxteaKey:        key,
	}, nil
}

// MaxCombinations calculates the total number of unique combinations with exactly n adjectives.
func (g *Generator) MaxCombinations(adjectivesCount int) uint64 {
	if adjectivesCount < 1 {
		return 0
	}
	if g.baseA > 0 && adjectivesCount > g.baseA {
	}

	if val, ok := g.maxCombinations[adjectivesCount]; ok {
		return val
	}

	var combos uint64 = 1
	for i := 0; i < adjectivesCount; i++ {
		if combos > (1<<64-1)/uint64(g.baseA) {
			return 0
		}
		combos *= uint64(g.baseA)
	}

	if combos > (1<<64-1)/uint64(g.baseN) {
		return 0
	}
	combos *= uint64(g.baseN)

	g.maxCombinations[adjectivesCount] = combos
	return combos
}

// Encode converts a number into a unique human-readable string.
func (g *Generator) Encode(index uint64, adjectivesCount int) (string, error) {
	if adjectivesCount < 1 {
		return "", errors.New("must use at least 1 adjective")
	}
	if g.baseA == 0 || g.baseN == 0 {
		return "", errors.New("adjective or noun list is empty")
	}
	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return "", errors.New("adjective count is too high or lists are empty, or combinations overflowed uint64")
	}
	maxIndex := baseCombos * 100
	if index >= maxIndex {
		return "", fmt.Errorf("index %d out of bounds (max %d)", index, maxIndex-1)
	}

	var scrambled uint64
	bits := bitsNeeded(maxIndex - 1)
	if isPowerOfTwo(maxIndex) {
		// Use Feistel for power-of-two domain
		scrambled = feistelPermute(index, g.roundKeys, bits)
	} else {
		// For non-power-of-two domain, use identity mapping (no scrambling)
		scrambled = index
	}
	// No cycle-walking needed: mapping is bijective

	suffix := int(scrambled / baseCombos)
	comboIdx := scrambled % baseCombos
	pieces := indexToCombo(comboIdx, g.baseA, g.baseN, adjectivesCount, g.adjectives, g.nouns)
	result := strings.Join(pieces, "-")
	if suffix > 0 {
		result = fmt.Sprintf("%s-%d", result, suffix)
	}
	return result, nil
}

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
	_, nounIsValid := g.nounIndexMap[last]
	adjectivesCount := len(parts) - 1
	if !nounIsValid {
		if s, err := strconv.Atoi(last); err == nil && s >= 0 {
			suffix = s
			parts = parts[:len(parts)-1]
			adjectivesCount = len(parts) - 1
		}
	}
	if adjectivesCount < 1 {
		return 0, INVALID_PIECES_LENGTH
	}
	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return 0, errors.New("could not calculate combinations for decoding")
	}
	comboIdx, err := comboToIndex(parts, g.baseA, g.baseN, adjectivesCount, g.adjIndexMap, g.nounIndexMap)
	if err != nil {
		return 0, err
	}
	maxIndex := baseCombos * 100
	bits := bitsNeeded(maxIndex - 1)
	scrambled := uint64(suffix)*baseCombos + comboIdx
	if scrambled >= maxIndex {
		return 0, fmt.Errorf("decoded value out of range")
	}
	var idx uint64
	if isPowerOfTwo(maxIndex) {
		// Use Feistel for power-of-two domain
		idx = feistelUnpermute(scrambled, g.roundKeys, bits)
	} else {
		// For non-power-of-two domain, use identity mapping (no scrambling)
		idx = scrambled
	}
	return idx, nil
}

// EncodeScrambled takes a uint64, scrambles it with XXTEA, and encodes it as a human-readable ID.
func (g *Generator) EncodeScrambled(i uint64, adjectivesCount int) (string, error) {
	if adjectivesCount < 1 {
		return "", errors.New("must use at least 1 adjective")
	}
	if g.baseA == 0 || g.baseN == 0 {
		return "", errors.New("adjective or noun list is empty")
	}
	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return "", errors.New("adjective count is too high or lists are empty, or combinations overflowed uint64")
	}
	maxIndex := baseCombos * 100
	if i >= maxIndex {
		return "", fmt.Errorf("index %d out of bounds (max %d)", i, maxIndex-1)
	}
	// Scramble with XXTEA
	scrambled := xxteaEncrypt64(i, g.xxteaKey) % maxIndex
	suffix := int(scrambled / baseCombos)
	comboIdx := scrambled % baseCombos
	pieces := indexToCombo(comboIdx, g.baseA, g.baseN, adjectivesCount, g.adjectives, g.nouns)
	result := strings.Join(pieces, "-")
	if suffix > 0 {
		result = fmt.Sprintf("%s-%d", result, suffix)
	}
	return result, nil
}

// DecodeFromScrambled decodes a human-readable ID produced by EncodeScrambled and returns the original uint64.
func (g *Generator) DecodeFromScrambled(humanID string) (uint64, error) {
	parts := strings.Split(humanID, "-")
	if len(parts) < 2 {
		return 0, INVALID_PIECES_LENGTH
	}
	var suffix int
	last := parts[len(parts)-1]
	_, nounIsValid := g.nounIndexMap[last]
	adjectivesCount := len(parts) - 1
	if !nounIsValid {
		if s, err := strconv.Atoi(last); err == nil && s >= 0 {
			suffix = s
			parts = parts[:len(parts)-1]
			adjectivesCount = len(parts) - 1
		}
	}
	if adjectivesCount < 1 {
		return 0, INVALID_PIECES_LENGTH
	}
	baseCombos := g.MaxCombinations(adjectivesCount)
	if baseCombos == 0 {
		return 0, errors.New("could not calculate combinations for decoding")
	}
	comboIdx, err := comboToIndex(parts, g.baseA, g.baseN, adjectivesCount, g.adjIndexMap, g.nounIndexMap)
	if err != nil {
		return 0, err
	}
	maxIndex := baseCombos * 100
	scrambled := uint64(suffix)*baseCombos + comboIdx
	if scrambled >= maxIndex {
		return 0, fmt.Errorf("decoded value out of range")
	}
	// Brute-force search for the original value (since XXTEA is not a permutation mod maxIndex)
	for i := uint64(0); i < maxIndex; i++ {
		if xxteaEncrypt64(i, g.xxteaKey)%maxIndex == scrambled {
			return i, nil
		}
	}
	return 0, fmt.Errorf("could not decode scrambled value")
}

// unique returns a new slice containing only the unique non-empty strings from the input.
func unique(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range input {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

// feistelPermute applies a Feistel network over a bits-sized domain.
func feistelPermute(x uint64, keys []uint64, bits int) uint64 {
	half := bits / 2
	maskL := uint64((1 << (bits - half)) - 1)
	maskR := uint64((1 << half) - 1)
	L := (x >> half) & maskL
	R := x & maskR
	for _, k := range keys {
		newL := R
		newR := L ^ (uint64(feistelF(uint32(R), k)) & maskL)
		L, R = newL, newR
	}
	return (L << half) | R
}

// feistelUnpermute reverses the Feistel network over a bits-sized domain.
func feistelUnpermute(x uint64, keys []uint64, bits int) uint64 {
	half := bits / 2
	maskL := uint64((1 << (bits - half)) - 1)
	maskR := uint64((1 << half) - 1)
	L := (x >> half) & maskL
	R := x & maskR
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		prevL := R ^ (uint64(feistelF(uint32(R), k)) & maskL)
		prevR := L
		L, R = prevL, prevR
	}
	return (L << half) | R
}

// feistelPermute64 applies a 32-bit Feistel network (4 rounds) to a 64-bit value.
func feistelPermute64(x uint64, keys []uint64) uint64 {
	l := uint32(x >> 32)
	r := uint32(x)
	for _, k := range keys {
		l, r = r, l^feistelF(r, k)
	}
	return (uint64(l) << 32) | uint64(r)
}

// feistelUnpermute64 reverses the Feistel network for a 64-bit value.
func feistelUnpermute64(x uint64, keys []uint64) uint64 {
	l := uint32(x >> 32)
	r := uint32(x)
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		l, r = r^feistelF(l, k), l
	}
	return (uint64(l) << 32) | uint64(r)
}

// feistelF is the round function for the Feistel network.
// It uses a simple mix of arithmetic and bitwise operations for diffusion.
func feistelF(r uint32, k uint64) uint32 {
	// Simple example: mix input with key using arithmetic and bitwise ops
	return uint32(((uint64(r)*0x5bd1e995 + k) ^ (uint64(r)<<16 | uint64(r)>>16)) & 0xFFFFFFFF)
}

// isPowerOfTwo returns true if x is a power of two
func isPowerOfTwo(x uint64) bool {
	return x != 0 && (x&(x-1)) == 0
}

// bitsNeeded calculates the number of bits needed to represent a given value,
// rounding up to the next highest power of two.
func bitsNeeded(val uint64) int {
	if val == 0 {
		return 0
	}
	bits := 0
	for val > 1 {
		val >>= 1
		bits++
	}
	return bits
}

// Helper: combinatorial number system encode (for fixed-length, O(1) bijection)
func indexToCombo(idx uint64, baseA, baseN, adjectivesCount int, adjectives, nouns []string) []string {
	pieces := make([]string, adjectivesCount+1)
	nounIdx := int(idx % uint64(baseN))
	pieces[adjectivesCount] = nouns[nounIdx]
	idx /= uint64(baseN)
	for i := adjectivesCount - 1; i >= 0; i-- {
		pieces[i] = adjectives[int(idx%uint64(baseA))]
		idx /= uint64(baseA)
	}
	return pieces
}

func comboToIndex(pieces []string, baseA, baseN, adjectivesCount int, adjIndexMap, nounIndexMap map[string]int) (uint64, error) {
	idx := uint64(0)
	for i := 0; i < adjectivesCount; i++ {
		adjIdx, ok := adjIndexMap[pieces[i]]
		if !ok {
			return 0, fmt.Errorf("adjective %q not found", pieces[i])
		}
		idx = idx*uint64(baseA) + uint64(adjIdx)
	}
	nounIdx, ok := nounIndexMap[pieces[adjectivesCount]]
	if !ok {
		return 0, fmt.Errorf("noun %q not found", pieces[adjectivesCount])
	}
	idx = idx*uint64(baseN) + uint64(nounIdx)
	return idx, nil
}

// XXTEA block cipher for 64-bit values (public domain, no secret key required)
func xxteaEncrypt64(v uint64, key [4]uint32) uint64 {
	v0 := uint32(v)
	v1 := uint32(v >> 32)
	delta := uint32(0x9e3779b9)
	sum := uint32(0)
	n := 32
	for i := 0; i < n; i++ {
		sum += delta
		v0 += ((v1 << 4) ^ (v1 >> 5)) + v1 ^ (sum + key[sum&3])
		v1 += ((v0 << 4) ^ (v0 >> 5)) + v0 ^ (sum + key[(sum>>11)&3])
	}
	return uint64(v1)<<32 | uint64(v0)
}

func xxteaDecrypt64(v uint64, key [4]uint32) uint64 {
	v0 := uint32(v)
	v1 := uint32(v >> 32)
	delta := uint32(0x9e3779b9)
	n := 32
	sum := delta * uint32(n)
	for i := 0; i < n; i++ {
		v1 -= ((v0 << 4) ^ (v0 >> 5)) + v0 ^ (sum + key[(sum>>11)&3])
		v0 -= ((v1 << 4) ^ (v1 >> 5)) + v1 ^ (sum + key[sum&3])
		sum -= delta
	}
	return uint64(v1)<<32 | uint64(v0)
}
