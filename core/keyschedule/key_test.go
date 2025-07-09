package keyschedule

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestEpochKey(t *testing.T) {
	got := EpochKey(0, db) // epoch 0 uses height 19
	// Pre-computed hex of epoch 0 with dummy header hash scheme.
	want, _ := hex.DecodeString("8beeab86a9ed9fe9457a0cea1ab78c65aad8bb2775bed6fc724c98d74763b8c0")

	if !bytes.Equal(got, want) {
		t.Errorf("EpochKey(0, db) = %x, want %x", got, want)
	}
}
