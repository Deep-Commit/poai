package core

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"poai/core/config"
	"poai/core/header"
	"poai/dataset"
	"runtime"
	"sync/atomic"
)

// Chain manages the local blockchain state.
type Chain struct {
	mu             sync.RWMutex
	blocks         map[uint64]*Block
	blockHashIndex map[[32]byte]*Block // NEW: fast hash lookup
	head           uint64
	dataDir        string

	store         *BadgerStore // Persistent storage
	genesisTarget int64        // Store the initial mining target for genesis

	// Head change notifications
	headChangeCh chan struct{}
	subscribers  []chan struct{}
	subMu        sync.RWMutex

	// Orphan pool for blocks with missing parents
	OrphanPool map[[32]byte][]*Block // parentHash -> slice of orphans (exported)
	OrphanMu   sync.RWMutex          // exported

	// Side branches for blocks that extend a different parent hash
	sideBranches map[[32]byte][]*Block // fork tip hash -> branch blocks

	// Callback to request a block by parent hash from P2P
	RequestBlockByHash func(parentHash [32]byte)
}

// NewChain creates a new chain instance.
func NewChain(dataDir string, genesisTarget int64) *Chain {
	os.MkdirAll(dataDir, 0755)
	store, err := OpenBadgerStore(dataDir)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}

	chain := &Chain{
		blocks:         make(map[uint64]*Block),
		blockHashIndex: make(map[[32]byte]*Block), // NEW
		dataDir:        dataDir,
		store:          store,
		genesisTarget:  genesisTarget,
		headChangeCh:   make(chan struct{}, 16), // Buffered channel
		subscribers:    make([]chan struct{}, 0),
		OrphanPool:     make(map[[32]byte][]*Block),
		sideBranches:   make(map[[32]byte][]*Block),
	}

	// Load existing blocks from BadgerDB
	tip, err := store.GetTipHeight()
	if err == nil {
		for h := uint64(0); h <= tip; h++ {
			blk, err := store.GetBlock(h)
			if err == nil && blk != nil {
				chain.blocks[h] = blk
				chain.blockHashIndex[blk.Hash()] = blk // NEW
				if h > chain.head {
					chain.head = h
				}
			}
		}
	}

	// Initialize genesis if empty
	if len(chain.blocks) == 0 {
		chain.createGenesis()
	}

	return chain
}

// createGenesis creates the genesis block.
func (c *Chain) createGenesis() {
	genesis := &Block{
		Header: header.Header{
			Height:     0,
			ParentHash: [32]byte{}, // Zero hash for genesis
			Lhat:       0,
			Bits:       big.NewInt(c.genesisTarget), // Use the passed-in target
			Timestamp:  time.Now(),
		},
		Records: []dataset.Record{},
		Time:    time.Now(),
	}

	c.blocks[0] = genesis
	c.blockHashIndex[genesis.Hash()] = genesis // NEW
	c.head = 0
	// Persist genesis block to BadgerDB
	if err := c.store.PutBlock(0, genesis); err != nil {
		log.Printf("[ERROR] Failed to persist genesis block to BadgerDB: %v", err)
	} else {
		log.Printf("🗄️  Genesis block persisted to BadgerDB")
	}
	log.Printf("📗 Created genesis block at height 0 with target=%d", c.genesisTarget)
}

// ImportBlock validates and imports a new block.
func (c *Chain) ImportBlock(block *Block) error {
	return c.importBlockInternal(block, true)
}

// importBlockInternal allows disabling orphan pool scan to avoid recursion.
func (c *Chain) importBlockInternal(block *Block, scanOrphans bool) error {
	c.mu.Lock()
	// *** do NOT defer yet ***
	unlocked := false
	defer func() {
		if !unlocked {
			c.mu.Unlock()
		}
	}()

	// Check if block already exists
	if existing, exists := c.blocks[block.Header.Height]; exists {
		// If the incoming block is not identical, and its parent is not our head, treat as side branch
		if existing.Hash() != block.Hash() && block.Header.ParentHash != c.blocks[c.head].Hash() {
			parentHash := block.Header.ParentHash
			localHeadHash := c.blocks[c.head].Hash()
			c.addToSideBranch(block)
			log.Printf("🌿 Block #%d from peer added to side branch (parent %x, local head %x)", block.Header.Height, parentHash[:8], localHeadHash[:8])
			c.checkReorg()
			return nil
		}
		return fmt.Errorf("block at height %d already exists", block.Header.Height)
	}

	// Check if parent exists anywhere in the chain
	var parent *Block
	parentFound := false
	for _, b := range c.blocks {
		if b.Hash() == block.Header.ParentHash {
			parent = b
			parentFound = true
			break
		}
	}

	if !parentFound {
		// Add to orphan pool instead of returning error
		c.addToOrphanPool(block)
		log.Printf("🧩 Block #%d added to orphan pool (parent %x not found in chain)", block.Header.Height, block.Header.ParentHash[:8])
		return fmt.Errorf("parent block with hash %x not found, queued in orphan pool", block.Header.ParentHash)
	}

	// If parent is not at height-1, treat as side branch
	if parent.Header.Height != block.Header.Height-1 {
		c.addToSideBranch(block)
		log.Printf("🌿 Block #%d added to side branch (parent at height %d, block height %d)", block.Header.Height, parent.Header.Height, block.Header.Height)
		return fmt.Errorf("parent at height %d, block at %d: side branch", parent.Header.Height, block.Header.Height)
	}

	// Validate parent hash (should always match here)
	if block.Header.ParentHash != parent.Hash() {
		c.addToSideBranch(block)
		log.Printf("🌿 Block #%d added to side branch (parent hash mismatch)", block.Header.Height)
		return fmt.Errorf("parent hash mismatch: expected %x, got %x (side branch)", parent.Hash(), block.Header.ParentHash)
	}

	// Validate block hash
	expectedHash := block.Hash()
	if block.Header.Hash() != expectedHash {
		return fmt.Errorf("block hash mismatch")
	}

	// Check if this is a retarget block and adjust difficulty
	if block.Header.Height%uint64(config.RetargetInterval) == 0 && block.Header.Height > 0 {
		log.Printf("🔧 Attempting difficulty retarget at height %d", block.Header.Height)

		// Temporarily release the lock to avoid deadlock during difficulty adjustment
		c.mu.Unlock()
		newTarget, err := Adjust(c, &block.Header)
		c.mu.Lock() // Re-acquire the lock

		if err != nil {
			log.Printf("❌ Difficulty adjustment failed: %v", err)
			return fmt.Errorf("difficulty adjustment failed: %w", err)
		}
		block.Header.Bits = newTarget
		log.Printf("🎯 Difficulty retarget at height %d: new target = %d", block.Header.Height, newTarget)
	} else {
		// Use parent's target for non-retarget blocks
		block.Header.Bits = parent.Header.Bits
	}

	// Import the block
	c.blocks[block.Header.Height] = block
	c.blockHashIndex[block.Hash()] = block // NEW
	c.head = block.Header.Height
	if err := c.store.PutBlock(block.Header.Height, block); err != nil {
		log.Printf("Failed to persist block %d: %v", block.Header.Height, err)
	} else {
		log.Printf("🗄️  Block #%d persisted to BadgerDB", block.Header.Height)
	}

	// Prune old blocks (if enabled)
	if config.PruneDepth > 0 {
		if err := c.store.PruneBlocks(config.PruneDepth, c.head); err == nil {
			log.Printf("🧹 Pruned blocks below height %d", int64(c.head)-int64(config.PruneDepth)+1)
		}
	}

	log.Printf("📗 Accepted block #%d loss=%d target=%d", block.Header.Height, block.Header.Lhat, block.Header.Bits)

	// Notify subscribers of head change
	c.notifyHeadChange()

	var importOrphansFor *[32]byte

	// Try to import orphaned blocks that depend on this block
	importOrphansFor = new([32]byte)
	*importOrphansFor = block.Hash()

	// After importing, check if any side branch is now longer than main chain
	c.checkReorg()

	// Remove scanOrphanPool call to avoid repeated full orphan pool scans
	// if scanOrphans {
	// 	c.scanOrphanPool()
	// }

	if importOrphansFor != nil {
		c.mu.Unlock()
		unlocked = true
		c.tryImportOrphans(*importOrphansFor)
		c.mu.Lock()
		unlocked = false
	}

	return nil
}

// Add a flag to prevent re-entrant orphan imports
var orphanImportInProgress int32 // 0 = not running, 1 = running

// tryImportOrphans attempts to import blocks from the orphan pool that have this block as their parent
func (c *Chain) tryImportOrphans(parentHash [32]byte) {
	// Prevent re-entrant orphan imports
	if !atomic.CompareAndSwapInt32(&orphanImportInProgress, 0, 1) {
		log.Printf("[ORPHAN] tryImportOrphans: already in progress, skipping")
		return
	}
	defer atomic.StoreInt32(&orphanImportInProgress, 0)

	var toImport []*Block
	var toSideBranch []*Block

	c.OrphanMu.Lock()
	orphans, exists := c.OrphanPool[parentHash]
	if exists {
		delete(c.OrphanPool, parentHash)
	}
	c.OrphanMu.Unlock()

	if exists {
		c.mu.RLock()
		for _, orphan := range orphans {
			parent := c.getBlockByHash(orphan.Header.ParentHash)
			parentFound := parent != nil
			if parentFound && parent.Header.Height == orphan.Header.Height-1 {
				toImport = append(toImport, orphan)
			} else if parentFound {
				toSideBranch = append(toSideBranch, orphan)
				log.Printf("🌿 Orphan block #%d promoted to side branch (parent at height %d, block height %d)", orphan.Header.Height, parent.Header.Height, orphan.Header.Height)
			}
		}
		c.mu.RUnlock()
	}

	// Import orphans and promote to side branch OUTSIDE the lock
	for _, orphan := range toImport {
		if err := c.ImportBlock(orphan); err != nil {
			log.Printf("Failed to import orphan block #%d: %v", orphan.Header.Height, err)
		} else {
			log.Printf("✅ Orphan block #%d imported by tryImportOrphans", orphan.Header.Height)
		}
	}
	for _, orphan := range toSideBranch {
		c.addToSideBranch(orphan)
	}
}

// addToOrphanPool adds a block to the orphan pool when its parent is missing
func (c *Chain) addToOrphanPool(block *Block) {
	log.Printf("[WATCHDOG] addToOrphanPool: about to lock OrphanMu (goroutine)")
	c.OrphanMu.Lock()
	log.Printf("[WATCHDOG] addToOrphanPool: OrphanMu locked (goroutine)")
	watchdogDone := make(chan struct{})
	go func() {
		select {
		case <-watchdogDone:
			return
		case <-time.After(5 * time.Second):
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			log.Printf("[WATCHDOG][WARN] addToOrphanPool: OrphanMu held >5s!\n%s", buf)
		}
	}()
	defer func() {
		close(watchdogDone)
		log.Printf("[WATCHDOG] addToOrphanPool: unlocking OrphanMu (goroutine)")
		c.OrphanMu.Unlock()
		log.Printf("[WATCHDOG] addToOrphanPool: OrphanMu unlocked (goroutine)")
	}()

	log.Printf("[DEBUG] addToOrphanPool: about to add to OrphanPool")
	// Append to the slice for this parentHash
	c.OrphanPool[block.Header.ParentHash] = append(c.OrphanPool[block.Header.ParentHash], block)
	log.Printf("📦 Added block #%d to orphan pool (parent: %x)", block.Header.Height, block.Header.ParentHash[:8])
	log.Printf("[DEBUG] Orphan pool length after add: %d", len(c.OrphanPool))
	for k := range c.OrphanPool {
		log.Printf("[DEBUG] Orphan pool key after add: %x", k[:8])
	}

	// Save parent hash for callback
	var parentHash [32]byte
	callCallback := false
	if c.RequestBlockByHash != nil {
		parentHash = block.Header.ParentHash
		callCallback = true
	}

	log.Printf("[DEBUG] addToOrphanPool: function end reached")

	// Call the callback OUTSIDE the lock
	if callCallback {
		log.Printf("[DEBUG] addToOrphanPool: calling RequestBlockByHash OUTSIDE lock")
		go c.RequestBlockByHash(parentHash)
	}
}

// addToSideBranch stores a block in the sideBranches map.
func (c *Chain) addToSideBranch(block *Block) {
	branch := c.sideBranches[block.Header.ParentHash]
	c.sideBranches[block.Header.ParentHash] = append(branch, block)
	log.Printf("🌿 Added block #%d to side branch (parent: %x, branch len: %d)", block.Header.Height, block.Header.ParentHash[:8], len(c.sideBranches[block.Header.ParentHash]))
	c.logSideBranches()
}

// logSideBranches prints the current state of all side branches.
func (c *Chain) logSideBranches() {
	for parentHash, branch := range c.sideBranches {
		if len(branch) == 0 {
			continue
		}
		log.Printf("🪵 Side branch (parent: %x) len=%d tipHeight=%d", parentHash[:8], len(branch), branch[len(branch)-1].Header.Height)
	}
}

// checkReorg checks if any side branch is now longer than the main chain and performs a reorg if needed.
func (c *Chain) checkReorg() {
	log.Printf("🔎 Checking for reorgs. Main head: %d", c.head)
	for parentHash, branch := range c.sideBranches {
		if len(branch) == 0 {
			continue
		}
		branchTip := branch[len(branch)-1]
		log.Printf("🔎 Considering side branch (parent: %x) tipHeight=%d mainHead=%d", parentHash[:8], branchTip.Header.Height, c.head)
		if branchTip.Header.Height > c.head {
			hash := branchTip.Hash()
			log.Printf("🔀 Reorg: switching to side branch at height %d (tip %x)", branchTip.Header.Height, hash[0:8])
			c.reorgToBranch(parentHash, branch)
			delete(c.sideBranches, parentHash)
		} else {
			log.Printf("❌ No reorg: side branch tipHeight=%d <= mainHead=%d", branchTip.Header.Height, c.head)
		}
	}
}

// reorgToBranch rolls back to the fork point and applies the new branch blocks.
func (c *Chain) reorgToBranch(parentHash [32]byte, branch []*Block) {
	// Roll back to fork point (parentHash)
	forkHeight := branch[0].Header.Height - 1
	c.head = forkHeight
	log.Printf("↩️  Rolled back to fork height %d", forkHeight)
	// Apply new branch blocks
	for _, blk := range branch {
		c.blocks[blk.Header.Height] = blk
		c.head = blk.Header.Height
		if err := c.store.PutBlock(blk.Header.Height, blk); err != nil {
			log.Printf("Failed to persist block %d during reorg: %v", blk.Header.Height, err)
		}
		log.Printf("🔗 Reorg applied block #%d", blk.Header.Height)
	}
	log.Printf("✅ Reorg complete. New head: %d", c.head)
}

// ScanOrphanPool scans all orphans and tries to import or promote them if their parent is now present.
func (c *Chain) ScanOrphanPool() {
	log.Printf("[WATCHDOG] scanOrphanPool: about to lock OrphanMu (goroutine)")
	c.OrphanMu.Lock()
	log.Printf("[WATCHDOG] scanOrphanPool: OrphanMu locked (goroutine)")
	watchdogDone := make(chan struct{})
	go func() {
		select {
		case <-watchdogDone:
			return
		case <-time.After(5 * time.Second):
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			log.Printf("[WATCHDOG][WARN] scanOrphanPool: OrphanMu held >5s!\n%s", buf)
		}
	}()
	defer func() {
		close(watchdogDone)
		log.Printf("[WATCHDOG] scanOrphanPool: unlocking OrphanMu (goroutine)")
		c.OrphanMu.Unlock()
		log.Printf("[WATCHDOG] scanOrphanPool: OrphanMu unlocked (goroutine)")
	}()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic in scanOrphanPool: %v", r)
			debug.PrintStack()
		}
	}()
	log.Printf("[DEBUG] scanOrphanPool called")
	log.Printf("[DEBUG] scanOrphanPool: attempting to acquire lock")
	if len(c.OrphanPool) == 0 {
		log.Printf("[DEBUG] Orphan pool empty, returning early")
		log.Printf("[DEBUG] scanOrphanPool: lock released, returning")
		return
	}
	log.Printf("[DEBUG] Orphan pool keys at scan:")
	for k := range c.OrphanPool {
		log.Printf("[DEBUG] Orphan pool key: %x", k[:8])
	}
	log.Printf("🔍 Scanning orphan pool (%d orphans)", len(c.OrphanPool))
	// Copy orphans to process after releasing lock
	orphansToProcess := make(map[[32]byte][]*Block)
	for parentHash, orphans := range c.OrphanPool {
		orphansCopy := make([]*Block, len(orphans))
		copy(orphansCopy, orphans)
		orphansToProcess[parentHash] = orphansCopy
	}
	// Remove all orphans from the pool
	c.OrphanPool = make(map[[32]byte][]*Block)
	go func() {
		// Wait for the lock to be released
		time.Sleep(10 * time.Millisecond)
		for _, orphans := range orphansToProcess {
			c.mu.RLock()
			for _, orphan := range orphans {
				parent := c.getBlockByHash(orphan.Header.ParentHash)
				parentFound := parent != nil
				if parentFound && parent.Header.Height == orphan.Header.Height-1 {
					if err := c.ImportBlock(orphan); err != nil {
						log.Printf("Failed to import orphan block #%d during scan: %v", orphan.Header.Height, err)
					} else {
						log.Printf("✅ Orphan block #%d imported during scan", orphan.Header.Height)
					}
				} else if parentFound {
					c.addToSideBranch(orphan)
					log.Printf("🌿 Orphan block #%d promoted to side branch (parent at height %d, block height %d)", orphan.Header.Height, parent.Header.Height, orphan.Header.Height)
				}
			}
			c.mu.RUnlock()
		}
		log.Printf("[DEBUG] scanOrphanPool completed")
	}()
}

// New: StartOrphanPoolScanner starts a background goroutine to periodically scan the orphan pool.
func (c *Chain) StartOrphanPoolScanner(interval time.Duration, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.ScanOrphanPool()
			case <-stopCh:
				return
			}
		}
	}()
}

// CurrentHeight returns the current chain height.
func (c *Chain) CurrentHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.head
}

// Height returns the current chain height (implements storage.Reader).
func (c *Chain) Height() uint64 {
	return c.CurrentHeight()
}

// HeaderByHeight returns the header at the given height.
func (c *Chain) HeaderByHeight(height uint64) *header.Header {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if blk, ok := c.blocks[height]; ok {
		if blk.Header.Bits == nil || blk.Header.Bits.Sign() == 0 {
			blk.Header.Bits = big.NewInt(1000) // or use config.MaximumTarget.Int64() for mainnet
		}
		return &blk.Header
	}
	// Try to load from BadgerDB if not in memory
	blk, err := c.store.GetBlock(height)
	if err == nil && blk != nil {
		if blk.Header.Bits == nil || blk.Header.Bits.Sign() == 0 {
			blk.Header.Bits = big.NewInt(1000) // or use config.MaximumTarget.Int64() for mainnet
		}
		c.blocks[height] = blk
		return &blk.Header
	}
	return nil
}

// BlockByHeight returns the block at the given height, or nil if not found.
func (c *Chain) BlockByHeight(height uint64) *Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.blocks[height]
}

// PreseedHeaders pre-populates the chain with dummy headers up to the given height (inclusive).
func (c *Chain) PreseedHeaders(upTo uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for h := uint64(1); h <= upTo; h++ {
		if _, exists := c.blocks[h]; exists {
			continue // Don't overwrite real blocks
		}
		parent := c.blocks[h-1]
		b := &Block{
			Header: header.Header{
				Height:     h,
				ParentHash: parent.Hash(),
				Lhat:       0,
				Bits:       parent.Header.Bits, // Inherit parent's target
				Timestamp:  time.Now(),
			},
			Records: []dataset.Record{},
			Time:    time.Now(),
		}
		c.blocks[h] = b
		if h > c.head {
			c.head = h
		}
		// In PreseedHeaders, remove c.saveBlock(b)
	}
	log.Printf("📗 Pre-seeded headers up to height %d", upTo)
}

// SubscribeToHeadChanges returns a channel that receives notifications when the chain head changes.
func (c *Chain) SubscribeToHeadChanges() chan struct{} {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	ch := make(chan struct{}, 1)
	c.subscribers = append(c.subscribers, ch)
	return ch
}

// notifyHeadChange notifies all subscribers that the head has changed.
func (c *Chain) notifyHeadChange() {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, ch := range c.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// Channel is full, skip this notification
		}
	}
}

// Diagnostic: Log chain state (head, blocks, orphans, side branches)
func (c *Chain) LogDiagnostics() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	log.Printf("[DIAG] Chain head: %d", c.head)
	var heights []uint64
	for h := range c.blocks {
		heights = append(heights, h)
	}
	log.Printf("[DIAG] Orphan pool size: %d", len(c.OrphanPool))
	for k, orphan := range c.OrphanPool {
		log.Printf("[DIAG] Orphan: parentHash=%x height=%d", k[:8], orphan[0].Header.Height) // Assuming all orphans for a parent have the same height
	}
	for parentHash, branch := range c.sideBranches {
		if len(branch) == 0 {
			continue
		}
		tip := branch[len(branch)-1]
		log.Printf("[DIAG] Side branch: parent=%x tipHeight=%d len=%d", parentHash[:8], tip.Header.Height, len(branch))
	}
}

// Reindex: Rebuild in-memory block index from BadgerDB
func (c *Chain) ReindexFromDB() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Printf("[REINDEX] Rebuilding in-memory block index from BadgerDB...")
	c.blocks = make(map[uint64]*Block)
	c.blockHashIndex = make(map[[32]byte]*Block)
	tip, err := c.store.GetTipHeight()
	if err != nil {
		if err.Error() == "Key not found" {
			log.Printf("[REINDEX][WARN] No blocks found in DB (empty chain). Will start fresh.")
			return nil
		}
		return err
	}
	for h := uint64(0); h <= tip; h++ {
		blk, err := c.store.GetBlock(h)
		if err == nil && blk != nil {
			c.blocks[h] = blk
			c.blockHashIndex[blk.Hash()] = blk
			if h > c.head {
				c.head = h
			}
		}
	}
	log.Printf("[REINDEX] Done. Head: %d, blocks loaded: %d", c.head, len(c.blocks))
	return nil
}

// getBlockByHash safely reads blockHashIndex with lock
func (c *Chain) getBlockByHash(h [32]byte) *Block {
	c.mu.RLock()
	b := c.blockHashIndex[h]
	c.mu.RUnlock()
	return b
}
