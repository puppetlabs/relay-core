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

type ConnectionID struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (ci ConnectionID) String() string {
	return fmt.Sprintf("connection type %q with name %q", ci.Type, ci.Name)
}

func (ci ConnectionID) Less(other ConnectionID) bool {
	return ci.Type < other.Type || (ci.Type == other.Type && ci.Name < other.Name)
}

type ConnectionTypeResolver interface {
	ResolveAllConnections(ctx context.Context) (map[string]map[string]any, error)
	ResolveTypeOfConnections(ctx context.Context, connectionType string) (map[string]any, error)
	ResolveConnection(ctx context.Context, connectionType, name string) (any, error)
}

type MemoryConnectionKey struct {
	Type string
	Name string
}

type MemoryConnectionTypeResolver struct {
	m map[MemoryConnectionKey]any
}

var _ ConnectionTypeResolver = &MemoryConnectionTypeResolver{}

func (mr *MemoryConnectionTypeResolver) ResolveAllConnections(ctx context.Context) (map[string]map[string]any, error) {
	if len(mr.m) == 0 {
		return nil, nil
	}

	cm := make(map[string]map[string]any)

	for k, c := range mr.m {
		tm, found := cm[k.Type]
		if !found {
			tm = make(map[string]any)
			cm[k.Type] = tm
		}

		tm[k.Name] = c
	}

	return cm, nil
}

func (mr *MemoryConnectionTypeResolver) ResolveTypeOfConnections(ctx context.Context, connectionType string) (map[string]any, error) {
	var tm map[string]any

	for k, c := range mr.m {
		if k.Type != connectionType {
			continue
		} else if tm == nil {
			tm = make(map[string]any)
		}

		tm[k.Name] = c
	}

	return tm, nil
}

func (mr *MemoryConnectionTypeResolver) ResolveConnection(ctx context.Context, connectionType, name string) (any, error) {
	o, ok := mr.m[MemoryConnectionKey{Type: connectionType, Name: name}]
	if !ok {
		return "", ErrNotFound
	}

	return o, nil
}

func NewMemoryConnectionTypeResolver(m map[MemoryConnectionKey]any) *MemoryConnectionTypeResolver {
	return &MemoryConnectionTypeResolver{m: m}
}

var NoOpConnectionTypeResolver ConnectionTypeResolver = NewMemoryConnectionTypeResolver(map[MemoryConnectionKey]any{})

type ConnectionMappingTypeResolver struct {
	ConnectionTypeResolver
}

var _ relspec.MappingTypeResolver[*References] = &ConnectionMappingTypeResolver{}

func (cmtr *ConnectionMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	connectionType, ok := tm["type"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "type"}
	}

	name, ok := tm["name"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "name"}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := cmtr.ResolveConnection(ctx, connectionType, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Connections.Set(ref.Errored(ConnectionID{Type: connectionType, Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Connections.Set(ref.OK(ConnectionID{Type: connectionType, Name: name}))
		r.SetValue(value)
	}

	return r, nil
}

type typeOfConnectionTemplateEnvironment struct {
	r   ConnectionTypeResolver
	typ string
}

var (
	_ eval.Indexable                   = &typeOfConnectionTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &typeOfConnectionTemplateEnvironment{}
)

func (tocte *typeOfConnectionTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := tocte.r.ResolveConnection(ctx, tocte.typ, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Connections.Set(ref.Errored(ConnectionID{Type: tocte.typ, Name: name}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Connections.Set(ref.OK(ConnectionID{Type: tocte.typ, Name: name}))
		r.SetValue(value)
	}

	return evaluate.StaticExpandable(r), nil
}

func (tocte *typeOfConnectionTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](tocte), nil
	}

	m, err := tocte.r.ResolveTypeOfConnections(ctx, tocte.typ)
	if err != nil {
		return nil, err
	}

	r := evaluate.NewResult(evaluate.NewMetadata(NewReferences()), m)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	for name := range m {
		r.References.Connections.Set(ref.OK(ConnectionID{Type: tocte.typ, Name: name}))
	}
	return r, nil
}

type ConnectionTemplateEnvironment struct {
	ConnectionTypeResolver
}

var (
	_ eval.Indexable                   = &ConnectionTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &ConnectionTemplateEnvironment{}
)

func (cte *ConnectionTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	typ, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	return &typeOfConnectionTemplateEnvironment{
		r:   cte.ConnectionTypeResolver,
		typ: typ,
	}, nil
}

func (cte *ConnectionTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](cte), nil
	}

	m, err := cte.ResolveAllConnections(ctx)
	if err != nil {
		return nil, err
	}

	refs := NewReferences()

	cm := make(map[string]any, len(m))
	for typ, v := range m {
		for name := range v {
			refs.Connections.Set(ref.OK(ConnectionID{Type: typ, Name: name}))
		}
		cm[typ] = v
	}

	r := evaluate.NewResult(evaluate.NewMetadata(refs), cm)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	return r, nil
}

type YAMLConnectionTransformer struct{}

func (YAMLConnectionTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.ShortTag() != "!Connection" {
		return false, nil
	}

	var connectionType, name *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 4 {
			return false, fmt.Errorf(`expected mapping-style !Connection to have exactly two keys, "type" and "name"`)
		}

		for i := 0; i < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "type":
				connectionType = node.Content[i+1]
			case "name":
				name = node.Content[i+1]
			default:
				return false, fmt.Errorf(`expected mapping-style !Connection to have exactly two keys, "type" and "name"`)
			}
		}
	case yaml.SequenceNode:
		if len(node.Content) != 2 {
			return false, fmt.Errorf(`expected mapping-style !Connection to have exactly two items`)
		}

		connectionType = node.Content[0]
		name = node.Content[1]
	default:
		return false, fmt.Errorf(`unexpected scalar value for !Connection, must be a mapping or sequence`)
	}

	// {$type: Connection, type: <type>, name: <name>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Connection"},
			{Kind: yaml.ScalarNode, Value: "type"},
			connectionType,
			{Kind: yaml.ScalarNode, Value: "name"},
			name,
		},
	}
	return true, nil
}
