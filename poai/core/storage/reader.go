package storage

import "poai/core/header"

// Reader is the tiny, read-only view EpochKey needs.
// Your full DB layer can satisfy it (leveldb, badger, etc.).
type Reader interface {
	// HeaderByHeight MUST return a canonical header pointer.
	// Panic or return nil if height is out of range—this is consensus-critical.
	HeaderByHeight(height uint64) *header.Header

	// Height returns the current chain height (tip).
	Height() uint64
}
