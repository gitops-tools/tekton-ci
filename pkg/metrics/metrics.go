package metrics

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	hooks *prometheus.CounterVec
}

// New creates and returns a PrometheusMetrics initialised with prometheus
// counters.
func New(reg prometheus.Registerer) *PrometheusMetrics {
	pm := &PrometheusMetrics{}
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	pm.hooks = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "dsl",
		Name:      "hooks_total",
		Help:      "Count of Hooks received",
	}, []string{"kind"})
	reg.MustRegister(pm.hooks)
	return pm
}

func (m *PrometheusMetrics) CountHook(h scm.Webhook) {
	m.hooks.With(prometheus.Labels{"kind": string(h.Kind())}).Inc()

}
