package spec

import (
	"context"
	"errors"
	"fmt"

	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
	"github.com/puppetlabs/leg/relspec/pkg/relspec"
	"gopkg.in/yaml.v3"
)

type AnswerID struct {
	AskRef string `json:"askRef"`
	Name   string `json:"name"`
}

func (ai AnswerID) String() string {
	return fmt.Sprintf("answer %q of ask %q", ai.Name, ai.AskRef)
}

func (ai AnswerID) Less(other AnswerID) bool {
	return ai.AskRef < other.AskRef || (ai.AskRef == other.AskRef && ai.Name < other.Name)
}

type AnswerTypeResolver interface {
	ResolveAnswer(ctx context.Context, askRef, name string) (any, error)
}

type MemoryAnswerKey struct {
	AskRef string
	Name   string
}

type MemoryAnswerTypeResolver struct {
	m map[MemoryAnswerKey]any
}

var _ AnswerTypeResolver = &MemoryAnswerTypeResolver{}

func (mr *MemoryAnswerTypeResolver) ResolveAnswer(ctx context.Context, askRef, name string) (any, error) {
	o, ok := mr.m[MemoryAnswerKey{AskRef: askRef, Name: name}]
	if !ok {
		return "", ErrNotFound
	}

	return o, nil
}

func NewMemoryAnswerTypeResolver(m map[MemoryAnswerKey]any) *MemoryAnswerTypeResolver {
	return &MemoryAnswerTypeResolver{m: m}
}

var NoOpAnswerTypeResolver AnswerTypeResolver = NewMemoryAnswerTypeResolver(map[MemoryAnswerKey]any{})

type AnswerMappingTypeResolver struct {
	AnswerTypeResolver
}

var _ relspec.MappingTypeResolver[*References] = &AnswerMappingTypeResolver{}

func (amtr *AnswerMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	askRef, ok := tm["askRef"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "askRef"}
	}

	name, ok := tm["name"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "name"}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := amtr.ResolveAnswer(ctx, askRef, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Answers.Set(ref.Errored(AnswerID{AskRef: askRef, Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Answers.Set(ref.OK(AnswerID{AskRef: askRef, Name: name}))
		r.SetValue(value)
	}

	return r, nil
}

type YAMLAnswerTransformer struct{}

func (YAMLAnswerTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.ShortTag() != "!Answer" {
		return false, nil
	}

	var askRef, name *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 4 {
			return false, fmt.Errorf(`expected mapping-style !Answer to have exactly two keys, "askRef" and "name"`)
		}

		for i := 0; i < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "askRef":
				askRef = node.Content[i+1]
			case "name":
				name = node.Content[i+1]
			default:
				return false, fmt.Errorf(`expected mapping-style !Answer to have exactly two keys, "askRef" and "name"`)
			}
		}
	case yaml.SequenceNode:
		if len(node.Content) != 2 {
			return false, fmt.Errorf(`expected mapping-style !Answer to have exactly two items`)
		}

		askRef = node.Content[0]
		name = node.Content[1]
	default:
		return false, fmt.Errorf(`unexpected scalar value for !Answer, must be a mapping or sequence`)
	}

	// {$type: Answer, askRef: <askRef>, name: <name>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Answer"},
			{Kind: yaml.ScalarNode, Value: "askRef"},
			askRef,
			{Kind: yaml.ScalarNode, Value: "name"},
			name,
		},
	}
	return true, nil
}
