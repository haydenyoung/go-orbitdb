package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"time"

	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	pinner "github.com/ipfs/boxo/pinning/pinner"
	"github.com/ipfs/boxo/pinning/pinner/dspinner"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	format "github.com/ipfs/go-ipld-format"
)

// DefaultTimeout is the timeout for IPFS block operations
const DefaultTimeout = 30 * time.Second

// IPFSBlockStorage is a Storage implementation backed by Boxo's blockservice.
type IPFSBlockStorage struct {
	blockstore blockstore.Blockstore
	blocksvc   blockservice.BlockService
	pinner     pinner.Pinner
	pin        bool
	timeout    time.Duration
}

type ipldPrimeNodeWrapper struct {
	node datamodel.Node
	cid  cid.Cid
}

func (w *ipldPrimeNodeWrapper) String() string {
	return fmt.Sprintf("ipldPrimeNodeWrapper(cid=%s)", w.cid.String())
}

func (w *ipldPrimeNodeWrapper) Tree(path string, depth int) []string {
	return []string{}
}

// Cid returns the CID of the node
func (w *ipldPrimeNodeWrapper) Cid() cid.Cid {
	return w.cid
}

// RawData returns the raw CBOR-encoded data of the node
func (w *ipldPrimeNodeWrapper) RawData() []byte {
	var buf bytes.Buffer
	if err := dagcbor.Encode(w.node, &buf); err != nil {
		panic(fmt.Sprintf("failed to encode node: %v", err))
	}
	return buf.Bytes()
}

// Loggable returns a loggable representation of the node
func (w *ipldPrimeNodeWrapper) Loggable() map[string]interface{} {
	return map[string]interface{}{"cid": w.cid.String()}
}

// Resolve resolves a path through the node (not implemented, returns an error for now)
func (w *ipldPrimeNodeWrapper) Resolve(path []string) (interface{}, []string, error) {
	return nil, nil, errors.New("Resolve not implemented")
}

// ResolveLink resolves a path to a link through the node (not implemented, returns an error for now)
func (w *ipldPrimeNodeWrapper) ResolveLink(path []string) (*format.Link, []string, error) {
	return nil, nil, errors.New("ResolveLink not implemented")
}

// Copy creates a copy of the node
func (w *ipldPrimeNodeWrapper) Copy() format.Node {
	return &ipldPrimeNodeWrapper{node: w.node, cid: w.cid}
}

// Links returns the links of the node (empty for now)
func (w *ipldPrimeNodeWrapper) Links() []*format.Link {
	// If your IPLD node has links, extract and convert them here
	return nil
}

// Stat returns statistics about the node
func (w *ipldPrimeNodeWrapper) Stat() (*format.NodeStat, error) {
	// You can provide a meaningful implementation if required
	return &format.NodeStat{}, nil
}

// Size returns the size of the node
func (w *ipldPrimeNodeWrapper) Size() (uint64, error) {
	return uint64(len(w.RawData())), nil
}

func wrapIPLDPrimeNode(node datamodel.Node, cid cid.Cid) format.Node {
	return &ipldPrimeNodeWrapper{node: node, cid: cid}
}

// NewIPFSBlockStorage creates a new IPFSBlockStorage instance using Boxo.
func NewIPFSBlockStorage(ctx context.Context, ds datastore.Batching, dserv format.DAGService, pin bool, timeout time.Duration) (*IPFSBlockStorage, error) {
	if ds == nil {
		return nil, errors.New("datastore is required")
	}

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create a blockstore
	bs := blockstore.NewBlockstore(ds)

	// Use nil for the exchange.Interface as this is a local implementation
	blocksvc := blockservice.New(bs, nil)

	// Create the pinner
	pinner, err := dspinner.New(ctx, ds, dserv)
	if err != nil {
		return nil, fmt.Errorf("failed to create pinner: %w", err)
	}

	return &IPFSBlockStorage{
		blockstore: bs,
		blocksvc:   blocksvc,
		pinner:     pinner,
		pin:        pin,
		timeout:    timeout,
	}, nil
}

// Put stores data as a block in IPFS.
func (s *IPFSBlockStorage) Put(key string, value []byte) error {
	c, err := cid.Decode(key)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Create a block with the given data and CID
	block, err := blocks.NewBlockWithCid(value, c)
	if err != nil {
		return fmt.Errorf("failed to create block: %w", err)
	}

	// Store the block in the blockstore
	err = s.blockstore.Put(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}

	// Optionally pin the block
	if s.pin {
		// Convert the block to an IPLD node
		nb := basicnode.Prototype.Any.NewBuilder()
		buf := bytes.NewReader(block.RawData())

		// Decode the block's raw data into an IPLD node
		if err := dagcbor.Decode(nb, buf); err != nil {
			return fmt.Errorf("failed to decode block data: %w", err)
		}
		node := nb.Build()

		// Check if the block is already pinned
		_, pinned, err := s.pinner.IsPinned(ctx, c)
		if err != nil {
			return fmt.Errorf("failed to check pin state: %w", err)
		}
		if pinned {
			return nil // Already pinned, no further action
		}

		wrappedNode := wrapIPLDPrimeNode(node, c)
		err = s.pinner.Pin(ctx, wrappedNode, false)
		if err != nil {
			return fmt.Errorf("failed to pin block: %w", err)
		}

		// Flush the pin state
		err = s.pinner.Flush(ctx)
		if err != nil {
			return fmt.Errorf("failed to flush pin state: %w", err)
		}
	}

	return nil
}

// Get retrieves data from a block in IPFS.
func (s *IPFSBlockStorage) Get(key string) ([]byte, error) {
	c, err := cid.Decode(key)
	if err != nil {
		return nil, fmt.Errorf("invalid CID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Retrieve the block from the blockservice
	block, err := s.blocksvc.GetBlock(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block: %w", err)
	}

	return block.RawData(), nil
}

// Delete removes a block from the blockstore.
func (s *IPFSBlockStorage) Delete(key string) error {
	c, err := cid.Decode(key)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Remove the block from the blockstore
	err = s.blockstore.DeleteBlock(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to delete block: %w", err)
	}

	// Optionally unpin the block
	if s.pin {
		err = s.pinner.Unpin(ctx, c, false)
		if err != nil && !errors.Is(err, pinner.ErrNotPinned) {
			return fmt.Errorf("failed to unpin block: %w", err)
		}

		// Flush the pin state
		err = s.pinner.Flush(ctx)
		if err != nil {
			return fmt.Errorf("failed to flush pin state: %w", err)
		}
	}

	return nil
}

// Iterator is not supported for IPFSBlockStorage
func (s *IPFSBlockStorage) Iterator() (<-chan [2]string, error) {
	return nil, errors.New("iterator not implemented for IPFSBlockStorage")
}

// Merge merges data from another storage instance
func (s *IPFSBlockStorage) Merge(other Storage) error {
	return errors.New("merge not implemented for IPFSBlockStorage")
}

// Clear removes all blocks (not implemented as IPFS typically handles this globally)
func (s *IPFSBlockStorage) Clear() error {
	return errors.New("clear not implemented for IPFSBlockStorage")
}

// Close releases resources used by the storage
func (s *IPFSBlockStorage) Close() error {
	// No resources to release in this implementation
	return nil
}
