#!/usr/bin/env bash
set -e
export POAI_CONFIG=config/testnet.toml

go run ./cmd/poaid init --datadir node1
go run ./cmd/poaid start --datadir node1 --rpc :8545 > logs/node1.log 2>&1 &
NODE_PID=$!

go run ./cmd/minectl start --datadir node1 --quiz-batch-cap 4 > logs/miner.log 2>&1 &

echo "⏳  waiting 3 minutes for blocks..."
sleep 180
kill $NODE_PID
grep "new block" logs/node1.log | tail 