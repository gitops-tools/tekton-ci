package githooks

import (
	"context"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
)

// GitEventHandler is implemented by values that can be used to handle incoming
// webhooks that have been parsed by the go-scm package.
type GitEventHandler interface {
	PullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter)
}
