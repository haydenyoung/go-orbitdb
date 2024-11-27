package syncutils

import (
	"orbitdb/go-orbitdb/oplog"
)

type Sync struct{}

func NewSync(ipfs interface{}, log *oplog.Log, events interface{}, onSynced func([]byte), start bool) (*Sync, error) {
	return &Sync{}, nil
}

func (s *Sync) Add(entry *oplog.EncodedEntry) error {
	// Stub: Log or track the entry
	return nil
}

func (s *Sync) Stop() error {
	// Stub: Stop syncing
	return nil
}
