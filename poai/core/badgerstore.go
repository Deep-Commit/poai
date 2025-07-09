package core

import (
	"path/filepath"
	"strconv"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	db *badger.DB
}

func OpenBadgerStore(dataDir string) (*BadgerStore, error) {
	dbPath := filepath.Join(dataDir, "badger")
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(nil))
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

func (s *BadgerStore) PutBlock(height uint64, block *Block) error {
	key := []byte("block:" + strconv.FormatUint(height, 10))
	val, err := block.Encode()
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(key, val); err != nil {
			return err
		}
		// Update tip
		tipKey := []byte("chain:tip")
		tipVal := []byte(strconv.FormatUint(height, 10))
		return txn.Set(tipKey, tipVal)
	})
}

func (s *BadgerStore) GetBlock(height uint64) (*Block, error) {
	key := []byte("block:" + strconv.FormatUint(height, 10))
	var block *Block
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			b, err := DecodeBlock(val)
			if err != nil {
				return err
			}
			block = b
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (s *BadgerStore) DeleteBlock(height uint64) error {
	key := []byte("block:" + strconv.FormatUint(height, 10))
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *BadgerStore) GetTipHeight() (uint64, error) {
	var height uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("chain:tip"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			h, err := strconv.ParseUint(string(val), 10, 64)
			if err != nil {
				return err
			}
			height = h
			return nil
		})
	})
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (s *BadgerStore) PruneBlocks(keepN uint64, tip uint64) error {
	minKeep := uint64(0)
	if tip >= keepN {
		minKeep = tip - keepN + 1
	}
	return s.db.Update(func(txn *badger.Txn) error {
		for h := uint64(0); h < minKeep; h++ {
			key := []byte("block:" + strconv.FormatUint(h, 10))
			err := txn.Delete(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}
		}
		return nil
	})
}

func (s *BadgerStore) Close() error {
	return s.db.Close()
}
