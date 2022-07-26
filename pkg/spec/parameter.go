package spec

import (
	"context"
	"errors"
	"fmt"

	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
	"github.com/puppetlabs/leg/relspec/pkg/relspec"
	"gopkg.in/yaml.v3"
)

type ParameterID struct {
	Name string `json:"name"`
}

func (pi ParameterID) String() string {
	return fmt.Sprintf("parameter %q", pi.Name)
}

func (pi ParameterID) Less(other ParameterID) bool {
	return pi.Name < other.Name
}

type ParameterTypeResolver interface {
	ResolveAllParameters(ctx context.Context) (map[string]any, error)
	ResolveParameter(ctx context.Context, name string) (any, error)
}

type MemoryParameterTypeResolver struct {
	m map[string]any
}

var _ ParameterTypeResolver = &MemoryParameterTypeResolver{}

func (mr *MemoryParameterTypeResolver) ResolveAllParameters(ctx context.Context) (map[string]any, error) {
	return mr.m, nil
}

func (mr *MemoryParameterTypeResolver) ResolveParameter(ctx context.Context, name string) (any, error) {
	p, ok := mr.m[name]
	if !ok {
		return nil, ErrNotFound
	}

	return p, nil
}

func NewMemoryParameterTypeResolver(m map[string]any) *MemoryParameterTypeResolver {
	return &MemoryParameterTypeResolver{m: m}
}

var NoOpParameterTypeResolver ParameterTypeResolver = NewMemoryParameterTypeResolver(map[string]any{})

type ParameterMappingTypeResolver struct {
	ParameterTypeResolver
}

var _ relspec.MappingTypeResolver[*References] = &ParameterMappingTypeResolver{}

func (pmtr *ParameterMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	name, ok := tm["name"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "name"}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := pmtr.ResolveParameter(ctx, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Parameters.Set(ref.Errored(ParameterID{Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Parameters.Set(ref.OK(ParameterID{Name: name}))
		r.SetValue(value)
	}

	return r, nil
}

type ParameterTemplateEnvironment struct {
	ParameterTypeResolver
}

var (
	_ eval.Indexable                   = &ParameterTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &ParameterTemplateEnvironment{}
)

func (pte *ParameterTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := pte.ResolveParameter(ctx, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Parameters.Set(ref.Errored(ParameterID{Name: name}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Parameters.Set(ref.OK(ParameterID{Name: name}))
		r.SetValue(value)
	}

	return evaluate.StaticExpandable(r), nil
}

func (pte *ParameterTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](pte), nil
	}

	m, err := pte.ResolveAllParameters(ctx)
	if err != nil {
		return nil, err
	}

	r := evaluate.NewResult(evaluate.NewMetadata(NewReferences()), m)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	for name := range m {
		r.References.Parameters.Set(ref.OK(ParameterID{Name: name}))
	}
	return r, nil
}

type YAMLParameterTransformer struct{}

func (YAMLParameterTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.ShortTag() != "!Parameter" {
		return false, nil
	}

	var name *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 2 || node.Content[0].Value != "name" {
			return false, fmt.Errorf(`expected mapping-style !Parameter to have exactly one key, "name"`)
		}

		name = node.Content[1]
	case yaml.SequenceNode:
		if len(node.Content) != 1 {
			return false, fmt.Errorf(`expected sequence-style !Parameter to have exactly one item`)
		}

		name = node.Content[0]
	case yaml.ScalarNode:
		name = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: node.Value,
		}
	}

	// {$type: Parameter, name: <name>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Parameter"},
			{Kind: yaml.ScalarNode, Value: "name"},
			name,
		},
	}
	return true, nil
}
