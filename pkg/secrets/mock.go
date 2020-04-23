package secrets

import "github.com/jenkins-x/go-scm/scm"

// NewMock returns a simple secret getter that returns an empty secret token.
//
// This causes the go-scm webhook parser to ignore the header.
func NewMock() mockSecret {
	return mockSecret{}
}

type mockSecret struct{}

func (k mockSecret) Secret(hook scm.Webhook) (string, error) {
	return "", nil
}
