package watcher

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"

	"github.com/bigkevmcd/tekton-ci/pkg/dsl"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

const (
	testSHA       = "9bb041d2f04027d96db99979c58531c3f6e39312"
	testSourceURL = "https://github.com/bigkevmcd/tekton-ci.git"
)

func TestHandlePipelineRun(t *testing.T) {
	fakeSCM, data := fake.NewDefault()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr := makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"}))
	fakeTektonClient := fakeclientset.NewSimpleClientset(pr)

	handlePipelineRun(fakeSCM, fakeTektonClient, pr, logger.Sugar())

	statuses := data.Statuses[testSHA]
	if l := len(statuses); l != 1 {
		t.Fatalf("incorrect number of statuses notifified, got %d, want 1", l)
	}
	if statuses[0].State != scm.StatePending {
		t.Fatalf("incorrect state notified, got %v, want %v", statuses[0].State, scm.StatePending)
	}

	loaded, err := fakeTektonClient.TektonV1beta1().PipelineRuns(pr.ObjectMeta.Namespace).Get(pr.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s := findNotificationState(loaded); s != "Pending" {
		t.Fatalf("post-handling last state got %s, want %s", s, "Pending")
	}
}

func TestHandlePipelineRunWithRepeatedState(t *testing.T) {
	fakeSCM, data := fake.NewDefault()
	fakeTektonClient := fakeclientset.NewSimpleClientset()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr := makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"}))
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = "Pending"

	handlePipelineRun(fakeSCM, fakeTektonClient, pr, logger.Sugar())

	statuses := data.Statuses[testSHA]
	if l := len(statuses); l != 0 {
		t.Fatalf("incorrect number of statuses notifified, got %d, want 0", l)
	}
}

func TestFindCommit(t *testing.T) {
	pr := makePipelineRun()

	commit := findCommit(pr)

	if commit != testSHA {
		t.Fatalf("findCommit() got %#v, want %#v", commit, testSHA)
	}
}

func TestFindNotificationState(t *testing.T) {
	pr := makePipelineRun()
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = "Pending"

	state := findNotificationState(pr)

	if state != Pending.String() {
		t.Fatalf("findNotificationState() got %s, want %s", state, Pending.String())
	}
}

func TestFindRepoURL(t *testing.T) {
	repoURL := findRepoURL(makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"})))
	if repoURL != testSourceURL {
		t.Fatalf("findRepoURL() got %#v, want %#v", repoURL, testSourceURL)
	}
}

func TestFindRepoURLWithNoRepoURL(t *testing.T) {
	repoURL := findRepoURL(makePipelineRun())

	if repoURL != "" {
		t.Fatalf("findRepoURL() got %#v, want ''", repoURL)
	}
}

func TestCommitStatusInput(t *testing.T) {
	want := &scm.StatusInput{
		State: scm.StatePending,
		Label: tektonCILabel,
		Desc:  "Tekton CI Status",
	}
	pr := makePipelineRun()

	cs := commitStatusInput(pr)

	if diff := cmp.Diff(want, cs); diff != "" {
		t.Fatalf("commitStatusInput failed:\n%s", diff)
	}
}

func TestParseRepoFromURL(t *testing.T) {
	r, err := parseRepoFromURL(testSourceURL)
	if err != nil {
		t.Fatal(err)
	}
	if r != "bigkevmcd/tekton-ci" {
		t.Fatalf("parseRepoFromURL got %s, want %s", r, "bigkevmcd/tekton-ci")
	}
}

func statusCondition(c apis.ConditionType, s corev1.ConditionStatus) resources.PipelineRunOpt {
	return func(pr *pipelinev1.PipelineRun) {
		pr.Status.Conditions = append(pr.Status.Conditions, apis.Condition{Type: c, Status: s})
	}
}

func makePipelineRun(opts ...resources.PipelineRunOpt) *pipelinev1.PipelineRun {
	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	}, opts...)

	pr.Status = pipelinev1.PipelineRunStatus{
		Status: duckv1beta1.Status{
			Conditions: []apis.Condition{},
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
	return pr
}
