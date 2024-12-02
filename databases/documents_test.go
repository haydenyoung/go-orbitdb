package databases_test

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"orbitdb/go-orbitdb/databases"
	"orbitdb/go-orbitdb/storage"
	"testing"
)

// setupKeyValueTest initializes a KeyValue instance for testing.
func setupKeyValueTest(t *testing.T) *databases.KeyValue {
	// Use the helper to create a KeyStore and Identity
	ks, identity := setupTestKeyStoreAndIdentity(t)

	// In-memory storage for the database
	entryStorage := storage.NewMemoryStorage()

	// Create a libp2p host
	ctx := context.Background()
	host, err := libp2p.New()
	require.NoError(t, err)

	// Create a pubsub instance
	ps, err := pubsub.NewGossipSub(ctx, host)
	require.NoError(t, err)

	// Create the KeyValue database instance
	kv, err := databases.NewKeyValue("test-address", "test-keyvalue", identity, entryStorage, ks, host, ps)
	require.NoError(t, err)

	return kv
}

// setupDocumentsTest initializes a Documents instance for testing.
func setupDocumentsTest(t *testing.T) *databases.Documents {
	kv := setupKeyValueTest(t)
	docs, err := databases.NewDocuments("_id", kv)
	require.NoError(t, err)
	fmt.Println("Debug (Setup): Initialized Documents instance")
	return docs
}

// TestDocuments_PutAndGet tests adding and retrieving a document.
func TestDocuments_PutAndGet(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Add a document
	doc := map[string]interface{}{
		"_id":   "doc1",
		"title": "Test Document",
		"body":  "This is a test.",
	}
	hash, err := docs.Put(doc)
	require.NoError(t, err, "Failed to put document")
	assert.NotEmpty(t, hash, "Expected a non-empty hash for the inserted document")

	// Retrieve the document
	retrieved, err := docs.Get("doc1")
	require.NoError(t, err, "Failed to get document")
	require.NotNil(t, retrieved, "Expected to retrieve a document, but got nil")

	// Debug output for verification
	fmt.Printf("Debug: Inserted document: %+v\n", doc)
	fmt.Printf("Debug: Retrieved document: %+v\n", retrieved)

	// Validate the document structure
	assert.Equal(t, doc, retrieved, "The retrieved document does not match the original")
}

// TestDocuments_Del tests deleting a document.
func TestDocuments_Del(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Add a document and delete it
	doc := map[string]interface{}{
		"_id":   "doc1",
		"title": "Test Document",
		"body":  "This is a test.",
	}
	_, err := docs.Put(doc)
	require.NoError(t, err)

	hash, err := docs.Del("doc1")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the document is deleted
	retrieved, err := docs.Get("doc1")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

// TestDocuments_Query tests querying documents with a filter function.
func TestDocuments_Query(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Add multiple documents
	_, err := docs.Put(map[string]interface{}{"_id": "doc1", "type": "test", "value": 10})
	require.NoError(t, err)
	_, err = docs.Put(map[string]interface{}{"_id": "doc2", "type": "test", "value": 20})
	require.NoError(t, err)
	_, err = docs.Put(map[string]interface{}{"_id": "doc3", "type": "other", "value": 30})
	require.NoError(t, err)

	// Query documents with type "test"
	results, err := docs.Query(func(doc map[string]interface{}) bool {
		fmt.Printf("Debug (Query): Filtering document: %+v\n", doc)
		return doc["type"] == "test"
	})
	require.NoError(t, err)

	// Convert expected values to float64 for compatibility
	expected := []map[string]interface{}{
		{"_id": "doc1", "type": "test", "value": float64(10)},
		{"_id": "doc2", "type": "test", "value": float64(20)},
	}
	assert.Len(t, results, 2)
	assert.ElementsMatch(t, expected, results)
}

// TestDocuments_All tests retrieving all documents in the database.
func TestDocuments_All(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Add multiple documents
	docs.Put(map[string]interface{}{"_id": "doc1", "type": "test", "value": 10})
	docs.Put(map[string]interface{}{"_id": "doc2", "type": "test", "value": 20})
	docs.Put(map[string]interface{}{"_id": "doc3", "type": "other", "value": 30})

	// Retrieve all documents
	all, err := docs.All()
	fmt.Printf("Debug (All): Retrieved documents: %+v\n", all)
	require.NoError(t, err)

	expected := map[string]map[string]interface{}{
		"doc1": {"_id": "doc1", "type": "test", "value": float64(10)},
		"doc2": {"_id": "doc2", "type": "test", "value": float64(20)},
		"doc3": {"_id": "doc3", "type": "other", "value": float64(30)},
	}
	assert.Equal(t, expected, all)
}

// TestDocuments_InvalidDocument tests behavior with invalid documents.
func TestDocuments_InvalidDocument(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Missing the `_id` field
	doc := map[string]interface{}{
		"title": "Invalid Document",
		"body":  "This is missing the '_id' field.",
	}
	_, err := docs.Put(doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document must contain field '_id'")

	// `_id` field is not a string
	doc = map[string]interface{}{
		"_id":   123, // Invalid type
		"title": "Invalid Document",
		"body":  "This '_id' field is not a string.",
	}
	_, err = docs.Put(doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document must contain field '_id' as a string")
}

func TestDocuments_Empty(t *testing.T) {
	docs := setupDocumentsTest(t)

	// Retrieve all documents from an empty database
	all, err := docs.All()
	require.NoError(t, err)
	assert.Empty(t, all)

	// Query an empty database
	results, err := docs.Query(func(doc map[string]interface{}) bool {
		return true
	})
	require.NoError(t, err)
	assert.Empty(t, results)
}
