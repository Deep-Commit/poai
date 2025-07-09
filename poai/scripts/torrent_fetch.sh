#!/bin/bash
# Fetch the Σ corpus file for POAI testnet/mainnet

set -e

CORPUS_URL="https://example.com/Σtiny.tar"
TORRENT_FILE="./dataset/corpus/Σ.tar.torrent"
DEST="./dataset/corpus/Σtiny.tar"

if command -v aria2c >/dev/null 2>&1; then
    aria2c --seed-time=0 "$TORRENT_FILE" -d "$(dirname "$DEST")"
else
    echo "aria2c not found, trying ipfs..."
    ipfs pin add --progress "<CID-for-Σtiny.tar>"
fi 