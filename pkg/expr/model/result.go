package model

import (
	"sort"

	"github.com/puppetlabs/leg/datastructure"
)

type UnresolvableData struct {
	Name string
}

type unresolvableDataSort []UnresolvableData

func (s unresolvableDataSort) Len() int           { return len(s) }
func (s unresolvableDataSort) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s unresolvableDataSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type UnresolvableSecret struct {
	Name string
}

type unresolvableSecretSort []UnresolvableSecret

func (s unresolvableSecretSort) Len() int           { return len(s) }
func (s unresolvableSecretSort) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s unresolvableSecretSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type UnresolvableConnection struct {
	Type string
	Name string
}

type unresolvableConnectionSort []UnresolvableConnection

func (s unresolvableConnectionSort) Len() int { return len(s) }
func (s unresolvableConnectionSort) Less(i, j int) bool {
	return s[i].Type < s[j].Type || (s[i].Type == s[j].Type && s[i].Name < s[j].Name)
}
func (s unresolvableConnectionSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type UnresolvableOutput struct {
	From string
	Name string
}

type unresolvableOutputSort []UnresolvableOutput

func (s unresolvableOutputSort) Len() int { return len(s) }
func (s unresolvableOutputSort) Less(i, j int) bool {
	return s[i].From < s[j].From || (s[i].From == s[j].From && s[i].Name < s[j].Name)
}
func (s unresolvableOutputSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type UnresolvableParameter struct {
	Name string
}

type unresolvableParameterSort []UnresolvableParameter

func (s unresolvableParameterSort) Len() int           { return len(s) }
func (s unresolvableParameterSort) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s unresolvableParameterSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type UnresolvableAnswer struct {
	AskRef string
	Name   string
}

type unresolvableAnswerSort []UnresolvableAnswer

func (s unresolvableAnswerSort) Len() int { return len(s) }
func (s unresolvableAnswerSort) Less(i, j int) bool {
	return s[i].AskRef < s[j].AskRef || (s[i].AskRef == s[j].AskRef && s[i].Name < s[j].Name)
}
func (s unresolvableAnswerSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type UnresolvableStatus struct {
	Name     string
	Property string
}

type unresolvableStatusSort []UnresolvableStatus

func (s unresolvableStatusSort) Len() int { return len(s) }
func (s unresolvableStatusSort) Less(i, j int) bool {
	return s[i].Name < s[j].Name || (s[i].Name == s[j].Name && s[i].Property < s[j].Property)
}

func (s unresolvableStatusSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type UnresolvableInvocation struct {
	Name  string
	Cause error
}

type unresolvableInvocationSort []UnresolvableInvocation

func (s unresolvableInvocationSort) Len() int { return len(s) }
func (s unresolvableInvocationSort) Less(i, j int) bool {
	return s[i].Name < s[j].Name || (s[i].Name == s[j].Name && s[i].Cause.Error() < s[j].Cause.Error())
}
func (s unresolvableInvocationSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type Unresolvable struct {
	Data        []UnresolvableData
	Secrets     []UnresolvableSecret
	Connections []UnresolvableConnection
	Outputs     []UnresolvableOutput
	Parameters  []UnresolvableParameter
	Answers     []UnresolvableAnswer
	Status      []UnresolvableStatus
	Invocations []UnresolvableInvocation
}

func (u *Unresolvable) AsError() error {
	err := &UnresolvableError{}

	for _, d := range u.Data {
		err.Causes = append(err.Causes, &DataNotFoundError{Name: d.Name})
	}

	for _, s := range u.Secrets {
		err.Causes = append(err.Causes, &SecretNotFoundError{Name: s.Name})
	}

	for _, c := range u.Connections {
		err.Causes = append(err.Causes, &ConnectionNotFoundError{Type: c.Type, Name: c.Name})
	}

	for _, o := range u.Outputs {
		err.Causes = append(err.Causes, &OutputNotFoundError{From: o.From, Name: o.Name})
	}

	for _, p := range u.Parameters {
		err.Causes = append(err.Causes, &ParameterNotFoundError{Name: p.Name})
	}

	for _, a := range u.Answers {
		err.Causes = append(err.Causes, &AnswerNotFoundError{AskRef: a.AskRef, Name: a.Name})
	}

	for _, a := range u.Status {
		err.Causes = append(err.Causes, &StatusNotFoundError{Name: a.Name, Property: a.Property})
	}

	for _, i := range u.Invocations {
		err.Causes = append(err.Causes, &FunctionResolutionError{Name: i.Name, Cause: i.Cause})
	}

	if len(err.Causes) == 0 {
		return nil
	}

	return err
}

func (u *Unresolvable) Extends(other Unresolvable) {
	// Data
	if len(u.Data) == 0 {
		u.Data = append(u.Data, other.Data...)
	} else if len(other.Data) != 0 {
		set := datastructure.NewHashSet()
		for _, p := range u.Data {
			set.Add(p)
		}
		for _, p := range other.Data {
			set.Add(p)
		}
		u.Data = nil
		set.ValuesInto(&u.Data)
		sort.Sort(unresolvableDataSort(u.Data))
	}

	// Secrets
	if len(u.Secrets) == 0 {
		u.Secrets = append(u.Secrets, other.Secrets...)
	} else if len(other.Secrets) != 0 {
		set := datastructure.NewHashSet()
		for _, s := range u.Secrets {
			set.Add(s)
		}
		for _, s := range other.Secrets {
			set.Add(s)
		}
		u.Secrets = nil
		set.ValuesInto(&u.Secrets)
		sort.Sort(unresolvableSecretSort(u.Secrets))
	}

	// Connections
	if len(u.Connections) == 0 {
		u.Connections = append(u.Connections, other.Connections...)
	} else if len(other.Connections) != 0 {
		set := datastructure.NewHashSet()
		for _, o := range u.Connections {
			set.Add(o)
		}
		for _, o := range other.Connections {
			set.Add(o)
		}
		u.Connections = nil
		set.ValuesInto(&u.Connections)
		sort.Sort(unresolvableConnectionSort(u.Connections))
	}

	// Outputs
	if len(u.Outputs) == 0 {
		u.Outputs = append(u.Outputs, other.Outputs...)
	} else if len(other.Outputs) != 0 {
		set := datastructure.NewHashSet()
		for _, o := range u.Outputs {
			set.Add(o)
		}
		for _, o := range other.Outputs {
			set.Add(o)
		}
		u.Outputs = nil
		set.ValuesInto(&u.Outputs)
		sort.Sort(unresolvableOutputSort(u.Outputs))
	}

	// Parameters
	if len(u.Parameters) == 0 {
		u.Parameters = append(u.Parameters, other.Parameters...)
	} else if len(other.Parameters) != 0 {
		set := datastructure.NewHashSet()
		for _, p := range u.Parameters {
			set.Add(p)
		}
		for _, p := range other.Parameters {
			set.Add(p)
		}
		u.Parameters = nil
		set.ValuesInto(&u.Parameters)
		sort.Sort(unresolvableParameterSort(u.Parameters))
	}

	// Answers
	if len(u.Answers) == 0 {
		u.Answers = append(u.Answers, other.Answers...)
	} else if len(other.Answers) != 0 {
		set := datastructure.NewHashSet()
		for _, o := range u.Answers {
			set.Add(o)
		}
		for _, o := range other.Answers {
			set.Add(o)
		}
		u.Answers = nil
		set.ValuesInto(&u.Answers)
		sort.Sort(unresolvableAnswerSort(u.Answers))
	}

	// Status
	if len(u.Status) == 0 {
		u.Status = append(u.Status, other.Status...)
	} else if len(other.Status) != 0 {
		set := datastructure.NewHashSet()
		for _, p := range u.Status {
			set.Add(p)
		}
		for _, p := range other.Status {
			set.Add(p)
		}
		u.Status = nil
		set.ValuesInto(&u.Status)
		sort.Sort(unresolvableStatusSort(u.Status))
	}

	// Invocations
	if len(u.Invocations) == 0 {
		u.Invocations = append(u.Invocations, other.Invocations...)
	} else if len(other.Invocations) != 0 {
		set := datastructure.NewHashSet()
		for _, i := range u.Invocations {
			set.Add(i)
		}
		for _, i := range other.Invocations {
			set.Add(i)
		}
		u.Invocations = nil
		set.ValuesInto(&u.Invocations)
		sort.Sort(unresolvableInvocationSort(u.Invocations))
	}
}

type Result struct {
	Value        interface{}
	Unresolvable Unresolvable
}

func (r *Result) Complete() bool {
	return r.Unresolvable.AsError() == nil
}

func (r *Result) Extends(other *Result) *Result {
	// For convenience, we can copy in the information from another result,
	// which extends the unresolvables here.
	if other != nil {
		r.Unresolvable.Extends(other.Unresolvable)
	}
	return r
}

func CombineResultSlice(s []*Result) *Result {
	vs := make([]interface{}, len(s))
	r := &Result{
		Value: vs,
	}

	for i, ri := range s {
		if ri == nil {
			continue
		}

		vs[i] = ri.Value
		r.Extends(ri)
	}

	return r
}

func CombineResultMap(m map[string]*Result) *Result {
	vm := make(map[string]interface{}, len(m))
	r := &Result{
		Value: vm,
	}

	for key, ri := range m {
		if ri == nil {
			continue
		}

		vm[key] = ri.Value
		r.Extends(ri)
	}

	return r
}
