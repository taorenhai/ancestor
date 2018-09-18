package pd

import (
	"encoding/json"
	"fmt"
	"path"
	"sync"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/cache"
)

type node struct {
	meta.NodeDescriptor
	meta.NodeStatsInfo
}

type cluster struct {
	prefix string
	nodes  map[meta.NodeID]*node
	hosts  map[string]*node
	sync.RWMutex
	master
}

func newCluster(m master) *cluster {
	return &cluster{
		nodes:  make(map[meta.NodeID]*node),
		hosts:  make(map[string]*node),
		master: m,
		prefix: path.Join(util.ETCDRootPath, "NodeID"),
	}
}

func (c *cluster) count() int {
	c.RLock()
	sum := len(c.nodes)
	c.RUnlock()
	return sum
}

func (c *cluster) getIdleNode(ids ...meta.NodeID) (meta.NodeDescriptor, error) {
	c.RLock()

	for _, n := range c.nodes {
		exist := false
		for _, id := range ids {
			if n.NodeDescriptor.NodeID == id {
				exist = true
				break
			}
		}
		if !exist {
			c.RUnlock()
			return n.NodeDescriptor, nil
		}
	}

	c.RUnlock()

	return meta.NodeDescriptor{}, errors.Trace(cache.ErrNotFound)
}

func (c *cluster) getNodes(ids ...meta.NodeID) []*meta.NodeDescriptor {
	var nds []*meta.NodeDescriptor

	if len(ids) == 0 {
		c.RLock()
		for _, nd := range c.nodes {
			nds = append(nds, &nd.NodeDescriptor)
		}
		c.RUnlock()
		return nds
	}

	for _, id := range ids {
		c.RLock()
		if nd, ok := c.nodes[id]; ok {
			nds = append(nds, &nd.NodeDescriptor)
		}
		c.RUnlock()
	}

	return nds
}

func (c *cluster) getNodeByAddr(addr string) (meta.NodeDescriptor, error) {
	c.RLock()
	if n, ok := c.hosts[addr]; ok {
		c.RUnlock()
		return n.NodeDescriptor, nil
	}
	c.RUnlock()
	return meta.NodeDescriptor{}, errors.Trace(cache.ErrNotFound)
}

func (c *cluster) newNodeDescriptor(addr string, cap int64) (meta.NodeDescriptor, error) {
	nd := meta.NodeDescriptor{Address: addr, Capacity: cap}

	id, err := c.newID()
	if err != nil {
		return nd, errors.Trace(err)
	}
	nd.NodeID = meta.NodeID(id)
	log.Debugf("new node:(%d) addr:%s, cap:%d", id, addr, cap)

	return nd, c.setNodes(nd)
}

// newNodeKey get nodeDescriptors path in etcd.
func (c *cluster) newNodeKey(id meta.NodeID) string {
	if id == 0 {
		return c.prefix
	}

	return fmt.Sprintf("%s_%d", c.prefix, id)
}

func (c *cluster) setNodes(nds ...meta.NodeDescriptor) error {
	var ops []clientv3.Op
	for _, nd := range nds {
		data, err := json.Marshal(nd)
		if err != nil {
			return errors.Trace(err)
		}
		ops = append(ops, clientv3.OpPut(c.newNodeKey(nd.NodeID), string(data)))
	}

	if err := c.masterPut(ops...); err != nil {
		return errors.Trace(err)
	}

	c.addNodeCache(nds...)

	return nil
}

func (c *cluster) setNodeStats(ns meta.NodeStatsInfo) {
	c.Lock()

	if n, ok := c.nodes[ns.NodeID]; ok {
		n.NodeStatsInfo = ns
		c.Unlock()
		return
	}
	c.Unlock()

	log.Errorf("node:(%d) not found", ns.NodeID)
}

func (c *cluster) getNodeStats(ids ...meta.NodeID) []*meta.NodeStatsInfo {
	var ns []*meta.NodeStatsInfo

	if len(ids) == 0 {
		c.RLock()
		for _, n := range c.nodes {
			ns = append(ns, &n.NodeStatsInfo)
		}
		c.RUnlock()
		return ns
	}

	c.RLock()
	for _, id := range ids {
		if n, ok := c.nodes[id]; ok {
			ns = append(ns, &n.NodeStatsInfo)
		}
	}
	c.RUnlock()

	return ns
}

func (c *cluster) addNodeCache(nds ...meta.NodeDescriptor) {
	c.Lock()
	for _, nd := range nds {
		n, ok := c.nodes[nd.NodeID]
		if ok {
			n.NodeDescriptor = nd
		}
		n = &node{NodeDescriptor: nd}

		c.nodes[nd.NodeID] = n
		c.hosts[nd.Address] = n
	}
	c.Unlock()
}

func (c *cluster) delNodeCache(nds ...meta.NodeDescriptor) {
	c.Lock()
	for _, nd := range nds {
		delete(c.nodes, nd.NodeID)
		delete(c.hosts, nd.Address)
	}
	c.Unlock()
}

func (c *cluster) loadNodes() error {
	kvs, err := c.getValues(c.newNodeKey(meta.NodeID(0)), clientv3.WithPrefix())
	if err != nil {
		return errors.Trace(err)
	}

	if kvs == nil {
		log.Infof("not found nodes")
		return nil
	}

	var nd meta.NodeDescriptor
	for _, kv := range kvs {
		if err := json.Unmarshal(kv.Value, &nd); err != nil {
			return errors.Trace(err)
		}

		c.addNodeCache(nd)
	}

	return nil
}

func (c *cluster) setRangeStats(rss ...meta.RangeStatsInfo) {
	c.Lock()
	for _, rs := range rss {
		n, ok := c.nodes[rs.NodeID]
		if !ok {
			log.Errorf("node:(%d) not found", rs.NodeID)
			continue
		}

		if n.RangeStatsInfo == nil {
			n.RangeStatsInfo = make(map[meta.RangeID]*meta.RangeStatsInfo)
		}

		stats := rs
		n.RangeStatsInfo[rs.RangeID] = &stats
	}

	c.Unlock()
}

func (c *cluster) getRangeStats(ids ...meta.RangeID) []*meta.RangeStatsInfo {
	var rs []*meta.RangeStatsInfo

	if len(ids) == 0 {
		c.RLock()
		for _, n := range c.nodes {
			for _, s := range n.RangeStatsInfo {
				rs = append(rs, s)
			}
		}
		c.RUnlock()
		return rs
	}

	c.RLock()
	for _, n := range c.nodes {
		for _, id := range ids {
			if s, ok := n.RangeStatsInfo[id]; ok {
				rs = append(rs, s)
			}
		}
	}
	c.RUnlock()

	return rs
}
