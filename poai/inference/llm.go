//go:build !llama
// +build !llama

package inference

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func init() {
	// Disable llama.cpp debug logs to prevent log file creation
	os.Setenv("GGML_LOG_LEVEL", "0")
}

type LLM struct{}

func NewLLM(modelPath string, gpuLayers int) (*LLM, error) {
	// Return stub implementation
	return &LLM{}, nil
}

// Infer runs a stub inference that returns a deterministic hash-based response
func (l *LLM) Infer(prompt string, seed int) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}

	// Create a deterministic response based on prompt and seed
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", prompt, seed)))
	response := fmt.Sprintf("stub_response_%x", h[:8])
	return response, nil
}
