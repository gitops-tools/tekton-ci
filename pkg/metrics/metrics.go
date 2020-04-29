package metrics

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics is a wrapper around Prometheus metrics for counting
// events in the system.
type PrometheusMetrics struct {
	hooks        *prometheus.CounterVec
	invalidHooks prometheus.Counter
	apiCalls     *prometheus.CounterVec
}

// New creates and returns a PrometheusMetrics initialised with prometheus
// counters.
func New(ns string, reg prometheus.Registerer) *PrometheusMetrics {
	pm := &PrometheusMetrics{}
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	pm.hooks = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "hooks_total",
		Help:      "Count of Hooks received",
	}, []string{"kind"})

	pm.invalidHooks = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "hooks_invalid",
		Help:      "Count of invalid hooks received",
	})

	pm.apiCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "api_calls_total",
		Help:      "Count of API Calls made",
	}, []string{"kind"})

	reg.MustRegister(pm.hooks)
	reg.MustRegister(pm.invalidHooks)
	reg.MustRegister(pm.apiCalls)
	return pm
}

// CountHook records this hook as having been received, along with it's kind.
func (m *PrometheusMetrics) CountHook(h scm.Webhook) {
	m.hooks.With(prometheus.Labels{"kind": string(h.Kind())}).Inc()
}

// CountInvalidHook records "bad" hooks, probably due to non-matching secrets.
func (m *PrometheusMetrics) CountInvalidHook() {
	m.invalidHooks.Inc()
}

// CountAPICall records outgoing API calls to upstream services.
func (m *PrometheusMetrics) CountAPICall(name string) {
	m.apiCalls.With(prometheus.Labels{"kind": name}).Inc()
}
