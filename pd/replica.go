package pd

import (
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

func (s *Server) addReplica(rID meta.RangeID, nID meta.NodeID) error {
	rd, err := s.region.getRangeDescriptor(rID)
	if err != nil {
		return errors.Errorf("range ID not exist, rangeID: %d, nodeID: %d", rID, nID)
	}

	// add raft node configure
	header := meta.RequestHeader{
		Key:     rd.StartKey,
		EndKey:  rd.EndKey,
		RangeID: rID,
		Flag:    meta.IsSystem,
	}

	arcr := meta.AddRaftConfRequest{
		RequestHeader: header,
		NodeID:        nID,
	}

	breq := meta.BatchRequest{RequestHeader: header}
	bresp := meta.BatchResponse{}
	breq.Add(&arcr)

	if err := s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("add raft node configure success")

	// CompactLog request to leader node
	header.Flag = meta.IsLeader
	clr := meta.CompactLogRequest{
		RequestHeader: header,
	}

	breq = meta.BatchRequest{RequestHeader: header}
	bresp.Reset()
	breq.Add(&clr)

	s.sendBatchRequestToLeader(&breq, &bresp, rd)
	log.Infof("compact log success, range id: %d", rID)

	// create replica
	header.Flag = meta.IsSystem
	crr := meta.CreateReplicaRequest{
		RequestHeader:   header,
		RangeDescriptor: *rd,
		NodeID:          nID,
	}

	breq = meta.BatchRequest{RequestHeader: header}
	bresp.Reset()
	breq.Add(&crr)

	if err := s.sendBatchRequest(&breq, &bresp, nID); err != nil {
		return errors.Trace(err)
	}
	log.Infof("create replica success, rangeID: %d, nodeID: %d", rID, nID)

	// update range descriptor
	nrd := *rd
	nrd.Replicas = append(nrd.Replicas, meta.ReplicaDescriptor{NodeID: nID})
	if err := s.updateRangeDescriptor(&nrd, rd); err != nil {
		log.Errorf("update range descriptor failed, %s", err.Error())
		return errors.Trace(err)
	}

	if err := s.region.setRangeDescriptors(nrd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("update range descriptor on pd success, rangeID: %d", nrd.RangeID)

	return nil
}

func (s *Server) deleteReplica(rID meta.RangeID, nID meta.NodeID) error {
	rd, err := s.region.getRangeDescriptor(rID)
	if err != nil {
		return errors.Errorf("range ID not exist, rangeID: %d, nodeID: %d", rID, nID)
	}

	exist := false
	var repls []meta.ReplicaDescriptor
	for _, rpl := range rd.Replicas {
		if rpl.NodeID == nID {
			exist = true
		} else {
			repl := rpl
			repls = append(repls, repl)
		}
	}
	if !exist {
		return errors.Errorf("replica already not exist, rangeID: %d, nodeID: %d", rID, nID)
	}

	// remove replica
	header := meta.RequestHeader{
		Key:     rd.StartKey,
		EndKey:  rd.EndKey,
		RangeID: rID,
		Flag:    meta.IsSystem,
	}

	rrr := meta.RemoveReplicaRequest{
		RequestHeader: header,
		NodeID:        nID,
	}

	breq := meta.BatchRequest{RequestHeader: header}
	bresp := meta.BatchResponse{}
	breq.Add(&rrr)

	if err := s.sendBatchRequest(&breq, &bresp, nID); err != nil {
		return errors.Trace(err)
	}
	log.Infof("remove replica success, rangeID: %d, nodeID: %d", rID, nID)

	// delete raft node configure
	header.Flag = meta.IsLeader
	drr := meta.DeleteRaftConfRequest{
		RequestHeader: header,
		NodeID:        nID,
	}

	breq = meta.BatchRequest{RequestHeader: header}
	bresp.Reset()
	breq.Add(&drr)

	if err := s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("delete raft node configure success, rangeID: %d, nodeID: %d", rID, nID)

	// update range descriptor
	nrd := *rd
	nrd.Replicas = repls
	if err := s.updateRangeDescriptor(&nrd, rd); err != nil {
		log.Errorf("update range descriptor failed, %s", err.Error())
		return errors.Trace(err)
	}

	if err := s.region.setRangeDescriptors(nrd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("update range descriptor on pd success, rangeID: %d", nrd.RangeID)

	return nil
}

func (s *Server) transferLeader(rID meta.RangeID, srcID, dstID meta.NodeID) error {
	rd, err := s.region.getRangeDescriptor(rID)
	if err != nil {
		return errors.Errorf("range ID not exist, rangeID: %d, srcNodeID: %d dstNodeID: %d", rID, srcID, dstID)
	}

	srcExist := false
	dstExist := false
	for _, repl := range rd.Replicas {
		if repl.NodeID == srcID {
			srcExist = true
		}
		if repl.NodeID == dstID {
			dstExist = true
		}
	}
	if !srcExist || !dstExist {
		return errors.Errorf("replica already not exist, rangeID: %d, srcNodeID: %d, dstNodeID: %d", rID, srcID, dstID)
	}

	header := meta.RequestHeader{
		Key:     rd.StartKey,
		EndKey:  rd.EndKey,
		RangeID: rID,
		Flag:    meta.IsLeader,
	}

	tlr := meta.TransferLeaderRequest{
		RequestHeader: header,
		From:          srcID,
		To:            dstID,
	}

	breq := meta.BatchRequest{RequestHeader: header}
	bresp := meta.BatchResponse{}
	breq.Add(&tlr)

	if err := s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("transfer leader success, rangeID: %d, srcNodeID: %d, dstNodeID: %d", rID, srcID, dstID)

	return nil
}

// updateRangeDescriptor updates range descriptor.
func (s *Server) updateRangeDescriptor(nrd *meta.RangeDescriptor, rd *meta.RangeDescriptor) error {
	header := meta.RequestHeader{
		Key:     nrd.StartKey,
		EndKey:  nrd.EndKey,
		RangeID: nrd.RangeID,
		Flag:    meta.IsWrite,
	}

	urr := meta.UpdateRangeRequest{
		RequestHeader:   header,
		RangeDescriptor: *nrd,
	}

	breq := meta.BatchRequest{RequestHeader: header}
	breq.Add(&urr)
	bresp := meta.BatchResponse{}

	if err := s.sendBatchRequestToLeader(&breq, &bresp, rd); err != nil {
		return errors.Trace(err)
	}
	log.Infof("update range descriptor on node success, rangeID: %d", nrd.RangeID)

	return nil
}
