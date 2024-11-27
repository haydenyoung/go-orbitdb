package databases

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Documents represents a database for storing structured documents.
type Documents struct {
	*KeyValue        // Embeds KeyValue for core functionality
	indexBy   string // Field to index documents by (default: "_id")
}

type DocumentPayload struct {
	Op    string                 `json:"op"`
	Key   string                 `json:"key"`
	Value map[string]interface{} `json:"value"`
}

// NewDocuments creates a new instance of the Documents database.
func NewDocuments(indexBy string, kv *KeyValue) (*Documents, error) {
	if indexBy == "" {
		indexBy = "_id" // Default index field
	}

	if kv == nil {
		return nil, errors.New("KeyValue instance is required")
	}

	return &Documents{
		KeyValue: kv,
		indexBy:  indexBy,
	}, nil
}

// Put adds or updates a document in the database.
func (d *Documents) Put(doc map[string]interface{}) (string, error) {
	key, ok := doc[d.indexBy].(string)
	if !ok || key == "" {
		return "", fmt.Errorf("document must contain field '%s' as a string", d.indexBy)
	}

	payload := DocumentPayload{
		Op:    "PUT",
		Key:   key,
		Value: doc,
	}

	// Serialize the DocumentPayload directly
	serializedPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to serialize payload: %w", err)
	}

	fmt.Printf("Debug (Put): Serialized Payload: %s\n", string(serializedPayload))

	// Store the serialized payload without additional encoding
	return d.KeyValue.AddOperation(string(serializedPayload))
}

// Get retrieves a document by its index field value (key).
func (d *Documents) Get(id string) (map[string]interface{}, error) {
	// Retrieve the stored document
	value, err := d.KeyValue.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	fmt.Printf("Debug (Get): Retrieved value for key '%s': %+v\n", id, value)

	if value == nil {
		return nil, nil // Document not found
	}

	// Check if the value is already a map[string]interface{}
	if payloadMap, ok := value.(map[string]interface{}); ok {
		// Assume payloadMap is already the document
		return payloadMap, nil
	}

	// If value is a JSON string, attempt to deserialize it
	var payload DocumentPayload
	err = json.Unmarshal([]byte(value.(string)), &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize document payload: %w", err)
	}

	return payload.Value, nil // Return the actual document
}

// Del deletes a document by its index field value (key).
func (d *Documents) Del(id string) (string, error) {
	// Use KeyValue.Del to delete the document
	return d.KeyValue.Del(id)
}

// Query retrieves documents matching a user-defined filter function.
func (d *Documents) Query(filterFn func(doc map[string]interface{}) bool) ([]map[string]interface{}, error) {
	entries, err := d.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		fmt.Printf("Debug (Query): Processing entry with hash %s and payload: %s\n", entry.Hash, entry.Payload)

		var rawPayload DocumentPayload

		// Attempt to unmarshal directly into DocumentPayload
		err := json.Unmarshal([]byte(entry.Payload), &rawPayload)
		if err != nil {
			fmt.Printf("Warning (Query): Failed to decode payload for entry %s: %v. Attempting fallback decoding.\n", entry.Hash, err)

			// Fallback: Attempt to decode double-encoded payload
			var doubleEncodedPayload string
			if json.Unmarshal([]byte(entry.Payload), &doubleEncodedPayload) == nil {
				if json.Unmarshal([]byte(doubleEncodedPayload), &rawPayload) == nil {
					fmt.Printf("Debug (Query): Fallback decoding succeeded for entry %s\n", entry.Hash)
				} else {
					fmt.Printf("Warning (Query): Fallback decoding failed for entry %s\n", entry.Hash)
					continue
				}
			} else {
				fmt.Printf("Warning (Query): Double decoding not applicable for entry %s\n", entry.Hash)
				continue
			}
		}

		// Skip deleted documents
		if rawPayload.Op == "DEL" {
			continue
		}

		// Apply the filter function
		if filterFn(rawPayload.Value) {
			results = append(results, rawPayload.Value)
		}
	}

	fmt.Printf("Debug (Query): Query results: %+v\n", results)
	return results, nil
}

// All retrieves all documents in the database.
func (d *Documents) All() (map[string]map[string]interface{}, error) {
	entries, err := d.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	results := make(map[string]map[string]interface{})
	for _, entry := range entries {
		var rawPayload DocumentPayload

		// Attempt to unmarshal directly into DocumentPayload
		err := json.Unmarshal([]byte(entry.Payload), &rawPayload)
		if err != nil {
			fmt.Printf("Warning: Failed to decode payload for entry %s. Attempting fallback decoding. Payload: %s, Error: %v\n", entry.Hash, entry.Payload, err)

			// Fallback: Attempt to decode double-encoded payload
			var doubleEncodedPayload string
			if json.Unmarshal([]byte(entry.Payload), &doubleEncodedPayload) == nil {
				if json.Unmarshal([]byte(doubleEncodedPayload), &rawPayload) == nil {
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

		// Skip deleted documents
		if rawPayload.Op == "DEL" {
			continue
		}

		// Add to results using the "key" field
		results[rawPayload.Key] = rawPayload.Value
	}

	if len(results) == 0 {
		fmt.Println("Debug: No entries found in the log")
	}

	return results, nil
}
