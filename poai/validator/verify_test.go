package validator_test

import (
	"poai/core/config"
	"poai/core/header"
	"poai/core/keyschedule"
	"poai/dataset"
	"poai/validator"
	"testing"
)

type dummyDB struct{ hdrs map[uint64]*header.Header }

func (d *dummyDB) HeaderByHeight(h uint64) *header.Header { return d.hdrs[h] }
func (d *dummyDB) Height() uint64 {
	// Return the highest height in the map
	max := uint64(0)
	for h := range d.hdrs {
		if h > max {
			max = h
		}
	}
	return max
}

func TestVerifyBlock_TinyCorpus(t *testing.T) {
	config.EpochBlocks = 20
	config.BatchSize = 1
	config.CorpusSize = 1

	t.Log("Loading test corpus...")
	if err := dataset.LoadTestCorpus("../dataset/testdata"); err != nil {
		t.Fatal(err)
	}

	t.Log("Building dummy headers...")
	db := &dummyDB{hdrs: map[uint64]*header.Header{}}
	for h := uint64(0); h < 40; h++ {
		db.hdrs[h] = &header.Header{Height: h}
	}

	t.Log("Computing epochKey, indices, and fetching records...")
	epochKey := keyschedule.EpochKey(0, db)
	t.Logf("Test epochKey: %x", epochKey)
	idx := dataset.Indexes(db.hdrs[19].Hash(), config.BatchSize)
	t.Logf("Indices: %v", idx)
	t.Log("About to call Fetch")
	recs, _ := dataset.Fetch(idx, epochKey)
	t.Log("Returned from Fetch")
	t.Logf("Fetched records: %+v", recs)
	loss := validator.ForwardPass(recs, validator.TestTinyWeights)
	lhat := validator.LossToInt(loss)
	t.Logf("Computed loss: %v, lhat: %v", loss, lhat)

	t.Log("Building block and running VerifyBlock...")
	blk := &validator.Block{
		Height: 19,
		Header: struct{ Lhat int64 }{Lhat: lhat},
	}

	if err := validator.VerifyBlock(blk, db); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	t.Log("VerifyBlock passed!")
}
