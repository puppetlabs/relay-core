package spec

import (
	"context"
	"errors"
	"fmt"

	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/pathlang"
	"github.com/puppetlabs/leg/relspec/pkg/query"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
	"github.com/puppetlabs/leg/relspec/pkg/relspec"
	"gopkg.in/yaml.v3"
)

type DataID struct {
	Name string `json:"name"`
}

func (di DataID) Less(other DataID) bool {
	return di.Name < other.Name
}

type DataResolverNotFoundError struct {
	Name string
}

func (e *DataResolverNotFoundError) Error() string {
	if e.Name == "" {
		return "default data resolver could not be found"
	}

	return fmt.Sprintf("data resolver %q could not be found", e.Name)
}

type DataQueryError struct {
	Query string
	Cause error
}

func (e *DataQueryError) Unwrap() error {
	return e.Cause
}

func (e *DataQueryError) Error() string {
	return fmt.Sprintf("query %q: %v", e.Query, e.Cause)
}

type DataTypeResolver interface {
	ResolveData(ctx context.Context) (any, error)
}

type MemoryDataTypeResolver struct {
	value any
}

var _ DataTypeResolver = &MemoryDataTypeResolver{}

func (mr *MemoryDataTypeResolver) ResolveData(ctx context.Context) (any, error) {
	return mr.value, nil
}

func NewMemoryDataTypeResolver(value any) *MemoryDataTypeResolver {
	return &MemoryDataTypeResolver{value: value}
}

type noOpDataTypeResolver struct{}

func (*noOpDataTypeResolver) ResolveData(ctx context.Context) (any, error) {
	return nil, ErrNotFound
}

var NoOpDataTypeResolver DataTypeResolver = &noOpDataTypeResolver{}

type DataMappingTypeResolver map[string]DataTypeResolver

var _ relspec.MappingTypeResolver[*References] = DataMappingTypeResolver(nil)

func (dmtr DataMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	// No name indicates that we should the default resolver, which is
	// always the empty string.
	name, _ := tm["name"].(string)

	q, ok := tm["query"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "query"}
	}

	resolver, found := dmtr[name]
	if !found {
		return nil, &DataResolverNotFoundError{Name: name}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	d, err := resolver.ResolveData(ctx)
	if errors.Is(err, ErrNotFound) {
		r.References.Data.Set(ref.Errored(DataID{Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Data.Set(ref.OK(DataID{Name: name}))

		qr, err := query.EvaluateQuery(ctx, evaluate.DefaultEvaluator[ref.EmptyReferences](), pathlang.New[ref.EmptyReferences]().Expression, d, q)
		if err != nil {
			return nil, &DataQueryError{Query: q, Cause: err}
		}

		r.SetValue(qr.Value)
	}

	return r, nil
}

type DataTemplateEnvironment struct {
	DataTypeResolver
	Name string
}

var (
	_ eval.Indexable                   = &DataTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &DataTemplateEnvironment{}
)

func (dte *DataTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	d, err := dte.ResolveData(ctx)
	if errors.Is(err, ErrNotFound) {
		r.References.Data.Set(ref.Errored(DataID{Name: dte.Name}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Data.Set(ref.OK(DataID{Name: dte.Name}))

		v, err := eval.Select(ctx, d, idx)
		if err != nil {
			return nil, err
		}

		r.SetValue(v)
	}

	return evaluate.StaticExpandable(r), nil
}

func (dte *DataTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](dte), nil
	}

	d, err := dte.ResolveData(ctx)
	if err != nil {
		return nil, err
	}

	r := evaluate.NewResult(evaluate.NewMetadata(NewReferences()), d)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	r.References.Data.Set(ref.OK(DataID{Name: dte.Name}))

	return r, nil
}

type YAMLDataTransformer struct{}

func (YAMLDataTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.Tag != "!Data" {
		return false, nil
	}

	var query *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 2 || node.Content[0].Value != "query" {
			return false, fmt.Errorf(`expected mapping-style !Data to have exactly one key, "query"`)
		}

		query = node.Content[1]
	case yaml.SequenceNode:
		if len(node.Content) != 1 {
			return false, fmt.Errorf(`expected sequence-style !Data to have exactly one item`)
		}

		query = node.Content[0]
	case yaml.ScalarNode:
		query = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: node.Value,
		}
	}

	// {$type: Data, query: <query>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Data"},
			{Kind: yaml.ScalarNode, Value: "query"},
			query,
		},
	}
	return true, nil
}
