# HumanID

HumanID is a Go package for generating human-readable, unique, and reversible IDs from numbers. It is designed for SaaS products, databases, and anywhere you want to map numeric or database IDs to friendly, memorable strings (e.g., `234 -> "graceful-experimental-monkey-41"`).

- **Bijective**: Every number maps to a unique human ID, and every human ID decodes to exactly one number.
- **Fast**: O(1) encode/decode for all valid inputs.
- **Scrambled/Unpredictable**: Optionally scramble IDs using a public-key block cipher (XXTEA) for non-sequential, unguessable IDs.
- **Open Source**: No secrets, no vendor lock-in, forever free.

**Note:** if the dictionary ever changes the generated conversion maps will also be off as the randomness will shuffle it differently now, i am working on making the indexes baked to a file with only new word being reshuffled.

## Installation

```
go get github.com/Reboxed/HumanID
```

## Usage

### Basic Encoding/Decoding

Loading the generator is an expensive process, please attempt to reuse the same generator as much as possible especially for web applications.

```go
import (
    "fmt"
    "log"
    "github.com/Reboxed/HumanID"
)

func main() {
    generator, err := HumanID.Load(100) // 100 is the seed
    if err != nil {
        log.Fatal(err)
    }
    id, err := generator.Encode(436436, 2) // 2 adjectives
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Generated HID: %s\n", id)
    decoded, err := generator.Decode(id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Decoded: %d\n", decoded)
}
```

### Scrambled (Unpredictable) IDs

For scrambled IDs, we do not recommend fully relying on decoding, as it can sometimes result in duplicate Human IDs — an issue that does not occur with sequential IDs. In such cases, it's safer to use sequential IDs and generate a random number based on MaxCombinations on your server, check if that key was already saved in the DB, regenerate if so, and then to use that number with the encoder before storing it in a database or similar system, as this approach is more reliable and secure.

```go
import (
    "fmt"
    "log"
    "github.com/Reboxed/HumanID"
)

func main() {
    key := [4]uint32{0x12345678, 0x9abcdef0, 0x0fedcba9, 0x87654321} // public key
    generator, err := HumanID.Load(100, key)
    if err != nil {
        log.Fatal(err)
    }
    id, err := generator.EncodeScrambled(436436, 2)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Scrambled HID: %s\n", id)
    decoded, err := generator.DecodeFromScrambled(id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Decoded: %d\n", decoded)
}
```

## Examples

See the [`examples/`](examples/) folder for runnable code:
- [`examples/basic.go`](examples/basic.go): Basic encode/decode
- [`examples/scrambled.go`](examples/scrambled.go): Scrambled encode/decode with XXTEA

## API

### Load

```
Load(seed int64, xxteaKey ...[4]uint32) (*Generator, error)
```
- `seed`: Shuffle the wordlists for uniqueness. Use the same seed for consistent encoding/decoding.
- `xxteaKey`: (Optional) 4-element array for XXTEA block cipher scrambling. Public, not secret.

### Encode / Decode

```
Encode(index uint64, adjectivesCount int) (string, error)
Decode(humanID string) (uint64, error)
```
- Maps a number to a human-readable ID and back. Bijection is guaranteed.

### EncodeScrambled / DecodeFromScrambled

```
EncodeScrambled(index uint64, adjectivesCount int) (string, error)
DecodeFromScrambled(humanID string) (uint64, error)
```
- Scrambles the mapping using XXTEA for unguessable, non-sequential IDs. Still bijective and reversible.

## Wordlists

- Place your `adjectives.txt` and `nouns.txt` in the same directory as your binary or in the package root.
- Each file should contain one word per line, lowercase, alphanumeric.

## License

MIT. See [LICENSE](LICENSE).

---

**Developed and maintained by Rebxd. Contributions welcome!**
