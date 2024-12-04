package syncutils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"log"
	"orbitdb/go-orbitdb/oplog"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

// Sync handles synchronization for the Log.
type Sync struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ID        string           // Peer ID
	pubsub    *pubsub.PubSub   // libp2p PubSub instance
	log       *oplog.Log       // Actual Log structure
	SyncedCh  chan SyncedEntry // Channel for synced entries
	TopicName string           // PubSub topic name
	topic     *pubsub.Topic    // Subscribed topic
	sub       *pubsub.Subscription
	mu        sync.Mutex      // Protects peer access
	wg        sync.WaitGroup  // WaitGroup for goroutines
	peerMap   map[string]bool // Tracks connected peers
}

// SyncedEntry represents an entry received from a peer.
type SyncedEntry struct {
	PeerID string
	Entry  oplog.EncodedEntry
}

// NewSync initializes a new Sync instance for the Log.
func NewSync(host host.Host, pubsub *pubsub.PubSub, log *oplog.Log) *Sync {
	ctx, cancel := context.WithCancel(context.Background())
	topicName := fmt.Sprintf("orbit-sync/%s", log.ID)

	return &Sync{
		ctx:       ctx,
		cancel:    cancel,
		ID:        host.ID().String(),
		pubsub:    pubsub,
		log:       log,
		SyncedCh:  make(chan SyncedEntry, 10),
		TopicName: topicName,
		peerMap:   make(map[string]bool),
	}
}

// Start begins the synchronization process.
func (s *Sync) Start() error {
	var err error

	// Join the PubSub topic
	s.topic, err = s.pubsub.Join(s.TopicName)
	if err != nil {
		return fmt.Errorf("failed to join topic: %w", err)
	}

	// Subscribe to the topic
	s.sub, err = s.topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	log.Printf("Sync started: subscribed to topic %s", s.TopicName)

	// Track peer joining
	go s.trackPeers()

	s.wg.Add(1)
	go s.processMessages()

	return nil
}

// Stop halts the synchronization process.
func (s *Sync) Stop() {
	s.cancel()
	s.wg.Wait()

	s.sub.Cancel()
	if err := s.topic.Close(); err != nil {
		log.Printf("Error closing topic: %v", err)
	}

	log.Println("Sync stopped.")
}

// Add creates an entry in the log and broadcasts it to peers.
func (s *Sync) Add(payload string) error {
	s.log.Mu.Lock()
	defer s.log.Mu.Unlock()

	entry := oplog.EncodedEntry{
		Entry: oplog.Entry{
			ID:       s.log.ID,
			Payload:  payload,
			Clock:    s.log.Clock,
			V:        1,
			Identity: s.log.Identity.ID,
		},
		Bytes: []byte(payload),                // Placeholder for actual encoding
		Hash:  fmt.Sprintf("%x", s.log.Clock), // Example hash generation
	}

	// Add to the log
	if err := s.log.Entries.Put(entry.Hash, entry.Bytes); err != nil {
		return fmt.Errorf("failed to store entry in log: %w", err)
	}

	// Broadcast to peers
	entryData := struct {
		PeerID string
		Entry  oplog.EncodedEntry
	}{
		PeerID: s.ID,
		Entry:  entry,
	}

	data, err := json.Marshal(entryData)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if err := s.topic.Publish(s.ctx, data); err != nil {
		return fmt.Errorf("failed to publish entry: %w", err)
	}

	log.Printf("Broadcasted entry: %s from peer: %s\n", payload, s.ID)
	return nil
}

// processMessages listens for incoming messages from PubSub.
func (s *Sync) processMessages() {
	defer s.wg.Done()

	for {
		msg, err := s.sub.Next(s.ctx)
		if err != nil {
			if s.ctx.Err() != nil {
				return // Context canceled
			}
			log.Printf("Error reading message: %v\n", err)
			continue
		}

		log.Printf("Received message from %s, expected self: %s\n", msg.ReceivedFrom, s.ID)

		// Ignore messages from self
		if msg.ReceivedFrom.String() == s.ID {
			log.Println("Ignoring message from self")
			continue
		}

		var payload struct {
			PeerID string
			Entry  oplog.EncodedEntry
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		log.Printf("Processing entry: %s from peer: %s\n", payload.Entry.Payload, payload.PeerID)

		// Add the entry to the log
		s.log.Mu.Lock()
		if err := s.log.Entries.Put(payload.Entry.Hash, payload.Entry.Bytes); err != nil {
			log.Printf("Failed to store entry in log: %v\n", err)
			s.log.Mu.Unlock()
			continue
		}
		s.log.Mu.Unlock()

		// Notify listeners
		s.SyncedCh <- SyncedEntry{PeerID: payload.PeerID, Entry: payload.Entry}
	}
}

// trackPeers listens for changes in peer list and tracks peers joining or leaving the topic.
func (s *Sync) trackPeers() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Discover peers
			peers := s.DiscoverPeers()
			for _, p := range peers {
				if _, exists := s.peerMap[p.String()]; !exists {
					s.peerMap[p.String()] = true
					s.PeerJoin(p.String()) // New peer has joined
				}
			}

			// Check for peers leaving (you might want to add a more robust method)
			for peerID := range s.peerMap {
				found := false
				for _, p := range peers {
					if peerID == p.String() {
						found = true
						break
					}
				}

				if !found {
					delete(s.peerMap, peerID)
					s.PeerLeave(peerID) // Peer has left
				}
			}

			// Sleep for a short period before re-checking the peers
			<-time.After(2 * time.Second)
		}
	}
}

// PeerJoin method to handle new peer joins
func (s *Sync) PeerJoin(peerID string) {
	// Log the event with additional context
	log.Printf("Peer joined: %s at %s", peerID, time.Now().Format(time.RFC3339))

	// Optionally send a "join" message to the `SyncedCh` channel
	joinEntry := oplog.EncodedEntry{
		Entry: oplog.Entry{
			ID:       fmt.Sprintf("%s-join", peerID),
			Payload:  fmt.Sprintf("Peer %s has joined the network", peerID),
			Clock:    s.log.Clock,
			V:        1,
			Identity: s.log.Identity.ID,
		},
		Bytes: []byte(fmt.Sprintf("Peer %s has joined the network", peerID)),
		Hash:  fmt.Sprintf("%x", time.Now().UnixNano()),
	}

	// Send the join event as an entry to the synced channel
	s.SyncedCh <- SyncedEntry{
		PeerID: peerID,
		Entry:  joinEntry,
	}
}

// PeerLeave method to handle peer disconnections
func (s *Sync) PeerLeave(peerID string) {
	// Log the event with additional context
	log.Printf("Peer left: %s at %s", peerID, time.Now().Format(time.RFC3339))

	// Optionally send a "leave" message to the `SyncedCh` channel
	leaveEntry := oplog.EncodedEntry{
		Entry: oplog.Entry{
			ID:       fmt.Sprintf("%s-leave", peerID),
			Payload:  fmt.Sprintf("Peer %s has left the network", peerID),
			Clock:    s.log.Clock,
			V:        1,
			Identity: s.log.Identity.ID,
		},
		Bytes: []byte(fmt.Sprintf("Peer %s has left the network", peerID)),
		Hash:  fmt.Sprintf("%x", time.Now().UnixNano()),
	}

	// Send the leave event as an entry to the synced channel
	s.SyncedCh <- SyncedEntry{
		PeerID: peerID,
		Entry:  leaveEntry,
	}
}

// DiscoverPeers lists peers connected to the topic.
func (s *Sync) DiscoverPeers() []peer.ID {
	return s.topic.ListPeers()
}
