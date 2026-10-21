package chi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/santekno/sdk/ctxx"
)

// HeaderRequestID is the header RequestID reads and echoes back.
const HeaderRequestID = "X-Request-ID"

// Incoming IDs are logged and echoed, so only accept short, printable tokens;
// anything else is replaced to keep log lines and headers clean.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// RequestID returns middleware that propagates the caller's X-Request-ID, or
// generates a random one, stores it with ctxx.WithRequestID, and sets it on
// the response so clients can quote it in bug reports.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if !validRequestID.MatchString(id) {
				id = randomHex(16)
			}
			w.Header().Set(HeaderRequestID, id)
			next.ServeHTTP(w, r.WithContext(ctxx.WithRequestID(r.Context(), id)))
		})
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
