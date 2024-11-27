package databases

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"orbitdb/go-orbitdb/storage"
)

// KeyValueIndexed represents a key-value database with an index for fast queries.
type KeyValueIndexed struct {
	BaseDB       *KeyValue       // Underlying KeyValue database
	indexStorage storage.Storage // Storage for the index
	processed    map[string]bool // Tracks processed log entries
	mu           sync.Mutex
}

// NewKeyValueIndexed creates a new KeyValueIndexed database instance.
func NewKeyValueIndexed(baseDB *KeyValue, indexStorage storage.Storage) (*KeyValueIndexed, error) {
	if baseDB == nil || indexStorage == nil {
		return nil, fmt.Errorf("base database and index storage are required")
	}

	return &KeyValueIndexed{
		BaseDB:       baseDB,
		indexStorage: indexStorage,
		processed:    make(map[string]bool),
	}, nil
}

// UpdateIndex updates the index by traversing the log and processing entries.
func (kvi *KeyValueIndexed) UpdateIndex() error {
	kvi.mu.Lock()
	defer kvi.mu.Unlock()

	fmt.Println("Debug: UpdateIndex invoked")
	entries, err := kvi.BaseDB.Log.Values()
	if err != nil {
		return fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	for _, entry := range entries {
		if kvi.processed[entry.Hash] {
			continue
		}

		fmt.Printf("Debug: Processing entry %s\n", entry.Hash)
		var payload map[string]interface{}

		// Attempt to unmarshal the payload
		err := json.Unmarshal([]byte(entry.Payload), &payload)
		if err != nil {
			fmt.Printf("Warning: Initial decoding failed for entry %s. Attempting fallback decoding. Error: %v\n", entry.Hash, err)

			// Fallback decoding for double-encoded payload
			var doubleEncodedPayload string
			if json.Unmarshal([]byte(entry.Payload), &doubleEncodedPayload) == nil {
				if json.Unmarshal([]byte(doubleEncodedPayload), &payload) == nil {
					fmt.Printf("Debug: Fallback decoding succeeded for entry %s\n", entry.Hash)
				} else {
					fmt.Printf("Warning: Fallback decoding failed for entry %s\n", entry.Hash)
					continue
				}
			} else {
				fmt.Printf("Warning: Double decoding not applicable for entry %s\n", entry.Hash)
				continue
			}
		}

		op := payload["op"]
		key, _ := payload["key"].(string)

		switch op {
		case "PUT":
			value := payload["value"]
			indexEntry := map[string]interface{}{
				"hash":  entry.Hash,
				"value": value,
			}

			serializedEntry, err := json.Marshal(indexEntry)
			if err != nil {
				return fmt.Errorf("failed to serialize index entry: %w", err)
			}

			if err := kvi.indexStorage.Put(key, serializedEntry); err != nil {
				fmt.Printf("Warning: Failed to index key %s: %v\n", key, err)
				continue
			}
		case "DEL":
			if err := kvi.indexStorage.Delete(key); err != nil {
				fmt.Printf("Warning: Failed to delete key %s from index: %v\n", key, err)
			}
		}

		kvi.processed[entry.Hash] = true
	}

	fmt.Println("Debug: UpdateIndex completed")
	return nil
}

// Get retrieves a value by its key using the index.
func (kvi *KeyValueIndexed) Get(key string) (interface{}, error) {
	data, err := kvi.indexStorage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve value for key %s: %w", key, err)
	}

	var indexEntry map[string]interface{}

	// Attempt to unmarshal the data directly
	err = json.Unmarshal([]byte(data), &indexEntry)
	if err != nil {
		fmt.Printf("Warning: Initial decoding failed for key %s. Attempting fallback decoding. Error: %v\n", key, err)

		// Attempt fallback decoding for double-encoded data
		var doubleEncodedPayload string
		if json.Unmarshal([]byte(data), &doubleEncodedPayload) == nil {
			if json.Unmarshal([]byte(doubleEncodedPayload), &indexEntry) == nil {
				fmt.Printf("Debug: Fallback decoding succeeded for key %s\n", key)
			} else {
				return nil, fmt.Errorf("fallback decoding failed for key %s: %w", key, err)
			}
		} else {
			return nil, fmt.Errorf("double decoding not applicable for key %s: %w", key, err)
		}
	}

	return indexEntry["value"], nil
}

// Iterator iterates over key-value pairs in the database.
func (kvi *KeyValueIndexed) Iterator(amount int) ([]map[string]interface{}, error) {
	iter, err := kvi.indexStorage.Iterator()
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}

	results := []map[string]interface{}{}

	for kv := range iter {
		var indexEntry map[string]interface{}
		if err := json.Unmarshal([]byte(kv[1]), &indexEntry); err != nil {
			fmt.Printf("Warning: Failed to decode index entry for key %s: %v\n", kv[0], err)
			continue
		}

		results = append(results, map[string]interface{}{
			"key":   kv[0],
			"value": indexEntry["value"],
			"hash":  indexEntry["hash"],
		})
	}

	// Sort results by key for deterministic order
	sort.Slice(results, func(i, j int) bool {
		return results[i]["key"].(string) < results[j]["key"].(string)
	})

	// Apply amount limit
	if amount > 0 && len(results) > amount {
		results = results[:amount]
	}

	return results, nil
}

// Close closes the index and underlying database.
func (kvi *KeyValueIndexed) Close() error {
	if err := kvi.BaseDB.Close(); err != nil {
		return fmt.Errorf("failed to close base database: %w", err)
	}

	if err := kvi.indexStorage.Close(); err != nil {
		return fmt.Errorf("failed to close index storage: %w", err)
	}

	return nil
}

// Drop clears all data from the index and underlying database.
func (kvi *KeyValueIndexed) Drop() error {
	if err := kvi.BaseDB.Drop(); err != nil {
		return fmt.Errorf("failed to drop base database: %w", err)
	}

	if err := kvi.indexStorage.Clear(); err != nil {
		return fmt.Errorf("failed to clear index storage: %w", err)
	}

	kvi.processed = make(map[string]bool)
	return nil
}
