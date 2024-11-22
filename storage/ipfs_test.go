package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// createTestDAGService creates a test DAGService.
func createTestDAGService(ds datastore.Batching) format.DAGService {
	bs := blockstore.NewBlockstore(ds)
	bsvc := blockservice.New(bs, nil) // Use a nil exchange interface for local blockservice
	return merkledag.NewDAGService(bsvc)
}

// encodeCBORBytes encodes a byte slice into CBOR.
func encodeCBORBytes(t *testing.T, data []byte) []byte {
	// Create a Bytes node builder.
	nb := basicnode.Prototype.Bytes.NewBuilder()

	// Assign the byte slice to the builder.
	err := nb.AssignBytes(data)
	require.NoError(t, err, "Failed to assign bytes to node builder")

	// Build the node.
	node := nb.Build()

	// Prepare a buffer to write the CBOR-encoded data.
	var buf bytes.Buffer

	// Encode the node into CBOR and write to the buffer.
	err = dagcbor.Encode(node, &buf)
	require.NoError(t, err, "Failed to encode node to CBOR")

	return buf.Bytes()
}

// encodeCBORMap encodes a map[string]string into CBOR.
func encodeCBORMap(t *testing.T, data map[string]string) []byte {
	// Create a Map node builder.
	nb := basicnode.Prototype.Map.NewBuilder()
	mBuilder, err := nb.BeginMap(int64(len(data)))
	require.NoError(t, err, "Failed to begin map builder")

	// Iterate over the map and assign key-value pairs.
	for k, v := range data {
		// Assemble an entry for the key.
		entryAssembler, err := mBuilder.AssembleEntry(k)
		require.NoError(t, err, "Failed to assemble entry for key: "+k)

		// Assign the value to the entry.
		err = entryAssembler.AssignString(v)
		require.NoError(t, err, "Failed to assign string value for key: "+k)
	}

	// Finish building the map.
	err = mBuilder.Finish()
	require.NoError(t, err, "Failed to finish map builder")

	// Build the node.
	node := nb.Build()

	// Prepare a buffer to write the CBOR-encoded data.
	var buf bytes.Buffer

	// Encode the node into CBOR and write to the buffer.
	err = dagcbor.Encode(node, &buf)
	require.NoError(t, err, "Failed to encode map node to CBOR")

	return buf.Bytes()
}

// encodeCBORComplexMap encodes a map[string]interface{} into CBOR.
func encodeCBORComplexMap(t *testing.T, data map[string]interface{}) []byte {
	// Create a Map node builder.
	nb := basicnode.Prototype.Map.NewBuilder()
	mBuilder, err := nb.BeginMap(int64(len(data)))
	require.NoError(t, err, "Failed to begin map builder")

	// Iterate over the map and assign key-value pairs.
	for k, v := range data {
		// Assemble an entry for the key.
		entryAssembler, err := mBuilder.AssembleEntry(k)
		require.NoError(t, err, "Failed to assemble entry for key: "+k)

		// Assign the value based on its type.
		switch val := v.(type) {
		case string:
			err = entryAssembler.AssignString(val)
			require.NoError(t, err, "Failed to assign string value for key: "+k)
		case []byte:
			err = entryAssembler.AssignBytes(val)
			require.NoError(t, err, "Failed to assign bytes value for key: "+k)
		// Add more cases as needed for different data types.
		default:
			t.Fatalf("Unsupported data type for key %s: %T", k, v)
		}
	}

	// Finish building the map.
	err = mBuilder.Finish()
	require.NoError(t, err, "Failed to finish map builder")

	// Build the node.
	node := nb.Build()

	// Prepare a buffer to write the CBOR-encoded data.
	var buf bytes.Buffer

	// Encode the node into CBOR and write to the buffer.
	err = dagcbor.Encode(node, &buf)
	require.NoError(t, err, "Failed to encode complex map node to CBOR")

	return buf.Bytes()
}

func TestIPFSBlockStorage_PutAndGet(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare data as a bytes node.
	originalData := []byte("test data")
	encodedData := encodeCBORBytes(t, originalData)

	// Generate a CID using DagCBOR codec.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Test Put.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Test Get.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get data from storage")
	require.Equal(t, encodedData, retrievedData, "Retrieved data does not match original")
}

func TestIPFSBlockStorage_Delete(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare data as a bytes node.
	originalData := []byte("test data")
	encodedData := encodeCBORBytes(t, originalData)

	// Generate a CID using DagCBOR codec.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Test Put.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Test Delete.
	err = storage.Delete(c.String())
	require.NoError(t, err, "Failed to delete data from storage")

	// Ensure the block is deleted.
	_, err = storage.Get(c.String())
	require.Error(t, err, "Expected error when retrieving deleted block, but got none")
}

func TestIPFSBlockStorage_Timeout(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance with a very short timeout
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, 1*time.Millisecond)
	require.NoError(t, err)

	// Generate a CID
	data := []byte("test data")
	c, err := cid.V1Builder{Codec: cid.Raw, MhType: mh.SHA2_256}.Sum(data)
	require.NoError(t, err)

	// Simulate a timeout scenario during Put
	err = storage.Put(c.String(), data)
	require.Error(t, err, "expected timeout error during Put")
}

func TestIPFSBlockStorage_UnsupportedFeatures(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err)

	// Test Iterator
	_, err = storage.Iterator()
	require.Error(t, err, "expected error for unsupported iterator")

	// Test Merge
	err = storage.Merge(nil)
	require.Error(t, err, "expected error for unsupported merge")

	// Test Clear
	err = storage.Clear()
	require.Error(t, err, "expected error for unsupported clear")
}

func TestIPFSBlockStorage_PutAndGet_ComplexMap(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare data as a complex map node.
	originalData := map[string]interface{}{
		"name":   "Alice",
		"age":    "30",
		"binary": []byte{0x01, 0x02, 0x03},
		// Add more key-value pairs as needed.
	}
	encodedData := encodeCBORComplexMap(t, originalData)

	// Generate a CID using DagCBOR codec.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Test Put.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Test Get.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get data from storage")
	require.Equal(t, encodedData, retrievedData, "Retrieved data does not match original")
}

func TestIPFSBlockStorage_Pinning(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance with pinning enabled.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance with pinning")

	// Prepare data.
	data := []byte("pinning test data")
	encodedData := encodeCBORBytes(t, data)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Put data into storage.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Verify that the block is pinned.
	_, isPinned, err := storage.pinner.IsPinned(ctx, c)
	require.NoError(t, err, "Failed to check pin state")
	require.True(t, isPinned, "Block should be pinned")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_NoPinning(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance with pinning disabled.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, false, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance without pinning")

	// Prepare data.
	data := []byte("no pinning test data")
	encodedData := encodeCBORBytes(t, data)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Put data into storage.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Verify that the block is not pinned.
	_, isPinned, err := storage.pinner.IsPinned(ctx, c)
	require.NoError(t, err, "Failed to check pin state")
	require.False(t, isPinned, "Block should not be pinned")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_InvalidCID(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	invalidCID := "invalidCID12345"

	// Test Put with invalid CID.
	err = storage.Put(invalidCID, []byte("data"))
	require.Error(t, err, "Expected error when putting data with invalid CID")

	// Test Get with invalid CID.
	_, err = storage.Get(invalidCID)
	require.Error(t, err, "Expected error when getting data with invalid CID")

	// Test Delete with invalid CID.
	err = storage.Delete(invalidCID)
	require.Error(t, err, "Expected error when deleting data with invalid CID")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_EmptyData(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare empty data.
	emptyData := []byte{}
	encodedData := encodeCBORBytes(t, emptyData)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Test Put.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put empty data into storage")

	// Test Get.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get empty data from storage")
	require.Equal(t, encodedData, retrievedData, "Retrieved empty data does not match original")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_LargeData(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare large data (e.g., 10MB).
	largeData := bytes.Repeat([]byte("A"), 10*1024*1024) // 10MB of 'A's
	encodedData := encodeCBORBytes(t, largeData)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Test Put.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put large data into storage")

	// Test Get.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get large data from storage")
	require.Equal(t, encodedData, retrievedData, "Retrieved large data does not match original")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_ConcurrentPutGet(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Number of concurrent operations.
	const numGoroutines = 50

	// Channel to signal completion.
	done := make(chan bool)

	// Start concurrent Put operations.
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			data := []byte(fmt.Sprintf("concurrent data %d", i))
			encodedData := encodeCBORBytes(t, data)
			c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
			require.NoError(t, err, "Failed to generate CID in goroutine")

			err = storage.Put(c.String(), encodedData)
			require.NoError(t, err, "Failed to put data in goroutine")

			// Retrieve the data.
			retrievedData, err := storage.Get(c.String())
			require.NoError(t, err, "Failed to get data in goroutine")
			require.Equal(t, encodedData, retrievedData, "Retrieved data does not match original in goroutine")

			done <- true
		}(i)
	}

	// Wait for all goroutines to finish.
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_Close(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage the first time")

	// Close the storage again to ensure it handles multiple close calls gracefully.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage the second time")
}

func TestIPFSBlockStorage_PutDuplicateCID(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare data.
	data := []byte("duplicate CID test data")
	encodedData := encodeCBORBytes(t, data)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Put data into storage for the first time.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage the first time")

	// Put data into storage again with the same CID.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage the second time with duplicate CID")

	// Retrieve the data.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get data from storage after duplicate Put")
	require.Equal(t, encodedData, retrievedData, "Retrieved data does not match original after duplicate Put")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_GetAfterDelete(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare data.
	data := []byte("get after delete test data")
	encodedData := encodeCBORBytes(t, data)

	// Generate CID.
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(encodedData)
	require.NoError(t, err, "Failed to generate CID")

	// Put data into storage.
	err = storage.Put(c.String(), encodedData)
	require.NoError(t, err, "Failed to put data into storage")

	// Delete the data.
	err = storage.Delete(c.String())
	require.NoError(t, err, "Failed to delete data from storage")

	// Attempt to get the deleted data.
	_, err = storage.Get(c.String())
	require.Error(t, err, "Expected error when getting deleted block")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_Initialization(t *testing.T) {
	ctx := context.Background()

	// Attempt to create IPFSBlockStorage without a datastore.
	_, err := NewIPFSBlockStorage(ctx, nil, nil, true, DefaultTimeout)
	require.Error(t, err, "Expected error when initializing storage without a datastore")
}

func TestIPFSBlockStorage_DifferentCodecs(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Prepare raw data.
	rawData := []byte("raw codec test data")

	// Generate CID using Raw codec.
	c, err := cid.V1Builder{Codec: cid.Raw, MhType: mh.SHA2_256}.Sum(rawData)
	require.NoError(t, err, "Failed to generate CID with Raw codec")

	// Put data into storage.
	err = storage.Put(c.String(), rawData)
	require.NoError(t, err, "Failed to put raw codec data into storage")

	// Get the data back.
	retrievedData, err := storage.Get(c.String())
	require.NoError(t, err, "Failed to get raw codec data from storage")
	require.Equal(t, rawData, retrievedData, "Retrieved raw codec data does not match original")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}

func TestIPFSBlockStorage_GetNonExistentBlock(t *testing.T) {
	ctx := context.Background()

	// Create a mock datastore and DAGService.
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	dagService := createTestDAGService(ds)

	// Create an IPFSBlockStorage instance.
	storage, err := NewIPFSBlockStorage(ctx, ds, dagService, true, DefaultTimeout)
	require.NoError(t, err, "Failed to create IPFSBlockStorage instance")

	// Generate a random CID.
	data := []byte("non-existent get test data")
	c, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(data)
	require.NoError(t, err, "Failed to generate CID")

	// Attempt to get the non-existent block.
	_, err = storage.Get(c.String())
	require.Error(t, err, "Expected error when getting non-existent block")

	// Close the storage.
	err = storage.Close()
	require.NoError(t, err, "Failed to close storage")
}
