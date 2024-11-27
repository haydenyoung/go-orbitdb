package databases_test

import (
	"encoding/json"
	"testing"

	"orbitdb/go-orbitdb/databases"
	"orbitdb/go-orbitdb/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up a KeyValue instance for testing
func setupKeyValueTest(t *testing.T) *databases.KeyValue {
	// Use the helper to create a KeyStore and Identity
	ks, identity := setupTestKeyStoreAndIdentity(t)

	// In-memory storage for the database
	entryStorage := storage.NewMemoryStorage()

	// Create the KeyValue database instance
	kv, err := databases.NewKeyValue("test-address", "test-keyvalue", identity, entryStorage, ks)
	require.NoError(t, err)

	return kv
}

// TestPut tests the Put method of KeyValue
func TestPut(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Add a key-value pair
	hash, err := kv.Put("key1", "value1")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Retrieve the value directly from the log to verify
	entries, err := kv.Log.Values()
	require.NoError(t, err)

	// Decode the double-encoded payload
	var outerPayload string
	err = json.Unmarshal([]byte(entries[0].Payload), &outerPayload)
	require.NoError(t, err)

	var payload map[string]interface{}
	err = json.Unmarshal([]byte(outerPayload), &payload)
	require.NoError(t, err)

	assert.Equal(t, "PUT", payload["op"])
	assert.Equal(t, "key1", payload["key"])
	assert.Equal(t, "value1", payload["value"])
}

// TestGet tests the Get method of KeyValue
func TestGet(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Add a key-value pair
	_, err := kv.Put("key1", "value1")
	require.NoError(t, err)

	// Retrieve the value
	value, err := kv.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", value)

	// Try retrieving a non-existent key
	value, err = kv.Get("key2")
	require.NoError(t, err)
	assert.Nil(t, value)
}

// TestDel tests the Del method of KeyValue
func TestDel(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Add a key-value pair and then delete it
	_, err := kv.Put("key1", "value1")
	require.NoError(t, err)

	hash, err := kv.Del("key1")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the key is deleted
	value, err := kv.Get("key1")
	require.NoError(t, err)
	assert.Nil(t, value)
}

// TestAll tests the All method of KeyValue
func TestAll(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Add multiple key-value pairs
	_, err := kv.Put("key1", "value1")
	require.NoError(t, err)
	_, err = kv.Put("key2", "value2")
	require.NoError(t, err)
	_, err = kv.Put("key3", "value3")
	require.NoError(t, err)

	// Retrieve all key-value pairs
	all, err := kv.All()
	require.NoError(t, err)

	expected := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	assert.Equal(t, expected, all)

	// Delete one key and check all again
	_, err = kv.Del("key2")
	require.NoError(t, err)

	all, err = kv.All()
	require.NoError(t, err)

	expected = map[string]interface{}{
		"key1": "value1",
		"key3": "value3",
	}
	assert.Equal(t, expected, all)
}

// TestOverwrite tests overwriting an existing key
func TestOverwrite(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Add a key-value pair
	_, err := kv.Put("key1", "value1")
	require.NoError(t, err)

	// Overwrite the key with a new value
	_, err = kv.Put("key1", "value2")
	require.NoError(t, err)

	// Verify the key's value is updated
	value, err := kv.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, "value2", value)
}

// TestInvalidKey tests behavior with invalid keys
func TestInvalidKey(t *testing.T) {
	kv := setupKeyValueTest(t)

	// Attempt to add a key-value pair with an empty key
	_, err := kv.Put("", "value1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	// Attempt to delete an empty key
	_, err = kv.Del("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
}
