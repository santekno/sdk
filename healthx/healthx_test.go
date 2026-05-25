package healthx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/santekno/sdk/healthx"
)

func TestLivenessHandler_AllOK(t *testing.T) {
	h := &healthx.Handler{
		Liveness: []healthx.NamedCheck{
			{Name: "self", Check: healthx.AlwaysOK()},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/livez", nil)
	h.LivenessHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Status string                    `json:"status"`
		Checks map[string]map[string]any `json:"checks"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

func TestReadinessHandler_OneFails(t *testing.T) {
	h := &healthx.Handler{
		Readiness: []healthx.NamedCheck{
			{Name: "good", Check: healthx.AlwaysOK()},
			{Name: "bad", Check: healthx.AlwaysFail("upstream down")},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	h.ReadinessHandler(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
	var resp struct {
		Status string                    `json:"status"`
		Checks map[string]map[string]any `json:"checks"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Checks["bad"]["error"] != "upstream down" {
		t.Errorf("bad check error = %v", resp.Checks["bad"]["error"])
	}
}

func TestPingHTTP_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	check := healthx.PingHTTP(srv.URL)
	if err := check(context.Background()); err != nil {
		t.Errorf("PingHTTP should succeed: %v", err)
	}
}

func TestPingHTTP_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	check := healthx.PingHTTP(srv.URL)
	if err := check(context.Background()); err == nil {
		t.Error("PingHTTP should fail on 500")
	}
}

type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func TestPingPinger(t *testing.T) {
	ok := healthx.PingPinger(fakePinger{})
	if err := ok(context.Background()); err != nil {
		t.Errorf("PingPinger success path: %v", err)
	}
}
