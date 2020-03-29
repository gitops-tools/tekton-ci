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
		case "stages":
			cfg.Stages = normaliseStringSlice(v)
		default:
			cfg.Jobs = append(cfg.Jobs, parseJob(k, v))
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

func parseJob(name string, v interface{}) *Job {
	j := &Job{Name: name}
	for k, v := range v.(map[string]interface{}) {
		switch k {
		case "stage":
			j.Stage = v.(string)
		case "script":
			j.Script = normaliseStringSlice(v)
		}
	}
	return j
}
