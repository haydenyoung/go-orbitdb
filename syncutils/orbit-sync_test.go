package syncutils_test

import (
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/oplog"
	"orbitdb/go-orbitdb/storage"
	"orbitdb/go-orbitdb/syncutils"
)

// setupTestKeyStoreAndIdentity initializes a keystore and identity for testing.
func setupTestKeyStoreAndIdentity(t *testing.T, identityID string) (*keystore.KeyStore, *identitytypes.Identity) {
	ks := keystore.NewKeyStore(storage.NewMemoryStorage())
	require.NotNil(t, ks, "Keystore should not be nil")

	provider := providers.NewPublicKeyProvider(ks)
	identity, err := provider.CreateIdentity(identityID)
	require.NoError(t, err, "Failed to create identity")
	require.NotNil(t, identity, "Identity should not be nil")

	// Ensure the key exists in the keystore
	if !ks.HasKey(identity.ID) {
		_, err := ks.CreateKey(identity.ID)
		require.NoError(t, err, "Failed to create key in keystore")
	}

	return ks, identity
}

// createMockLog initializes a mock Log instance for testing.
func createMockLog(t *testing.T, logID string, identityID string) *oplog.Log {
	ks, identity := setupTestKeyStoreAndIdentity(t, identityID)
	entryStorage := storage.NewMemoryStorage()

	log, err := oplog.NewLog(logID, identity, entryStorage, ks)
	require.NoError(t, err, "Failed to create new log")
	require.NotNil(t, log, "Expected log to be non-nil")

	assert.Equal(t, logID, log.ID, "Log ID does not match")
	assert.Equal(t, identity, log.Identity, "Log identity does not match")
	assert.Equal(t, identity.ID, log.Clock.ID, "Clock ID does not match")
	assert.Equal(t, 0, log.Clock.Time, "Clock time should be initialized to 0")

	return log
}

func TestSyncStartStop(t *testing.T) {
	ctx := context.Background()

	host, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host")
	defer host.Close()

	ps, err := pubsub.NewGossipSub(ctx, host)
	require.NoError(t, err, "Failed to create GossipSub instance")

	log := createMockLog(t, "test-log", "test-identity")
	assert.Equal(t, "test-log", log.ID)

	sync := syncutils.NewSync(host, ps, log)

	err = sync.Start()
	assert.NoError(t, err)

	sync.Stop()
	assert.NotNil(t, sync)
}

func TestSyncAddAndBroadcast(t *testing.T) {
	ctx := context.Background()

	host, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host")
	defer host.Close()

	ps, err := pubsub.NewGossipSub(ctx, host)
	require.NoError(t, err, "Failed to create GossipSub instance")

	log := createMockLog(t, "test-log", "test-identity")
	assert.Equal(t, "test-log", log.ID)

	sync := syncutils.NewSync(host, ps, log)

	err = sync.Start()
	assert.NoError(t, err)

	// Add an entry to the log and broadcast
	err = sync.Add("test-entry")
	assert.NoError(t, err)

	// No message should be processed because we ignore self-broadcasts
	select {
	case synced := <-sync.SyncedCh:
		t.Fatalf("Unexpected message received: %v", synced)
	case <-time.After(100 * time.Millisecond):
		// Expected timeout as self-broadcasts are ignored
	}

	sync.Stop()
}

func TestSyncReceiveFromPeer(t *testing.T) {
	ctx := context.Background()

	// Create two libp2p hosts for the peers
	hostSelf, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for self")
	defer func(hostSelf host.Host) {
		err := hostSelf.Close()
		if err != nil {
			t.Logf("Failed to close hostSelf: %v", err)
		}
	}(hostSelf)

	hostPeer, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for peer")
	defer func(hostPeer host.Host) {
		err := hostPeer.Close()
		if err != nil {
			t.Logf("Failed to close hostPeer: %v", err)
		}
	}(hostPeer)

	// Explicitly connect the two hosts
	hostSelf.Peerstore().AddAddr(hostPeer.ID(), hostPeer.Addrs()[0], peerstore.PermanentAddrTTL)
	hostPeer.Peerstore().AddAddr(hostSelf.ID(), hostSelf.Addrs()[0], peerstore.PermanentAddrTTL)

	err = hostSelf.Connect(ctx, peer.AddrInfo{ID: hostPeer.ID()})
	require.NoError(t, err, "Failed to connect hostSelf to hostPeer")
	err = hostPeer.Connect(ctx, peer.AddrInfo{ID: hostSelf.ID()})
	require.NoError(t, err, "Failed to connect hostPeer to hostSelf")

	// Create GossipSub instances for each host
	psSelf, err := pubsub.NewGossipSub(ctx, hostSelf)
	require.NoError(t, err, "Failed to create GossipSub for self")
	psPeer, err := pubsub.NewGossipSub(ctx, hostPeer)
	require.NoError(t, err, "Failed to create GossipSub for peer")

	// Create logs and Sync instances
	logSelf := createMockLog(t, "shared-log", "self-identity")
	logPeer := createMockLog(t, "shared-log", "peer-identity")

	syncSelf := syncutils.NewSync(hostSelf, psSelf, logSelf)
	syncPeer := syncutils.NewSync(hostPeer, psPeer, logPeer)

	// Start Sync instances
	err = syncSelf.Start()
	assert.NoError(t, err, "Failed to start syncSelf")
	err = syncPeer.Start()
	assert.NoError(t, err, "Failed to start syncPeer")

	// Wait for the peers to discover each other on the topic
	discoveryTimeout := time.After(2 * time.Second)
	for {
		select {
		case <-discoveryTimeout:
			t.Fatal("Timeout waiting for peer discovery")
		default:
			peersSelf := psSelf.ListPeers("orbit-sync/shared-log")
			peersPeer := psPeer.ListPeers("orbit-sync/shared-log")

			// Convert peers to string for comparison
			peerSelfStrings := make([]string, len(peersSelf))
			for i, p := range peersSelf {
				peerSelfStrings[i] = p.String()
			}
			peerPeerStrings := make([]string, len(peersPeer))
			for i, p := range peersPeer {
				peerPeerStrings[i] = p.String()
			}

			if len(peerSelfStrings) > 0 && len(peerPeerStrings) > 0 {
				assert.Contains(t, peerSelfStrings, hostPeer.ID().String())
				assert.Contains(t, peerPeerStrings, hostSelf.ID().String())
				goto PeerDiscoveryComplete
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

PeerDiscoveryComplete:

	// Peer sends an entry
	err = syncPeer.Add("peer-entry")
	assert.NoError(t, err, "Failed to add entry to syncPeer")

	// Verify the message is received by syncSelf
	select {
	case synced := <-syncSelf.SyncedCh:
		assert.Equal(t, "peer-entry", synced.Entry.Payload, "Received entry payload mismatch")
		assert.Equal(t, hostPeer.ID().String(), synced.PeerID, "Received PeerID mismatch")
	case <-time.After(1 * time.Second): // Allow more time for PubSub propagation
		t.Fatal("Timeout waiting for synced message from peer")
	}

	syncSelf.Stop()
	syncPeer.Stop()
}

func TestSyncPeerJoinLeave(t *testing.T) {
	ctx := context.Background()

	// Create two libp2p hosts for the peers
	hostSelf, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for self")
	defer func(hostSelf host.Host) {
		err := hostSelf.Close()
		if err != nil {
			t.Logf("Failed to close hostSelf: %v", err)
		}
	}(hostSelf)

	hostPeer, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for peer")
	defer func(hostPeer host.Host) {
		err := hostPeer.Close()
		if err != nil {
			t.Logf("Failed to close hostPeer: %v", err)
		}
	}(hostPeer)

	// Explicitly connect the two hosts
	hostSelf.Peerstore().AddAddr(hostPeer.ID(), hostPeer.Addrs()[0], peerstore.PermanentAddrTTL)
	hostPeer.Peerstore().AddAddr(hostSelf.ID(), hostSelf.Addrs()[0], peerstore.PermanentAddrTTL)

	err = hostSelf.Connect(ctx, peer.AddrInfo{ID: hostPeer.ID()})
	require.NoError(t, err, "Failed to connect hostSelf to hostPeer")
	err = hostPeer.Connect(ctx, peer.AddrInfo{ID: hostSelf.ID()})
	require.NoError(t, err, "Failed to connect hostPeer to hostSelf")

	// Create GossipSub instances for each host
	psSelf, err := pubsub.NewGossipSub(ctx, hostSelf)
	require.NoError(t, err, "Failed to create GossipSub for self")
	psPeer, err := pubsub.NewGossipSub(ctx, hostPeer)
	require.NoError(t, err, "Failed to create GossipSub for peer")

	// Create logs and Sync instances
	logSelf := createMockLog(t, "shared-log", "self-identity")
	logPeer := createMockLog(t, "shared-log", "peer-identity")

	syncSelf := syncutils.NewSync(hostSelf, psSelf, logSelf)
	syncPeer := syncutils.NewSync(hostPeer, psPeer, logPeer)

	// Start Sync instances
	err = syncSelf.Start()
	assert.NoError(t, err, "Failed to start syncSelf")
	err = syncPeer.Start()
	assert.NoError(t, err, "Failed to start syncPeer")

	// Wait for the peers to discover each other on the topic
	discoveryTimeout := time.After(2 * time.Second)
	for {
		select {
		case <-discoveryTimeout:
			t.Fatal("Timeout waiting for peer discovery")
		default:
			peersSelf := psSelf.ListPeers("orbit-sync/shared-log")
			peersPeer := psPeer.ListPeers("orbit-sync/shared-log")

			// Convert peers to string for comparison
			peerSelfStrings := make([]string, len(peersSelf))
			for i, p := range peersSelf {
				peerSelfStrings[i] = p.String()
			}
			peerPeerStrings := make([]string, len(peersPeer))
			for i, p := range peersPeer {
				peerPeerStrings[i] = p.String()
			}

			if len(peerSelfStrings) > 0 && len(peerPeerStrings) > 0 {
				assert.Contains(t, peerSelfStrings, hostPeer.ID().String())
				assert.Contains(t, peerPeerStrings, hostSelf.ID().String())
				goto PeerDiscoveryComplete
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

PeerDiscoveryComplete:

	// Peer sends a "join" event
	peerID1 := hostPeer.ID().String()
	syncSelf.PeerJoin(peerID1)

	// Verify the "join" event is received by syncSelf
	select {
	case synced := <-syncSelf.SyncedCh:
		assert.Contains(t, synced.Entry.Payload, "has joined the network", "Join payload mismatch")
		assert.Equal(t, peerID1, synced.PeerID, "PeerID mismatch in join event")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for PeerJoin message")
	}

	// Peer sends a "leave" event
	peerID2 := hostPeer.ID().String()
	syncSelf.PeerLeave(peerID2)

	// Verify the "leave" event is received by syncSelf
	select {
	case synced := <-syncSelf.SyncedCh:
		assert.Contains(t, synced.Entry.Payload, "has left the network", "Leave payload mismatch")
		assert.Equal(t, peerID2, synced.PeerID, "PeerID mismatch in leave event")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for PeerLeave message")
	}

	// Stop Sync instances
	syncSelf.Stop()
	syncPeer.Stop()
}

func TestSyncSendAndReceiveHead(t *testing.T) {
	ctx := context.Background()

	// Create two libp2p hosts for the peers
	hostSelf, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for self")
	defer func(hostSelf host.Host) {
		err := hostSelf.Close()
		if err != nil {
			t.Logf("Failed to close hostSelf: %v", err)
		}
	}(hostSelf)

	hostPeer, err := libp2p.New()
	require.NoError(t, err, "Failed to create libp2p host for peer")
	defer func(hostPeer host.Host) {
		err := hostPeer.Close()
		if err != nil {
			t.Logf("Failed to close hostPeer: %v", err)
		}
	}(hostPeer)

	// Explicitly connect the two hosts
	hostSelf.Peerstore().AddAddr(hostPeer.ID(), hostPeer.Addrs()[0], peerstore.PermanentAddrTTL)
	hostPeer.Peerstore().AddAddr(hostSelf.ID(), hostSelf.Addrs()[0], peerstore.PermanentAddrTTL)

	err = hostSelf.Connect(ctx, peer.AddrInfo{ID: hostPeer.ID()})
	require.NoError(t, err, "Failed to connect hostSelf to hostPeer")
	err = hostPeer.Connect(ctx, peer.AddrInfo{ID: hostSelf.ID()})
	require.NoError(t, err, "Failed to connect hostPeer to hostSelf")

	// Create GossipSub instances for each host
	psSelf, err := pubsub.NewGossipSub(ctx, hostSelf)
	require.NoError(t, err, "Failed to create GossipSub for self")
	psPeer, err := pubsub.NewGossipSub(ctx, hostPeer)
	require.NoError(t, err, "Failed to create GossipSub for peer")

	// Create logs and Sync instances
	logSelf := createMockLog(t, "shared-log", "self-identity")
	logPeer := createMockLog(t, "shared-log", "peer-identity")

	syncSelf := syncutils.NewSync(hostSelf, psSelf, logSelf)
	syncPeer := syncutils.NewSync(hostPeer, psPeer, logPeer)

	// Start Sync instances
	err = syncSelf.Start()
	assert.NoError(t, err, "Failed to start syncSelf")
	err = syncPeer.Start()
	assert.NoError(t, err, "Failed to start syncPeer")

	// Wait for the peers to discover each other on the topic
	discoveryTimeout := time.After(2 * time.Second)
	for {
		select {
		case <-discoveryTimeout:
			t.Fatal("Timeout waiting for peer discovery")
		default:
			peersSelf := psSelf.ListPeers("orbit-sync/shared-log")
			peersPeer := psPeer.ListPeers("orbit-sync/shared-log")

			// Convert peers to string for comparison
			peerSelfStrings := make([]string, len(peersSelf))
			for i, p := range peersSelf {
				peerSelfStrings[i] = p.String()
			}
			peerPeerStrings := make([]string, len(peersPeer))
			for i, p := range peersPeer {
				peerPeerStrings[i] = p.String()
			}

			if len(peerSelfStrings) > 0 && len(peerPeerStrings) > 0 {
				assert.Contains(t, peerSelfStrings, hostPeer.ID().String())
				assert.Contains(t, peerPeerStrings, hostSelf.ID().String())
				goto PeerDiscoveryComplete
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

PeerDiscoveryComplete:

	// Self sends a head to the peer
	err = syncSelf.Add("test-head-entry")
	assert.NoError(t, err, "Failed to add entry to syncSelf")

	// Verify the head is received by peer
	select {
	case synced := <-syncPeer.SyncedCh:
		assert.Equal(t, "test-head-entry", synced.Entry.Payload, "Received head entry payload mismatch")
		assert.Equal(t, hostSelf.ID().String(), synced.PeerID, "Received PeerID mismatch for head entry")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for head message from self")
	}

	// Peer sends a head to the self
	err = syncPeer.Add("peer-head-entry")
	assert.NoError(t, err, "Failed to add entry to syncPeer")

	// Verify the head is received by self
	select {
	case synced := <-syncSelf.SyncedCh:
		assert.Equal(t, "peer-head-entry", synced.Entry.Payload, "Received head entry payload mismatch")
		assert.Equal(t, hostPeer.ID().String(), synced.PeerID, "Received PeerID mismatch for head entry")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for head message from peer")
	}

	// Stop Sync instances
	syncSelf.Stop()
	syncPeer.Stop()
}
