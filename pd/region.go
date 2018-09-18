package pd

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/cache"
)

type region struct {
	prefix string
	s      *Server
	ranges *cache.RangeDescriptorCache
}

func newRegion(s *Server) *region {
	return &region{
		prefix: path.Join(util.ETCDRootPath, "RangeID"),
		s:      s,
		ranges: cache.NewRangeDescriptorCache(minReplica),
	}
}

func (r *region) getDefaultRanges() ([]meta.RangeDescriptor, error) {
	var rds []meta.RangeDescriptor
	for i := 1; i < 3; i++ {
		id, e := r.s.idAllocator.newID()
		if e != nil {
			return nil, errors.Trace(e)
		}
		rds = append(rds, meta.RangeDescriptor{RangeID: meta.RangeID(id), StartKey: []byte{byte(i)}, EndKey: []byte{byte(i + 1)}})
	}
	return rds, nil
}

func (r *region) newRangeKey(id meta.RangeID) string {
	if id == 0 {
		return r.prefix
	}
	return fmt.Sprintf("%s_%d", r.prefix, id)
}

func (r *region) getRangeDescriptor(id meta.RangeID) (*meta.RangeDescriptor, error) {
	return r.ranges.Get(id)
}

func (r *region) unstable() []*meta.RangeDescriptor {
	return r.ranges.Unstable()
}

func (r *region) getRangeDescriptors(key meta.Key) ([]*meta.RangeDescriptor, error) {
	var rds []*meta.RangeDescriptor

	if len(key) != 0 {
		rd, err := r.ranges.Lookup(key)
		if err != nil {
			return nil, errors.Trace(err)
		}

		rds = append(rds, rd)
		return rds, nil
	}

	r.ranges.Iterate(func(k, v interface{}) {
		if rd, ok := v.(*meta.RangeDescriptor); ok {
			rds = append(rds, rd)
		}
	})

	return rds, nil
}

func (r *region) addSystemRangeDescriptor(id meta.NodeID) error {
	rd := meta.RangeDescriptor{
		RangeID:  meta.RangeID(id),
		StartKey: meta.NewKey(util.SystemPrefix, []byte(fmt.Sprintf("%d", id))),
		EndKey:   meta.NewKey(util.SystemPrefix, []byte(fmt.Sprintf("%d", id+1))),
		Replicas: []meta.ReplicaDescriptor{
			meta.ReplicaDescriptor{NodeID: id, ReplicaID: meta.ReplicaID(id)},
			meta.ReplicaDescriptor{NodeID: id, ReplicaID: meta.ReplicaID(id)},
			meta.ReplicaDescriptor{NodeID: id, ReplicaID: meta.ReplicaID(id)},
		},
	}

	log.Debugf("new system range %+v", rd)

	return r.setRangeDescriptors(rd)
}

func (r *region) newRangeDescriptor(id meta.RangeID, start, end meta.Key, replica ...meta.ReplicaDescriptor) meta.RangeDescriptor {
	rd := meta.RangeDescriptor{
		RangeID:  id,
		StartKey: start,
		EndKey:   end,
		Replicas: replica,
	}

	log.Debugf("new range %+v", rd)
	return rd
}

func (r *region) setRangeDescriptors(rds ...meta.RangeDescriptor) error {
	var ops []clientv3.Op

	for _, rd := range rds {
		data, err := json.Marshal(rd)
		if err != nil {
			return errors.Trace(err)
		}
		ops = append(ops, clientv3.OpPut(r.newRangeKey(rd.RangeID), string(data)))
	}

	if err := putValues(r.s.etcdClient, r.s.newMasterCmp(), ops...); err != nil {
		return errors.Trace(err)
	}

	r.ranges.Add(rds...)

	return nil
}

func (r *region) loadRangeDescriptors() error {
	var rds []meta.RangeDescriptor

	kvs, err := getValues(r.s.etcdClient, r.newRangeKey(meta.RangeID(0)), clientv3.WithPrefix())
	if err != nil {
		return errors.Trace(err)
	}

	// init range
	if kvs == nil {
		if rds, err = r.getDefaultRanges(); err != nil {
			return errors.Trace(err)
		}
		return r.setRangeDescriptors(rds...)
	}

	for _, kv := range kvs {
		rd := meta.RangeDescriptor{}
		if err := json.Unmarshal(kv.Value, &rd); err != nil {
			return errors.Trace(err)
		}

		rds = append(rds, rd)
	}

	r.ranges.Add(rds...)

	return nil
}
