package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"orbitdb/go-orbitdb/oplog"
)

const (
	DefaultTimeout         = 30 * time.Second
	MaxMessageSize         = 4096 // Adjust as needed for your log entry size
	DefaultEventBufferSize = 100
)

// Sync represents the synchronization protocol for a single-head log.
type Sync struct {
	ctx       context.Context
	cancel    context.CancelFunc
	Host      host.Host
	Pubsub    *pubsub.PubSub
	Log       *oplog.Log
	peers     map[peer.ID]bool
	peersLock sync.Mutex
	onSynced  func(peer.ID, oplog.EncodedEntry)
	TopicName string
	Topic     *pubsub.Topic
	events    chan SyncEvent
}

// SyncEvent represents synchronization events.
type SyncEvent struct {
	Type  string
	Peer  peer.ID
	Error error
	Head  oplog.EncodedEntry
}

// NewSync creates a new Sync instance for single-head logs.
func NewSync(host host.Host, ps *pubsub.PubSub, log *oplog.Log, onSynced func(peer.ID, oplog.EncodedEntry)) (*Sync, error) {
	ctx, cancel := context.WithCancel(context.Background())

	topicName := fmt.Sprintf("/orbitdb/heads/%s", log.ID)
	topic, err := ps.Join(topicName)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to join pubsub topic: %w", err)
	}

	return &Sync{
		ctx:       ctx,
		cancel:    cancel,
		Host:      host,
		Pubsub:    ps,
		Log:       log,
		peers:     make(map[peer.ID]bool),
		onSynced:  onSynced,
		TopicName: topicName,
		Topic:     topic,
		events:    make(chan SyncEvent, DefaultEventBufferSize),
	}, nil
}

// Start begins the synchronization protocol.
func (s *Sync) Start() error {
	// Subscribe to the pubsub topic for receiving updates
	sub, err := s.Topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to pubsub topic: %w", err)
	}

	log.Println("Sync started: subscribed to pubsub topic")

	// Handle incoming pubsub messages
	go s.handlePubSubMessages(sub)

	// Automatically connect to existing peers in the topic
	peers := s.Pubsub.ListPeers(s.TopicName)
	for _, p := range peers {
		if p == s.Host.ID() {
			continue
		}
		go s.connectToPeer(p)
	}

	return nil
}

// Stop terminates the synchronization protocol.
func (s *Sync) Stop() error {
	s.cancel()
	close(s.events)

	if err := s.Topic.Close(); err != nil {
		return fmt.Errorf("failed to close pubsub topic: %w", err)
	}

	log.Println("Sync stopped")
	return nil
}

// Add adds a log head to be broadcasted.
func (s *Sync) Add(head oplog.EncodedEntry) error {
	data, err := json.Marshal(head)
	if err != nil {
		return fmt.Errorf("failed to marshal head: %w", err)
	}

	log.Printf("Broadcasting head: %s\n", head.Hash)

	if err := s.Topic.Publish(s.ctx, data); err != nil {
		return fmt.Errorf("failed to publish head: %w", err)
	}

	return nil
}

// handlePubSubMessages processes incoming pubsub messages.
func (s *Sync) handlePubSubMessages(sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(s.ctx)
		if err != nil {
			if s.ctx.Err() != nil {
				log.Println("Pubsub context canceled, stopping message handler.")
				return
			}
			s.emitError(fmt.Errorf("failed to read pubsub message: %w", err))
			continue
		}

		// Ignore messages from self
		if msg.GetFrom() == s.Host.ID() {
			continue
		}

		var head oplog.EncodedEntry
		if err := json.Unmarshal(msg.Data, &head); err != nil {
			log.Printf("Failed to unmarshal pubsub message from peer %s: %v\n", msg.GetFrom(), err)
			s.emitError(fmt.Errorf("failed to unmarshal head: %w", err))
			continue
		}

		peerID := msg.GetFrom()

		log.Printf("Processing message from peer %s: %s\n", peerID, head.Hash)

		if s.onSynced != nil {
			s.onSynced(peerID, head)
		}

		s.emitEvent(SyncEvent{Type: "join", Peer: peerID, Head: head})
	}
}

// connectToPeer establishes synchronization via PubSub only.
func (s *Sync) connectToPeer(p peer.ID) {
	s.peersLock.Lock()
	if s.peers[p] {
		s.peersLock.Unlock()
		return
	}
	s.peers[p] = true
	s.peersLock.Unlock()

	// No direct stream handling here; synchronization relies solely on PubSub
	log.Printf("Connected to peer %s via PubSub\n", p)
}

// addPeer and removePeer methods can be simplified or removed if not needed
// depending on your synchronization logic.

// emitEvent emits a synchronization event.
func (s *Sync) emitEvent(event SyncEvent) {
	select {
	case s.events <- event:
	default:
		log.Println("Event queue full, dropping event")
	}
}

// emitError emits an error synchronization event.
func (s *Sync) emitError(err error) {
	log.Printf("Error: %v\n", err)
	s.emitEvent(SyncEvent{Type: "error", Error: err})
}
