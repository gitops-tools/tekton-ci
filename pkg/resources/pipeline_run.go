package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// PipelineRunOpt is a type that can modify a PipelineRun after it's created,
// but before it's returned.
type PipelineRunOpt func(*pipelinev1.PipelineRun)

// PipelineRun creates a PipelineRun with the name, standard labels, and the
// provided Spec.
func PipelineRun(component, prName string, spec pipelinev1.PipelineRunSpec, options ...PipelineRunOpt) *pipelinev1.PipelineRun {
	pr := &pipelinev1.PipelineRun{
		TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prName,
			Labels:       labels(component),
			Annotations:  map[string]string{},
		},
		Spec: spec,
	}

	for _, o := range options {
		o(pr)
	}
	return pr
}

func labels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": component,
		"app.kubernetes.io/part-of":    "Tekton-CI",
	}
}
