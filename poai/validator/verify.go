// Package validator implements the CPU replay path for POAI.
package validator

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"poai/core/config"
	"poai/core/keyschedule"
	"poai/core/storage"
	"poai/dataset"
	"poai/inference"
)

// Remove flag definitions
// var useProcedural = flag.Bool("use-procedural", false, "Use procedural dataset generation")
// var proceduralBatchSize = flag.Int("procedural-batch-size", 4, "Batch size for procedural dataset generation")
// var modelPath = flag.String("model-path", "models/qwen2.5-0.5b-instruct-q4k.gguf", "Path to GGUF LLM model file")
// var gpuLayers = flag.Int("gpu-layers", 0, "Number of LLM layers to offload to GPU (0=CPU only)")

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

// Refactor VerifyBlock signature
func VerifyBlock(b *Block, st storage.Reader, modelPath string, gpuLayers int) error {
	// Remove flag.Parse() and use parameters
	// if *useProcedural {
	// 	dataset.SetProcedural(true, *proceduralBatchSize)
	// }
	llm, err := inference.NewLLM(modelPath, gpuLayers)
	if err != nil {
		return fmt.Errorf("Failed to load LLM: %v", err)
	}
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
	// LLM inference: concatenate Qs as prompt
	prompt := ""
	for _, r := range recs {
		prompt += string(r.Q) + "\n"
	}
	llmSeed := int(binary.LittleEndian.Uint64(epochKey[:8]))
	output, err := llm.Infer(prompt, llmSeed)
	if err != nil {
		return fmt.Errorf("LLM inference failed: %v", err)
	}
	hash := sha256.Sum256([]byte(output))
	lossInt := int64(binary.LittleEndian.Uint64(hash[:8]))
	// 5. Compare ℓ̂ in header
	if lossInt != b.Header.Lhat {
		return errors.New("invalid loss")
	}
	return nil
}
