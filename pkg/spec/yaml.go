package spec

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAMLTransformer interface {
	Transform(node *yaml.Node) (bool, error)
}

type YAMLInvocationTransformer struct{}

func (YAMLInvocationTransformer) Transform(node *yaml.Node) (bool, error) {
	tag := node.ShortTag()
	prefix := "!Fn."

	if !strings.HasPrefix(tag, prefix) {
		return false, nil
	}

	name := tag[len(prefix):]
	if len(name) == 0 {
		return false, fmt.Errorf(`expected function name to have the syntax !Fn.<name>`)
	}

	var args *yaml.Node
	switch node.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		args = &yaml.Node{
			Kind:    node.Kind,
			Content: node.Content,
		}
	case yaml.ScalarNode:
		args = &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: node.Value},
			},
		}
	}

	// {$fn.<name>: <args>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: fmt.Sprintf("$fn.%s", name)},
			args,
		},
	}
	return true, nil
}

type YAMLBinaryToEncodingTransformer struct{}

func (YAMLBinaryToEncodingTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.ShortTag() != "!!binary" || node.Kind != yaml.ScalarNode {
		return false, nil
	}

	// {$encoding: base64, data: <value>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$encoding"},
			{Kind: yaml.ScalarNode, Value: "base64"},
			{Kind: yaml.ScalarNode, Value: "data"},
			{Kind: yaml.ScalarNode, Value: node.Value},
		},
	}
	return true, nil
}

type YAMLUnknownTagTransformer struct{}

func (YAMLUnknownTagTransformer) Transform(node *yaml.Node) (bool, error) {
	if tag := node.ShortTag(); tag != "" && !strings.HasPrefix(tag, "!!") {
		return false, fmt.Errorf(`unknown tag %q`, node.ShortTag())
	}

	return false, nil
}

var YAMLTransformers = []YAMLTransformer{
	YAMLDataTransformer{},
	YAMLSecretTransformer{},
	YAMLConnectionTransformer{},
	YAMLOutputTransformer{},
	YAMLParameterTransformer{},
	YAMLAnswerTransformer{},
	YAMLInvocationTransformer{},
	YAMLBinaryToEncodingTransformer{},
	YAMLUnknownTagTransformer{},
}

func ParseYAML(r io.Reader) (any, error) {
	node := &yaml.Node{}
	if err := yaml.NewDecoder(r).Decode(node); err != nil {
		return nil, err
	}

	return ParseYAMLNode(node)
}

func ParseYAMLString(data string) (any, error) {
	return ParseYAML(strings.NewReader(data))
}

func ParseYAMLNode(node *yaml.Node) (any, error) {
	stack := []*yaml.Node{node}
	for len(stack) > 0 {
		node := stack[0]

		for node.Kind == yaml.AliasNode {
			node = node.Alias
		}

		for _, t := range YAMLTransformers {
			if ok, err := t.Transform(node); err != nil {
				return nil, err
			} else if ok {
				break
			}
		}

		// Remove head and append children for further analysis.
		stack = append(stack[1:], node.Content...)
	}

	var tree any
	if err := node.Decode(&tree); err != nil {
		return nil, err
	}

	return tree, nil
}

type YAMLTree struct {
	Tree any
}

func (yt *YAMLTree) UnmarshalYAML(value *yaml.Node) error {
	tree, err := ParseYAMLNode(value)
	if err != nil {
		return err
	}

	*yt = YAMLTree{Tree: tree}
	return nil
}

func (yt YAMLTree) MarshalYAML() (any, error) {
	return yt.Tree, nil
}
