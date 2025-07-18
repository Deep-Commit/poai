package config

import (
	"math/big"
)

// EpochBlocks is injected at program startup from TOML.
// Default for unit tests = 20 (Testnet-0).
var EpochBlocks uint64 = 20

// CorpusSize is no longer used with procedural generation
// Keeping for backward compatibility but setting to 0
var CorpusSize uint64 = 0

// BatchSize is injected at program startup from TOML.
// Default for unit tests = 2 (Testnet-0).
var BatchSize int = 2

// Difficulty retarget parameters
const (
	RetargetInterval      = 2016 // # of blocks between adjustments
	TargetBlockSpacingSec = 600  // desired seconds per block (10 minutes)
	MaxAdjustmentFactor   = 4    // clamp A / B to [1/4, 4×]
)

// MaximumTarget is the easiest possible target (highest value)
var MaximumTarget = new(big.Int).Lsh(big.NewInt(1), 256).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// PruneDepth controls how many blocks to keep (0 = keep all, i.e., archival node)
var PruneDepth uint64 = 100
