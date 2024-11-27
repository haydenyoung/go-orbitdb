package databases

import (
	"encoding/json"
	"errors"
	"fmt"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/storage"
)

// KeyValue extends the base Database with key-value functionality.
type KeyValue struct {
	*Database
}

// NewKeyValue creates a new KeyValue database instance.
func NewKeyValue(address, name string, identity *identitytypes.Identity, entryStorage storage.Storage, keyStore *keystore.KeyStore) (*KeyValue, error) {
	baseDB, err := NewDatabase(address, name, identity, entryStorage, keyStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create base database: %w", err)
	}
	return &KeyValue{Database: baseDB}, nil
}

// Put adds or updates a key-value pair.
func (kv *KeyValue) Put(key string, value interface{}) (string, error) {
	if key == "" {
		return "", errors.New("key cannot be empty")
	}

	op := map[string]interface{}{
		"op":    "PUT",
		"key":   key,
		"value": value,
	}

	payload, err := json.Marshal(op)
	if err != nil {
		return "", fmt.Errorf("failed to serialize operation: %w", err)
	}

	return kv.AddOperation(string(payload))
}

// Get retrieves the value for a given key.
func (kv *KeyValue) Get(key string) (interface{}, error) {
	entries, err := kv.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	// Traverse log entries in reverse order (most recent first)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		// Decode the outer JSON-encoded payload string
		var rawPayload string
		err := json.Unmarshal([]byte(entry.Payload), &rawPayload)
		if err != nil {
			fmt.Printf("Warning: Failed to decode outer payload for entry %s: %v\n", entry.Hash, err)
			continue
		}

		// Decode the inner JSON string into a map
		var payload map[string]interface{}
		err = json.Unmarshal([]byte(rawPayload), &payload)
		if err != nil {
			fmt.Printf("Warning: Failed to decode inner payload for entry %s: %v\n", entry.Hash, err)
			continue
		}

		op, ok := payload["op"].(string)
		entryKey, _ := payload["key"].(string)
		if !ok || entryKey != key {
			continue
		}

		// Handle the operation
		if op == "PUT" {
			return payload["value"], nil
		} else if op == "DEL" {
			return nil, nil
		}
	}

	// If the key is not found, return nil
	return nil, nil
}

// Del removes a key-value pair.
func (kv *KeyValue) Del(key string) (string, error) {
	if key == "" {
		return "", errors.New("key cannot be empty")
	}

	op := map[string]interface{}{
		"op":  "DEL",
		"key": key,
	}

	payload, err := json.Marshal(op)
	if err != nil {
		return "", fmt.Errorf("failed to serialize operation: %w", err)
	}

	return kv.AddOperation(string(payload))
}

// All retrieves all key-value pairs in the database.
func (kv *KeyValue) All() (map[string]interface{}, error) {
	entries, err := kv.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	result := make(map[string]interface{})
	processedKeys := make(map[string]bool)

	// Traverse log entries in reverse order (most recent first)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		// Decode the outer JSON-encoded payload string
		var rawPayload string
		err := json.Unmarshal([]byte(entry.Payload), &rawPayload)
		if err != nil {
			fmt.Printf("Warning: Failed to decode outer payload for entry %s: %v\n", entry.Hash, err)
			continue
		}

		// Decode the inner JSON string into a map
		var payload map[string]interface{}
		err = json.Unmarshal([]byte(rawPayload), &payload)
		if err != nil {
			fmt.Printf("Warning: Failed to decode inner payload for entry %s: %v\n", entry.Hash, err)
			continue
		}

		op, ok := payload["op"].(string)
		key, _ := payload["key"].(string)
		value, _ := payload["value"].(interface{})

		// If the key has already been processed, skip it
		if processedKeys[key] {
			continue
		}

		if ok && op == "PUT" {
			result[key] = value
		} else if op == "DEL" {
			delete(result, key)
		}

		processedKeys[key] = true
	}

	return result, nil
}
