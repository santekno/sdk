package chi

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
)

type clientIPKey struct{}

// ClientIP returns middleware that resolves the caller's IP address and stores
// it in the request context for [ClientIPFromContext].
//
// trustedHeader names the single header the deployment's edge or proxy
// overwrites with the real client address (for example "X-Real-IP" on
// Railway). It is the only header ever read: X-Forwarded-For, True-Client-IP,
// Forwarded and every other header are ignored, because a client can send
// them. Only configure a header your edge sets itself and overwrites when a
// client supplies it; otherwise callers can forge their address and escape a
// per-IP rate limit.
//
// With an empty trustedHeader, or when the header is absent or malformed, the
// address comes from r.RemoteAddr. Behind a proxy that is the proxy's address,
// so every such request shares one key and stays rate limited together; this
// fails closed rather than trusting the client.
//
// A header value is accepted only when it is a single IPv4 or IPv6 address,
// optionally with a port, after trimming surrounding spaces. Comma-separated
// lists, repeated headers, zones ("fe80::1%eth0"), hostnames and other
// garbage are rejected. IPv4-mapped IPv6 addresses are normalized to IPv4 and
// the result uses the canonical textual form, so one client maps to one key.
//
// The stored value is raw personal data. Hash it (hashx HMAC) before it
// reaches Redis, logs, or any third party.
func ClientIP(trustedHeader string) func(http.Handler) http.Handler {
	header := strings.TrimSpace(trustedHeader)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ""
			if header != "" {
				if vals := r.Header.Values(header); len(vals) == 1 {
					ip = parseIP(vals[0])
				}
			}
			if ip == "" {
				ip = parseIP(r.RemoteAddr)
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// ClientIPFromContext returns the address stored by [ClientIP], or "" when the
// middleware did not run or no valid address was found.
func ClientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

// parseIP returns the canonical form of a single IP address, with or without
// a port, or "" if s is anything else.
func parseIP(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, ", \t") {
		return ""
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return canonical(ap.Addr())
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return canonical(a)
	}
	// A bracketed IPv6 literal without a port ("[::1]") is not covered above.
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		if a, err := netip.ParseAddr(s[1 : len(s)-1]); err == nil {
			return canonical(a)
		}
	}
	return ""
}

func canonical(a netip.Addr) string {
	if a.Zone() != "" {
		return ""
	}
	return a.Unmap().String()
}
