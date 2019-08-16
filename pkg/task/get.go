package task

import (
	"encoding/json"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

func (ti *TaskInterface) ReadData(path string) ([]byte, error) {
	var spec map[string]interface{}

	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, ti.opts); err != nil {
		return nil, err
	}

	if path != "" {
		output, _ := taskutil.EvaluateJSONPath(path, spec)
		if output != nil {
			return output.Bytes(), nil
		}
	} else {
		output, _ := json.Marshal(spec)
		return output, nil
	}

	return nil, nil
}
