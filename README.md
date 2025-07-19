# PoAI: Proof of AI

![Demo](demo.gif)

## Status

**What works:**
* ✅ LLM inference for proof of AI (real llama.cpp integration)
* ✅ Deterministic LLM inference with nonce-based mining
* ✅ Node daemon `poaid` with CLI commands
* ✅ P2P networking with libp2p
* ✅ Procedural quiz generation for mining datasets (dynamic, verifiable Q&A without external files)
* ✅ Block mining and validation using LLM outputs
* ✅ Bitcoin-style difficulty retargeting with compact bits
* ✅ Local testnet with multiple nodes syncing chains and competing for blocks
* ✅ Key generation and management for mining rewards
* ✅ Ethereum-compatible addresses for block subsidies
* ✅ **Wallet system with balance checking**
* ✅ **Transaction creation and signing**
* ✅ **Secure cryptographic transfers between addresses**

**What doesn't work yet:**
* ❌ Transaction broadcasting and mempool integration
* ❌ On-chain governance (deprecated; procedural generation handles datasets)
* ❌ Full EVM compatibility for smart contracts
* ❌ Advanced features like staking or AI model upgrades

Check repo traffic trends: [Traffic Graphs](https://github.com/Deep-Commit/poai/graphs/traffic)

## Abstract

PoAI is a blockchain protocol that replaces traditional Proof of Work (PoW) with Proof of AI (PoAI), where miners perform verifiable forward passes on large language models (LLMs) to generate blocks. This leverages AI computation for consensus, making mining useful for AI training/inference while maintaining security. Datasets for inference are now procedurally generated (e.g., dynamic math quizzes), ensuring determinism and eliminating the need for curated datasets or on-chain governance.

## Introduction

Traditional blockchains like Bitcoin use energy-intensive hashing for PoW. PoAI shifts this to AI workloads: Miners run inference on procedural quizzes using a fixed model (e.g., TinyLlama-1.1B), and valid outputs (meeting difficulty targets) create blocks. Rewards (subsidies) are earned automatically on successful mines. The chain is EVM-compatible in design, with future support for transactions and contracts.

Why now? With GGUF models and go-llama.cpp, deterministic AI proofs are feasible locally. This repo implements the core node in Go.

## Quick Start

Want to get mining quickly? Here's the complete workflow:

```bash
# 1. Clone and setup
git clone https://github.com/Deep-Commit/poai.git
cd poai
git submodule update --init --recursive

# 2. Build the daemon (with real LLM support)
brew install llama.cpp  # macOS
# OR
sudo apt install llama-cpp  # Ubuntu/Debian

go build -tags llama -o poaid cmd/poaid/*.go
# Alternative for Linux (if shell expansion fails):
# go build -tags llama -o poaid cmd/poaid/main.go cmd/poaid/cli.go

# 3. Generate mining keys
./scripts/generate_keys.sh

# 4. Set your Hugging Face token (required for model download)
export HF_TOKEN=hf_XXXX  # Get from https://huggingface.co/settings/tokens

# 5. Start mining! (automatically downloads model)
./scripts/start_mining.sh

# Or manually:
# ./poaid --miner-address=$(cat keys/poai_address.txt) \
#         --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
#         --target=500 \
#         --data-dir=data1 \
#         --p2p-port=4001
```

That's it! You're now mining PoAI blocks and earning rewards. See the detailed setup instructions below for more options.

## Setup Instructions

### Clone the Repository
Clone the repo and initialize submodules:
```bash
git clone https://github.com/Deep-Commit/poai.git
cd poai
git submodule update --init --recursive  # Initializes go-llama.cpp submodule
```

**Important: Directory Check**
Ensure you are in the repository root directory before running any build or run commands. Run `ls` to verify—you should see files and directories like `go.mod`, `go.sum`, `core/`, `cmd/`, `README.md`, etc. If you see something else (e.g., a nested `poai/` subdir), you may have cloned into an existing directory—delete and re-clone into a clean location. Running commands outside the root will cause errors like "go.mod file not found".

### Prerequisites
- Go 1.21+ installed.
- A C compiler (e.g., clang or gcc).
- Hardware: CPU is sufficient (use `--gpu-layers=0`); GPU optional for acceleration (e.g., Metal on macOS, CUDA on Linux/Windows).
- Disk space: ~1GB for models and data directories.

### Download the LLM Model
PoAI uses the TinyLlama-1.1B-Chat-v1.0 GGUF model for deterministic inference. Note: Hugging Face may require a free account and access token for command-line downloads due to Git LFS restrictions. Do not commit models to the repo.

Create a models directory:
```bash
mkdir -p models
```

#### Option 1: Basic Curl (May Require Authentication)
```bash
curl -L -o models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf
```

**Note**: This will likely fail due to Hugging Face's authentication requirements. Use Option 2 or 3 instead.

#### Option 2: Curl with Hugging Face Access Token
1. Sign up for a free Hugging Face account at https://huggingface.co/join.
2. Create an access token (Read role) at https://huggingface.co/settings/tokens.
3. Replace `hf_XXXX` with your token and run:
```bash
curl -L -H "Authorization: Bearer hf_XXXX" -o models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf
```

#### Option 3: Hugging Face CLI (Recommended for Reliability)
1. Install the CLI (requires Python 3.8+ and pip):
```bash
pip install -U huggingface_hub
```
2. Log in (paste your access token when prompted):
```bash
huggingface-cli login
```
3. Download:
```bash
huggingface-cli download TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --local-dir models
```

Verify the download: The file should be ~669MB. Update the `--model-path` flag in node commands to `models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf`.

### Build the Daemon
Ensure you're in the repo root (check with `ls` as above).

#### Option 1: Real LLM Build (Recommended)
This builds with real LLM inference using llama.cpp. Requires `llama-cpp` to be installed:

```bash
# Install llama.cpp first
brew install llama.cpp

# Build with real LLM support
go build -tags llama -o poaid cmd/poaid/*.go
```

#### Option 2: Stub LLM Build (Fast Testing)
This builds with stub LLM for fast testing without LLM dependencies:

```bash
# Build with stub LLM (no external dependencies)
go build -o poaid cmd/poaid/*.go
```

#### Option 3: GPU Acceleration (macOS)
For macOS with Metal GPU acceleration:

```bash
CGO_LDFLAGS="-framework Metal -framework Foundation" go build -tags llama -o poaid cmd/poaid/*.go
```

**Note**: The `cmd/poaid/*.go` pattern includes all Go files in the cmd/poaid directory (main.go and cli.go).

If you encounter "go.mod not found", double-check your directory—you must be in the root where `go.mod` exists.

### Generate Mining Keys
Before starting to mine, you need to generate a keypair to receive block rewards. PoAI uses Ethereum-compatible addresses for mining rewards.

#### Option 1: Using the Key Generation Script (Recommended)
```bash
# Make the script executable and run it
chmod +x scripts/generate_keys.sh
./scripts/generate_keys.sh
```

This will:
- Generate a new keypair
- Save the private key to `keys/poai_private_key.txt` (secure)
- Save the address to `keys/poai_address.txt`
- Create a miner config file with usage examples

#### Option 2: Using the CLI Command Directly
```bash
# Generate keys and display them
./poaid generate-key

# Generate keys and save to files
./poaid generate-key --save --output-dir=./keys
```

#### Option 3: Manual Key Generation
If you prefer to generate keys manually using standard tools:
```bash
# Using OpenSSL (if available)
openssl ecparam -genkey -name secp256k1 -out private.pem
openssl ec -in private.pem -pubout -out public.pem

# Convert to hex format (you'll need to implement this conversion)
# The address is the last 20 bytes of the Keccak256 hash of the public key
```

### Run a Local Testnet
This sets up a local blockchain with LLM-based mining on procedurally generated quizzes. Use `--target=500` for easier mining (lower values = harder difficulty; adjusts automatically like Bitcoin). Run all commands from the repo root.

**Note**: Procedural quiz generation is enabled by default. If you want to use a test corpus instead, add `--test-corpus=./dataset/testdata` to the commands below.

**Pro Tip**: Use `./scripts/start_mining.sh` to automatically download the model and start mining with your generated keys.

#### Start Node 1 (with mining address)
```bash
# If you used the key generation script
./poaid --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
        --gpu-layers=0 \
        --data-dir=data1 \
        --p2p-port=4001 \
        --target=500 \
        --miner-address=$(cat keys/poai_address.txt)
```

Or manually specify the address:
```bash
./poaid --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
        --gpu-layers=0 \
        --data-dir=data1 \
        --p2p-port=4001 \
        --target=500 \
        --miner-address=YOUR_GENERATED_ADDRESS_HERE
```

- Note the peer ID from logs (e.g., `/ip4/127.0.0.1/tcp/4001/p2p/<PEER_ID>`).

#### Start Node 2 (Connect to Node 1)
Replace `<PEER_ID_FROM_NODE_1>` with the actual ID and use a different mining address:
```bash
# Generate a second keypair for node 2
./poaid generate-key --save --output-dir=./keys2

# Start node 2 with the new address
./poaid --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
        --gpu-layers=0 \
        --data-dir=data2 \
        --p2p-port=4002 \
        --target=500 \
        --miner-address=$(cat keys2/poai_address.txt) \
        --peer-multiaddr="/ip4/127.0.0.1/tcp/4001/p2p/<PEER_ID_FROM_NODE_1>"
```

#### Blockchain Settings and Verification
- **Difficulty/Targets**: Set via `--target` (e.g., -1000000000000000000 for realistic mining; more negative values = harder). Retargets every 2016 blocks based on timestamps (see `core/difficulty.go`). Blocks mine when LLM inference loss (hashed) <= target. Uses nonce-based probabilistic search like Bitcoin.
- **Subsidies/Rewards**: Automatic on mined blocks (fixed amount, halving model). Rewards credit to miner's address; future transactions will enable sending/receiving.
- **Procedural Quizzes**: Mining auto-generates deterministic quizzes (e.g., math problems seeded by block height) for LLM inference—no external files needed.
- Verify: Watch logs for "Generated quiz: ...", "Block mined!", and chain sync. Nodes compete; successful mining earns subsidies.
- Troubleshooting: If LLM fails, check model path/threads. Data persists in `data1`/`data2` for restarts. If commands fail, confirm you're in the repo root.

### Key Management and Security

#### Understanding PoAI Keys
- **Private Key**: 64-character hex string (32 bytes) - KEEP SECRET
- **Public Key**: 128-character hex string (64 bytes) - derived from private key
- **Address**: 40-character hex string (20 bytes) - derived from public key, used for mining rewards

#### Security Best Practices
1. **Backup your private key** - If lost, you cannot access mining rewards
2. **Use different addresses** - Generate separate keys for different mining operations
3. **Secure storage** - Store private keys offline or in encrypted storage
4. **Never share private keys** - Only share your address for receiving rewards

#### Checking Your Mining Rewards
```bash
# Check balance for your address
./poaid balance --addr=$(cat keys/poai_address.txt) --data-dir=data1

# Check balance on a different node
./poaid balance --addr=$(cat keys/poai_address.txt) --data-dir=data2
```

**Note**: Balance checking works even when mining nodes are running. The command will show an error if the database is locked by another process.

#### Key Recovery
If you lose your private key, you cannot recover access to mining rewards. Always:
- Keep multiple secure backups
- Use hardware wallets for large amounts (when supported)
- Test your backup recovery process

### Transaction System

PoAI includes a complete transaction system for secure transfers between wallets.

#### Creating Transactions
```bash
# Send 1000 POAI to another address
./poaid send --to=RECIPIENT_ADDRESS_HERE \
             --amount=1000 \
             --privkey=YOUR_PRIVATE_KEY_HERE
```

#### Transaction Security Features
- **Cryptographic Signatures**: Only the private key holder can spend funds
- **Transaction Hash**: Unique identifier for each transaction
- **Balance Verification**: Network verifies sufficient funds before execution
- **Double-Spend Protection**: Prevents spending the same funds twice
- **Immutable Ledger**: Once confirmed, transactions cannot be reversed

#### Transaction Workflow
1. **Create**: Generate a transaction with recipient and amount
2. **Sign**: Cryptographically sign with your private key
3. **Broadcast**: Send to the network for inclusion in blocks
4. **Confirm**: Transaction is included in a mined block

**Note**: Currently, transactions are created and signed but need to be manually added to the mempool for broadcasting. Full mempool integration is planned for future releases.

## Protocol Overview

### Quiz-Based Consensus

1. **Epoch math**  
   - `EpochBlocks` (e.g. 2016 on main-net) defines each epoch's length.  
   - Seed = block-hash of the _previous_ epoch's last block.  

2. **EpochKey derivation**  
   ```text
   epochKey = Keccak256( seed ∥ epochIndex )
   ```

3. **Procedural quiz generation**
   - Deterministic quiz generation based on block height and epoch key
   - No external dataset files required
   - Examples: math problems, logic puzzles, text completion tasks

4. **Forward-pass & loss reduction**
   - Fixed, "eval"-mode AI model (e.g. TinyLlama-1.1B) runs inference on generated quizzes
   - Compute scalar loss → hash-reduce to 256-bit integer ℓ̂
   - Block is valid if `ℓ̂ < T` (difficulty target)

### Block Production & Verification

* **Miner** runs a single-goroutine loop driven by `state.SubscribeHeads()`, attempts quizzes on each tip, and broadcasts valid blocks via libp2p
* **Validator** replays the same quiz pipeline in `validator/verify.go` and rejects any block with mismatched ℓ̂ or header

## Economic Model

* **Block subsidy**: 5 POAI per block (halving every 4 years)
* **Transaction fees**: gas charged in POAI for all EVM calls (planned)
* **Inference-job fees**: posters lock POAI in `InferenceMarket.sol`; miners stake additional POAI and earn payments upon successful proof or get slashed on fraud (planned)
* **Stake-slash**: invalid blocks lose GPU cost; mis-served jobs burn 90 % of worker stake, 10 % to challenger (planned)
* **Security budget**: subsidy + fees + job revenues align miner incentives to remain honest

## Current Status & Roadmap

| Component                                         | Status            |
| ------------------------------------------------- | ----------------- |
| Quiz pipeline & procedural generation             | ✅ Complete       |
| Single-worker mining loop & orphan-pool import    | ✅ Complete       |
| Difficulty retarget (core/difficulty.go)          | ✅ Complete        |
| Persistent on-disk DB + pruning (core/storage)    | ✅ Complete       |
| libp2p gossip & peer discovery (net/p2p.go)       | ⚪️ Stub           |
| **Wallet system & balance checking**              | ✅ **Complete**   |
| **Transaction creation & signing**                | ✅ **Complete**   |
| **Real LLM inference (llama.cpp)**                | ✅ **Complete**   |
| Transaction broadcasting & mempool integration    | ⚪️ Planned        |
| InferenceMarket.sol + Go bindings                 | ⚪️ In progress    |
| Metrics & Grafana exporter                        | ⚪️ Missing        |
| Documentation & spec polish                       | ✅ **Updated**    |

## Troubleshooting

### General Issues
- **go.mod not found**: Ensure you're in the repo root directory (run `ls` to see `go.mod`). Avoid running commands from parent or nested dirs.
- **Model download issues**: Use the auth-required options above; Hugging Face enforces this for large files.
- **Mining too slow**: Increase `--target` (easier difficulty) or use GPU layers.
- **Flag errors**: Use `--peer-multiaddr` (not `--multiaddr`) for connecting to peers.
- **Build errors on Linux**: If you get "malformed import path" with `*.go`, use explicit file listing:
  ```bash
  # Instead of: go build -tags llama -o poaid cmd/poaid/*.go
  # Use: go build -tags llama -o poaid cmd/poaid/main.go cmd/poaid/cli.go
  ```

### Key and Mining Issues
- **Invalid miner address**: Ensure the address is a 40-character hex string (20 bytes). Use `./poaid generate-key` to create a valid address.
- **Address format error**: The `--miner-address` flag expects hex format without the `0x` prefix.
- **Permission denied on key files**: The private key file has restricted permissions (600). This is intentional for security.
- **No mining rewards**: Check that you're using the correct `--miner-address` and that blocks are being successfully mined.

### CLI Commands

PoAI provides a comprehensive CLI with multiple commands:

#### Available Commands
```bash
# Show all available commands
./poaid help

# Run as mining daemon
./poaid [flags]

# Generate new keypair
./poaid generate-key [flags]

# Check wallet balance
./poaid balance [flags]

# Send transaction
./poaid send [flags]

# Show help
./poaid help
```

#### Command Examples
```bash
# Generate a new keypair
./poaid generate-key

# Generate and save keys to files
./poaid generate-key --save --output-dir=./keys

# Check balance for an address
./poaid balance --addr=YOUR_ADDRESS_HERE --data-dir=data1

# Send 1000 POAI to another address
./poaid send --to=RECIPIENT_ADDRESS_HERE \
             --amount=1000 \
             --privkey=YOUR_PRIVATE_KEY_HERE

# Start mining with your address
./poaid --miner-address=YOUR_ADDRESS --target=500 --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf
```

#### Command Flags
- **Daemon Flags**: `--model-path`, `--target`, `--data-dir`, `--p2p-port`, `--peer-multiaddr`, `--miner-address`
- **Generate Key Flags**: `--save`, `--output-dir`
- **Balance Flags**: `--addr`, `--data-dir`
- **Send Flags**: `--to`, `--amount`, `--privkey`

- Open an issue with logs for other problems.

## Contributing
Open issues for bugs, logs, or suggestions. PRs welcome, especially for setup improvements or new features like transactions. Target v0.3.0 for economic layer.

## License & Code of Conduct

* **Apache 2.0** for all code.

---

*PoAI unites provable AI work with blockchain security. Together we'll finish the last mile to a live test-net and beyond.* 
