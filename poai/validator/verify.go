// Package validator implements the CPU replay path for POAI.
package validator

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"poai/core"
	"poai/core/storage"
	"poai/dataset"
	"poai/inference"
)

// Remove flag definitions
// var useProcedural = flag.Bool("use-procedural", false, "Use procedural dataset generation")
// var proceduralBatchSize = flag.Int("procedural-batch-size", 4, "Batch size for procedural dataset generation")
// var modelPath = flag.String("model-path", "models/qwen2.5-0.5b-instruct-q4k.gguf", "Path to GGUF LLM model file")
// var gpuLayers = flag.Int("gpu-layers", 0, "Number of LLM layers to offload to GPU (0=CPU only)")

// These functions are no longer used with procedural generation
// Keeping for backward compatibility
func lossToInt(loss float64) int64 { return int64(loss) }

// LossToInt is exported for tests.
func LossToInt(loss float64) int64 { return int64(loss) }

// VerifyBlock validates a block using the new nonce-based approach
func VerifyBlock(b *core.Block, st storage.Reader, modelPath string, gpuLayers int) error {
	llm, err := inference.NewLLM(modelPath, gpuLayers)
	if err != nil {
		return fmt.Errorf("Failed to load LLM: %v", err)
	}

	// Validate transactions first
	if len(b.Transactions) > 0 {
		// TODO: Create a temporary state for validation
		// For now, just verify transaction signatures
		for i, tx := range b.Transactions {
			if err := tx.Verify(); err != nil {
				return fmt.Errorf("transaction %d verification failed: %v", i, err)
			}
		}
	}

	// Reconstruct the procedural quiz using the block's nonce
	quizzes := dataset.ProceduralQuiz(b.Header.Height, b.Header.Nonce)

	// Create prompt from quizzes (same as mining)
	prompt := ""
	for _, quiz := range quizzes {
		prompt += quiz + "\n"
	}

	if prompt == "" {
		return fmt.Errorf("empty prompt generated from nonce %d", b.Header.Nonce)
	}

	// Run LLM inference with same seed as mining
	var heightBytes [8]byte
	binary.LittleEndian.PutUint64(heightBytes[:], b.Header.Height)
	llmSeed := int(binary.LittleEndian.Uint64(heightBytes[:]))
	output, err := llm.Infer(prompt, llmSeed)
	if err != nil {
		return fmt.Errorf("LLM inference failed: %v", err)
	}

	// Calculate loss from LLM output (same as mining)
	hash := sha256.Sum256([]byte(output))
	lossInt := int64(binary.LittleEndian.Uint64(hash[:8]))

	// Verify the loss matches the block header
	if lossInt != b.Header.Lhat {
		return fmt.Errorf("invalid loss: got %d, expected %d", lossInt, b.Header.Lhat)
	}

	// Verify the loss meets the difficulty target
	if lossInt > b.Header.Bits.Int64() {
		return fmt.Errorf("loss %d does not meet target %d", lossInt, b.Header.Bits.Int64())
	}

	return nil
}
