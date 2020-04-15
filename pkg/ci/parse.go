package ci

import (
	"fmt"
	"io"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

const DefaultStage = "default"

// Parse decodes YAML describing a CI pipeline and returns the configuration.
//
// Decoded tasks are given put into the "default" Stage.
//
// If no explicit ordering of the Stages is provided, they're subject to hash
// ordering.
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
			cfg.Variables = stringMap(v)
		case "before_script":
			cfg.BeforeScript = stringSlice(v)
		case "after_script":
			cfg.AfterScript = stringSlice(v)
		case "stages":
			cfg.Stages = stringSlice(v)
		default:
			task, err := parseTask(k, v)
			if err != nil {
				return nil, err
			}
			cfg.Tasks = append(cfg.Tasks, task)
		}
	}
	applyDefaultsToPipeline(cfg)
	return cfg, nil
}

func applyDefaultsToPipeline(p *Pipeline) {
	if len(p.Stages) == 0 {
		p.Stages = findStages(p.Tasks)
	}
}

func stringMap(vars interface{}) map[string]string {
	newVars := map[string]string{}
	for k, v := range vars.(map[string]interface{}) {
		newVars[k] = v.(string)
	}
	return newVars
}

func stringSlice(vars interface{}) []string {
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
			t.Script = stringSlice(v)
		case "tekton":
			tekton, err := parseTekton(v)
			if err != nil {
				return nil, err
			}
			t.Tekton = tekton
		case "rules":
			t.Rules = parseRules(v)
		case "artifacts":
			t.Artifacts = parseArtifacts(v)
		}
	}
	if len(t.Script) == 0 && t.Tekton == nil {
		return nil, fmt.Errorf("invalid task %#v: missing script", name)
	}
	if len(t.Script) > 0 && t.Tekton != nil {
		return nil, fmt.Errorf("invalid task %#v: provided Tekton task and Script", name)
	}
	if t.Stage == "" {
		t.Stage = DefaultStage
	}
	return t, nil
}

func parseArtifacts(v interface{}) Artifacts {
	a := Artifacts{Paths: []string{}}
	for k, v := range v.(map[string]interface{}) {
		switch k {
		case "paths":
			a.Paths = stringSlice(v)
		}
	}
	return a
}

func parseTekton(v interface{}) (*TektonTask, error) {
	t := &TektonTask{}
	for k, v := range v.(map[string]interface{}) {
		switch k {
		case "taskRef":
			t.TaskRef = v.(string)
		case "serviceAccountName":
			t.ServiceAccountName = v.(string)
		case "params":
			params, err := parseTektonTaskParams(v)
			if err != nil {
				return nil, err
			}
			t.Params = params
		}
	}
	return t, nil
}

func parseRules(v interface{}) []Rule {
	rules := []Rule{}
	for _, rule := range v.([]interface{}) {
		currentRule := Rule{}
		for k, v := range rule.(map[string]interface{}) {
			switch k {
			case "if":
				currentRule.If = v.(string)
			case "when":
				currentRule.When = v.(string)
			}
		}
		rules = append(rules, currentRule)
	}
	return rules
}

func findStages(tasks []*Task) []string {
	foundStages := map[string]bool{}
	for _, t := range tasks {
		foundStages[t.Stage] = true
	}
	stages := []string{}
	for k := range foundStages {
		stages = append(stages, k)
	}
	if len(stages) > 0 {
		return stages
	}
	return []string{DefaultStage}
}

// TODO: this should validate params.
func parseTektonTaskParams(v interface{}) ([]TektonTaskParam, error) {
	params := []TektonTaskParam{}
	for _, rule := range v.([]interface{}) {
		param := TektonTaskParam{}
		for k, v := range rule.(map[string]interface{}) {
			switch k {
			case "name":
				param.Name = v.(string)
			case "expr":
				param.Expression = v.(string)
			}
		}
		if param.Expression == "" || param.Name == "" {
			return nil, fmt.Errorf("bad Tekton task parameter: %#v", v)
		}
		params = append(params, param)
	}
	return params, nil
}
