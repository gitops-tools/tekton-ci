package spec

import (
	"fmt"

	"github.com/google/cel-go/common/types"
	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	"github.com/gitops-tools/tekton-ci/pkg/cel"
	"github.com/gitops-tools/tekton-ci/pkg/resources"
)

// Execute takes a PipelineDefinition and a hook, and returns a PipelineRun
// and possibly an error.
//
// The Filter on the definition is evaluated, and if it returns true, then the
// ParamBindings are evaluated and appended to the PipelineRunSpec's Params.
//
// Finally a PipelineRun is returned, populated with the spec from the
// definition.
func Execute(pd *PipelineDefinition, hook scm.Webhook, generateName string) (*pipelinev1.PipelineRun, error) {
	ctx, err := cel.New(hook)
	if err != nil {
		return nil, fmt.Errorf("failed to make CEL environment: %w", err)
	}
	if pd.Filter != "" {
		match, err := ctx.Evaluate(pd.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate the expression '%s': %w", pd.Filter, err)
		}
		if match != types.True {
			return nil, fmt.Errorf("expression %s did not return true", pd.Filter)
		}
	}
	for _, v := range pd.ParamBindings {
		evaluated, err := ctx.EvaluateToString(v.Expression)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate the expression '%s': %w", v.Expression, err)
		}
		pd.PipelineRunSpec.Params = append(pd.PipelineRunSpec.Params, pipelinev1.Param{Name: v.Name, Value: valToString(evaluated)})
	}
	return resources.PipelineRun("pipelineRun", generateName, pd.PipelineRunSpec), nil
}

func valToString(v string) pipelinev1.ParamValue {
	return pipelinev1.ParamValue{StringVal: v, Type: "string"}
}
