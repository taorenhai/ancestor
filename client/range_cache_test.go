package client

import (
	"testing"

	"github.com/taorenhai/ancestor/meta"
)

func TestGetNextNodeID(t *testing.T) {
	rd := &rangeDesc{
		RangeDescriptor: meta.RangeDescriptor{
			Replicas: []meta.ReplicaDescriptor{
				{NodeID: 0},
				{NodeID: 1},
				{NodeID: 2},
			},
		},
	}

	nodeID, leaderIdx := rd.getLeaderNodeID()
	for i := 1; i < 10; i++ {
		nodeID, leaderIdx = rd.getNextNodeID(leaderIdx)
		if int(nodeID) != i%3 {
			t.Fatalf("idx:%d, expect %d nodeID:%d", i, i%3, nodeID)
		}
	}
}

func BenchmarkMutilGetNextNodeID(b *testing.B) {
	rd := &rangeDesc{
		RangeDescriptor: meta.RangeDescriptor{
			Replicas: []meta.ReplicaDescriptor{
				{NodeID: 0},
				{NodeID: 1},
				{NodeID: 2},
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			nodeID, leaderIdx := rd.getLeaderNodeID()
			newNodeID, newLeaderIdx := rd.getNextNodeID(leaderIdx)
			if nodeID == newNodeID || leaderIdx == newLeaderIdx {
				b.Fatalf("old NodeID:%d, leaderIdx:%d, new NodeID:%d, leaderIdx:%d", nodeID, leaderIdx, newNodeID, newLeaderIdx)
			}
		}
	})
}
