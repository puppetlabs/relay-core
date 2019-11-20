package evaluate

import (
	"sort"

	"github.com/puppetlabs/horsehead/v2/datastructure"
)

type UnresolvableSecret struct {
	Name string
}

type unresolvableSecretSort []UnresolvableSecret

func (s unresolvableSecretSort) Len() int           { return len(s) }
func (s unresolvableSecretSort) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s unresolvableSecretSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

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
	Secrets     []UnresolvableSecret
	Outputs     []UnresolvableOutput
	Parameters  []UnresolvableParameter
	Invocations []UnresolvableInvocation
}

type Result struct {
	Value        interface{}
	Unresolvable Unresolvable
}

func (r *Result) Complete() bool {
	return len(r.Unresolvable.Secrets) == 0 &&
		len(r.Unresolvable.Outputs) == 0 &&
		len(r.Unresolvable.Parameters) == 0 &&
		len(r.Unresolvable.Invocations) == 0
}

func (r *Result) extends(other *Result) *Result {
	// For convenience, we can copy in the information from another result,
	// which extends the unresolvables here.

	r.Unresolvable.extends(other.Unresolvable)
	return r
}

func (u *Unresolvable) extends(other Unresolvable) {
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
