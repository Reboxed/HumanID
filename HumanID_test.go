package HumanID

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	g, err := Load(12345)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	adjectivesCount := 2
	// Test a series of sample indices to ensure bijection
	for idx := uint64(0); idx < 1000; idx += 137 {
		id, err := g.Encode(idx, adjectivesCount)
		if err != nil {
			t.Errorf("Encode error at %d: %v", idx, err)
			continue
		}
		dec, err := g.Decode(id)
		if err != nil {
			t.Errorf("Decode error for id %q: %v", id, err)
			continue
		}
		if dec != idx {
			t.Errorf("Value mismatch at %d: got %d for id %q", idx, dec, id)
		}
	}
}

func TestEncodeDecodeScrambled(t *testing.T) {
	key := [4]uint32{0x12345678, 0x9abcdef0, 0x0fedcba9, 0x87654321}
	g, err := Load(54321, key)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	adjectivesCount := 2
	// Test a series of sample indices to ensure bijection for scrambled encoding
	for idx := uint64(0); idx < 1000; idx += 113 {
		hid, err := g.EncodeScrambled(idx, adjectivesCount)
		if err != nil {
			t.Errorf("EncodeScrambled error at %d: %v", idx, err)
			continue
		}
		dec, err := g.DecodeFromScrambled(hid)
		if err != nil {
			t.Errorf("DecodeFromScrambled error for id %q: %v", hid, err)
			continue
		}
		if dec != idx {
			t.Errorf("Scrambled value mismatch at %d: got %d for id %q", idx, dec, hid)
		}
	}
}
