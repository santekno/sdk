package storagex

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMemStorage(t *testing.T) {
	s := NewMemStorage()
	ctx := context.Background()

	url, err := s.Put(ctx, "report.csv", "text/csv", strings.NewReader("a,b,c"))
	if err != nil {
		t.Fatal(err)
	}
	if url != "report.csv" {
		t.Errorf("url = %s, want report.csv", url)
	}

	r, err := s.Get(ctx, "report.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	if string(b) != "a,b,c" {
		t.Errorf("got %q, want %q", string(b), "a,b,c")
	}

	if err := s.Delete(ctx, "report.csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "report.csv"); err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestLocalStorage(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()

	path, err := s.Put(ctx, "report.csv", "text/csv", strings.NewReader("x,y,z"))
	if err != nil {
		t.Fatal(err)
	}

	r, err := s.Get(ctx, "report.csv")
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	b, _ := io.ReadAll(r)
	r.Close() // close before Delete — Windows holds a lock until the handle is released
	if string(b) != "x,y,z" {
		t.Errorf("got %q, want %q", string(b), "x,y,z")
	}

	if err := s.Delete(ctx, "report.csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

var _ Storage = (*MemStorage)(nil)
var _ Storage = (*LocalStorage)(nil)
