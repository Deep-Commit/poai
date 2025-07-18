// Package header defines the canonical block header for POAI.
package header

import (
	"crypto/sha3"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/big"
	"time"
)

// Header is a *minimal* canonical representation.
// Extend with parentHash, merkleRoot, etc. as you flesh out the chain.
type Header struct {
	Height     uint64
	ParentHash [32]byte
	Lhat       int64
	Bits       *big.Int `json:"bits,string"`
	Timestamp  time.Time
	StateRoot  [32]byte // Placeholder for state trie root
	Nonce      uint64   `json:"nonce"` // Mining nonce for probabilistic search
	// Add real fields here…
}

// MarshalJSON ensures Bits is encoded as a string
func (h *Header) MarshalJSON() ([]byte, error) {
	type Alias Header
	return json.Marshal(&struct {
		Bits string `json:"bits"`
		*Alias
	}{
		Bits: func() string {
			if h.Bits != nil {
				return h.Bits.String()
			} else {
				return "0"
			}
		}(),
		Alias: (*Alias)(h),
	})
}

// UnmarshalJSON ensures Bits is decoded from a string
func (h *Header) UnmarshalJSON(data []byte) error {
	type Alias Header
	temp := &struct {
		Bits string `json:"bits"`
		*Alias
	}{
		Alias: (*Alias)(h),
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	if temp.Bits != "" {
		bi := new(big.Int)
		bi.SetString(temp.Bits, 10)
		h.Bits = bi
	} else {
		h.Bits = big.NewInt(0)
	}
	return nil
}

type Block struct {
	Header *Header
	// Add real fields here…
}

// Hash returns the Keccak-256 of the RLP-encoded header.
// For now we hash Height, ParentHash, and Nonce; swap in full RLP once ready.
func (h *Header) Hash() [32]byte {
	if h == nil {
		log.Printf("[ERROR] Header.Hash() called on nil header, returning zero hash")
		return [32]byte{}
	}
	var buf [48]byte // 8 bytes height + 32 bytes parent hash + 8 bytes nonce
	binary.LittleEndian.PutUint64(buf[:8], h.Height)
	copy(buf[8:40], h.ParentHash[:])
	binary.LittleEndian.PutUint64(buf[40:], h.Nonce)
	return sha3.Sum256(buf[:])
}

// ... header logic will go here ...
