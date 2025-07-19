package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"poai/core"
	"poai/core/config"
	"poai/miner"
	"poai/net"

	"runtime/debug"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	// Handle CLI commands first
	handleCLICommands()

	// Disable llama.cpp debug logs to prevent log file creation
	os.Setenv("GGML_LOG_LEVEL", "0")

	var (
		target        = flag.Int64("target", -1000000000000000000, "Mining difficulty target (more negative = harder)")
		epochBlocks   = flag.Uint64("epoch-blocks", 20, "Blocks per epoch")
		batchSize     = flag.Int("batch-size", 2, "Records per batch")
		dataDir       = flag.String("data-dir", "data", "Directory for chain data")
		pruneDepth    = flag.Uint64("prune-depth", 0, "Blocks to keep (0 = keep all, disables pruning)")
		p2pPort       = flag.Int("p2p-port", 4001, "P2P listen port")
		peerMultiaddr = flag.String("peer-multiaddr", "", "Multiaddr of peer to connect to (optional)")
		modelPath     = flag.String("model-path", "models/qwen2.5-0.5b-instruct-q4k.gguf", "Path to GGUF LLM model file")
		gpuLayers     = flag.Int("gpu-layers", 0, "Number of LLM layers to offload to GPU (0=CPU only)")
		minerAddress  = flag.String("miner-address", "", "Miner address (hex) for block rewards")
	)
	flag.Parse()

	// Set config from flags
	config.EpochBlocks = *epochBlocks
	config.BatchSize = *batchSize
	config.PruneDepth = *pruneDepth

	log.Printf("Starting POAI daemon...")
	log.Printf("Config: EpochBlocks=%d, BatchSize=%d, PruneDepth=%d",
		config.EpochBlocks, config.BatchSize, config.PruneDepth)
	log.Printf("Mining target: %d", *target)

	// Open chain
	chain := core.NewChain(*dataDir, int64(*target))

	// FULL REINDEX from DB before starting anything else
	if err := chain.ReindexFromDB(); err != nil {
		log.Fatalf("[FATAL] Failed to reindex chain from DB: %v", err)
	}
	chain.LogDiagnostics()

	// If orphan pool is non-empty after reindex, log and scan
	if len(chain.OrphanPool) > 0 {
		log.Printf("[WARN] Orphan pool non-empty after reindex: %d orphans", len(chain.OrphanPool))
		chain.ScanOrphanPool()
	}

	// Now start networking, mining, orphan pool scanner, etc.
	// Initialize local broadcaster
	blocksDir := filepath.Join(*dataDir, "blocks")
	broadcaster := core.NewLocalBroadcaster(blocksDir, chain)

	// Start P2P node
	ctx := context.Background()
	node, err := net.NewP2PNode(ctx, *p2pPort, chain)
	if err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	log.Printf("P2P node started. Peer ID: %s", node.Host.ID())
	for _, addr := range node.Host.Addrs() {
		log.Printf("Listening on: %s/p2p/%s", addr, node.Host.ID())
	}

	// Wire up orphan pool parent request callback
	chain.RequestBlockByHash = node.RequestBlockByHash

	// Start orphan pool auto-cleaner
	stopScan := make(chan struct{})
	chain.StartOrphanPoolScanner(30*time.Second, stopScan)

	// Manual peer connect if provided
	if *peerMultiaddr != "" {
		log.Printf("[P2P] Attempting to connect to peer: %s", *peerMultiaddr)
		addr, err := ma.NewMultiaddr(*peerMultiaddr)
		if err != nil {
			log.Fatalf("Invalid multiaddr: %v", err)
		}
		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Fatalf("Invalid AddrInfo: %v", err)
		}
		if err := node.Host.Connect(ctx, *pi); err != nil {
			log.Printf("[P2P] Failed to connect to peer: %v", err)
		} else {
			log.Printf("[P2P] Connected to peer: %s", pi.ID.String())
		}
	}

	// Announce new heads after each block is accepted
	headCh := chain.SubscribeToHeadChanges()
	go func() {
		var lastHeight uint64 = 0
		for range headCh {
			h := chain.CurrentHeight()
			if h == lastHeight {
				continue // avoid duplicate publish
			}
			lastHeight = h
			blk := chain.BlockByHeight(h)
			if blk == nil {
				continue
			}
			node.AnnounceHead(blk)
			// Log diagnostics after head change (sync/reorg)
			chain.LogDiagnostics()
		}
	}()

	// TODO: On new block mined/imported, serialize and call node.PublishBlock
	// TODO: On receiving a block, deserialize and call chain.ImportBlock

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Remove the syncCtl and pause/resume goroutine for now
	// syncCtl := miner.NewSyncControl()
	// importing := false
	// bestKnown := func() uint64 { return chain.CurrentHeight() }
	// chainHeadCh := chain.SubscribeToHeadChanges()
	// go func() {
	// 	for range chainHeadCh {
	// 		if importing {
	// 			syncCtl.PauseCh <- true
	// 			for bestKnown() < chain.CurrentHeight() {
	// 				time.Sleep(100 * time.Millisecond)
	// 			}
	// 			syncCtl.PauseCh <- false
	// 			importing = false
	// 		}
	// 	}
	// }()

	// Start block processing in a goroutine
	go func() {
		broadcaster.ProcessBlocks()
	}()

	// Start mining in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[MINER] PANIC: %v\n%s", r, debug.Stack())
			}
		}()
		// modelPath and gpuLayers are parsed here for LLM integration in miner/validator
		_ = modelPath
		_ = gpuLayers
		miner.WorkLoop(chain, *target, broadcaster, node, *modelPath, *gpuLayers, *minerAddress)
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Shutting down...")
	close(stopScan)
}
