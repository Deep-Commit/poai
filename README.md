# PoAI: A "Proof-of-AI" Sovereign EVM-Compatible Blockchain

> **Abstract**  
> PoAI replaces hash-based proof-of-work with verifiable inference: miners grind on forward-passes of a fixed AI model over encrypted minibatches, producing blocks when the batch loss (hash-reduced to a 256-bit integer) falls below a network-wide difficulty target. An on-chain DAO rotates both the reference model weights and the hidden dataset each epoch, while an upcoming inference-market smart contract lets users post paid AI jobs secured by stake-slash guarantees.

---

## 1 Introduction

Modern blockchains rely on wasted SHA-256 or Ethash cycles to secure consensus. PoAI harnesses that compute for _useful_ work—running AI inferences—and still preserves the "straight, no-fork" security model of Bitcoin:

- **Deterministic proof**: every block header includes a "quiz loss" that any full node can recompute via a forward-only pass.
- **Difficulty retarget**: the loss threshold auto-adjusts to maintain a fixed block cadence.
- **On-chain governance**: Solidity-based DAOs (`ModelRegistry.sol`, `DatasetDAO.sol`) hot-swap the model snapshot and encrypted corpus on a timelocked, epoch-boundary schedule.
- **EVM compatibility**: deploy and interact with PoAI contracts exactly as you would on Ethereum.

---

## 2 Protocol Overview

### 2.1 Quiz-Based Consensus

1. **Epoch math**  
   - `EpochBlocks` (e.g. 2016 on main-net) defines each epoch's length.  
   - Seed = block-hash of the _previous_ epoch's last block.  

2. **EpochKey derivation**  
   ```text
   epochKey = Keccak256( seed ∥ epochIndex )
   ```

3. **Index selection**

   * A VRF or simple hash selects a minibatch of K sample IDs from the encrypted dataset.

4. **Fetch & decrypt**

   * Records encrypted under AES-GCM with per-record keys derived from `epochKey`.
   * Deterministic `mmapSlice` ➔ verify SHA-256 header ➔ `aesgcm.Open`.

5. **Forward-pass & loss reduction**

   * Fixed, "eval"-mode AI model (e.g. Llama 7B) runs K sample forward-passes.
   * Compute scalar loss → hash-reduce to 256-bit integer ℓ̂.
   * Block is valid if `ℓ̂ < T` (difficulty target).

### 2.2 Block Production & Verification

* **Miner** runs a single-goroutine loop driven by `state.SubscribeHeads()`, attempts quizzes on each tip, and broadcasts valid blocks via libp2p (stubbed today).
* **Validator** replays the same quiz pipeline in `validator/verify.go` and rejects any block with mismatched ℓ̂ or header.

---

## 3 Economic Model

* **Block subsidy**: 5 POAI per block (halving every 4 years).
* **Transaction fees**: gas charged in POAI for all EVM calls.
* **Inference-job fees**: posters lock POAI in `InferenceMarket.sol`; miners stake additional POAI and earn payments upon successful proof or get slashed on fraud.
* **Stake-slash**: invalid blocks lose GPU cost; mis-served jobs burn 90 % of worker stake, 10 % to challenger.
* **Security budget**: subsidy + fees + job revenues align miner incentives to remain honest.

---

## 4 On-Chain Governance

### 4.1 ModelRegistry.sol

* Propose new weight snapshot (CID, parameter count, SHA-256 hash, activationEpoch).
* Token-weighted vote + timelock ensures orderly upgrades.

### 4.2 DatasetDAO.sol

* Propose encrypted dataset CID + key-hash for each epoch.
* Commit-reveal pattern: key is only published at activation boundary to prevent precomputation.

Full-nodes and miners subscribe to events, prefetch assets, and switch atomically at each epoch boundary.

---

## 5 Current Status & Roadmap

| Component                                         | Status            |
| ------------------------------------------------- | ----------------- |
| Quiz pipeline & golden-vector tests               | ✅ Complete       |
| Single-worker mining loop & orphan-pool import    | ✅ Complete       |
| EVM governance stubs (ModelRegistry & DatasetDAO) | ✅ Deployed (stub)|
| Difficulty retarget (core/difficulty.go)          | ⚪️ Blank          |
| Persistent on-disk DB + pruning (core/storage)    | ⚪️ Stub           |
| libp2p gossip & peer discovery (net/p2p.go)       | ⚪️ Stub           |
| InferenceMarket.sol + Go bindings                 | ⚪️ In progress    |
| Metrics & Grafana exporter                        | ⚪️ Missing        |
| Documentation & spec polish                       | ⚪️ Needs work     |

---

## 6 Getting Involved

We're polishing PoAI's core before open-sourcing the repo. If you:

* Write Go, Solidity, or Rust
* Build distributed systems or GPU kernels
* Are passionate about decentralized AI

…join our Discord at **gswarm.dev** or open an Issue/PR on GitHub once the repo goes live.

---

## 7 License & Code of Conduct

* **Apache 2.0** for all code.
---

*PoAI unites provable AI work with blockchain security. Stay tuned for the public repo — together we'll finish the last mile to a live test-net and beyond.* 
