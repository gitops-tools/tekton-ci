package metrics

import (
	"github.com/jenkins-x/go-scm/scm"
)

// MockMetrics is a value that provides a wrapper around Mock
// metrics for counting events in the system.
type MockMetrics struct {
	Hooks        int
	InvalidHooks int
	APICalls     int
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
