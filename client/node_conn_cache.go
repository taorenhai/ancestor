package client

import (
	"sync"

	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
)

// nodeConnCache is a connect cache for nodeAPI
type nodeConnCache struct {
	pool   map[meta.NodeID]*nodeAPI
	isStop bool
	sync.Mutex
}

// newNodeConnCache use to create a new nodeConnCache object
func newNodeConnCache() *nodeConnCache {
	return &nodeConnCache{
		pool: make(map[meta.NodeID]*nodeAPI),
	}
}

type getSenderFunc func(meta.NodeID) (Sender, error)

func (ncc *nodeConnCache) get(nodeID meta.NodeID, getSender getSenderFunc) (*nodeAPI, error) {
	ncc.Lock()
	defer ncc.Unlock()

	if ncc.isStop {
		return nil, errors.Trace(ErrInvalidState)
	}

	if node, ok := ncc.pool[nodeID]; ok {
		return node, nil
	}

	sender, err := getSender(nodeID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	node := newNodeAPI(sender, nodeID)
	ncc.pool[nodeID] = node
	return node, nil
}

func (ncc *nodeConnCache) stop() {
	ncc.Lock()
	defer ncc.Unlock()

	ncc.isStop = true

	for key, node := range ncc.pool {
		node.stop()
		delete(ncc.pool, key)
	}
}
