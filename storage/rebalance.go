package storage

import (
	"fmt"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util/retry"
)

const (
	idLen           = 8
	prefixLen       = 1 + idLen /*tableID*/ + 2
	recordRowKeyLen = prefixLen + idLen /*handle*/

	raftInitialBackOff = 50
	raftMaxBackOff     = 8
	raftMultiplier     = 2
	raftMaxRetries     = 20
)

var (
	retryOptions = retry.Options{
		InitialBackoff: raftInitialBackOff * time.Millisecond,
		MaxBackoff:     raftMaxBackOff * time.Second,
		Multiplier:     raftMultiplier,
		MaxRetries:     raftMaxRetries,
	}

	errInvalidRange  = errors.New("request wrong range")
	errExistNode     = errors.New("already exist node")
	errNotFoundNode  = errors.New("not found node")
	errNoLeader      = errors.New("not found leader")
	errNoSplitKey    = errors.New("not found split key")
	errNoRangeStats  = errors.New("not found range stats")
	errRequestNode   = errors.New("request node error")
	errDecodeMVCCKey = errors.New("decode mvcckey error")
)

func (r *replica) getSplitKey(eng engine.Engine, req *meta.GetSplitKeyRequest, resp *meta.GetSplitKeyResponse) error {
	it := eng.NewIterator(false)
	defer it.Close()

	start := meta.NewMVCCKey(r.rangeDesc.StartKey)
	end := meta.NewMVCCKey(r.rangeDesc.EndKey)

	log.Debugf("[%d %d] getSplitKey range:[%q,%q] start:%q end:%q", r.store.nodeID, r.rangeID, r.rangeDesc.StartKey, r.rangeDesc.EndKey, start, end)

	for it.Seek(start); it.Valid() && it.Key().Less(end); it.Next() {
		resp.RangeBytes += int64(len(it.Key()) + len(it.Value()))
		resp.RangeCount++

		if resp.RangeBytes >= req.SplitSize {
			key, _, _, err := it.Key().Decode()
			if err != nil {
				log.Errorf("Decode mvccKey error:%s, key:%q", err.Error(), it.Key())
				return errors.Trace(err)
			}

			if key.Compare(r.rangeDesc.StartKey) <= 0 {
				log.Debugf("split key(%q) must be more then start(%q)", key, r.rangeDesc.StartKey)
				continue
			}

			resp.SplitKey = key
			log.Debugf("[%d %d] getSplitKey start:%q end:%q, find split key:%q", r.store.nodeID, r.rangeID, start, end, key)
			return nil
		}
	}

	log.Errorf("[%d %d] getSplitKey range start:%q, end:%q not found split key", r.store.nodeID, r.rangeID, r.rangeDesc.StartKey, r.rangeDesc.EndKey)

	return errNoSplitKey
}

// split splits oldRangeID[startKey, endKey) with newRangeID and splitKey into two parts,
// one is newRangeID[startKey, splitKey), another is oldRangeID[splitKey, endKey).
// create new raft group with newRangeID, then update local range descriptor and update pd server.
func (r *replica) split(req *meta.SplitRequest, resp *meta.SplitResponse) error {
	oldID := r.rangeID
	newID := req.NewRangeID

	if oldID != req.RangeID || !r.validateKeyRange(req.SplitKey) {
		log.Errorf("range:%v split, old:%v new:%v splitKey:%v desc[%v, %v]", r.rangeID, oldID,
			newID, req.SplitKey, r.rangeDesc.StartKey, r.rangeDesc.EndKey)
		return errInvalidRange
	}

	prevDesc := makeRangeDesc(newID, r.rangeDesc.StartKey, req.SplitKey, r.rangeDesc.Replicas)
	postDesc := makeRangeDesc(oldID, req.SplitKey, r.rangeDesc.EndKey, r.rangeDesc.Replicas)

	_, err := r.store.createReplica(prevDesc, false)
	if err != nil {
		return err
	}

	stats := fillRangeStats(r.store, oldID, newID, req.NewRangeBytes, req.NewRangeCount)
	if err := setRangeStats(r.store, stats); err != nil {
		log.Errorf("range:%v split setRangeStats error:%v", r.rangeID, err)
	}

	resp.RangeDescriptors = []meta.RangeDescriptor{prevDesc, postDesc}

	return nil
}

func (r *replica) addRaftConf(req *meta.AddRaftConfRequest, resp *meta.AddRaftConfResponse) error {
	if req.RangeID != r.rangeID {
		return errInvalidRange
	}
	if isExistNode(req.NodeID, r) {
		return errExistNode
	}

	done := make(chan error, 1)
	r.raftAddConfChan <- &raftAddConf{
		nodeID: uint64(req.NodeID) + uint64(r.rangeID),
		done:   done,
	}

	err := <-done
	resp.RangeDescriptor = r.rangeDesc

	return err
}

func (r *replica) deleteRaftConf(req *meta.DeleteRaftConfRequest, resp *meta.DeleteRaftConfResponse) error {
	if req.RangeID != r.rangeID {
		return errInvalidRange
	}

	done := make(chan error, 1)
	r.raftDeleteConfChan <- &raftDeleteConf{
		nodeID: uint64(req.NodeID) + uint64(r.rangeID),
		done:   done,
	}

	err := <-done
	resp.RangeDescriptor = r.rangeDesc

	return err
}

func (r *replica) updateRange(req *meta.UpdateRangeRequest) error {
	if r.rangeID != req.RangeID {
		return errInvalidRange
	}
	r.rangeDesc = req.RangeDescriptor

	return nil
}

func (r *replica) compactLog(req *meta.CompactLogRequest) error {
	if r.rangeID != req.RangeID {
		return errInvalidRange
	}

	return r.compact(r.raftAppliedIndex)
}

func (r *replica) transferLeader(req *meta.TransferLeaderRequest) error {
	if r.store.nodeID != req.From {
		return errRequestNode
	}

	to := uint64(r.rangeID) + uint64(req.To)
	r.raftGroup.TransferLeader(to)

	for t := retry.Start(retryOptions); t.Next(); {
		leader := r.raftGroup.Status().Lead
		if leader == to {
			return nil
		}
	}

	return fmt.Errorf("range:%v from:%v transfer leader to:%v error", r.rangeID, req.From, req.To)
}

func isExistNode(id meta.NodeID, r *replica) bool {
	for _, repl := range r.rangeDesc.Replicas {
		if repl.NodeID == id {
			return true
		}
	}

	return false
}

func makeRangeDesc(id meta.RangeID, skey, ekey meta.Key,
	replicas []meta.ReplicaDescriptor) meta.RangeDescriptor {
	return meta.RangeDescriptor{
		RangeID:  id,
		StartKey: skey,
		EndKey:   ekey,
		Replicas: replicas,
	}
}

func setRangeStats(s *store, rangeStats []meta.RangeStatsInfo) error {
	if len(rangeStats) == 0 {
		return errNoRangeStats
	}

	if err := s.client.SetRangeStats(rangeStats); err != nil {
		return err
	}

	return nil
}

func cleanRaftNode(s *store, id meta.RangeID) {
	r, err := s.getReplica(id)
	if err != nil {
		log.Errorf("range:%v get replica error:%v", id, err)
		return
	}

	s.deleteReplica(id)
	r.raftStopChan <- struct{}{}
}

func retryGetRaftGroupLeader(r *replica) (meta.NodeID, bool) {
	for t := retry.Start(retryOptions); t.Next(); {
		id, isLeader := r.getRaftGroupLeader()
		if id != meta.NodeID(0) {
			return id, isLeader
		}
	}

	return meta.NodeID(0), false
}

func fillRangeStats(s *store, oldID, newID meta.RangeID, newBytes, newCount int64) []meta.RangeStatsInfo {
	var stats []meta.RangeStatsInfo

	prevStats := meta.RangeStatsInfo{
		NodeID:     s.nodeID,
		RangeID:    newID,
		TotalBytes: newBytes,
		TotalCount: newCount,
		IsRemoved:  false,
	}
	s.rangeStats[newID].RangeStatsInfo.TotalBytes = newBytes
	s.rangeStats[newID].RangeStatsInfo.TotalCount = newCount

	s.rangeStats[oldID].Lock()
	s.rangeStats[oldID].RangeStatsInfo.TotalBytes -= newBytes
	s.rangeStats[oldID].RangeStatsInfo.TotalCount -= newCount

	postStats := meta.RangeStatsInfo{
		NodeID:     s.nodeID,
		RangeID:    oldID,
		TotalBytes: s.rangeStats[oldID].RangeStatsInfo.TotalBytes,
		TotalCount: s.rangeStats[oldID].RangeStatsInfo.TotalCount,
		IsRemoved:  false,
	}
	s.rangeStats[oldID].Unlock()

	stats = append(stats, prevStats)
	stats = append(stats, postStats)

	return stats
}
