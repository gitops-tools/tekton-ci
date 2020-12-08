package spec

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm/factory"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gitops-tools/tekton-ci/pkg/git"
	"github.com/gitops-tools/tekton-ci/pkg/metrics"
	"github.com/gitops-tools/tekton-ci/pkg/secrets"
	"github.com/gitops-tools/tekton-ci/test"
)

const testNS = "testing"

func TestHandlePullRequestOpenedEvent(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton/pull_request.yaml", "refs/pull/2/head", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeKube := fakeclientset.NewSimpleClientset()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	h := New(gitClient, fakeKube, testNS, logger.Sugar())
	req := test.MakeHookRequest(t, "../testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusNotFound, mustReadBody(t, w))
	}
	pr, err := fakeKube.TektonV1beta1().PipelineRuns(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if l := len(pr.Spec.PipelineSpec.Tasks); l != 1 {
		t.Fatalf("got %d tasks, want 1", l)
	}
	// check that it evaluated the parameters in the example.
	want := []pipelinev1.Param{
		{
			Name: "COMMIT_SHA",
			Value: pipelinev1.ArrayOrString{
				Type:      "string",
				StringVal: "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
			},
		},
	}
	if diff := cmp.Diff(want, pr.Spec.Params); diff != "" {
		t.Fatalf("pipelinerun parameters incorrect, diff\n%s", diff)
	}
}

func TestHandlePullRequestEventNoPipeline(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton/pull_request.yaml", "refs/pull/2/head", "")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeKube := fakeclientset.NewSimpleClientset()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	h := New(gitClient, fakeKube, testNS, logger.Sugar())
	req := test.MakeHookRequest(t, "../testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusOK, mustReadBody(t, w))
	}
	_, err = fakeKube.TektonV1beta1().PipelineRuns(testNS).Get(context.TODO(), defaultPipelineRunPrefix, metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatalf("pipelinerun was created when no pipeline definition exists")
	}
}

func TestHandlePushEvent(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton/push.yaml", "refs/tags/simple-tag", "testdata/push_content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeKube := fakeclientset.NewSimpleClientset()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	h := New(gitClient, fakeKube, testNS, logger.Sugar())
	req := test.MakeHookRequest(t, "../testdata/github_push.json", "push")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusNotFound, mustReadBody(t, w))
	}
	pr, err := fakeKube.TektonV1beta1().PipelineRuns(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if l := len(pr.Spec.PipelineSpec.Tasks); l != 1 {
		t.Fatalf("got %d tasks, want 1", l)
	}
	// check that it evaluated the parameters in the example.
	want := []pipelinev1.Param{
		{
			Name: "COMMIT_SHA",
			Value: pipelinev1.ArrayOrString{
				Type:      "string",
				StringVal: "6113728f27ae82c7b1a177c8d03f9e96e0adf246",
			},
		},
	}
	if diff := cmp.Diff(want, pr.Spec.Params); diff != "" {
		t.Fatalf("pipelinerun parameters incorrect, diff\n%s", diff)
	}
}

func mustReadBody(t *testing.T, req *http.Response) []byte {
	t.Helper()
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
