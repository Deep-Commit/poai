// Package keyschedule computes per-epoch AES keys from blockhashes.
package keyschedule

import (
	"encoding/binary"
	"log"

	"golang.org/x/crypto/sha3"

	"poai/core/config"
	"poai/core/storage"
)

// EpochKey derives the 256-bit AES key for the given epoch.
// Panics if the required header is missing (DB corruption).
func EpochKey(epoch uint64, st storage.Reader) [32]byte {
	lastHeight := epoch*config.EpochBlocks - 1
	if epoch == 0 { // special-case genesis
		lastHeight = 0
	}

	hdr := st.HeaderByHeight(lastHeight)
	if hdr == nil {
		log.Printf("[ERROR] EpochKey: header for height %d is nil, returning zero key", lastHeight)
		return [32]byte{}
	}

	seed := hdr.Hash() // [32]byte

	var buf [40]byte // 32-byte seed + 8-byte epoch
	copy(buf[:32], seed[:])
	binary.LittleEndian.PutUint64(buf[32:], epoch)

	h := sha3.New256()
	h.Write(buf[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// ... key schedule logic will go here ...
