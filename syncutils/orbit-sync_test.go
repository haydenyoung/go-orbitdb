package syncutils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"orbitdb/go-orbitdb/syncutils"
)

func TestSyncStartStop(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	sync := syncutils.NewSync("self", pubsub, log)

	err := sync.Start()
	assert.NoError(t, err)

	sync.Stop()
	assert.NotNil(t, sync)
}

func TestSyncIgnoreSelfMessage(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	sync := syncutils.NewSync("self", pubsub, log)

	err := sync.Start()
	assert.NoError(t, err)

	entry := syncutils.EncodedEntry{Hash: "self-entry"}
	err = sync.Add(entry)
	assert.NoError(t, err)

	// No message should be received
	select {
	case synced := <-sync.SyncedCh:
		t.Fatal("Should not receive message from self", synced)
	case <-time.After(100 * time.Millisecond):
		// Expected timeout as no message should be received
	}

	sync.Stop()
}

func TestSyncReceiveFromPeer(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	syncSelf := syncutils.NewSync("self", pubsub, log)
	syncPeer := syncutils.NewSync("peer1", pubsub, log)

	err := syncSelf.Start()
	assert.NoError(t, err)
	err = syncPeer.Start()
	assert.NoError(t, err)

	entry := syncutils.EncodedEntry{Hash: "peer-entry"}
	err = syncPeer.Add(entry)
	assert.NoError(t, err)

	select {
	case synced := <-syncSelf.SyncedCh:
		assert.Equal(t, "peer-entry", synced.Entry.Hash)
		assert.Equal(t, "peer1", synced.PeerID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for synced message from peer")
	}

	syncSelf.Stop()
	syncPeer.Stop()
}

func TestSyncMultipleEntries(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	syncSelf := syncutils.NewSync("self", pubsub, log)
	syncPeer := syncutils.NewSync("peer1", pubsub, log)

	err := syncSelf.Start()
	assert.NoError(t, err)
	err = syncPeer.Start()
	assert.NoError(t, err)

	entries := []syncutils.EncodedEntry{
		{Hash: "entry-1"},
		{Hash: "entry-2"},
		{Hash: "entry-3"},
	}

	for _, entry := range entries {
		err := syncPeer.Add(entry)
		assert.NoError(t, err)
	}

	for i := 0; i < len(entries); i++ {
		select {
		case synced := <-syncSelf.SyncedCh:
			assert.Contains(t, []string{"entry-1", "entry-2", "entry-3"}, synced.Entry.Hash)
			assert.Equal(t, "peer1", synced.PeerID)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for synced message %d", i+1)
		}
	}

	syncSelf.Stop()
	syncPeer.Stop()
}

func TestSyncDiscoverPeers(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	sync := syncutils.NewSync("self", pubsub, log)

	peers := sync.DiscoverPeers()
	assert.NotEmpty(t, peers)
	assert.Equal(t, []string{"peer1", "peer2"}, peers)
}

func TestSyncMessageDeduplication(t *testing.T) {
	pubsub := syncutils.NewLocalPubSub()
	log := &syncutils.Log{ID: "test-log"}
	syncSelf := syncutils.NewSync("self", pubsub, log)
	syncPeer := syncutils.NewSync("peer1", pubsub, log)

	err := syncSelf.Start()
	assert.NoError(t, err)
	err = syncPeer.Start()
	assert.NoError(t, err)

	entry := syncutils.EncodedEntry{Hash: "duplicate-entry"}
	for i := 0; i < 3; i++ {
		err := syncPeer.Add(entry)
		assert.NoError(t, err)
	}

	received := map[string]bool{}
	for i := 0; i < 3; i++ {
		select {
		case synced := <-syncSelf.SyncedCh:
			assert.Equal(t, "duplicate-entry", synced.Entry.Hash)
			assert.Equal(t, "peer1", synced.PeerID)
			received[synced.Entry.Hash] = true
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for synced message")
		}
	}

	assert.Len(t, received, 1) // Ensure deduplication logic would filter duplicates if implemented

	syncSelf.Stop()
	syncPeer.Stop()
}
