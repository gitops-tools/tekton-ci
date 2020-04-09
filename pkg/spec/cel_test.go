package spec

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/jenkins-x/go-scm/scm/factory"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/test"
)

func TestExpressionEvaluation(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		fixture   string
		eventType string
		want      ref.Val
	}{
		{
			name:      "simple body value",
			expr:      "hook.Action",
			fixture:   "testdata/github_pull_request.json",
			eventType: "pull_request",
			want:      types.String("opened"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			env, err := makeCelEnv()
			if err != nil {
				rt.Errorf("failed to make env: %s", err)
				return
			}
			ectx, err := makeEvalContext(makeHookFromFixture(t, tt.fixture, tt.eventType))
			if err != nil {
				rt.Errorf("failed to make eval context %s", err)
				return
			}
			got, err := evaluate(tt.expr, env, ectx)
			if err != nil {
				rt.Errorf("evaluate() got an error %s", err)
				return
			}
			_, ok := got.(*types.Err)
			if ok {
				rt.Errorf("error evaluating expression: %s", got)
				return
			}

			if !got.Equal(tt.want).(types.Bool) {
				rt.Errorf("evaluate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExpressionEvaluation_Error(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "unknown value",
			expr: "hook.Unknown",
			want: "no such key: Unknown",
		},
		{
			name: "invalid syntax",
			expr: "body.value = 'testing'",
			want: "Syntax error: token recognition error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			env, err := makeCelEnv()
			if err != nil {
				rt.Errorf("failed to make env: %s", err)
				return
			}
			ectx, err := makeEvalContext(makeHookFromFixture(t, "testdata/github_pull_request.json", "pull_request"))
			if err != nil {
				rt.Errorf("failed to make eval context %s", err)
				return
			}
			_, err = evaluate(tt.expr, env, ectx)
			if !matchError(t, tt.want, err) {
				rt.Errorf("evaluate() got %s, wanted %s", err, tt.want)
			}
		})
	}
}

// TODO move this and share via a specific test package.
func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}

func makeHookFromFixture(t *testing.T, filename, eventType string) interface{} {
	t.Helper()
	req := makeHookRequest(t, filename, eventType)
	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := git.New(scmClient)
	hook, err := client.ParseWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	return hook
}

// TODO use uuid to generate the Delivery ID.
func makeHookRequest(t *testing.T, fixture, eventType string) *http.Request {
	req := httptest.NewRequest("POST", "/", serialiseToJSON(t, test.ReadJSONFixture(t, fixture)))
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	return req
}

func makeClient(t *testing.T, as *httptest.Server) *git.SCMClient {
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	return git.New(scmClient)
}

func serialiseToJSON(t *testing.T, e interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal %#v to JSON: %s", e, err)
	}
	return bytes.NewBuffer(b)
}
