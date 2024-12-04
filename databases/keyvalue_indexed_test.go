package databases_test

import (
	"context"
	"fmt"
	"orbitdb/go-orbitdb/databases"
	"orbitdb/go-orbitdb/storage"
	"testing"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupHostAndPubsub initializes a libp2p host and pubsub instance for testing.
func setupHostAndPubsub(t *testing.T) (host.Host, *pubsub.PubSub) {
	// Create a libp2p host
	h, err := libp2p.New()
	require.NoError(t, err)

	// Create a pubsub instance
	ps, err := pubsub.NewGossipSub(context.Background(), h)
	require.NoError(t, err)

	return h, ps
}

// setupKeyValueIndexedTest initializes a KeyValueIndexed instance for testing.
func setupKeyValueIndexedTest(t *testing.T) *databases.KeyValueIndexed {
	// Create a mock in-memory KeyStore
	keyStore, identity := setupTestKeyStoreAndIdentity(t)
	require.NotNil(t, keyStore)

	// Create mock storages for entries and index
	entryStorage := storage.NewMemoryStorage()
	indexStorage := storage.NewMemoryStorage()
	require.NotNil(t, entryStorage)
	require.NotNil(t, indexStorage)

	// Set up libp2p host and pubsub
	host1, ps := setupHostAndPubsub(t)
	defer func(host1 host.Host) {
		err := host1.Close()
		if err != nil {

		}
	}(host1)

	// Create the base KeyValue database
	baseDB, err := databases.NewKeyValue("test-address", "test-db", identity, entryStorage, keyStore, host1, ps)
	require.NoError(t, err)

	// Create the KeyValueIndexed database using the BaseDB and indexStorage
	kvi, err := databases.NewKeyValueIndexed(baseDB, indexStorage)
	require.NoError(t, err)

	// Update the index after initialization
	err = kvi.UpdateIndex()
	require.NoError(t, err, "Failed to update index after initialization")

	return kvi
}

// TestKeyValueIndexed_PutAndGet tests adding and retrieving values
func TestKeyValueIndexed_PutAndGet(t *testing.T) {
	kvi := setupKeyValueIndexedTest(t)

	// Add a key-value pair
	key := "key1"
	value := map[string]interface{}{"type": "test", "value": float64(42)}
	_, err := kvi.BaseDB.Put(key, value)
	require.NoError(t, err, "Failed to put key-value pair")

	// Trigger index update explicitly
	err = kvi.UpdateIndex()
	require.NoError(t, err, "Failed to update index after put operation")

	// Retrieve the value
	retrieved, err := kvi.Get(key)
	require.NoError(t, err, "Failed to get value for key")
	assert.Equal(t, value, retrieved, "Retrieved value does not match the inserted value")
}

// TestKeyValueIndexed_Drop tests clearing all entries
func TestKeyValueIndexed_Drop(t *testing.T) {
	kvi := setupKeyValueIndexedTest(t)

	// Add multiple key-value pairs
	_, err := kvi.BaseDB.Put("key1", "value1")
	require.NoError(t, err)
	_, err = kvi.BaseDB.Put("key2", "value2")
	require.NoError(t, err)

	// Drop all entries
	err = kvi.Drop()
	require.NoError(t, err, "Failed to drop database")

	// Verify that the database is empty
	all, err := kvi.BaseDB.All()
	require.NoError(t, err, "Failed to retrieve all entries")
	assert.Empty(t, all, "Expected database to be empty after drop")
}

// TestKeyValueIndexed_UpdateIndex tests the index updating
func TestKeyValueIndexed_UpdateIndex(t *testing.T) {
	kvi := setupKeyValueIndexedTest(t)

	// Add multiple key-value pairs
	_, err := kvi.BaseDB.Put("key1", "value1")
	require.NoError(t, err)
	_, err = kvi.BaseDB.Put("key2", "value2")
	require.NoError(t, err)

	// Manually update the index
	err = kvi.UpdateIndex()
	require.NoError(t, err, "Failed to update index")

	// Verify that the values can be retrieved after index update
	retrieved, err := kvi.Get("key1")
	require.NoError(t, err, "Failed to get value for key after index update")
	assert.Equal(t, "value1", retrieved, "Retrieved value does not match the inserted value after index update")
}

func TestKeyValueIndexed_Iterator(t *testing.T) {
	kvi := setupKeyValueIndexedTest(t)

	// Add multiple key-value pairs
	_, err := kvi.BaseDB.Put("key1", map[string]interface{}{"type": "test", "value": 1})
	require.NoError(t, err, "Failed to put key1")
	_, err = kvi.BaseDB.Put("key2", map[string]interface{}{"type": "test", "value": 2})
	require.NoError(t, err, "Failed to put key2")
	_, err = kvi.BaseDB.Put("key3", map[string]interface{}{"type": "test", "value": 3})
	require.NoError(t, err, "Failed to put key3")

	// Update the index after adding values
	err = kvi.UpdateIndex()
	require.NoError(t, err, "Failed to update index")

	// Test retrieving all entries
	allEntries, err := kvi.Iterator(-1)
	require.NoError(t, err, "Failed to iterate over entries")
	assert.Len(t, allEntries, 3, "Expected 3 entries")
	fmt.Printf("Debug: Retrieved all entries: %+v\n", allEntries)

	// Validate entries are in the correct format
	assert.Equal(t, "key1", allEntries[0]["key"])
	assert.Equal(t, "key2", allEntries[1]["key"])
	assert.Equal(t, "key3", allEntries[2]["key"])

	// Test limiting the number of results
	limitedEntries, err := kvi.Iterator(2)
	require.NoError(t, err, "Failed to iterate with limit")
	assert.Len(t, limitedEntries, 2, "Expected 2 entries")
	fmt.Printf("Debug: Retrieved limited entries: %+v\n", limitedEntries)

	// Validate that limiting works correctly
	assert.Equal(t, "key1", limitedEntries[0]["key"])
	assert.Equal(t, "key2", limitedEntries[1]["key"])
}
