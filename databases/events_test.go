package databases_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"orbitdb/go-orbitdb/databases"
	"orbitdb/go-orbitdb/identities/providers"
	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/storage"
)

// setupDatabaseTest initializes a mock database instance for testing.
func setupDatabaseTest(t *testing.T) *databases.Database {
	// Create a unique in-memory KeyStore and Storage for each test
	keyStore := keystore.NewKeyStore(storage.NewMemoryStorage())
	require.NotNil(t, keyStore)

	provider := providers.NewPublicKeyProvider(keyStore)
	identity, err := provider.CreateIdentity(fmt.Sprintf("test-ID-%d", time.Now().UnixNano()))
	require.NoError(t, err)

	entryStorage := storage.NewMemoryStorage()
	require.NotNil(t, entryStorage)

	// Create a libp2p host
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)

	// Create a pubsub instance
	ps, err := pubsub.NewGossipSub(ctx, host)
	require.NoError(t, err)

	// Create the database instance
	db, err := databases.NewDatabase("test-address", "test-db", identity, entryStorage, keyStore, host, ps)
	require.NoError(t, err)

	return db
}

func TestEvents_AddAndGet(t *testing.T) {
	db := setupDatabaseTest(t)
	events := databases.NewEvents(db)

	// Add an event
	event := map[string]interface{}{"type": "test", "value": map[string]interface{}{"count": float64(42), "nested": true}}
	hash, err := events.Add(event)
	require.NoError(t, err, "Failed to add event")
	assert.NotEmpty(t, hash, "Expected a non-empty hash for the added event")

	fmt.Printf("Debug (Test Add): Hash returned by Add: %s\n", hash)

	// Retrieve the event
	value, err := events.Get(hash)
	require.NoError(t, err, "Failed to get event by hash")
	assert.Equal(t, event, value, "The retrieved event does not match the added event")
}

func TestEvents_All(t *testing.T) {
	db := setupDatabaseTest(t)
	events := databases.NewEvents(db)

	// Add multiple events
	_, err1 := events.Add("Event 1")
	require.NoError(t, err1)
	_, err2 := events.Add("Event 2")
	require.NoError(t, err2)
	_, err3 := events.Add("Event 3")
	require.NoError(t, err3)

	// Retrieve all events
	all, err := events.All()
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, "Event 3", all[0]["value"])
	assert.Equal(t, "Event 1", all[2]["value"])

	// Test that the events are in reverse chronological order
	expectedOrder := []string{"Event 3", "Event 2", "Event 1"}
	for i, event := range all {
		assert.Equal(t, expectedOrder[i], event["value"])
	}
}

func TestEvents_Iterator(t *testing.T) {
	db := setupDatabaseTest(t)
	events := databases.NewEvents(db)

	// Add multiple events
	hash1, err := events.Add("Event 1")
	require.NoError(t, err)
	hash2, err := events.Add("Event 2")
	require.NoError(t, err)
	hash3, err := events.Add("Event 3")
	require.NoError(t, err)

	// Test iterator with "gte" filter
	it, err := events.Iterator("", fmt.Sprintf("%d:%s", 2, hash2), "", "", 2)
	require.NoError(t, err, "Failed to get iterator with 'gte' filter")
	assert.Len(t, it, 2, "Expected 2 entries with 'gte' filter")
	assert.Equal(t, hash2, it[0]["hash"], "First entry should match the second added hash")
	assert.Equal(t, hash3, it[1]["hash"], "Second entry should match the last added hash")

	// Test iterator with "lte" filter
	it, err = events.Iterator("", "", "", fmt.Sprintf("%d:%s", 2, hash2), 2)
	require.NoError(t, err, "Failed to get iterator with 'lte' filter")
	assert.Len(t, it, 2, "Expected 2 entries with 'lte' filter")
	assert.Equal(t, hash1, it[0]["hash"], "First entry should match the first added hash")
	assert.Equal(t, hash2, it[1]["hash"], "Second entry should match the second added hash")
}
