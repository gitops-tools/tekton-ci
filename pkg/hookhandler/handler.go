package hookhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/pipelines"
)

const (
	pipelineFilename   = ".tekton_ci.yaml"
	defaultPipelineRun = "test-pipelinerun"
)

// Handler decodes Webhook requests, and attempts to trigger a pipelinerun based
// on the CI configuration in the repository.
type Handler struct {
	httpClient     *http.Client
	scmClient      *scm.Client
	pipelineClient pipelineclientset.Interface
	namespace      string
	log            logger
}

func New(client *http.Client, scmClient *scm.Client, pipelineClient pipelineclientset.Interface, namespace string, l logger) *Handler {
	return &Handler{
		httpClient:     client,
		scmClient:      scmClient,
		pipelineClient: pipelineClient,
		namespace:      namespace,
		log:            l,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hook, err := h.scmClient.Webhooks.Parse(r, func(scm.Webhook) (string, error) {
		return "", nil
	})
	if err != nil {
		h.log.Errorf("error parsing webhook: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch evt := hook.(type) {
	case *scm.PullRequestHook:
		repo := fmt.Sprintf("%s/%s", evt.Repo.Namespace, evt.Repo.Name)
		h.log.Infow("processing request", "repo", repo)
		content, _, err := h.scmClient.Contents.Find(r.Context(), repo, pipelineFilename, evt.PullRequest.Ref)
		// TODO: detect NotFound errors and drop.
		if err != nil {
			h.log.Errorf("error fetching pipeline file: %s", err)
			// TODO: should this return a 404?
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parsed, err := ci.Parse(bytes.NewReader(content.Data))
		if err != nil {
			h.log.Errorf("error parsing pipeline definition: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pr := pipelines.Convert(parsed, nameFromPullRequest(evt), sourceFromPullRequest(evt))
		created, err := h.pipelineClient.TektonV1beta1().PipelineRuns(h.namespace).Create(pr)
		if err != nil {
			h.log.Errorf("error creating pipelinerun file: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b, err := json.Marshal(created)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		h.log.Infow("completed request")
	}
}

func nameFromPullRequest(pr *scm.PullRequestHook) string {
	return defaultPipelineRun
}

func sourceFromPullRequest(pr *scm.PullRequestHook) *pipelines.Source {
	return &pipelines.Source{
		RepoURL: pr.Repo.Clone,
		Ref:     pr.PullRequest.Source,
	}
}
