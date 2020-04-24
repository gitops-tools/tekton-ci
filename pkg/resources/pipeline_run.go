package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type pipelineRunOption func(*pipelinev1.PipelineRun)

// PipelineRun creates a PipelineRun with the name, standard labels, and the
// provided Spec.
func PipelineRun(component, prName string, spec pipelinev1.PipelineRunSpec, options ...pipelineRunOption) *pipelinev1.PipelineRun {
	pr := &pipelinev1.PipelineRun{
		TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prName,
			Annotations:  annotations(),
			Labels:       labels(component),
		},
		Spec: spec,
	}

	for _, o := range options {
		o(pr)
	}
	return pr
}

func annotations() map[string]string {
	return map[string]string{
		"tekton.dev/git-status":     "true",
		"tekton.dev/status-context": "tekton-ci",
	}
}

func labels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": component,
		"app.kubernetes.io/part-of":    "Tekton-CI",
	}
}
