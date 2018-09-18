package meta

import (
	"bytes"
	"github.com/taorenhai/ancestor/util"

	proto "github.com/gogo/protobuf/proto"
)

// NodeID is a custom type for a ancestor node ID. (not a raft node ID)
type NodeID int32

// ReplicaID is a custom type for a range replica ID.
type ReplicaID int32

// RangeID is a range for key
type RangeID int64

// Clone creates a deep copy of the given transaction.
func (t *Transaction) Clone() *Transaction {
	return proto.Clone(t).(*Transaction)
}

// GetNextNodeID get next replica nodeID.
func (r *RangeDescriptor) GetNextNodeID(id NodeID) NodeID {
	for i, rp := range r.Replicas {
		if rp.NodeID == id {
			return r.Replicas[(i+1)%len(r.Replicas)].NodeID
		}
	}
	return 0
}

// IsSystem return true if rangeDescriptor is system.
func (r *RangeDescriptor) IsSystem() bool {
	if bytes.HasPrefix(r.StartKey, util.SystemPrefix) {
		return true
	}
	return false
}
