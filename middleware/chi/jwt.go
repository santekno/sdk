package chi

import (
	"context"
	"net/http"
	"strings"

	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/errx"
	"github.com/santekno/sdk/jwtx"
)

type claimsKey struct{}

// JWTAuth returns middleware that requires a valid "Authorization: Bearer <token>"
// header. On success the token subject is stored with ctxx.WithUserID and the
// full claims are available through ClaimsFromContext. On failure the request
// ends with 401 and an UNAUTHORIZED error.
func JWTAuth(key jwtx.Key, opts ...Option) func(http.Handler) http.Handler {
	o := newOptions(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				o.writeError(w, r, unauthorized("missing Authorization header"))
				return
			}
			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				o.writeError(w, r, unauthorized("invalid Authorization header format"))
				return
			}
			claims, err := jwtx.Verify(strings.TrimSpace(token), key)
			if err != nil {
				o.writeError(w, r, unauthorized("invalid or expired token"))
				return
			}
			ctx := ctxx.WithUserID(r.Context(), claims.Subject)
			ctx = context.WithValue(ctx, claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext returns the claims JWTAuth verified for this request.
func ClaimsFromContext(ctx context.Context) (*jwtx.Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*jwtx.Claims)
	return c, ok && c != nil
}

func unauthorized(msg string) *errx.AppError {
	return errx.New("UNAUTHORIZED", msg).WithHTTPStatus(http.StatusUnauthorized)
}
