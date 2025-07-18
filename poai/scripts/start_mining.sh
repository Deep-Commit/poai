#!/bin/bash

# PoAI Mining Starter Script
# This script demonstrates how to start mining with generated keys

set -e

echo "⛏️  PoAI Mining Starter"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if poaid binary exists
if [ ! -f "./poaid" ]; then
    echo "❌ Error: poaid binary not found!"
    echo "   Please build the daemon first:"
    echo "   go build -o poaid ./cmd/poaid"
    echo ""
    exit 1
fi

# Check if model exists
if [ ! -f "./models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf" ]; then
    echo "❌ Error: LLM model not found!"
    echo ""
    echo "📥 Downloading the LLM model..."
    echo "   Note: Hugging Face requires authentication for large files"
    echo ""
    
    # Check if HF token is provided
    if [ -z "$HF_TOKEN" ]; then
        echo "🔑 Please provide your Hugging Face token:"
        echo "   1. Sign up at https://huggingface.co/join"
        echo "   2. Create a token at https://huggingface.co/settings/tokens"
        echo "   3. Set the token: export HF_TOKEN=hf_XXXX"
        echo ""
        echo "   Or run: HF_TOKEN=hf_XXXX ./scripts/start_mining.sh"
        echo ""
        exit 1
    fi
    
    # Create models directory
    mkdir -p models
    
    # Download with authentication
    echo "⬇️  Downloading TinyLlama model (this may take a few minutes)..."
    if curl -L -H "Authorization: Bearer $HF_TOKEN" \
        -o models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
        https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf; then
        echo "✅ Model downloaded successfully!"
    else
        echo "❌ Failed to download model. Please check your HF_TOKEN and try again."
        exit 1
    fi
    echo ""
fi

# Check if keys exist
if [ ! -f "./keys/poai_address.txt" ]; then
    echo "❌ Error: Mining keys not found!"
    echo "   Please generate keys first:"
    echo "   ./scripts/generate_keys.sh"
    echo ""
    exit 1
fi

# Read the miner address
MINER_ADDRESS=$(cat ./keys/poai_address.txt)

echo "🔑 Using miner address: $MINER_ADDRESS"
echo ""

# Set default values
TARGET=${1:--1000000000000000000}
DATA_DIR=${2:-data1}
P2P_PORT=${3:-4001}
GPU_LAYERS=${4:-0}

echo "📊 Mining Configuration:"
echo "   Target: $TARGET (lower = easier)"
echo "   Data Directory: $DATA_DIR"
echo "   P2P Port: $P2P_PORT"
echo "   GPU Layers: $GPU_LAYERS"
echo ""

echo "🚀 Starting PoAI miner..."
echo "   Press Ctrl+C to stop"
echo ""

# Start the miner
./poaid --miner-address="$MINER_ADDRESS" \
        --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
        --target="$TARGET" \
        --data-dir="$DATA_DIR" \
        --p2p-port="$P2P_PORT" \
        --gpu-layers="$GPU_LAYERS" 