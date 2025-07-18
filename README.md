# PoAI: Proof of AI

![Demo](demo.gif)

## Status

**What works:**
* ✅ LLM inference for proof of AI
* ✅ Deterministic LLM inference
* ✅ Node daemon `poaid`
* ✅ P2P networking with libp2p
* ✅ Procedural quiz generation for mining datasets (dynamic, verifiable Q&A without external files)
* ✅ Block mining and validation using LLM outputs
* ✅ Basic difficulty retargeting
* ✅ Local testnet with multiple nodes syncing chains and competing for blocks

**What doesn't work yet:**
* ❌ Transactions layer (send/receive tokens, secured by blockchain; subsidies transferable via tx — planned next)
* ❌ On-chain governance (deprecated; procedural generation handles datasets)
* ❌ Full EVM compatibility for smart contracts
* ❌ Advanced features like staking or AI model upgrades

Check repo traffic trends: [Traffic Graphs](https://github.com/Deep-Commit/poai/graphs/traffic)

## Abstract

PoAI is a blockchain protocol that replaces traditional Proof of Work (PoW) with Proof of AI (PoAI), where miners perform verifiable forward passes on large language models (LLMs) to generate blocks. This leverages AI computation for consensus, making mining useful for AI training/inference while maintaining security. Datasets for inference are now procedurally generated (e.g., dynamic math quizzes), ensuring determinism and eliminating the need for curated datasets or on-chain governance.

## Introduction

Traditional blockchains like Bitcoin use energy-intensive hashing for PoW. PoAI shifts this to AI workloads: Miners run inference on procedural quizzes using a fixed model (e.g., TinyLlama-1.1B), and valid outputs (meeting difficulty targets) create blocks. Rewards (subsidies) are earned automatically on successful mines. The chain is EVM-compatible in design, with future support for transactions and contracts.

Why now? With GGUF models and go-llama.cpp, deterministic AI proofs are feasible locally. This repo implements the core node in Go.

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

If this fails (e.g., 403 Forbidden or redirects to login), use Option 2 or 3.

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

For CPU-only:
```bash
go build -o poaid ./cmd/poaid
```

For macOS with Metal GPU (optional):
```bash
CGO_LDFLAGS="-framework Metal -framework Foundation" go build -o poaid ./cmd/poaid
```

If you encounter "go.mod not found", double-check your directory—you must be in the root where `go.mod` exists.

### Run a Local Testnet
This sets up a local blockchain with LLM-based mining on procedurally generated quizzes. Use `--target=500` for easier mining (lower values = harder difficulty; adjusts automatically like Bitcoin). Run all commands from the repo root.

**Note**: Procedural quiz generation is enabled by default. If you want to use a test corpus instead, add `--test-corpus=./dataset/testdata` to the commands below.

#### Start Node 1
```bash
./poaid --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --gpu-layers=0 --data-dir=data1 --p2p-port=4001 --target=500
```
- Note the peer ID from logs (e.g., `/ip4/127.0.0.1/tcp/4001/p2p/<PEER_ID>`).

#### Start Node 2 (Connect to Node 1)
Replace `<PEER_ID_FROM_NODE_1>` with the actual ID:
```bash
./poaid --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --gpu-layers=0 --data-dir=data2 --p2p-port=4002 --target=500 --peer-multiaddr="/ip4/127.0.0.1/tcp/4001/p2p/<PEER_ID_FROM_NODE_1>"
```

#### Blockchain Settings and Verification
- **Difficulty/Targets**: Set via `--target` (e.g., 500 for testing; 1000+ for realism). Retargets every few blocks based on timestamps (see `core/difficulty.go`). Blocks mine when LLM inference loss (hashed) < target.
- **Subsidies/Rewards**: Automatic on mined blocks (fixed amount, halving model). Rewards credit to miner's address; future transactions will enable sending/receiving.
- **Procedural Quizzes**: Mining auto-generates deterministic quizzes (e.g., math problems seeded by block height) for LLM inference—no external files needed.
- Verify: Watch logs for "Generated quiz: ...", "Block mined!", and chain sync. Nodes compete; successful mining earns subsidies.
- Troubleshooting: If LLM fails, check model path/threads. Data persists in `data1`/`data2` for restarts. If commands fail, confirm you're in the repo root.

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
| Transactions layer (send/receive tokens)          | ⚪️ Planned        |
| InferenceMarket.sol + Go bindings                 | ⚪️ In progress    |
| Metrics & Grafana exporter                        | ⚪️ Missing        |
| Documentation & spec polish                       | ⚪️ Needs work     |

## Troubleshooting
- **go.mod not found**: Ensure you're in the repo root directory (run `ls` to see `go.mod`). Avoid running commands from parent or nested dirs.
- **Model download issues**: Use the auth-required options above; Hugging Face enforces this for large files.
- **Mining too slow**: Increase `--target` (easier difficulty) or use GPU layers.
- **Flag errors**: Use `--peer-multiaddr` (not `--multiaddr`) for connecting to peers.
- Open an issue with logs for other problems.

## Contributing
Open issues for bugs, logs, or suggestions. PRs welcome, especially for setup improvements or new features like transactions. Target v0.3.0 for economic layer.

## License & Code of Conduct

* **Apache 2.0** for all code.

---

*PoAI unites provable AI work with blockchain security. Together we'll finish the last mile to a live test-net and beyond.* 
