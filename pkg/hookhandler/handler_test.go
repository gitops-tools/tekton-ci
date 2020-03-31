package hookhandler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
)

func TestHandlePushEvent(t *testing.T) {
	// Path:"/api/v3/repos//contents/.tekton_ci.yaml", RawPath:"", ForceQuery:false, RawQuery:"ref=refs/pull/0/head", Fragment:""}

	as := makeAPIServer(t, "/api/v3/repos//contents/.tekton_ci.yaml", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	scmClient.Client = as.Client()
	h := Handler{httpClient: as.Client(), scmClient: scmClient}
	req := makeHookRequest(t, makePullRequestHookEvent(), "pull_request")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d", w.StatusCode, http.StatusOK)
	}

}

func makeAPIServer(t *testing.T, urlPath, fixture string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := ioutil.ReadFile(fixture)
		if err != nil {
			t.Fatalf("failed to read %s: %s", fixture, err)
		}
		// TODO assert on the Query
		if r.URL.Path != urlPath {
			http.NotFound(w, r)
		}
		w.Write(f)
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
func makeHookRequest(t *testing.T, evt interface{}, eventType string) *http.Request {
	req := httptest.NewRequest("POST", "/", serialiseToJSON(t, evt))
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	return req
}

func makePullRequestHookEvent() *scm.PullRequestHook {
	return &scm.PullRequestHook{
		Action: scm.ActionOpen,
		Repo: scm.Repository{
			Namespace: "testing",
			Name:      "repository",
			FullName:  "testing/repository",
			Clone:     "https://example.com/project/repo.git",
		},
		PullRequest: scm.PullRequest{
			Number: 2,
			Ref:    "refs/pull/2/head",
		},
	}
}
