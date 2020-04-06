package githooks

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/test"
)

const deliveryID = "72d3162e-cc78-11e3-81ab-4c9367dc0958"

func TestHandlePullRequestEvent(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "refs/pull/2/head", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	scmClient.Client = as.Client()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	mock := &mockEventHandler{}
	h := New(git.New(scmClient), mock, logger.Sugar())
	req := makeHookRequest(t, "testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	mock.AssertPullRequestReceived(t, deliveryID)
}

func TestHandleUnknown(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "refs/pull/2/head", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	scmClient.Client = as.Client()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
	mock := &mockEventHandler{}
	h := New(git.New(scmClient), mock, logger.Sugar())
	req := makeHookRequest(t, "testdata/github_pull_request.json", "unknown")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if s := rec.Result().StatusCode; s != http.StatusInternalServerError {
		t.Fatalf("response status got %d, want %d", s, http.StatusInternalServerError)
	}
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
	req.Header.Add("X-GitHub-Delivery", deliveryID)
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

type mockEventHandler struct {
	pullRequests []*scm.PullRequestHook
}

func (m *mockEventHandler) PullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter) {
	if m.pullRequests == nil {
		m.pullRequests = []*scm.PullRequestHook{}
	}
	m.pullRequests = append(m.pullRequests, evt)
}

func (m *mockEventHandler) AssertPullRequestReceived(t *testing.T, s string) {
	for _, v := range m.pullRequests {
		if v.GUID == s {
			return
		}
	}
	t.Fatalf("pull request %#v was not received", s)
}
