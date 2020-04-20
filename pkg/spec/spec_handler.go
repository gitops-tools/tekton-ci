package spec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/pkg/logger"
)

const (
	pullRequestFilename      = ".tekton/pull_request.yaml"
	pushFilename             = ".tekton/push.yaml"
	defaultPipelineRunPrefix = "test-pipelinerun-"
)

// Handler implements the http.Handler interface, it grabs pipeline
// configurations from the incoming Hook's repository and attempts to generate a
// PipelineRun from them.
type Handler struct {
	scmClient      git.SCM
	log            logger.Logger
	pipelineClient pipelineclientset.Interface
	namespace      string
}

// New creates and returns a new Handler.
func New(scmClient git.SCM, pipelineClient pipelineclientset.Interface, namespace string, l logger.Logger) *Handler {
	return &Handler{
		scmClient:      scmClient,
		pipelineClient: pipelineClient,
		log:            l,
		namespace:      namespace,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hook, err := h.scmClient.ParseWebhookRequest(r)
	if err != nil {
		h.log.Errorf("error parsing webhook: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch evt := hook.(type) {
	case *scm.PullRequestHook:
		h.pullRequest(r.Context(), evt, w)
	case *scm.PushHook:
		h.push(r.Context(), evt, w)
	}
}

// TODO: refactor to remove the duplication.
func (h *Handler) pullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter) {
	repo := fmt.Sprintf("%s/%s", evt.Repo.Namespace, evt.Repo.Name)
	h.handleEvent(ctx, repo, evt.PullRequest.Ref, pullRequestFilename, evt, w)
}

func (h *Handler) push(ctx context.Context, evt *scm.PushHook, w http.ResponseWriter) {
	repo := fmt.Sprintf("%s/%s", evt.Repo.Namespace, evt.Repo.Name)
	h.handleEvent(ctx, repo, evt.Ref, pushFilename, evt, w)
}

func (h *Handler) handleEvent(ctx context.Context, repo, ref, filename string, evt scm.Webhook, w http.ResponseWriter) {
	h.log.Infow(fmt.Sprintf("processing event '%T'", evt), "repo", repo)
	content, err := h.scmClient.FileContents(ctx, repo, filename, ref)
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
	parsed, err := Parse(bytes.NewReader(content))
	if err != nil {
		h.log.Errorf("error parsing pipeline definition: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pr, err := Execute(parsed, evt, defaultPipelineRunPrefix)
	if err != nil {
		h.log.Errorf("error executing pipeline definition: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	created, err := h.pipelineClient.TektonV1beta1().PipelineRuns(h.namespace).Create(pr)
	if err != nil {
		h.log.Errorf("error creating pipelinerun file: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b, err := json.Marshal(created)
	if err != nil {
		h.log.Errorf("error marshaling response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		h.log.Errorf("error writing response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.log.Infow("completed request")
}
