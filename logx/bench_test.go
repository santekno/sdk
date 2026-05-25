package logx_test

import (
	"io"
	"testing"

	"github.com/santekno/sdk/logx"
)

func BenchmarkLoggerInfo10Fields(b *testing.B) {
	l := logx.New(logx.WithOutput(io.Discard))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		l.Info("benchmark message",
			"k1", "v1", "k2", "v2", "k3", "v3", "k4", "v4", "k5", "v5",
			"k6", "v6", "k7", "v7", "k8", "v8", "k9", "v9", "k10", "v10",
		)
	}
}
