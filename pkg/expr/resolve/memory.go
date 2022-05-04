package resolve

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type MemoryDataTypeResolver struct {
	value interface{}
}

var _ DataTypeResolver = &MemoryDataTypeResolver{}

func (mr *MemoryDataTypeResolver) ResolveData(ctx context.Context) (interface{}, error) {
	return mr.value, nil
}

func NewMemoryDataTypeResolver(value interface{}) *MemoryDataTypeResolver {
	return &MemoryDataTypeResolver{value: value}
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
		return "", &model.SecretNotFoundError{Name: name}
	}

	return s, nil
}

func NewMemorySecretTypeResolver(m map[string]string) *MemorySecretTypeResolver {
	return &MemorySecretTypeResolver{m: m}
}

type MemoryConnectionKey struct {
	Type string
	Name string
}

type MemoryConnectionTypeResolver struct {
	m map[MemoryConnectionKey]interface{}
}

var _ ConnectionTypeResolver = &MemoryConnectionTypeResolver{}

func (mr *MemoryConnectionTypeResolver) ResolveAllConnections(ctx context.Context) (map[string]map[string]interface{}, error) {
	if len(mr.m) == 0 {
		return nil, nil
	}

	cm := make(map[string]map[string]interface{})

	for k, c := range mr.m {
		tm, found := cm[k.Type]
		if !found {
			tm = make(map[string]interface{})
			cm[k.Type] = tm
		}

		tm[k.Name] = c
	}

	return cm, nil
}

func (mr *MemoryConnectionTypeResolver) ResolveTypeOfConnections(ctx context.Context, connectionType string) (map[string]interface{}, error) {
	var tm map[string]interface{}

	for k, c := range mr.m {
		if k.Type != connectionType {
			continue
		} else if tm == nil {
			tm = make(map[string]interface{})
		}

		tm[k.Name] = c
	}

	return tm, nil
}

func (mr *MemoryConnectionTypeResolver) ResolveConnection(ctx context.Context, connectionType, name string) (interface{}, error) {
	o, ok := mr.m[MemoryConnectionKey{Type: connectionType, Name: name}]
	if !ok {
		return "", &model.ConnectionNotFoundError{Type: connectionType, Name: name}
	}

	return o, nil
}

func NewMemoryConnectionTypeResolver(m map[MemoryConnectionKey]interface{}) *MemoryConnectionTypeResolver {
	return &MemoryConnectionTypeResolver{m: m}
}

type MemoryOutputKey struct {
	From string
	Name string
}

type MemoryOutputTypeResolver struct {
	m map[MemoryOutputKey]interface{}
}

var _ OutputTypeResolver = &MemoryOutputTypeResolver{}

func (mr *MemoryOutputTypeResolver) ResolveAllOutputs(ctx context.Context) (map[string]map[string]interface{}, error) {
	if len(mr.m) == 0 {
		return nil, nil
	}

	om := make(map[string]map[string]interface{})

	for k, c := range mr.m {
		sm, found := om[k.From]
		if !found {
			sm = make(map[string]interface{})
			om[k.From] = sm
		}

		sm[k.Name] = c
	}

	return om, nil
}

func (mr *MemoryOutputTypeResolver) ResolveStepOutputs(ctx context.Context, from string) (map[string]interface{}, error) {
	var sm map[string]interface{}

	for k, c := range mr.m {
		if k.From != from {
			continue
		} else if sm == nil {
			sm = make(map[string]interface{})
		}

		sm[k.Name] = c
	}

	return sm, nil
}

func (mr *MemoryOutputTypeResolver) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	o, ok := mr.m[MemoryOutputKey{From: from, Name: name}]
	if !ok {
		return "", &model.OutputNotFoundError{From: from, Name: name}
	}

	return o, nil
}

func NewMemoryOutputTypeResolver(m map[MemoryOutputKey]interface{}) *MemoryOutputTypeResolver {
	return &MemoryOutputTypeResolver{m: m}
}

type MemoryParameterTypeResolver struct {
	m map[string]interface{}
}

var _ ParameterTypeResolver = &MemoryParameterTypeResolver{}

func (mr *MemoryParameterTypeResolver) ResolveAllParameters(ctx context.Context) (map[string]interface{}, error) {
	return mr.m, nil
}

func (mr *MemoryParameterTypeResolver) ResolveParameter(ctx context.Context, name string) (interface{}, error) {
	p, ok := mr.m[name]
	if !ok {
		return nil, &model.ParameterNotFoundError{Name: name}
	}

	return p, nil
}

func NewMemoryParameterTypeResolver(m map[string]interface{}) *MemoryParameterTypeResolver {
	return &MemoryParameterTypeResolver{m: m}
}

type MemoryAnswerKey struct {
	AskRef string
	Name   string
}

type MemoryAnswerTypeResolver struct {
	m map[MemoryAnswerKey]interface{}
}

var _ AnswerTypeResolver = &MemoryAnswerTypeResolver{}

func (mr *MemoryAnswerTypeResolver) ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error) {
	o, ok := mr.m[MemoryAnswerKey{AskRef: askRef, Name: name}]
	if !ok {
		return "", &model.AnswerNotFoundError{AskRef: askRef, Name: name}
	}

	return o, nil
}

func NewMemoryAnswerTypeResolver(m map[MemoryAnswerKey]interface{}) *MemoryAnswerTypeResolver {
	return &MemoryAnswerTypeResolver{m: m}
}

type MemoryStatusKey struct {
	Name     string
	Property string
}

type MemoryStatusTypeResolver struct {
	m map[MemoryStatusKey]bool
}

var _ StatusTypeResolver = &MemoryStatusTypeResolver{}

func (mr *MemoryStatusTypeResolver) ResolveStatus(ctx context.Context, name, property string) (bool, error) {
	o, ok := mr.m[MemoryStatusKey{Name: name, Property: property}]
	if !ok {
		return false, &model.StatusNotFoundError{Name: name, Property: property}
	}

	return o, nil
}

func NewMemoryStatusTypeResolver(m map[MemoryStatusKey]bool) *MemoryStatusTypeResolver {
	return &MemoryStatusTypeResolver{m: m}
}
