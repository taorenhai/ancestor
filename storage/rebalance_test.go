package storage

import (
	"testing"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"
)

const (
	startKeyConst = ""
	endKeyConst   = "tz"

	nodeID1 = 1
	nodeID2 = 2
	nodeID3 = 3
	nodeID4 = 4

	nodeAddress1 = "127.0.0.1:9001"
	nodeAddress2 = "127.0.0.1:9002"
	nodeAddress3 = "127.0.0.1:9003"
	nodeAddress4 = "127.0.0.1:9004"

	splitBytesSize = 10000
	newRangeBytes  = 69147
	newRangeCount  = 1033

	splitKey = "t\200\000\000\000\000\000\000>_r\200\000\000\000\000\000\000\001"
)

var (
	splitRangeID    = meta.RangeID(40)
	newRangeID      = meta.RangeID(100)
	raftTestRangeID = meta.RangeID(40)

	nodeDescs = []meta.NodeDescriptor{
		{NodeID: meta.NodeID(0), Address: ""},
		{NodeID: meta.NodeID(nodeID1), Address: nodeAddress1},
		{NodeID: meta.NodeID(nodeID2), Address: nodeAddress2},
		{NodeID: meta.NodeID(nodeID3), Address: nodeAddress3},
		{NodeID: meta.NodeID(nodeID4), Address: nodeAddress4},
	}

	rangeDesc = meta.RangeDescriptor{
		RangeID:  raftTestRangeID,
		StartKey: meta.Key(startKeyConst),
		EndKey:   meta.Key(endKeyConst),
		Replicas: []meta.ReplicaDescriptor{
			{NodeID: meta.NodeID(nodeID1)},
			{NodeID: meta.NodeID(nodeID2)},
			{NodeID: meta.NodeID(nodeID3)},
			{NodeID: meta.NodeID(nodeID4)},
		},
	}
)

func TestGetSplitKey(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: splitRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsRead,
		}

		req = &meta.GetSplitKeyRequest{
			RequestHeader: header,
			SplitSize:     int64(splitBytesSize),
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

// first start three nodes in one raft group, then
// for leader to handle split: SplitRequest
func TestSplit(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: splitRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsAdmin,
		}

		req = &meta.SplitRequest{
			RequestHeader: header,
			NewRangeID:    newRangeID,
			SplitKey:      meta.Key(splitKey),
			NewRangeBytes: int64(newRangeBytes),
			NewRangeCount: int64(newRangeCount),
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func TestAddRaftConf(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLeader,
		}

		req = &meta.AddRaftConfRequest{
			RequestHeader: header,
			NodeID:        nodeID4,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func TestDeleteRaftConf(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLeader,
		}

		req = &meta.DeleteRaftConfRequest{
			RequestHeader: header,
			NodeID:        nodeID4,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func TestCreateReplica(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLocal,
		}

		req = &meta.CreateReplicaRequest{
			RequestHeader:   header,
			RangeDescriptor: rangeDesc,
			NodeID:          nodeID4,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		createNode = &nodeDescs[nodeID4]
	)

	breq.Add(req)
	client, conn, err := util.NewNodeClient(createNode.Address)
	if err != nil {
		log.Errorf("rpc new node:%v client error:%v", createNode.Address, err)
		return
	}
	defer conn.Close()

	bresp, err = client.NodeMessage(context.TODO(), breq)
	if err != nil {
		log.Errorf("%v rpc get message error:%v", createNode.Address, err)
		return
	}

	log.Infof("TestCreateReplica response:%v", *bresp)
}

func TestRemoveReplica(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLocal,
		}

		req = &meta.RemoveReplicaRequest{
			RequestHeader: header,
			NodeID:        testNodeID,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		removeNode = nodeDescs[nodeID1]
	)

	breq.Add(req)
	client, conn, err := util.NewNodeClient(removeNode.Address)
	if err != nil {
		log.Errorf("rpc new node:%v client error:%v", removeNode.Address, err)
		return
	}
	defer conn.Close()

	bresp, err = client.NodeMessage(context.TODO(), breq)
	if err != nil {
		log.Errorf("%v rpc get message error:%v", removeNode.Address, err)
		return
	}
	log.Infof("TestCreateReplica response:%v", *bresp)
}

func TestUpdateRange(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsWrite,
		}

		req = &meta.UpdateRangeRequest{
			RequestHeader:   header,
			RangeDescriptor: rangeDesc,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func TestCompactLog(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLeader,
		}

		req = &meta.CompactLogRequest{
			RequestHeader: header,
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func TestTransferLeader(t *testing.T) {
	var (
		header = meta.RequestHeader{
			RangeID: raftTestRangeID,
			Key:     meta.Key(startKeyConst),
			EndKey:  meta.Key(endKeyConst),
			Flag:    meta.IsLeader,
		}

		req = &meta.TransferLeaderRequest{
			RequestHeader: header,
			From:          meta.NodeID(nodeID1),
			To:            meta.NodeID(nodeID3),
		}

		breq       = &meta.BatchRequest{RequestHeader: header}
		bresp      = &meta.BatchResponse{}
		leaderNode = &nodeDescs[nodeID1]
	)

	breq.Add(req)
	rpcLeaderCall(leaderNode, breq, bresp)
}

func rpcLeaderCall(nd *meta.NodeDescriptor, breq *meta.BatchRequest, bresp *meta.BatchResponse) {
	client, conn, err := util.NewNodeClient(nd.Address)
	if err != nil {
		log.Errorf("rpc new node:%v client error:%v", nd.Address, err)
		return
	}
	defer conn.Close()

	bresp, err = client.NodeMessage(context.TODO(), breq)
	if err != nil {
		log.Errorf("%v rpc get message error:%v", nd.Address, err)
		return
	}

	if pError := bresp.GetError(); pError != nil {
		err := pError.GoError()
		e, ok := errors.Cause(err).(*meta.NotLeaderError)
		if !ok {
			log.Errorf("first rpc request leader error:%v, resp:%v", err, bresp)
			return
		}

		leaderID := e.NodeID
		log.Infof("first rpc request not leader, right leader:%v", leaderID)
		switch leaderID {
		case meta.NodeID(nodeID1):
			nd = &nodeDescs[nodeID1]
		case meta.NodeID(nodeID2):
			nd = &nodeDescs[nodeID2]
		case meta.NodeID(nodeID3):
			nd = &nodeDescs[nodeID3]
		case meta.NodeID(nodeID4):
			nd = &nodeDescs[nodeID4]
		default:
			log.Errorf("rpcLeaderCall not found leader node")
			return
		}

		client, conn, err := util.NewNodeClient(nd.Address)
		if err != nil {
			log.Errorf("rpc new node:%v client error:%v", nd.Address, err)
			return
		}
		defer conn.Close()

		bresp, err = client.NodeMessage(context.TODO(), breq)
		if err != nil {
			log.Errorf("%v rpc get message error:%v", nd.Address, err)
			return
		}
	}

	log.Infof("rpc request leader success resp:%v", *bresp)
}
