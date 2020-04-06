package githooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/pkg/logger"
	"github.com/bigkevmcd/tekton-ci/pkg/pipelines"
)

const (
	pipelineFilename   = ".tekton_ci.yaml"
	defaultPipelineRun = "test-pipelinerun"
)

// PipelineHandler implements the GitEventHandler interface and processes
// .tekton_ci.yaml files in a repository.
type PipelineHandler struct {
	scmClient      git.SCM
	log            logger.Logger
	pipelineClient pipelineclientset.Interface
	namespace      string
}

func New(scmClient git.SCM, pipelineClient pipelineclientset.Interface, namespace string, l logger.Logger) *PipelineHandler {
	return &PipelineHandler{
		scmClient:      scmClient,
		pipelineClient: pipelineClient,
		log:            l,
		namespace:      namespace,
	}
}

func (h *PipelineHandler) handlePullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter) {
	repo := fmt.Sprintf("%s/%s", evt.Repo.Namespace, evt.Repo.Name)
	h.log.Infow("processing request", "repo", repo)
	content, err := h.scmClient.FileContents(ctx, repo, pipelineFilename, evt.PullRequest.Ref)
	if git.IsNotFound(err) {
		h.log.Infof("no pipeline definition found in %s", repo)
		return
	}
	if err != nil {
		h.log.Errorf("error fetching pipeline file: %s", err)
		// TODO: should this return a 404?
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	parsed, err := ci.Parse(bytes.NewReader(content))
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

func nameFromPullRequest(pr *scm.PullRequestHook) string {
	return defaultPipelineRun
}

func sourceFromPullRequest(pr *scm.PullRequestHook) *pipelines.Source {
	return &pipelines.Source{
		RepoURL: pr.Repo.Clone,
		Ref:     pr.PullRequest.Sha,
	}
}
