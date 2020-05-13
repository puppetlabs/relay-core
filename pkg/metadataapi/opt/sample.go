package opt

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/serialize"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/memory"
	"gopkg.in/yaml.v3"
)

type SampleConfigConnections map[memory.ConnectionKey]map[string]interface{}

func (scc *SampleConfigConnections) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]map[string]interface{}
	if err := value.Decode(&m); err != nil {
		return err
	}

	*scc = make(SampleConfigConnections, len(m))
	for tn, attrs := range m {
		parts := strings.SplitN(tn, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("connection keys must be in the format `<type>/<name>`")
		}

		(*scc)[memory.ConnectionKey{Type: parts[0], Name: parts[1]}] = attrs
	}

	return nil
}

type SampleConfigSpec map[string]serialize.YAMLTree

func (sp SampleConfigSpec) Interface() map[string]interface{} {
	copy := make(map[string]interface{})

	for k, v := range sp {
		copy[k] = v.Tree
	}

	return copy
}

type SampleConfigStep struct {
	Conditions serialize.YAMLTree     `yaml:"conditions"`
	Spec       SampleConfigSpec       `yaml:"spec"`
	Outputs    map[string]interface{} `yaml:"outputs"`
	State      map[string]interface{} `yaml:"state"`
}

type SampleConfigRun struct {
	Steps map[string]*SampleConfigStep `yaml:"steps"`
}

type SampleConfigTrigger struct{}

type SampleConfig struct {
	Connections SampleConfigConnections         `yaml:"connections"`
	Secrets     map[string]string               `yaml:"secrets"`
	Runs        map[string]*SampleConfigRun     `yaml:"runs"`
	Triggers    map[string]*SampleConfigTrigger `yaml:"triggers"`
}

func (sc *SampleConfig) AppendTo(other *SampleConfig) {
	for name, attrs := range sc.Connections {
		other.Connections[name] = attrs
	}

	for name, value := range sc.Secrets {
		other.Secrets[name] = value
	}

	for id, run := range sc.Runs {
		other.Runs[id] = run
	}

	for name, trigger := range sc.Triggers {
		other.Triggers[name] = trigger
	}
}
