// Package uuidx provides UUID and ULID generation helpers.
package uuidx

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

// NewUUID returns a randomly-generated UUID v4 string in canonical
// 8-4-4-4-12 hex format.
func NewUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return format(b)
}

// NewV7 returns a UUID version 7 (RFC 9562) string: a 48-bit Unix millisecond
// timestamp followed by 74 random bits. IDs sort by creation time to the
// millisecond, which keeps B-tree primary keys append-friendly; IDs created
// within the same millisecond are in random order relative to each other.
func NewV7() string {
	return newV7(time.Now())
}

func newV7(now time.Time) string {
	var b [16]byte
	_, _ = rand.Read(b[6:])
	ms := uint64(now.UnixMilli()) // #nosec G115 -- Unix ms is positive for any real clock
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], ms)
	copy(b[0:6], ts[2:8])
	b[6] = (b[6] & 0x0f) | 0x70 // version 7
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 9562 variant
	return format(b)
}

// format renders 16 bytes in the canonical 8-4-4-4-12 lower-case hex form.
func format(b [16]byte) string {
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}

// IsValidUUID reports whether s is a syntactically valid 36-character UUID string.
func IsValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if r > 127 || !isHex(byte(r)) { // #nosec G115 -- ASCII-guarded by r>127 check
			return false
		}
	}
	return true
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// NewShortID returns a random 12-character hex identifier (48 bits of entropy).
// Useful for non-cryptographic short IDs (request IDs, correlation keys).
func NewShortID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("uuidx: rand.Read: %w", err))
	}
	return hex.EncodeToString(b[:])
}
