package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func PipelineRun(component, prName string, spec pipelinev1.PipelineRunSpec) *pipelinev1.PipelineRun {
	return &pipelinev1.PipelineRun{
		TypeMeta: metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prName,
			Annotations:  annotations(),
			Labels:       labels(component),
		},
		Spec: spec,
	}
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
