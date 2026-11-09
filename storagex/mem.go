package storagex

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type memObject struct {
	data        []byte
	contentType string
	etag        string
	modTime     time.Time
}

// MemStorage stores content in memory. Returned URLs from Put are the key itself.
// Not suitable for production; intended for tests and in-process previews.
// Keys are checked with ValidateKey like every other backend.
type MemStorage struct {
	mu   sync.RWMutex
	data map[string]memObject
	opts Options
}

// NewMemStorage returns an empty MemStorage.
func NewMemStorage(opts ...Option) *MemStorage {
	return &MemStorage{data: map[string]memObject{}, opts: NewOptions(opts...)}
}

// Put stores r under key and returns key as the URL.
func (m *MemStorage) Put(ctx context.Context, key, contentType string, r io.Reader) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	if m.opts.NoOverwrite && m.exists(key) {
		return "", ErrAlreadyExists
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	obj := memObject{data: b, contentType: contentType, etag: hex.EncodeToString(sum[:16]), modTime: time.Now().UTC()}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok && m.opts.NoOverwrite {
		return "", ErrAlreadyExists
	}
	m.data[key] = obj
	return key, nil
}

func (m *MemStorage) exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

// Get returns the content stored under key, or an error matching ErrNotFound.
func (m *MemStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	m.mu.RLock()
	obj, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(obj.data)), nil
}

// Delete removes the blob stored under key. Deleting a missing key succeeds.
func (m *MemStorage) Delete(_ context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}

// Stat describes the object under key, or returns an error matching ErrNotFound.
func (m *MemStorage) Stat(_ context.Context, key string) (ObjectInfo, error) {
	if err := ValidateKey(key); err != nil {
		return ObjectInfo{}, err
	}
	m.mu.RLock()
	obj, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return ObjectInfo{}, ErrNotFound
	}
	return obj.info(key), nil
}

func (o memObject) info(key string) ObjectInfo {
	return ObjectInfo{Key: key, Size: int64(len(o.data)), ContentType: o.contentType, ETag: o.etag, LastModified: o.modTime}
}

// List returns one page of objects whose key starts with opts.Prefix, sorted
// by key. NextToken is the last key of the page.
func (m *MemStorage) List(_ context.Context, opts ListOptions) (ListPage, error) {
	if err := ValidatePrefix(opts.Prefix); err != nil {
		return ListPage{}, err
	}
	m.mu.RLock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		if strings.HasPrefix(k, opts.Prefix) && k > opts.Token {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	limit := opts.PageLimit()
	var page ListPage
	for i, k := range keys {
		if i == limit {
			page.NextToken = keys[i-1]
			break
		}
		page.Objects = append(page.Objects, m.data[k].info(k))
	}
	m.mu.RUnlock()
	return page, nil
}
