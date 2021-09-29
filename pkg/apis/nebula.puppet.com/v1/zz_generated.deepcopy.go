// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRun) DeepCopyInto(out *WorkflowRun) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.State.DeepCopyInto(&out.State)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRun.
func (in *WorkflowRun) DeepCopy() *WorkflowRun {
	if in == nil {
		return nil
	}
	out := new(WorkflowRun)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkflowRun) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRunList) DeepCopyInto(out *WorkflowRunList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WorkflowRun, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRunList.
func (in *WorkflowRunList) DeepCopy() *WorkflowRunList {
	if in == nil {
		return nil
	}
	out := new(WorkflowRunList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkflowRunList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRunSpec) DeepCopyInto(out *WorkflowRunSpec) {
	*out = *in
	out.WorkflowRef = in.WorkflowRef
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(v1beta1.UnstructuredObject, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.TenantRef != nil {
		in, out := &in.TenantRef, &out.TenantRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	in.WorkflowExecutionSink.DeepCopyInto(&out.WorkflowExecutionSink)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRunSpec.
func (in *WorkflowRunSpec) DeepCopy() *WorkflowRunSpec {
	if in == nil {
		return nil
	}
	out := new(WorkflowRunSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRunState) DeepCopyInto(out *WorkflowRunState) {
	*out = *in
	if in.Workflow != nil {
		in, out := &in.Workflow, &out.Workflow
		*out = make(v1beta1.UnstructuredObject, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make(map[string]v1beta1.UnstructuredObject, len(*in))
		for key, val := range *in {
			var outVal map[string]v1beta1.Unstructured
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(v1beta1.UnstructuredObject, len(*in))
				for key, val := range *in {
					(*out)[key] = *val.DeepCopy()
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRunState.
func (in *WorkflowRunState) DeepCopy() *WorkflowRunState {
	if in == nil {
		return nil
	}
	out := new(WorkflowRunState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRunStatus) DeepCopyInto(out *WorkflowRunStatus) {
	*out = *in
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make(map[string]WorkflowRunStatusSummary, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(map[string]WorkflowRunStatusSummary, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRunStatus.
func (in *WorkflowRunStatus) DeepCopy() *WorkflowRunStatus {
	if in == nil {
		return nil
	}
	out := new(WorkflowRunStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowRunStatusSummary) DeepCopyInto(out *WorkflowRunStatusSummary) {
	*out = *in
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = make(v1beta1.UnstructuredObject, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.InitTime != nil {
		in, out := &in.InitTime, &out.InitTime
		*out = (*in).DeepCopy()
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowRunStatusSummary.
func (in *WorkflowRunStatusSummary) DeepCopy() *WorkflowRunStatusSummary {
	if in == nil {
		return nil
	}
	out := new(WorkflowRunStatusSummary)
	in.DeepCopyInto(out)
	return out
}
