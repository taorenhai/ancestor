package cache

import (
	"errors"
	"net"
	"sync"

	"github.com/taorenhai/ancestor/meta"
)

var (
	// ErrNotFound not found information in cache.
	ErrNotFound = errors.New("not found in cache")
)

// NodesCache store node's id and ip address
type NodesCache struct {
	sync.Mutex
	nodeDescs map[meta.NodeID]meta.NodeDescriptor
}

// NewNodesCache create new NodesCache object
func NewNodesCache() *NodesCache {
	return &NodesCache{
		nodeDescs: make(map[meta.NodeID]meta.NodeDescriptor),
	}
}

// IsExistNodeCache check the exist of node cache
func (nc *NodesCache) IsExistNodeCache(nodeID meta.NodeID) bool {
	if _, ok := nc.nodeDescs[nodeID]; ok {
		return true
	}
	return false
}

// GetNodeAddress return node's IP:Port string
func (nc *NodesCache) GetNodeAddress(nodeID meta.NodeID) (net.Addr, error) {
	nc.Lock()
	nodeDesc, ok := nc.nodeDescs[nodeID]
	nc.Unlock()

	if !ok {
		return nil, ErrNotFound
	}

	addr, err := net.ResolveTCPAddr("tcp", nodeDesc.Address)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// AddNodes add node Desctiptors to local cache
func (nc *NodesCache) AddNodes(nodeDescs []meta.NodeDescriptor) {
	nc.Lock()
	defer nc.Unlock()
	for _, nodeDesc := range nodeDescs {
		nc.nodeDescs[nodeDesc.NodeID] = nodeDesc
	}
}

// ClearNodes clear local cache
func (nc *NodesCache) ClearNodes() {
	nc.Lock()
	defer nc.Unlock()
	nc.nodeDescs = make(map[meta.NodeID]meta.NodeDescriptor)
}
