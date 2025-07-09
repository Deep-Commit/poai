// Package net implements libp2p gossip and sync for POAI.
package net

import (
	"context"
	"fmt"
	"log"
	"time"

	"encoding/json"
	"poai/core"

	"runtime/debug"
	"sync/atomic"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const BlockTopic = "poai-blocks"
const maxWireBlock = 256 * 1024 // 256 KB, adjust as needed

// Add Chain reference to P2PNode for sync
// P2PNode represents a minimal libp2p node for block gossip and sync.
type P2PNode struct {
	Host     host.Host
	PubSub   *pubsub.PubSub
	BlockSub *pubsub.Subscription
	Chain    *core.Chain

	bestKnownHeight uint64 // Track best known height from peers (atomic)
}

// NewP2PNode creates a new libp2p node, joins the block gossip topic, and enables mDNS discovery.
func NewP2PNode(ctx context.Context, listenPort int, chain *core.Chain) (*P2PNode, error) {
	h, err := libp2p.New(libp2p.ListenAddrStrings(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
	))
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	blockSub, err := ps.Subscribe(BlockTopic)
	if err != nil {
		return nil, err
	}

	n := &P2PNode{
		Host:     h,
		PubSub:   ps,
		BlockSub: blockSub,
		Chain:    chain,
	}

	// mDNS for local peer discovery
	notifee := &mdnsNotifee{}
	mdns.NewMdnsService(h, "poai-mdns", notifee)
	log.Printf("[P2P] mDNS peer discovery enabled")

	// Periodically log connected peers
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			peers := h.Network().Peers()
			ids := make([]string, 0, len(peers))
			for _, p := range peers {
				ids = append(ids, p.String())
			}
			log.Printf("[P2P] Connected peers: %v", ids)
		}
	}()

	// Periodically announce our head to help lagging peers catch up
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var lastHeight uint64 = 0
		for range ticker.C {
			h := chain.CurrentHeight()
			if h == lastHeight {
				continue
			}
			lastHeight = h
			blk := chain.BlockByHeight(h)
			if blk == nil {
				continue
			}
			n.AnnounceHead(blk)
		}
	}()

	// --- Chain sync topics ---
	newHeadSub, err := ps.Subscribe(TopicNewHead)
	if err != nil {
		log.Fatal(err)
	}
	go n.handleNewHead(ctx, newHeadSub)

	subReq, _ := ps.Subscribe(TopicBlockReq)
	go n.handleBlockReq(ctx, subReq)

	subResp, _ := ps.Subscribe(TopicBlockResp)
	go n.handleBlockResp(ctx, subResp)

	n.HandleBlockMessages(ctx)

	return n, nil
}

// PublishBlock publishes a serialized block to the block gossip topic.
func (n *P2PNode) PublishBlock(ctx context.Context, data []byte) error {
	return n.PubSub.Publish(BlockTopic, data)
}

// HandleBlockMessages listens for new block messages and calls the provided handler with the data.
func (n *P2PNode) HandleBlockMessages(ctx context.Context) {
	go func() {
		for {
			msg, err := n.BlockSub.Next(ctx)
			if err != nil {
				log.Printf("[P2P] BlockSub error: %v", err)
				return
			}
			log.Printf("[P2P] BlockSub message from %s (self: %v)", msg.ReceivedFrom, msg.ReceivedFrom == n.Host.ID())
			// Ignore messages from self
			if msg.ReceivedFrom == n.Host.ID() {
				continue
			}
			if len(msg.Data) > maxWireBlock {
				log.Printf("[P2P] oversized block msg (%d bytes) from %s", len(msg.Data), msg.ReceivedFrom)
				continue
			}
			var blk core.Block
			if err := json.Unmarshal(msg.Data, &blk); err != nil {
				log.Printf("[P2P] Failed to decode block: %v", err)
				continue
			}
			log.Printf("[P2P] Received block #%d from peer", blk.Header.Height)
			if err := n.Chain.ImportBlock(&blk); err != nil {
				log.Printf("[P2P] Failed to import block #%d: %v", blk.Header.Height, err)
			} else {
				log.Printf("[P2P] Imported block #%d from peer", blk.Header.Height)
			}
		}
	}()
}

// AnnounceHead publishes a NewHeadMsg for a freshly-minted block header.
func (n *P2PNode) AnnounceHead(b *core.Block) {
	msg := NewHeadMsg{
		Height: b.Header.Height,
		Hash:   b.Header.Hash(),
		Parent: b.Header.ParentHash,
	}
	payload, _ := json.Marshal(msg)
	log.Printf("[P2P] NewHead %d %x...", msg.Height, msg.Hash[:4])
	n.PubSub.Publish(TopicNewHead, payload)
}

// handleNewHead processes inbound NewHead messages and requests missing blocks if behind.
func (n *P2PNode) handleNewHead(ctx context.Context, sub *pubsub.Subscription) {
	for {
		raw, err := sub.Next(ctx)
		if err != nil {
			return
		}
		var msg NewHeadMsg
		if err := json.Unmarshal(raw.Data, &msg); err != nil {
			continue
		}
		if msg.Height == 0 {
			continue
		}
		best := n.Chain.CurrentHeight()
		if msg.Height > atomic.LoadUint64(&n.bestKnownHeight) {
			atomic.StoreUint64(&n.bestKnownHeight, msg.Height)
		}
		if msg.Height <= best {
			continue
		}
		log.Printf("[SYNC] NewHead %d > local %d, requesting blocks %d-%d", msg.Height, best, best+1, msg.Height)
		req := BlockRequest{From: best + 1, To: msg.Height}
		payload, _ := json.Marshal(req)
		n.PubSub.Publish(TopicBlockReq, payload)
	}
}

// BestKnownHeight returns the highest height seen from peers (atomic).
func (n *P2PNode) BestKnownHeight() uint64 {
	return atomic.LoadUint64(&n.bestKnownHeight)
}

// handleBlockReq serves block requests from peers.
func (n *P2PNode) handleBlockReq(ctx context.Context, sub *pubsub.Subscription) {
	for {
		raw, _ := sub.Next(ctx)
		var req BlockRequest
		_ = json.Unmarshal(raw.Data, &req)
		if req.To-req.From > 512 {
			req.To = req.From + 512
		}
		log.Printf("[SYNC] Serving block request for %d-%d", req.From, req.To)
		blocks := make([]*core.Block, 0, req.To-req.From+1)
		for h := req.From; h <= req.To; h++ {
			if blk := n.Chain.BlockByHeight(h); blk != nil {
				blocks = append(blocks, blk)
				log.Printf("[SYNC] Sending block #%d in response to request", h)
			} else {
				log.Printf("[SYNC] Block #%d not found for request", h)
			}
		}
		resp := BlockResponse{Blocks: blocks}
		data, _ := json.Marshal(resp)
		n.PubSub.Publish(TopicBlockResp, data)
	}
}

// handleBlockResp consumes block responses and applies them to the chain.
func (n *P2PNode) handleBlockResp(ctx context.Context, sub *pubsub.Subscription) {
	for {
		raw, _ := sub.Next(ctx)
		var resp BlockResponse
		_ = json.Unmarshal(raw.Data, &resp)
		for _, blk := range resp.Blocks {
			log.Printf("[SYNC] Received block #%d in response", blk.Header.Height)
			log.Printf("[SYNC] Importing block #%d from peer", blk.Header.Height)
			if err := n.Chain.ImportBlock(blk); err != nil {
				log.Printf("[SYNC] Failed to import block #%d: %v", blk.Header.Height, err)
				continue
			}
			// Optionally: state transition, fork-choice, etc.
		}
	}
}

// After mining a block, publish it to the P2P network
// (This should be called after a block is mined and accepted)
func (n *P2PNode) PublishBlockFromStruct(b *core.Block) error {
	if len(n.Host.Network().Peers()) == 0 {
		log.Printf("[P2P] No peers connected, skipping block publication.")
		return nil
	}
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	log.Printf("[P2P] Publishing block #%d to network", b.Header.Height)
	return n.PublishBlock(context.Background(), data)
}

// RequestBlockByHash requests a block with the given parent hash from peers.
func (n *P2PNode) RequestBlockByHash(parentHash [32]byte) {
	log.Printf("[DEBUG] RequestBlockByHash: ENTER parentHash=%x", parentHash[:8])
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] RequestBlockByHash: PANIC: %v", r)
			log.Printf("[ERROR] RequestBlockByHash: stack trace:\n%s", debugStack())
		}
		log.Printf("[DEBUG] RequestBlockByHash: EXIT parentHash=%x", parentHash[:8])
	}()
	// Try to find the height of the missing parent
	var orphanHeight uint64 = 0
	found := false
	for h := uint64(0); h <= n.Chain.CurrentHeight(); h++ {
		blk := n.Chain.BlockByHeight(h)
		if blk != nil && blk.Hash() == parentHash {
			// Already have it
			return
		}
	}
	// Try to find the orphan that references this parent
	n.Chain.OrphanMu.RLock()
	for _, orphans := range n.Chain.OrphanPool {
		for _, orphan := range orphans {
			if orphan.Header.ParentHash == parentHash {
				orphanHeight = orphan.Header.Height
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	n.Chain.OrphanMu.RUnlock()
	if found && orphanHeight > 1 {
		from := uint64(1)
		to := orphanHeight
		req := BlockRequest{From: from, To: to}
		payload, _ := json.Marshal(req)
		n.PubSub.Publish(TopicBlockReq, payload)
		log.Printf("[SYNC] Requested parent block %x (range %d-%d)", parentHash[:8], from, to)
		return
	}
	// Fallback: request last 100 blocks
	best := n.Chain.CurrentHeight()
	var from uint64 = 0
	if best > 100 {
		from = best - 100
	}
	to := best
	req := BlockRequest{From: from, To: to}
	payload, _ := json.Marshal(req)
	n.PubSub.Publish(TopicBlockReq, payload)
	log.Printf("[SYNC] Requested parent block %x (range %d-%d)", parentHash[:8], from, to)
}

// mDNS Notifee for peer discovery
type mdnsNotifee struct{}

func (n *mdnsNotifee) HandlePeerFound(info peer.AddrInfo) {
	log.Printf("[P2P] mDNS discovered peer: %s", info.ID.String())
}

// debugStack returns the current stack trace as a string.
func debugStack() string {
	return string(debug.Stack())
}
