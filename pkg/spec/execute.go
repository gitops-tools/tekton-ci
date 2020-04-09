package spec

import (
	"fmt"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

// Execute takes a PipelineDefinition and a hook, and returns a PipelineRun
// and possibly an error.
//
// The Filter on the definition is evaluated, and if it returns true, then the
// ParamBindings are evaluated and appended to the PipelineRunSpec's Params.
//
// Finally a PipelineRun is returned, populated with the spec from the
// definition.
func Execute(pd *PipelineDefinition, hook interface{}, generateName string) (*pipelinev1.PipelineRun, error) {
	env, err := makeCelEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to make CEL environment: %w", err)
	}
	ectx, err := makeEvalContext(hook)
	if err != nil {
		return nil, fmt.Errorf("failed to make CEL context: %w", err)
	}

	if pd.Filter != "" {
		match, err := evaluate(pd.Filter, env, ectx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate the expression '%s': %w", pd.Filter, err)
		}

		// TODO: should this return a specific type, so that the HTTP endpoint
		// can decide whether or not this is actually an error, or merely a
		// signal that it should not continue?
		if match != types.True {
			return nil, fmt.Errorf("expression %s did not return true", pd.Filter)
		}
	}
	for _, v := range pd.ParamBindings {
		evaluated, err := evaluate(v.Expression, env, ectx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate the expression '%s': %w", v.Expression, err)
		}
		pd.PipelineRunSpec.Params = append(pd.PipelineRunSpec.Params, pipelinev1.Param{Name: v.Name, Value: valToString(evaluated)})
	}
	return resources.PipelineRun("pipelineRun", generateName, pd.PipelineRunSpec), nil
}

// TODO: This should probably stringify other ref.Types
func valToString(v ref.Val) pipelinev1.ArrayOrString {
	switch val := v.(type) {
	case types.String:
		return pipelinev1.ArrayOrString{StringVal: val.Value().(string), Type: "string"}
	}
	return pipelinev1.ArrayOrString{StringVal: "unknown", Type: "string"}
}

func trackerAnnotations() map[string]string {
	return map[string]string{
		"tekton.dev/git-status":     "true",
		"tekton.dev/status-context": "tekton-ci",
	}
}
