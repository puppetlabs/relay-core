package v1beta1

import (
	"encoding/json"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"k8s.io/apimachinery/pkg/runtime"
)

// Unstructured is arbitrary JSON data, which may also include base64-encoded
// binary data.
//
// +kubebuilder:validation:Type=object
type Unstructured struct {
	value transfer.JSONInterface `json:"-"`
}

func (u Unstructured) Value() interface{} {
	return u.value.Data
}

func (u Unstructured) MarshalJSON() ([]byte, error) {
	return u.value.MarshalJSON()
}

func (u *Unstructured) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u.value)
}

func (in *Unstructured) DeepCopy() *Unstructured {
	if in == nil {
		return nil
	}

	out := &Unstructured{}
	*out = *in
	out.value = transfer.JSONInterface{
		Data: runtime.DeepCopyJSONValue(in.value.Data),
	}
	return out
}

func AsUnstructured(value interface{}) Unstructured {
	return Unstructured{
		value: transfer.JSONInterface{Data: value},
	}
}

type UnstructuredObject map[string]Unstructured

func (uo UnstructuredObject) Value() map[string]interface{} {
	out := make(map[string]interface{}, len(uo))
	for k, v := range uo {
		out[k] = v.Value()
	}
	return out
}

func NewUnstructuredObject(value map[string]interface{}) UnstructuredObject {
	out := make(map[string]Unstructured, len(value))
	for k, v := range value {
		out[k] = AsUnstructured(v)
	}
	return out
}
