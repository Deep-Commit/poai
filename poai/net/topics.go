package net

import "poai/core"

const (
	TopicNewHead   = "poai/newhead/1"
	TopicBlockReq  = "poai/blockreq/1"
	TopicBlockResp = "poai/blockresp/1"
)

type NewHeadMsg struct {
	Height uint64
	Hash   [32]byte
	Parent [32]byte
}

type BlockRequest struct {
	From uint64 // inclusive
	To   uint64 // inclusive, max 512 for DOS safety
}

type BlockResponse struct {
	Blocks []*core.Block // your canonical block type
}
