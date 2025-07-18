package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

// Transaction represents a value transfer on the PoAI blockchain
type Transaction struct {
	From      []byte   `json:"from"`      // Sender address (pubkey hash)
	To        []byte   `json:"to"`        // Recipient address
	Amount    *big.Int `json:"amount"`    // Value to transfer
	Nonce     uint64   `json:"nonce"`     // Replay protection
	GasLimit  uint64   `json:"gasLimit"`  // Fixed for now (21000)
	GasPrice  *big.Int `json:"gasPrice"`  // For priority; stub
	Signature []byte   `json:"signature"` // ECDSA signature
	Hash      []byte   `json:"hash"`      // Cached hash
}

// NewCoinbaseTx creates a coinbase transaction for block subsidies
func NewCoinbaseTx(minerAddr []byte, subsidy *big.Int) *Transaction {
	return &Transaction{
		From:     []byte{}, // No sender for coinbase
		To:       minerAddr,
		Amount:   subsidy,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: big.NewInt(0),
	}
}

// NewTx creates a regular value transfer transaction
func NewTx(from, to []byte, amount *big.Int, nonce uint64) *Transaction {
	return &Transaction{
		From:     from,
		To:       to,
		Amount:   amount,
		Nonce:    nonce,
		GasLimit: 21000, // Standard ETH transfer gas
		GasPrice: big.NewInt(1),
	}
}

// CalculateHash computes the transaction hash (keccak256 for EVM compatibility)
func (tx *Transaction) CalculateHash() []byte {
	// Create a deterministic representation for hashing
	data := struct {
		From     []byte   `json:"from"`
		To       []byte   `json:"to"`
		Amount   *big.Int `json:"amount"`
		Nonce    uint64   `json:"nonce"`
		GasLimit uint64   `json:"gasLimit"`
		GasPrice *big.Int `json:"gasPrice"`
	}{
		From:     tx.From,
		To:       tx.To,
		Amount:   tx.Amount,
		Nonce:    tx.Nonce,
		GasLimit: tx.GasLimit,
		GasPrice: tx.GasPrice,
	}

	// Serialize to JSON for consistent hashing
	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal transaction: %v", err))
	}

	return crypto.Keccak256(jsonData)
}

// Sign signs the transaction with the provided private key
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.CalculateHash()
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}
	tx.Signature = sig
	tx.Hash = hash
	return nil
}

// Verify verifies the transaction signature
func (tx *Transaction) Verify() error {
	// Coinbase transactions don't need signature verification
	if len(tx.From) == 0 {
		return nil
	}

	if len(tx.Signature) == 0 {
		return errors.New("transaction has no signature")
	}

	hash := tx.CalculateHash()
	pubKey, err := crypto.SigToPub(hash, tx.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %v", err)
	}

	sender := crypto.PubkeyToAddress(*pubKey).Bytes()
	if !bytes.Equal(sender, tx.From) {
		return errors.New("signature does not match sender address")
	}

	return nil
}

// IsCoinbase returns true if this is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.From) == 0
}

// String returns a string representation of the transaction
func (tx *Transaction) String() string {
	from := "coinbase"
	if len(tx.From) > 0 {
		from = hex.EncodeToString(tx.From[:8]) + "..."
	}
	to := hex.EncodeToString(tx.To[:8]) + "..."
	return fmt.Sprintf("Tx{From: %s, To: %s, Amount: %s, Nonce: %d}",
		from, to, tx.Amount.String(), tx.Nonce)
}

// Encode serializes the transaction to JSON
func (tx *Transaction) Encode() ([]byte, error) {
	return json.Marshal(tx)
}

// Decode deserializes the transaction from JSON
func DecodeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %v", err)
	}
	return &tx, nil
}
