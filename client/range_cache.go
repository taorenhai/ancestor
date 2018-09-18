package client

import (
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/biogo/store/interval"
	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/cache"
)

var (
	rangeCache *rangeDescriptorCache

	errRangeCacheNotFound = errors.New("not found rangeDescriptor in cache")
)

func init() {
	rangeCache = newrangeDescriptorCache()
}

type rangeDesc struct {
	leaderIndex int32
	meta.RangeDescriptor
}

type rangeDescriptorCache struct {
	*cache.IntervalCache
	sync.RWMutex
}

func newrangeDescriptorCache() *rangeDescriptorCache {
	return &rangeDescriptorCache{
		IntervalCache: cache.NewIntervalCache(cache.Config{Policy: cache.CacheNone}),
	}
}

type rangeCompareKey []byte

// Compare for rangeCompareKey
func (rck rangeCompareKey) Compare(b interval.Comparable) int {
	if len(b.(rangeCompareKey)) == 0 {
		return -1
	}

	if len(rck) == 0 {
		return 1
	}

	return bytes.Compare(rck, b.(rangeCompareKey))
}

func (rdc *rangeDescriptorCache) add(rd meta.RangeDescriptor) {
	rdc.Lock()
	defer rdc.Unlock()

	key := rdc.NewKey(rangeCompareKey(rd.StartKey), rangeCompareKey(rd.EndKey))
	val := &rangeDesc{RangeDescriptor: rd}
	rdc.Add(key, val)
}

func (rdc *rangeDescriptorCache) lookup(key meta.Key) (*rangeDesc, error) {
	b := rangeCompareKey(key)
	e := rangeCompareKey(key.Next())

	rdc.RLock()
	defer rdc.RUnlock()

	for _, r := range rdc.GetOverlaps(b, e) {
		d, _ := r.Value.(*rangeDesc)
		return d, nil
	}
	return nil, errRangeCacheNotFound
}

func (rdc *rangeDescriptorCache) clear() {
	rdc.Lock()
	rdc.Clear()
	rdc.Unlock()
}

func (rd *rangeDesc) getLeaderNodeID() (meta.NodeID, int) {
	index := int(atomic.LoadInt32(&rd.leaderIndex))
	return rd.GetReplicas()[index].NodeID, index
}

func (rd *rangeDesc) getNextNodeID(oldIndex int) (meta.NodeID, int) {
	index := (oldIndex + 1) % len(rd.GetReplicas())
	return rd.GetReplicas()[index].NodeID, index
}

func (rd *rangeDesc) setLeaderNodeIdx(index int) {
	atomic.StoreInt32(&rd.leaderIndex, int32(index))
}

func (rd *rangeDesc) setLeaderNodeID(nodeID meta.NodeID) int {
	for i, r := range rd.GetReplicas() {
		if r.NodeID == nodeID {
			atomic.StoreInt32(&rd.leaderIndex, int32(i))
			return i
		}
	}
	return 0
}
