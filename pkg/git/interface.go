package git

import (
	"context"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
)

// SCM is a wrapper around go-scm's Client implementation.
type SCM interface {
	// ParseWebhookRequest returns the webhook payload.
	ParseWebhookRequest(req *http.Request) (scm.Webhook, error)
	// FileContents returns the contents of a file within a repo.
	FileContents(ctx context.Context, repo, path, ref string) ([]byte, error)
}
