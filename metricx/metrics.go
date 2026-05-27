package metricx

import "sync"

// Counter is a monotonically-increasing metric.
type Counter interface {
	Inc()
	Add(delta float64)
}

// Gauge is a metric that can be set, increased, or decreased.
type Gauge interface {
	Set(v float64)
	Inc()
	Dec()
	Add(delta float64)
}

// Histogram observes durations or sizes for percentile aggregation.
type Histogram interface {
	Observe(v float64)
}

// Registry creates and tracks named metrics.
type Registry interface {
	Counter(name string, labels ...string) Counter
	Gauge(name string, labels ...string) Gauge
	Histogram(name string, labels ...string) Histogram
}

// --- in-memory implementation ---

type inMemoryRegistry struct {
	mu         sync.Mutex
	counters   map[string]*atomicCounter
	gauges     map[string]*atomicGauge
	histograms map[string]*atomicHistogram
}

// NewRegistry returns a thread-safe in-memory metric registry suitable for
// tests and minimal deployments. Production users should swap to a
// Prometheus-backed Registry implementation.
func NewRegistry() Registry {
	return &inMemoryRegistry{
		counters:   make(map[string]*atomicCounter),
		gauges:     make(map[string]*atomicGauge),
		histograms: make(map[string]*atomicHistogram),
	}
}

func (r *inMemoryRegistry) Counter(name string, _ ...string) Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.counters[name]
	if !ok {
		c = &atomicCounter{}
		r.counters[name] = c
	}
	return c
}

func (r *inMemoryRegistry) Gauge(name string, _ ...string) Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.gauges[name]
	if !ok {
		g = &atomicGauge{}
		r.gauges[name] = g
	}
	return g
}

func (r *inMemoryRegistry) Histogram(name string, _ ...string) Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.histograms[name]
	if !ok {
		h = &atomicHistogram{}
		r.histograms[name] = h
	}
	return h
}

type atomicCounter struct {
	mu sync.Mutex
	v  float64
}

func (c *atomicCounter) Inc()              { c.Add(1) }
func (c *atomicCounter) Add(delta float64) { c.mu.Lock(); c.v += delta; c.mu.Unlock() }

// Value returns the current counter value (for tests).
func (c *atomicCounter) Value() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v
}

type atomicGauge struct {
	mu sync.Mutex
	v  float64
}

func (g *atomicGauge) Set(v float64)     { g.mu.Lock(); g.v = v; g.mu.Unlock() }
func (g *atomicGauge) Inc()              { g.Add(1) }
func (g *atomicGauge) Dec()              { g.Add(-1) }
func (g *atomicGauge) Add(delta float64) { g.mu.Lock(); g.v += delta; g.mu.Unlock() }

func (g *atomicGauge) Value() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.v
}

type atomicHistogram struct {
	mu       sync.Mutex
	count    int64
	sum      float64
	observed []float64
}

func (h *atomicHistogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += v
	h.observed = append(h.observed, v)
}

// Snapshot returns (count, sum) for tests.
func (h *atomicHistogram) Snapshot() (int64, float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count, h.sum
}
