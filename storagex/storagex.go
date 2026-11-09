// Package storagex provides a Storage abstraction for binary blobs (T016,
// Constitution II). MemStorage is provided for tests; LocalStorage writes to the
// local filesystem (see LocalStorage for the key rules). A cloud (S3/Supabase)
// implementation can be added later.
package storagex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// Storage stores and retrieves binary blobs by key.
type Storage interface {
	// Put writes r under key and returns a URL (or path) to the stored blob.
	Put(ctx context.Context, key, contentType string, r io.Reader) (url string, err error)
	// Get returns a reader for the blob stored under key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the blob stored under key.
	Delete(ctx context.Context, key string) error
}

// MemStorage stores content in memory. Returned URLs from Put are the key itself.
// Not suitable for production; intended for tests and in-process previews.
type MemStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemStorage returns an empty MemStorage.
func NewMemStorage() *MemStorage { return &MemStorage{data: map[string][]byte{}} }

// Put stores r under key and returns key as the URL.
func (m *MemStorage) Put(_ context.Context, key, _ string, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.data[key] = b
	m.mu.Unlock()
	return key, nil
}

// Get returns the content stored under key.
func (m *MemStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.RLock()
	b, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("storagex: %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// Delete removes the blob stored under key.
func (m *MemStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}
