package cache

import (
	"sync"

	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
)

// RangeDescriptorCache is the cache define for range descriptor
type RangeDescriptorCache struct {
	minReplica int
	unstable   map[meta.RangeID]*meta.RangeDescriptor
	ranges     map[meta.RangeID]*meta.RangeDescriptor
	cache      *OrderedCache
	sync.RWMutex
}

// NewRangeDescriptorCache range descriptor cache.
func NewRangeDescriptorCache(minReplica int) *RangeDescriptorCache {
	return &RangeDescriptorCache{
		minReplica: minReplica,
		unstable:   make(map[meta.RangeID]*meta.RangeDescriptor),
		ranges:     make(map[meta.RangeID]*meta.RangeDescriptor),
		cache:      NewOrderedCache(Config{Policy: CacheNone}),
	}
}

// Get get RangeDescriptor by rangeID.
func (r *RangeDescriptorCache) Get(id meta.RangeID) (*meta.RangeDescriptor, error) {
	r.RLock()
	if rd, ok := r.ranges[id]; ok {
		r.RUnlock()
		return rd, nil
	}
	r.RUnlock()
	return nil, ErrNotFound

}

// Unstable return Unstable rangeDescriptors.
func (r *RangeDescriptorCache) Unstable() []*meta.RangeDescriptor {
	var rds []*meta.RangeDescriptor
	r.RLock()
	for _, rd := range r.unstable {
		rds = append(rds, rd)
	}
	r.RUnlock()
	return rds
}

// Add used to add cache data
func (r *RangeDescriptorCache) Add(rds ...meta.RangeDescriptor) {
	r.Lock()
	for _, rd := range rds {
		key := rd.StartKey
		val := rd
		if _, ok := r.cache.Get(key); ok {
			r.cache.Del(key)
			delete(r.ranges, rd.RangeID)
			delete(r.unstable, rd.RangeID)
		}

		r.cache.Add(key, &val)
		r.ranges[rd.RangeID] = &val

		if len(rd.Replicas) < r.minReplica {
			r.unstable[rd.RangeID] = &val
		}
	}
	r.Unlock()
}

// Lookup used to locate a descriptor for the range containing the given Key.
func (r *RangeDescriptorCache) Lookup(key meta.Key) (*meta.RangeDescriptor, error) {
	r.RLock()
	defer r.RUnlock()

	_, v, ok := r.cache.Floor(key)
	if !ok {
		return nil, errors.Trace(ErrNotFound)
	}

	rd := v.(*meta.RangeDescriptor)

	// Check that key actually belongs to the range.
	if !(key.Compare(rd.StartKey) >= 0 && key.Compare(rd.EndKey) < 0) {
		return nil, errors.Annotatef(ErrNotFound, "key:%q", key)
	}

	return rd, nil
}

// Iterate return all date from range descriptor cache.
func (r *RangeDescriptorCache) Iterate(f func(k, v interface{})) {
	r.RLock()
	r.cache.Do(f)
	r.RUnlock()
}

// AdjustLeader Adjust replicas order.
func (r *RangeDescriptorCache) AdjustLeader(rangeID meta.RangeID, nodeID meta.NodeID) {
	r.RLock()
	rd, ok := r.ranges[rangeID]
	if !ok {
		r.Unlock()
		return
	}
	r.RUnlock()

	r.Lock()
	for i, rp := range rd.Replicas {
		if rp.NodeID == nodeID {
			if i != 0 {
				tmp := rd.Replicas[0]
				rd.Replicas[0] = rd.Replicas[i]
				rd.Replicas[i] = tmp
			}
			break
		}
	}
	r.Unlock()
}
