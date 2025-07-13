package inference

import (
	"github.com/go-skynet/go-llama.cpp"
)

type LLM struct {
	model *llama.LLama
}

func NewLLM(modelPath string, gpuLayers int) (*LLM, error) {
	m, err := llama.New(modelPath,
		llama.EnableF16Memory,
		llama.SetContext(2048),
		llama.SetGPULayers(gpuLayers),
	)
	if err != nil {
		return nil, err
	}
	return &LLM{model: m}, nil
}

// Infer runs inference with a deterministic seed.
func (l *LLM) Infer(prompt string, seed int) (string, error) {
	output, err := l.model.Predict(
		prompt,
		llama.SetTokens(128),
		llama.SetTopK(40),
		llama.SetTopP(0.95),
		llama.SetTemperature(0), // Deterministic
		llama.SetSeed(seed),
		llama.SetTokenCallback(func(token string) bool { return true }),
	)
	if err != nil {
		return "", err
	}
	return output, nil
}
