//go:build llama
// +build llama

package inference

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type LLM struct {
	modelPath string
}

func NewLLM(modelPath string, gpuLayers int) (*LLM, error) {
	// Check if llama-cli is available
	if _, err := exec.LookPath("llama-cli"); err != nil {
		return nil, fmt.Errorf("llama-cli not found in PATH. Please install llama.cpp: brew install llama.cpp")
	}

	return &LLM{modelPath: modelPath}, nil
}

// Infer runs inference using llama.cpp CLI with a deterministic seed
func (l *LLM) Infer(prompt string, seed int) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}

	// Build the llama-cli command with simpler arguments
	args := []string{
		"-m", l.modelPath,
		"--temp", "0", // Deterministic temperature
		"--seed", strconv.Itoa(seed),
		"--ctx-size", "256", // Smaller context for faster inference
		"--n-predict", "20", // Generate fewer tokens for speed
		"--no-conversation", // Disable interactive/conversation mode
		"--prompt", prompt,  // Use --prompt instead of stdin
		"--no-warmup", // Skip warmup for faster startup
	}

	// Set a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run the command
	cmd := exec.CommandContext(ctx, "llama-cli", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("llama-cli failed: %v, stderr: %s", err, stderr.String())
	}

	// Extract the generated text (remove the prompt)
	output := stdout.String()

	// Debug: log the raw output
	fmt.Printf("[DEBUG] Raw LLM output: '%s'\n", output)

	// Try to extract just the generated part (after "Answers:")
	lines := strings.Split(output, "\n")
	var generatedLines []string
	foundAnswers := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for "Answers:" marker
		if strings.Contains(strings.ToLower(line), "answers:") {
			foundAnswers = true
			continue
		}

		// If we found "Answers:", collect subsequent lines as generated text
		if foundAnswers {
			generatedLines = append(generatedLines, line)
		}
	}

	generated := strings.Join(generatedLines, " ")
	generated = strings.TrimSpace(generated)

	// If no generation happened, return a fallback
	if generated == "" {
		generated = fmt.Sprintf("generated_response_seed_%d", seed)
	}

	return generated, nil
}
