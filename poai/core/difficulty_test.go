package core

import (
	"math/big"
	"testing"
	"time"

	"poai/core/header"
)

// mockChain implements ChainReader for testing
type mockChain struct {
	headers map[uint64]*header.Header
	height  uint64
}

func (m *mockChain) HeaderByHeight(height uint64) *header.Header {
	return m.headers[height]
}

func (m *mockChain) Height() uint64 {
	return m.height
}

func TestDifficultyAdjust(t *testing.T) {
	// Create a mock chain with synthetic timestamps
	chain := &mockChain{
		headers: make(map[uint64]*header.Header),
		height:  2016, // At retarget interval
	}

	// Create headers with synthetic timestamps
	// Simulate blocks coming too fast (every 5 minutes instead of 10)
	baseTime := time.Now()
	for i := uint64(0); i <= 2016; i++ {
		// Each block 5 minutes apart (should result in difficulty increase)
		blockTime := baseTime.Add(time.Duration(i) * 5 * time.Minute)
		chain.headers[i] = &header.Header{
			Height:    i,
			Bits:      big.NewInt(1000), // Initial target
			Timestamp: blockTime,
		}
	}

	// Test difficulty adjustment
	tip := chain.headers[2016]
	newTarget, err := Adjust(chain, tip)
	if err != nil {
		t.Fatalf("Adjust failed: %v", err)
	}

	// Since blocks came too fast (5 min vs 10 min expected), difficulty should increase
	// This means the target should decrease
	if newTarget.Cmp(big.NewInt(1000)) >= 0 {
		t.Errorf("Expected target to decrease (difficulty increase), got %d", newTarget)
	}

	t.Logf("Original target: 1000, New target: %d", newTarget)
}

func TestDifficultyAdjustClamping(t *testing.T) {
	// Test that extreme time differences are clamped
	chain := &mockChain{
		headers: make(map[uint64]*header.Header),
		height:  2016,
	}

	baseTime := time.Now()
	for i := uint64(0); i <= 2016; i++ {
		// Simulate extremely fast blocks (1 second apart)
		blockTime := baseTime.Add(time.Duration(i) * time.Second)
		chain.headers[i] = &header.Header{
			Height:    i,
			Bits:      big.NewInt(1000),
			Timestamp: blockTime,
		}
	}

	tip := chain.headers[2016]
	newTarget, err := Adjust(chain, tip)
	if err != nil {
		t.Fatalf("Adjust failed: %v", err)
	}

	// Should be clamped to MaxAdjustmentFactor (4x)
	expectedMin := new(big.Int).Div(big.NewInt(1000), big.NewInt(4))
	if newTarget.Cmp(expectedMin) < 0 {
		t.Errorf("Target should be clamped to minimum %d, got %d", expectedMin, newTarget)
	}

	t.Logf("Clamped target: %d", newTarget)
}

func TestDifficultyAdjustInsufficientHistory(t *testing.T) {
	// Test that blocks with insufficient history return unchanged target
	chain := &mockChain{
		headers: make(map[uint64]*header.Header),
		height:  1000, // Less than RetargetInterval
	}

	baseTime := time.Now()
	for i := uint64(0); i <= 1000; i++ {
		blockTime := baseTime.Add(time.Duration(i) * 10 * time.Minute)
		chain.headers[i] = &header.Header{
			Height:    i,
			Bits:      big.NewInt(1000),
			Timestamp: blockTime,
		}
	}

	tip := chain.headers[1000]
	newTarget, err := Adjust(chain, tip)
	if err != nil {
		t.Fatalf("Adjust failed: %v", err)
	}

	// Should return unchanged target
	if newTarget.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("Expected unchanged target 1000, got %d", newTarget)
	}

	t.Logf("Unchanged target: %d", newTarget)
}
