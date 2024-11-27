package databases_test

import (
	"orbitdb/go-orbitdb/databases"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/oplog"
	"orbitdb/go-orbitdb/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestKeyStoreAndIdentity(t *testing.T) (*keystore.KeyStore, *identitytypes.Identity) {
	// Use LRUStorage as the storage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		panic(err) // Ensure setup failure is immediately apparent
	}
	ks := keystore.NewKeyStore(lruStorage)
	provider := providers.NewPublicKeyProvider(ks)
	identity, err := provider.CreateIdentity("test-ID")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}
	return ks, identity
}

// TestNewDatabase tests the initialization of a new Database instance.
func TestNewDatabase(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	entryStorage := storage.NewMemoryStorage()

	db, err := databases.NewDatabase("test-address", "test-db", identity, entryStorage, ks)
	require.NoError(t, err)
	require.NotNil(t, db)

	assert.Equal(t, "test-address", db.Address)
	assert.Equal(t, "test-db", db.Name)
	assert.Equal(t, identity, db.Identity)
	assert.NotNil(t, db.Log)
	assert.NotNil(t, db.Sync)
	assert.NotNil(t, db.Events)
}

// TestAddOperation tests adding an operation to the database.
func TestAddOperation(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	entryStorage := storage.NewMemoryStorage()

	db, err := databases.NewDatabase("test-address", "test-db", identity, entryStorage, ks)
	require.NoError(t, err)

	op := map[string]string{"key": "test", "value": "123"}
	hash, err := db.AddOperation(op)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the event is emitted
	select {
	case event := <-db.Events:
		entry, ok := event.(*oplog.EncodedEntry)
		require.True(t, ok)
		assert.Equal(t, hash, entry.Hash)
	default:
		t.Error("Expected an event to be emitted")
	}
}

// TestAddOperationSerializationError tests serialization errors in AddOperation.
func TestAddOperationSerializationError(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	entryStorage := storage.NewMemoryStorage()

	db, err := databases.NewDatabase("test-address", "test-db", identity, entryStorage, ks)
	require.NoError(t, err)

	// Pass an unmarshalable value to trigger a serialization error
	_, err = db.AddOperation(make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to serialize operation")
}

// TestApplyOperation tests applying an operation received via synchronization.
func TestApplyOperation(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	// Create the in-memory storage
	entryStorage := storage.NewMemoryStorage()

	// Log ID and database setup
	logID := "test-log"
	db, err := databases.NewDatabase(logID, "test-db", identity, entryStorage, ks)
	require.NoError(t, err)

	// Create a payload for the test entry
	payload := "test-payload"

	// Create the clock and the entry
	clock := oplog.NewClock(identity.ID, 1)
	entry := oplog.NewEntry(ks, identity, logID, payload, clock, nil, nil)

	// Encode the entry to bytes
	data := entry.Bytes

	// Apply the operation to the database
	db.ApplyOperation(data)

	// Verify the event is emitted
	select {
	case event := <-db.Events:
		emittedEntry, ok := event.(*oplog.EncodedEntry)
		require.True(t, ok, "Expected emitted event to be of type *oplog.EncodedEntry")
		assert.Equal(t, entry.Hash, emittedEntry.Hash, "Emitted entry hash does not match")
		assert.Equal(t, entry.Payload, emittedEntry.Payload, "Emitted entry payload does not match")
	case <-time.After(1 * time.Second): // Add a timeout
		t.Error("Expected an event to be emitted, but timed out")
	}
}

// TestClose tests closing the database.
func TestClose(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	entryStorage := storage.NewMemoryStorage()

	db, err := databases.NewDatabase("test-address", "test-db", identity, entryStorage, ks)
	require.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)

	// Verify channels are closed
	_, ok := <-db.Events
	assert.False(t, ok, "Events channel should be closed")
}
