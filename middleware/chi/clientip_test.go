package chi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/santekno/sdk/middleware/chi"
)

func TestClientIP(t *testing.T) {
	const remote = "198.51.100.7:4321"
	tests := []struct {
		name    string
		trusted string
		headers [][2]string // added in order, so repeated names make multi-value headers
		remote  string
		want    string
	}{
		{name: "no trusted header uses RemoteAddr", remote: remote, want: "198.51.100.7"},
		{name: "XFF ignored when unconfigured", remote: remote, headers: [][2]string{{"X-Forwarded-For", "1.2.3.4"}}, want: "198.51.100.7"},
		{name: "True-Client-IP ignored when unconfigured", remote: remote, headers: [][2]string{{"True-Client-IP", "1.2.3.4"}}, want: "198.51.100.7"},
		{name: "X-Real-IP ignored when unconfigured", remote: remote, headers: [][2]string{{"X-Real-IP", "1.2.3.4"}}, want: "198.51.100.7"},
		{name: "XFF and TCIP ignored when other header trusted", trusted: "X-Real-IP", remote: remote,
			headers: [][2]string{{"X-Forwarded-For", "1.2.3.4"}, {"True-Client-IP", "5.6.7.8"}, {"X-Real-IP", "203.0.113.9"}}, want: "203.0.113.9"},
		{name: "trusted header absent falls back", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Forwarded-For", "1.2.3.4"}}, want: "198.51.100.7"},
		{name: "trusted header name case-insensitive", trusted: "x-real-ip", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9"}}, want: "203.0.113.9"},
		{name: "trusted name with spaces", trusted: " X-Real-IP ", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9"}}, want: "203.0.113.9"},
		{name: "surrounding spaces trimmed", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "  203.0.113.9 "}}, want: "203.0.113.9"},
		{name: "ipv4 with port", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9:5555"}}, want: "203.0.113.9"},
		{name: "ipv6", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "2001:DB8::1"}}, want: "2001:db8::1"},
		{name: "ipv6 bracketed with port", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "[2001:db8::1]:443"}}, want: "2001:db8::1"},
		{name: "ipv6 bracketed no port", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "[2001:db8::1]"}}, want: "2001:db8::1"},
		{name: "ipv4-mapped ipv6 normalized", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "::ffff:203.0.113.9"}}, want: "203.0.113.9"},
		{name: "comma list rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9, 1.2.3.4"}}, want: "198.51.100.7"},
		{name: "repeated header rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9"}, {"X-Real-IP", "1.2.3.4"}}, want: "198.51.100.7"},
		{name: "zone rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "fe80::1%eth0"}}, want: "198.51.100.7"},
		{name: "embedded space rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "203.0.113.9 1.2.3.4"}}, want: "198.51.100.7"},
		{name: "hostname rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "example.com"}}, want: "198.51.100.7"},
		{name: "garbage rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "999.1.1.1"}}, want: "198.51.100.7"},
		{name: "empty value rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", ""}}, want: "198.51.100.7"},
		{name: "bad bracket rejected", trusted: "X-Real-IP", remote: remote, headers: [][2]string{{"X-Real-IP", "[nope]"}}, want: "198.51.100.7"},
		{name: "ipv6 RemoteAddr", remote: "[2001:db8::5]:80", want: "2001:db8::5"},
		{name: "RemoteAddr without port", remote: "198.51.100.7", want: "198.51.100.7"},
		{name: "unparseable RemoteAddr yields empty", remote: "pipe", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := chimw.ClientIP(tt.trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = chimw.ClientIPFromContext(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			for _, kv := range tt.headers {
				req.Header.Add(kv[0], kv[1])
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.want {
				t.Fatalf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientIPFromContextEmpty(t *testing.T) {
	if got := chimw.ClientIPFromContext(context.Background()); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}
