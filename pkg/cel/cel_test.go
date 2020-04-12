package cel

import (
	"regexp"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

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
			ectx, err := makeEvalContext(test.MakeHookFromFixture(rt, tt.fixture, tt.eventType))
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
			ectx, err := makeEvalContext(test.MakeHookFromFixture(rt, "testdata/github_pull_request.json", "pull_request"))
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
	hook := test.MakeHookFromFixture(t, "testdata/github_pull_request.json", "pull_request")
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
	hook := test.MakeHookFromFixture(t, "testdata/github_pull_request.json", "pull_request")
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

// TODO move this and share via a specific test package.
func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
