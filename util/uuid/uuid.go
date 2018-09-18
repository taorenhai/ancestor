package uuid

import (
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// UUIDSize is the size in bytes of a UUID.
	UUIDSize = 16
)

// UUID is a 16 byte UUID.
type UUID []byte

// NewUUID4 returns a new UUID (Version 4) using 16 random bytes or panics.
//
// The uniqueness depends on the strength of crypto/rand. Version 4
// UUIDs have 122 random bits.
func NewUUID4() UUID {
	uuid := make([]byte, UUIDSize)
	if _, err := io.ReadFull(rand.Reader, uuid); err != nil {
		panic(err) // rand should never fail
	}
	// UUID (Version 4) compliance.
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return uuid
}

// String formats as hex xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,
// or "" if u is invalid.
func (u UUID) String() string {
	if len(u) != UUIDSize {
		return ""
	}
	b := []byte(u)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Short formats the UUID using only the first four bytes for brevity.
func (u UUID) Short() string {
	if len(u) != UUIDSize {
		return ""
	}
	b := []byte(u)
	return fmt.Sprintf("%08x", b[:4])
}
