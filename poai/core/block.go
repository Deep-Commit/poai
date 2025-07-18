// Package core implements consensus and state logic for POAI.
package core

import (
	"encoding/json"
	"math/big"
	"time"

	"poai/core/header"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// Constants for block subsidies
const (
	InitialSubsidy = 50     // Initial block subsidy in POAI
	HalvingBlocks  = 210000 // Blocks between halvings (like Bitcoin)
)

// Block represents a complete POAI block with header and body.
type Block struct {
	Header       header.Header  `json:"header"`
	Transactions []*Transaction `json:"transactions"`
	MerkleRoot   []byte         `json:"merkleRoot"`
	Time         time.Time      `json:"time"`
	Receipts     []byte         `json:"receipts"` // Placeholder for receipts
}

// NewBlock creates a new block with the given parameters.
func NewBlock(height uint64, parentHash [32]byte, loss int64, parentBits *big.Int, txs []*Transaction, nonce uint64) *Block {
	block := &Block{
		Header: header.Header{
			Height:     height,
			ParentHash: parentHash,
			Lhat:       loss,
			Bits:       new(big.Int).Set(parentBits), // always non-nil
			Timestamp:  time.Now(),
			Nonce:      nonce,
		},
		Transactions: txs,
		Time:         time.Now(),
	}

	// Calculate merkle root for transactions
	block.MerkleRoot = block.CalculateMerkleRoot()

	return block
}

// Hash returns the block's hash (same as header hash for now).
func (b *Block) Hash() [32]byte {
	return b.Header.Hash()
}

// CalculateMerkleRoot calculates the merkle root of all transactions
func (b *Block) CalculateMerkleRoot() []byte {
	if len(b.Transactions) == 0 {
		return []byte{} // Empty merkle root for blocks with no transactions
	}

	// Simple merkle root: concatenate all transaction hashes and hash the result
	var hashes []byte
	for _, tx := range b.Transactions {
		if len(tx.Hash) == 0 {
			tx.Hash = tx.CalculateHash()
		}
		hashes = append(hashes, tx.Hash...)
	}

	// Use keccak256 for EVM compatibility
	return crypto.Keccak256(hashes)
}

// GetSubsidy calculates the block subsidy for a given height
func GetSubsidy(height uint64) *big.Int {
	halvings := height / HalvingBlocks
	if halvings >= 64 {
		return big.NewInt(0) // No more subsidies after 64 halvings
	}

	subsidy := big.NewInt(InitialSubsidy)
	subsidy.Rsh(subsidy, uint(halvings)) // Right shift by halvings (divide by 2^halvings)
	return subsidy
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
			Nonce:      12345,
		},
		Time: time.Now(),
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
