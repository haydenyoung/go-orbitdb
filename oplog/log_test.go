package oplog

import (
	"testing"

	"orbitdb/go-orbitdb/storage"
)

func TestNewLog(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create new log: %v", err)
	}

	if log == nil {
		t.Fatal("Expected log to be non-nil")
	}

	if logID != log.ID {
		t.Errorf("Expected log ID to be '%s', got '%s'", logID, log.ID)
	}

	if log.Identity != identity {
		t.Error("Log identity does not match the provided identity")
	}

	if log.Clock.ID != identity.ID || log.Clock.Time != 0 {
		t.Errorf("Expected clock to be initialized with ID '%s' and Time 0, got ID '%s' and Time %d",
			identity.ID, log.Clock.ID, log.Clock.Time)
	}
}

func TestLog_AppendAndGet(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create new log: %v", err)
	}

	payloads := []string{"first entry", "second entry", "third entry"}
	appendedEntries := make([]*EncodedEntry, 0, len(payloads))

	for _, payload := range payloads {
		entry, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
		appendedEntries = append(appendedEntries, entry)
	}

	for i, appendedEntry := range appendedEntries {
		retrievedEntry, err := log.Get(appendedEntry.Hash)
		if err != nil {
			t.Fatalf("Failed to get entry: %v", err)
		}
		if retrievedEntry.Hash != appendedEntry.Hash {
			t.Errorf("Retrieved entry hash does not match appended entry hash. Expected %s, got %s",
				appendedEntry.Hash, retrievedEntry.Hash)
		}
		if retrievedEntry.Payload != appendedEntry.Payload {
			t.Errorf("Retrieved entry payload does not match appended entry payload. Expected %s, got %s",
				appendedEntry.Payload, retrievedEntry.Payload)
		}
		if retrievedEntry.Payload != payloads[i] {
			t.Errorf("Retrieved entry payload does not match expected payload. Expected %s, got %s",
				payloads[i], retrievedEntry.Payload)
		}
	}
}

func TestLog_AppendAndRetrieve(t *testing.T) {
	// Step 1: Set up the test keystore and identity
	ks, identity := setupTestKeyStoreAndIdentity(t)

	// Step 2: Create a new log instance
	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create new log: %v", err)
	}

	// Step 3: Append multiple Entries to the log
	payloads := []string{"first entry", "second entry", "third entry"}
	appendedEntries := make([]*EncodedEntry, 0, len(payloads))

	for _, payload := range payloads {
		entry, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
		appendedEntries = append(appendedEntries, entry)
	}

	// Step 4: Retrieve each appended entry using the Get method and verify
	for i, appendedEntry := range appendedEntries {
		retrievedEntry, err := log.Get(appendedEntry.Hash)
		if err != nil {
			t.Fatalf("Failed to get entry: %v", err)
		}
		if retrievedEntry.Hash != appendedEntry.Hash {
			t.Errorf("Retrieved entry hash does not match appended entry hash. Expected %s, got %s", appendedEntry.Hash, retrievedEntry.Hash)
		}
		if retrievedEntry.Payload != appendedEntry.Payload {
			t.Errorf("Retrieved entry payload does not match appended entry payload. Expected %s, got %s", appendedEntry.Payload, retrievedEntry.Payload)
		}
		if retrievedEntry.Payload != payloads[i] {
			t.Errorf("Retrieved entry payload does not match expected payload. Expected %s, got %s", payloads[i], retrievedEntry.Payload)
		}
	}

	// Step 5: Test the Values method to ensure all Entries are correctly stored and ordered
	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to get log values: %v", err)
	}

	if len(entries) != len(payloads) {
		t.Errorf("Expected %d Entries, got %d", len(payloads), len(entries))
	}

	// The Entries should be sorted in the order they were appended
	for i, entry := range entries {
		if entry.Payload != payloads[i] {
			t.Errorf("Entry %d payload mismatch: expected '%s', got '%s'", i, payloads[i], entry.Payload)
		}
	}
}

func TestLog_Values(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create new log: %v", err)
	}

	payloads := []string{"entry1", "entry2", "entry3"}
	for _, payload := range payloads {
		_, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to get log values: %v", err)
	}

	if len(entries) != len(payloads) {
		t.Errorf("Expected %d Entries, got %d", len(payloads), len(entries))
	}

	for i, entry := range entries {
		if entry.Payload != payloads[i] {
			t.Errorf("Entry %d payload mismatch: expected '%s', got '%s'",
				i, payloads[i], entry.Payload)
		}
	}
}

func TestLog_Traverse(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create new log: %v", err)
	}

	payloads := []string{"entry1", "entry2", "entry3", "entry4", "entry5"}
	for _, payload := range payloads {
		_, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	count := 0
	shouldStop := func(e *EncodedEntry) bool {
		count++
		return count >= 3
	}

	traversedEntries, err := log.Traverse("", shouldStop)
	if err != nil {
		t.Fatalf("Failed to traverse log: %v", err)
	}

	if len(traversedEntries) != 3 {
		t.Errorf("Expected to traverse 3 Entries, got %d", len(traversedEntries))
	}

	for i, entry := range traversedEntries {
		expectedPayload := payloads[len(payloads)-1-i]
		if entry.Payload != expectedPayload {
			t.Errorf("Traversed entry %d payload mismatch: expected '%s', got '%s'",
				i, expectedPayload, entry.Payload)
		}
	}
}

func TestLog_JoinEntry(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	// Create a new entry to join
	clock := NewClock(identity.ID, 1)
	entry := NewEntry(ks, identity, logID, "joined entry", clock, nil, nil)

	processed := make(map[string]bool)
	err = log.JoinEntry(&entry, processed)
	if err != nil {
		t.Fatalf("Failed to join entry: %v", err)
	}

	retrievedEntry, err := log.Get(entry.Hash)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrievedEntry.Hash != entry.Hash {
		t.Errorf("Joined entry hash does not match. Expected %s, got %s",
			entry.Hash, retrievedEntry.Hash)
	}
	if retrievedEntry.Payload != entry.Payload {
		t.Errorf("Joined entry payload does not match. Expected '%s', got '%s'",
			entry.Payload, retrievedEntry.Payload)
	}
}

func TestLog_Join(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage1 := storage.NewMemoryStorage()
	log1, err := NewLog(logID, identity, entryStorage1, ks)
	if err != nil {
		t.Fatalf("Failed to create log1: %v", err)
	}

	entryStorage2 := storage.NewMemoryStorage()
	log2, err := NewLog(logID, identity, entryStorage2, ks)
	if err != nil {
		t.Fatalf("Failed to create log2: %v", err)
	}

	payloads1 := []string{"entry1-log1", "entry2-log1"}
	for _, payload := range payloads1 {
		_, err := log1.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append to log1: %v", err)
		}
	}

	payloads2 := []string{"entry1-log2", "entry2-log2"}
	for _, payload := range payloads2 {
		_, err := log2.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append to log2: %v", err)
		}
	}

	err = log1.Join(log2)
	if err != nil {
		t.Fatalf("Failed to join log2 into log1: %v", err)
	}

	entries, err := log1.Values()
	if err != nil {
		t.Fatalf("Failed to get log1 values: %v", err)
	}

	expectedEntryCount := len(payloads1) + len(payloads2)
	if len(entries) != expectedEntryCount {
		t.Errorf("Expected %d Entries after join, got %d",
			expectedEntryCount, len(entries))
	}

	// Optionally, check that the Entries contain the expected payloads
	payloadSet := make(map[string]bool)
	for _, payload := range append(payloads1, payloads2...) {
		payloadSet[payload] = true
	}
	for _, entry := range entries {
		if !payloadSet[entry.Payload] {
			t.Errorf("Unexpected entry payload: '%s'", entry.Payload)
		}
	}
}

func TestLog_Clear(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	payloads := []string{"entry1", "entry2", "entry3"}
	for _, payload := range payloads {
		_, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	err = log.Clear()
	if err != nil {
		t.Fatalf("Failed to clear the log: %v", err)
	}

	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to get log values: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 Entries after clear, got %d", len(entries))
	}

	head, err := log.Head()
	if err == nil {
		t.Errorf("Expected head to be nil after clear, but got head with hash %s", head.Hash)
	}
}

func TestLog_Head(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)

	logID := "test-log"
	entryStorage := storage.NewMemoryStorage()
	log, err := NewLog(logID, identity, entryStorage, ks)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	_, err = log.Head()
	if err == nil {
		t.Error("Expected error when getting head of empty log, but got none")
	}

	entry, err := log.Append("first entry")
	if err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	head, err := log.Head()
	if err != nil {
		t.Fatalf("Failed to get log head: %v", err)
	}

	if head.Hash != entry.Hash {
		t.Errorf("Head hash does not match the last appended entry. Expected %s, got %s",
			entry.Hash, head.Hash)
	}
}
