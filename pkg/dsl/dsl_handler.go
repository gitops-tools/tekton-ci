package dsl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gitops-tools/tekton-ci/pkg/cel"
	"github.com/gitops-tools/tekton-ci/pkg/ci"
	"github.com/gitops-tools/tekton-ci/pkg/git"
	"github.com/gitops-tools/tekton-ci/pkg/logger"
	"github.com/gitops-tools/tekton-ci/pkg/metrics"
	"github.com/gitops-tools/tekton-ci/pkg/volumes"
)

const (
	pipelineFilename = ".tekton_ci.yaml"
)

// Handler implements the GitEventHandler interface and processes
// .tekton_ci.yaml files in a repository.
type Handler struct {
	scmClient git.SCM
	log       logger.Logger
	m         metrics.Interface
	converter *DSLConverter
}

// New creates and returns a new Handler for converting ci.Pipelines into
// PipelineRuns.
func New(scmClient git.SCM, l logger.Logger, m metrics.Interface, d *DSLConverter) *Handler {
	return &Handler{
		scmClient: scmClient,
		converter: d,
		log:       l,
		m:         m,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hook, err := h.scmClient.ParseWebhookRequest(r)
	if err != nil {
		h.log.Errorf("error parsing webhook: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.m.CountInvalidHook()
		return
	}

	h.m.CountHook(hook)

	if hook.Kind() == scm.WebhookKindPush {
		created, err := h.converter.convert(r.Context(), hook.(*scm.PushHook))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		b, err := json.Marshal(created)
		if err != nil {
			h.log.Errorf("error marshaling response: %s", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			h.log.Errorf("error writing response: %s", err)
			return
		}
	}
}

// NewDSLConverter creates and returns a converter.
func NewDSLConverter(
	scmClient git.SCM,
	pipelineClient pipelineclientset.Interface,
	volumeCreator volumes.Creator,
	m metrics.Interface, cfg *Configuration,
	namespace string, l logger.Logger) *DSLConverter {
	return &DSLConverter{
		pipelineClient: pipelineClient,
		volumeCreator:  volumeCreator,
		log:            l,
		config:         cfg,
		m:              m,
		namespace:      namespace,
		scmClient:      scmClient,
	}

}

type DSLConverter struct {
	scmClient      git.SCM
	log            logger.Logger
	pipelineClient pipelineclientset.Interface
	namespace      string
	volumeCreator  volumes.Creator
	config         *Configuration
	m              metrics.Interface
}

func (d *DSLConverter) convert(ctx context.Context, evt *scm.PushHook) (*pipelinev1.PipelineRun, error) {
	repo := fmt.Sprintf("%s/%s", evt.Repo.Namespace, evt.Repo.Name)
	logItems := []interface{}{"repo", repo, "sha", evt.Commit.Sha}
	d.log.Infow("processing push event", logItems...)
	content, err := d.scmClient.FileContents(ctx, repo, pipelineFilename, evt.Commit.Sha)
	// This does not return an error if the pipeline definition can't be found.
	if git.IsNotFound(err) {
		d.log.Infof("no pipeline definition found in %s", repo)
		return nil, nil
	}
	if err != nil {
		d.log.Errorf("error fetching pipeline file: %s", err)
		return nil, err
	}
	if skip(evt) {
		d.log.Infow("skipping pipeline conversion", logItems...)
		return nil, nil
	}

	celCtx, err := cel.New(evt)
	if err != nil {
		d.log.Errorf("error creating a CEL context: %s", err)
		return nil, err
	}
	parsed, err := ci.Parse(bytes.NewReader(content))
	if err != nil {
		d.log.Errorf("error parsing pipeline definition: %s", err)
		return nil, nil
	}

	vc, err := d.volumeCreator.Create(ctx, d.namespace, d.config.VolumeSize)
	if err != nil {
		d.log.Errorf("error creating volume: %s", err)
		return nil, nil
	}
	pr, err := Convert(parsed, d.log, d.config, sourceFromPushEvent(evt), vc.ObjectMeta.Name, celCtx, evt.GUID)
	if err != nil {
		d.log.Errorf("error converting pipeline to pipelinerun: %s %#v", err, celCtx.Data)
		return nil, nil
	}
	if pr == nil {
		return nil, nil
	}
	created, err := d.pipelineClient.TektonV1().PipelineRuns(d.namespace).Create(ctx, pr, metav1.CreateOptions{})
	if err != nil {
		d.log.Errorf("error creating pipelinerun file: %s", err)
		return nil, nil
	}
	return created, nil
}

func sourceFromPushEvent(p *scm.PushHook) *Source {
	return &Source{
		RepoURL: p.Repo.Clone,
		Ref:     p.Commit.Sha,
	}
}

func skip(p *scm.PushHook) bool {
	matches := []string{"[ci skip]", "[skip ci]"}
	for _, m := range matches {
		if strings.Contains(p.Commit.Message, m) {
			return true
		}
	}
	return false
}
