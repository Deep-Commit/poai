package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"poai/core"

	"github.com/ethereum/go-ethereum/crypto"
)

// CLI commands for transaction operations
func handleCLICommands() {
	if len(os.Args) < 2 {
		return // No subcommand, run as daemon
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "send":
		handleSendCommand()
	case "balance":
		handleBalanceCommand()
	case "generate-key":
		handleGenerateKeyCommand()
	case "help":
		printHelp()
	default:
		// Unknown subcommand, run as daemon
		return
	}

	os.Exit(0)
}

func handleSendCommand() {
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	toAddr := sendCmd.String("to", "", "Recipient address (hex)")
	amount := sendCmd.String("amount", "", "Amount to send")
	privKeyHex := sendCmd.String("privkey", "", "Private key (hex)")

	sendCmd.Parse(os.Args[2:])

	if *toAddr == "" || *amount == "" || *privKeyHex == "" {
		fmt.Println("Usage: poaid send -to=<address> -amount=<amount> -privkey=<private_key>")
		os.Exit(1)
	}

	// Parse private key
	privKeyBytes, err := hex.DecodeString(*privKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		log.Fatalf("Invalid private key format: %v", err)
	}

	// Parse recipient address
	toAddrBytes, err := hex.DecodeString(*toAddr)
	if err != nil {
		log.Fatalf("Invalid recipient address: %v", err)
	}

	// Parse amount
	amountInt, ok := new(big.Int).SetString(*amount, 10)
	if !ok {
		log.Fatalf("Invalid amount: %s", *amount)
	}

	// Get sender address from private key
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	senderAddr := crypto.PubkeyToAddress(*pubKey).Bytes()

	// Create transaction
	tx := core.NewTx(senderAddr, toAddrBytes, amountInt, 0) // Nonce will be set by state

	// Sign transaction
	if err := tx.Sign(privKey); err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	fmt.Printf("Transaction created:\n")
	fmt.Printf("  From: %s\n", hex.EncodeToString(senderAddr))
	fmt.Printf("  To: %s\n", *toAddr)
	fmt.Printf("  Amount: %s\n", amountInt.String())
	fmt.Printf("  Hash: %s\n", hex.EncodeToString(tx.Hash))
	fmt.Printf("  Signature: %s\n", hex.EncodeToString(tx.Signature))

	// TODO: Send transaction to mempool via RPC or file
	fmt.Printf("\nTransaction signed successfully. Add to mempool to broadcast.\n")
}

func handleBalanceCommand() {
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	addr := balanceCmd.String("addr", "", "Address to check balance for (hex)")
	dataDir := balanceCmd.String("data-dir", "data1", "Data directory containing the blockchain state")

	balanceCmd.Parse(os.Args[2:])

	if *addr == "" {
		fmt.Println("Usage: poaid balance -addr=<address> [-data-dir=<directory>]")
		os.Exit(1)
	}

	addrBytes, err := hex.DecodeString(*addr)
	if err != nil {
		log.Fatalf("Invalid address: %v", err)
	}

	// Try to open BadgerDB
	store, err := core.OpenBadgerStore(*dataDir)
	if err != nil {
		fmt.Printf("❌ Cannot access database: %v\n", err)
		fmt.Printf("💡 This usually means a mining node is running. Try:\n")
		fmt.Printf("   1. Stop the mining node first, or\n")
		fmt.Printf("   2. Use a different data directory\n")
		os.Exit(1)
	}
	defer store.Close()

	// Create state manager
	state := core.NewState(store.GetDB())

	// Get balance
	balance := state.GetBalance(addrBytes)

	fmt.Printf("💰 Balance for %s: %s POAI\n", *addr, balance.String())
}

func handleGenerateKeyCommand() {
	generateCmd := flag.NewFlagSet("generate-key", flag.ExitOnError)
	saveToFile := generateCmd.Bool("save", false, "Save keys to files")
	outputDir := generateCmd.String("output-dir", ".", "Directory to save key files")

	generateCmd.Parse(os.Args[2:])

	// Generate a new keypair
	privKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	pubKey := privKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*pubKey)
	privKeyHex := hex.EncodeToString(crypto.FromECDSA(privKey))
	addressHex := hex.EncodeToString(address.Bytes())

	fmt.Printf("🔑 Generated new PoAI keypair:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📝 Private Key (hex):\n")
	fmt.Printf("   %s\n", privKeyHex)
	fmt.Printf("\n🔐 Public Key (hex):\n")
	fmt.Printf("   %s\n", hex.EncodeToString(crypto.FromECDSAPub(pubKey)))
	fmt.Printf("\n💰 Miner Address (hex):\n")
	fmt.Printf("   %s\n", addressHex)
	fmt.Printf("\n🏠 Address (checksum):\n")
	fmt.Printf("   %s\n", address.Hex())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Save to files if requested
	if *saveToFile {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		// Save private key
		privKeyFile := filepath.Join(*outputDir, "poai_private_key.txt")
		if err := os.WriteFile(privKeyFile, []byte(privKeyHex), 0600); err != nil {
			log.Fatalf("Failed to save private key: %v", err)
		}

		// Save address
		addressFile := filepath.Join(*outputDir, "poai_address.txt")
		if err := os.WriteFile(addressFile, []byte(addressHex), 0644); err != nil {
			log.Fatalf("Failed to save address: %v", err)
		}

		// Save miner configuration
		minerConfigFile := filepath.Join(*outputDir, "miner_config.txt")
		minerConfig := fmt.Sprintf("# PoAI Miner Configuration\n"+
			"# Copy this address to use with --miner-address flag\n\n"+
			"MINER_ADDRESS=%s\n\n"+
			"# Example usage:\n"+
			"# ./poaid --miner-address=%s --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --target=500\n",
			addressHex, addressHex)
		if err := os.WriteFile(minerConfigFile, []byte(minerConfig), 0644); err != nil {
			log.Fatalf("Failed to save miner config: %v", err)
		}

		fmt.Printf("\n💾 Keys saved to files:\n")
		fmt.Printf("   Private Key: %s\n", privKeyFile)
		fmt.Printf("   Address: %s\n", addressFile)
		fmt.Printf("   Miner Config: %s\n", minerConfigFile)
		fmt.Printf("\n⚠️  SECURITY WARNING:\n")
		fmt.Printf("   • Keep your private key secure and never share it\n")
		fmt.Printf("   • The private key file has restricted permissions (600)\n")
		fmt.Printf("   • Use the address for mining and receiving rewards\n")
	}

	fmt.Printf("\n🚀 Ready to mine! Use the address with --miner-address flag:\n")
	fmt.Printf("   ./poaid --miner-address=%s --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --target=500\n", addressHex)
}

func printHelp() {
	fmt.Println("PoAI Daemon - Proof of AI Blockchain")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  poaid [flags]                    - Run as daemon")
	fmt.Println("  poaid send [flags]               - Send a transaction")
	fmt.Println("  poaid balance [flags]            - Check balance")
	fmt.Println("  poaid generate-key [flags]       - Generate new keypair")
	fmt.Println("  poaid help                       - Show this help")
	fmt.Println()
	fmt.Println("Daemon Flags:")
	fmt.Println("  --model-path=<path>              - Path to LLM model")
	fmt.Println("  --target=<difficulty>            - Mining difficulty target")
	fmt.Println("  --data-dir=<path>                - Data directory")
	fmt.Println("  --p2p-port=<port>                - P2P listen port")
	fmt.Println("  --peer-multiaddr=<addr>          - Peer to connect to")
	fmt.Println("  --miner-address=<hex>            - Miner address for block rewards")
	fmt.Println()
	fmt.Println("Generate Key Flags:")
	fmt.Println("  --save                           - Save keys to files")
	fmt.Println("  --output-dir=<path>              - Directory to save key files")
	fmt.Println()
	fmt.Println("Send Flags:")
	fmt.Println("  --to=<address>                   - Recipient address (hex)")
	fmt.Println("  --amount=<amount>                - Amount to send")
	fmt.Println("  --privkey=<private_key>          - Private key (hex)")
	fmt.Println()
	fmt.Println("Balance Flags:")
	fmt.Println("  --addr=<address>                 - Address to check (hex)")
}
