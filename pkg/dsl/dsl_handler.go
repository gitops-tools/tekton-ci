package dsl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/pkg/logger"
	"github.com/bigkevmcd/tekton-ci/pkg/volumes"
)

const (
	pipelineFilename = ".tekton_ci.yaml"
)

var defaultVolumeSize = resource.MustParse("1Gi")

// Handler implements the GitEventHandler interface and processes
// .tekton_ci.yaml files in a repository.
type Handler struct {
	scmClient      git.SCM
	log            logger.Logger
	pipelineClient pipelineclientset.Interface
	namespace      string
	volumeCreator  volumes.Creator
	config         *Configuration
}

// New creates and returns a new Handler for converting ci.Pipelines into
// PipelineRuns.
func New(scmClient git.SCM, pipelineClient pipelineclientset.Interface, volumeCreator volumes.Creator, cfg *Configuration, namespace string, l logger.Logger) *Handler {
	return &Handler{
		scmClient:      scmClient,
		pipelineClient: pipelineClient,
		volumeCreator:  volumeCreator,
		log:            l,
		config:         cfg,
		namespace:      namespace,
	}
}

func isAction(evt *scm.PullRequestHook, acts ...scm.Action) bool {
	for _, a := range acts {
		if evt.Action == a {
			return true
		}
	}
	return false
}

// PullRequest implements the GitEventHandler interface.
func (h *Handler) PullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter) {
	if !isAction(evt, scm.ActionOpen, scm.ActionSync) {
		return
	}
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

	vc, err := h.volumeCreator.Create(h.namespace, defaultVolumeSize)
	if err != nil {
		h.log.Errorf("error creating volume: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pr := Convert(parsed, h.config, sourceFromPullRequest(evt), vc.ObjectMeta.Name)

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

func sourceFromPullRequest(pr *scm.PullRequestHook) *Source {
	return &Source{
		RepoURL: pr.Repo.Clone,
		Ref:     pr.PullRequest.Sha,
	}
}
