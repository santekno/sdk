package logx

import (
	"io"
	"os"
)

type config struct {
	Level   string
	Format  string // "json" or "console"
	Output  io.Writer
	Backend string // "slog" (default), "zap" (future)
	Fields  []any
	Caller  bool
	Stack   bool
}

// Option is a functional option for configuring a [Logger].
type Option func(*config)

func defaultConfig() *config {
	return &config{
		Level:   "info",
		Format:  "json",
		Output:  os.Stdout,
		Backend: "slog",
	}
}

// WithLevel sets the minimum log level: "debug", "info", "warn", "error".
// Defaults to "info".
func WithLevel(l string) Option { return func(c *config) { c.Level = l } }

// WithFormat sets the output format: "json" (default) or "console".
func WithFormat(f string) Option { return func(c *config) { c.Format = f } }

// WithOutput sets the output writer. Defaults to os.Stdout.
func WithOutput(w io.Writer) Option { return func(c *config) { c.Output = w } }

// WithBackend selects the logger backend: "slog" (default) or "zap".
func WithBackend(b string) Option { return func(c *config) { c.Backend = b } }

// WithFields attaches initial key-value pairs to every emitted log line.
func WithFields(kv ...any) Option { return func(c *config) { c.Fields = append(c.Fields, kv...) } }

// WithCaller enables capturing the file:line of the calling site.
func WithCaller(b bool) Option { return func(c *config) { c.Caller = b } }

// WithStack enables capturing a stack trace on Error-level lines.
func WithStack(b bool) Option { return func(c *config) { c.Stack = b } }
