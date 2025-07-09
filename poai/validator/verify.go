// Package validator implements the CPU replay path for POAI.
package validator

import (
	"errors"
	"fmt"

	"poai/core/config"
	"poai/core/keyschedule"
	"poai/core/storage"
	"poai/dataset"
)

type Block struct {
	Height uint64
	Header struct {
		Lhat int64
	}
}

// Dummy stubs for forwardPass and modelWeights
var modelWeights interface{}

func forwardPass([]dataset.Record, interface{}) float64 { return 0 }
func lossToInt(loss float64) int64                      { return int64(loss) }

// TestTinyWeights is a trivial 1-B parameter slice for unit tests.
var TestTinyWeights = []byte{0}

// ForwardPass is exported for tests.
func ForwardPass(records []dataset.Record, weights interface{}) float64 { return 0 }

// LossToInt is exported for tests.
func LossToInt(loss float64) int64 { return int64(loss) }

func VerifyBlock(b *Block, st storage.Reader) error {
	// 1. Epoch maths
	epoch := b.Height / config.EpochBlocks
	seed := st.HeaderByHeight((epoch+1)*config.EpochBlocks - 1).Hash()
	epochKey := keyschedule.EpochKey(epoch, st)

	// 2. Which indices does this block claim?
	idx := dataset.Indexes(seed, config.BatchSize) // K=2 or 4 from TOML

	// 3. Pull & decrypt the records
	recs, err := dataset.Fetch(idx, epochKey)
	if err != nil {
		return fmt.Errorf("dataset fetch: %w", err)
	}

	// 4. Run forward pass (existing CUDA/WASM call)
	loss := forwardPass(recs, modelWeights)

	// 5. Compare ℓ̂ in header
	if lossToInt(loss) != b.Header.Lhat {
		return errors.New("invalid loss")
	}
	return nil
}
