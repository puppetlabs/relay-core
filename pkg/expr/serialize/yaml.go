package serialize

import (
	"github.com/puppetlabs/nebula-tasks/pkg/expr/parse"
	"gopkg.in/yaml.v3"
)

type YAMLTree struct {
	parse.Tree
}

func (yt *YAMLTree) UnmarshalYAML(value *yaml.Node) error {
	tree, err := parse.ParseYAMLNode(value)
	if err != nil {
		return err
	}

	*yt = YAMLTree{Tree: tree}
	return nil
}
