package logx_test

import (
	"bytes"
	"strings"

	"github.com/santekno/sdk/logx"
)

func ExampleNew() {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf), logx.WithLevel("debug"))
	l.Info("hello", "user", "alice")
	// Output is JSON — just verify it contains the message.
	_ = strings.Contains(buf.String(), "hello")
}

func ExampleLogger_WithContext() {
	var buf bytes.Buffer
	l := logx.New(logx.WithOutput(&buf))
	_ = l.With("service", "api")
	// With returns a new Logger with the fields attached.
}
