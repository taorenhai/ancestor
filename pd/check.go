package pd

import (
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/client"
	"github.com/taorenhai/ancestor/meta"
)

const (
	maxRetryInterval = time.Minute
	minReplica       = 3
)

type delayRecord struct {
	timeout  time.Time
	interval time.Duration
}

func newDelayRecord() *delayRecord {
	return &delayRecord{interval: time.Second, timeout: time.Now().Add(time.Second)}
}
func (d *delayRecord) valid() bool {
	if time.Since(d.timeout) > 0 {
		return false
	}
	return true
}

func (d *delayRecord) next() {
	d.interval = d.interval * 2
	if d.interval > maxRetryInterval {
		d.interval = maxRetryInterval
	}
	d.timeout = time.Now().Add(d.interval)
}

func (s *Server) checkDelayRecord(rs *meta.RangeStatsInfo) bool {
	d, ok := s.delayRecord[rs.RangeID]
	if !ok {
		return false
	}

	if d.valid() {
		return true
	}

	if err := s.checkRangeSplit(rs); err != nil {
		d.next()
		log.Infof("checkRangeSplit error:%s", err.Error())
		return true
	}

	delete(s.delayRecord, rs.RangeID)
	return true
}

var (
	store client.Storage
)

func (s *Server) checkReplica() {
	var err error

	if s.cluster.count() < minReplica {
		log.Infof("current nodes count:%d, minReplica:%d", s.cluster.count(), minReplica)
		return
	}

	if store == nil {
		store, err = client.Open(strings.Join(s.cfg.EtcdHosts, ";"))
		if err != nil {
			log.Errorf(" client.Open(%s) error:%s", strings.Join(s.cfg.EtcdHosts, ";"), err.Error())
			return
		}
	}

	for _, rd := range s.region.unstable() {

		log.Debugf("unstable range:(%+v)", rd)

		var ids []meta.NodeID
		var reps []meta.ReplicaDescriptor

		for _, r := range rd.Replicas {
			ids = append(ids, r.NodeID)
		}

		for len(ids) < minReplica {
			n, err := s.cluster.getIdleNode(ids...)
			if err != nil {
				log.Errorf("getIdleNode error:%s", err.Error())
				return
			}
			ids = append(ids, n.NodeID)
			rID, err := s.newID()
			if err != nil {
				log.Errorf("newID error:%s", err.Error())
				return
			}
			reps = append(reps, meta.ReplicaDescriptor{NodeID: n.NodeID, ReplicaID: meta.ReplicaID(rID)})
		}

		rd.Replicas = append(rd.Replicas, reps...)

		for _, rep := range reps {
			if err := store.GetAdmin().CreateReplica(rep.NodeID, *rd); err != nil {
				log.Errorf("CreateReplica node:(%d), rd:%+v, error:%s", rep.NodeID, rd, err.Error())
				continue
			}
		}

		if err := s.region.setRangeDescriptors(*rd); err != nil {
			log.Errorf("setRangeDescriptors error:%s, rd:%+v", err.Error(), rd)
		}

		log.Debugf("add replica success, range:(%+v)", rd)
	}
}

func (s *Server) checkSplit() {
	for _, ns := range s.cluster.getNodeStats() {
		for _, rs := range ns.RangeStatsInfo {
			if s.checkDelayRecord(rs) {
				continue
			}
			if err := s.checkRangeSplit(rs); err != nil {
				log.Errorf("checkRangeSplit error:%s", err.Error())
				s.delayRecord[rs.RangeID] = newDelayRecord()
			}
		}
	}

}

func (s *Server) checkLoop() {
	s.stopper.RunWorker(func() {
		ticker := time.NewTicker(time.Duration(s.cfg.CheckInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if s.cfg.RangeSplitType != splitTypeManual {
					s.checkSplit()
				}

				s.checkReplica()

			case <-s.stopper.ShouldStop():
				return
			}
		}
	})
}

func (s *Server) checkRangeSplit(r *meta.RangeStatsInfo) error {
	if s.dm.checkSplit(r, s.cfg.RangeCapacity, s.cfg.RangeSplitThreshold) {
		k, b, c, err := s.getSplitKey(r)
		if err != nil {
			log.Errorf(errors.ErrorStack(err))
			return errors.Trace(err)
		}

		if err := s.requestSplit(r, k, b, c); err != nil {
			log.Errorf(errors.ErrorStack(err))
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *Server) getSplitKey(rsi *meta.RangeStatsInfo) (key meta.Key, bytes int64, count int64, err error) {
	var rd *meta.RangeDescriptor
	if rd, err = s.region.getRangeDescriptor(rsi.RangeID); err != nil {
		return
	}

	req := meta.GetSplitKeyRequest{
		RequestHeader: meta.RequestHeader{
			Key:     rd.StartKey,
			RangeID: rsi.RangeID,
			Flag:    meta.IsRead,
		},
		SplitSize: rsi.TotalBytes / 2,
	}

	breq := meta.BatchRequest{RequestHeader: req.RequestHeader}
	breq.Add(&req)

	bresp := meta.BatchResponse{}

	log.Infof("get split key request to node %d, range: %d, split size: %d", rsi.NodeID, req.RangeID, req.SplitSize)
	if err = s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return
	}

	key = bresp.Resp[0].GetSplitKey.SplitKey
	bytes = bresp.Resp[0].GetSplitKey.RangeBytes
	count = bresp.Resp[0].GetSplitKey.RangeCount

	log.Infof("get split key success, split key: %v, range bytes: %d, range count: %d", key, bytes, count)
	return
}

func (s *Server) requestSplit(rsi *meta.RangeStatsInfo, key meta.Key, bytes int64, count int64) error {
	id, err := s.idAllocator.newID()
	if err != nil {
		return errors.Errorf("get new rangeID failed")
	}

	rd, err := s.region.getRangeDescriptor(rsi.RangeID)
	if err != nil {
		return errors.Trace(err)
	}

	req := meta.SplitRequest{
		RequestHeader: meta.RequestHeader{
			Key:     key,
			RangeID: rsi.RangeID,
			Flag:    meta.IsWrite,
		},
		SplitKey:      key,
		NewRangeID:    meta.RangeID(id),
		NewRangeBytes: bytes,
		NewRangeCount: count,
	}

	breq := meta.BatchRequest{RequestHeader: req.RequestHeader}
	breq.Add(&req)

	bresp := meta.BatchResponse{}

	log.Infof("split request to node %d, range %d, split key: %s, new range ID: %d, bytes: %d, count: %d",
		rsi.NodeID, req.RangeID, key, id, bytes, count)

	if err := s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return errors.Trace(err)
	}
	log.Info("split request execute end")

	newPostRD := bresp.Resp[0].Split.RangeDescriptors[1]
	if err := s.updateRangeDescriptor(&newPostRD, rd); err != nil {
		log.Errorf("update range descriptor failed, %s\n", err.Error())
		return errors.Trace(err)
	}
	log.Info("updateRangeDescriptor end")

	newPrevRD := bresp.Resp[0].Split.RangeDescriptors[0]
	if err := s.region.setRangeDescriptors(newPostRD, newPrevRD); err != nil {
		log.Errorf("%s\n", err.Error())
		return errors.Trace(err)
	}
	log.Info("update prev range descriptor on pd success")

	log.Info("split request success")

	return nil
}
