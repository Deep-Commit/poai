package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LocalBroadcaster handles local block broadcasting via files.
type LocalBroadcaster struct {
	blocksDir string
	chain     *Chain
	processed map[string]bool // Track processed files to avoid duplicates
	mu        sync.RWMutex
}

// NewLocalBroadcaster creates a new local broadcaster.
func NewLocalBroadcaster(blocksDir string, chain *Chain) *LocalBroadcaster {
	os.MkdirAll(blocksDir, 0755)
	return &LocalBroadcaster{
		blocksDir: blocksDir,
		chain:     chain,
		processed: make(map[string]bool),
	}
}

// BroadcastBlock writes a block to a file for local processing.
func (b *LocalBroadcaster) BroadcastBlock(block *Block) error {
	// Create a unique filename with timestamp
	timestamp := time.Now().UnixNano()
	filename := filepath.Join(b.blocksDir, fmt.Sprintf("block_%d_%d.json", block.Header.Height, timestamp))

	// Encode and write the block
	data, err := block.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode block: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write block file: %w", err)
	}

	log.Printf("📤 Broadcasted block #%d loss=%d to %s", block.Header.Height, block.Header.Lhat, filename)
	return nil
}

// ProcessBlocks reads and imports blocks from the blocks directory.
func (b *LocalBroadcaster) ProcessBlocks() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		files, err := os.ReadDir(b.blocksDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}

			// Check if we've already processed this file
			b.mu.Lock()
			if b.processed[file.Name()] {
				b.mu.Unlock()
				continue
			}
			b.processed[file.Name()] = true
			b.mu.Unlock()

			filepath := filepath.Join(b.blocksDir, file.Name())
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			block, err := DecodeBlock(data)
			if err != nil {
				log.Printf("Failed to decode block from %s: %v", file.Name(), err)
				os.Remove(filepath) // Remove corrupted file
				continue
			}

			// Try to import the block
			err = b.chain.ImportBlock(block)
			if err != nil {
				// Don't log orphan pool messages as errors
				if err.Error() == fmt.Sprintf("parent block at height %d not found, added to orphan pool", block.Header.Height-1) {
					continue
				}
				continue
			}

			// Successfully imported, remove the file
			os.Remove(filepath)
		}
	}
}
