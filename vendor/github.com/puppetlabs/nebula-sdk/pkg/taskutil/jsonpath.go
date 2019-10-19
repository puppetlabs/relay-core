package taskutil

import (
	"bytes"

	"k8s.io/client-go/util/jsonpath"
)

func EvaluateJSONPath(jsonPath string, data interface{}) (*bytes.Buffer, error) {
	j := jsonpath.New("expression")
	if err := j.Parse(jsonPath); err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err := j.Execute(buf, data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
