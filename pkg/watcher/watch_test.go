package watcher

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"

	"github.com/gitops-tools/tekton-ci/pkg/dsl"
	"github.com/gitops-tools/tekton-ci/pkg/resources"
)

const (
	testSHA       = "9bb041d2f04027d96db99979c58531c3f6e39312"
	testSourceURL = "https://github.com/bigkevmcd/tekton-ci.git"
)

func TestHandlePipelineRun(t *testing.T) {
	ctx := context.TODO()
	fakeSCM, data := fake.NewDefault()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr := makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"}),
		taskResult())
	fakeTektonClient := fakeclientset.NewSimpleClientset(pr)

	err := handlePipelineRun(ctx, fakeSCM, fakeTektonClient, pr, logger.Sugar())
	if err != nil {
		t.Fatal(err)
	}

	statuses := data.Statuses[testSHA]
	if l := len(statuses); l != 1 {
		t.Fatalf("incorrect number of statuses notifified, got %d, want 1", l)
	}
	if statuses[0].State != scm.StatePending {
		t.Fatalf("incorrect state notified, got %v, want %v", statuses[0].State, scm.StatePending)
	}

	loaded, err := fakeTektonClient.TektonV1().
		PipelineRuns(pr.ObjectMeta.Namespace).
		Get(ctx, pr.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s := notificationState(loaded); s != "Pending" {
		t.Fatalf("post-handling last state got %s, want %s", s, "Pending")
	}
}

func TestHandlePipelineRunWithRepeatedState(t *testing.T) {
	ctx := context.TODO()
	fakeSCM, data := fake.NewDefault()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr := makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"}),
		taskResult())
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = "Pending"
	fakeTektonClient := fakeclientset.NewSimpleClientset(pr)

	err := handlePipelineRun(ctx, fakeSCM, fakeTektonClient, pr, logger.Sugar())
	if err != nil {
		t.Fatal(err)
	}

	statuses := data.Statuses[testSHA]
	if l := len(statuses); l != 0 {
		t.Fatalf("incorrect number of statuses notifified, got %d, want 0", l)
	}
	loaded, err := fakeTektonClient.TektonV1().
		PipelineRuns(pr.ObjectMeta.Namespace).
		Get(ctx, pr.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s := notificationState(loaded); s != "Pending" {
		t.Fatalf("post-handling last state got %s, want %s", s, "Pending")
	}
}

func TestHandlePipelineRunWithNewState(t *testing.T) {
	ctx := context.TODO()
	fakeSCM, data := fake.NewDefault()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr := makePipelineRun(
		dsl.AnnotateSource("test-id",
			&dsl.Source{RepoURL: testSourceURL, Ref: "master"}),
		taskResult(),
		statusCondition(apis.ConditionSucceeded, corev1.ConditionTrue),
	)
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = "Pending"
	fakeTektonClient := fakeclientset.NewSimpleClientset(pr)

	err := handlePipelineRun(ctx, fakeSCM, fakeTektonClient, pr, logger.Sugar())
	if err != nil {
		t.Fatal(err)
	}

	statuses := data.Statuses[testSHA]
	if l := len(statuses); l != 1 {
		t.Fatalf("incorrect number of statuses notifified, got %d, want 1", l)
	}
	loaded, err := fakeTektonClient.TektonV1().
		PipelineRuns(pr.ObjectMeta.Namespace).
		Get(ctx, pr.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if s := notificationState(loaded); s != "Successful" {
		t.Fatalf("post-handling last state got %s, want %s", s, "Successful")
	}
}

func TestFindCommit(t *testing.T) {
	pr := makePipelineRun(taskResult())

	commit := findCommit(pr)

	if commit != testSHA {
		t.Fatalf("findCommit() got %#v, want %#v", commit, testSHA)
	}
}

func TestFindNotificationState(t *testing.T) {
	pr := makePipelineRun()
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = "Pending"

	state := notificationState(pr)

	if state != Pending.String() {
		t.Fatalf("notificationState() got %s, want %s", state, Pending.String())
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

func taskResult() resources.PipelineRunOpt {
	return func(pr *pipelinev1.PipelineRun) {
		ann := pr.GetAnnotations()
		ann[dsl.CISourceRefAnnotation] = testSHA
		pr.SetAnnotations(ann)
	}
}

func makePipelineRun(opts ...resources.PipelineRunOpt) *pipelinev1.PipelineRun {
	return resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	}, opts...)
}
