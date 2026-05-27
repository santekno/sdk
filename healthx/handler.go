package healthx

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// checkResult is the JSON shape returned for each individual check.
type checkResult struct {
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// response is the top-level JSON shape returned to the caller.
type response struct {
	Status string                 `json:"status"`
	Checks map[string]checkResult `json:"checks,omitempty"`
}

// LivenessHandler serves the liveness probe. Returns 200 if all Liveness
// checks pass, 503 otherwise.
func (h *Handler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, h.Liveness)
}

// ReadinessHandler serves the readiness probe. Returns 200 if all Readiness
// checks pass, 503 otherwise.
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, h.Readiness)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, checks []NamedCheck) {
	timeout := h.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	resp := response{Status: "ok", Checks: make(map[string]checkResult, len(checks))}
	overallOK := true

	for _, c := range checks {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		start := time.Now()
		err := c.Check(ctx)
		cancel()
		dur := time.Since(start).Milliseconds()

		if err != nil {
			overallOK = false
			resp.Checks[c.Name] = checkResult{Status: "unavailable", Error: err.Error(), DurationMs: dur}
		} else {
			resp.Checks[c.Name] = checkResult{Status: "ok", DurationMs: dur}
		}
	}

	if !overallOK {
		resp.Status = "unavailable"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
