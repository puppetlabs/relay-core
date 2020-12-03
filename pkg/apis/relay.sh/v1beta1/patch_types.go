package v1beta1

type PatchType string

const (
	PatchTypeJSONPatch PatchType = "JSONPatch"
)

type Patch struct {
	// Type is the type of patch being applied. Currently only JSONPatch (RFC
	// 6902) is supported.
	//
	// +kubebuilder:default="JSONPatch"
	// +kubebuilder:validation:Enum=JSONPatch
	Type PatchType `json:"type,omitempty"`

	// Data is the patch data to apply.
	Data []byte `json:"patch"`
}
