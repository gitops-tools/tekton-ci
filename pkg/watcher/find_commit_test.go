package watcher

import (
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

func TestFindCommit(t *testing.T) {
	pr := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{},
		},
	})
	pr.Status = pipelinev1.PipelineRunStatus{
		PipelineRunStatusFields: pipelinev1.PipelineRunStatusFields{
			TaskRuns: map[string]*pipelinev1.PipelineRunTaskRunStatus{
				"testing": {
					Status: &pipelinev1.TaskRunStatus{
						TaskRunStatusFields: pipelinev1.TaskRunStatusFields{
							ResourcesResult: []pipelinev1.PipelineResourceResult{
								pipelinev1.PipelineResourceResult{Key: "commit", Value: "9bb041d2f04027d96db99979c58531c3f6e39312"},
							},
						},
					},
				},
			},
		},
	}
}
