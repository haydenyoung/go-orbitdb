package syncutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type Sync struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ID        string           // Simplified host ID
	pubsub    *LocalPubSub     // Simplified in-memory PubSub
	log       *Log             // Simplified Log
	SyncedCh  chan SyncedEntry // Channel for synced entries
	TopicName string           // Topic name
	peers     map[string]bool  // Simplified peer tracking
	mu        sync.Mutex       // Mutex for thread-safe peer access
	wg        sync.WaitGroup   // WaitGroup for goroutines
}

type SyncedEntry struct {
	PeerID string
	Entry  EncodedEntry
}

type EncodedEntry struct {
	Hash string
}

// Log is a simplified version of a log for testing purposes.
type Log struct {
	ID string
}

// NewSync creates a new simplified Sync instance.
func NewSync(id string, pubsub *LocalPubSub, log *Log) *Sync {
	ctx, cancel := context.WithCancel(context.Background())
	topicName := fmt.Sprintf("local-test/%s", log.ID)

	return &Sync{
		ctx:       ctx,
		cancel:    cancel,
		ID:        id,
		pubsub:    pubsub,
		log:       log,
		SyncedCh:  make(chan SyncedEntry, 10),
		TopicName: topicName,
		peers:     make(map[string]bool),
	}
}

// Start begins the synchronization process.
func (s *Sync) Start() error {
	sub, err := s.pubsub.Subscribe(s.TopicName)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}
	log.Println("Sync started: subscribed to topic")

	s.wg.Add(1)
	go s.processMessages(sub)

	return nil
}

// Stop halts the synchronization process and cleans up resources.
func (s *Sync) Stop() {
	s.cancel()
	s.wg.Wait()
	log.Println("Sync stopped.")
}

// Add broadcasts a log head to all peers.
func (s *Sync) Add(entry EncodedEntry) error {
	entryData := struct {
		PeerID string
		Entry  EncodedEntry
	}{
		PeerID: s.ID,
		Entry:  entry,
	}

	data, err := json.Marshal(entryData)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if err := s.pubsub.Publish(s.TopicName, data); err != nil {
		return fmt.Errorf("failed to publish entry: %w", err)
	}

	log.Printf("Broadcasted entry: %s from peer: %s\n", entry.Hash, s.ID)
	return nil
}

// processMessages listens for messages and processes them.
func (s *Sync) processMessages(sub *LocalSubscription) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case msg := <-sub.Messages:
			var payload struct {
				PeerID string
				Entry  EncodedEntry
			}
			if err := json.Unmarshal(msg, &payload); err != nil {
				log.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}

			// Ignore messages from self
			if payload.PeerID == s.ID {
				log.Println("Ignoring message from self")
				continue
			}

			// Process messages from other peers
			log.Printf("Received entry: %s from peer: %s\n", payload.Entry.Hash, payload.PeerID)
			s.SyncedCh <- SyncedEntry{PeerID: payload.PeerID, Entry: payload.Entry}
		}
	}
}

// DiscoverPeers lists the peers in the topic.
func (s *Sync) DiscoverPeers() []string {
	return s.pubsub.ListPeers(s.TopicName)
}
