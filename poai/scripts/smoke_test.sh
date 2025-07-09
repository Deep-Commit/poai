#!/bin/bash

# POAI Smoke Test Script
# Runs a single miner daemon for stability testing

set -e

cleanup() {
    echo "\n🛑 Stopping miner..."
    pkill -f bin/poaid || true
}
trap cleanup EXIT

echo "🚀 Starting POAI smoke test with 1 miner..."

# Build the daemon
echo "📦 Building poaid..."
cd "$(dirname "$0")/.."
go build -o bin/poaid ./cmd/poaid

# Create logs directory
mkdir -p logs

# Ensure test corpus exists
if [ ! -f "dataset/testdata/sigma_tiny.tar" ]; then
    echo "📝 Generating test corpus..."
    cd dataset/testdata
    go run gen_fixture.go
    cd ../..
fi

# Run a single miner in the background
echo "⛏️  Starting miner with target=10..."
./bin/poaid \
    -target=10 \
    -epoch-blocks=20 \
    -batch-size=2 \
    -corpus-size=5000000000 \
    -test-corpus=dataset/testdata \
    2>&1 | tee logs/miner_1_$(date +%Y%m%d_%H%M%S).log &

echo "🟢 Miner launched. Press Ctrl+C to stop."
wait

echo "✅ Smoke test completed!" 