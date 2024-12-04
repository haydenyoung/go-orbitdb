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
	ID       string
	Identity *identitytypes.Identity
	Clock    Clock
	Head     *EncodedEntry
	Entries  storage.Storage
	keystore *keystore.KeyStore
	Mu       sync.RWMutex
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
		ID:       id,
		Identity: identity,
		Clock:    NewClock(identity.ID, 0),
		Entries:  entryStorage,
		keystore: keyStore,
	}, nil
}

// Append adds a new entry to the log
func (l *Log) Append(payload string) (*EncodedEntry, error) {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	if payload == "" {
		return nil, errors.New("payload is required")
	}

	l.Clock = TickClock(l.Clock)

	var next []string
	if l.Head != nil {
		next = []string{l.Head.Hash}
	}

	entry := NewEntry(l.keystore, l.Identity, l.ID, payload, l.Clock, next, nil)

	if err := l.Entries.Put(entry.Hash, entry.Bytes); err != nil {
		return nil, fmt.Errorf("failed to store entry: %w", err)
	}

	l.Head = &entry
	return &entry, nil
}

// Get retrieves an entry by its hash
func (l *Log) Get(hash string) (*EncodedEntry, error) {
	l.Mu.RLock()
	defer l.Mu.RUnlock()

	data, err := l.Entries.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry for hash %s: %w", hash, err)
	}

	entry, err := Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode entry for hash %s: %w", hash, err)
	}

	if !VerifyEntrySignature(l.keystore, entry) {
		return nil, fmt.Errorf("invalid signature for entry %s", hash)
	}

	return &entry, nil
}

// Values retrieves all Entries in the log, sorted using CompareClocks
func (l *Log) Values() ([]EncodedEntry, error) {
	l.Mu.RLock()
	defer l.Mu.RUnlock()

	entries := make([]EncodedEntry, 0)
	ch, err := l.Entries.Iterator()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over Entries: %w", err)
	}

	for kv := range ch {
		entry, err := Decode([]byte(kv[1]))
		if err != nil {
			fmt.Printf("Warning: Skipping invalid entry with error: %s\n", err)
			continue
		}

		if !VerifyEntrySignature(l.keystore, entry) {
			fmt.Printf("Warning: Skipping entry with invalid signature: %s\n", entry.Hash)
			continue
		}

		entries = append(entries, entry)
	}

	// Sort Entries using CompareClocks
	sort.Slice(entries, func(i, j int) bool {
		return CompareClocks(entries[i].Clock, entries[j].Clock) < 0
	})

	return entries, nil
}

func (l *Log) Traverse(startHash string, shouldStop func(*EncodedEntry) bool) ([]*EncodedEntry, error) {
	l.Mu.RLock()
	defer l.Mu.RUnlock()

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
	} else if l.Head != nil {
		stack = []*EncodedEntry{l.Head}
	} else {
		return nil, errors.New("no starting point for traversal")
	}

	// Perform the traversal
	for len(stack) > 0 {
		// Pop the last element from the stack
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Skip already visited Entries
		if visited[entry.Hash] {
			continue
		}

		// Verify the signature before processing
		if !VerifyEntrySignature(l.keystore, *entry) {
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

		// Load and add the `next` Entries to the stack
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

func (l *Log) JoinEntry(entry *EncodedEntry, processed map[string]bool) error {
	// Check if the entry belongs to the current log
	if entry.Entry.ID != l.ID {
		return fmt.Errorf("entry ID '%s' does not match log ID '%s'", entry.Entry.ID, l.ID)
	}

	if !VerifyEntrySignature(l.keystore, *entry) {
		return fmt.Errorf("invalid signature for entry %s", entry.Hash)
	}

	// Initialize a stack for iterative processing
	stack := []*EncodedEntry{entry}

	for len(stack) > 0 {
		// Pop an entry from the stack
		currentEntry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check if the entry is already processed
		if processed[currentEntry.Hash] {
			continue
		}
		processed[currentEntry.Hash] = true

		// Add the entry to storage
		err := l.Entries.Put(currentEntry.Hash, currentEntry.Bytes)
		if err != nil {
			return fmt.Errorf("failed to store entry: %w", err)
		}

		// Update the log head if the new entry has a more recent clock
		if l.Head == nil || CompareClocks(currentEntry.Clock, l.Head.Clock) > 0 {
			l.Head = currentEntry
		}
	}

	return nil
}

func (l *Log) Join(otherLog *Log) error {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	// Check if the other log has the same ID
	if otherLog.ID != l.ID {
		return fmt.Errorf("log ID '%s' does not match other log ID '%s'", l.ID, otherLog.ID)
	}

	// Get all Entries from the other log
	otherEntries, err := otherLog.Values()
	if err != nil {
		return fmt.Errorf("failed to retrieve Entries from other log: %w", err)
	}

	// Process each entry using the JoinEntry method
	processed := make(map[string]bool)
	for _, entry := range otherEntries {
		if err := l.JoinEntry(&entry, processed); err != nil {
			fmt.Printf("Warning: Skipping invalid or duplicate entry %s: %v\n", entry.Hash, err)
		}
	}

	return nil
}

// Clear removes all Entries from the log
func (l *Log) Clear() error {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	if err := l.Entries.Clear(); err != nil {
		return fmt.Errorf("failed to clear Entries: %w", err)
	}

	l.Head = nil
	return nil
}

// Close closes the log and its underlying storage
func (l *Log) Close() error {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	return l.Entries.Close()
}
