package task

import (
	"encoding/json"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"gopkg.in/yaml.v2"
)

func (ti *TaskInterface) WriteFile(file, path, output string) error {
	var spec map[string]interface{}
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, ti.opts); err != nil {
		return err
	}

	data, err := GetData(spec, path, output)
	if err != nil {
		return err
	}

	if data != nil {
		return taskutil.WriteDataToFile(file, data)
	}

	return nil
}

func GetData(spec map[string]interface{}, path, output string) ([]byte, error) {
	switch output {
	case "json":
		return json.Marshal(spec[path])

	default:
		return yaml.Marshal(spec[path])
	}

	return nil, nil
}
