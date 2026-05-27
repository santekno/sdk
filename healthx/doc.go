// Package healthx provides liveness and readiness HTTP handlers.
//
// A liveness probe MUST pass while the process is alive. A readiness probe
// MUST pass only when all dependencies are reachable.
//
//	h := &healthx.Handler{
//	    Liveness:  []healthx.Check{healthx.AlwaysOK()},
//	    Readiness: []healthx.Check{healthx.PingHTTP("http://upstream/health")},
//	}
//	mux.HandleFunc("/livez",  h.LivenessHandler)
//	mux.HandleFunc("/readyz", h.ReadinessHandler)
package healthx
