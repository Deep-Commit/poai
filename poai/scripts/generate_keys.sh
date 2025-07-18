#!/bin/bash

# PoAI Key Generation Script
# This script generates a new keypair for PoAI mining and provides setup instructions

set -e

echo "🔑 PoAI Key Generation Utility"
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

# Create keys directory
KEYS_DIR="./keys"
mkdir -p "$KEYS_DIR"

echo "📁 Generating keys in: $KEYS_DIR"
echo ""

# Generate keys using the built-in command
./poaid generate-key --save --output-dir="$KEYS_DIR"

echo ""
echo "📝 Note: When you start mining, you'll need to download the LLM model."
echo "   If you haven't already, you can:"
echo ""
echo "   1. Set your Hugging Face token:"
echo "      export HF_TOKEN=hf_XXXX"
echo ""
echo "   2. Use the mining starter script which will download it automatically:"
echo "      ./scripts/start_mining.sh"
echo ""

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 Next Steps:"
echo ""
echo "1. 📖 Read the generated files:"
echo "   cat $KEYS_DIR/miner_config.txt"
echo ""
echo "2. ⛏️  Start mining with your new address:"
echo "   ./poaid --miner-address=\$(cat $KEYS_DIR/poai_address.txt) \\"
echo "           --model-path=models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \\"
echo "           --target=500 \\"
echo "           --data-dir=data1 \\"
echo "           --p2p-port=4001"
echo ""
echo "3. 🔒 Security:"
echo "   • Keep $KEYS_DIR/poai_private_key.txt secure"
echo "   • Never share your private key"
echo "   • Use the address for mining rewards"
echo ""
echo "✅ Key generation complete!" 