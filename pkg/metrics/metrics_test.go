package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gitops-tools/tekton-ci/test/hook"
)

var _ Interface = (*PrometheusMetrics)(nil)

func TestCountHook(t *testing.T) {
	m := New("dsl", prometheus.NewRegistry())
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
	m := New("dsl", prometheus.NewRegistry())
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

func TestCountAPICall(t *testing.T) {
	m := New("dsl", prometheus.NewRegistry())
	m.CountAPICall("file_contents")

	err := testutil.CollectAndCompare(m.apiCalls, strings.NewReader(`
# HELP dsl_api_calls_total Count of API Calls made
# TYPE dsl_api_calls_total counter
dsl_api_calls_total{kind="file_contents"} 1
`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCountFailedAPICall(t *testing.T) {
	m := New("dsl", prometheus.NewRegistry())
	m.CountFailedAPICall("commit_status")

	err := testutil.CollectAndCompare(m.failedAPICalls, strings.NewReader(`
# HELP dsl_failed_api_calls_total Count of failed API Calls made
# TYPE dsl_failed_api_calls_total counter
dsl_failed_api_calls_total{kind="commit_status"} 1
`))
	if err != nil {
		t.Fatal(err)
	}
}
