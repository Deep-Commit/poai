package dataset

type IndexEntry struct {
	Offset int64
	Size   int64
	Hash   [32]byte
}

// indexTable is populated at node startup from Σ.idx
var indexTable []IndexEntry

func SetIndexTable(tab []IndexEntry) { indexTable = tab }
