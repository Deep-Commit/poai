package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestTransactionCreation(t *testing.T) {
	// Generate a keypair
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	pubKey := privKey.Public().(*ecdsa.PublicKey)
	senderAddr := crypto.PubkeyToAddress(*pubKey).Bytes()
	recipientAddr := []byte("recipient-12345678901234567890123456789012")

	// Create a transaction
	amount := big.NewInt(100)
	tx := NewTx(senderAddr, recipientAddr, amount, 0)

	// Sign the transaction
	if err := tx.Sign(privKey); err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify the transaction
	if err := tx.Verify(); err != nil {
		t.Fatalf("Transaction verification failed: %v", err)
	}

	// Check that it's not a coinbase transaction
	if tx.IsCoinbase() {
		t.Fatal("Regular transaction should not be coinbase")
	}

	t.Logf("Transaction created successfully: %s", tx.String())
}

func TestCoinbaseTransaction(t *testing.T) {
	minerAddr := []byte("miner-12345678901234567890123456789012")
	subsidy := big.NewInt(50)

	// Create coinbase transaction
	tx := NewCoinbaseTx(minerAddr, subsidy)

	// Check that it's a coinbase transaction
	if !tx.IsCoinbase() {
		t.Fatal("Coinbase transaction should be identified as coinbase")
	}

	// Coinbase transactions don't need signatures
	if err := tx.Verify(); err != nil {
		t.Fatalf("Coinbase verification failed: %v", err)
	}

	t.Logf("Coinbase transaction created successfully: %s", tx.String())
}

func TestTransactionEncoding(t *testing.T) {
	// Generate a keypair
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	pubKey := privKey.Public().(*ecdsa.PublicKey)
	senderAddr := crypto.PubkeyToAddress(*pubKey).Bytes()
	recipientAddr := []byte("recipient-12345678901234567890123456789012")

	// Create and sign a transaction
	amount := big.NewInt(100)
	tx := NewTx(senderAddr, recipientAddr, amount, 0)
	if err := tx.Sign(privKey); err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Encode
	encoded, err := tx.Encode()
	if err != nil {
		t.Fatalf("Failed to encode transaction: %v", err)
	}

	// Decode
	decoded, err := DecodeTransaction(encoded)
	if err != nil {
		t.Fatalf("Failed to decode transaction: %v", err)
	}

	// Verify the decoded transaction
	if err := decoded.Verify(); err != nil {
		t.Fatalf("Decoded transaction verification failed: %v", err)
	}

	// Check that amounts match
	if tx.Amount.Cmp(decoded.Amount) != 0 {
		t.Fatalf("Amount mismatch: original %s, decoded %s", tx.Amount.String(), decoded.Amount.String())
	}

	t.Logf("Transaction encoding/decoding successful")
}

func TestGetSubsidy(t *testing.T) {
	// Test initial subsidy
	subsidy := GetSubsidy(0)
	if subsidy.Cmp(big.NewInt(InitialSubsidy)) != 0 {
		t.Fatalf("Genesis subsidy should be %d, got %s", InitialSubsidy, subsidy.String())
	}

	// Test first halving
	subsidy = GetSubsidy(HalvingBlocks)
	if subsidy.Cmp(big.NewInt(InitialSubsidy/2)) != 0 {
		t.Fatalf("First halving subsidy should be %d, got %s", InitialSubsidy/2, subsidy.String())
	}

	// Test second halving
	subsidy = GetSubsidy(HalvingBlocks * 2)
	if subsidy.Cmp(big.NewInt(InitialSubsidy/4)) != 0 {
		t.Fatalf("Second halving subsidy should be %d, got %s", InitialSubsidy/4, subsidy.String())
	}

	t.Logf("Subsidy calculation working correctly")
}
