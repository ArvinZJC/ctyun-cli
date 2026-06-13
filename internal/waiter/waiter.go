package waiter

import (
	"fmt"
	"strings"
)

type State string

const (
	Success State = "success"
	Failure State = "failure"
	Pending State = "pending"
	Timeout State = "timeout"
)

type Spec struct {
	Path    string
	Success string
	Failure string
}

func Evaluate(spec Spec, payload map[string]any) (State, error) {
	value, err := valueAtPath(payload, spec.Path)
	if err != nil {
		return Pending, err
	}
	text := fmt.Sprint(value)
	switch text {
	case spec.Success:
		return Success, nil
	case spec.Failure:
		return Failure, nil
	default:
		return Pending, nil
	}
}

func valueAtPath(value any, path string) (any, error) {
	current := value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path %q cannot read %q", path, part)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("path %q is missing %q", path, part)
		}
	}
	return current, nil
}
