package opt

import (
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/serialize"
)

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

type SampleConfig struct {
	Secrets map[string]string           `yaml:"secrets"`
	Runs    map[string]*SampleConfigRun `yaml:"runs"`
}

func (sc *SampleConfig) AppendTo(other *SampleConfig) {
	for name, value := range sc.Secrets {
		other.Secrets[name] = value
	}

	for id, run := range sc.Runs {
		other.Runs[id] = run
	}
}
