package ci

import (
	"fmt"
	"io"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

// Parse decodes YAML describing a CI pipeline and returns the configuration.
func Parse(in io.Reader) (*Pipeline, error) {
	body, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML: %w", err)
	}

	raw := map[string]interface{}{}
	err = yaml.Unmarshal(body, &raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return parseRaw(raw)
}

func parseRaw(raw map[string]interface{}) (*Pipeline, error) {
	cfg := &Pipeline{}
	for k, v := range raw {
		switch k {
		case "image":
			cfg.Image = v.(string)
		case "variables":
			cfg.Variables = normaliseStringMap(v)
		case "before_script":
			cfg.BeforeScript = normaliseStringSlice(v)
		case "after_script":
			cfg.AfterScript = normaliseStringSlice(v)
		case "stages":
			cfg.Stages = normaliseStringSlice(v)
		default:
			task, err := parseTask(k, v)
			if err != nil {
				return nil, err
			}
			cfg.Tasks = append(cfg.Tasks, task)
		}
	}
	return cfg, nil
}

func normaliseStringMap(vars interface{}) map[string]string {
	newVars := map[string]string{}
	for k, v := range vars.(map[string]interface{}) {
		newVars[k] = v.(string)
	}
	return newVars
}

func normaliseStringSlice(vars interface{}) []string {
	strings := []string{}
	for _, v := range vars.([]interface{}) {
		strings = append(strings, v.(string))
	}
	return strings
}

func parseTask(name string, v interface{}) (*Task, error) {
	t := &Task{Name: name}
	for k, v := range v.(map[string]interface{}) {
		switch k {
		case "stage":
			t.Stage = v.(string)
		case "script":
			t.Script = normaliseStringSlice(v)
		case "artifacts":
			artifacts, err := parseArtifacts(k, v)
			if err != nil {
				return nil, err
			}
			t.Artifacts = artifacts
		}
	}
	if len(t.Script) == 0 {
		return nil, fmt.Errorf("invalid task %#v missing script", name)
	}
	return t, nil
}

func parseArtifacts(name string, v interface{}) (Artifacts, error) {
	a := Artifacts{Paths: []string{}}
	for k, v := range v.(map[string]interface{}) {
		switch k {
		case "paths":
			a.Paths = normaliseStringSlice(v)
		}
	}
	return a, nil
}
