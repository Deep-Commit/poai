package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
)

// Mempool manages pending transactions
type Mempool struct {
	txs   map[string]*Transaction // Key: transaction hash hex
	mu    sync.RWMutex
	state *State
}

// NewMempool creates a new mempool
func NewMempool(state *State) *Mempool {
	return &Mempool{
		txs:   make(map[string]*Transaction),
		state: state,
	}
}

// AddTransaction adds a transaction to the mempool
func (mp *Mempool) AddTransaction(tx *Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Calculate transaction hash if not set
	if len(tx.Hash) == 0 {
		tx.Hash = tx.CalculateHash()
	}

	// Check if transaction already exists
	txHash := hex.EncodeToString(tx.Hash)
	if _, exists := mp.txs[txHash]; exists {
		return fmt.Errorf("transaction already in mempool")
	}

	// Validate transaction
	if err := mp.state.ValidateTransaction(tx); err != nil {
		return fmt.Errorf("transaction validation failed: %v", err)
	}

	// Add to mempool
	mp.txs[txHash] = tx
	log.Printf("[MEMPOOL] Added transaction %s: %s", txHash[:8], tx.String())

	return nil
}

// GetTransaction returns a transaction by hash
func (mp *Mempool) GetTransaction(hash []byte) *Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txHash := hex.EncodeToString(hash)
	return mp.txs[txHash]
}

// GetTransactionsForBlock returns transactions to include in a block
func (mp *Mempool) GetTransactionsForBlock(maxTxs int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var txs []*Transaction
	for _, tx := range mp.txs {
		txs = append(txs, tx)
		if len(txs) >= maxTxs {
			break
		}
	}

	return txs
}

// RemoveTransaction removes a transaction from the mempool
func (mp *Mempool) RemoveTransaction(hash []byte) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	txHash := hex.EncodeToString(hash)
	if tx, exists := mp.txs[txHash]; exists {
		delete(mp.txs, txHash)
		log.Printf("[MEMPOOL] Removed transaction %s: %s", txHash[:8], tx.String())
	}
}

// RemoveTransactions removes multiple transactions from the mempool
func (mp *Mempool) RemoveTransactions(txs []*Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, tx := range txs {
		txHash := hex.EncodeToString(tx.Hash)
		delete(mp.txs, txHash)
	}
}

// Size returns the number of transactions in the mempool
func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.txs)
}

// GetAllTransactions returns all transactions in the mempool
func (mp *Mempool) GetAllTransactions() []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var txs []*Transaction
	for _, tx := range mp.txs {
		txs = append(txs, tx)
	}
	return txs
}

// Cleanup removes invalid transactions from the mempool
func (mp *Mempool) Cleanup() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var toRemove []string
	for txHash, tx := range mp.txs {
		if err := mp.state.ValidateTransaction(tx); err != nil {
			log.Printf("[MEMPOOL] Removing invalid transaction %s: %v", txHash[:8], err)
			toRemove = append(toRemove, txHash)
		}
	}

	for _, txHash := range toRemove {
		delete(mp.txs, txHash)
	}
}

// StartCleanup starts a background goroutine to periodically clean up invalid transactions
func (mp *Mempool) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			mp.Cleanup()
		}
	}()
}

// GetStats returns mempool statistics
func (mp *Mempool) GetStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	totalValue := big.NewInt(0)
	for _, tx := range mp.txs {
		if !tx.IsCoinbase() {
			totalValue.Add(totalValue, tx.Amount)
		}
	}

	return map[string]interface{}{
		"size":        len(mp.txs),
		"total_value": totalValue.String(),
	}
}
