package githooks

import (
	"net/http"

	"github.com/jenkins-x/go-scm/scm"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/pkg/logger"
)

// HookHandler decodes Webhook requests, and attempts to trigger a pipelinerun based
// on the CI configuration in the repository.
type HookHandler struct {
	scmClient    git.SCM
	eventHandler GitEventHandler
	log          logger.Logger
}

// New creates and returns a new HookHandler that can process incoming HTTP hook
// requests from SCM services.
func New(scmClient git.SCM, g GitEventHandler, l logger.Logger) *HookHandler {
	return &HookHandler{
		scmClient:    scmClient,
		log:          l,
		eventHandler: g,
	}
}

func (h *HookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hook, err := h.scmClient.ParseWebhookRequest(r)
	if err != nil {
		h.log.Errorf("error parsing webhook: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch evt := hook.(type) {
	case *scm.PullRequestHook:
		h.eventHandler.PullRequest(r.Context(), evt, w)
	case *scm.PushHook:
		h.eventHandler.Push(r.Context(), evt, w)
	}
}
