package cel

import (
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Context makes it easy to execute CEL expressions on a hook body.
type Context struct {
	env *cel.Env
	ctx map[string]interface{}
}

// New creates and returns a Context for evaluating expressions.
func New(hook interface{}) (*Context, error) {
	env, err := makeCelEnv()
	if err != nil {
		return nil, err
	}
	ctx, err := makeEvalContext(hook)
	if err != nil {
		return nil, err
	}
	return &Context{
		env: env,
		ctx: ctx,
	}, nil
}

// Evaluate evaluates the provided expression and returns the results of doing
// so.
func (c *Context) Evaluate(expr string) (ref.Val, error) {
	return evaluate(expr, c.env, c.ctx)
}

func (c *Context) EvaluateToString(expr string) (string, error) {
	res, err := c.Evaluate(expr)
	if err != nil {
		return "", err
	}
	return valToString(res)
}

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

// TODO: This should probably stringify other ref.Types
func valToString(v ref.Val) (string, error) {
	switch val := v.(type) {
	case types.String:
		return val.Value().(string), nil
	case types.Double:
		return fmt.Sprintf("%g", val.Value().(float64)), nil
	}
	return "", fmt.Errorf("unknown result type %T, expression must be a string", v)
}
