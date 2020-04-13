package githooks

import (
	"context"
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
	req := test.MakeHookRequest(t, "testdata/github_pull_request.json", "pull_request")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	mock.AssertPullRequestReceived(t, deliveryID)
}

func TestHandlePushEvent(t *testing.T) {
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
	req := test.MakeHookRequest(t, "testdata/github_push.json", "push")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	mock.AssertPushReceived(t, deliveryID)
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
	req := test.MakeHookRequest(t, "testdata/github_pull_request.json", "unknown")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if s := rec.Result().StatusCode; s != http.StatusInternalServerError {
		t.Fatalf("response status got %d, want %d", s, http.StatusInternalServerError)
	}
}

type mockEventHandler struct {
	pullRequests []*scm.PullRequestHook
	pushes       []*scm.PushHook
}

func (m *mockEventHandler) PullRequest(ctx context.Context, evt *scm.PullRequestHook, w http.ResponseWriter) {
	if m.pullRequests == nil {
		m.pullRequests = []*scm.PullRequestHook{}
	}
	m.pullRequests = append(m.pullRequests, evt)
}

func (m *mockEventHandler) Push(ctx context.Context, evt *scm.PushHook, w http.ResponseWriter) {
	if m.pushes == nil {
		m.pushes = []*scm.PushHook{}
	}
	m.pushes = append(m.pushes, evt)
}

func (m *mockEventHandler) AssertPullRequestReceived(t *testing.T, s string) {
	t.Helper()
	for _, v := range m.pullRequests {
		if v.GUID == s {
			return
		}
	}
	t.Fatalf("pull request %#v was not received", s)
}

func (m *mockEventHandler) AssertPushReceived(t *testing.T, s string) {
	t.Helper()
	for _, v := range m.pushes {
		if v.GUID == s {
			return
		}
	}
	t.Fatalf("push %#v was not received", s)
}
