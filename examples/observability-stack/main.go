// Example: observability-stack shows tracex, logx, httpx, and healthx wired together.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/santekno/sdk/healthx"
	"github.com/santekno/sdk/httpx"
	"github.com/santekno/sdk/logx"
	"github.com/santekno/sdk/tracex"
)

func main() {
	ctx := context.Background()

	// 1. Initialize structured logging.
	logger := logx.New(logx.WithFormat("console"), logx.WithLevel("debug"))
	logger.Info("observability stack starting")

	// 2. Set up a no-op tracer (swap for OTel in production).
	tracer := tracex.NoopTracer()
	tracex.SetGlobal(tracer)
	spanCtx, span := tracer.Start(ctx, "observability-demo.main")
	defer span.End()
	ctx = spanCtx

	// 3. Create an HTTP client with logger attached.
	client := httpx.New(
		httpx.WithTimeout(5*time.Second),
		httpx.WithLogger(logger),
	)

	// 4. Make a test request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"ok":true}`)
	}))
	defer srv.Close()

	var result map[string]bool
	if err := client.GetJSON(ctx, srv.URL, &result); err != nil {
		log.Fatalf("GetJSON: %v", err)
	}
	fmt.Printf("HTTP response: %v\n", result)

	// 5. Set up health checks.
	h := &healthx.Handler{}
	h.Readiness = append(h.Readiness, healthx.NamedCheck{
		Name: "self",
		Check: func(ctx context.Context) error {
			return nil
		},
	})

	healthSrv := httptest.NewServer(http.HandlerFunc(h.ReadinessHandler))
	defer healthSrv.Close()

	resp, err := http.Get(healthSrv.URL) //nolint:noctx
	if err != nil {
		log.Fatalf("health check: %v", err)
	}
	defer resp.Body.Close()
	fmt.Printf("Health status: %d\n", resp.StatusCode)

	logger.Info("observability stack demo complete")
}
