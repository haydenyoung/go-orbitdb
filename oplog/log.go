package oplog

import (
	"errors"
	"fmt"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/keystore"
	"sort"
	"sync"

	"orbitdb/go-orbitdb/storage"
)

// Log represents an append-only log
type Log struct {
	id       string
	identity *identitytypes.Identity
	clock    Clock
	head     *EncodedEntry
	entries  storage.Storage
	keystore *keystore.KeyStore
	mu       sync.RWMutex
}

// NewLog creates a new log instance
func NewLog(id string, identity *identitytypes.Identity, entryStorage storage.Storage, keyStore *keystore.KeyStore) (*Log, error) {
	if id == "" {
		return nil, errors.New("log ID is required")
	}
	if identity == nil || !identitytypes.IsIdentity(identity) {
		return nil, errors.New("valid identity is required")
	}

	// Default to memory storage if none is provided
	if entryStorage == nil {
		entryStorage = storage.NewMemoryStorage()
	}

	// If no KeyStore is provided, create a new in-memory KeyStore
	if keyStore == nil {
		keyStore = keystore.NewKeyStore(storage.NewMemoryStorage())
	}

	// Ensure the KeyStore has the key for this identity
	if !keyStore.HasKey(identity.ID) {
		_, err := keyStore.CreateKey(identity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create key for identity %s: %w", identity.ID, err)
		}
	}

	return &Log{
		id:       id,
		identity: identity,
		clock:    NewClock(identity.ID, 0),
		entries:  entryStorage,
		keystore: keyStore,
	}, nil
}

// Append adds a new entry to the log
func (l *Log) Append(payload string) (*EncodedEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if payload == "" {
		return nil, errors.New("payload is required")
	}

	l.clock = TickClock(l.clock)

	var next []string
	if l.head != nil {
		next = []string{l.head.Hash}
	}

	entry := NewEntry(l.keystore, l.identity, l.id, payload, l.clock, next, nil)

	if err := l.entries.Put(entry.Hash, entry.Bytes); err != nil {
		return nil, fmt.Errorf("failed to store entry: %w", err)
	}

	l.head = &entry
	return &entry, nil
}

// Get retrieves an entry by its hash
func (l *Log) Get(hash string) (*EncodedEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	data, err := l.entries.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry for hash %s: %w", hash, err)
	}

	entry, err := Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode entry for hash %s: %w", hash, err)
	}

	if !VerifyEntrySignature(l.keystore, l.identity, entry) {
		return nil, fmt.Errorf("invalid signature for entry %s", hash)
	}

	return &entry, nil
}

// Values retrieves all entries in the log, sorted using CompareClocks
func (l *Log) Values() ([]EncodedEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]EncodedEntry, 0)
	ch, err := l.entries.Iterator()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over entries: %w", err)
	}

	for kv := range ch {
		entry, err := Decode([]byte(kv[1]))
		if err != nil {
			fmt.Printf("Warning: Skipping invalid entry with error: %s\n", err)
			continue
		}

		if !VerifyEntrySignature(l.keystore, l.identity, entry) {
			fmt.Printf("Warning: Skipping entry with invalid signature: %s\n", entry.Hash)
			continue
		}

		entries = append(entries, entry)
	}

	// Sort entries using CompareClocks
	sort.Slice(entries, func(i, j int) bool {
		return CompareClocks(entries[i].Clock, entries[j].Clock) < 0
	})

	return entries, nil
}

func (l *Log) Traverse(startHash string, shouldStop func(*EncodedEntry) bool) ([]*EncodedEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var traversed []*EncodedEntry
	visited := make(map[string]bool)

	// Start traversal from the specified entry or the current head
	var stack []*EncodedEntry
	if startHash != "" {
		startEntry, err := l.Get(startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to start traversal from entry: %w", err)
		}
		stack = []*EncodedEntry{startEntry}
	} else if l.head != nil {
		stack = []*EncodedEntry{l.head}
	} else {
		return nil, errors.New("no starting point for traversal")
	}

	// Perform the traversal
	for len(stack) > 0 {
		// Pop the last element from the stack
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Skip already visited entries
		if visited[entry.Hash] {
			continue
		}

		// Verify the signature before processing
		if !VerifyEntrySignature(l.keystore, l.identity, *entry) {
			fmt.Printf("Warning: Skipping entry with invalid signature: %s\n", entry.Hash)
			continue
		}

		// Mark as visited
		visited[entry.Hash] = true

		// Add the entry to the traversed list
		traversed = append(traversed, entry)

		// Apply the stopping condition
		if shouldStop != nil && shouldStop(entry) {
			break
		}

		// Load and add the `next` entries to the stack
		for _, nextHash := range entry.Entry.Next {
			nextEntry, err := l.Get(nextHash)
			if err != nil {
				fmt.Printf("Warning: Failed to load next entry %s: %s\n", nextHash, err)
				continue
			}
			stack = append(stack, nextEntry)
		}
	}

	return traversed, nil
}

// Clear removes all entries from the log
func (l *Log) Clear() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.entries.Clear(); err != nil {
		return fmt.Errorf("failed to clear entries: %w", err)
	}

	l.head = nil
	return nil
}

// Head returns the current head of the log or an error if the head is nil
func (l *Log) Head() (*EncodedEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.head == nil {
		return nil, errors.New("log head is nil")
	}

	return l.head, nil
}

// Close closes the log and its underlying storage
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.entries.Close()
}
