// Package core implements consensus and state logic for POAI.
package core

import (
	"encoding/json"
	"math/big"
	"time"

	"poai/core/header"
	"poai/dataset"
	"testing"
)

// Block represents a complete POAI block with header and body.
type Block struct {
	Header   header.Header    `json:"header"`
	Records  []dataset.Record `json:"records"`
	Time     time.Time        `json:"time"`
	Receipts []byte           `json:"receipts"` // Placeholder for receipts
}

// NewBlock creates a new block with the given parameters.
func NewBlock(height uint64, parentHash [32]byte, loss int64, records []dataset.Record, parentBits *big.Int) *Block {
	return &Block{
		Header: header.Header{
			Height:     height,
			ParentHash: parentHash,
			Lhat:       loss,
			Bits:       new(big.Int).Set(parentBits), // always non-nil
			Timestamp:  time.Now(),
		},
		Records: records,
		Time:    time.Now(),
	}
}

// Hash returns the block's hash (same as header hash for now).
func (b *Block) Hash() [32]byte {
	return b.Header.Hash()
}

// Encode serializes the block to JSON for storage/transmission.
func (b *Block) Encode() ([]byte, error) {
	return json.Marshal(b)
}

// DecodeBlock deserializes a block from JSON.
func DecodeBlock(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	return &block, err
}

// Unit test: round-trip block encode/decode preserves Bits
func TestBlockBitsRoundTrip(t *testing.T) {
	b := &Block{
		Header: header.Header{
			Height:     42,
			ParentHash: [32]byte{1, 2, 3},
			Lhat:       123,
			Bits:       big.NewInt(987654321),
			Timestamp:  time.Now(),
		},
		Records: nil,
		Time:    time.Now(),
	}
	data, err := b.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	b2, err := DecodeBlock(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b2.Header.Bits == nil || b2.Header.Bits.Cmp(b.Header.Bits) != 0 {
		t.Fatalf("Bits did not survive round-trip: got %v, want %v", b2.Header.Bits, b.Header.Bits)
	}
}

// ... block logic will go here ...
