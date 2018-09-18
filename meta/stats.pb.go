// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: stats.proto

package meta

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// skipping weak import gogoproto "github.com/gogo/protobuf/gogoproto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// RangeStats tracks byte and instance counts for various groups of keys,
// values, or key-value pairs; see the field comments for details.
//
// It also tracks two cumulative ages, namely that of intents and non-live
// (i.e. GC-able) bytes. This computation is intrinsically linked to
// last_update_nanos and is easy to get wrong. Updates happen only once every
// full second, as measured by last_update_nanos/1e9. That is, forward updates
// don't change last_update_nanos until an update at a timestamp which,
// truncated to the second, is ahead of last_update_nanos/1e9. Then, that
// difference in seconds times the base quantity (excluding the currently
// running update) is added to the age. It gets more complicated when data is
// accounted for with a timestamp behind last_update_nanos. In this case, if
// more than a second has passed (computed via truncation above), the ages have
// to be adjusted to account for this late addition. This isn't hard: add the
// new data's base quantity times the (truncated) number of seconds behind.
// Important to keep in mind with those computations is that (x/1e9 - y/1e9)
// does not equal (x-y)/1e9 in most cases.
type RangeStatsInfo struct {
	RangeID RangeID `protobuf:"varint,1,opt,name=range_id,json=rangeId,proto3,casttype=RangeID" json:"range_id,omitempty"`
	NodeID  NodeID  `protobuf:"varint,2,opt,name=node_id,json=nodeId,proto3,casttype=NodeID" json:"node_id,omitempty"`
	// total_bytes is the number of bytes stored in all non-system
	// keys, including live, meta, old, and deleted keys.
	// Only meta keys really account for the "full" key; value
	// keys only for the timestamp suffix.
	TotalBytes int64 `protobuf:"varint,3,opt,name=total_bytes,json=totalBytes,proto3" json:"total_bytes,omitempty"`
	// total_count is the number of meta keys tracked under total_bytes.
	TotalCount           int64    `protobuf:"varint,4,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	IsRaftLeader         bool     `protobuf:"varint,5,opt,name=is_raft_leader,json=isRaftLeader,proto3" json:"is_raft_leader,omitempty"`
	IsRemoved            bool     `protobuf:"varint,6,opt,name=is_removed,json=isRemoved,proto3" json:"is_removed,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RangeStatsInfo) Reset()         { *m = RangeStatsInfo{} }
func (m *RangeStatsInfo) String() string { return proto.CompactTextString(m) }
func (*RangeStatsInfo) ProtoMessage()    {}
func (*RangeStatsInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_stats_7dcac4a364cafb57, []int{0}
}
func (m *RangeStatsInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RangeStatsInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RangeStatsInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RangeStatsInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RangeStatsInfo.Merge(dst, src)
}
func (m *RangeStatsInfo) XXX_Size() int {
	return m.Size()
}
func (m *RangeStatsInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_RangeStatsInfo.DiscardUnknown(m)
}

var xxx_messageInfo_RangeStatsInfo proto.InternalMessageInfo

func (m *RangeStatsInfo) GetRangeID() RangeID {
	if m != nil {
		return m.RangeID
	}
	return 0
}

func (m *RangeStatsInfo) GetNodeID() NodeID {
	if m != nil {
		return m.NodeID
	}
	return 0
}

func (m *RangeStatsInfo) GetTotalBytes() int64 {
	if m != nil {
		return m.TotalBytes
	}
	return 0
}

func (m *RangeStatsInfo) GetTotalCount() int64 {
	if m != nil {
		return m.TotalCount
	}
	return 0
}

func (m *RangeStatsInfo) GetIsRaftLeader() bool {
	if m != nil {
		return m.IsRaftLeader
	}
	return false
}

func (m *RangeStatsInfo) GetIsRemoved() bool {
	if m != nil {
		return m.IsRemoved
	}
	return false
}

type NodeStatsInfo struct {
	NodeID               NodeID                      `protobuf:"varint,1,opt,name=node_id,json=nodeId,proto3,casttype=NodeID" json:"node_id,omitempty"`
	TotalBytes           int64                       `protobuf:"varint,2,opt,name=total_bytes,json=totalBytes,proto3" json:"total_bytes,omitempty"`
	TotalCount           int64                       `protobuf:"varint,3,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	LeaderCount          uint64                      `protobuf:"varint,4,opt,name=leader_count,json=leaderCount,proto3" json:"leader_count,omitempty"`
	RangeStatsInfo       map[RangeID]*RangeStatsInfo `protobuf:"bytes,5,rep,name=range_stats_info,json=rangeStatsInfo,castkey=RangeID" json:"range_stats_info,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *NodeStatsInfo) Reset()         { *m = NodeStatsInfo{} }
func (m *NodeStatsInfo) String() string { return proto.CompactTextString(m) }
func (*NodeStatsInfo) ProtoMessage()    {}
func (*NodeStatsInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_stats_7dcac4a364cafb57, []int{1}
}
func (m *NodeStatsInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NodeStatsInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NodeStatsInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *NodeStatsInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NodeStatsInfo.Merge(dst, src)
}
func (m *NodeStatsInfo) XXX_Size() int {
	return m.Size()
}
func (m *NodeStatsInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_NodeStatsInfo.DiscardUnknown(m)
}

var xxx_messageInfo_NodeStatsInfo proto.InternalMessageInfo

func (m *NodeStatsInfo) GetNodeID() NodeID {
	if m != nil {
		return m.NodeID
	}
	return 0
}

func (m *NodeStatsInfo) GetTotalBytes() int64 {
	if m != nil {
		return m.TotalBytes
	}
	return 0
}

func (m *NodeStatsInfo) GetTotalCount() int64 {
	if m != nil {
		return m.TotalCount
	}
	return 0
}

func (m *NodeStatsInfo) GetLeaderCount() uint64 {
	if m != nil {
		return m.LeaderCount
	}
	return 0
}

func (m *NodeStatsInfo) GetRangeStatsInfo() map[RangeID]*RangeStatsInfo {
	if m != nil {
		return m.RangeStatsInfo
	}
	return nil
}

func init() {
	proto.RegisterType((*RangeStatsInfo)(nil), "ancestor.meta.RangeStatsInfo")
	proto.RegisterType((*NodeStatsInfo)(nil), "ancestor.meta.NodeStatsInfo")
	proto.RegisterMapType((map[RangeID]*RangeStatsInfo)(nil), "ancestor.meta.NodeStatsInfo.RangeStatsInfoEntry")
}
func (m *RangeStatsInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RangeStatsInfo) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.RangeID != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.RangeID))
	}
	if m.NodeID != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.NodeID))
	}
	if m.TotalBytes != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.TotalBytes))
	}
	if m.TotalCount != 0 {
		dAtA[i] = 0x20
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.TotalCount))
	}
	if m.IsRaftLeader {
		dAtA[i] = 0x28
		i++
		if m.IsRaftLeader {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i++
	}
	if m.IsRemoved {
		dAtA[i] = 0x30
		i++
		if m.IsRemoved {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i++
	}
	return i, nil
}

func (m *NodeStatsInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NodeStatsInfo) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.NodeID != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.NodeID))
	}
	if m.TotalBytes != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.TotalBytes))
	}
	if m.TotalCount != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.TotalCount))
	}
	if m.LeaderCount != 0 {
		dAtA[i] = 0x20
		i++
		i = encodeVarintStats(dAtA, i, uint64(m.LeaderCount))
	}
	if len(m.RangeStatsInfo) > 0 {
		for k, _ := range m.RangeStatsInfo {
			dAtA[i] = 0x2a
			i++
			v := m.RangeStatsInfo[k]
			msgSize := 0
			if v != nil {
				msgSize = v.Size()
				msgSize += 1 + sovStats(uint64(msgSize))
			}
			mapSize := 1 + sovStats(uint64(k)) + msgSize
			i = encodeVarintStats(dAtA, i, uint64(mapSize))
			dAtA[i] = 0x8
			i++
			i = encodeVarintStats(dAtA, i, uint64(k))
			if v != nil {
				dAtA[i] = 0x12
				i++
				i = encodeVarintStats(dAtA, i, uint64(v.Size()))
				n1, err := v.MarshalTo(dAtA[i:])
				if err != nil {
					return 0, err
				}
				i += n1
			}
		}
	}
	return i, nil
}

func encodeVarintStats(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *RangeStatsInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RangeID != 0 {
		n += 1 + sovStats(uint64(m.RangeID))
	}
	if m.NodeID != 0 {
		n += 1 + sovStats(uint64(m.NodeID))
	}
	if m.TotalBytes != 0 {
		n += 1 + sovStats(uint64(m.TotalBytes))
	}
	if m.TotalCount != 0 {
		n += 1 + sovStats(uint64(m.TotalCount))
	}
	if m.IsRaftLeader {
		n += 2
	}
	if m.IsRemoved {
		n += 2
	}
	return n
}

func (m *NodeStatsInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.NodeID != 0 {
		n += 1 + sovStats(uint64(m.NodeID))
	}
	if m.TotalBytes != 0 {
		n += 1 + sovStats(uint64(m.TotalBytes))
	}
	if m.TotalCount != 0 {
		n += 1 + sovStats(uint64(m.TotalCount))
	}
	if m.LeaderCount != 0 {
		n += 1 + sovStats(uint64(m.LeaderCount))
	}
	if len(m.RangeStatsInfo) > 0 {
		for k, v := range m.RangeStatsInfo {
			_ = k
			_ = v
			l = 0
			if v != nil {
				l = v.Size()
				l += 1 + sovStats(uint64(l))
			}
			mapEntrySize := 1 + sovStats(uint64(k)) + l
			n += mapEntrySize + 1 + sovStats(uint64(mapEntrySize))
		}
	}
	return n
}

func sovStats(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozStats(x uint64) (n int) {
	return sovStats(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *RangeStatsInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowStats
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RangeStatsInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RangeStatsInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeID", wireType)
			}
			m.RangeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RangeID |= (RangeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeID", wireType)
			}
			m.NodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NodeID |= (NodeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalBytes", wireType)
			}
			m.TotalBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalCount", wireType)
			}
			m.TotalCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IsRaftLeader", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.IsRaftLeader = bool(v != 0)
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IsRemoved", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.IsRemoved = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipStats(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthStats
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *NodeStatsInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowStats
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: NodeStatsInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NodeStatsInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeID", wireType)
			}
			m.NodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NodeID |= (NodeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalBytes", wireType)
			}
			m.TotalBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalCount", wireType)
			}
			m.TotalCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LeaderCount", wireType)
			}
			m.LeaderCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LeaderCount |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeStatsInfo", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStats
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthStats
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.RangeStatsInfo == nil {
				m.RangeStatsInfo = make(map[RangeID]*RangeStatsInfo)
			}
			var mapkey int64
			var mapvalue *RangeStatsInfo
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowStats
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowStats
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapkey |= (int64(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
				} else if fieldNum == 2 {
					var mapmsglen int
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowStats
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapmsglen |= (int(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					if mapmsglen < 0 {
						return ErrInvalidLengthStats
					}
					postmsgIndex := iNdEx + mapmsglen
					if mapmsglen < 0 {
						return ErrInvalidLengthStats
					}
					if postmsgIndex > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = &RangeStatsInfo{}
					if err := mapvalue.Unmarshal(dAtA[iNdEx:postmsgIndex]); err != nil {
						return err
					}
					iNdEx = postmsgIndex
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipStats(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if skippy < 0 {
						return ErrInvalidLengthStats
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.RangeStatsInfo[RangeID(mapkey)] = mapvalue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipStats(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthStats
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipStats(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowStats
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowStats
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowStats
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthStats
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowStats
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipStats(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthStats = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowStats   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("stats.proto", fileDescriptor_stats_7dcac4a364cafb57) }

var fileDescriptor_stats_7dcac4a364cafb57 = []byte{
	// 397 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x31, 0x6f, 0x9b, 0x40,
	0x1c, 0xc5, 0x7d, 0x80, 0xb1, 0x7b, 0xd8, 0x96, 0x75, 0xb5, 0x2c, 0x64, 0xc9, 0x98, 0x5a, 0x1d,
	0x58, 0x4a, 0x5b, 0x7b, 0xa9, 0x3a, 0xd2, 0x76, 0x40, 0xaa, 0x3a, 0x5c, 0x97, 0xaa, 0x0b, 0x3d,
	0x9b, 0x03, 0xa1, 0xda, 0x5c, 0x05, 0x67, 0x4b, 0x5e, 0xfb, 0x09, 0xfa, 0xb1, 0x3a, 0xe6, 0x13,
	0x38, 0x11, 0xd9, 0xb3, 0x27, 0x53, 0x74, 0x77, 0x91, 0x30, 0x49, 0xa4, 0x28, 0xdb, 0xd3, 0xe3,
	0xf7, 0xd0, 0xff, 0xf1, 0x80, 0x56, 0xc9, 0x09, 0x2f, 0xfd, 0x3f, 0x05, 0xe3, 0x0c, 0xf5, 0x49,
	0xbe, 0xa6, 0x25, 0x67, 0x85, 0xbf, 0xa5, 0x9c, 0x4c, 0x46, 0x29, 0x4b, 0x99, 0x7c, 0xf2, 0x56,
	0x28, 0x05, 0xcd, 0xaf, 0x01, 0x1c, 0x60, 0x92, 0xa7, 0xf4, 0xbb, 0x48, 0x86, 0x79, 0xc2, 0xd0,
	0x7b, 0xd8, 0x2d, 0x84, 0x13, 0x65, 0xb1, 0x0d, 0x5c, 0xe0, 0xf5, 0x83, 0x71, 0x75, 0x9c, 0x75,
	0x24, 0x15, 0x7e, 0xbe, 0xa9, 0x25, 0xee, 0x48, 0x2e, 0x8c, 0xd1, 0x1b, 0xd8, 0xc9, 0x59, 0x2c,
	0x13, 0x9a, 0x0b, 0x3c, 0x23, 0x18, 0x55, 0xc7, 0x99, 0xf9, 0x8d, 0xc5, 0x2a, 0x70, 0xa7, 0xb0,
	0x29, 0xa0, 0x30, 0x46, 0x33, 0x68, 0x71, 0xc6, 0xc9, 0x26, 0x5a, 0x1d, 0x38, 0x2d, 0x6d, 0xdd,
	0x05, 0x9e, 0x8e, 0xa1, 0xb4, 0x02, 0xe1, 0xd4, 0xc0, 0x9a, 0xed, 0x72, 0x6e, 0x1b, 0x27, 0xc0,
	0x27, 0xe1, 0xa0, 0xd7, 0x70, 0x90, 0x95, 0x51, 0x41, 0x12, 0x1e, 0x6d, 0x28, 0x89, 0x69, 0x61,
	0xb7, 0x5d, 0xe0, 0x75, 0x71, 0x2f, 0x2b, 0x31, 0x49, 0xf8, 0x57, 0xe9, 0xa1, 0x29, 0x84, 0x82,
	0xa2, 0x5b, 0xb6, 0xa7, 0xb1, 0x6d, 0x4a, 0xe2, 0x45, 0x56, 0x62, 0x65, 0xcc, 0xaf, 0x34, 0xd8,
	0x17, 0x97, 0xd5, 0xd5, 0x4f, 0x7a, 0x80, 0xe7, 0xf7, 0xd0, 0x9e, 0xea, 0xa1, 0x3f, 0xe8, 0xf1,
	0x0a, 0xf6, 0xd4, 0xfd, 0x27, 0x4d, 0x0d, 0x6c, 0x29, 0x4f, 0x21, 0x29, 0x1c, 0xaa, 0x39, 0xe4,
	0xb6, 0x51, 0x96, 0x27, 0xcc, 0x6e, 0xbb, 0xba, 0x67, 0x2d, 0xde, 0xf9, 0x8d, 0x85, 0xfd, 0x46,
	0x17, 0xbf, 0xb9, 0xea, 0x97, 0x9c, 0x17, 0x87, 0xc0, 0xfa, 0x7b, 0x5e, 0xaf, 0x37, 0x28, 0x1a,
	0xc4, 0xe4, 0x17, 0x7c, 0xf9, 0x48, 0x06, 0x0d, 0xa1, 0xfe, 0x9b, 0x1e, 0xe4, 0xf7, 0xd0, 0xb1,
	0x90, 0x68, 0x09, 0xdb, 0x7b, 0xb2, 0xd9, 0x51, 0x59, 0xd8, 0x5a, 0x4c, 0xef, 0x9d, 0xd1, 0x7c,
	0x09, 0x56, 0xec, 0x47, 0xed, 0x03, 0x08, 0xc6, 0xff, 0x2b, 0x07, 0x9c, 0x55, 0x0e, 0xb8, 0xa8,
	0x1c, 0xf0, 0xef, 0xd2, 0x69, 0xfd, 0x34, 0x44, 0xe2, 0x47, 0x6b, 0x65, 0xca, 0xbf, 0x71, 0x79,
	0x1b, 0x00, 0x00, 0xff, 0xff, 0x81, 0xa9, 0xb9, 0xb9, 0xc1, 0x02, 0x00, 0x00,
}
