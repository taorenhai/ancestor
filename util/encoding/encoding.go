package encoding

import (
	"bytes"
	"fmt"
)

// EncodeUint32 encodes the uint32 value using a big-endian 8 byte
// representation. The bytes are appended to the supplied buffer and
// the final buffer is returned.
func EncodeUint32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// EncodeUint32Decreasing encodes the uint32 value so that it sorts in
// reverse order, from largest to smallest.
func EncodeUint32Decreasing(b []byte, v uint32) []byte {
	return EncodeUint32(b, ^v)
}

// DecodeUint32 decodes a uint32 from the input buffer, treating
// the input as a big-endian 8 byte uint32 representation. The remainder
// of the input buffer and the decoded uint32 are returned.
func DecodeUint32(b []byte) ([]byte, uint32, error) {
	if len(b) < 4 {
		//return nil, 0, util.Errorf("insufficient bytes to decode uint32 int value")
		return nil, 0, fmt.Errorf("insufficient bytes to decode uint32 int value")
	}
	v := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) |
		(uint32(b[2]) << 8) | uint32(b[3])
	return b[4:], v, nil
}

// DecodeUint32Decreasing decodes a uint32 value which was encoded
// using EncodeUint32Decreasing.
func DecodeUint32Decreasing(b []byte) ([]byte, uint32, error) {
	leftover, v, err := DecodeUint32(b)
	return leftover, ^v, err
}

// EncodeUint64 encodes the uint64 value using a big-endian 8 byte
// representation. The bytes are appended to the supplied buffer and
// the final buffer is returned.
func EncodeUint64(b []byte, v uint64) []byte {
	return append(b,
		byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32),
		byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// EncodeUint64Decreasing encodes the uint64 value so that it sorts in
// reverse order, from largest to smallest.
func EncodeUint64Decreasing(b []byte, v uint64) []byte {
	return EncodeUint64(b, ^v)
}

// DecodeUint64 decodes a uint64 from the input buffer, treating
// the input as a big-endian 8 byte uint64 representation. The remainder
// of the input buffer and the decoded uint64 are returned.
func DecodeUint64(b []byte) ([]byte, uint64, error) {
	if len(b) < 8 {
		//return nil, 0, util.Errorf("insufficient bytes to decode uint64 int value")
		return nil, 0, fmt.Errorf("insufficient bytes to decode uint64 int value")
	}
	v := (uint64(b[0]) << 56) | (uint64(b[1]) << 48) |
		(uint64(b[2]) << 40) | (uint64(b[3]) << 32) |
		(uint64(b[4]) << 24) | (uint64(b[5]) << 16) |
		(uint64(b[6]) << 8) | uint64(b[7])
	return b[8:], v, nil
}

// DecodeUint64Decreasing decodes a uint64 value which was encoded
// using EncodeUint64Decreasing.
func DecodeUint64Decreasing(b []byte) ([]byte, uint64, error) {
	leftover, v, err := DecodeUint64(b)
	return leftover, ^v, err
}

const (
	// <term>     -> \x00\x01
	// \x00       -> \x00\xff
	escape      byte = 0x00
	escapedTerm byte = 0x01
	escaped00   byte = 0xff
	escapedFF   byte = 0x00
)

type escapes struct {
	escape      byte
	escapedTerm byte
	escaped00   byte
	escapedFF   byte
}

var (
	ascendingEscapes  = escapes{escape, escapedTerm, escaped00, escapedFF}
	descendingEscapes = escapes{^escape, ^escapedTerm, ^escaped00, ^escapedFF}
)

// EncodeBytes encodes the []byte value using an escape-based
// encoding. The encoded value is terminated with the sequence
// "\x00\x01" which is guaranteed to not occur elsewhere in the
// encoded value. The encoded bytes are append to the supplied buffer
// and the resulting buffer is returned.
func EncodeBytes(b []byte, data []byte) []byte {
	for {
		// IndexByte is implemented by the go runtime in assembly and is
		// much faster than looping over the bytes in the slice.
		i := bytes.IndexByte(data, escape)
		if i == -1 {
			break
		}
		b = append(b, data[:i]...)
		b = append(b, escape, escaped00)
		data = data[i+1:]
	}
	b = append(b, data...)
	return append(b, escape, escapedTerm)
}

func decodeBytes(b []byte, r []byte, e escapes) ([]byte, []byte, error) {
	if len(b) == 0 {
		return nil, nil, nil
	}

	for {
		i := bytes.IndexByte(b, e.escape)
		if i == -1 {
			//return nil, nil, util.Errorf("did not find terminator")
			return nil, nil, fmt.Errorf("did not find terminator")
		}
		if i+1 >= len(b) {
			//return nil, nil, util.Errorf("malformed escape")
			return nil, nil, fmt.Errorf("malformed escape")
		}

		v := b[i+1]
		if v == e.escapedTerm {
			if r == nil {
				r = b[:i]
			} else {
				r = append(r, b[:i]...)
			}
			return b[i+2:], r, nil
		}

		if v == e.escaped00 {
			r = append(r, b[:i]...)
			r = append(r, e.escapedFF)
		} else {
			//return nil, nil, util.Errorf("unknown escape")
			return nil, nil, fmt.Errorf("unknown escape")
		}

		b = b[i+2:]
	}
}

// DecodeBytes decodes a []byte value from the input buffer which was
// encoded using EncodeBytes. The decoded bytes are appended to r. The
// remainder of the input buffer and the decoded []byte are returned.
func DecodeBytes(b []byte, r []byte) ([]byte, []byte, error) {
	return decodeBytes(b, r, ascendingEscapes)
}
