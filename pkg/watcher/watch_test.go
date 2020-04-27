package watcher

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"

	"github.com/bigkevmcd/tekton-ci/pkg/dsl"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

const (
	testSHA   = "9bb041d2f04027d96db99979c58531c3f6e39312"
	sourceURL = "https://github.com/bigkevmcd/tekton-ci.git"
)

func TestFindCommit(t *testing.T) {
	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	})
	pr.Status = pipelinev1.PipelineRunStatus{
		PipelineRunStatusFields: pipelinev1.PipelineRunStatusFields{
			TaskRuns: map[string]*pipelinev1.PipelineRunTaskRunStatus{
				"testing": {
					Status: &pipelinev1.TaskRunStatus{
						TaskRunStatusFields: pipelinev1.TaskRunStatusFields{
							ResourcesResult: []pipelinev1.PipelineResourceResult{
								{Key: "commit", Value: testSHA},
							},
						},
					},
				},
			},
		},
	}

	commit := findCommit(pr)

	if commit != testSHA {
		t.Fatalf("findCommit() got %#v, want %#v", commit, testSHA)
	}
}

func TestFindRepoURL(t *testing.T) {
	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	}, dsl.AnnotateSource("test-id", &dsl.Source{RepoURL: sourceURL, Ref: "master"}))

	repoURL := findRepoURL(pr)

	if repoURL != sourceURL {
		t.Fatalf("findRepoURL() got %#v, want %#v", repoURL, sourceURL)
	}
}

func TestCommitStatusInput(t *testing.T) {
	want := &scm.StatusInput{
		State: scm.StatePending,
		Label: TektonCILabel,
		Desc:  "Tekton CI Status",
	}

	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	})
	pr.Status = pipelinev1.PipelineRunStatus{
		Status: duckv1beta1.Status{
			Conditions: []apis.Condition{
				{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown},
			},
		},
		PipelineRunStatusFields: pipelinev1.PipelineRunStatusFields{
			TaskRuns: map[string]*pipelinev1.PipelineRunTaskRunStatus{
				"testing": {
					Status: &pipelinev1.TaskRunStatus{
						TaskRunStatusFields: pipelinev1.TaskRunStatusFields{
							ResourcesResult: []pipelinev1.PipelineResourceResult{
								{Key: "commit", Value: testSHA},
							},
						},
					},
				},
			},
		},
	}

	cs := commitStatusInput(pr)
	if diff := cmp.Diff(want, cs); diff != "" {
		t.Fatalf("commitStatusInput failed:\n%s", diff)
	}
}

func TestParseRepoFromURL(t *testing.T) {
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	r, err := parseRepoFromURL(sourceURL, logger.Sugar())
	if err != nil {
		t.Fatal(err)
	}
	if r != "bigkevmcd/tekton-ci" {
		t.Fatalf("parseRepoFromURL got %s, want %s", r, "bigkevmcd/tekton-ci")
	}
}
