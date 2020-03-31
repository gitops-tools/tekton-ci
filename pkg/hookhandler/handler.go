package hookhandler

import (
	"bytes"
	"log"
	"net/http"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/pipelines"
	"github.com/jenkins-x/go-scm/scm"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

const (
	pipelineFilename = ".tekton_ci.yaml"
)

type Handler struct {
	httpClient     *http.Client
	scmClient      *scm.Client
	triggersClient pipelineclientset.Interface
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hook, err := h.scmClient.Webhooks.Parse(r, func(scm.Webhook) (string, error) {
		return "", nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch evt := hook.(type) {
	case *scm.PullRequestHook:
		content, _, err := h.scmClient.Contents.Find(r.Context(), evt.Repo.FullName, pipelineFilename, evt.PullRequest.Ref)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parsed, err := ci.Parse(bytes.NewReader(content.Data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = pipelines.Convert(parsed, nameFromPullRequest(evt), sourceFromPullRequest(evt))
	}
}

func nameFromPullRequest(pr *scm.PullRequestHook) string {
	return "test-pipelinerun"
}

func sourceFromPullRequest(pr *scm.PullRequestHook) *pipelines.Source {
	return &pipelines.Source{
		RepoURL: pr.Repo.Clone,
		Ref:     pr.PullRequest.Ref,
	}
}
