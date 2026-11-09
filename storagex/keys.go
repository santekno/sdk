package storagex

import (
	"strings"
	"unicode/utf8"
)

// MaxKeyLen is the longest key in bytes, the S3 object key limit.
const MaxKeyLen = 1024

// tmpPrefix marks LocalStorage temporary files. Keys with a segment that starts
// with it are rejected, so a temporary file is never read or listed as an object.
const tmpPrefix = ".storagex-tmp-"

// ValidateKey reports whether key is a safe object key: a relative,
// slash-separated path of at most MaxKeyLen bytes of valid UTF-8, without a
// backslash or a control character, and without an empty (leading "/",
// "a//b", trailing "/"), ".", "..", or ".storagex-tmp-*" segment. Every
// backend applies it, so a key accepted by one is accepted by all. It returns
// ErrInvalidKey otherwise, without echoing the key.
func ValidateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	return validatePath(key, false)
}

// ValidatePrefix reports whether prefix can select keys in a listing: empty,
// or a key as accepted by ValidateKey, optionally ending in "/" (a directory
// prefix such as "v/"). A prefix may end in a partial segment ("v/a" matches
// "v/a.webp" and "v/ab/c").
func ValidatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	return validatePath(prefix, true)
}

func validatePath(p string, allowTrailingSlash bool) error {
	if len(p) > MaxKeyLen || !utf8.ValidString(p) {
		return ErrInvalidKey
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return ErrInvalidKey
		}
	}
	segs := strings.Split(p, "/")
	if allowTrailingSlash && len(segs) > 1 && segs[len(segs)-1] == "" {
		segs = segs[:len(segs)-1]
	}
	for _, seg := range segs {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, tmpPrefix) {
			return ErrInvalidKey
		}
	}
	return nil
}
