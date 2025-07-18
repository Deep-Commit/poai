// Package core implements consensus and state logic for POAI.
package core

import (
	"fmt"
	"math/big"
	"time"

	"poai/core/config"
	"poai/core/header"
)

// ChainReader interface for fetching historic headers
type ChainReader interface {
	HeaderByHeight(height uint64) *header.Header
	Height() uint64
}

// Always use new(big.Int) or big.NewInt(0) for any *big.Int you intend to mutate.
// Never declare var x *big.Int and then call x.Set(...), as this will panic.
// Defensive: always return a non-nil *big.Int on error.
func Adjust(chain ChainReader, tip *header.Header) (*big.Int, error) {
	if tip == nil {
		return big.NewInt(1), fmt.Errorf("Adjust: nil header")
	}
	if tip.Bits == nil {
		return big.NewInt(1), fmt.Errorf("Adjust: header Bits nil at height %d", tip.Height)
	}
	interval := uint64(config.RetargetInterval)
	if tip.Height < interval {
		// Not enough history yet; return genesis target unmodified.
		return new(big.Int).Set(tip.Bits), nil
	}

	// 1) Locate the first header in this window
	firstHeight := tip.Height - interval + 1
	first := chain.HeaderByHeight(firstHeight)
	if first == nil {
		// If we can't find the required header, just return unchanged target
		return new(big.Int).Set(tip.Bits), fmt.Errorf("Adjust: missing header at height %d", firstHeight)
	}

	// 2) Compute actual timespan
	actual := tip.Timestamp.Sub(first.Timestamp)
	expected := time.Duration(interval) * time.Second * config.TargetBlockSpacingSec

	// 3) Clamp actual to [expected/MaxFactor, expected×MaxFactor]
	minSpan := expected / config.MaxAdjustmentFactor
	maxSpan := expected * config.MaxAdjustmentFactor
	if actual < minSpan {
		actual = minSpan
	} else if actual > maxSpan {
		actual = maxSpan
	}

	// 4) Scale the previous target
	// newT = oldT × actual / expected
	oldT := new(big.Int).Set(tip.Bits)
	expectedSeconds := int64(expected.Seconds())
	if expectedSeconds == 0 {
		// Avoid division by zero - use a minimum of 1 second
		expectedSeconds = 1
	}
	newT := new(big.Int).Mul(oldT, big.NewInt(int64(actual.Seconds())))
	newT = newT.Div(newT, big.NewInt(expectedSeconds))

	// 5) Enforce some sanity bounds for negative targets
	// For PoAI, we use negative targets where more negative = harder
	// Set minimum target (easiest) to -1, maximum target (hardest) to -2^63
	minTarget := big.NewInt(-1)
	maxTarget := new(big.Int).Lsh(big.NewInt(1), 63).Neg(new(big.Int).Lsh(big.NewInt(1), 63)) // -2^63

	if newT.Cmp(minTarget) > 0 {
		newT.Set(minTarget) // Don't allow positive targets
	}
	if newT.Cmp(maxTarget) < 0 {
		newT.Set(maxTarget) // Don't allow targets more negative than -2^63
	}

	return newT, nil
}
