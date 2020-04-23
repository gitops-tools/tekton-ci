package secrets

import (
	"github.com/jenkins-x/go-scm/scm"
)

// SecretGetter is provided by values that implement Secret, to look up the
// correct secret for a Webhook in order to validate the origin of the Webhook.
type SecretGetter interface {
	Secret(hook scm.Webhook) (string, error)
}
