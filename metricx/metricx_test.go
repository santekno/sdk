package metricx_test

import (
	"testing"

	"github.com/santekno/sdk/metricx"
)

func TestRegistry_Counter(t *testing.T) {
	r := metricx.NewRegistry()
	c := r.Counter("santekno_test_total")
	c.Inc()
	c.Add(4)

	// Re-fetching by name should return the same counter.
	same := r.Counter("santekno_test_total")
	if c != same {
		t.Error("registry should return the same counter for same name")
	}
}

func TestRegistry_Gauge(t *testing.T) {
	r := metricx.NewRegistry()
	g := r.Gauge("santekno_test_gauge")
	g.Set(5)
	g.Inc()
	g.Dec()
	g.Add(10)
}

func TestRegistry_Histogram(t *testing.T) {
	r := metricx.NewRegistry()
	h := r.Histogram("santekno_test_duration")
	h.Observe(0.1)
	h.Observe(0.2)
	h.Observe(0.3)
}
