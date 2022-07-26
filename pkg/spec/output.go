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

type OutputID struct {
	From string `json:"from"`
	Name string `json:"name"`
}

func (oi OutputID) String() string {
	return fmt.Sprintf("output %q of step %q", oi.Name, oi.From)
}

func (oi OutputID) Less(other OutputID) bool {
	return oi.From < other.From || (oi.From == other.From && oi.Name < other.Name)
}

type OutputTypeResolver interface {
	ResolveAllOutputs(ctx context.Context) (map[string]map[string]any, error)
	ResolveStepOutputs(ctx context.Context, from string) (map[string]any, error)
	ResolveOutput(ctx context.Context, from, name string) (any, error)
}

type MemoryOutputKey struct {
	From string
	Name string
}

type MemoryOutputTypeResolver struct {
	m map[MemoryOutputKey]any
}

var _ OutputTypeResolver = &MemoryOutputTypeResolver{}

func (mr *MemoryOutputTypeResolver) ResolveAllOutputs(ctx context.Context) (map[string]map[string]any, error) {
	if len(mr.m) == 0 {
		return nil, nil
	}

	om := make(map[string]map[string]any)

	for k, c := range mr.m {
		sm, found := om[k.From]
		if !found {
			sm = make(map[string]any)
			om[k.From] = sm
		}

		sm[k.Name] = c
	}

	return om, nil
}

func (mr *MemoryOutputTypeResolver) ResolveStepOutputs(ctx context.Context, from string) (map[string]any, error) {
	var sm map[string]any

	for k, c := range mr.m {
		if k.From != from {
			continue
		} else if sm == nil {
			sm = make(map[string]any)
		}

		sm[k.Name] = c
	}

	return sm, nil
}

func (mr *MemoryOutputTypeResolver) ResolveOutput(ctx context.Context, from, name string) (any, error) {
	o, ok := mr.m[MemoryOutputKey{From: from, Name: name}]
	if !ok {
		return "", ErrNotFound
	}

	return o, nil
}

func NewMemoryOutputTypeResolver(m map[MemoryOutputKey]any) *MemoryOutputTypeResolver {
	return &MemoryOutputTypeResolver{m: m}
}

var NoOpOutputTypeResolver OutputTypeResolver = NewMemoryOutputTypeResolver(map[MemoryOutputKey]any{})

type OutputMappingTypeResolver struct {
	OutputTypeResolver
}

var _ relspec.MappingTypeResolver[*References] = &OutputMappingTypeResolver{}

func (omtr *OutputMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	from, ok := tm["from"].(string)
	if !ok {
		// Fall back to old syntax.
		//
		// TODO: Remove this in a second version.
		from, ok = tm["taskName"].(string)
		if !ok {
			return nil, &FieldNotFoundError{Name: "from"}
		}
	}

	name, ok := tm["name"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "name"}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := omtr.ResolveOutput(ctx, from, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Outputs.Set(ref.Errored(OutputID{From: from, Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Outputs.Set(ref.OK(OutputID{From: from, Name: name}))
		r.SetValue(value)
	}

	return r, nil
}

type stepOutputTemplateEnvironment struct {
	r    OutputTypeResolver
	from string
}

var (
	_ eval.Indexable                   = &stepOutputTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &stepOutputTemplateEnvironment{}
)

func (sote *stepOutputTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := sote.r.ResolveOutput(ctx, sote.from, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Outputs.Set(ref.Errored(OutputID{From: sote.from, Name: name}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Outputs.Set(ref.OK(OutputID{From: sote.from, Name: name}))
		r.SetValue(value)
	}

	return evaluate.StaticExpandable(r), nil
}

func (sote *stepOutputTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](sote), nil
	}

	m, err := sote.r.ResolveStepOutputs(ctx, sote.from)
	if err != nil {
		return nil, err
	}

	r := evaluate.NewResult(evaluate.NewMetadata(NewReferences()), m)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	for name := range m {
		r.References.Outputs.Set(ref.OK(OutputID{From: sote.from, Name: name}))
	}
	return r, nil
}

type OutputTemplateEnvironment struct {
	OutputTypeResolver
}

var (
	_ eval.Indexable                   = &OutputTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &OutputTemplateEnvironment{}
)

func (ote *OutputTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	from, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	return &stepOutputTemplateEnvironment{
		r:    ote.OutputTypeResolver,
		from: from,
	}, nil
}

func (ote *OutputTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](ote), nil
	}

	m, err := ote.ResolveAllOutputs(ctx)
	if err != nil {
		return nil, err
	}

	refs := NewReferences()

	cm := make(map[string]any, len(m))
	for from, v := range m {
		for name := range v {
			refs.Outputs.Set(ref.OK(OutputID{From: from, Name: name}))
		}
		cm[from] = v
	}

	return evaluate.NewResult(evaluate.NewMetadata(refs), cm), nil
}

type YAMLOutputTransformer struct{}

func (YAMLOutputTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.ShortTag() != "!Output" {
		return false, nil
	}

	var from, name *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 4 {
			return false, fmt.Errorf(`expected mapping-style !Output to have exactly two keys, "from" and "name"`)
		}

		for i := 0; i < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "from":
				from = node.Content[i+1]
			case "name":
				name = node.Content[i+1]
			default:
				return false, fmt.Errorf(`expected mapping-style !Output to have exactly two keys, "from" and "name"`)
			}
		}
	case yaml.SequenceNode:
		if len(node.Content) != 2 {
			return false, fmt.Errorf(`expected mapping-style !Output to have exactly two items`)
		}

		from = node.Content[0]
		name = node.Content[1]
	default:
		return false, fmt.Errorf(`unexpected scalar value for !Output, must be a mapping or sequence`)
	}

	// {$type: Output, from: <from>, name: <name>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Output"},
			{Kind: yaml.ScalarNode, Value: "from"},
			from,
			{Kind: yaml.ScalarNode, Value: "name"},
			name,
		},
	}
	return true, nil
}
