package githooks

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm/factory"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
)

const testNS = "testing"

func TestHandlePullRequestEvent(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "refs/pull/2/head", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	fakeKube := fakeclientset.NewSimpleClientset()
	scmClient.Client = as.Client()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	h := Handler{scmClient: git.New(scmClient), pipelineClient: fakeKube, namespace: testNS, log: logger.Sugar()}
	req := makeHookRequest(t, "testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusNotFound, mustReadBody(t, w))
	}
	pr, err := fakeKube.TektonV1beta1().PipelineRuns(testNS).Get(defaultPipelineRun, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if l := len(pr.Spec.PipelineSpec.Tasks); l != 4 {
		t.Fatalf("got %d tasks, want 4", l)
	}
	// check that it picked up the correct source URL and branch from the
	// fixture file.
	want := []string{
		"/ko-app/git-init",
		"-url", "https://github.com/Codertocat/Hello-World.git",
		"-revision", "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		"-path", "$(workspaces.source.path)",
	}
	if diff := cmp.Diff(want, pr.Spec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Container.Command); diff != "" {
		t.Fatalf("git command incorrect, diff\n%s", diff)
	}
}

func TestHandlePullRequestEventNoPipeline(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "refs/pull/2/head", "")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	fakeKube := fakeclientset.NewSimpleClientset()
	scmClient.Client = as.Client()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	h := Handler{scmClient: git.New(scmClient), pipelineClient: fakeKube, namespace: testNS, log: logger.Sugar()}
	req := makeHookRequest(t, "testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusOK, mustReadBody(t, w))
	}
	_, err = fakeKube.TektonV1beta1().PipelineRuns(testNS).Get(defaultPipelineRun, metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatalf("pipelinerun was created when no pipeline definition exists")
	}

}

func makeAPIServer(t *testing.T, urlPath, ref, fixture string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != urlPath {
			t.Fatalf("request path got %s, want %s", r.URL.Path, urlPath)
		}
		if queryRef := r.URL.Query().Get("ref"); queryRef != ref {
			t.Fatalf("failed to match ref, got %s, want %s", queryRef, ref)
		}
		if fixture == "" {
			http.NotFound(w, r)
			return
		}
		b, err := ioutil.ReadFile(fixture)
		if err != nil {
			t.Fatalf("failed to read %s: %s", fixture, err)
		}
		w.Write(b)
	}))
}

func serialiseToJSON(t *testing.T, e interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal %#v to JSON: %s", e, err)
	}
	return bytes.NewBuffer(b)
}

// TODO use uuid to generate the Delivery ID.
func makeHookRequest(t *testing.T, fixture, eventType string) *http.Request {
	req := httptest.NewRequest("POST", "/", serialiseToJSON(t, readFixture(t, fixture)))
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	return req
}

func readFixture(t *testing.T, filename string) map[string]interface{} {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read %s: %s", filename, err)
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		t.Fatalf("failed to unmarshal %s: %s", filename, err)
	}
	return result
}

func mustReadBody(t *testing.T, req *http.Response) []byte {
	t.Helper()
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
