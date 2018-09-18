package meta

import (
	"bytes"
	"fmt"
	"math"

	"github.com/biogo/store/llrb"
	proto "github.com/gogo/protobuf/proto"

	"github.com/taorenhai/ancestor/util/encoding"
)

const (
	// EncodeTimestampSize mvcc timestamp length.
	EncodeTimestampSize = 12
)

const (
	//TxnChangeTypeHeartbeat only change lastHeartbeat field
	TxnChangeTypeHeartbeat = 1 << iota

	//TxnChangeTypeBegin set whatever it requests
	TxnChangeTypeBegin

	//TxnChangeTypeEnd only change txn status and timestamp
	TxnChangeTypeEnd

	//TxnChangeTypePushTimestamp only change txn timestamp
	TxnChangeTypePushTimestamp

	//TxnChangeTypePushAbort only change txn status
	TxnChangeTypePushAbort
)

// Less compares two timestamps.
func (t Timestamp) Less(s Timestamp) bool {
	return t.WallTime < s.WallTime || (t.WallTime == s.WallTime && t.Logical < s.Logical)
}

// Large compares two timestamps.
func (t Timestamp) Large(s Timestamp) bool {
	return t.WallTime > s.WallTime || (t.WallTime == s.WallTime && t.Logical > s.Logical)
}

// Equal returns whether two timestamps are the same.
func (t Timestamp) Equal(s Timestamp) bool {
	return t.WallTime == s.WallTime && t.Logical == s.Logical
}

// Next returns the timestamp with the next later timestamp.
func (t *Timestamp) Next() Timestamp {
	if t.Logical == math.MaxInt32 {
		if t.WallTime == math.MaxInt64 {
			panic("cannot take the next value to a max timestamp")
		}
		return Timestamp{
			WallTime: t.WallTime + 1,
		}
	}
	return Timestamp{
		WallTime: t.WallTime,
		Logical:  t.Logical + 1,
	}
}

// Key is a custom type for a byte string in proto
// messages which refer to ancestor keys.
type Key []byte

// BytesNext returns the next possible byte by appending an \x00.
func BytesNext(b []byte) []byte {
	return append(append([]byte(nil), b...), 0)
}

// NewKey return a Key
func NewKey(bytes ...[]byte) Key {
	var k Key
	for _, b := range bytes {
		k = append(k, b...)
	}
	return k
}

// Next returns the next key in lexicographic sort order.
func (k Key) Next() Key {
	return Key(BytesNext(k))
}

// Less compares two keys.
func (k Key) Less(l Key) bool {
	return bytes.Compare(k, l) < 0
}

// Compare implements the interval.Comparable interface for tree nodes.
func (k Key) Compare(b llrb.Comparable) int {
	return bytes.Compare(k, b.(Key))
}

// String returns a string-formatted version of the key.
func (k Key) String() string {
	return fmt.Sprintf("%q", []byte(k))
}

// SetBytes sets the bytes and tag field of the receiver.
func (v *Value) SetBytes(b []byte) {
	v.Bytes = b
	v.Tag = ValueType_BYTES
}

// SetFloat encodes the specified float64 value into the bytes field of the
// receiver and sets the tag.
func (v *Value) SetFloat(f float64) {
	v.Bytes = encoding.EncodeUint64(nil, math.Float64bits(f))
	v.Tag = ValueType_FLOAT
}

// SetInt encodes the specified int64 value into the bytes field of the
// receiver and sets the tag.
func (v *Value) SetInt(i int64) {
	v.Bytes = encoding.EncodeUint64(nil, uint64(i))
	v.Tag = ValueType_INT
}

// SetProto encodes the specified proto message into the bytes field of the receiver.
func (v *Value) SetProto(msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	v.SetBytes(data)
	return nil
}

// GetBytes returns the bytes field of the receiver. If the tag is not
// BYTES an error will be returned.
func (v Value) GetBytes() ([]byte, error) {
	if tag := v.Tag; tag != ValueType_BYTES {
		return nil, fmt.Errorf("value type is not %s: %s", ValueType_BYTES, tag)
	}
	return v.Bytes, nil
}

// GetFloat decodes a float64 value from the bytes field of the receiver. If
// the bytes field is not 8 bytes in length or the tag is not FLOAT an error
// will be returned.
func (v Value) GetFloat() (float64, error) {
	if tag := v.Tag; tag != ValueType_FLOAT {
		return 0, fmt.Errorf("value type is not %s: %s", ValueType_FLOAT, tag)
	}
	if len(v.Bytes) != 8 {
		return 0, fmt.Errorf("float64 value should be exactly 8 bytes: %d", len(v.Bytes))
	}
	_, u, err := encoding.DecodeUint64(v.Bytes)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(u), nil
}

// GetInt decodes an int64 value from the bytes field of the receiver. If the
// bytes field is not 8 bytes in length or the tag is not INT an error will be
// returned.
func (v Value) GetInt() (int64, error) {
	if tag := v.Tag; tag != ValueType_INT {
		return 0, fmt.Errorf("value type is not %s: %s", ValueType_INT, tag)
	}
	if len(v.Bytes) != 8 {
		return 0, fmt.Errorf("uint64 value should be exactly 8 bytes: %d", len(v.Bytes))
	}
	_, u, err := encoding.DecodeUint64(v.Bytes)
	if err != nil {
		return 0, err
	}
	return int64(u), nil
}

// GetProto unmarshals the bytes field of the receiver into msg. If
// unmarshalling fails or the tag is not BYTES, an error will be
// returned.
func (v Value) GetProto(msg proto.Message) error {
	expectedTag := ValueType_BYTES
	if tag := v.Tag; tag != expectedTag {
		return fmt.Errorf("value type is not %s: %s", expectedTag, tag)
	}
	return proto.Unmarshal(v.Bytes, msg)
}

// MVCCKey is an encoded key, distinguished from Key in that it
// is an encoded version.
type MVCCKey []byte

// Next returns the next key in lexicographic sort order.
func (k MVCCKey) Next() MVCCKey {
	return MVCCKey(BytesNext(k))
}

// Less compares two keys.
func (k MVCCKey) Less(l MVCCKey) bool {
	return bytes.Compare(k, l) < 0
}

// Equal returns whether two keys are identical.
func (k MVCCKey) Equal(l MVCCKey) bool {
	return bytes.Equal(k, l)
}

// String returns a string-formatted version of the key.
func (k MVCCKey) String() string {
	return fmt.Sprintf("%q", []byte(k))
}

// NewMVCCKey encode the key to MVCCKey without timestamp
func NewMVCCKey(key Key) MVCCKey {
	return encoding.EncodeBytes(nil, key)
}

// NewMVCCTimeKey encode the key to MVCCKey with timestamp
func NewMVCCTimeKey(k Key, ts Timestamp) MVCCKey {
	key := encoding.EncodeBytes(nil, k)
	key = encoding.EncodeUint64Decreasing(key, uint64(ts.WallTime))
	key = encoding.EncodeUint32Decreasing(key, uint32(ts.Logical))
	return key
}

// Decode decode mvccKey
func (k MVCCKey) Decode() (key Key, ts Timestamp, ok bool, err error) {
	var (
		buf []byte
		wt  uint64
		lt  uint32
	)

	if buf, key, err = encoding.DecodeBytes(k, nil); err != nil {
		return
	}

	if len(buf) == 0 {
		return
	}

	if len(buf) != EncodeTimestampSize {
		err = fmt.Errorf("there should be 12 bytes for encoded timestamp: %q", ts)
		return
	}

	if buf, wt, err = encoding.DecodeUint64Decreasing(buf); err != nil {
		return
	}

	if _, lt, err = encoding.DecodeUint32Decreasing(buf); err != nil {
		return
	}

	ts.WallTime = int64(wt)
	ts.Logical = int32(lt)

	ok = true

	return
}
