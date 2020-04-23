package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/bigkevmcd/tekton-ci/test/hook"
)

func TestCountHook(t *testing.T) {
	m := New(prometheus.NewRegistry())
	hook := hook.MakeHookFromFixture(t, "../testdata/github_pull_request.json", "pull_request")

	m.CountHook(hook)

	err := testutil.CollectAndCompare(m.hooks, strings.NewReader(`
# HELP dsl_hooks_total Count of Hooks received
# TYPE dsl_hooks_total counter
dsl_hooks_total{kind="pull_request"} 1
`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCountInvalidHook(t *testing.T) {
	m := New(prometheus.NewRegistry())
	m.CountInvalidHook()

	err := testutil.CollectAndCompare(m.invalidHooks, strings.NewReader(`
# HELP dsl_hooks_invalid Count of invalid hooks received
# TYPE dsl_hooks_invalid counter
dsl_hooks_invalid 1
`))
	if err != nil {
		t.Fatal(err)
	}
}
