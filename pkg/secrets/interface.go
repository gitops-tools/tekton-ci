package secrets

import (
	"github.com/jenkins-x/go-scm/scm"
)

type SecretGetter interface {
	Secret(hook scm.Webhook) (string, error)
}
