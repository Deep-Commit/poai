// Package miner implements the CUDA hot-path for POAI.
package miner

import (
	"log"
	"runtime"
	"time"

	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"poai/core"
	"poai/core/config"
	"poai/dataset"
	"poai/inference"
)

// Dummy stubs for forwardPass and modelWeights
var modelWeights interface{}

// These functions are no longer used with procedural generation
// Keeping for backward compatibility
func lossToInt(loss float64) int64 { return int64(loss) }

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

// WorkLoop implements Bitcoin-style probabilistic mining with nonce-based search
func WorkLoop(chain *core.Chain, target int64, broadcaster *core.LocalBroadcaster, p2pNode interface{ PublishBlockFromStruct(*core.Block) error }, modelPath string, gpuLayers int, minerAddress string) {
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

		height := parent.Height + 1
		log.Printf("⛏️  Starting mining at height %d", height)

		// Get current target (difficulty)
		currentTarget := parent.Bits.Int64()
		if currentTarget <= 0 {
			log.Printf("[BUG] parent.Bits is nil or zero! Falling back to CLI target %d", target)
			currentTarget = target
		}

		// Check if we need to retarget difficulty
		if height%config.RetargetInterval == 0 && parent.Height > 0 {
			// Use the parent header for difficulty adjustment
			if t, err := core.Adjust(chain, parent); err == nil {
				currentTarget = t.Int64()
				log.Printf("🎯 Difficulty retarget: new target = %d", currentTarget)
			} else {
				log.Printf("[WARN] Difficulty adjustment failed: %v", err)
			}
		}

		// Start probabilistic search with nonce
		nonce := uint64(0)
		tries := 0
		lastLog := time.Now()
		startTime := time.Now()

		for {
			// Generate procedural quiz based on block height and nonce
			quizzes := dataset.ProceduralQuiz(height, nonce)

			// Create prompt from quizzes - ask for answers
			prompt := "Please answer these questions:\n"
			for _, quiz := range quizzes {
				prompt += quiz + "\n"
			}
			prompt += "Answers:\n"

			if prompt == "" {
				log.Printf("Skipping LLM inference: prompt is empty")
				nonce++
				runtime.Gosched()
				continue
			}

			// Log the quiz being solved on every attempt
			log.Printf("[MINER] Solving quiz: %s", func() string {
				if len(quizzes) > 0 {
					return quizzes[0]
				}
				return "empty quiz"
			}())

			// Run LLM inference (the "work")
			// Create a deterministic seed from height
			var heightBytes [8]byte
			binary.LittleEndian.PutUint64(heightBytes[:], height)
			llmSeed := int(binary.LittleEndian.Uint64(heightBytes[:]))

			// Log LLM inference start on every attempt
			log.Printf("[MINER] 🧠 Starting LLM inference (seed=%d, nonce=%d)...", llmSeed, nonce)

			output, err := llm.Infer(prompt, llmSeed)
			if err != nil {
				log.Printf("LLM inference failed: %v", err)
				nonce++
				runtime.Gosched()
				continue
			}

			// Calculate loss from LLM output (like hash in Bitcoin)
			hash := sha256.Sum256([]byte(output))
			lossInt := int64(binary.LittleEndian.Uint64(hash[:8]))

			tries++

			// Log every attempt to show progress
			log.Printf("[MINER] Try %d: nonce=%d, loss=%d, target=%d, output='%s...'",
				tries, nonce, lossInt, currentTarget,
				func() string {
					if len(output) > 50 {
						return output[:50] + "..."
					}
					return output
				}())

			if tries%100 == 0 && time.Since(lastLog) > 5*time.Second {
				elapsed := time.Since(startTime)
				rate := float64(tries) / elapsed.Seconds()
				log.Printf("[MINER] %.1f attempts/sec, %d tries, elapsed: %v", rate, tries, elapsed)
				lastLog = time.Now()
			}

			// Check if we found a valid block (loss <= target)
			if lossInt <= currentTarget {
				log.Printf("🎉 BLOCK FOUND! Loss: %d <= Target: %d after %d tries", lossInt, currentTarget, tries)
				log.Printf("⏱️  Mining time: %v", time.Since(startTime))

				// Get transactions from mempool
				transactions := chain.Mempool.GetTransactionsForBlock(100) // Max 100 txs per block

				// Add coinbase transaction for miner
				var minerAddr []byte
				if minerAddress != "" {
					// Parse the hex address
					if addrBytes, err := hex.DecodeString(minerAddress); err == nil {
						minerAddr = addrBytes
					} else {
						log.Printf("[WARN] Invalid miner address %s, using default", minerAddress)
						minerAddr = []byte("miner-address-12345678901234567890123456789012")
					}
				} else {
					minerAddr = []byte("miner-address-12345678901234567890123456789012")
				}
				subsidy := core.GetSubsidy(height)
				coinbaseTx := core.NewCoinbaseTx(minerAddr, subsidy)
				transactions = append([]*core.Transaction{coinbaseTx}, transactions...)

				log.Printf("💰 Including %d transactions (1 coinbase + %d mempool)", len(transactions), len(transactions)-1)

				// Create block with nonce
				block := core.NewBlock(height, parent.Hash(), lossInt, parent.Bits, transactions, nonce)
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
				break // break out of nonce loop, restart mining with new parent
			}

			// Increment nonce for next attempt
			nonce++
			runtime.Gosched()

			// Check for head changes (other miners found blocks)
			select {
			case <-headChangeCh:
				// Got a new canonical head -> update parent and start fresh
				newParent := chain.HeaderByHeight(chain.Height())
				if newParent != nil && newParent.Height > parent.Height {
					parent = newParent
					log.Printf("📈 Chain advanced to height %d, mining template invalidated, starting fresh", parent.Height)
					goto restart_mining
				}
			default:
				// No head change, continue mining
			}
		}

	restart_mining:
		continue
	}
}
