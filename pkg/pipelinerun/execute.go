package pipelinerun

import (
	"fmt"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Execute takes a PipelineDefinition and a hook, and returns a PipelineRun
// or an error.
func Execute(pd *PipelineDefinition, hook interface{}, name string) (*pipelinev1.PipelineRun, error) {
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

	pr := &pipelinev1.PipelineRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: name},
		Spec:       pd.PipelineRunSpec,
	}
	return pr, nil
}

// TODO: This should probably stringify other ref.Types
func valToString(v ref.Val) pipelinev1.ArrayOrString {
	switch val := v.(type) {
	case types.String:
		return pipelinev1.ArrayOrString{StringVal: val.Value().(string), Type: "string"}
	}
	return pipelinev1.ArrayOrString{StringVal: "unknown", Type: "string"}
}
