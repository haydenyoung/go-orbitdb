package syncutils

import (
	"sync"
)

// LocalPubSub is a simple in-memory PubSub for testing purposes.
type LocalPubSub struct {
	topics map[string]*LocalTopic
	mu     sync.Mutex
}

type LocalTopic struct {
	subscribers []*LocalSubscription
	mu          sync.Mutex
}

type LocalSubscription struct {
	Messages chan []byte
}

// NewLocalPubSub initializes a LocalPubSub instance.
func NewLocalPubSub() *LocalPubSub {
	return &LocalPubSub{
		topics: make(map[string]*LocalTopic),
	}
}

// Subscribe subscribes to a topic.
func (ps *LocalPubSub) Subscribe(topicName string) (*LocalSubscription, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	topic, exists := ps.topics[topicName]
	if !exists {
		topic = &LocalTopic{}
		ps.topics[topicName] = topic
	}

	sub := &LocalSubscription{Messages: make(chan []byte, 10)}
	topic.mu.Lock()
	topic.subscribers = append(topic.subscribers, sub)
	topic.mu.Unlock()

	return sub, nil
}

// Publish publishes a message to a topic.
func (ps *LocalPubSub) Publish(topicName string, data []byte) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	topic, exists := ps.topics[topicName]
	if !exists {
		return nil
	}

	topic.mu.Lock()
	defer topic.mu.Unlock()
	for _, sub := range topic.subscribers {
		select {
		case sub.Messages <- data:
		default:
			// Drop message if buffer is full
		}
	}

	return nil
}

// ListPeers lists peers for a topic (for simplicity, return static values).
func (ps *LocalPubSub) ListPeers(topicName string) []string {
	return []string{"peer1", "peer2"}
}
