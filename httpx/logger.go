package httpx

// Logger is the structured logging interface accepted by [WithLogger].
// Any [logx.Logger] satisfies this interface.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}
