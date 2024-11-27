package databases

import (
	"encoding/json"
	"fmt"
	"orbitdb/go-orbitdb/oplog"
	"sort"
)

// Events represents an immutable, append-only event log database.
type Events struct {
	*Database // Embeds the base Database for shared functionality
}

// NewEvents creates a new Events database instance.
func NewEvents(db *Database) *Events {
	return &Events{
		Database: db,
	}
}

// Add adds an event to the event log.
func (e *Events) Add(value interface{}) (string, error) {
	op := map[string]interface{}{
		"op":    "ADD",
		"key":   nil,
		"value": value,
	}

	payload, err := json.Marshal(op)
	if err != nil {
		return "", fmt.Errorf("failed to serialize operation: %w", err)
	}

	fmt.Printf("Debug (Add): Payload being added: %s\n", string(payload))

	return e.AddOperation(string(payload))
}

// Get retrieves an event from the event log by its hash.
func (e *Events) Get(hash string) (interface{}, error) {
	entry, err := e.Log.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get log entry: %w", err)
	}

	fmt.Printf("Debug (Get): Raw payload: %s\n", entry.Payload)

	var rawPayload string
	// Attempt to unmarshal the outer payload
	err = json.Unmarshal([]byte(entry.Payload), &rawPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode outer payload: %w", err)
	}

	fmt.Printf("Debug (Get): Outer decoded payload: %s\n", rawPayload)

	var payload map[string]interface{}
	// Attempt to unmarshal the inner payload
	err = json.Unmarshal([]byte(rawPayload), &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode inner payload: %w", err)
	}

	return payload["value"], nil
}

// Iterator retrieves events from the event log with optional filters.
func (e *Events) Iterator(gt, gte, lt, lte string, amount int) ([]map[string]interface{}, error) {
	entries, err := e.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	// Sort entries deterministically by Clock and Hash as a tiebreaker
	sort.Slice(entries, func(i, j int) bool {
		clockCompare := CompareClocks(entries[i].Clock, entries[j].Clock)
		if clockCompare == 0 {
			return entries[i].Hash < entries[j].Hash
		}
		return clockCompare < 0
	})

	results := make([]map[string]interface{}, 0)

	for _, entry := range entries {
		// Construct combined key for deterministic filter comparison
		entryKey := fmt.Sprintf("%d:%s", entry.Clock.Time, entry.Hash)

		// Apply filters based on combined keys
		if gt != "" && entryKey <= gt {
			continue
		}
		if gte != "" && entryKey < gte {
			continue
		}
		if lt != "" && entryKey >= lt {
			continue
		}
		if lte != "" && entryKey > lte {
			continue
		}

		// Decode payload
		var payload map[string]interface{}
		err := json.Unmarshal([]byte(entry.Payload), &payload)
		if err != nil {
			// Attempt fallback decoding
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

		results = append(results, map[string]interface{}{
			"hash":  entry.Hash,
			"value": payload["value"],
		})

		// Limit results to the specified amount
		if amount > 0 && len(results) >= amount {
			break
		}
	}

	return results, nil
}

// All retrieves all events in the event log.
func (e *Events) All() ([]map[string]interface{}, error) {
	// Retrieve all log entries
	entries, err := e.Log.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log entries: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		var payload map[string]interface{}

		// Attempt to decode the payload
		err := json.Unmarshal([]byte(entry.Payload), &payload)
		if err != nil {
			// Fallback: Try to decode as double-encoded payload
			var doubleEncodedPayload string
			err = json.Unmarshal([]byte(entry.Payload), &doubleEncodedPayload)
			if err == nil {
				// Decode the inner payload
				err = json.Unmarshal([]byte(doubleEncodedPayload), &payload)
			}
		}

		if err != nil {
			// Log a warning and skip the entry if decoding fails
			fmt.Printf("Warning: Failed to decode payload for entry %s. Payload: %s, Error: %v\n", entry.Hash, entry.Payload, err)
			continue
		}

		// Append the result
		results = append(results, map[string]interface{}{
			"hash":  entry.Hash,
			"value": payload["value"],
		})
	}

	return results, nil
}

func CompareClocks(clock1 oplog.Clock, clock2 oplog.Clock) int {
	if clock1.Time != clock2.Time {
		if clock1.Time < clock2.Time {
			return -1
		}
		return 1
	}
	// Break ties using the identity string
	if clock1.ID < clock2.ID {
		return -1
	}
	if clock1.ID > clock2.ID {
		return 1
	}
	return 0
}
