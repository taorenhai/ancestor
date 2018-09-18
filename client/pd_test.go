package client

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

type testPDSender struct {
	timestamp  int64
	rangeID    meta.RangeID
	rangeDescs map[meta.RangeID]meta.RangeDescriptor
	nodeDescs  map[meta.NodeID]meta.NodeDescriptor
	rangeStats map[meta.RangeID]meta.RangeStatsInfo
	isStop     bool
	sync.Mutex
}

func newTestPDSender() *testPDSender {
	return &testPDSender{
		rangeDescs: make(map[meta.RangeID]meta.RangeDescriptor),
		nodeDescs:  make(map[meta.NodeID]meta.NodeDescriptor),
		rangeStats: make(map[meta.RangeID]meta.RangeStatsInfo),
	}
}

func (tms *testPDSender) getTimestamp() meta.Timestamp {
	tms.Lock()
	defer tms.Unlock()

	tms.timestamp++

	ts := meta.Timestamp{WallTime: tms.timestamp}

	return ts
}

func (tms *testPDSender) getNewID() meta.RangeID {
	tms.Lock()
	defer tms.Unlock()

	tms.rangeID++

	return tms.rangeID
}

func (tms *testPDSender) setRangeDescs(rangeDescs []meta.RangeDescriptor) {
	tms.Lock()
	defer tms.Unlock()

	for _, rangeDesc := range rangeDescs {
		tms.rangeDescs[rangeDesc.RangeID] = rangeDesc
	}
}

func (tms *testPDSender) getRangeDescs() []meta.RangeDescriptor {
	tms.Lock()
	defer tms.Unlock()

	rangeDescs := []meta.RangeDescriptor{}

	for _, v := range tms.rangeDescs {
		rangeDescs = append(rangeDescs, v)
	}

	return rangeDescs
}

func (tms *testPDSender) setNodeDescs(nodeDescs []meta.NodeDescriptor) {
	tms.Lock()
	defer tms.Unlock()

	for _, nodeDesc := range nodeDescs {
		tms.nodeDescs[nodeDesc.NodeID] = nodeDesc
	}
}

func (tms *testPDSender) getNodeDescs() []meta.NodeDescriptor {
	tms.Lock()
	defer tms.Unlock()

	nodeDescs := []meta.NodeDescriptor{}

	for _, v := range tms.nodeDescs {
		nodeDescs = append(nodeDescs, v)
	}

	return nodeDescs
}

func (tms *testPDSender) setRangeStats(rangeStats []meta.RangeStatsInfo) {
	tms.Lock()
	defer tms.Unlock()

	for _, rs := range rangeStats {
		localRS := rs
		tms.rangeStats[rs.RangeID] = localRS
	}
}

func (tms *testPDSender) getRangeStats() []meta.RangeStatsInfo {
	tms.Lock()
	defer tms.Unlock()

	rsList := []meta.RangeStatsInfo{}

	for _, rs := range tms.rangeStats {
		rsList = append(rsList, rs)
		log.Debugf("append rs:%v", rs)
	}
	return rsList
}

func (tms *testPDSender) ExecuteCmd(inner interface{}) meta.Response {
	switch req := inner.(type) {
	case *meta.GetRangeDescriptorsRequest:
		resp := &meta.GetRangeDescriptorsResponse{}
		for _, v := range tms.getRangeDescs() {
			rangeDesc := v
			resp.RangeDescriptor = append(resp.RangeDescriptor, &rangeDesc)
		}
		return resp

	case *meta.SetRangeDescriptorsRequest:
		resp := &meta.SetRangeDescriptorsResponse{}
		tms.setRangeDescs(req.RangeDescriptor)
		return resp

	case *meta.GetNodeDescriptorsRequest:
		resp := &meta.GetNodeDescriptorsResponse{}
		for _, v := range tms.getNodeDescs() {
			nodeDesc := v
			resp.NodeDescriptor = append(resp.NodeDescriptor, &nodeDesc)
		}
		return resp

	case *meta.SetNodeDescriptorsRequest:
		resp := &meta.SetNodeDescriptorsResponse{}
		tms.setNodeDescs(req.NodeDescriptor)
		return resp

	case *meta.GetTimestampRequest:
		resp := &meta.GetTimestampResponse{}
		resp.Timestamp = tms.getTimestamp()
		return resp

	case *meta.GetNewIDRequest:
		resp := &meta.GetNewIDResponse{}
		resp.ID = tms.getNewID()
		return resp

	case *meta.SetRangeStatsRequest:
		resp := &meta.SetRangeStatsResponse{}
		tms.setRangeStats(req.GetRangeStatsInfo())
		return resp

	case *meta.GetRangeStatsRequest:
		resp := &meta.GetRangeStatsResponse{}
		for _, v := range tms.getRangeStats() {
			rangeStats := v
			resp.Rows = append(resp.Rows, &rangeStats)
		}
		return resp
	}

	return nil

}

func (tms *testPDSender) send(req *meta.BatchRequest, resp *meta.BatchResponse) error {
	for _, childReq := range req.GetReq() {
		inner := childReq.GetInner()
		childResp := tms.ExecuteCmd(inner)
		resp.Add(childResp)
	}

	return nil
}

func (tms *testPDSender) stop() {
	tms.isStop = true
}

func (tms *testPDSender) getAddr() string {
	return "TestHost:8888"
}

func TestPDGetNewRangeID(t *testing.T) {
	tms := newTestPDSender()
	pd := newPDAPI(tms)

	for i := 0; i < 10; i++ {
		id, err := pd.getNewID()
		if err != nil {
			t.Fatalf("getNewID err:%s", err.Error())
		}
		t.Logf("new RangeID:%d", id)
	}
}

func TestPDGetTimestamp(t *testing.T) {
	tms := newTestPDSender()
	pd := newPDAPI(tms)

	for i := 0; i < 10; i++ {
		ts, err := pd.getTimestamp()
		if err != nil {
			t.Fatalf("getNewID err:%s", err.Error())
		}
		t.Logf("new timestamp:%d", ts.WallTime)
	}
}

func TestPDRangeDescs(t *testing.T) {
	tms := newTestPDSender()
	pd := newPDAPI(tms)

	rangeDescs := []meta.RangeDescriptor{}

	for i := 0; i < 10; i++ {
		rangeID, err := pd.getNewID()
		if err != nil {
			t.Fatalf("getNewID error:%s", err.Error())
		}
		t.Logf("recv rangeID:%d", rangeID)
		rangeDesc := meta.RangeDescriptor{RangeID: rangeID}
		rangeDescs = append(rangeDescs, rangeDesc)
	}

	if err := pd.setRangeDescriptors(rangeDescs); err != nil {
		t.Fatalf("setRangeDescriptors err:%s", err.Error())
	}

	newRangeDescs, err := pd.getRangeDescriptors()
	if err != nil {
		t.Fatalf("getRangeDescriptors err:%s", err.Error())
	}

	if len(newRangeDescs) != 10 {
		t.Fatalf("getRangeDescriptors expect size:10, recv:%d", len(newRangeDescs))
	}

	for i, r := range newRangeDescs {
		t.Logf("getRangeDescriptors recv range[%d] id:%d", i, r.RangeID)
	}
}

func TestPDRangeStats(t *testing.T) {
	tms := newTestPDSender()
	pd := newPDAPI(tms)

	rangeStats := []meta.RangeStatsInfo{}

	for i := 0; i < 10; i++ {
		rangeID, err := pd.getNewID()
		if err != nil {
			t.Fatalf("getNewID error:%s", err.Error())
		}
		t.Logf("recv rangeID:%d", rangeID)
		rs := meta.RangeStatsInfo{RangeID: rangeID, NodeID: meta.NodeID(i)}
		rangeStats = append(rangeStats, rs)
	}

	if err := pd.setRangeStats(rangeStats); err != nil {
		t.Fatalf("setRangeStats err:%s", err.Error())
	}

	newRangeStats, err := pd.getRangeStats([]meta.RangeID{0})
	if err != nil {
		t.Fatalf("getRangeStats err:%s", err.Error())
	}

	if len(newRangeStats) != 10 {
		t.Fatalf("getRangeStats expect size:10, recv:%d", len(newRangeStats))
	}

	for i, r := range newRangeStats {
		t.Logf("getRangeStats recv range[%d] id:%d, nodeID:%d", i, r.RangeID, r.NodeID)
	}

}

func TestPDNode(t *testing.T) {
	tms := newTestPDSender()
	pd := newPDAPI(tms)

	nodeDescs := []meta.NodeDescriptor{}

	for i := 0; i < 10; i++ {
		nodeDesc := meta.NodeDescriptor{NodeID: meta.NodeID(i), Address: fmt.Sprintf("127.0.0.%d", i)}
		nodeDescs = append(nodeDescs, nodeDesc)
	}

	if err := pd.setNodeDescriptors(nodeDescs); err != nil {
		t.Fatalf("setNodeDescriptors err:%s", err.Error())
	}

	newNodeDescs, err := pd.getNodeDescriptors()
	if err != nil {
		t.Fatalf("getNodeDescriptors err:%s", err.Error())
	}

	if len(newNodeDescs) != 10 {
		t.Fatalf("getNodeDescriptors expect size:10, recv:%d", len(newNodeDescs))
	}

	for _, r := range newNodeDescs {
		t.Logf("getNodeDescriptors recv node[%d]:%s", r.NodeID, r.Address)
	}

}
