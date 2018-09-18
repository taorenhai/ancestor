package client

import (
	"net"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/retry"
)

type base struct {
	hosts       []string
	connCache   *nodeConnCache
	pd          atomic.Value
	etcd        *etcdAPI
	nodesCache  *cache.NodesCache
	isStop      bool
	localNode   meta.NodeID
	localSender Sender

	*admin
}

func newBase(hosts []string) *base {
	log.Debugf("etcdAddrs:%v", hosts)
	return &base{hosts: hosts, isStop: true}
}

func (b *base) start(a *admin) error {
	etcd, err := newEtcdAPI(b.hosts)
	if err != nil {
		return errors.Trace(err)
	}

	addr, err := etcd.getMasterAddr()
	if err != nil {
		return errors.Trace(err)
	}

	b.pd.Store(newPDAPI(newRPCSender(addr, defaultRetryOptions)))
	b.nodesCache = cache.NewNodesCache()
	b.connCache = newNodeConnCache()
	b.admin = a
	b.etcd = etcd
	b.isStop = false

	return nil
}

// getRangeDescFromCache get RangeDescriptor from cache, when err is errRangeCacheNotFound, call getRangeDescFromMaster.
func (b *base) getRangeDescFromCache(key meta.Key) (*rangeDesc, error) {
	rd, err := rangeCache.lookup(key)
	if err == errRangeCacheNotFound {
		rds, err := b.GetRangeDescriptors(key)
		if err != nil {
			return nil, errors.Trace(err)
		}

		rangeCache.add(rds[0])

		return rangeCache.lookup(key)
	}

	return rd, nil
}

// getNodeAddrFromCache get addr from cache, if cache miss, reload nodeList from pd.
func (b *base) getNodeAddrFromCache(nodeID meta.NodeID) (net.Addr, error) {
	addr, err := b.nodesCache.GetNodeAddress(nodeID)
	if err == cache.ErrNotFound {
		var nodes []meta.NodeDescriptor
		nodes, err = b.GetNodes()
		if err != nil {
			return nil, errors.Trace(err)
		}
		b.nodesCache.ClearNodes()
		b.nodesCache.AddNodes(nodes)
		addr, err = b.nodesCache.GetNodeAddress(nodeID)
	}

	if err != nil {
		return nil, errors.Trace(err)
	}

	return addr, nil
}

// getNodeConn for nodeConnCache interface.
func (b *base) getNodeConn(nodeID meta.NodeID) (*nodeAPI, error) {
	return b.connCache.get(nodeID, func(nodeID meta.NodeID) (Sender, error) {
		log.Debugf("localNode:%d, nodeID:%d", b.localNode, nodeID)
		if nodeID == b.localNode {
			return b.localSender, nil
		}
		addr, err := b.getNodeAddrFromCache(nodeID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return newRPCSender(addr, defaultRetryOptions), nil
	})
}

func (b *base) doNodeRequest(key meta.Key, call func(*nodeAPI, meta.RangeDescriptor) error) error {
	rd, err := b.getRangeDescFromCache(key)
	if err != nil {
		return errors.Trace(err)
	}
	id, i := rd.getLeaderNodeID()

	for r := retry.Start(defaultRetryOptions); r.Next(); {
		nc, err := b.getNodeConn(id)
		if err != nil {
			return errors.Trace(err)
		}

		if err = call(nc, rd.RangeDescriptor); err == nil {
			rd.setLeaderNodeIdx(i)
			return nil
		}

		if b.isStop {
			return ErrInvalidState
		}

		switch e := errors.Cause(err).(type) {
		case *meta.ReadPriorityError:
			continue

		case *meta.RPCConnectError:
			id, i = rd.getNextNodeID(i)

		case *meta.NotLeaderError:
			id = e.NodeID
			if id <= 0 {
				id, i = rd.getNextNodeID(i)
			} else {
				i = rd.setLeaderNodeID(id)
			}

		case *meta.RangeNotFoundError:
			rangeCache.clear()
			if rd, err = b.getRangeDescFromCache(key); err != nil {
				return errors.Trace(err)
			}
			id, i = rd.getLeaderNodeID()

		default:
			return errors.Trace(err)
		}
	}

	return meta.NewNeedRetryError("doNodeRequest retry error")
}

func (b *base) doPDRequest(call func(*pdAPI) error) error {
	var err error
	for r := retry.Start(defaultRetryOptions); r.Next(); {
		pd := b.pd.Load().(*pdAPI)
		if err = call(pd); err == nil {
			return nil
		}

		if b.isStop {
			return ErrInvalidState
		}

		switch errors.Cause(err).(type) {
		case *meta.NotLeaderError, *meta.RPCConnectError:
			pd.stop()
			addr, err1 := b.etcd.getMasterAddr()
			if err1 != nil {
				return errors.Trace(err1)
			}
			log.Debugf("new pd master %s", addr.String())
			b.pd.Store(newPDAPI(newRPCSender(addr, defaultRetryOptions)))
			continue
		}

		return err
	}

	return meta.NewNeedRetryError("doPDRequest retry error")
}

// stop close node connCache, pdClient, etcdClient.
func (b *base) stop() {
	b.connCache.stop()
	pd := b.pd.Load().(*pdAPI)
	pd.stop()
	b.etcd.stop()
	b.isStop = true
}
