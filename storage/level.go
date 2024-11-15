package storage

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelStorage implements the Storage interface using LevelDB.
type LevelStorage struct {
	db *leveldb.DB
}

// NewLevelStorage initializes a LevelStorage instance with the specified path.
func NewLevelStorage(path string) (*LevelStorage, error) {
	db, err := leveldb.OpenFile(path, &opt.Options{})
	if err != nil {
		return nil, err
	}
	return &LevelStorage{db: db}, nil
}

// Put stores a key-value pair in LevelDB.
func (s *LevelStorage) Put(key string, value []byte) error {
	return s.db.Put([]byte(key), value, nil)
}

// Get retrieves a value by its key from LevelDB.
func (s *LevelStorage) Get(key string) ([]byte, error) {
	value, err := s.db.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return nil, errors.New("key not found")
	}
	return value, err
}

// Delete removes a key-value pair from LevelDB.
func (s *LevelStorage) Delete(key string) error {
	return s.db.Delete([]byte(key), nil)
}

// Iterator returns a channel that yields key-value pairs.
func (s *LevelStorage) Iterator() (<-chan [2]string, error) {
	iter := s.db.NewIterator(&util.Range{}, nil)
	ch := make(chan [2]string)

	go func() {
		defer iter.Release()
		defer close(ch)

		for iter.Next() {
			key := string(iter.Key())
			value := string(iter.Value())
			ch <- [2]string{key, value}
		}
	}()

	return ch, nil
}

// Merge merges data from another storage instance.
func (s *LevelStorage) Merge(other Storage) error {
	iter, err := other.Iterator()
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)
	for kv := range iter {
		batch.Put([]byte(kv[0]), []byte(kv[1]))
	}
	return s.db.Write(batch, nil)
}

// Clear removes all key-value pairs from LevelDB.
func (s *LevelStorage) Clear() error {
	iter := s.db.NewIterator(&util.Range{}, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)
	for iter.Next() {
		batch.Delete(iter.Key())
	}
	return s.db.Write(batch, nil)
}

// Close closes the LevelDB instance.
func (s *LevelStorage) Close() error {
	return s.db.Close()
}
