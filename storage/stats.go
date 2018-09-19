package storage

import (
	"fmt"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
)

var (
	statsUploadInterval = time.Second * 30
)

func (s *store) loadRangeStats() (map[meta.RangeID]*meta.RangeStats, error) {
	rsmap := make(map[meta.RangeID]*meta.RangeStats)
	key := makeRangeStatsKey(s.nodeID)

	val, err := s.engine.Get(key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if val == nil {
		return rsmap, nil
	}

	nsi := &meta.NodeStatsInfo{}
	if err := nsi.Unmarshal(val); err != nil {
		return nil, errors.Trace(err)
	}

	for id, rsi := range nsi.RangeStatsInfo {
		rsmap[id] = &meta.RangeStats{RangeStatsInfo: rsi}
	}

	return rsmap, nil
}

func (s *store) uploadRangeStats() {
	s.stopper.RunWorker(func() {
		timer := time.Tick(statsUploadInterval)
		for {
			select {
			case <-timer:
				stats := s.newNodeStats()
				if len(stats.RangeStatsInfo) > 0 {
					if err := s.storeNodeStats(stats); err != nil {
						log.Errorf("store node:%v stats err:%v", s.nodeID, err)
					}

					if err := s.client.SetNodeStats(*stats); err != nil {
						log.Errorf("node:%v upload range stats err:%v", s.nodeID, err)
					}
				}

			case <-s.stopper.ShouldStop():
				return
			}
		}
	})
}

func (s *store) newNodeStats() *meta.NodeStatsInfo {
	nsi := &meta.NodeStatsInfo{
		NodeID:         s.nodeID,
		RangeStatsInfo: make(map[meta.RangeID]*meta.RangeStatsInfo),
	}

	for id, rs := range s.rangeStats {
		rs.Lock()
		var isLeader bool
		if !rs.RangeStatsInfo.IsRemoved {
			r, err := s.getReplica(id)
			if err != nil {
				rs.Unlock()
				continue
			}

			_, isLeader = r.getRaftGroupLeader()
			if isLeader {
				nsi.LeaderCount++
			}
		}

		nsi.TotalBytes += rs.RangeStatsInfo.TotalBytes
		nsi.TotalCount += rs.RangeStatsInfo.TotalCount
		rs.RangeStatsInfo.IsRaftLeader = isLeader
		nsi.RangeStatsInfo[id] = rs.RangeStatsInfo
		rs.Unlock()
	}

	return nsi
}

func (s *store) storeNodeStats(nsi *meta.NodeStatsInfo) error {
	key := makeRangeStatsKey(s.nodeID)
	val, err := nsi.Marshal()
	if err != nil {
		return fmt.Errorf("node:%v stats marshal error:%v", s.nodeID, err)
	}
	if err := s.engine.Put(key, val); err != nil {
		return fmt.Errorf("node:%v, put stats local error:%v", s.nodeID, err)
	}

	return nil
}

func makeRangeStatsKey(id meta.NodeID) meta.MVCCKey {
	return meta.MVCCKey(fmt.Sprintf("%s_%d", util.RangeStatsKeyPrefix, id))
}
