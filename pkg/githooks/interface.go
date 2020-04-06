package githooks

import (
	"context"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
)

type GitEventHandler interface {
	PullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter)
}
