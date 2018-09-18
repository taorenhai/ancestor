package pd

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"

	"github.com/taorenhai/ancestor/meta"
)

const (
	testNodeNum = 10
)

type clusterMaster struct {
	id int64
	kv map[string][]byte
	sync.Mutex
}

func (c *clusterMaster) put(cmp clientv3.Cmp, ops ...clientv3.Op) error {
	c.Lock()
	for _, op := range ops {
		k := reflect.ValueOf(op).FieldByName("key")
		v := reflect.ValueOf(op).FieldByName("val")
		c.kv[string(k.Bytes())] = v.Bytes()
	}
	c.Unlock()
	return nil
}

func (c *clusterMaster) masterPut(ops ...clientv3.Op) error {
	c.Lock()
	for _, op := range ops {
		k := reflect.ValueOf(op).FieldByName("key")
		v := reflect.ValueOf(op).FieldByName("val")
		c.kv[string(k.Bytes())] = v.Bytes()
	}
	c.Unlock()
	return nil
}

func (c *clusterMaster) getValues(key string, opts ...clientv3.OpOption) ([]*mvccpb.KeyValue, error) {
	var kvs []*mvccpb.KeyValue

	if len(opts) == 1 {
		for k, v := range c.kv {
			if strings.HasPrefix(k, key) {
				kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(k), Value: []byte(v)})
			}
		}

		return kvs, nil
	}

	if v, ok := c.kv[key]; ok {
		kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(key), Value: []byte(v)})
	}
	return kvs, nil
}

func (c *clusterMaster) newID() (int64, error) {
	c.Lock()
	c.id++
	id := c.id
	c.Unlock()

	return id, nil
}

func newClusterMaster() master {
	return &clusterMaster{
		kv: make(map[string][]byte),
	}
}

func TestNewNodes(t *testing.T) {
	c := newCluster(newClusterMaster())
	for i := 0; i < testNodeNum; i++ {
		_, err := c.newNodeDescriptor(fmt.Sprintf("127.0.0.%d", i), int64(i))
		if err != nil {
			t.Fatalf("newNodeDescriptor error:%s", err.Error())
		}
	}
}

func TestGetNodes(t *testing.T) {
	var nodes []meta.NodeDescriptor

	c := newCluster(newClusterMaster())

	for i := 0; i < testNodeNum; i++ {
		n, err := c.newNodeDescriptor(fmt.Sprintf("127.0.0.%d", i), int64(i))
		if err != nil {
			t.Fatalf("newNodeDescriptor error:%s", err.Error())
		}
		nodes = append(nodes, n)
	}

	c.setNodes(nodes...)

	for i, n := range nodes {
		ns := c.getNodes(n.NodeID)
		if len(ns) == 0 {
			t.Fatalf("not found node:(%d)", n.NodeID)
		}

		if ns[0].Address != n.Address || ns[0].NodeID != n.NodeID {
			t.Fatalf("found nodes:%+v, expect: %+v", ns, n)
		}

		nd, err := c.getNodeByAddr(nodes[i].Address)
		if err != nil {
			t.Fatalf("getNodeByAddr addr:%s error:%s", nodes[i].Address, err.Error())
		}

		if nd.NodeID != nodes[i].NodeID {
			t.Fatalf("getNodeByAddr addr:%s node:%+v, expect:%+v", nodes[i].Address, nd, nodes[i])
		}
	}

	n, err := c.getNodeByAddr("127.0.0.222")
	if err == nil {
		t.Fatalf("getNodeByAddr find node:%+v, expect: not found", n)
	}
}

func TestDelNodes(t *testing.T) {
	var nodes []meta.NodeDescriptor

	c := newCluster(newClusterMaster())

	for i := 0; i < testNodeNum; i++ {
		n, err := c.newNodeDescriptor(fmt.Sprintf("127.0.0.%d", i), int64(i))
		if err != nil {
			t.Fatalf("newNodeDescriptor error:%s", err.Error())
		}
		nodes = append(nodes, n)
	}

	c.setNodes(nodes...)

	if len(c.getNodes()) != testNodeNum {
		t.Fatalf("found nodes:%d, expect: %d", len(c.getNodes()), testNodeNum)
	}

	c.delNodeCache(nodes...)

	if len(c.getNodes()) != 0 {
		t.Fatalf("found nodes:%+v, expect: not found", c.getNodes())
	}

}

func TestLoadNodes(t *testing.T) {
	c := newCluster(newClusterMaster())
	nodes := []meta.NodeDescriptor{
		meta.NodeDescriptor{Address: "127.0.0.1", NodeID: 1},
		meta.NodeDescriptor{Address: "127.0.0.2", NodeID: 2},
		meta.NodeDescriptor{Address: "127.0.0.3", NodeID: 3},
		meta.NodeDescriptor{Address: "127.0.0.4", NodeID: 4},
		meta.NodeDescriptor{Address: "127.0.0.5", NodeID: 5},
		meta.NodeDescriptor{Address: "127.0.0.6", NodeID: 6},
	}

	c.setNodes(nodes...)

	c.delNodeCache(nodes...)

	if err := c.loadNodes(); err != nil {
		t.Fatalf("loadNodes error:%s", err.Error())
	}

	for _, n := range nodes {
		ns := c.getNodes(n.NodeID)
		if len(ns) == 0 {
			t.Fatalf("not found node:(%d)", n.NodeID)
		}

		if ns[0].Address != n.Address || ns[0].NodeID != n.NodeID {
			t.Fatalf("found nodes:%+v, expect: %+v", ns, n)
		}
	}
}

func TestRangeStats(t *testing.T) {
	c := newCluster(newClusterMaster())
	for i := 0; i < testNodeNum; i++ {
		n, err := c.newNodeDescriptor(fmt.Sprintf("127.0.0.%d", i), int64(i))
		if err != nil {
			t.Fatalf("newNodeDescriptor error:%s", err.Error())
		}
		c.setNodes(n)
	}

	rss := []meta.RangeStatsInfo{
		meta.RangeStatsInfo{NodeID: 1, RangeID: 4, TotalBytes: 100, TotalCount: 10},
		meta.RangeStatsInfo{NodeID: 2, RangeID: 4, TotalBytes: 100, TotalCount: 10},
		meta.RangeStatsInfo{NodeID: 3, RangeID: 4, TotalBytes: 100, TotalCount: 10},

		meta.RangeStatsInfo{NodeID: 100, RangeID: 4, TotalBytes: 100, TotalCount: 10},
	}

	c.setRangeStats(rss...)

	for i := range rss {
		ns := c.getNodeStats(meta.NodeID(i + 1))
		if len(ns) != 1 {
			t.Fatalf("getNodeStats node:(%d) error, size:%d", i, len(ns))
		}
		fmt.Printf("node stats:%+v\n", ns)
	}

	rs := c.getRangeStats(meta.RangeID(4))
	if len(rs) != 3 {
		t.Fatalf("getRangeStats range:(4) error, size:%d", len(rs))
	}

	fmt.Printf("range stats:%+v\n", rs)

	nss := c.getRangeStats()
	if len(nss) != 3 {
		t.Fatalf("getRangeStats ranges error, size:%d", len(nss))
	}
}

func TestNodeStats(t *testing.T) {
	c := newCluster(newClusterMaster())
	for i := 0; i < testNodeNum; i++ {
		n, err := c.newNodeDescriptor(fmt.Sprintf("127.0.0.%d", i), int64(i))
		if err != nil {
			t.Fatalf("newNodeDescriptor error:%s", err.Error())
		}
		c.setNodes(n)
	}

	ns := meta.NodeStatsInfo{
		LeaderCount:    1,
		NodeID:         1,
		RangeStatsInfo: map[meta.RangeID]*meta.RangeStatsInfo{1: &meta.RangeStatsInfo{NodeID: 1, RangeID: 4, TotalBytes: 100, TotalCount: 10}},
	}

	c.setNodeStats(ns)
	ns.NodeID = 100
	c.setNodeStats(ns)

	nds := c.getNodes(1)
	if len(nds) != 1 {
		t.Fatalf("getNodes node 1 , find node size:%d", len(nds))
	}

	nss := c.getNodeStats()
	if len(nss) != testNodeNum {
		t.Fatalf("getNodes nodes , find node size:%d", len(nss))
	}

}
