package core

import (
	"fmt"
	"log"
	"math/big"

	"github.com/dgraph-io/badger/v4"
)

// State manages account balances and transaction execution
type State struct {
	db *badger.DB
}

// NewState creates a new state manager
func NewState(db *badger.DB) *State {
	return &State{db: db}
}

// GetBalance returns the balance for the given address
func (s *State) GetBalance(addr []byte) *big.Int {
	balance := big.NewInt(0)
	err := s.db.View(func(txn *badger.Txn) error {
		key := append([]byte("balance:"), addr...)
		item, err := txn.Get(key)
		if err == nil {
			return item.Value(func(val []byte) error {
				balance.SetBytes(val)
				return nil
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("[STATE] Error getting balance: %v", err)
	}
	return balance
}

// SetBalance sets the balance for the given address
func (s *State) SetBalance(addr []byte, amount *big.Int) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := append([]byte("balance:"), addr...)
		return txn.Set(key, amount.Bytes())
	})
}

// AddBalance adds to the balance for the given address
func (s *State) AddBalance(addr []byte, amount *big.Int) error {
	balance := s.GetBalance(addr)
	balance.Add(balance, amount)
	return s.SetBalance(addr, balance)
}

// SubBalance subtracts from the balance for the given address
func (s *State) SubBalance(addr []byte, amount *big.Int) error {
	balance := s.GetBalance(addr)
	if balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance: have %s, need %s", balance.String(), amount.String())
	}
	balance.Sub(balance, amount)
	return s.SetBalance(addr, balance)
}

// GetNonce returns the current nonce for the given address
func (s *State) GetNonce(addr []byte) uint64 {
	var nonce uint64
	err := s.db.View(func(txn *badger.Txn) error {
		key := append([]byte("nonce:"), addr...)
		item, err := txn.Get(key)
		if err == nil {
			return item.Value(func(val []byte) error {
				// Simple conversion from bytes to uint64
				for i, b := range val {
					if i >= 8 {
						break
					}
					nonce |= uint64(b) << (i * 8)
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("[STATE] Error getting nonce: %v", err)
	}
	return nonce
}

// SetNonce sets the nonce for the given address
func (s *State) SetNonce(addr []byte, nonce uint64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := append([]byte("nonce:"), addr...)
		// Convert uint64 to bytes
		val := make([]byte, 8)
		for i := 0; i < 8; i++ {
			val[i] = byte(nonce >> (i * 8))
		}
		return txn.Set(key, val)
	})
}

// IncrementNonce increments the nonce for the given address
func (s *State) IncrementNonce(addr []byte) error {
	nonce := s.GetNonce(addr)
	return s.SetNonce(addr, nonce+1)
}

// ExecuteTransaction executes a transaction and updates state
func (s *State) ExecuteTransaction(tx *Transaction) error {
	// Verify transaction signature
	if err := tx.Verify(); err != nil {
		return fmt.Errorf("transaction verification failed: %v", err)
	}

	// Handle coinbase transactions
	if tx.IsCoinbase() {
		return s.AddBalance(tx.To, tx.Amount)
	}

	// Check nonce
	expectedNonce := s.GetNonce(tx.From)
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("invalid nonce: expected %d, got %d", expectedNonce, tx.Nonce)
	}

	// Calculate gas cost (simplified)
	gasCost := big.NewInt(int64(tx.GasLimit))
	gasCost.Mul(gasCost, tx.GasPrice)
	totalCost := new(big.Int).Add(tx.Amount, gasCost)

	// Check balance
	balance := s.GetBalance(tx.From)
	if balance.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient balance: have %s, need %s", balance.String(), totalCost.String())
	}

	// Execute the transaction
	if err := s.SubBalance(tx.From, totalCost); err != nil {
		return fmt.Errorf("failed to subtract from sender: %v", err)
	}

	if err := s.AddBalance(tx.To, tx.Amount); err != nil {
		return fmt.Errorf("failed to add to recipient: %v", err)
	}

	// Increment nonce
	if err := s.IncrementNonce(tx.From); err != nil {
		return fmt.Errorf("failed to increment nonce: %v", err)
	}

	return nil
}

// ValidateTransaction validates a transaction without executing it
func (s *State) ValidateTransaction(tx *Transaction) error {
	// Verify transaction signature
	if err := tx.Verify(); err != nil {
		return fmt.Errorf("transaction verification failed: %v", err)
	}

	// Handle coinbase transactions
	if tx.IsCoinbase() {
		return nil
	}

	// Check nonce
	expectedNonce := s.GetNonce(tx.From)
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("invalid nonce: expected %d, got %d", expectedNonce, tx.Nonce)
	}

	// Calculate gas cost
	gasCost := big.NewInt(int64(tx.GasLimit))
	gasCost.Mul(gasCost, tx.GasPrice)
	totalCost := new(big.Int).Add(tx.Amount, gasCost)

	// Check balance
	balance := s.GetBalance(tx.From)
	if balance.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient balance: have %s, need %s", balance.String(), totalCost.String())
	}

	return nil
}

// InitializeGenesisState sets up initial balances for genesis
func (s *State) InitializeGenesisState() error {
	// Create a test account with some initial balance
	testAddr := []byte("test-account-12345678901234567890123456789012")
	initialBalance := big.NewInt(1000) // 1000 POAI for testing

	log.Printf("[STATE] Initializing genesis state with test account balance: %s", initialBalance.String())
	return s.SetBalance(testAddr, initialBalance)
}
