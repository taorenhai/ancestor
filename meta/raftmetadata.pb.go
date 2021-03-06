// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: raftmetadata.proto

package meta

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import raftpb "github.com/coreos/etcd/raft/raftpb"

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

// RaftMessageRequest is the request used to send raft messages using
// our protobuf-based RPC codec.
type RaftMessageRequest struct {
	RangeID              RangeID        `protobuf:"varint,1,opt,name=range_id,json=rangeId,proto3,casttype=RangeID" json:"range_id,omitempty"`
	FromNodeID           NodeID         `protobuf:"varint,2,opt,name=from_node_id,json=fromNodeId,proto3,casttype=NodeID" json:"from_node_id,omitempty"`
	ToNodeID             NodeID         `protobuf:"varint,3,opt,name=to_node_id,json=toNodeId,proto3,casttype=NodeID" json:"to_node_id,omitempty"`
	Message              raftpb.Message `protobuf:"bytes,4,opt,name=message" json:"message"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *RaftMessageRequest) Reset()         { *m = RaftMessageRequest{} }
func (m *RaftMessageRequest) String() string { return proto.CompactTextString(m) }
func (*RaftMessageRequest) ProtoMessage()    {}
func (*RaftMessageRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{0}
}
func (m *RaftMessageRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftMessageRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftMessageRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftMessageRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftMessageRequest.Merge(dst, src)
}
func (m *RaftMessageRequest) XXX_Size() int {
	return m.Size()
}
func (m *RaftMessageRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftMessageRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RaftMessageRequest proto.InternalMessageInfo

// RaftMessageResponse is an empty message returned by raft RPCs. If a
// response is needed it will be sent as a separate message.
type RaftMessageResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RaftMessageResponse) Reset()         { *m = RaftMessageResponse{} }
func (m *RaftMessageResponse) String() string { return proto.CompactTextString(m) }
func (*RaftMessageResponse) ProtoMessage()    {}
func (*RaftMessageResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{1}
}
func (m *RaftMessageResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftMessageResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftMessageResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftMessageResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftMessageResponse.Merge(dst, src)
}
func (m *RaftMessageResponse) XXX_Size() int {
	return m.Size()
}
func (m *RaftMessageResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftMessageResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RaftMessageResponse proto.InternalMessageInfo

// RaftMessageBatchRequest A RaftMessageBatchRequest contains one or more requests
type RaftMessageBatchRequest struct {
	Requests             []RaftMessageRequest `protobuf:"bytes,1,rep,name=requests" json:"requests"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *RaftMessageBatchRequest) Reset()         { *m = RaftMessageBatchRequest{} }
func (m *RaftMessageBatchRequest) String() string { return proto.CompactTextString(m) }
func (*RaftMessageBatchRequest) ProtoMessage()    {}
func (*RaftMessageBatchRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{2}
}
func (m *RaftMessageBatchRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftMessageBatchRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftMessageBatchRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftMessageBatchRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftMessageBatchRequest.Merge(dst, src)
}
func (m *RaftMessageBatchRequest) XXX_Size() int {
	return m.Size()
}
func (m *RaftMessageBatchRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftMessageBatchRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RaftMessageBatchRequest proto.InternalMessageInfo

// RaftMessageBatchResponse A RaftMessageBatchResponse contains one or more responses
type RaftMessageBatchResponse struct {
	Responses            []RaftMessageResponse `protobuf:"bytes,1,rep,name=responses" json:"responses"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *RaftMessageBatchResponse) Reset()         { *m = RaftMessageBatchResponse{} }
func (m *RaftMessageBatchResponse) String() string { return proto.CompactTextString(m) }
func (*RaftMessageBatchResponse) ProtoMessage()    {}
func (*RaftMessageBatchResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{3}
}
func (m *RaftMessageBatchResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftMessageBatchResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftMessageBatchResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftMessageBatchResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftMessageBatchResponse.Merge(dst, src)
}
func (m *RaftMessageBatchResponse) XXX_Size() int {
	return m.Size()
}
func (m *RaftMessageBatchResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftMessageBatchResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RaftMessageBatchResponse proto.InternalMessageInfo

// A RaftCommand is a command which can be serialized and sent via raft.
type RaftCommand struct {
	RangeID              RangeID      `protobuf:"varint,1,opt,name=range_id,json=rangeId,proto3,casttype=RangeID" json:"range_id,omitempty"`
	Request              BatchRequest `protobuf:"bytes,2,opt,name=request" json:"request"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *RaftCommand) Reset()         { *m = RaftCommand{} }
func (m *RaftCommand) String() string { return proto.CompactTextString(m) }
func (*RaftCommand) ProtoMessage()    {}
func (*RaftCommand) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{4}
}
func (m *RaftCommand) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftCommand) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftCommand.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftCommand) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftCommand.Merge(dst, src)
}
func (m *RaftCommand) XXX_Size() int {
	return m.Size()
}
func (m *RaftCommand) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftCommand.DiscardUnknown(m)
}

var xxx_messageInfo_RaftCommand proto.InternalMessageInfo

// raft message
type RaftData struct {
	Id                   string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Cmd                  RaftCommand `protobuf:"bytes,2,opt,name=cmd" json:"cmd"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *RaftData) Reset()         { *m = RaftData{} }
func (m *RaftData) String() string { return proto.CompactTextString(m) }
func (*RaftData) ProtoMessage()    {}
func (*RaftData) Descriptor() ([]byte, []int) {
	return fileDescriptor_raftmetadata_28e5b29587c8e462, []int{5}
}
func (m *RaftData) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaftData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaftData.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RaftData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftData.Merge(dst, src)
}
func (m *RaftData) XXX_Size() int {
	return m.Size()
}
func (m *RaftData) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftData.DiscardUnknown(m)
}

var xxx_messageInfo_RaftData proto.InternalMessageInfo

func init() {
	proto.RegisterType((*RaftMessageRequest)(nil), "ancestor.meta.RaftMessageRequest")
	proto.RegisterType((*RaftMessageResponse)(nil), "ancestor.meta.RaftMessageResponse")
	proto.RegisterType((*RaftMessageBatchRequest)(nil), "ancestor.meta.RaftMessageBatchRequest")
	proto.RegisterType((*RaftMessageBatchResponse)(nil), "ancestor.meta.RaftMessageBatchResponse")
	proto.RegisterType((*RaftCommand)(nil), "ancestor.meta.RaftCommand")
	proto.RegisterType((*RaftData)(nil), "ancestor.meta.RaftData")
}
func (m *RaftMessageRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftMessageRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.RangeID != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintRaftmetadata(dAtA, i, uint64(m.RangeID))
	}
	if m.FromNodeID != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintRaftmetadata(dAtA, i, uint64(m.FromNodeID))
	}
	if m.ToNodeID != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintRaftmetadata(dAtA, i, uint64(m.ToNodeID))
	}
	dAtA[i] = 0x22
	i++
	i = encodeVarintRaftmetadata(dAtA, i, uint64(m.Message.Size()))
	n1, err := m.Message.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	return i, nil
}

func (m *RaftMessageResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftMessageResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	return i, nil
}

func (m *RaftMessageBatchRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftMessageBatchRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Requests) > 0 {
		for _, msg := range m.Requests {
			dAtA[i] = 0xa
			i++
			i = encodeVarintRaftmetadata(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *RaftMessageBatchResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftMessageBatchResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Responses) > 0 {
		for _, msg := range m.Responses {
			dAtA[i] = 0xa
			i++
			i = encodeVarintRaftmetadata(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *RaftCommand) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftCommand) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.RangeID != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintRaftmetadata(dAtA, i, uint64(m.RangeID))
	}
	dAtA[i] = 0x12
	i++
	i = encodeVarintRaftmetadata(dAtA, i, uint64(m.Request.Size()))
	n2, err := m.Request.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	return i, nil
}

func (m *RaftData) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaftData) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Id) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintRaftmetadata(dAtA, i, uint64(len(m.Id)))
		i += copy(dAtA[i:], m.Id)
	}
	dAtA[i] = 0x12
	i++
	i = encodeVarintRaftmetadata(dAtA, i, uint64(m.Cmd.Size()))
	n3, err := m.Cmd.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n3
	return i, nil
}

func encodeVarintRaftmetadata(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *RaftMessageRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RangeID != 0 {
		n += 1 + sovRaftmetadata(uint64(m.RangeID))
	}
	if m.FromNodeID != 0 {
		n += 1 + sovRaftmetadata(uint64(m.FromNodeID))
	}
	if m.ToNodeID != 0 {
		n += 1 + sovRaftmetadata(uint64(m.ToNodeID))
	}
	l = m.Message.Size()
	n += 1 + l + sovRaftmetadata(uint64(l))
	return n
}

func (m *RaftMessageResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *RaftMessageBatchRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Requests) > 0 {
		for _, e := range m.Requests {
			l = e.Size()
			n += 1 + l + sovRaftmetadata(uint64(l))
		}
	}
	return n
}

func (m *RaftMessageBatchResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Responses) > 0 {
		for _, e := range m.Responses {
			l = e.Size()
			n += 1 + l + sovRaftmetadata(uint64(l))
		}
	}
	return n
}

func (m *RaftCommand) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RangeID != 0 {
		n += 1 + sovRaftmetadata(uint64(m.RangeID))
	}
	l = m.Request.Size()
	n += 1 + l + sovRaftmetadata(uint64(l))
	return n
}

func (m *RaftData) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovRaftmetadata(uint64(l))
	}
	l = m.Cmd.Size()
	n += 1 + l + sovRaftmetadata(uint64(l))
	return n
}

func sovRaftmetadata(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozRaftmetadata(x uint64) (n int) {
	return sovRaftmetadata(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *RaftMessageRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftMessageRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftMessageRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeID", wireType)
			}
			m.RangeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return fmt.Errorf("proto: wrong wireType = %d for field FromNodeID", wireType)
			}
			m.FromNodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.FromNodeID |= (NodeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ToNodeID", wireType)
			}
			m.ToNodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ToNodeID |= (NodeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Message.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func (m *RaftMessageResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftMessageResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftMessageResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func (m *RaftMessageBatchRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftMessageBatchRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftMessageBatchRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Requests", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Requests = append(m.Requests, RaftMessageRequest{})
			if err := m.Requests[len(m.Requests)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func (m *RaftMessageBatchResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftMessageBatchResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftMessageBatchResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Responses", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Responses = append(m.Responses, RaftMessageResponse{})
			if err := m.Responses[len(m.Responses)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func (m *RaftCommand) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftCommand: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftCommand: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeID", wireType)
			}
			m.RangeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Request", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Request.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func (m *RaftData) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaftmetadata
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
			return fmt.Errorf("proto: RaftData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Cmd", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaftmetadata
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
				return ErrInvalidLengthRaftmetadata
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Cmd.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaftmetadata(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaftmetadata
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
func skipRaftmetadata(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRaftmetadata
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
					return 0, ErrIntOverflowRaftmetadata
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
					return 0, ErrIntOverflowRaftmetadata
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
				return 0, ErrInvalidLengthRaftmetadata
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowRaftmetadata
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
				next, err := skipRaftmetadata(dAtA[start:])
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
	ErrInvalidLengthRaftmetadata = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRaftmetadata   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("raftmetadata.proto", fileDescriptor_raftmetadata_28e5b29587c8e462) }

var fileDescriptor_raftmetadata_28e5b29587c8e462 = []byte{
	// 437 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x92, 0xbb, 0x8e, 0xd3, 0x40,
	0x14, 0x86, 0x33, 0x49, 0xd8, 0x38, 0x27, 0x5c, 0xa4, 0xe1, 0x66, 0x05, 0xc9, 0x09, 0xae, 0xd2,
	0x60, 0x8b, 0x40, 0x05, 0x9d, 0x77, 0xb5, 0xd2, 0x16, 0x6c, 0x61, 0x51, 0x20, 0x0a, 0x56, 0x13,
	0xcf, 0xc4, 0x9b, 0xc2, 0x3e, 0x61, 0x66, 0xb6, 0xe4, 0x1d, 0x78, 0xac, 0x94, 0x3c, 0x41, 0x04,
	0xa6, 0xe3, 0x11, 0xa8, 0xd0, 0x5c, 0xbc, 0x17, 0x8c, 0xd8, 0xc6, 0xfe, 0x75, 0x7c, 0xbe, 0xf3,
	0x9f, 0x8b, 0x81, 0x4a, 0xb6, 0xd6, 0x95, 0xd0, 0x8c, 0x33, 0xcd, 0x92, 0xad, 0x44, 0x8d, 0xf4,
	0x1e, 0xab, 0x0b, 0xa1, 0x34, 0xca, 0xc4, 0x7c, 0x98, 0xbe, 0x28, 0x37, 0xfa, 0xfc, 0x62, 0x95,
	0x14, 0x58, 0xa5, 0x05, 0x4a, 0x81, 0x2a, 0x15, 0xba, 0xe0, 0xa9, 0x21, 0xed, 0x63, 0xbb, 0xb2,
	0x2f, 0x47, 0x4f, 0xc7, 0x6c, 0xbb, 0xf1, 0xf2, 0x51, 0x89, 0x25, 0x5a, 0x99, 0x1a, 0xe5, 0xa2,
	0xf1, 0x2f, 0x02, 0x34, 0x67, 0x6b, 0xfd, 0x4e, 0x28, 0xc5, 0x4a, 0x91, 0x8b, 0xcf, 0x17, 0x42,
	0x69, 0xfa, 0x12, 0x02, 0xc9, 0xea, 0x52, 0x9c, 0x6d, 0x78, 0x48, 0xe6, 0x64, 0x31, 0xcc, 0x9e,
	0x34, 0xfb, 0xd9, 0x28, 0x37, 0xb1, 0x93, 0xa3, 0xdf, 0x57, 0x32, 0x1f, 0xd9, 0xbc, 0x13, 0x4e,
	0xdf, 0xc0, 0xdd, 0xb5, 0xc4, 0xea, 0xac, 0x46, 0x6e, 0xb1, 0xfe, 0x9c, 0x2c, 0xee, 0x64, 0x61,
	0xb3, 0x9f, 0xc1, 0xb1, 0xc4, 0xea, 0x14, 0xb9, 0x23, 0x0f, 0x9c, 0xca, 0x61, 0xdd, 0x46, 0x39,
	0x7d, 0x0d, 0xa0, 0xf1, 0x92, 0x1c, 0x58, 0xd2, 0x18, 0x06, 0xef, 0xb1, 0xc3, 0x05, 0x1a, 0x3d,
	0x95, 0xc2, 0xa8, 0x72, 0x6d, 0x87, 0xc3, 0x39, 0x59, 0x4c, 0x96, 0x0f, 0x12, 0xb7, 0x81, 0xc4,
	0x4f, 0x93, 0x0d, 0x77, 0xfb, 0x59, 0x2f, 0x6f, 0xb3, 0xe2, 0xc7, 0xf0, 0xf0, 0xc6, 0xac, 0x6a,
	0x8b, 0xb5, 0x12, 0xf1, 0x27, 0x78, 0x7a, 0x2d, 0x9c, 0x31, 0x5d, 0x9c, 0xb7, 0x7b, 0x38, 0x84,
	0x40, 0x3a, 0xa9, 0x42, 0x32, 0x1f, 0x2c, 0x26, 0xcb, 0xe7, 0xc9, 0x8d, 0x83, 0x24, 0xdd, 0xe5,
	0x79, 0xd7, 0x4b, 0x30, 0x5e, 0x41, 0xd8, 0xad, 0xef, 0xbc, 0xe9, 0x31, 0x8c, 0xa5, 0xd7, 0xad,
	0x43, 0xfc, 0x3f, 0x07, 0x97, 0xea, 0x2d, 0xae, 0xd0, 0xf8, 0x0b, 0x4c, 0x4c, 0xde, 0x21, 0x56,
	0x15, 0xab, 0x79, 0xe7, 0x7e, 0x83, 0xdb, 0xef, 0xf7, 0x16, 0x46, 0xbe, 0x63, 0x7b, 0xba, 0xc9,
	0xf2, 0xd9, 0x5f, 0x7d, 0x5c, 0x5f, 0x4c, 0xbb, 0x59, 0x4f, 0xc4, 0xa7, 0x10, 0x18, 0xfb, 0x23,
	0xa6, 0x19, 0xbd, 0x0f, 0x7d, 0xef, 0x3a, 0xce, 0xfb, 0x1b, 0x4e, 0x97, 0x30, 0x28, 0x2a, 0xee,
	0x8b, 0x4e, 0xff, 0x31, 0x9c, 0x6f, 0xda, 0xd7, 0x34, 0xc9, 0xd9, 0x74, 0xf7, 0x23, 0xea, 0xed,
	0x9a, 0x88, 0x7c, 0x6b, 0x22, 0xf2, 0xbd, 0x89, 0xc8, 0xd7, 0x9f, 0x51, 0xef, 0xe3, 0xd0, 0x20,
	0x1f, 0xfa, 0xab, 0x03, 0xfb, 0xef, 0xbe, 0xfa, 0x13, 0x00, 0x00, 0xff, 0xff, 0xbc, 0xe9, 0x66,
	0x05, 0x30, 0x03, 0x00, 0x00,
}
