package secrets

import "github.com/jenkins-x/go-scm/scm"

// NewMock returns a simple secret getter that returns an empty secret token.
//
// This causes the go-scm webhook parser to ignore the header.
func NewMock() MockSecret {
	return MockSecret{}
}

// MockSecret implements the SecretGetter but returns an empty string, which
// go-scm uses to indicate that it should not check the secret.
type MockSecret struct{}

// Secret implements the SecretGetter interface.
func (k MockSecret) Secret(hook scm.Webhook) (string, error) {
	return "", nil
}
