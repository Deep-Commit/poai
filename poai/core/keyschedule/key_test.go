package keyschedule_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"poai/core/header"
	"poai/core/keyschedule"
)

// dummyDB satisfies storage.Reader for tests.
type dummyDB struct{ hdrs map[uint64]*header.Header }

func (d *dummyDB) HeaderByHeight(h uint64) *header.Header { return d.hdrs[h] }
func (d *dummyDB) Height() uint64 {
	// Return the highest height in the map
	max := uint64(0)
	for h := range d.hdrs {
		if h > max {
			max = h
		}
	}
	return max
}

func TestEpochKeyGolden(t *testing.T) {
	db := &dummyDB{hdrs: map[uint64]*header.Header{}}
	// Build 2 epochs, EpochBlocks = 20 -> heights 0..39
	for i := uint64(0); i < 40; i++ {
		db.hdrs[i] = &header.Header{Height: i}
	}

	got := keyschedule.EpochKey(0, db) // epoch 0 uses height 19
	// Pre-computed hex of epoch 0 with dummy header hash scheme.
	want, _ := hex.DecodeString("8beeab86a9ed9fe9457a0cea1ab78c65aad8bb2775bed6fc724c98d74763b8c0")

	if !bytes.Equal(got[:], want) {
		t.Fatalf("mismatch:\n got  %x\n want %x", got, want)
	}
}
