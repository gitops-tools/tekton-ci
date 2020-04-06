package pipelinerun

import (
	"fmt"
	"io"
	"io/ioutil"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

type ParamBinding struct {
	Name       string `yaml:"name"`
	Expression string `yaml:"expression"`
}

type PipelineDefinition struct {
	Filter          string                     `yaml:"expression"`
	ParamBindings   []ParamBinding             `yaml:"param_bindings"`
	PipelineRunSpec pipelinev1.PipelineRunSpec `yaml:"pipeline_run_spec"`
}

// Parse decodes YAML describing a PipelineDefinition and returns the resource.
func Parse(in io.Reader) (*PipelineDefinition, error) {
	body, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML: %w", err)
	}

	var pr PipelineDefinition
	err = yaml.Unmarshal(body, &pr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &pr, nil
}
