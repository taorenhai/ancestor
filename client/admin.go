package client

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
)

// admin implement Admin interface.
type admin struct {
	localNodeID meta.NodeID
	localSender Sender
	*base
}

func newAdmin(b *base) *admin {
	return &admin{base: b}
}

// GetTimestamp get timestamp froa pd.
func (a *admin) GetTimestamp() (meta.Timestamp, error) {
	if a.isStop {
		return meta.Timestamp{}, ErrInvalidState
	}

	var err error
	var ts meta.Timestamp

	return ts, a.doPDRequest(func(pd *pdAPI) error {
		ts, err = pd.getTimestamp()
		return err
	})
}

// GetRangeDescriptors get RangeDescriptors froa pd.
func (a *admin) GetRangeDescriptors(keys ...meta.Key) ([]meta.RangeDescriptor, error) {
	if a.isStop {
		return nil, ErrInvalidState
	}
	var err error
	var rds []meta.RangeDescriptor

	return rds, a.doPDRequest(func(pd *pdAPI) error {
		rds, err = pd.getRangeDescriptors(keys...)
		return err
	})
}

// SetRangeDescriptors set RangeDescriptors to pd.
func (a *admin) SetRangeDescriptors(rds []meta.RangeDescriptor) error {
	if a.isStop {
		return ErrInvalidState
	}

	return a.doPDRequest(func(pd *pdAPI) error {
		return pd.setRangeDescriptors(rds)
	})
}

// GetNodes get all nodeDesc for localAPI.
func (a *admin) GetNodes() ([]meta.NodeDescriptor, error) {
	if a.isStop {
		return nil, ErrInvalidState
	}
	var err error
	nodeDescList := []meta.NodeDescriptor{}

	return nodeDescList, a.doPDRequest(func(pd *pdAPI) error {
		nodeDescList, err = pd.getNodeDescriptors()
		return err
	})
}

// SetNodes set nodeDesc for localAPI.
func (a *admin) SetNodes(nds []meta.NodeDescriptor) error {
	if a.isStop {
		return ErrInvalidState
	}
	return a.doPDRequest(func(pd *pdAPI) error {
		return pd.setNodeDescriptors(nds)
	})
}

// PushTxn push transaction.
func (a *admin) PushTxn(pushType meta.PushTxnType, key meta.Key, txn *meta.Transaction) (*meta.Transaction, error) {
	if a.isStop {
		return nil, ErrInvalidState
	}
	var result *meta.Transaction

	return result, a.doNodeRequest(key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
		ts, err := a.GetTimestamp()
		if err != nil {
			log.Errorf("%s getTimestamp error:%s", txn.Name, errors.ErrorStack(err))
			return errors.Trace(err)
		}
		result, err = node.pushTxn(pushType, rd.RangeID, key, txn, ts)
		return errors.Trace(err)
	})
}

// ResolveIntent clear key's meta data or change timestamp.
func (a *admin) ResolveIntent(key meta.Key, txn *meta.Transaction) error {
	if a.isStop {
		return ErrInvalidState
	}
	return a.doNodeRequest(key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
		return node.resolveIntent(txn, rd.RangeID, []meta.Key{key})
	})
}

// SetNodeStats for local client.
func (a *admin) SetNodeStats(stats meta.NodeStatsInfo) error {
	if a.isStop {
		return ErrInvalidState
	}

	return a.doPDRequest(func(pd *pdAPI) error {
		return pd.setNodeStats(stats)
	})
}

// SetRangeStats for local client.
func (a *admin) SetRangeStats(stats []meta.RangeStatsInfo) error {
	if a.isStop {
		return ErrInvalidState
	}

	return a.doPDRequest(func(pd *pdAPI) error {
		return pd.setRangeStats(stats)
	})
}

// GetRangeStats for local client.
func (a *admin) GetRangeStats(ids []meta.RangeID) ([]meta.RangeStatsInfo, error) {
	if a.isStop {
		return nil, ErrInvalidState
	}

	var stats []meta.RangeStatsInfo
	var err error

	return stats, a.doPDRequest(func(pd *pdAPI) error {
		stats, err = pd.getRangeStats(ids)
		return err
	})
}

// GetNewID for local client.
func (a *admin) GetNewID() (meta.RangeID, error) {
	var id meta.RangeID
	var err error

	if a.isStop {
		return id, ErrInvalidState
	}

	return id, a.doPDRequest(func(pd *pdAPI) error {
		id, err = pd.getNewID()
		return err
	})
}

// GetRangeDescriptor get rangeDesc by key for local client.
func (a *admin) GetRangeDescriptor(key meta.Key) (meta.RangeDescriptor, error) {
	rd, err := a.getRangeDescFromCache(key)
	if err != nil {
		return meta.RangeDescriptor{}, errors.Trace(err)
	}

	return rd.RangeDescriptor, nil
}

// SetLocalServer set local node server for local client
func (a *admin) SetLocalServer(id meta.NodeID, nodeServer meta.Executer) {
	a.localNodeID = id
	a.localSender = newLocalSender(id, nodeServer)
}

// Bootstrap
func (a *admin) Bootstrap(addr string, capacity int64) (meta.NodeDescriptor, error) {
	var node meta.NodeDescriptor
	var err error

	return node, a.doPDRequest(func(pd *pdAPI) error {
		node, err = pd.bootstrap(addr, capacity)
		return err
	})
}

// CreateReplica create a replica in the specified node.
func (a *admin) CreateReplica(id meta.NodeID, rd meta.RangeDescriptor) error {
	key := meta.NewKey(util.SystemPrefix, []byte(fmt.Sprintf("%d", id)))

	return a.doNodeRequest(key, func(node *nodeAPI, position meta.RangeDescriptor) error {
		return node.createReplica(position, rd)
	})

}
