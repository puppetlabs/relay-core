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

type SecretID struct {
	Name string `json:"name"`
}

func (si SecretID) String() string {
	return fmt.Sprintf("secret %q", si.Name)
}

func (si SecretID) Less(other SecretID) bool {
	return si.Name < other.Name
}

type SecretTypeResolver interface {
	ResolveAllSecrets(ctx context.Context) (map[string]string, error)
	ResolveSecret(ctx context.Context, name string) (string, error)
}

type MemorySecretTypeResolver struct {
	m map[string]string
}

var _ SecretTypeResolver = &MemorySecretTypeResolver{}

func (mr *MemorySecretTypeResolver) ResolveAllSecrets(ctx context.Context) (map[string]string, error) {
	return mr.m, nil
}

func (mr *MemorySecretTypeResolver) ResolveSecret(ctx context.Context, name string) (string, error) {
	s, ok := mr.m[name]
	if !ok {
		return "", ErrNotFound
	}

	return s, nil
}

func NewMemorySecretTypeResolver(m map[string]string) *MemorySecretTypeResolver {
	return &MemorySecretTypeResolver{m: m}
}

var NoOpSecretTypeResolver SecretTypeResolver = NewMemorySecretTypeResolver(map[string]string{})

type SecretMappingTypeResolver struct {
	SecretTypeResolver
}

var _ relspec.MappingTypeResolver[*References] = &SecretMappingTypeResolver{}

func (smtr *SecretMappingTypeResolver) ResolveMappingType(ctx context.Context, tm map[string]any) (*evaluate.Result[*References], error) {
	name, ok := tm["name"].(string)
	if !ok {
		return nil, &FieldNotFoundError{Name: "name"}
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := smtr.ResolveSecret(ctx, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Secrets.Set(ref.Errored(SecretID{Name: name}, err))
		r.SetValue(tm)
	} else if err != nil {
		return nil, err
	} else {
		r.References.Secrets.Set(ref.OK(SecretID{Name: name}))
		r.SetValue(value)
	}

	return r, nil
}

type SecretTemplateEnvironment struct {
	SecretTypeResolver
}

var (
	_ eval.Indexable                   = &SecretTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &SecretTemplateEnvironment{}
)

func (ste *SecretTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := ste.ResolveSecret(ctx, name)
	if errors.Is(err, ErrNotFound) {
		r.References.Secrets.Set(ref.Errored(SecretID{Name: name}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Secrets.Set(ref.OK(SecretID{Name: name}))
		r.SetValue(value)
	}

	return evaluate.StaticExpandable(r), nil
}

func (ste *SecretTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	if depth == 0 {
		return evaluate.StaticResult[*References](ste), nil
	}

	m, err := ste.ResolveAllSecrets(ctx)
	if err != nil {
		return nil, err
	}

	refs := NewReferences()

	cm := make(map[string]any, len(m))
	for name, v := range m {
		refs.Secrets.Set(ref.OK(SecretID{Name: name}))
		cm[name] = v
	}

	r := evaluate.NewResult(evaluate.NewMetadata(refs), cm)
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())
	return r, nil
}

type YAMLSecretTransformer struct{}

func (YAMLSecretTransformer) Transform(node *yaml.Node) (bool, error) {
	if node.Tag != "!Secret" {
		return false, nil
	}

	var name *yaml.Node
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) != 2 || node.Content[0].Value != "name" {
			return false, fmt.Errorf(`expected mapping-style !Secret to have exactly one key, "name"`)
		}

		name = node.Content[1]
	case yaml.SequenceNode:
		if len(node.Content) != 1 {
			return false, fmt.Errorf(`expected sequence-style !Secret to have exactly one item`)
		}

		name = node.Content[0]
	case yaml.ScalarNode:
		name = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: node.Value,
		}
	}

	// {$type: Secret, name: <name>}
	*node = yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$type"},
			{Kind: yaml.ScalarNode, Value: "Secret"},
			{Kind: yaml.ScalarNode, Value: "name"},
			name,
		},
	}
	return true, nil
}
