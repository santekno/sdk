package healthx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santekno/sdk/healthx"
)

func TestHandler_ContentTypeIsJSON(t *testing.T) {
	cases := []struct {
		name string
		h    *healthx.Handler
		want int
	}{
		{"ok", &healthx.Handler{Readiness: []healthx.NamedCheck{{Name: "db", Check: healthx.AlwaysOK()}}}, http.StatusOK},
		{"failing", &healthx.Handler{Readiness: []healthx.NamedCheck{{Name: "db", Check: healthx.AlwaysFail("down")}}}, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.h.ReadinessHandler(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
			if ct := w.Result().Header.Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestHandler_HideErrors(t *testing.T) {
	checks := []healthx.NamedCheck{{Name: "db", Check: healthx.AlwaysFail("dial tcp 10.0.0.5:5432: connection refused")}}
	for _, hide := range []bool{false, true} {
		w := httptest.NewRecorder()
		h := &healthx.Handler{Readiness: checks, HideErrors: hide}
		h.ReadinessHandler(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		body := w.Body.String()
		leaked := strings.Contains(body, "10.0.0.5")
		if hide && leaked {
			t.Fatalf("HideErrors=true leaked the check error: %s", body)
		}
		if !hide && !leaked {
			t.Fatalf("HideErrors=false dropped the check error: %s", body)
		}
		if !strings.Contains(body, `"unavailable"`) {
			t.Fatalf("body lacks the unavailable status: %s", body)
		}
	}
}
