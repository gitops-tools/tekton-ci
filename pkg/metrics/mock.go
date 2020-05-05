package metrics

import (
	"github.com/jenkins-x/go-scm/scm"
)

var _ Interface = (*MockMetrics)(nil)

// MockMetrics is a type that provides a simple counter for metrics for test
// purposes.
type MockMetrics struct {
	Hooks          int
	InvalidHooks   int
	APICalls       int
	FailedAPICalls int
}

// NewMock creates and returns a MockMetrics.
func NewMock() *MockMetrics {
	return &MockMetrics{}
}

// CountHook records this hook as having been received, along with it's kind.
func (m *MockMetrics) CountHook(h scm.Webhook) {
	m.Hooks++
}

// CountInvalidHook records "bad" hooks, probably due to non-matching secrets.
func (m *MockMetrics) CountInvalidHook() {
	m.InvalidHooks++
}

// CountAPICall records outgoing API calls to upstream services.
func (m *MockMetrics) CountAPICall(name string) {
	m.APICalls++
}

// CountFailedAPICall records failed outgoing API calls to upstream services.
func (m *MockMetrics) CountFailedAPICall(name string) {
	m.FailedAPICalls++
}
