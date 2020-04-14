package cel

import (
	"regexp"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/go-cmp/cmp"

	"github.com/bigkevmcd/tekton-ci/test/hook"
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
			fixture:   "../testdata/github_pull_request.json",
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
			ectx, err := makeEvalContext(hook.MakeHookFromFixture(rt, tt.fixture, tt.eventType))
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
			ectx, err := makeEvalContext(hook.MakeHookFromFixture(rt, "../testdata/github_pull_request.json", "pull_request"))
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

func TestContextEvaluate(t *testing.T) {
	hook := hook.MakeHookFromFixture(t, "../testdata/github_pull_request.json", "pull_request")
	ctx, err := New(hook)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ctx.Evaluate("hook.Action")
	if err != nil {
		t.Fatal(err)
	}
	if result != types.String("opened") {
		t.Fatalf("got %#v, want %#v\n", result, types.String("opened"))
	}
}

func TestContextEvaluateToString(t *testing.T) {
	hook := hook.MakeHookFromFixture(t, "../testdata/github_pull_request.json", "pull_request")
	ctx, err := New(hook)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ctx.EvaluateToString("hook.PullRequest.Number")
	if err != nil {
		t.Fatal(err)
	}
	if result != "2" {
		t.Fatalf("got %#v, want %#v\n", result, "2")
	}
}

func TestMakeEvalContext(t *testing.T) {
	hook := hook.MakeHookFromFixture(t, "../testdata/github_push.json", "push")
	ctx, err := makeEvalContext(hook)
	if err != nil {
		t.Fatal(err)
	}
	if v := ctx["hook"].(map[string]interface{})["Ref"]; v != "refs/tags/simple-tag" {
		t.Fatalf("hook.ref got %s, want %s", v, "refs/tags/simple-tag")
	}
}

func TestEvalContextVars(t *testing.T) {
	tests := []struct {
		fixture   string
		eventType string
		want      map[string]string
	}{
		{"../testdata/github_pull_request.json", "pull_request", map[string]string{
			"CI_COMMIT_SHA":       "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
			"CI_COMMIT_SHORT_SHA": "ec26c3e",
			"CI_COMMIT_BRANCH":    "changes",
		}},
		{"../testdata/github_push.json", "push", map[string]string{
			"CI_COMMIT_SHA":       "6113728f27ae82c7b1a177c8d03f9e96e0adf246",
			"CI_COMMIT_SHORT_SHA": "6113728",
			"CI_COMMIT_BRANCH":    "simple-tag",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(rt *testing.T) {
			hook := hook.MakeHookFromFixture(t, tt.fixture, tt.eventType)
			ctx, err := makeEvalContext(hook)
			if err != nil {
				rt.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, ctx["vars"].(map[string]string)); diff != "" {
				rt.Fatalf("vars didn't match: %s\n", diff)
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
