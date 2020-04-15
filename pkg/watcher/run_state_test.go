package watcher

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

var (
	testNamespace   = "test-namespace"
	pipelineRunName = "test-pipeline-run"
)

func TestGetPipelineRunStatus(t *testing.T) {
	statusTests := []struct {
		conditionType   apis.ConditionType
		conditionStatus corev1.ConditionStatus
		want            State
	}{
		{apis.ConditionSucceeded, corev1.ConditionTrue, Successful},
		{apis.ConditionSucceeded, corev1.ConditionUnknown, Pending},
		{apis.ConditionSucceeded, corev1.ConditionFalse, Failed},
	}

	for _, tt := range statusTests {
		s := getPipelineRunState(makePipelineRunWithCondition(apis.Condition{Type: tt.conditionType, Status: tt.conditionStatus}))
		if s != tt.want {
			t.Errorf("getPipelineRunState(%s) got %v, want %v", tt.conditionStatus, s, tt.want)
		}
	}
}

func makePipelineRunWithCondition(condition apis.Condition) *pipelinev1.PipelineRun {
	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		Workspaces: []pipelinev1.WorkspaceBinding{},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks:      []pipelinev1.PipelineTask{},
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{},
		},
	})

	pr.Status.Conditions = append(pr.Status.Conditions, condition)
	return pr
}
