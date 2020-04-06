package pipelinerun

import (
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type ParamBinding struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type PipelineRun struct {
	Expression      string                     `yaml:"expression"`
	ParamBindings   []ParamBinding             `yaml:"param_bindings"`
	PipelineRunSpec pipelinev1.PipelineRunSpec `yaml:"pipeline_run"`
}
