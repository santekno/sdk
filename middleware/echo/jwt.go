package echo

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/santekno/sdk/ctxx"
	"github.com/santekno/sdk/jwtx"
)

// JWTAuth returns an Echo middleware that validates Bearer tokens.
//
// On success the validated subject claim is stored on the request context via
// ctxx.WithUserID. On failure the request is aborted with HTTP 401.
//
// Mirrors github.com/santekno/sdk/middleware/gin.JWTAuth.
func JWTAuth(key jwtx.Key) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing Authorization header",
				})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid Authorization header format",
				})
			}

			claims, err := jwtx.Verify(parts[1], key)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": err.Error(),
				})
			}

			ctx := ctxx.WithUserID(req.Context(), claims.Subject)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}
