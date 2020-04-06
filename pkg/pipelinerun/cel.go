package pipelinerun

import (
	"encoding/json"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
)

func evaluate(expr string, env *cel.Env, data map[string]interface{}) (ref.Val, error) {
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prg, err := env.Program(checked)
	if err != nil {
		return nil, err
	}

	out, _, err := prg.Eval(data)
	return out, err
}

func makeCelEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("hook", decls.Dyn, nil)))
}

func makeEvalContext(hook interface{}) (map[string]interface{}, error) {
	m, err := hookToMap(hook)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"hook": m}, nil
}

func hookToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err
}
