// Example: http-client-resilience demonstrates httpx.Client with automatic
// retry, circuit breaker, and configurable timeouts.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/santekno/sdk/httpx"
	"github.com/santekno/sdk/logx"
)

func main() {
	// A demo server that fails twice then succeeds.
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	logger := logx.New(logx.WithLevel("debug"), logx.WithFormat("console"))

	client := httpx.New(
		httpx.WithTimeout(5*time.Second),
		httpx.WithRetries(3),
		httpx.WithLogger(logger),
	)

	var result map[string]string
	ctx := context.Background()
	if err := client.GetJSON(ctx, srv.URL, &result); err != nil {
		log.Fatalf("GetJSON failed: %v", err)
	}

	fmt.Printf("Response: %v (after %d attempts)\n", result, attempts.Load())
}
