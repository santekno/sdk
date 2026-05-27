package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/santekno/sdk/httpx"
)

func TestNew_Defaults(t *testing.T) {
	c := httpx.New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := httpx.New(httpx.WithTimeout(5 * time.Second))
	var out struct {
		Status string `json:"status"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
	if out.Status != "ok" {
		t.Errorf("Status = %q, want ok", out.Status)
	}
}

func TestGetJSON_RetriesOn503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := httpx.New(httpx.WithTimeout(5*time.Second), httpx.WithRetries(3))
	var out struct {
		Status string `json:"status"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestGetJSON_Non2xxReturnsErrNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := httpx.New()
	var out map[string]any
	err := c.GetJSON(context.Background(), srv.URL, &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, httpx.ErrNon2xx) {
		t.Errorf("expected ErrNon2xx, got %v", err)
	}
}

func TestPostJSON_SendsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		_ = json.NewEncoder(w).Encode(map[string]string{"echo": in["msg"]})
	}))
	defer srv.Close()

	c := httpx.New()
	var out struct {
		Echo string `json:"echo"`
	}
	err := c.PostJSON(context.Background(), srv.URL,
		map[string]string{"msg": "hello"}, &out)
	if err != nil {
		t.Fatalf("PostJSON error: %v", err)
	}
	if out.Echo != "hello" {
		t.Errorf("Echo = %q, want hello", out.Echo)
	}
}

func TestBearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer mytoken" {
			t.Errorf("Authorization = %q, want Bearer mytoken", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := httpx.New(httpx.WithBearerToken("mytoken"))
	var out map[string]string
	_ = c.GetJSON(context.Background(), srv.URL, &out)
}

func TestDefaultHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "v1" {
			t.Errorf("X-Test = %q, want v1", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := httpx.New(httpx.WithDefaultHeader("X-Test", "v1"))
	var out map[string]string
	_ = c.GetJSON(context.Background(), srv.URL, &out)
}

func ExampleClient_GetJSON() {
	c := httpx.New(httpx.WithTimeout(5 * time.Second))
	var result struct {
		Status string `json:"status"`
	}
	_ = c.GetJSON(context.Background(), "https://api.example.com/health", &result)
}
