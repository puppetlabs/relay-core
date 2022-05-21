package spec

import (
	"github.com/puppetlabs/leg/relspec/pkg/ref"
)

type References struct {
	Data        *ref.Log[DataID]
	Secrets     *ref.Log[SecretID]
	Connections *ref.Log[ConnectionID]
	Outputs     *ref.Log[OutputID]
	Parameters  *ref.Log[ParameterID]
	Answers     *ref.Log[AnswerID]
	Statuses    *ref.Log[StatusID]
}

func (r *References) collect() ref.Collection {
	if r == nil {
		return ref.Collections(nil)
	}

	return ref.Collections{
		r.Data,
		r.Secrets,
		r.Connections,
		r.Outputs,
		r.Parameters,
		r.Answers,
		r.Statuses,
	}
}

func (r *References) SetUsed(flag bool) {
	r.collect().SetUsed(flag)
}

func (r *References) Used() bool {
	return r.collect().Used()
}

func (r *References) Resolved() bool {
	return r.collect().Resolved()
}

func (r *References) OK() bool {
	return r.collect().OK()
}

func (r *References) Merge(others ...*References) *References {
	if len(others) == 0 {
		return r
	} else if r == nil {
		return NewReferences().Merge(others...)
	}

	for _, refs := range others {
		if refs == nil {
			continue
		}

		r.Data = r.Data.Merge(refs.Data)
		r.Secrets = r.Secrets.Merge(refs.Secrets)
		r.Connections = r.Connections.Merge(refs.Connections)
		r.Outputs = r.Outputs.Merge(refs.Outputs)
		r.Parameters = r.Parameters.Merge(refs.Parameters)
		r.Answers = r.Answers.Merge(refs.Answers)
		r.Statuses = r.Statuses.Merge(refs.Statuses)
	}

	return r
}

func (r *References) ToError() *UnresolvableError {
	if r == nil {
		return nil
	}

	err := &UnresolvableError{}
	AddUnresolvableErrorsTo(err, r.Data)
	AddUnresolvableErrorsTo(err, r.Secrets)
	AddUnresolvableErrorsTo(err, r.Connections)
	AddUnresolvableErrorsTo(err, r.Outputs)
	AddUnresolvableErrorsTo(err, r.Parameters)
	AddUnresolvableErrorsTo(err, r.Answers)
	AddUnresolvableErrorsTo(err, r.Statuses)

	if len(err.Causes) == 0 {
		return nil
	}

	return err
}

func NewReferences() *References {
	return &References{
		Data:        ref.NewLog[DataID](),
		Secrets:     ref.NewLog[SecretID](),
		Connections: ref.NewLog[ConnectionID](),
		Outputs:     ref.NewLog[OutputID](),
		Parameters:  ref.NewLog[ParameterID](),
		Answers:     ref.NewLog[AnswerID](),
		Statuses:    ref.NewLog[StatusID](),
	}
}

func CopyReferences(from *References) *References {
	return NewReferences().Merge(from)
}

func InitialReferences(fn func(r *References)) *References {
	r := NewReferences()
	fn(r)
	return r
}
