package spec

import (
	"context"
	"errors"
	"fmt"

	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
)

type StatusID struct {
	Action   string `json:"action"`
	Property string `json:"property"`
}

func (si StatusID) String() string {
	return fmt.Sprintf("status property %q of step %q", si.Property, si.Action)
}

func (si StatusID) Less(other StatusID) bool {
	return si.Action < other.Action || (si.Action == other.Action && si.Property < other.Property)
}

type StatusTypeResolver interface {
	ResolveStatus(ctx context.Context, name, property string) (bool, error)
}

type MemoryStatusKey struct {
	Action   string
	Property string
}

type MemoryStatusTypeResolver struct {
	m map[MemoryStatusKey]bool
}

var _ StatusTypeResolver = &MemoryStatusTypeResolver{}

func (mr *MemoryStatusTypeResolver) ResolveStatus(ctx context.Context, action, property string) (bool, error) {
	o, ok := mr.m[MemoryStatusKey{Action: action, Property: property}]
	if !ok {
		return false, ErrNotFound
	}

	return o, nil
}

func NewMemoryStatusTypeResolver(m map[MemoryStatusKey]bool) *MemoryStatusTypeResolver {
	return &MemoryStatusTypeResolver{m: m}
}

var NoOpStatusTypeResolver StatusTypeResolver = NewMemoryStatusTypeResolver(map[MemoryStatusKey]bool{})

type actionStatusTemplateEnvironment struct {
	r      StatusTypeResolver
	action string
}

var (
	_ eval.Indexable                   = &actionStatusTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &actionStatusTemplateEnvironment{}
)

func (aste *actionStatusTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	property, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r := evaluate.ContextualizedResult(evaluate.NewMetadata(NewReferences()))
	r.SetEvaluator(evaluate.DefaultEvaluator[*References]())

	value, err := aste.r.ResolveStatus(ctx, aste.action, property)
	if errors.Is(err, ErrNotFound) {
		r.References.Statuses.Set(ref.Errored(StatusID{Action: aste.action, Property: property}, err))
	} else if err != nil {
		return nil, err
	} else {
		r.References.Statuses.Set(ref.OK(StatusID{Action: aste.action, Property: property}))
		r.SetValue(value)
	}

	return evaluate.StaticExpandable(r), nil
}

func (aste *actionStatusTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	return evaluate.EmptyResult[*References](), nil
}

type StatusTemplateEnvironment struct {
	StatusTypeResolver
}

var (
	_ eval.Indexable                   = &StatusTemplateEnvironment{}
	_ evaluate.Expandable[*References] = &StatusTemplateEnvironment{}
)

func (ste *StatusTemplateEnvironment) Index(ctx context.Context, idx any) (any, error) {
	action, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	return &actionStatusTemplateEnvironment{
		r:      ste.StatusTypeResolver,
		action: action,
	}, nil
}

func (ste *StatusTemplateEnvironment) Expand(ctx context.Context, depth int) (*evaluate.Result[*References], error) {
	return nil, nil
}
