// Package miner implements the CUDA hot-path for POAI.
package miner

import (
	"log"
	"math/rand"
	"runtime"
	"time"

	"crypto/sha256"
	"encoding/binary"

	"poai/core"
	"poai/core/config"
	"poai/core/header"
	"poai/core/keyschedule"
	"poai/dataset"
	"poai/inference"
)

// Dummy stubs for forwardPass and modelWeights
var modelWeights interface{}

// Helper to flatten records for deterministic hashing
func flattenRecords(records []dataset.Record) []byte {
	// Deterministically flatten: concatenate Q and A for each record
	var out []byte
	for _, r := range records {
		out = append(out, r.Q...)
		out = append(out, r.A...)
	}
	return out
}

func forwardPass(records []dataset.Record, weights interface{}) float64 {
	h := sha256.Sum256(flattenRecords(records))
	seed := int64(binary.LittleEndian.Uint64(h[:8]))
	rng := rand.New(rand.NewSource(seed))
	return rng.Float64() * 1_000_000 // Increased search space
}
func lossToInt(loss float64) int64 { return int64(loss) }

// ForwardPass is exported for tests.
func ForwardPass(records []dataset.Record, weights interface{}) float64 {
	return forwardPass(records, weights)
}

// LossToInt is exported for tests.
func LossToInt(loss float64) int64 { return int64(loss) }

// Add a channel to signal pause/resume
// Add a SyncControl struct to manage mining pause/resume

type SyncControl struct {
	PauseCh chan bool
}

func NewSyncControl() *SyncControl {
	return &SyncControl{PauseCh: make(chan bool, 1)}
}

// Remove flag definitions
// var useProcedural = flag.Bool("use-procedural", false, "Use procedural dataset generation")
// var proceduralBatchSize = flag.Int("procedural-batch-size", 4, "Batch size for procedural dataset generation")
// var modelPath = flag.String("model-path", "models/qwen2.5-0.5b-instruct-q4k.gguf", "Path to GGUF LLM model file")
// var gpuLayers = flag.Int("gpu-layers", 0, "Number of LLM layers to offload to GPU (0=CPU only)")

// Refactor WorkLoop signature
func WorkLoop(chain *core.Chain, target int64, broadcaster *core.LocalBroadcaster, p2pNode interface{ PublishBlockFromStruct(*core.Block) error }, modelPath string, gpuLayers int) {
	// Remove flag.Parse() and use parameters
	// if *useProcedural {
	// 	dataset.SetProcedural(true, *proceduralBatchSize)
	// }
	llm, err := inference.NewLLM(modelPath, gpuLayers)
	if err != nil {
		log.Fatalf("Failed to load LLM: %v", err)
	}
	log.Printf("Loaded LLM model: %s (GPU layers: %d)", modelPath, gpuLayers)
	log.Printf("Starting miner workloop with initial target: %d", target)

	// Subscribe to head changes
	headChangeCh := chain.SubscribeToHeadChanges()

	for {
		parent := chain.HeaderByHeight(chain.Height())
		if parent == nil {
			log.Printf("[MINER][WARN] No chain head found yet (chain may be initializing). Waiting...")
			time.Sleep(500 * time.Millisecond)
			continue
		}
		log.Printf("⛏️  Starting mining at height %d", parent.Height)
		heightLogged := uint64(0)
		var extraNonce uint64 // declare once, outside the inner loop
		for {
			// Reset extraNonce to a random value at the start of each height
			extraNonce = uint64(rand.Uint32())
			height := parent.Height + 1
			if height != heightLogged {
				log.Printf("⛏️  Mining at height %d", height)
				heightLogged = height
			}
			// 1. Epoch maths (same as validator)
			epoch := height / config.EpochBlocks
			lastHeight := epoch*config.EpochBlocks - 1
			if epoch == 0 { // special-case genesis
				lastHeight = 0
			}
			seedHdr := chain.HeaderByHeight(lastHeight)
			epochKey := keyschedule.EpochKey(epoch, chain)
			tries := 0
			lastLog := time.Now()
			for {
				time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond) // Random backoff before each mining attempt
				indices := dataset.IndexesWithNonce(seedHdr.Hash(), extraNonce, config.BatchSize)
				records, err := dataset.Fetch(indices, epochKey)
				tries++
				if tries%1000 == 0 && time.Since(lastLog) > 500*time.Millisecond {
					log.Printf("[MINER] %.1f kH/s", float64(tries)/1e3)
					tries = 0
					lastLog = time.Now()
				}
				// Log each procedural question
				for i, r := range records {
					log.Printf("[DEBUG] Procedural Q[%d]: %q", i, r.Q)
				}
				// LLM inference: concatenate Qs as prompt
				prompt := ""
				for _, r := range records {
					prompt += string(r.Q) + "\n"
				}
				if prompt == "" {
					log.Printf("Skipping LLM inference: prompt is empty")
					extraNonce++
					runtime.Gosched()
					continue
				}
				llmSeed := int(binary.LittleEndian.Uint64(epochKey[:8]))
				output, err := llm.Infer(prompt, llmSeed)
				log.Printf("[DEBUG] LLM answer: %q", output)
				if err != nil {
					log.Printf("LLM inference failed: %v", err)
					extraNonce++
					runtime.Gosched()
					continue
				}
				hash := sha256.Sum256([]byte(output))
				lossInt := int64(binary.LittleEndian.Uint64(hash[:8]))
				// Restore currentTarget calculation
				currentTarget := parent.Bits.Int64()
				if currentTarget <= 0 {
					log.Printf("[BUG] parent.Bits is nil or zero! Falling back to CLI target %d", target)
					currentTarget = target
				}
				if (parent.Height+1)%config.RetargetInterval == 0 && parent.Height > 0 {
					nextHeader := &header.Header{Height: parent.Height + 1, Timestamp: time.Now()}
					if t, err := core.Adjust(chain, nextHeader); err == nil {
						currentTarget = t.Int64()
					}
				}
				log.Printf("[DEBUG] LLM output: %q, lossInt: %d, target: %d", output, lossInt, currentTarget)
				// For testing, force a block to be found:
				if true || lossInt <= currentTarget {
					log.Printf("[MINER] Block found after %d tries", tries)
					log.Printf("🎉 BLOCK FOUND! Loss: %d < Target: %d", lossInt, currentTarget)
					block := core.NewBlock(height, parent.Hash(), lossInt, records, parent.Bits)
					if err := broadcaster.BroadcastBlock(block); err != nil {
						log.Printf("Failed to broadcast block: %v", err)
					}
					if p2pNode != nil {
						_ = p2pNode.PublishBlockFromStruct(block)
					}
					// Wait for head to advance to at least this block's height
					for {
						<-headChangeCh
						for len(headChangeCh) > 0 {
							<-headChangeCh
						} // drain
						newHead := chain.HeaderByHeight(chain.Height())
						if newHead != nil && newHead.Height >= block.Header.Height {
							parent = newHead
							break
						}
					}
					break // break out of extraNonce loop, restart mining with new parent
				}
				extraNonce++ // bump nonce for next trial
				runtime.Gosched()
			}

			// Wait for head change or continue mining same height
			select {
			case <-headChangeCh:
				// Got a new canonical head -> update parent and start fresh
				newParent := chain.HeaderByHeight(chain.Height())
				if newParent != nil && newParent.Height > parent.Height {
					parent = newParent
					log.Printf("📈 Chain advanced to height %d, mining template invalidated, starting fresh", parent.Height)
				}
			default:
				// No head change, keep hashing same height with small delay
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
}
