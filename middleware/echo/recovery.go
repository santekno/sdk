package echo

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/santekno/sdk/logx"
)

// Recovery returns an Echo middleware that recovers from panics and responds
// with HTTP 500. The panic value is logged via l.
//
// Mirrors github.com/santekno/sdk/middleware/gin.Recovery.
func Recovery(l logx.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if val := recover(); val != nil {
					req := c.Request()
					l.Error("panic recovered",
						"error", val,
						"method", req.Method,
						"path", req.URL.Path,
					)
					err = c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "internal server error",
					})
				}
			}()
			return next(c)
		}
	}
}
